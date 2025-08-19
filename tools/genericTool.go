package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	hyancie "github.com/liu599/hyancie"
	"github.com/liu599/hyancie/logging"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yosida95/uritemplate/v3"
)

// AddGenericTools registers all tools defined in the global config with the MCP server.
func AddGenericTools(s *server.MCPServer) error {
	configs := hyancie.Config.McpTools

	for _, config := range configs {
		currentConfig := config

		tool := mcp.Tool{
			Name:        currentConfig.ToolName,
			Description: currentConfig.Description,
			InputSchema: currentConfig.InputSchema,
		}

		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			// Apply default values
			if currentConfig.InputSchema.Properties != nil {
				for propName, propDetailsInterface := range currentConfig.InputSchema.Properties {
					if _, argProvided := args[propName]; !argProvided {
						if propDetails, ok := propDetailsInterface.(map[string]interface{}); ok {
							if defaultValue, defaultExists := propDetails["default"]; defaultExists {
								args[propName] = defaultValue
							}
						}
					}
				}
			}

			// Log incoming request
			logging.Logger.Info("Tool called", "tool_name", currentConfig.ToolName, "arguments", args)

			// Expand URL template
			logging.Logger.Info("Expanding URL template", "url", currentConfig.Request.URL)
			fmt.Println("Attempting to expand URL template:", currentConfig.Request.URL)
			template, err := uritemplate.New(currentConfig.Request.URL)
			if err != nil {
				return nil, fmt.Errorf("invalid url template: %w", err)
			}

			// Convert args to uritemplate.Values
			values := uritemplate.Values{}
			for k, v := range args {
				strValue := fmt.Sprintf("%v", v)
				// Check if the method is GET, and if so, escape the string.
				if strings.ToUpper(currentConfig.Request.Method) == "GET" {
					strValue = url.QueryEscape(strValue)
				}
				values.Set(k, uritemplate.String(strValue))
			}

			expandedURL, err := template.Expand(values)
			if err != nil {
				return nil, fmt.Errorf("failed to expand url template: %w", err)
			}

			var req *http.Request
			method := strings.ToUpper(currentConfig.Request.Method)

			if method == "POST" || method == "PUT" {
				jsonBody, err := json.Marshal(args)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal request body: %w", err)
				}
				req, err = http.NewRequestWithContext(ctx, method, expandedURL, bytes.NewBuffer(jsonBody))
				if err == nil {
					req.Header.Set("Content-Type", "application/json")
				}
			} else {
				req, err = http.NewRequestWithContext(ctx, method, expandedURL, nil)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to create http request: %w", err)
			}

			for _, header := range currentConfig.Headers {
				req.Header.Set(header.Name, header.Value)
			}

			client := &http.Client{}
			// Log the request details just before sending
			if req.Body != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(req.Body)
				bodyStr := buf.String()
				// And now set a new body, since you can't read it twice.
				req.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
				logging.Logger.Info("Sending HTTP request", "method", req.Method, "url", req.URL.String(), "body", bodyStr)
			} else {
				logging.Logger.Info("Sending HTTP request", "method", req.Method, "url", req.URL.String())
			}
			resp, err := client.Do(req)
			if err != nil {
				logging.Logger.Error("HTTP request failed", "error", err)
				return nil, fmt.Errorf("http request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				logging.Logger.Error("Failed to read response body", "error", err)
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			// Log the response
			logging.Logger.Info("Received HTTP response", "status_code", resp.StatusCode, "body", string(bodyBytes))

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var responseData map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
				return nil, fmt.Errorf("failed to decode json response: %w", err)
			}

			results, err := processMappings(responseData, currentConfig.OutputMapping)
			if err != nil {
				return nil, fmt.Errorf("failed to process output mappings: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: strings.Join(results, "|"),
					},
				},
			}
			result.Meta = map[string]interface{}{
				"expandedURL": expandedURL,
			}
			return result, nil
		}

		s.AddTool(tool, handler)
	}

	return nil
}

// processMappings recursively processes data according to the mapping configuration.
func processMappings(contextData interface{}, mappings []hyancie.OutputMap) ([]string, error) {
	var results []string
	contextMap, isMap := contextData.(map[string]interface{})

	for _, mapping := range mappings {
		var value interface{}
		var found bool

		if isMap {
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
				continue
			}

			limit := mapping.Limit
			if limit == 0 || limit > len(arrayValue) {
				limit = len(arrayValue)
			}

			var allItemsFormatted []string
			for i := 0; i < limit; i++ {
				itemContext := arrayValue[i]
				subResults, err := processMappings(itemContext, mapping.Items)
				if err != nil {
					return nil, err
				}
				allItemsFormatted = append(allItemsFormatted, fmt.Sprintf("项%d:{%s}", i+1, strings.Join(subResults, ", ")))
			}
			results = append(results, fmt.Sprintf("%s:[%s]", mapping.Description, strings.Join(allItemsFormatted, " | ")))
		}
	}
	return results, nil
}

// getValueFromNestedMap extracts a value from a nested map[string]interface{} using a dot-separated key.
func getValueFromNestedMap(data map[string]interface{}, key string) (interface{}, bool) {
	if !strings.ContainsAny(key, ".[") {
		val, exists := data[key]
		return val, exists
	}

	keys := strings.Split(key, ".")
	var current interface{} = data

	for _, k := range keys {
		if strings.Contains(k, "[") && strings.HasSuffix(k, "]") {
			parts := strings.SplitN(k, "[", 2)
			arrayKey := parts[0]
			indexStr := strings.TrimSuffix(parts[1], "]")
			var index int
			_, err := fmt.Sscanf(indexStr, "%d", &index)
			if err != nil {
				return nil, false
			}

			var currentMap map[string]interface{}
			var ok bool

			if current == nil {
				currentMap = data
			} else {
				currentMap, ok = current.(map[string]interface{})
				if !ok {
					return nil, false
				}
			}

			var arrayVal interface{}
			var exists bool

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
