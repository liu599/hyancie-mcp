package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	hyancieMCP "github.com/liu599/hyancie"
	"io"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AddGenericTools registers all tools defined in the global config with the MCP server.
func AddGenericTools(s *server.MCPServer) error {
	configs := hyancieMCP.Config.McpTools

	for _, config := range configs {
		// Capture the current config for the handler closure
		currentConfig := config

		tool := mcp.Tool{
			Name:        currentConfig.ToolName,
			Description: currentConfig.Description,
			InputSchema: currentConfig.InputSchema,
		}

		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.Params.Arguments

			var req *http.Request
			var err error
			method := strings.ToUpper(currentConfig.Request.Method)
			url := currentConfig.Request.URL

			if method == "POST" || method == "PUT" {
				// For POST/PUT, send args as JSON body
				jsonBody, err := json.Marshal(args)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal request body: %w", err)
				}
				req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
				if err == nil {
					req.Header.Set("Content-Type", "application/json")
				}
			} else {
				// For GET/DELETE, replace placeholders in URL
				if argMap, ok := args.(map[string]interface{}); ok {
					for key, val := range argMap {
						placeholder := "{" + key + "}"
						url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", val))
					}
				}
				req, err = http.NewRequestWithContext(ctx, method, url, nil)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to create http request: %w", err)
			}

			// Add authentication headers if configured
			if currentConfig.Authentication != nil {
				switch currentConfig.Authentication.Type {
				case "bearer":
					req.Header.Set("Authorization", "Bearer "+currentConfig.Authentication.Token)
				case "header":
					req.Header.Set(currentConfig.Authentication.Name, currentConfig.Authentication.Value)
				}
			}

			// Send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("http request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// 4. Parse the JSON response
			var responseData map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
				return nil, fmt.Errorf("failed to decode json response: %w", err)
			}

			// 5. Process the response data using the new recursive mapping logic
			results, err := processMappings(responseData, currentConfig.OutputMapping)
			if err != nil {
				return nil, fmt.Errorf("failed to process output mappings: %w", err)
			}

			return mcp.NewToolResultText(strings.Join(results, "|")), nil
		}

		s.AddTool(tool, handler)
	}

	return nil
}

// processMappings recursively processes data according to the mapping configuration.
func processMappings(contextData interface{}, mappings []hyancieMCP.OutputMap) ([]string, error) {
	var results []string

	// The context can be a map (object) or we might be mapping inside an array.
	contextMap, isMap := contextData.(map[string]interface{})

	for _, mapping := range mappings {
		var value interface{}
		var found bool

		// We can only look up keys if the current data context is a map.
		if isMap {
			// Use getValueFromNestedMap for powerful lookups like "a.b" or "a[0].c"
			value, found = getValueFromNestedMap(contextMap, mapping.JsonKey)
		}

		if !found {
			continue
		}

		switch mapping.Type {
		case "primitive":
			results = append(results, fmt.Sprintf("%s:%v", mapping.Description, value))

		case "array":
			arrayValue, ok := value.([]interface{})
			if !ok {
				continue // Skip if the key does not point to an array
			}

			limit := mapping.Limit
			if limit == 0 || limit > len(arrayValue) {
				limit = len(arrayValue)
			}

			var allItemsFormatted []string
			for i := 0; i < limit; i++ {
				itemContext := arrayValue[i]
				// RECURSIVE CALL: process the sub-mappings on the item's context.
				// The json_key in sub-mappings will be looked up within the itemContext.
				subResults, err := processMappings(itemContext, mapping.Items)
				if err != nil {
					return nil, err
				}
				// Format as: 项1:{标题:..., 链接:...}
				allItemsFormatted = append(allItemsFormatted, fmt.Sprintf("项%d:{%s}", i+1, strings.Join(subResults, ", ")))
			}
			// Format as: 搜索结果:[项1:{...} | 项2:{...}]
			results = append(results, fmt.Sprintf("%s:[%s]", mapping.Description, strings.Join(allItemsFormatted, " | ")))
		}
	}
	return results, nil
}

// getValueFromNestedMap extracts a value from a nested map[string]interface{} using a dot-separated key.
func getValueFromNestedMap(data map[string]interface{}, key string) (interface{}, bool) {
	// This function is now also called from within a recursive context.
	// If the key is simple (no dots or brackets), just do a direct lookup.
	if !strings.ContainsAny(key, ".[") {
		val, exists := data[key]
		return val, exists
	}

	keys := strings.Split(key, ".")
	var current interface{} = data

	for _, k := range keys {
		// Handle array access like "weather[0]"
		if strings.Contains(k, "[") && strings.HasSuffix(k, "]") {
			parts := strings.SplitN(k, "[", 2)
			arrayKey := parts[0]
			indexStr := strings.TrimSuffix(parts[1], "]")
			var index int
			_, err := fmt.Sscanf(indexStr, "%d", &index)
			if err != nil {
				return nil, false // Invalid index format
			}

			var currentMap map[string]interface{}
			var ok bool

			// The 'current' context could be the root map, or a sub-map.
			if current == nil { // Should only happen for the first key part
				currentMap = data
			} else {
				currentMap, ok = current.(map[string]interface{})
				if !ok {
					return nil, false
				}
			}

			var arrayVal interface{}
			var exists bool

			// If arrayKey is empty, it means we are indexing the current context, e.g., "[0]"
			if arrayKey == "" {
				if arraySlice, ok := current.([]interface{}); ok {
					if index < len(arraySlice) {
						current = arraySlice[index]
						continue
					}
				}
				return nil, false
			}

			arrayVal, exists = currentMap[arrayKey]
			if !exists {
				return nil, false
			}

			if arraySlice, ok := arrayVal.([]interface{}); ok {
				if index < len(arraySlice) {
					current = arraySlice[index]
					continue
				}
			}
			return nil, false

		} else {
			// Handle simple key access
			if currentMap, ok := current.(map[string]interface{}); ok {
				val, exists := currentMap[k]
				if !exists {
					return nil, false
				}
				current = val
			} else {
				return nil, false
			}
		}
	}
	return current, true
}