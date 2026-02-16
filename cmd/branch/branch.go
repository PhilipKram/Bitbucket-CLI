package branch

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Branch struct {
	Name   string `json:"name"`
	Target struct {
		Hash    string `json:"hash"`
		Date    string `json:"date"`
		Message string `json:"message"`
		Author  struct {
			Raw string `json:"raw"`
		} `json:"author"`
	} `json:"target"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type Tag struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
		Date string `json:"date"`
	} `json:"target"`
	Message string `json:"message"`
	Links   struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type BranchRestriction struct {
	ID      int    `json:"id"`
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Value   *int   `json:"value"`
}

func NewCmdBranch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage branches and tags",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdDelete())
	cmd.AddCommand(newCmdTags())
	cmd.AddCommand(newCmdTagCreate())
	cmd.AddCommand(newCmdTagDelete())
	cmd.AddCommand(newCmdRestrictions())

	return cmd
}

func newCmdList() *cobra.Command {
	var jsonOut bool
	var page int

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List branches",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/refs/branches?pagelen=25&page=%d", args[0], page)
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var branches []Branch
			if err := json.Unmarshal(paginated.Values, &branches); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(branches)
				return nil
			}

			table := output.NewTable("NAME", "HASH", "AUTHOR", "DATE", "MESSAGE")
			for _, b := range branches {
				date := ""
				if len(b.Target.Date) >= 10 {
					date = b.Target.Date[:10]
				}
				table.AddRow(
					b.Name,
					b.Target.Hash[:12],
					output.Truncate(b.Target.Author.Raw, 25),
					date,
					output.Truncate(b.Target.Message, 40),
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "create <workspace/repo-slug> <branch-name>",
		Short: "Create a branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"name": args[1],
				"target": map[string]string{
					"hash": target,
				},
			}
			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/refs/branches", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var branch Branch
			if err := json.Unmarshal(data, &branch); err != nil {
				return err
			}
			output.PrintMessage("Branch '%s' created at %s.", branch.Name, branch.Target.Hash[:12])
			return nil
		},
	}
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target commit hash (required)")
	cmd.MarkFlagRequired("target")
	return cmd
}

func newCmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <workspace/repo-slug> <branch-name>",
		Short: "Delete a branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/refs/branches/%s", args[0], url.PathEscape(args[1]))
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Branch '%s' deleted.", args[1])
			return nil
		},
	}
}

func newCmdTags() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "tags <workspace/repo-slug>",
		Short: "List tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/refs/tags?pagelen=25", args[0])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var tags []Tag
			if err := json.Unmarshal(paginated.Values, &tags); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(tags)
				return nil
			}

			table := output.NewTable("NAME", "HASH", "DATE", "MESSAGE")
			for _, t := range tags {
				date := ""
				if len(t.Target.Date) >= 10 {
					date = t.Target.Date[:10]
				}
				table.AddRow(t.Name, t.Target.Hash[:12], date, output.Truncate(t.Message, 50))
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdTagCreate() *cobra.Command {
	var target string
	var message string

	cmd := &cobra.Command{
		Use:   "tag-create <workspace/repo-slug> <tag-name>",
		Short: "Create a tag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			body := map[string]interface{}{
				"name": args[1],
				"target": map[string]string{
					"hash": target,
				},
			}
			if message != "" {
				body["message"] = message
			}
			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/refs/tags", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var tag Tag
			if err := json.Unmarshal(data, &tag); err != nil {
				return err
			}
			output.PrintMessage("Tag '%s' created at %s.", tag.Name, tag.Target.Hash[:12])
			return nil
		},
	}
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target commit hash (required)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Tag message")
	cmd.MarkFlagRequired("target")
	return cmd
}

func newCmdTagDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "tag-delete <workspace/repo-slug> <tag-name>",
		Short: "Delete a tag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/refs/tags/%s", args[0], url.PathEscape(args[1]))
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Tag '%s' deleted.", args[1])
			return nil
		},
	}
}

func newCmdRestrictions() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "restrictions <workspace/repo-slug>",
		Short: "List branch restrictions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/repositories/%s/branch-restrictions?pagelen=50", args[0])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var restrictions []BranchRestriction
			if err := json.Unmarshal(paginated.Values, &restrictions); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(restrictions)
				return nil
			}

			table := output.NewTable("ID", "KIND", "PATTERN")
			for _, r := range restrictions {
				table.AddRow(fmt.Sprintf("%d", r.ID), r.Kind, r.Pattern)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
