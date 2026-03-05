package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	internalapi "github.com/PhilipKram/bitbucket-cli/internal/api"
)

// NewCmdAPI returns a cobra.Command that makes authenticated Bitbucket API requests.
func NewCmdAPI() *cobra.Command {
	var method string
	var body string
	var headers []string
	var fields []string

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated Bitbucket API request",
		Long: `Make an authenticated HTTP request to the Bitbucket Cloud API.

The endpoint is relative to the Bitbucket API base URL
(https://api.bitbucket.org/2.0). For example:

  bb api /repositories/workspace/repo-slug
  bb api /user
  bb api -X POST /repositories/workspace/repo/pullrequests -b '{"title":"My PR"}'

Use --field to construct a JSON body from key=value pairs:

  bb api -X POST /repositories/workspace/repo/pullrequests \
    -f title="My PR" -f source.branch.name=feature`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoint := args[0]

			// Ensure the endpoint starts with /
			if !strings.HasPrefix(endpoint, "/") {
				endpoint = "/" + endpoint
			}

			// Build body from --field flags if --body is not set
			requestBody := body
			if requestBody == "" && len(fields) > 0 {
				fieldBody, err := buildFieldBody(fields)
				if err != nil {
					return err
				}
				requestBody = fieldBody
			}

			// Validate headers
			if len(headers) > 0 {
				return fmt.Errorf("--header is not yet supported: custom headers cannot be applied to requests")
			}

			client, err := internalapi.NewClient()
			if err != nil {
				return err
			}

			var data []byte
			switch strings.ToUpper(method) {
			case "GET":
				data, err = client.Get(endpoint)
			case "POST":
				data, err = client.Post(endpoint, requestBody)
			case "PUT":
				data, err = client.Put(endpoint, requestBody)
			case "DELETE":
				data, err = client.Delete(endpoint)
			default:
				return fmt.Errorf("unsupported HTTP method: %s (supported: GET, POST, PUT, DELETE)", method)
			}
			if err != nil {
				return err
			}

			if data == nil {
				return nil
			}

			// Pretty-print JSON if possible, otherwise print raw
			var prettyJSON json.RawMessage
			if err := json.Unmarshal(data, &prettyJSON); err == nil {
				formatted, err := json.MarshalIndent(prettyJSON, "", "  ")
				if err == nil {
					fmt.Fprintln(cmd.OutOrStdout(), string(formatted))
					return nil
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method (GET, POST, PUT, DELETE)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Request body (JSON string)")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Additional headers in 'Key: Value' format (repeatable)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Body fields in key=value format, combined into a JSON object (repeatable)")

	return cmd
}

// buildFieldBody combines key=value pairs into a JSON object.
// Supports nested keys using dot notation (e.g., "source.branch.name=main").
func buildFieldBody(fields []string) (string, error) {
	obj := make(map[string]interface{})

	for _, f := range fields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid field format %q: expected key=value", f)
		}
		key, value := parts[0], parts[1]
		setNestedField(obj, key, value)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fields to JSON: %w", err)
	}
	return string(data), nil
}

// setNestedField sets a value in a nested map using dot-separated keys.
func setNestedField(obj map[string]interface{}, key, value string) {
	parts := strings.Split(key, ".")
	current := obj
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}
		if nested, ok := current[part].(map[string]interface{}); ok {
			current = nested
		} else {
			// Overwrite non-map value with a new map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
}

// parseHeaders parses "Key: Value" strings into a map.
func parseHeaders(headers []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format %q: expected 'Key: Value'", h)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid header format %q: empty key", h)
		}
		result[key] = value
	}
	return result, nil
}
