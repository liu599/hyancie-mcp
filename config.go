package hyancie

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// --- Structs for parsing config.json ---

// GenericToolConfig defines the structure for a single tool configuration.
type GenericToolConfig struct {
	ToolName      string              `json:"tool_name"`
	Description   string              `json:"description"`
	Request       RequestConfig       `json:"request"`
	Headers       []Header            `json:"headers,omitempty"`
	InputSchema   mcp.ToolInputSchema `json:"input_schema"`
	OutputMapping []OutputMap         `json:"output_mapping"`
}

// RequestConfig defines the HTTP request details.
type RequestConfig struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

// Header represents a single HTTP header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// OutputMap defines how to map a key from the JSON response to a human-readable description.
type OutputMap struct {
	JsonKey     string      `json:"json_key"`
	Description string      `json:"description"`
	Type        string      `json:"type"` // "primitive", "array"
	Limit       int         `json:"limit,omitempty"`
	Items       []OutputMap `json:"items,omitempty"` // For type "array"
}

// LoggingConfig defines the structure for logging settings.
type LoggingConfig struct {
	FilePath string `json:"file_path"`
}

// ConfigType is the top-level structure for the entire config.json file.
type ConfigType struct {
	ServerName    string              `json:"server_name"`
	ServerVersion string              `json:"server_version"`
	SseAddress    string              `json:"sse_address"`
	Logging       LoggingConfig       `json:"logging"`
	McpTools      []GenericToolConfig `json:"mcp_tools"`

	// Deprecated fields, kept for compatibility with old static tools if needed.
	WebSearchURL string `yaml:"web_search_url"`
	APIKey       string `yaml:"X-API-Key"`
}

// Config holds the single, global instance of the application's configuration.
var Config = &ConfigType{}

// LoadConfig loads all configuration from the config.json file
// located in the same directory as the executable.
func LoadConfig() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.json")

	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config.json at %s: %w", configPath, err)
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(Config)
	if err != nil {
		return fmt.Errorf("failed to decode config.json: %w", err)
	}

	return nil
}
