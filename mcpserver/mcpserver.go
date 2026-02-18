// SPDX-License-Identifier: MIT

package mcpserver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
)

// projectDir is the directory to analyze when no path is provided.
var projectDir string

// Serve starts the MCP server on stdin/stdout for LLM integration.
// dir is the default directory to analyze; if empty, the current working directory is used.
func Serve(dir string) {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	dir, _ = filepath.Abs(dir)
	projectDir = dir

	s := server.NewMCPServer(
		"scc",
		"1.0.0",
		server.WithInstructions(fmt.Sprintf(
			"scc is a code line counter. The project directory available for analysis is: %s", dir)),
	)

	registerTools(s)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}
