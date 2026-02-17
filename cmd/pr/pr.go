package pr

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/cmdutil"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Author      struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CloseSourceBranch bool `json:"close_source_branch"`
	MergeCommit       *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	CommentCount int `json:"comment_count"`
	TaskCount    int `json:"task_count"`
	Links        struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Reviewers []struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"reviewers"`
	Participants []struct {
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
	} `json:"participants"`
}

func NewCmdPR() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pr",
		Aliases: []string{"pull-request"},
		Short:   "Manage pull requests",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdMerge())
	cmd.AddCommand(newCmdApprove())
	cmd.AddCommand(newCmdUnapprove())
	cmd.AddCommand(newCmdDecline())
	cmd.AddCommand(newCmdComments())
	cmd.AddCommand(newCmdComment())
	cmd.AddCommand(newCmdDiff())
	cmd.AddCommand(newCmdActivity())

	return cmd
}

func newCmdList() *cobra.Command {
	var state string
	var page int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List pull requests",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests?pagelen=25&page=%d", args[0], page)
			if state != "" {
				path += "&state=" + url.QueryEscape(strings.ToUpper(state))
			}
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var prs []PullRequest
			if err := json.Unmarshal(paginated.Values, &prs); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(prs)
				return nil
			}

			table := output.NewTable("ID", "TITLE", "AUTHOR", "SOURCE", "DEST", "STATE")
			for _, pr := range prs {
				table.AddRow(
					fmt.Sprintf("#%d", pr.ID),
					output.Truncate(pr.Title, 50),
					pr.Author.DisplayName,
					pr.Source.Branch.Name,
					pr.Destination.Branch.Name,
					pr.State,
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state (OPEN, MERGED, DECLINED, SUPERSEDED)")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <workspace/repo-slug> <pr-id>",
		Short: "View pull request details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s", args[0], args[1])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var pr PullRequest
			if err := json.Unmarshal(data, &pr); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(pr)
				return nil
			}

			output.PrintMessage("PR #%d: %s", pr.ID, pr.Title)
			output.PrintMessage("State:       %s", pr.State)
			output.PrintMessage("Author:      %s", pr.Author.DisplayName)
			output.PrintMessage("Source:      %s", pr.Source.Branch.Name)
			output.PrintMessage("Destination: %s", pr.Destination.Branch.Name)
			output.PrintMessage("Created:     %s", pr.CreatedOn)
			output.PrintMessage("Updated:     %s", pr.UpdatedOn)
			output.PrintMessage("Comments:    %d", pr.CommentCount)
			output.PrintMessage("URL:         %s", pr.Links.HTML.Href)
			if pr.Description != "" {
				output.PrintMessage("\nDescription:\n%s", pr.Description)
			}
			if len(pr.Reviewers) > 0 {
				names := make([]string, len(pr.Reviewers))
				for i, r := range pr.Reviewers {
					names[i] = r.DisplayName
				}
				output.PrintMessage("\nReviewers: %s", strings.Join(names, ", "))
			}
			if len(pr.Participants) > 0 {
				output.PrintMessage("\nParticipants:")
				for _, p := range pr.Participants {
					approved := ""
					if p.Approved {
						approved = " (approved)"
					}
					output.PrintMessage("  %s [%s]%s", p.User.DisplayName, p.Role, approved)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var title string
	var description string
	var source string
	var destination string
	var closeBranch bool
	var reviewers []string

	cmd := &cobra.Command{
		Use:   "create <workspace/repo-slug>",
		Short: "Create a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			body := map[string]interface{}{
				"title":               title,
				"description":         description,
				"close_source_branch": closeBranch,
				"source": map[string]interface{}{
					"branch": map[string]string{"name": source},
				},
			}
			if destination != "" {
				body["destination"] = map[string]interface{}{
					"branch": map[string]string{"name": destination},
				}
			}
			if len(reviewers) > 0 {
				revList := make([]map[string]string, len(reviewers))
				for i, r := range reviewers {
					revList[i] = map[string]string{"uuid": r}
				}
				body["reviewers"] = revList
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/pullrequests", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var pr PullRequest
			if err := json.Unmarshal(data, &pr); err != nil {
				return err
			}
			output.PrintMessage("Pull request #%d created: %s", pr.ID, pr.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "PR title (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "PR description")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Source branch (required)")
	cmd.Flags().StringVar(&destination, "destination", "", "Destination branch (defaults to main branch)")
	cmd.Flags().BoolVar(&closeBranch, "close-branch", false, "Close source branch after merge")
	cmd.Flags().StringSliceVarP(&reviewers, "reviewer", "r", nil, "Reviewer UUIDs")
	cmd.MarkFlagRequired("title")
	cmd.MarkFlagRequired("source")
	return cmd
}

func newCmdMerge() *cobra.Command {
	var strategy string
	var closeBranch bool
	var message string

	cmd := &cobra.Command{
		Use:   "merge <workspace/repo-slug> <pr-id>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"close_source_branch": closeBranch,
			}
			if strategy != "" {
				body["merge_strategy"] = strategy
			}
			if message != "" {
				body["message"] = message
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/merge", args[0], args[1])
			_, err = client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}
			output.PrintMessage("Pull request #%s merged.", args[1])
			return nil
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "", "Merge strategy (merge_commit, squash, fast_forward)")
	cmd.Flags().BoolVar(&closeBranch, "close-branch", true, "Close source branch after merge")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Merge commit message")
	return cmd
}

func newCmdApprove() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <workspace/repo-slug> <pr-id>",
		Short: "Approve a pull request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/approve", args[0], args[1])
			_, err = client.Post(path, "")
			if err != nil {
				return err
			}
			output.PrintMessage("Pull request #%s approved.", args[1])
			return nil
		},
	}
}

func newCmdUnapprove() *cobra.Command {
	return &cobra.Command{
		Use:   "unapprove <workspace/repo-slug> <pr-id>",
		Short: "Remove approval from a pull request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/approve", args[0], args[1])
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Approval removed from PR #%s.", args[1])
			return nil
		},
	}
}

func newCmdDecline() *cobra.Command {
	return &cobra.Command{
		Use:   "decline <workspace/repo-slug> <pr-id>",
		Short: "Decline a pull request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/decline", args[0], args[1])
			_, err = client.Post(path, "")
			if err != nil {
				return err
			}
			output.PrintMessage("Pull request #%s declined.", args[1])
			return nil
		},
	}
}

func newCmdComments() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "comments <workspace/repo-slug> <pr-id>",
		Short: "List comments on a pull request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/comments?pagelen=50", args[0], args[1])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var comments []struct {
				ID      int `json:"id"`
				Content struct {
					Raw string `json:"raw"`
				} `json:"content"`
				User struct {
					DisplayName string `json:"display_name"`
				} `json:"user"`
				CreatedOn string `json:"created_on"`
				Inline    *struct {
					Path string `json:"path"`
					From *int   `json:"from"`
					To   *int   `json:"to"`
				} `json:"inline"`
			}
			if err := json.Unmarshal(paginated.Values, &comments); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(comments)
				return nil
			}

			for _, c := range comments {
				loc := ""
				if c.Inline != nil {
					loc = fmt.Sprintf(" [%s", c.Inline.Path)
					if c.Inline.To != nil {
						loc += fmt.Sprintf(":%d", *c.Inline.To)
					}
					loc += "]"
				}
				output.PrintMessage("--- Comment #%d by %s (%s)%s ---", c.ID, c.User.DisplayName, c.CreatedOn[:10], loc)
				output.PrintMessage("%s\n", c.Content.Raw)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdComment() *cobra.Command {
	var body string
	var bodyFile string
	var useEditor bool
	var file string
	var line int

	cmd := &cobra.Command{
		Use:   "comment <workspace/repo-slug> <pr-id>",
		Short: "Add a comment to a pull request (supports inline comments on specific files/lines)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedBody, err := cmdutil.ResolveBody(
				body, bodyFile, useEditor,
				cmd.Flags().Changed("body"),
				cmd.Flags().Changed("body-file"),
				cmd.Flags().Changed("editor"),
			)
			if err != nil {
				return err
			}

			fileSet := cmd.Flags().Changed("file")
			lineSet := cmd.Flags().Changed("line")
			if fileSet != lineSet {
				return fmt.Errorf("--file and --line must be used together")
			}

			client, err := api.NewClient()
			if err != nil {
				return err
			}
			reqBody := map[string]interface{}{
				"content": map[string]string{"raw": resolvedBody},
			}
			if fileSet {
				reqBody["inline"] = map[string]interface{}{
					"path": file,
					"to":   line,
				}
			}
			jsonBody, _ := json.Marshal(reqBody)
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/comments", args[0], args[1])
			_, err = client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}
			if fileSet {
				output.PrintMessage("Inline comment added to PR #%s on %s:%d.", args[1], file, line)
			} else {
				output.PrintMessage("Comment added to PR #%s.", args[1])
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read body from file (use - for stdin)")
	cmd.Flags().BoolVarP(&useEditor, "editor", "e", false, "Open editor to compose comment")
	cmd.Flags().StringVarP(&file, "file", "f", "", "File path in the diff for inline comment")
	cmd.Flags().IntVarP(&line, "line", "l", 0, "Line number in the file for inline comment")
	return cmd
}

func newCmdDiff() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <workspace/repo-slug> <pr-id>",
		Short: "View pull request diff",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/diff", args[0], args[1])
			data, err := client.Get(path)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func newCmdActivity() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "activity <workspace/repo-slug> <pr-id>",
		Short: "View pull request activity log",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/pullrequests/%s/activity?pagelen=50", args[0], args[1])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			if jsonOut {
				var raw interface{}
				if err := json.Unmarshal(paginated.Values, &raw); err != nil {
					return err
				}
				output.PrintJSON(raw)
				return nil
			}

			// Activity is a heterogeneous list; render a summary table
			var activities []struct {
				Update *struct {
					State  string `json:"state"`
					Author struct {
						DisplayName string `json:"display_name"`
					} `json:"author"`
					Date string `json:"date"`
				} `json:"update"`
				Approval *struct {
					User struct {
						DisplayName string `json:"display_name"`
					} `json:"user"`
					Date string `json:"date"`
				} `json:"approval"`
				Comment *struct {
					User struct {
						DisplayName string `json:"display_name"`
					} `json:"user"`
					Content struct {
						Raw string `json:"raw"`
					} `json:"content"`
					CreatedOn string `json:"created_on"`
				} `json:"comment"`
			}
			if err := json.Unmarshal(paginated.Values, &activities); err != nil {
				return err
			}

			for _, a := range activities {
				switch {
				case a.Update != nil:
					date := a.Update.Date
					if len(date) > 10 {
						date = date[:10]
					}
					output.PrintMessage("[%s] %s changed state to %s", date, a.Update.Author.DisplayName, a.Update.State)
				case a.Approval != nil:
					date := a.Approval.Date
					if len(date) > 10 {
						date = date[:10]
					}
					output.PrintMessage("[%s] %s approved", date, a.Approval.User.DisplayName)
				case a.Comment != nil:
					date := a.Comment.CreatedOn
					if len(date) > 10 {
						date = date[:10]
					}
					output.PrintMessage("[%s] %s commented: %s", date, a.Comment.User.DisplayName, output.Truncate(a.Comment.Content.Raw, 80))
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
