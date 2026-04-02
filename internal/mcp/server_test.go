package mcp

import (
	"context"
	"strings"
	"testing"
)

type stubTool struct{}

func (stubTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "echo_tool",
		Description: "Echoes arguments",
		InputSchema: map[string]any{"type": "object"},
	}
}

func (stubTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	return ToolResult{
		Content:           []ToolContent{{Type: "text", Text: "ok"}},
		StructuredContent: args,
	}, nil
}

func TestServerServesInitializeAndToolCalls(t *testing.T) {
	t.Parallel()

	server := NewServer("omnethdb-mcp", "test", []Tool{stubTool{}})

	input := "" +
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`) +
		frame(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`) +
		frame(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo_tool","arguments":{"hello":"world"}}}`)

	var output strings.Builder
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Serve returned unexpected error: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("expected initialize response, got %s", got)
	}
	if !strings.Contains(got, `"name":"echo_tool"`) {
		t.Fatalf("expected tools/list response, got %s", got)
	}
	if !strings.Contains(got, `"hello":"world"`) {
		t.Fatalf("expected tool call structured content, got %s", got)
	}
}

func TestServerSupportsPlainJSONRPCMode(t *testing.T) {
	t.Parallel()

	server := NewServer("omnethdb-mcp", "test", []Tool{stubTool{}})

	input := "" +
		"{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-11-25\"}}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n"

	var output strings.Builder
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Serve returned unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two plain JSON lines, got %q", output.String())
	}
	if !strings.Contains(lines[0], `"protocolVersion":"2025-11-25"`) {
		t.Fatalf("expected initialize response in plain mode, got %s", lines[0])
	}
	if !strings.Contains(lines[1], `"name":"echo_tool"`) {
		t.Fatalf("expected tools/list response in plain mode, got %s", lines[1])
	}
}

func frame(payload string) string {
	return "Content-Length: " + strconvItoa(len(payload)) + "\r\n\r\n" + payload
}

func strconvItoa(v int) string {
	if v == 0 {
		return "0"
	}
	out := make([]byte, 0, 12)
	for v > 0 {
		out = append([]byte{byte('0' + v%10)}, out...)
		v /= 10
	}
	return string(out)
}
