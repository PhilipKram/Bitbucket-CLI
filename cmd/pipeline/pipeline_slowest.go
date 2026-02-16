package pipeline

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/completion"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

func newCmdSlowest() *cobra.Command {
	var pipelineUUID string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "slowest <workspace/repo-slug>",
		Short: "Show slowest pipeline steps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			repo := args[0]

			// If no pipeline UUID specified, get the latest pipeline
			if pipelineUUID == "" {
				path := fmt.Sprintf("/repositories/%s/pipelines/?pagelen=1&sort=-created_on", repo)
				data, err := client.Get(path)
				if err != nil {
					return err
				}

				var paginated api.PaginatedResponse
				if err := json.Unmarshal(data, &paginated); err != nil {
					return err
				}

				var pipelines []Pipeline
				if err := json.Unmarshal(paginated.Values, &pipelines); err != nil {
					return err
				}

				if len(pipelines) == 0 {
					return fmt.Errorf("no pipelines found")
				}
				pipelineUUID = pipelines[0].UUID
			}

			// Fetch steps
			stepsPath := fmt.Sprintf("/repositories/%s/pipelines/%s/steps/",
				repo, url.PathEscape(pipelineUUID))
			data, err := client.Get(stepsPath)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var steps []PipelineStep
			if err := json.Unmarshal(paginated.Values, &steps); err != nil {
				return err
			}

			// Sort by duration descending
			sort.Slice(steps, func(i, j int) bool {
				return steps[i].DurationInSeconds > steps[j].DurationInSeconds
			})

			// Apply limit
			if limit > 0 && limit < len(steps) {
				steps = steps[:limit]
			}

			if jsonOut {
				output.PrintJSON(steps)
				return nil
			}

			table := output.NewTable("STEP", "DURATION", "STATUS")
			for _, s := range steps {
				duration := "–"
				if s.DurationInSeconds > 0 {
					duration = fmt.Sprintf("%ds", s.DurationInSeconds)
				}
				status := s.State.Name
				if s.State.Result != nil {
					status = s.State.Result.Name
				}
				table.AddRow(s.Name, duration, status)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().StringVar(&pipelineUUID, "pipeline", "", "Pipeline UUID (default: latest)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Number of steps to show")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.ValidArgsFunction = completion.RepositoryNamesWithDescriptions
	return cmd
}
