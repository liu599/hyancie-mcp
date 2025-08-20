package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func joinContents(contents []mcp.Content) string {
	var parts []string
	for _, c := range contents {
		if textContent, ok := c.(mcp.TextContent); ok {
			parts = append(parts, textContent.Text)
		}
	}
	return strings.Join(parts, "")
}

type toolHandler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)

// getToolHandler uses reflection to access the private toolHandlers map in the MCPServer.
func getToolHandler(s *server.MCPServer, toolName string) toolHandler {
	serverValue := reflect.ValueOf(s).Elem()
	handlersField := serverValue.FieldByName("toolHandlers")

	// Use unsafe to access the unexported field.
	// This is necessary because the field is not exported.
	handlersFieldPtr := unsafe.Pointer(handlersField.UnsafeAddr())
	handlersMap := *(*map[string]toolHandler)(handlersFieldPtr)

	if handlersMap == nil {
		return nil
	}

	return handlersMap[toolName]
}

// TestMain sets up a mock server for all tests in this package.
func TestAddGenericTools(t *testing.T) {
	// Mock server that will act as the external API
	mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/get-user":
			fmt.Fprintln(w, `{"name": "test-user", "id": 123}`)
		case "/create-user":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(r.Body)
			fmt.Fprintf(w, `{"message": "User created", "received": %s}`, string(body))
		case "/secure-data":
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			fmt.Fprintln(w, `{"data": "secret"}`)
		case "/api-key-data":
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "test-api-key" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			fmt.Fprintln(w, `{"data": "api-key-secret"}`)
		case "/complex-response":
			fmt.Fprintln(w, `{
				"results": [
					{"item": {"name": "A", "value": 1}},
					{"item": {"name": "B", "value": 2}},
					{"item": {"name": "C", "value": 3}}
				],
				"metadata": {"count": 3}
			}`)
		case "/not-found":
			http.Error(w, "Not Found", http.StatusNotFound)
		case "/malformed-json":
			fmt.Fprintln(w, `{"key": "value"`) // Intentionally malformed
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockAPIServer.Close()

	// --- Test Cases ---

	t.Run("GET request", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_user",
			Description: "Get user info",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/get-user?id={id}"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"id": map[string]string{"type": "number"}}, Required: []string{"id"}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "name", Description: "Name", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := getToolHandler(s, "get_user")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"id": 123}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Name:test-user", joinContents(result.Content))
	})

	t.Run("POST request", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "create_user",
			Description: "Create a user",
			Request:     hyancieMCP.RequestConfig{Method: "POST", URL: mockAPIServer.URL + "/create-user"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"name": map[string]string{"type": "string"}}, Required: []string{"name"}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "message", Description: "Status", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := getToolHandler(s, "create_user")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"name": "new-user"}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Status:User created", joinContents(result.Content))
	})

	t.Run("Complex Response Parsing", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_complex",
			Description: "Get complex data",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/complex-response"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
			OutputMapping: []hyancieMCP.OutputMap{
				{
					JsonKey:     "results",
					Description: "Results",
					Type:        "array",
					Limit:       2, // Test limit
					Items: []hyancieMCP.OutputMap{
						{JsonKey: "item.name", Description: "Name", Type: "primitive"},
						{JsonKey: "item.value", Description: "Value", Type: "primitive"},
					},
				},
				{JsonKey: "metadata.count", Description: "Count", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := getToolHandler(s, "get_complex")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		expected := "Results:[项1:{Name:A, Value:1} | 项2:{Name:B, Value:2}]|Count:3"
		assert.Equal(t, expected, joinContents(result.Content))
	})

	t.Run("HTTP Error", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_not_found",
			Description: "Get something that doesn't exist",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/not-found"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := getToolHandler(s, "get_not_found")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
		_, err = handler(context.Background(), req)

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "request failed with status 404"))
	})

	t.Run("Malformed JSON Response", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_malformed",
			Description: "Get malformed JSON",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/malformed-json"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := getToolHandler(s, "get_malformed")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, `{"key": "value"`, joinContents(result.Content))
	})

	t.Run("GET request with Chinese characters in URL", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_user_with_chinese_name",
			Description: "Get user info with Chinese name",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/get-user?name={name}"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{"name": map[string]string{"type": "string"}}, Required: []string{"name"}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "name", Description: "Name", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := getToolHandler(s, "get_user_with_chinese_name")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]interface{}{"name": "张三"}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Name:test-user", joinContents(result.Content)) // The mock server returns a fixed user, we just check the call succeeds.
	})
}
