package pipeline

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Pipeline struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	State       struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
		Stage *struct {
			Name string `json:"name"`
		} `json:"stage"`
	} `json:"state"`
	Target struct {
		Type     string `json:"type"`
		RefType  string `json:"ref_type"`
		RefName  string `json:"ref_name"`
		Selector struct {
			Type    string `json:"type"`
			Pattern string `json:"pattern"`
		} `json:"selector"`
	} `json:"target"`
	Creator struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	CreatedOn   string `json:"created_on"`
	CompletedOn string `json:"completed_on"`
	DurationInSeconds int `json:"duration_in_seconds"`
}

type PipelineStep struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	State  struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	StartedOn   string `json:"started_on"`
	CompletedOn string `json:"completed_on"`
	DurationInSeconds int `json:"duration_in_seconds"`
}

func NewCmdPipeline() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pipeline",
		Aliases: []string{"pipe", "ci"},
		Short:   "Manage pipelines (CI/CD)",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdTrigger())
	cmd.AddCommand(newCmdStop())
	cmd.AddCommand(newCmdSteps())
	cmd.AddCommand(newCmdLog())

	return cmd
}

func newCmdList() *cobra.Command {
	var page int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List pipelines",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pipelines/?pagelen=20&page=%d&sort=-created_on", args[0], page)
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

			if jsonOut {
				output.PrintJSON(pipelines)
				return nil
			}

			table := output.NewTable("BUILD#", "STATE", "RESULT", "BRANCH", "CREATOR", "CREATED", "DURATION")
			for _, p := range pipelines {
				state := p.State.Name
				result := "–"
				if p.State.Result != nil {
					result = p.State.Result.Name
				}
				duration := "–"
				if p.DurationInSeconds > 0 {
					duration = fmt.Sprintf("%ds", p.DurationInSeconds)
				}
				created := ""
				if len(p.CreatedOn) >= 10 {
					created = p.CreatedOn[:10]
				}
				table.AddRow(
					fmt.Sprintf("#%d", p.BuildNumber),
					state,
					result,
					p.Target.RefName,
					p.Creator.DisplayName,
					created,
					duration,
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <workspace/repo-slug> <pipeline-uuid>",
		Short: "View pipeline details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pipelines/%s", args[0], url.PathEscape(args[1]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var p Pipeline
			if err := json.Unmarshal(data, &p); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(p)
				return nil
			}

			result := "–"
			if p.State.Result != nil {
				result = p.State.Result.Name
			}
			output.PrintMessage("Build #%d", p.BuildNumber)
			output.PrintMessage("UUID:      %s", p.UUID)
			output.PrintMessage("State:     %s", p.State.Name)
			output.PrintMessage("Result:    %s", result)
			output.PrintMessage("Branch:    %s", p.Target.RefName)
			output.PrintMessage("Creator:   %s", p.Creator.DisplayName)
			output.PrintMessage("Created:   %s", p.CreatedOn)
			output.PrintMessage("Completed: %s", p.CompletedOn)
			if p.DurationInSeconds > 0 {
				output.PrintMessage("Duration:  %ds", p.DurationInSeconds)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdTrigger() *cobra.Command {
	var branch string
	var pattern string
	var customPipe bool

	cmd := &cobra.Command{
		Use:   "trigger <workspace/repo-slug>",
		Short: "Trigger a new pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			target := map[string]interface{}{
				"ref_type": "branch",
				"type":     "pipeline_ref_target",
				"ref_name": branch,
			}
			if customPipe && pattern != "" {
				target["selector"] = map[string]string{
					"type":    "custom",
					"pattern": pattern,
				}
			}

			body := map[string]interface{}{
				"target": target,
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/pipelines/", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var p Pipeline
			if err := json.Unmarshal(data, &p); err != nil {
				return err
			}
			output.PrintMessage("Pipeline #%d triggered (UUID: %s)", p.BuildNumber, p.UUID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch to run pipeline on")
	cmd.Flags().StringVar(&pattern, "pattern", "", "Custom pipeline pattern name")
	cmd.Flags().BoolVar(&customPipe, "custom", false, "Trigger a custom pipeline")
	return cmd
}

func newCmdStop() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <workspace/repo-slug> <pipeline-uuid>",
		Short: "Stop a running pipeline",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pipelines/%s/stopPipeline", args[0], url.PathEscape(args[1]))
			_, err = client.Post(path, "")
			if err != nil {
				return err
			}
			output.PrintMessage("Pipeline %s stop requested.", args[1])
			return nil
		},
	}
}

func newCmdSteps() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "steps <workspace/repo-slug> <pipeline-uuid>",
		Short: "List steps for a pipeline",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pipelines/%s/steps/", args[0], url.PathEscape(args[1]))
			data, err := client.Get(path)
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

			if jsonOut {
				output.PrintJSON(steps)
				return nil
			}

			table := output.NewTable("UUID", "NAME", "STATE", "RESULT", "DURATION")
			for _, s := range steps {
				result := "–"
				if s.State.Result != nil {
					result = s.State.Result.Name
				}
				duration := "–"
				if s.DurationInSeconds > 0 {
					duration = fmt.Sprintf("%ds", s.DurationInSeconds)
				}
				table.AddRow(s.UUID[:12], s.Name, s.State.Name, result, duration)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdLog() *cobra.Command {
	return &cobra.Command{
		Use:   "log <workspace/repo-slug> <pipeline-uuid> <step-uuid>",
		Short: "View logs for a pipeline step",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pipelines/%s/steps/%s/log",
				args[0], url.PathEscape(args[1]), url.PathEscape(args[2]))
			data, err := client.Get(path)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}
