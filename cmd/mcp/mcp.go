package mcp

import (
	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/mcp"
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
		Long: `Start an MCP (Model Context Protocol) server on stdio.

The server exposes Bitbucket CLI capabilities as MCP tools that can be
invoked by AI agents and LLM-powered development tools.

The server communicates via JSON-RPC 2.0 over stdin/stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create MCP server
			server := mcp.NewServer(
				"bb-mcp",
				"1.0.0",
				"Bitbucket CLI MCP server - exposes bb commands as MCP tools",
			)

			// Create tool registry and register default tools
			registry := mcp.NewToolRegistry()
			if err := mcp.RegisterDefaultTools(registry); err != nil {
				return err
			}

			// Set registry on server
			server.SetRegistry(registry)

			return server.Start()
		},
	}

	return cmd
}
