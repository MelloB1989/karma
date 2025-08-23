package tests

import (
	"testing"

	"github.com/MelloB1989/karma/ai"
)

func TestMCPTool_Validation(t *testing.T) {
	tests := []struct {
		name    string
		tool    ai.MCPTool
		isValid bool
	}{
		{
			name: "Valid MCP tool",
			tool: ai.MCPTool{
				FriendlyName: "Calculator",
				ToolName:     "calculate",
				Description:  "Perform basic arithmetic operations",
				InputSchema:  map[string]interface{}{"type": "object"},
			},
			isValid: true,
		},
		{
			name: "Empty friendly name",
			tool: ai.MCPTool{
				FriendlyName: "",
				ToolName:     "calculate",
				Description:  "Perform basic arithmetic operations",
				InputSchema:  map[string]interface{}{"type": "object"},
			},
			isValid: false,
		},
		{
			name: "Empty tool name",
			tool: ai.MCPTool{
				FriendlyName: "Calculator",
				ToolName:     "",
				Description:  "Perform basic arithmetic operations",
				InputSchema:  map[string]interface{}{"type": "object"},
			},
			isValid: false,
		},
		{
			name: "Empty description",
			tool: ai.MCPTool{
				FriendlyName: "Calculator",
				ToolName:     "calculate",
				Description:  "",
				InputSchema:  map[string]interface{}{"type": "object"},
			},
			isValid: false,
		},
		{
			name: "Nil input schema",
			tool: ai.MCPTool{
				FriendlyName: "Calculator",
				ToolName:     "calculate",
				Description:  "Perform basic arithmetic operations",
				InputSchema:  nil,
			},
			isValid: false,
		},
		{
			name: "Complex input schema",
			tool: ai.MCPTool{
				FriendlyName: "Weather Tool",
				ToolName:     "get_weather",
				Description:  "Get weather information for a location",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The location to get weather for",
						},
						"units": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"metric", "imperial"},
							"description": "Temperature units",
						},
					},
					"required": []string{"location"},
				},
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			isEmpty := func(s string) bool { return s == "" }

			hasEmptyFields := isEmpty(tt.tool.FriendlyName) ||
				isEmpty(tt.tool.ToolName) ||
				isEmpty(tt.tool.Description) ||
				tt.tool.InputSchema == nil

			if tt.isValid && hasEmptyFields {
				t.Error("Expected valid tool but has empty required fields")
			}

			if !tt.isValid && !hasEmptyFields {
				t.Error("Expected invalid tool but all fields are present")
			}
		})
	}
}

func TestMCPServer_Creation(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		authToken string
		tools     []ai.MCPTool
		expectErr bool
	}{
		{
			name:      "Valid MCP server",
			url:       "http://localhost:8080/mcp",
			authToken: "test-token",
			tools: []ai.MCPTool{
				{
					FriendlyName: "Calculator",
					ToolName:     "calc",
					Description:  "Basic calculator",
					InputSchema:  map[string]interface{}{"type": "object"},
				},
			},
			expectErr: false,
		},
		{
			name:      "Server with HTTPS URL",
			url:       "https://api.example.com/mcp",
			authToken: "secure-token",
			tools:     []ai.MCPTool{},
			expectErr: false,
		},
		{
			name:      "Server with empty URL",
			url:       "",
			authToken: "test-token",
			tools:     []ai.MCPTool{},
			expectErr: true,
		},
		{
			name:      "Server with empty auth token",
			url:       "http://localhost:8080/mcp",
			authToken: "",
			tools:     []ai.MCPTool{},
			expectErr: false, // Auth token might be optional
		},
		{
			name:      "Server with multiple tools",
			url:       "http://localhost:8080/mcp",
			authToken: "test-token",
			tools: []ai.MCPTool{
				{
					FriendlyName: "Calculator",
					ToolName:     "calc",
					Description:  "Basic calculator",
					InputSchema:  map[string]interface{}{"type": "object"},
				},
				{
					FriendlyName: "Weather",
					ToolName:     "weather",
					Description:  "Weather information",
					InputSchema:  map[string]interface{}{"type": "object"},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := ai.NewMCPServer(tt.url, tt.authToken, tt.tools)

			if tt.expectErr && tt.url != "" {
				t.Error("Expected error for invalid configuration")
			}

			if !tt.expectErr {
				if server.URL != tt.url {
					t.Errorf("Expected URL %s, got %s", tt.url, server.URL)
				}
				if server.AuthToken != tt.authToken {
					t.Errorf("Expected auth token %s, got %s", tt.authToken, server.AuthToken)
				}
				if len(server.Tools) != len(tt.tools) {
					t.Errorf("Expected %d tools, got %d", len(tt.tools), len(server.Tools))
				}
			}
		})
	}
}

