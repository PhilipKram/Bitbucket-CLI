package mcp

import (
	"github.com/spf13/cobra"
)

func NewCmdMCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server",
	}

	cmd.AddCommand(newCmdServe())

	return cmd
}

func newCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement MCP server
			return nil
		},
	}

	return cmd
}
