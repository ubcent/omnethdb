package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type wireMode int

const (
	wireModeUnknown wireMode = iota
	wireModeFramed
	wireModePlain
)

type Tool interface {
	Definition() ToolDefinition
	Call(context.Context, map[string]any) (ToolResult, error)
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type ToolResult struct {
	Content           []ToolContent `json:"content,omitempty"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Server struct {
	serverName    string
	serverVersion string
	tools         map[string]Tool
}

func NewServer(serverName string, serverVersion string, tools []Tool) *Server {
	indexed := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		indexed[tool.Definition().Name] = tool
	}
	return &Server{
		serverName:    serverName,
		serverVersion: serverVersion,
		tools:         indexed,
	}
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	mode := wireModeUnknown

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, detectedMode, err := readMessage(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				_ = writer.Flush()
				return nil
			}
			return err
		}

		var req rpcRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			if err := writeFrame(writer, rpcErrorResponse{JSONRPC: "2.0", Error: rpcError{Code: -32700, Message: "parse error"}}); err != nil {
				return err
			}
			if err := writer.Flush(); err != nil {
				return err
			}
			continue
		}
		if mode == wireModeUnknown {
			mode = detectedMode
		}

		resp, ok := s.handleRequest(ctx, req)
		if !ok {
			continue
		}
		if err := writeMessage(writer, resp, mode); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req rpcRequest) (any, bool) {
	switch req.Method {
	case "initialize":
		version := "2025-03-26"
		if requested, _ := req.paramString("protocolVersion"); strings.TrimSpace(requested) != "" {
			version = requested
		}
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": version,
				"capabilities": map[string]any{
					"tools":     map[string]any{"listChanged": false},
					"resources": map[string]any{"subscribe": false, "listChanged": false},
					"prompts":   map[string]any{"listChanged": false},
					"logging":   map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    s.serverName,
					"version": s.serverVersion,
				},
			},
		}, true
	case "notifications/initialized":
		return nil, false
	case "$/cancelRequest":
		return nil, false
	case "logging/setLevel":
		return rpcSuccessResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}, true
	case "ping":
		return rpcSuccessResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}, true
	case "tools/list":
		definitions := make([]ToolDefinition, 0, len(s.tools))
		for _, tool := range s.tools {
			definitions = append(definitions, tool.Definition())
		}
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": definitions},
		}, true
	case "resources/list":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"resources": []any{}},
		}, true
	case "resources/templates/list":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"resourceTemplates": []any{}},
		}, true
	case "resources/read":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"contents": []any{}},
		}, true
	case "prompts/list":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"prompts": []any{}},
		}, true
	case "prompts/get":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"description": "",
				"messages":    []any{},
			},
		}, true
	case "roots/list":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"roots": []any{}},
		}, true
	case "completion/complete":
		return rpcSuccessResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"completion": map[string]any{
					"values":  []string{},
					"hasMore": false,
				},
			},
		}, true
	case "tools/call":
		name, _ := req.paramString("name")
		tool := s.tools[name]
		if tool == nil {
			return rpcSuccessResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolResult{
					IsError: true,
					Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("unknown tool %q", name)}},
				},
			}, true
		}
		args := req.paramObject("arguments")
		result, err := tool.Call(ctx, args)
		if err != nil {
			return rpcSuccessResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolResult{
					IsError: true,
					Content: []ToolContent{{Type: "text", Text: err.Error()}},
				},
			}, true
		}
		return rpcSuccessResponse{JSONRPC: "2.0", ID: req.ID, Result: result}, true
	default:
		if strings.HasPrefix(req.Method, "notifications/") {
			return nil, false
		}
		return rpcErrorResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcError{Code: -32601, Message: "method not found"},
		}, true
	}
}

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

func (r rpcRequest) paramString(key string) (string, bool) {
	if r.Params == nil {
		return "", false
	}
	raw, ok := r.Params[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok
}

func (r rpcRequest) paramObject(key string) map[string]any {
	if r.Params == nil {
		return map[string]any{}
	}
	raw, ok := r.Params[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	if obj, ok := raw.(map[string]any); ok {
		return obj
	}
	return map[string]any{}
}

type rpcSuccessResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result"`
}

type rpcErrorResponse struct {
	JSONRPC string   `json:"jsonrpc"`
	ID      any      `json:"id,omitempty"`
	Error   rpcError `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func readMessage(reader *bufio.Reader) ([]byte, wireMode, error) {
	peeked, err := reader.Peek(1)
	if err != nil {
		return nil, wireModeUnknown, err
	}
	if len(peeked) > 0 && (peeked[0] == '{' || peeked[0] == '[') {
		payload, err := readPlainMessage(reader)
		return payload, wireModePlain, err
	}
	payload, err := readFrame(reader)
	return payload, wireModeFramed, err
}

func readPlainMessage(reader *bufio.Reader) ([]byte, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && len(bytes.TrimSpace(line)) > 0 {
			return bytes.TrimSpace(line), nil
		}
		return nil, err
	}
	return bytes.TrimSpace(line), nil
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			var parsed int
			if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &parsed); err != nil {
				return nil, err
			}
			contentLength = parsed
		}
	}
	if contentLength < 0 {
		return nil, io.ErrUnexpectedEOF
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeMessage(writer *bufio.Writer, message any, mode wireMode) error {
	switch mode {
	case wireModePlain:
		return writePlainMessage(writer, message)
	default:
		return writeFrame(writer, message)
	}
}

func writePlainMessage(writer *bufio.Writer, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	return writer.WriteByte('\n')
}

func writeFrame(writer *bufio.Writer, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))); err != nil {
		return err
	}
	_, err = io.Copy(writer, bytes.NewReader(payload))
	return err
}
