package issue

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/cmdutil"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Issue struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Priority  string `json:"priority"`
	Kind      string `json:"kind"`
	Content   struct {
		Raw string `json:"raw"`
	} `json:"content"`
	Reporter struct {
		DisplayName string `json:"display_name"`
	} `json:"reporter"`
	Assignee *struct {
		DisplayName string `json:"display_name"`
	} `json:"assignee"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Votes     int    `json:"votes"`
	Component *struct {
		Name string `json:"name"`
	} `json:"component"`
	Milestone *struct {
		Name string `json:"name"`
	} `json:"milestone"`
	Version *struct {
		Name string `json:"name"`
	} `json:"version"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

func NewCmdIssue() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues (issue tracker)",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdEdit())
	cmd.AddCommand(newCmdDelete())
	cmd.AddCommand(newCmdComments())
	cmd.AddCommand(newCmdComment())
	cmd.AddCommand(newCmdVote())
	cmd.AddCommand(newCmdWatch())

	return cmd
}

func newCmdList() *cobra.Command {
	var state string
	var page int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List issues",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/issues?pagelen=25&page=%d", args[0], page)
			if state != "" {
				path += fmt.Sprintf("&q=state%%3D%%22%s%%22", url.QueryEscape(state))
			}

			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var issues []Issue
			if err := json.Unmarshal(paginated.Values, &issues); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(issues)
				return nil
			}

			table := output.NewTable("ID", "TITLE", "STATE", "PRIORITY", "KIND", "ASSIGNEE")
			for _, i := range issues {
				assignee := "–"
				if i.Assignee != nil {
					assignee = i.Assignee.DisplayName
				}
				table.AddRow(
					fmt.Sprintf("#%d", i.ID),
					output.Truncate(i.Title, 50),
					i.State,
					i.Priority,
					i.Kind,
					assignee,
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().StringVarP(&state, "state", "s", "", "Filter by state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <workspace/repo-slug> <issue-id>",
		Short: "View issue details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/issues/%s", args[0], args[1])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var issue Issue
			if err := json.Unmarshal(data, &issue); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(issue)
				return nil
			}

			assignee := "–"
			if issue.Assignee != nil {
				assignee = issue.Assignee.DisplayName
			}
			output.PrintMessage("Issue #%d: %s", issue.ID, issue.Title)
			output.PrintMessage("State:    %s", issue.State)
			output.PrintMessage("Priority: %s", issue.Priority)
			output.PrintMessage("Kind:     %s", issue.Kind)
			output.PrintMessage("Reporter: %s", issue.Reporter.DisplayName)
			output.PrintMessage("Assignee: %s", assignee)
			output.PrintMessage("Votes:    %d", issue.Votes)
			output.PrintMessage("Created:  %s", issue.CreatedOn)
			output.PrintMessage("Updated:  %s", issue.UpdatedOn)
			output.PrintMessage("URL:      %s", issue.Links.HTML.Href)
			if issue.Component != nil {
				output.PrintMessage("Component: %s", issue.Component.Name)
			}
			if issue.Milestone != nil {
				output.PrintMessage("Milestone: %s", issue.Milestone.Name)
			}
			if issue.Version != nil {
				output.PrintMessage("Version:   %s", issue.Version.Name)
			}
			if issue.Content.Raw != "" {
				output.PrintMessage("\nDescription:\n%s", issue.Content.Raw)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var title string
	var content string
	var kind string
	var priority string

	cmd := &cobra.Command{
		Use:   "create <workspace/repo-slug>",
		Short: "Create an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"title":    title,
				"kind":     kind,
				"priority": priority,
				"content":  map[string]string{"raw": content},
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/issues", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var issue Issue
			if err := json.Unmarshal(data, &issue); err != nil {
				return err
			}
			output.PrintMessage("Issue #%d created: %s", issue.ID, issue.Links.HTML.Href)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "Issue title (required)")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Issue description")
	cmd.Flags().StringVarP(&kind, "kind", "k", "bug", "Issue kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVar(&priority, "priority", "major", "Priority (trivial, minor, major, critical, blocker)")
	cmd.MarkFlagRequired("title")
	return cmd
}

func newCmdEdit() *cobra.Command {
	var title string
	var state string
	var priority string
	var kind string

	cmd := &cobra.Command{
		Use:   "edit <workspace/repo-slug> <issue-id>",
		Short: "Edit an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{}
			if title != "" {
				body["title"] = title
			}
			if state != "" {
				body["state"] = state
			}
			if priority != "" {
				body["priority"] = priority
			}
			if kind != "" {
				body["kind"] = kind
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/issues/%s", args[0], args[1])
			_, err = client.Put(path, string(jsonBody))
			if err != nil {
				return err
			}
			output.PrintMessage("Issue #%s updated.", args[1])
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "New title")
	cmd.Flags().StringVarP(&state, "state", "s", "", "New state")
	cmd.Flags().StringVar(&priority, "priority", "", "New priority")
	cmd.Flags().StringVarP(&kind, "kind", "k", "", "New kind")
	return cmd
}

func newCmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <workspace/repo-slug> <issue-id>",
		Short: "Delete an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/issues/%s", args[0], args[1])
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Issue #%s deleted.", args[1])
			return nil
		},
	}
}

func newCmdComments() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "comments <workspace/repo-slug> <issue-id>",
		Short: "List issue comments",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/issues/%s/comments?pagelen=50", args[0], args[1])
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
			}
			if err := json.Unmarshal(paginated.Values, &comments); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(comments)
				return nil
			}

			for _, c := range comments {
				output.PrintMessage("--- Comment #%d by %s (%s) ---", c.ID, c.User.DisplayName, c.CreatedOn[:10])
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

	cmd := &cobra.Command{
		Use:   "comment <workspace/repo-slug> <issue-id>",
		Short: "Add a comment to an issue",
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

			client, err := api.NewClient()
			if err != nil {
				return err
			}
			reqBody := map[string]interface{}{
				"content": map[string]string{"raw": resolvedBody},
			}
			jsonBody, _ := json.Marshal(reqBody)
			path := fmt.Sprintf("/repositories/%s/issues/%s/comments", args[0], args[1])
			_, err = client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}
			output.PrintMessage("Comment added to issue #%s.", args[1])
			return nil
		},
	}
	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read body from file (use - for stdin)")
	cmd.Flags().BoolVarP(&useEditor, "editor", "e", false, "Open editor to compose comment")
	return cmd
}

func newCmdVote() *cobra.Command {
	return &cobra.Command{
		Use:   "vote <workspace/repo-slug> <issue-id>",
		Short: "Vote on an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/issues/%s/vote", args[0], args[1])
			_, err = client.Put(path, "")
			if err != nil {
				return err
			}
			output.PrintMessage("Voted on issue #%s.", args[1])
			return nil
		},
	}
}

func newCmdWatch() *cobra.Command {
	return &cobra.Command{
		Use:   "watch <workspace/repo-slug> <issue-id>",
		Short: "Watch an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/issues/%s/watch", args[0], args[1])
			_, err = client.Put(path, "")
			if err != nil {
				return err
			}
			output.PrintMessage("Now watching issue #%s.", args[1])
			return nil
		},
	}
}
