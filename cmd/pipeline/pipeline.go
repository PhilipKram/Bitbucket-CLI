package pipeline

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	cmd.AddCommand(newCmdWatch())

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

func newCmdWatch() *cobra.Command {
	var buildNumber int
	var interval int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "watch <workspace/repo-slug>",
		Short: "Watch pipeline status in real-time",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			repo := args[0]
			var pipelineUUID string

			// If buildNumber is 0, get the latest pipeline
			if buildNumber == 0 {
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
			} else {
				// Get pipeline by build number
				path := fmt.Sprintf("/repositories/%s/pipelines/?pagelen=100&sort=-created_on", repo)
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

				found := false
				for _, p := range pipelines {
					if p.BuildNumber == buildNumber {
						pipelineUUID = p.UUID
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("pipeline #%d not found", buildNumber)
				}
			}

			// Set up signal handling for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigChan)

			// Poll the pipeline status
			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			for {
				// Fetch pipeline details
				path := fmt.Sprintf("/repositories/%s/pipelines/%s", repo, url.PathEscape(pipelineUUID))
				data, err := client.Get(path)
				if err != nil {
					return err
				}

				var p Pipeline
				if err := json.Unmarshal(data, &p); err != nil {
					return err
				}

				// Fetch pipeline steps
				stepsPath := fmt.Sprintf("/repositories/%s/pipelines/%s/steps/", repo, url.PathEscape(pipelineUUID))
				stepsData, err := client.Get(stepsPath)
				if err != nil {
					return err
				}

				var stepsPaginated api.PaginatedResponse
				if err := json.Unmarshal(stepsData, &stepsPaginated); err != nil {
					return err
				}

				var steps []PipelineStep
				if err := json.Unmarshal(stepsPaginated.Values, &steps); err != nil {
					return err
				}

				if jsonOut {
					output.PrintJSON(map[string]interface{}{
						"pipeline": p,
						"steps":    steps,
					})
				} else {
					// Clear screen for clean display
					output.ClearScreen()

					// Display pipeline status with colors
					result := "–"
					resultColor := "gray"
					if p.State.Result != nil {
						result = p.State.Result.Name
						switch result {
						case "SUCCESSFUL":
							resultColor = "green"
						case "FAILED", "ERROR":
							resultColor = "red"
						case "STOPPED":
							resultColor = "yellow"
						}
					}

					stateColor := "gray"
					switch p.State.Name {
					case "COMPLETED":
						stateColor = resultColor // Use result color for completed
					case "IN_PROGRESS", "PENDING":
						stateColor = "yellow"
					}

					output.PrintMessage("\n=== Pipeline #%d ===", p.BuildNumber)
					output.PrintMessage("State:  %s", output.ColorText(p.State.Name, stateColor))
					output.PrintMessage("Result: %s", output.ColorText(result, resultColor))
					output.PrintMessage("Branch: %s", p.Target.RefName)

					// Display steps with colors
					if len(steps) > 0 {
						output.PrintMessage("\nSteps:")
						for _, s := range steps {
							stepResult := "–"
							stepResultColor := "gray"
							if s.State.Result != nil {
								stepResult = s.State.Result.Name
								switch stepResult {
								case "SUCCESSFUL":
									stepResultColor = "green"
								case "FAILED", "ERROR":
									stepResultColor = "red"
								case "STOPPED":
									stepResultColor = "yellow"
								}
							}

							stepStateColor := "gray"
							switch s.State.Name {
							case "COMPLETED":
								stepStateColor = stepResultColor // Use result color for completed
							case "IN_PROGRESS":
								stepStateColor = "yellow"
							case "PENDING":
								stepStateColor = "gray"
							}

							output.PrintMessage("  - %s: %s (%s)",
								s.Name,
								output.ColorText(s.State.Name, stepStateColor),
								output.ColorText(stepResult, stepResultColor))
						}
					}
				}

				// Check if pipeline is in a terminal state
				if p.State.Name == "COMPLETED" {
					if p.State.Result != nil {
						if p.State.Result.Name == "SUCCESSFUL" {
							output.PrintMessage("\nPipeline completed successfully")
							os.Exit(0)
						} else {
							output.PrintMessage("\nPipeline failed: %s", p.State.Result.Name)
							os.Exit(1)
						}
					}
					output.PrintMessage("\nPipeline completed")
					os.Exit(0)
				}

				// Wait for next poll or signal interrupt
				select {
				case <-ticker.C:
					// Continue to next iteration
				case <-sigChan:
					output.PrintMessage("\nWatch interrupted. Exiting gracefully...")
					return nil
				}
			}
		},
	}
	cmd.Flags().IntVarP(&buildNumber, "build", "b", 0, "Build number to watch (0 = latest)")
	cmd.Flags().IntVarP(&interval, "interval", "i", 5, "Polling interval in seconds")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
