package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{"id": {"type": "number"}}, Required: []string{"id"}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "name", Description: "Name", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := s.GetToolHandler("get_user")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "get_user", Params: mcp.ToolParams{Arguments: map[string]any{"id": 123}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Name:test-user", result.Text)
	})

	t.Run("POST request", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "create_user",
			Description: "Create a user",
			Request:     hyancieMCP.RequestConfig{Method: "POST", URL: mockAPIServer.URL + "/create-user"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{"name": {"type": "string"}}, Required: []string{"name"}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "message", Description: "Status", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := s.GetToolHandler("create_user")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "create_user", Params: mcp.ToolParams{Arguments: map[string]any{"name": "new-user"}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Status:User created", result.Text)
	})

	t.Run("Bearer Token Auth", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:       "get_secure_data",
			Description:    "Get secure data",
			Request:        hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/secure-data"},
			Authentication: &hyancieMCP.AuthenticationConfig{Type: "bearer", Token: "test-token"},
			InputSchema:    mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "data", Description: "Data", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := s.GetToolHandler("get_secure_data")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "get_secure_data", Params: mcp.ToolParams{Arguments: map[string]any{}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Data:secret", result.Text)
	})

	t.Run("API Key Header Auth", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:       "get_api_key_data",
			Description:    "Get api key data",
			Request:        hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/api-key-data"},
			Authentication: &hyancieMCP.AuthenticationConfig{Type: "header", Name: "X-API-Key", Value: "test-api-key"},
			InputSchema:    mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
			OutputMapping: []hyancieMCP.OutputMap{
				{JsonKey: "data", Description: "Data", Type: "primitive"},
			},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := s.GetToolHandler("get_api_key_data")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "get_api_key_data", Params: mcp.ToolParams{Arguments: map[string]any{}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Data:api-key-secret", result.Text)
	})

	t.Run("Complex Response Parsing", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_complex",
			Description: "Get complex data",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/complex-response"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
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

		handler := s.GetToolHandler("get_complex")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "get_complex", Params: mcp.ToolParams{Arguments: map[string]any{}}}
		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		expected := "Results:[项1:{Name:A, Value:1} | 项2:{Name:B, Value:2}]|Count:3"
		assert.Equal(t, expected, result.Text)
	})

	t.Run("HTTP Error", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_not_found",
			Description: "Get something that doesn't exist",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/not-found"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := s.GetToolHandler("get_not_found")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "get_not_found", Params: mcp.ToolParams{Arguments: map[string]any{}}}
		_, err = handler(context.Background(), req)

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "request failed with status 404"))
	})

	t.Run("Malformed JSON Response", func(t *testing.T) {
		toolConfig := hyancieMCP.GenericToolConfig{
			ToolName:    "get_malformed",
			Description: "Get malformed JSON",
			Request:     hyancieMCP.RequestConfig{Method: "GET", URL: mockAPIServer.URL + "/malformed-json"},
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]any{}},
		}
		hyancieMCP.Config.McpTools = []hyancieMCP.GenericToolConfig{toolConfig}

		s := server.NewMCPServer("test", "1.0")
		err := AddGenericTools(s)
		require.NoError(t, err)

		handler := s.GetToolHandler("get_malformed")
		require.NotNil(t, handler)

		req := mcp.CallToolRequest{Tool: "get_malformed", Params: mcp.ToolParams{Arguments: map[string]any{}}}
		_, err = handler(context.Background(), req)

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to decode json response"))
	})
}