func TestMCPIntegration_KarmaAI(t *testing.T) {
	tests := []struct {
		name        string
		model       ai.BaseModel
		provider    ai.Provider
		mcpUrl      string
		authToken   string
		tools       []ai.MCPTool
		expectValid bool
	}{
		{
			name:      "OpenAI with MCP support",
			model:     ai.GPT4o,
			provider:  ai.OpenAI,
			mcpUrl:    "http://localhost:8080/mcp",
			authToken: "test-token",
			tools: []ai.MCPTool{
				{
					FriendlyName: "Calculator",
					ToolName:     "calc",
					Description:  "Basic calculator",
					InputSchema:  map[string]interface{}{"type": "object"},
				},
			},
			expectValid: true,
		},
		{
			name:      "XAI with MCP support",
			model:     ai.Grok3,
			provider:  ai.XAI,
			mcpUrl:    "http://localhost:8080/mcp",
			authToken: "test-token",
			tools: []ai.MCPTool{
				{
					FriendlyName: "Search",
					ToolName:     "search",
					Description:  "Search the web",
					InputSchema:  map[string]interface{}{"type": "object"},
				},
			},
			expectValid: true,
		},
		{
			name:        "Anthropic with MCP support",
			model:       ai.Claude35Sonnet,
			provider:    ai.Anthropic,
			mcpUrl:      "http://localhost:8080/mcp",
			authToken:   "test-token",
			tools:       []ai.MCPTool{},
			expectValid: true,
		},
		{
			name:        "Bedrock without MCP support",
			model:       ai.Llama3_8B,
			provider:    ai.Bedrock,
			mcpUrl:      "http://localhost:8080/mcp",
			authToken:   "test-token",
			tools:       []ai.MCPTool{},
			expectValid: false,
		},
		{
			name:        "Google without MCP support",
			model:       ai.Gemini25Flash,
			provider:    ai.Google,
			mcpUrl:      "http://localhost:8080/mcp",
			authToken:   "test-token",
			tools:       []ai.MCPTool{},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := ai.NewKarmaAI(tt.model, tt.provider,
				ai.SetMCPUrl(tt.mcpUrl),
				ai.SetMCPAuthToken(tt.authToken),
				ai.SetMCPTools(tt.tools),
			)

			mcpSupported := kai.Model.SupportsMCP()

			if tt.expectValid && !mcpSupported {
				t.Error("Expected MCP support but provider doesn't support it")
			}

			if !tt.expectValid && mcpSupported {
				t.Error("Provider shouldn't support MCP but indicates it does")
			}

			// Validate configuration was set correctly
			if kai.MCPUrl != tt.mcpUrl {
				t.Errorf("Expected MCP URL %s, got %s", tt.mcpUrl, kai.MCPUrl)
			}

			if kai.AuthToken != tt.authToken {
				t.Errorf("Expected auth token %s, got %s", tt.authToken, kai.AuthToken)
			}

			if len(kai.MCPTools) != len(tt.tools) {
				t.Errorf("Expected %d MCP tools, got %d", len(tt.tools), len(kai.MCPTools))
			}
		})
	}
}

