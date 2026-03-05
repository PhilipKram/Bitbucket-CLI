package environment

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Environment struct {
	UUID            string `json:"uuid"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	EnvironmentType struct {
		Name string `json:"name"`
	} `json:"environment_type"`
	Rank     int `json:"rank"`
	Category struct {
		Name string `json:"name"`
	} `json:"category"`
	Lock *struct {
		Name string `json:"name"`
	} `json:"lock"`
	DeploymentGate *struct {
		Name string `json:"name"`
	} `json:"deployment_gate"`
}

func NewCmdEnvironment() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Manage deployment environments",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdView())
	cmd.AddCommand(newCmdCreate())
	cmd.AddCommand(newCmdDelete())

	return cmd
}

func newCmdList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List deployment environments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/environments?pagelen=25", args[0])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var environments []Environment
			if err := json.Unmarshal(paginated.Values, &environments); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(environments)
				return nil
			}

			table := output.NewTable("UUID", "NAME", "TYPE", "CATEGORY", "RANK", "LOCK")
			for _, e := range environments {
				lock := ""
				if e.Lock != nil {
					lock = e.Lock.Name
				}
				table.AddRow(
					e.UUID,
					e.Name,
					e.EnvironmentType.Name,
					e.Category.Name,
					fmt.Sprintf("%d", e.Rank),
					lock,
				)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdView() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "view <workspace/repo-slug> <env-uuid>",
		Short: "View environment details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/environments/%s", args[0], args[1])
			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var env Environment
			if err := json.Unmarshal(data, &env); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(env)
				return nil
			}

			lock := "none"
			if env.Lock != nil {
				lock = env.Lock.Name
			}
			gate := "none"
			if env.DeploymentGate != nil {
				gate = env.DeploymentGate.Name
			}

			output.PrintMessage("UUID:            %s", env.UUID)
			output.PrintMessage("Name:            %s", env.Name)
			output.PrintMessage("Slug:            %s", env.Slug)
			output.PrintMessage("Type:            %s", env.EnvironmentType.Name)
			output.PrintMessage("Category:        %s", env.Category.Name)
			output.PrintMessage("Rank:            %d", env.Rank)
			output.PrintMessage("Lock:            %s", lock)
			output.PrintMessage("Deployment Gate: %s", gate)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdCreate() *cobra.Command {
	var name string
	var envType string

	cmd := &cobra.Command{
		Use:   "create <workspace/repo-slug>",
		Short: "Create a deployment environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			body := map[string]interface{}{
				"name": name,
				"environment_type": map[string]string{
					"name": envType,
				},
			}

			jsonBody, _ := json.Marshal(body)
			path := fmt.Sprintf("/repositories/%s/environments", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var env Environment
			if err := json.Unmarshal(data, &env); err != nil {
				return err
			}
			output.PrintMessage("Environment '%s' created (UUID: %s)", env.Name, env.UUID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "Environment name (required)")
	cmd.Flags().StringVarP(&envType, "type", "t", "", "Environment type: Test, Staging, or Production (required)")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("type")
	return cmd
}

func newCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <workspace/repo-slug> <env-uuid>",
		Short: "Delete a deployment environment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/environments/%s", args[0], args[1])
			_, err = client.Delete(path)
			if err != nil {
				return err
			}
			output.PrintMessage("Environment '%s' deleted.", args[1])
			return nil
		},
	}
	return cmd
}
