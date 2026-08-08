package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pcpl2/github-actions-versions-mcp/internal/github"
)

// connectClient spins up the server with a fake fetcher over an in-memory
// transport and returns a connected client session.
func connectClient(t *testing.T, f releaseFetcher) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	Register(server, f)

	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestE2EListTools(t *testing.T) {
	session := connectClient(t, &fakeFetcher{})

	res, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"latest_release", "list_releases", "check_workflow_actions"} {
		if !names[want] {
			t.Errorf("tool %q not registered; got %v", want, names)
		}
	}
}

func TestE2ELatestRelease(t *testing.T) {
	f := &fakeFetcher{latest: map[string]*github.Release{
		"actions/checkout": {TagName: "v4.2.2", HTMLURL: "https://example/v4.2.2"},
	}}
	session := connectClient(t, f)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "latest_release",
		Arguments: map[string]any{"owner": "actions", "repo": "checkout"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}
	text := contentText(res)
	if !strings.Contains(text, "v4.2.2") {
		t.Errorf("response does not mention tag v4.2.2: %s", text)
	}
}

func TestE2ELatestReleaseMissingArgs(t *testing.T) {
	session := connectClient(t, &fakeFetcher{})

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "latest_release",
		Arguments: map[string]any{"owner": "actions"}, // repo missing
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Errorf("expected IsError=true when repo is missing")
	}
}

func contentText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}
