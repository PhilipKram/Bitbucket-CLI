package pipeline

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/completion"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type dailyStats struct {
	Date   string `json:"date"`
	Total  int    `json:"total"`
	Passed int    `json:"passed"`
	Failed int    `json:"failed"`
	Rate   string `json:"rate"`
}

func newCmdTrends() *cobra.Command {
	var days int
	var branch string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "trends <workspace/repo-slug>",
		Short: "Show pipeline trends over time",
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

			// Group by day
			byDay := make(map[string]*dailyStats)
			for _, p := range pipelines {
				date := ""
				if len(p.CreatedOn) >= 10 {
					date = p.CreatedOn[:10]
				}
				if date == "" {
					continue
				}

				ds, ok := byDay[date]
				if !ok {
					ds = &dailyStats{Date: date}
					byDay[date] = ds
				}
				ds.Total++
				if p.State.Result != nil {
					switch p.State.Result.Name {
					case "SUCCESSFUL":
						ds.Passed++
					case "FAILED", "ERROR":
						ds.Failed++
					}
				}
			}

			// Sort dates
			dates := make([]string, 0, len(byDay))
			for d := range byDay {
				dates = append(dates, d)
			}
			sort.Strings(dates)

			// Compute rates
			results := make([]dailyStats, 0, len(dates))
			for _, d := range dates {
				ds := byDay[d]
				rate := 0.0
				if ds.Total > 0 {
					rate = float64(ds.Passed) / float64(ds.Total) * 100
				}
				ds.Rate = fmt.Sprintf("%.0f%%", rate)
				results = append(results, *ds)
			}

			if jsonOut {
				output.PrintJSON(results)
				return nil
			}

			table := output.NewTable("DATE", "TOTAL", "PASSED", "FAILED", "RATE")
			for _, ds := range results {
				table.AddRow(
					ds.Date,
					fmt.Sprintf("%d", ds.Total),
					fmt.Sprintf("%d", ds.Passed),
					fmt.Sprintf("%d", ds.Failed),
					ds.Rate,
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Number of days to look back")
	cmd.Flags().StringVar(&branch, "branch", "", "Filter by branch name")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.ValidArgsFunction = completion.RepositoryNamesWithDescriptions
	return cmd
}
