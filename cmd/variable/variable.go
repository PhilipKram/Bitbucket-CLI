package variable

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

// Variable represents a Bitbucket pipeline variable.
type Variable struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// NewCmdVariable returns the top-level "variable" command with subcommands.
func NewCmdVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"var"},
		Short:   "Manage pipeline variables",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdGet())
	cmd.AddCommand(newCmdSet())
	cmd.AddCommand(newCmdUpdate())
	cmd.AddCommand(newCmdDelete())

	return cmd
}

// listVariables fetches all pipeline variables for a repository.
func listVariables(client *api.Client, repo string) ([]Variable, error) {
	path := fmt.Sprintf("/repositories/%s/pipelines_config/variables?pagelen=100", repo)
	data, err := client.Get(path)
	if err != nil {
		return nil, err
	}

	var paginated api.PaginatedResponse
	if err := json.Unmarshal(data, &paginated); err != nil {
		return nil, err
	}

	var variables []Variable
	if err := json.Unmarshal(paginated.Values, &variables); err != nil {
		return nil, err
	}
	return variables, nil
}

// findVariableByKey searches the variable list for a matching key.
func findVariableByKey(variables []Variable, key string) (*Variable, error) {
	for i := range variables {
		if variables[i].Key == key {
			return &variables[i], nil
		}
	}
	return nil, fmt.Errorf("variable with key %q not found", key)
}

func newCmdList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List pipeline variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			variables, err := listVariables(client, args[0])
			if err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(variables)
				return nil
			}

			table := output.NewTable("KEY", "VALUE", "SECURED", "UUID")
			for _, v := range variables {
				value := v.Value
				if v.Secured {
					value = "********"
				}
				table.AddRow(v.Key, value, fmt.Sprintf("%v", v.Secured), v.UUID)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdGet() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "get <workspace/repo-slug> <variable-key>",
		Short: "Get a pipeline variable by key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			variables, err := listVariables(client, args[0])
			if err != nil {
				return err
			}

			v, err := findVariableByKey(variables, args[1])
			if err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(v)
				return nil
			}

			value := v.Value
			if v.Secured {
				value = "********"
			}
			output.PrintMessage("Key:     %s", v.Key)
			output.PrintMessage("Value:   %s", value)
			output.PrintMessage("Secured: %v", v.Secured)
			output.PrintMessage("UUID:    %s", v.UUID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdSet() *cobra.Command {
	var key string
	var value string
	var secured bool

	cmd := &cobra.Command{
		Use:   "set <workspace/repo-slug>",
		Short: "Create a new pipeline variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			body := map[string]interface{}{
				"key":     key,
				"value":   value,
				"secured": secured,
			}
			jsonBody, _ := json.Marshal(body)

			path := fmt.Sprintf("/repositories/%s/pipelines_config/variables", args[0])
			data, err := client.Post(path, string(jsonBody))
			if err != nil {
				return err
			}

			var v Variable
			if err := json.Unmarshal(data, &v); err != nil {
				return err
			}

			output.PrintMessage("Variable %q created (UUID: %s)", v.Key, v.UUID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&key, "key", "k", "", "Variable key (required)")
	cmd.Flags().StringVarP(&value, "value", "v", "", "Variable value (required)")
	cmd.Flags().BoolVar(&secured, "secured", false, "Mark variable as secured")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("value")
	return cmd
}

func newCmdUpdate() *cobra.Command {
	var key string
	var value string
	var secured bool

	cmd := &cobra.Command{
		Use:   "update <workspace/repo-slug>",
		Short: "Update an existing pipeline variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			// List variables to find UUID by key
			variables, err := listVariables(client, args[0])
			if err != nil {
				return err
			}

			existing, err := findVariableByKey(variables, key)
			if err != nil {
				return err
			}

			body := map[string]interface{}{
				"key":     key,
				"value":   value,
				"secured": secured,
			}
			jsonBody, _ := json.Marshal(body)

			path := fmt.Sprintf("/repositories/%s/pipelines_config/variables/%s",
				args[0], url.PathEscape(existing.UUID))
			data, err := client.Put(path, string(jsonBody))
			if err != nil {
				return err
			}

			var v Variable
			if err := json.Unmarshal(data, &v); err != nil {
				return err
			}

			output.PrintMessage("Variable %q updated (UUID: %s)", v.Key, v.UUID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&key, "key", "k", "", "Variable key (required)")
	cmd.Flags().StringVarP(&value, "value", "v", "", "Variable value (required)")
	cmd.Flags().BoolVar(&secured, "secured", false, "Mark variable as secured")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("value")
	return cmd
}

func newCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <workspace/repo-slug> <variable-key>",
		Short: "Delete a pipeline variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.NewClient()
			if err != nil {
				return err
			}

			// List variables to find UUID by key
			variables, err := listVariables(client, args[0])
			if err != nil {
				return err
			}

			existing, err := findVariableByKey(variables, args[1])
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/pipelines_config/variables/%s",
				args[0], url.PathEscape(existing.UUID))
			_, err = client.Delete(path)
			if err != nil {
				return err
			}

			output.PrintMessage("Variable %q deleted.", existing.Key)
			return nil
		},
	}
	return cmd
}