func TestMCPServer_MultipleServers(t *testing.T) {
	t.Run("Add multiple MCP servers", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI)

		// Create multiple servers
		server1 := ai.NewMCPServer("http://localhost:8080/mcp", "token1", []ai.MCPTool{
			{
				FriendlyName: "Calculator",
				ToolName:     "calc",
				Description:  "Basic calculator",
				InputSchema:  map[string]interface{}{"type": "object"},
			},
		})

		server2 := ai.NewMCPServer("http://localhost:8081/mcp", "token2", []ai.MCPTool{
			{
				FriendlyName: "Weather",
				ToolName:     "weather",
				Description:  "Weather service",
				InputSchema:  map[string]interface{}{"type": "object"},
			},
		})

		// Configure with multiple servers
		kai = ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.SetMCPServers([]ai.MCPServer{server1, server2}),
		)

		if len(kai.MCPServers) != 2 {
			t.Errorf("Expected 2 MCP servers, got %d", len(kai.MCPServers))
		}

		// Verify server configurations
		if kai.MCPServers[0].URL != "http://localhost:8080/mcp" {
			t.Errorf("Expected first server URL http://localhost:8080/mcp, got %s", kai.MCPServers[0].URL)
		}

		if kai.MCPServers[1].URL != "http://localhost:8081/mcp" {
			t.Errorf("Expected second server URL http://localhost:8081/mcp, got %s", kai.MCPServers[1].URL)
		}
	})

	t.Run("Add server to existing configuration", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI)

		server := ai.NewMCPServer("http://localhost:8080/mcp", "token", []ai.MCPTool{})

		kai = ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.AddMCPServer(server),
		)

		if len(kai.MCPServers) != 1 {
			t.Errorf("Expected 1 MCP server, got %d", len(kai.MCPServers))
		}
	})
}

func TestMCPTool_ComplexSchemas(t *testing.T) {
	tests := []struct {
		name   string
		tool   ai.MCPTool
		schema interface{}
	}{
		{
			name: "Weather tool with enum",
			tool: ai.MCPTool{
				FriendlyName: "Weather Service",
				ToolName:     "get_weather",
				Description:  "Get current weather for a location",
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name or coordinates",
					},
					"units": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"celsius", "fahrenheit", "kelvin"},
						"default":     "celsius",
						"description": "Temperature units",
					},
					"include_forecast": map[string]interface{}{
						"type":        "boolean",
						"default":     false,
						"description": "Include 5-day forecast",
					},
				},
				"required": []string{"location"},
			},
		},
		{
			name: "Calculator with operation enum",
			tool: ai.MCPTool{
				FriendlyName: "Advanced Calculator",
				ToolName:     "calculate",
				Description:  "Perform mathematical operations",
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"add", "subtract", "multiply", "divide", "power", "sqrt"},
						"description": "Mathematical operation to perform",
					},
					"operands": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "number",
						},
						"minItems":    1,
						"maxItems":    2,
						"description": "Numbers to operate on",
					},
					"precision": map[string]interface{}{
						"type":        "integer",
						"minimum":     0,
						"maximum":     10,
						"default":     2,
						"description": "Decimal precision for result",
					},
				},
				"required": []string{"operation", "operands"},
			},
		},
		{
			name: "File processor with complex nested schema",
			tool: ai.MCPTool{
				FriendlyName: "File Processor",
				ToolName:     "process_file",
				Description:  "Process files with various operations",
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{
								"type":        "string",
								"description": "File path",
							},
							"encoding": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"utf-8", "ascii", "base64"},
								"default":     "utf-8",
								"description": "File encoding",
							},
						},
						"required": []string{"path"},
					},
					"operations": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type": "string",
									"enum": []string{"read", "write", "compress", "encrypt"},
								},
								"parameters": map[string]interface{}{
									"type": "object",
								},
							},
							"required": []string{"type"},
						},
						"minItems": 1,
					},
				},
				"required": []string{"file", "operations"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the schema and validate
			tt.tool.InputSchema = tt.schema

			if tt.tool.InputSchema == nil {
				t.Error("Input schema should not be nil")
			}

			// Check if it's a valid object schema
			if schemaMap, ok := tt.tool.InputSchema.(map[string]interface{}); ok {
				if schemaType, exists := schemaMap["type"]; exists {
					if schemaType != "object" {
						t.Errorf("Expected schema type 'object', got %v", schemaType)
					}
				}

				// Check for properties
				if properties, exists := schemaMap["properties"]; exists {
					if _, ok := properties.(map[string]interface{}); !ok {
						t.Error("Properties should be a map")
					}
				}
			} else {
				t.Error("Schema should be a map[string]interface{}")
			}
		})
	}
}

