package download

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/output"
)

type Download struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedOn string `json:"created_on"`
	Downloads int    `json:"downloads"`
	Links     struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

func NewCmdDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download",
		Short:   "Manage repository downloads",
		Aliases: []string{"downloads", "dl"},
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdUpload())
	cmd.AddCommand(newCmdGet())
	cmd.AddCommand(newCmdDelete())

	return cmd
}

func parseRepoArg(arg string) (string, string, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format %q, expected workspace/repo-slug", arg)
	}
	return parts[0], parts[1], nil
}

func newCmdList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list <workspace/repo-slug>",
		Short: "List repository downloads",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, repo, err := parseRepoArg(args[0])
			if err != nil {
				return err
			}

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/%s/downloads?pagelen=25",
				url.PathEscape(ws), url.PathEscape(repo))

			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var downloads []Download
			if err := json.Unmarshal(paginated.Values, &downloads); err != nil {
				return err
			}

			if jsonOut {
				output.PrintJSON(downloads)
				return nil
			}

			if len(downloads) == 0 {
				output.PrintMessage("No downloads found.")
				return nil
			}

			table := output.NewTable("NAME", "SIZE", "DOWNLOADS", "CREATED")
			for _, d := range downloads {
				created := ""
				if len(d.CreatedOn) >= 10 {
					created = d.CreatedOn[:10]
				}
				table.AddRow(d.Name, formatSize(d.Size), fmt.Sprintf("%d", d.Downloads), created)
			}
			table.Print()
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newCmdUpload() *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "upload <workspace/repo-slug>",
		Short: "Upload a file to repository downloads",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, repo, err := parseRepoArg(args[0])
			if err != nil {
				return err
			}

			if filePath == "" {
				return fmt.Errorf("--file is required")
			}

			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/%s/downloads",
				url.PathEscape(ws), url.PathEscape(repo))

			fileName := filepath.Base(filePath)
			_, err = client.PostMultipart(path, "files", fileName, f)
			if err != nil {
				return err
			}

			output.PrintMessage("Uploaded '%s' to %s/%s downloads.", fileName, ws, repo)
			return nil
		},
	}
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the file to upload (required)")
	cmd.MarkFlagRequired("file")
	return cmd
}

func newCmdGet() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "get <workspace/repo-slug> <filename>",
		Short: "Download a file from repository downloads",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, repo, err := parseRepoArg(args[0])
			if err != nil {
				return err
			}
			filename := args[1]

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			// List downloads to find the file's download link
			path := fmt.Sprintf("/repositories/%s/%s/downloads?pagelen=100",
				url.PathEscape(ws), url.PathEscape(repo))

			data, err := client.Get(path)
			if err != nil {
				return err
			}

			var paginated api.PaginatedResponse
			if err := json.Unmarshal(data, &paginated); err != nil {
				return err
			}

			var downloads []Download
			if err := json.Unmarshal(paginated.Values, &downloads); err != nil {
				return err
			}

			var found *Download
			for i := range downloads {
				if downloads[i].Name == filename {
					found = &downloads[i]
					break
				}
			}
			if found == nil {
				return fmt.Errorf("file %q not found in downloads", filename)
			}

			fileData, err := client.GetRaw(found.Links.Self.Href)
			if err != nil {
				return err
			}

			dest := outputPath
			if dest == "" {
				dest = filename
			}

			if err := os.WriteFile(dest, fileData, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			output.PrintMessage("Downloaded '%s' (%s)", filename, formatSize(int64(len(fileData))))
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (defaults to filename)")
	return cmd
}

func newCmdDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <workspace/repo-slug> <filename>",
		Short: "Delete a file from repository downloads",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, repo, err := parseRepoArg(args[0])
			if err != nil {
				return err
			}
			filename := args[1]

			client, err := api.NewClient()
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/repositories/%s/%s/downloads/%s",
				url.PathEscape(ws), url.PathEscape(repo), url.PathEscape(filename))

			_, err = client.Delete(path)
			if err != nil {
				return err
			}

			output.PrintMessage("Deleted '%s' from %s/%s downloads.", filename, ws, repo)
			return nil
		},
	}
	return cmd
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
