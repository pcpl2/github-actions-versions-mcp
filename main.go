// Command gha-mcp is a Model Context Protocol server that reports the latest
// released versions of GitHub Actions, using the GitHub REST API.
//
// It speaks MCP over stdio, so it can be launched directly by MCP clients such
// as Claude Desktop and Claude Code.
//
// Set the GITHUB_TOKEN environment variable to raise the GitHub API rate limit
// from 60 to 5000 requests per hour.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pcpl2/github-actions-versions-mcp/internal/github"
	"github.com/pcpl2/github-actions-versions-mcp/internal/tools"
)

// version is the release version, injected at build time via
// -ldflags "-X main.version=...". It stays "dev" for plain `go build`.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "github-actions-mcp %s\n", version)
		return
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "github-actions-mcp",
		Version: version,
	}, nil)

	tools.Register(server, github.NewClient())

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("github-actions-mcp: %v", err)
	}
}