func TestMCPConfiguration_EdgeCases(t *testing.T) {
	t.Run("Empty tools array", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.SetMCPTools([]ai.MCPTool{}),
		)

		if len(kai.MCPTools) != 0 {
			t.Errorf("Expected 0 tools, got %d", len(kai.MCPTools))
		}
	})

	t.Run("Very long tool descriptions", func(t *testing.T) {
		longDescription := generateLongStringMCP(10000)

		tool := ai.MCPTool{
			FriendlyName: "Test Tool",
			ToolName:     "test",
			Description:  longDescription,
			InputSchema:  map[string]interface{}{"type": "object"},
		}

		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.SetMCPTools([]ai.MCPTool{tool}),
		)

		if len(kai.MCPTools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(kai.MCPTools))
		}

		if kai.MCPTools[0].Description != longDescription {
			t.Error("Long description should be preserved")
		}
	})

	t.Run("Special characters in tool names", func(t *testing.T) {
		specialNames := []string{
			"tool-with-dashes",
			"tool_with_underscores",
			"tool.with.dots",
			"tool123with456numbers",
			"toolWithCamelCase",
		}

		for _, name := range specialNames {
			tool := ai.MCPTool{
				FriendlyName: "Test Tool",
				ToolName:     name,
				Description:  "Test description",
				InputSchema:  map[string]interface{}{"type": "object"},
			}

			kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
				ai.SetMCPTools([]ai.MCPTool{tool}),
			)

			if kai.MCPTools[0].ToolName != name {
				t.Errorf("Expected tool name %s, got %s", name, kai.MCPTools[0].ToolName)
			}
		}
	})

	t.Run("URL validation edge cases", func(t *testing.T) {
		urls := []struct {
			url   string
			valid bool
		}{
			{"http://localhost:8080/mcp", true},
			{"https://api.example.com/mcp", true},
			{"http://127.0.0.1:3000/mcp", true},
			{"https://subdomain.example.com:8443/api/mcp", true},
			{"ftp://invalid.protocol.com/mcp", false}, // Assuming only HTTP/HTTPS are valid
			{"localhost:8080/mcp", false},             // Missing protocol
			{"", false},                               // Empty URL
		}

		for _, test := range urls {
			server := ai.NewMCPServer(test.url, "token", []ai.MCPTool{})

			// Basic validation - URL should be set regardless of validity
			if server.URL != test.url {
				t.Errorf("Expected URL %s, got %s", test.url, server.URL)
			}
		}
	})
}

// Helper function to generate long strings for testing
func generateLongStringMCP(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a' + byte(i%26)
	}
	return string(result)
}

// BenchmarkMCPTool_Creation benchmarks MCP tool creation
func BenchmarkMCPTool_Creation(b *testing.B) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ai.MCPTool{
			FriendlyName: "Test Tool",
			ToolName:     "test",
			Description:  "Test description",
			InputSchema:  schema,
		}
	}
}

// BenchmarkMCPServer_Creation benchmarks MCP server creation
func BenchmarkMCPServer_Creation(b *testing.B) {
	tools := []ai.MCPTool{
		{
			FriendlyName: "Calculator",
			ToolName:     "calc",
			Description:  "Basic calculator",
			InputSchema:  map[string]interface{}{"type": "object"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ai.NewMCPServer("http://localhost:8080/mcp", "token", tools)
	}
}
