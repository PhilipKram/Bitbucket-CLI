package pipeline

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/completion"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

func newCmdStats() *cobra.Command {
	var days int
	var branch string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "stats <workspace/repo-slug>",
		Short: "Show pipeline statistics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			repo := args[0]
			cutoff := time.Now().UTC().AddDate(0, 0, -days)

			pipelines, err := fetchPipelinesInWindow(client, repo, branch, cutoff)
			if err != nil {
				return err
			}

			total := len(pipelines)
			var success, failed, canceled int
			for _, p := range pipelines {
				if p.State.Result != nil {
					switch p.State.Result.Name {
					case "SUCCESSFUL":
						success++
					case "FAILED", "ERROR":
						failed++
					case "STOPPED":
						canceled++
					}
				}
			}

			successRate := 0.0
			failureRate := 0.0
			if total > 0 {
				successRate = float64(success) / float64(total) * 100
				failureRate = float64(failed) / float64(total) * 100
			}

			if jsonOut {
				output.PrintJSON(map[string]interface{}{
					"total":        total,
					"successful":   success,
					"failed":       failed,
					"canceled":     canceled,
					"success_rate": successRate,
					"failure_rate": failureRate,
					"days":         days,
				})
				return nil
			}

			output.PrintMessage("Pipeline Statistics (last %d days)", days)
			output.PrintMessage("")
			output.PrintMessage("Total:        %d", total)
			output.PrintMessage("Successful:   %d", success)
			output.PrintMessage("Failed:       %d", failed)
			output.PrintMessage("Canceled:     %d", canceled)
			output.PrintMessage("Success Rate: %.1f%%", successRate)
			output.PrintMessage("Failure Rate: %.1f%%", failureRate)
			return nil
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Number of days to look back")
	cmd.Flags().StringVar(&branch, "branch", "", "Filter by branch name")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.ValidArgsFunction = completion.RepositoryNamesWithDescriptions
	return cmd
}

// fetchPipelinesInWindow fetches all pipelines within the given time window,
// optionally filtered by branch.
func fetchPipelinesInWindow(client *api.Client, repo, branch string, cutoff time.Time) ([]Pipeline, error) {
	var all []Pipeline
	page := 1

	// repo is "workspace/repo-slug"; split and escape each segment separately
	repoParts := strings.SplitN(repo, "/", 2)
	if len(repoParts) != 2 {
		return nil, fmt.Errorf("invalid repository format %q, expected workspace/repo-slug", repo)
	}
	escapedRepo := url.PathEscape(repoParts[0]) + "/" + url.PathEscape(repoParts[1])

	for {
		path := fmt.Sprintf("/repositories/%s/pipelines/?pagelen=100&page=%d&sort=-created_on",
			escapedRepo, page)
		data, err := client.Get(path)
		if err != nil {
			return nil, err
		}

		var paginated api.PaginatedResponse
		if err := json.Unmarshal(data, &paginated); err != nil {
			return nil, err
		}

		var pipelines []Pipeline
		if err := json.Unmarshal(paginated.Values, &pipelines); err != nil {
			return nil, err
		}

		if len(pipelines) == 0 {
			break
		}

		reachedCutoff := false
		for _, p := range pipelines {
			created, err := time.Parse(time.RFC3339, p.CreatedOn)
			if err != nil {
				// Try parsing with nanoseconds
				created, err = time.Parse("2006-01-02T15:04:05.999999999+00:00", p.CreatedOn)
				if err != nil {
					continue
				}
			}
			if created.Before(cutoff) {
				reachedCutoff = true
				break
			}
			if branch != "" && p.Target.RefName != branch {
				continue
			}
			all = append(all, p)
		}

		if reachedCutoff || paginated.Next == "" {
			break
		}
		page++
	}

	return all, nil
}
