package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"google.golang.org/genai"
)

const defaultMaxToolPasses = 5

// FuncParams represents function parameters for tool calls
type FuncParams map[string]any

// GoFunctionTool represents a Go function that can be called as a tool
type GoFunctionTool struct {
	Name        string
	Description string
	Parameters  *genai.Schema
	Handler     func(context.Context, FuncParams) (string, error)
}

// Gemini represents a Gemini client configuration
type Gemini struct {
	Client          *genai.Client
	Model           string
	Temperature     float32
	TopP            float32
	TopK            float32
	MaxTokens       int32
	SystemMessage   string
	ResponseType    string
	MCPManager      *mcp.Manager
	MultiMCPManager *mcp.MultiManager
	FunctionTools   map[string]GoFunctionTool
	maxToolPasses   int
}

// NewGemini creates a new Gemini client using environment variables for Vertex AI config
func NewGemini(model, systemMessage string, temperature, topP, topK float64, maxTokens int64) (*Gemini, error) {
	projectID := config.GetEnvRaw("GOOGLE_PROJECT_ID")
	location := config.GetEnvRaw("GOOGLE_LOCATION")
	return NewGeminiWithVertexAI(model, systemMessage, temperature, topP, topK, maxTokens, projectID, location)
}

// NewGeminiWithVertexAI creates a new Gemini client using Vertex AI backend with explicit project and location
func NewGeminiWithVertexAI(model, systemMessage string, temperature, topP, topK float64, maxTokens int64, projectID, location string) (*Gemini, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  projectID,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Gemini{
		Client:        client,
		Model:         model,
		Temperature:   float32(temperature),
		TopP:          float32(topP),
		TopK:          float32(topK),
		MaxTokens:     int32(maxTokens),
		SystemMessage: systemMessage,
		FunctionTools: make(map[string]GoFunctionTool),
		maxToolPasses: defaultMaxToolPasses,
	}, nil
}

// NewGeminiWithAPIKey creates a new Gemini client using API key authentication
func NewGeminiWithAPIKey(model, systemMessage string, temperature, topP, topK float64, maxTokens int64, apiKey string) (*Gemini, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Gemini{
		Client:        client,
		Model:         model,
		Temperature:   float32(temperature),
		TopP:          float32(topP),
		TopK:          float32(topK),
		MaxTokens:     int32(maxTokens),
		SystemMessage: systemMessage,
		FunctionTools: make(map[string]GoFunctionTool),
		maxToolPasses: defaultMaxToolPasses,
	}, nil
}

// SetMCPServer configures the MCP server for tool calling
func (g *Gemini) SetMCPServer(serverURL string, authToken string) {
	mcpClient := mcp.NewClient(serverURL, authToken)
	g.MCPManager = mcp.NewManager(mcpClient)
}

// SetMultiMCPManager sets a multi-MCP manager for multiple MCP servers
func (g *Gemini) SetMultiMCPManager(multiManager *mcp.MultiManager) {
	g.MultiMCPManager = multiManager
}

// SetMaxToolPasses sets the maximum number of tool execution passes
func (g *Gemini) SetMaxToolPasses(max int) {
	g.maxToolPasses = max
}

// AddGoFunctionTool adds a Go function tool
func (g *Gemini) AddGoFunctionTool(tool GoFunctionTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler required")
	}
	if tool.Parameters == nil {
		tool.Parameters = &genai.Schema{
			Type:       genai.TypeObject,
			Properties: map[string]*genai.Schema{},
		}
	}
	g.FunctionTools[tool.Name] = tool
	return nil
}

// ClearGoFunctionTools removes all Go function tools
func (g *Gemini) ClearGoFunctionTools() {
	g.FunctionTools = make(map[string]GoFunctionTool)
}

// GetMCPManager returns the MCP manager
func (g *Gemini) GetMCPManager() *mcp.Manager {
	return g.MCPManager
}

// SetResponseType sets the response MIME type
func (g *Gemini) SetResponseType(responseType string) {
	g.ResponseType = responseType
}

// CreateChat creates a chat completion with tool calling support
func (g *Gemini) CreateChat(messages *models.AIChatHistory, enableTools bool, useMCPExecution bool) (*genai.GenerateContentResponse, error) {
	ctx := context.Background()
	contents := g.formatMessages(*messages)
	config := g.buildConfig(enableTools)

	for range g.toolPassLimit() {
		response, err := g.Client.Models.GenerateContent(ctx, g.Model, contents, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate content: %w", err)
		}

		if !g.shouldExecuteTools(response, enableTools, useMCPExecution) {
			// Add assistant message to history
			if len(response.Candidates) > 0 && response.Candidates[0].Content != nil {
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:      models.Assistant,
					Message:   response.Text(),
					Timestamp: time.Now(),
					UniqueId:  utils.GenerateID(16),
				})
			}
			return response, nil
		}

		// Get function calls from response
		functionCalls := response.FunctionCalls()
		if len(functionCalls) == 0 {
			// Add assistant message to history even when no function calls
			if len(response.Candidates) > 0 && response.Candidates[0].Content != nil {
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:      models.Assistant,
					Message:   response.Text(),
					Timestamp: time.Now(),
					UniqueId:  utils.GenerateID(16),
				})
			}
			return response, nil
		}

		// Add assistant message with tool calls to history
		assistantMsg := models.AIMessage{
			Role:      models.Assistant,
			Message:   response.Text(),
			Timestamp: time.Now(),
			UniqueId:  utils.GenerateID(16),
		}
		if len(functionCalls) > 0 {
			assistantMsg.ToolCalls = make([]models.OpenAIToolCall, len(functionCalls))
			for i, fc := range functionCalls {
				argsJSON, _ := json.Marshal(fc.Args)
				assistantMsg.ToolCalls[i] = models.OpenAIToolCall{
					ID:   generateShortToolCallID(fc.Name + "_" + utils.GenerateID(8)),
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      fc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}
		messages.Messages = append(messages.Messages, assistantMsg)

		// Add model response with function calls to contents
		// The model's response should be added as-is (it already has RoleModel)
		if len(response.Candidates) > 0 && response.Candidates[0].Content != nil {
			contents = append(contents, response.Candidates[0].Content)
		}

		// Execute tools and collect ALL function responses in a single Content
		// Gemini API requires: number of function response parts == number of function call parts
		functionResponseParts := make([]*genai.Part, 0, len(functionCalls))
		for i, fc := range functionCalls {
			result, err := g.callAnyTool(ctx, fc.Name, fc.Args)
			var responseMap map[string]any
			if err != nil {
				responseMap = map[string]any{"error": err.Error()}
			} else {
				// Try to parse result as JSON, otherwise wrap in output key
				if jsonErr := json.Unmarshal([]byte(result), &responseMap); jsonErr != nil {
					responseMap = map[string]any{"output": result}
				}
			}

			// Collect function response part
			functionResponseParts = append(functionResponseParts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					ID:       fc.ID,
					Name:     fc.Name,
					Response: responseMap,
				},
			})

			// Add tool response to history
			toolCallID := ""
			if i < len(assistantMsg.ToolCalls) {
				toolCallID = assistantMsg.ToolCalls[i].ID
			}
			messages.Messages = append(messages.Messages, models.AIMessage{
				Role:       models.Tool,
				Message:    result,
				ToolCallId: toolCallID,
				Timestamp:  time.Now(),
				UniqueId:   utils.GenerateID(16),
			})
		}

		// Add ALL function responses as a single Content with multiple Parts
		// This ensures the number of response parts matches the number of function calls
		contents = append(contents, &genai.Content{
			Parts: functionResponseParts,
		})
	}

	return nil, fmt.Errorf("exceeded tool execution passes")
}

// CreateChatStream creates a streaming chat completion with tool calling support
func (g *Gemini) CreateChatStream(messages *models.AIChatHistory, chunkHandler func(*genai.GenerateContentResponse), enableTools bool, useMCPExecution bool) (*genai.GenerateContentResponse, error) {
	ctx := context.Background()
	contents := g.formatMessages(*messages)
	config := g.buildConfig(enableTools)

	for range g.toolPassLimit() {
		// Accumulate the streamed response
		acc, err := g.streamAndAccumulate(ctx, contents, config, chunkHandler)
		if err != nil {
			return nil, fmt.Errorf("failed to stream content: %w", err)
		}

		if !g.shouldExecuteTools(acc, enableTools, useMCPExecution) {
			// Add assistant message to history
			if len(acc.Candidates) > 0 && acc.Candidates[0].Content != nil {
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:      models.Assistant,
					Message:   acc.Text(),
					Timestamp: time.Now(),
					UniqueId:  utils.GenerateID(16),
				})
			}
			return acc, nil
		}

		// Get function calls from response
		functionCalls := acc.FunctionCalls()
		if len(functionCalls) == 0 {
			// Add assistant message to history even when no function calls
			if len(acc.Candidates) > 0 && acc.Candidates[0].Content != nil {
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:      models.Assistant,
					Message:   acc.Text(),
					Timestamp: time.Now(),
					UniqueId:  utils.GenerateID(16),
				})
			}
			return acc, nil
		}

		// Add assistant message with tool calls to history
		assistantMsg := models.AIMessage{
			Role:      models.Assistant,
			Message:   acc.Text(),
			Timestamp: time.Now(),
			UniqueId:  utils.GenerateID(16),
		}
		if len(functionCalls) > 0 {
			assistantMsg.ToolCalls = make([]models.OpenAIToolCall, len(functionCalls))
			for i, fc := range functionCalls {
				argsJSON, _ := json.Marshal(fc.Args)
				assistantMsg.ToolCalls[i] = models.OpenAIToolCall{
					ID:   generateShortToolCallID(fc.Name + "_" + utils.GenerateID(8)),
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      fc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}
		messages.Messages = append(messages.Messages, assistantMsg)

		// Add model response with function calls to contents
		// The model's response should be added as-is (it already has RoleModel)
		if len(acc.Candidates) > 0 && acc.Candidates[0].Content != nil {
			contents = append(contents, acc.Candidates[0].Content)
		}

		// Execute tools and collect ALL function responses in a single Content
		// Gemini API requires: number of function response parts == number of function call parts
		functionResponseParts := make([]*genai.Part, 0, len(functionCalls))
		for i, fc := range functionCalls {
			result, err := g.callAnyTool(ctx, fc.Name, fc.Args)
			var responseMap map[string]any
			if err != nil {
				responseMap = map[string]any{"error": err.Error()}
			} else {
				// Try to parse result as JSON, otherwise wrap in output key
				if jsonErr := json.Unmarshal([]byte(result), &responseMap); jsonErr != nil {
					responseMap = map[string]any{"output": result}
				}
			}

			// Collect function response part
			functionResponseParts = append(functionResponseParts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					ID:       fc.ID,
					Name:     fc.Name,
					Response: responseMap,
				},
			})

			// Add tool response to history
			toolCallID := ""
			if i < len(assistantMsg.ToolCalls) {
				toolCallID = assistantMsg.ToolCalls[i].ID
			}
			messages.Messages = append(messages.Messages, models.AIMessage{
				Role:       models.Tool,
				Message:    result,
				ToolCallId: toolCallID,
				Timestamp:  time.Now(),
				UniqueId:   utils.GenerateID(16),
			})
		}

		// Add ALL function responses as a single Content with multiple Parts
		// This ensures the number of response parts matches the number of function calls
		contents = append(contents, &genai.Content{
			Parts: functionResponseParts,
		})
	}

	return nil, fmt.Errorf("exceeded tool execution passes")
}

// formatMessages converts AIChatHistory to Gemini content format
func (g *Gemini) formatMessages(messages models.AIChatHistory) []*genai.Content {
	contents := make([]*genai.Content, 0, len(messages.Messages))

	for _, msg := range messages.Messages {
		switch msg.Role {
		case models.User:
			parts := []*genai.Part{{Text: msg.Message}}

			// Add images if present
			for _, image := range msg.Images {
				imagePart := parseImageToPart(image)
				if imagePart != nil {
					parts = append(parts, imagePart)
				}
			}

			contents = append(contents, &genai.Content{
				Parts: parts,
				Role:  genai.RoleUser,
			})

		case models.Assistant:
			parts := []*genai.Part{}
			if msg.Message != "" {
				parts = append(parts, &genai.Part{Text: msg.Message})
			}

			// Add function calls if present
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}

			if len(parts) > 0 {
				contents = append(contents, &genai.Content{
					Parts: parts,
					Role:  genai.RoleModel,
				})
			}

		case models.Tool:
			// Tool responses are added as function responses
			var responseMap map[string]any
			if err := json.Unmarshal([]byte(msg.Message), &responseMap); err != nil {
				responseMap = map[string]any{"output": msg.Message}
			}

			// Find the corresponding function name and ID from the tool call ID
			funcName := "function_response"
			funcID := ""
			for i := len(messages.Messages) - 1; i >= 0; i-- {
				prevMsg := messages.Messages[i]
				if prevMsg.Role == models.Assistant {
					for _, tc := range prevMsg.ToolCalls {
						if tc.ID == msg.ToolCallId {
							funcName = tc.Function.Name
							funcID = tc.ID
							break
						}
					}
					break
				}
			}

			contents = append(contents, &genai.Content{
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							ID:       funcID,
							Name:     funcName,
							Response: responseMap,
						},
					},
				},
			})

		case models.System:
			// System messages are handled via SystemInstruction in config
			// Skip here as they're added to the config
		}
	}

	return contents
}

// buildConfig creates the GenerateContentConfig with tools if enabled
func (g *Gemini) buildConfig(enableTools bool) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		Temperature:     &g.Temperature,
		TopP:            &g.TopP,
		TopK:            &g.TopK,
		MaxOutputTokens: g.MaxTokens,
	}

	if g.SystemMessage != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: g.SystemMessage}},
			Role:  genai.RoleUser,
		}
	}

	if g.ResponseType != "" {
		config.ResponseMIMEType = g.ResponseType
	}

	if enableTools {
		tools := g.buildTools()
		if len(tools) > 0 {
			config.Tools = tools
		}
	}

	return config
}

// buildTools creates the tools configuration for Gemini
func (g *Gemini) buildTools() []*genai.Tool {
	var functionDeclarations []*genai.FunctionDeclaration

	// Add MCP tools
	if g.hasMCPTools() {
		mcpFunctions := g.convertMCPToolsToGemini()
		functionDeclarations = append(functionDeclarations, mcpFunctions...)
	}

	// Add Go function tools
	if g.hasGoFunctionTools() {
		goFunctions := g.convertGoFunctionToolsToGemini()
		functionDeclarations = append(functionDeclarations, goFunctions...)
	}

	if len(functionDeclarations) == 0 {
		return nil
	}

	return []*genai.Tool{
		{FunctionDeclarations: functionDeclarations},
	}
}

// hasMCPTools checks if MCP tools are configured
func (g *Gemini) hasMCPTools() bool {
	if g.MultiMCPManager != nil {
		return g.MultiMCPManager.Count() > 0
	}
	return g.MCPManager != nil && g.MCPManager.Count() > 0
}

// hasGoFunctionTools checks if Go function tools are configured
func (g *Gemini) hasGoFunctionTools() bool {
	return len(g.FunctionTools) > 0
}

// convertMCPToolsToGemini converts MCP tools to Gemini function declarations
func (g *Gemini) convertMCPToolsToGemini() []*genai.FunctionDeclaration {
	if !g.hasMCPTools() {
		return nil
	}

	var mcpTools []*mcp.Tool
	if g.MultiMCPManager != nil {
		mcpTools = g.MultiMCPManager.GetAllTools()
	} else {
		mcpTools = g.MCPManager.GetAllTools()
	}

	declarations := make([]*genai.FunctionDeclaration, len(mcpTools))
	for i, mcpTool := range mcpTools {
		declarations[i] = &genai.FunctionDeclaration{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Parameters:  convertMCPSchemaToGemini(mcpTool.InputSchema),
		}
	}

	return declarations
}

// convertGoFunctionToolsToGemini converts Go function tools to Gemini function declarations
func (g *Gemini) convertGoFunctionToolsToGemini() []*genai.FunctionDeclaration {
	declarations := make([]*genai.FunctionDeclaration, 0, len(g.FunctionTools))
	for _, tool := range g.FunctionTools {
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	return declarations
}

// convertMCPSchemaToGemini converts MCP input schema to Gemini Schema
func convertMCPSchemaToGemini(inputSchema map[string]any) *genai.Schema {
	schema := &genai.Schema{
		Type:       "object",
		Properties: make(map[string]*genai.Schema),
	}

	if properties, ok := inputSchema["properties"].(map[string]any); ok {
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]any); ok {
				schema.Properties[name] = convertPropertyToSchema(propMap)
			}
		}
	}

	if required, ok := inputSchema["required"].([]any); ok {
		reqStrings := make([]string, len(required))
		for i, r := range required {
			if s, ok := r.(string); ok {
				reqStrings[i] = s
			}
		}
		schema.Required = reqStrings
	}

	return schema
}

// convertPropertyToSchema converts a property map to a Gemini Schema
func convertPropertyToSchema(propMap map[string]any) *genai.Schema {
	schema := &genai.Schema{}

	if typeStr, ok := propMap["type"].(string); ok {
		// Use lowercase type strings to match OpenAPI/JSON Schema spec
		switch typeStr {
		case "string":
			schema.Type = "string"
		case "number":
			schema.Type = "number"
		case "integer":
			schema.Type = "integer"
		case "boolean":
			schema.Type = "boolean"
		case "array":
			schema.Type = "array"
			if items, ok := propMap["items"].(map[string]any); ok {
				schema.Items = convertPropertyToSchema(items)
			}
		case "object":
			schema.Type = "object"
			if props, ok := propMap["properties"].(map[string]any); ok {
				schema.Properties = make(map[string]*genai.Schema)
				for name, prop := range props {
					if propMap, ok := prop.(map[string]any); ok {
						schema.Properties[name] = convertPropertyToSchema(propMap)
					}
				}
			}
		default:
			schema.Type = "string" // Default to string for unknown types
		}
	}

	if desc, ok := propMap["description"].(string); ok {
		schema.Description = desc
	}

	if enum, ok := propMap["enum"].([]any); ok {
		enumStrings := make([]string, len(enum))
		for i, e := range enum {
			if s, ok := e.(string); ok {
				enumStrings[i] = s
			}
		}
		schema.Enum = enumStrings
	}

	return schema
}

// parseImageToPart parses an image URL or data URL and returns a genai.Part for Gemini
// Handles:
// - data: URLs (base64 encoded images)
// - gs:// URLs (Google Cloud Storage - used directly)
// - http:// and https:// URLs (fetched and converted to inline data)
func parseImageToPart(image string) *genai.Part {
	// Handle data URLs (base64 encoded)
	if strings.HasPrefix(image, "data:") {
		return parseDataURLToPart(image)
	}

	// Handle Google Cloud Storage URLs - use FileData directly
	if strings.HasPrefix(image, "gs://") {
		mimeType := getMIMETypeFromURL(image)
		return &genai.Part{
			FileData: &genai.FileData{
				FileURI:  image,
				MIMEType: mimeType,
			},
		}
	}

	// Handle HTTP/HTTPS URLs - fetch and convert to inline data
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		return fetchImageAsPart(image)
	}

	// Unknown format - try to use as inline data with default MIME type
	return &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: "image/jpeg",
			Data:     []byte(image),
		},
	}
}

// parseDataURLToPart parses a data URL and extracts the MIME type and decoded bytes
// Format: data:image/jpeg;base64,/9j/4AAQ...
func parseDataURLToPart(dataURL string) *genai.Part {
	// Find the comma that separates metadata from data
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx == -1 {
		return nil
	}

	// Parse metadata (e.g., "data:image/jpeg;base64")
	metadata := dataURL[:commaIdx]
	base64Data := dataURL[commaIdx+1:]

	// Extract MIME type from metadata
	mimeType := "image/jpeg" // default
	if strings.HasPrefix(metadata, "data:") {
		metadata = metadata[5:] // remove "data:"
		if semicolonIdx := strings.Index(metadata, ";"); semicolonIdx != -1 {
			mimeType = metadata[:semicolonIdx]
		} else {
			mimeType = metadata
		}
	}

	// Decode base64 data
	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		// Try URL-safe base64
		decoded, err = base64.URLEncoding.DecodeString(base64Data)
		if err != nil {
			// Try raw base64 (no padding)
			decoded, err = base64.RawStdEncoding.DecodeString(base64Data)
			if err != nil {
				return nil
			}
		}
	}

	return &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: mimeType,
			Data:     decoded,
		},
	}
}

// fetchImageAsPart fetches an image from an HTTP/HTTPS URL and returns it as inline data
func fetchImageAsPart(imageURL string) *genai.Part {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Read the image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Determine MIME type from Content-Type header or URL
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = getMIMETypeFromURL(imageURL)
	}
	// Clean up MIME type (remove charset and other params)
	if semicolonIdx := strings.Index(mimeType, ";"); semicolonIdx != -1 {
		mimeType = strings.TrimSpace(mimeType[:semicolonIdx])
	}

	return &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: mimeType,
			Data:     data,
		},
	}
}

// getMIMETypeFromURL attempts to determine the MIME type from a URL's extension
func getMIMETypeFromURL(url string) string {
	lowerURL := strings.ToLower(url)

	// Check for common image extensions
	switch {
	case strings.HasSuffix(lowerURL, ".jpg") || strings.HasSuffix(lowerURL, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lowerURL, ".png"):
		return "image/png"
	case strings.HasSuffix(lowerURL, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lowerURL, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lowerURL, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lowerURL, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lowerURL, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(lowerURL, ".tiff") || strings.HasSuffix(lowerURL, ".tif"):
		return "image/tiff"
	case strings.HasSuffix(lowerURL, ".heic"):
		return "image/heic"
	case strings.HasSuffix(lowerURL, ".heif"):
		return "image/heif"
	case strings.HasSuffix(lowerURL, ".avif"):
		return "image/avif"
	default:
		return "image/jpeg" // default assumption
	}
}
func (g *Gemini) shouldExecuteTools(response *genai.GenerateContentResponse, enableTools bool, useMCPExecution bool) bool {
	if !enableTools || !useMCPExecution || response == nil {
		return false
	}
	functionCalls := response.FunctionCalls()
	return len(functionCalls) > 0
}

// toolPassLimit returns the maximum number of tool passes
func (g *Gemini) toolPassLimit() int {
	if g.maxToolPasses > 0 {
		return g.maxToolPasses
	}
	return defaultMaxToolPasses
}

// callAnyTool calls either a Go function tool or an MCP tool
func (g *Gemini) callAnyTool(ctx context.Context, name string, arguments map[string]any) (string, error) {
	if fn, ok := g.FunctionTools[name]; ok && fn.Handler != nil {
		return fn.Handler(ctx, FuncParams(arguments))
	}
	return g.callMCPTool(ctx, name, arguments)
}

// callMCPTool calls an MCP tool and returns the result
func (g *Gemini) callMCPTool(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	var result *mcp.ToolResult
	var err error

	if g.MultiMCPManager != nil {
		result, err = g.MultiMCPManager.CallTool(ctx, toolName, arguments)
	} else if g.MCPManager != nil {
		result, err = g.MCPManager.CallTool(ctx, toolName, arguments)
	} else {
		return "", fmt.Errorf("MCP server not configured")
	}

	if err != nil {
		return "", err
	}

	if result.IsError {
		return "", fmt.Errorf("MCP tool error %d: %s", result.ErrorCode, result.Content)
	}

	return result.Content, nil
}

// generateShortToolCallID creates a short, unique tool call ID
func generateShortToolCallID(originalID string) string {
	if len(originalID) <= 40 {
		return originalID
	}

	hash := sha256.Sum256([]byte(originalID))
	hashStr := fmt.Sprintf("%x", hash)[:24]
	prefix := originalID[:8]
	return prefix + "_" + hashStr[:23]
}

// streamAndAccumulate streams content and accumulates the response
func (g *Gemini) streamAndAccumulate(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig, chunkHandler func(*genai.GenerateContentResponse)) (*genai.GenerateContentResponse, error) {
	stream := g.Client.Models.GenerateContentStream(ctx, g.Model, contents, config)

	var accumulated *genai.GenerateContentResponse
	var accumulatedText string
	var accumulatedFunctionCalls []*genai.FunctionCall
	var lastUsageMetadata *genai.GenerateContentResponseUsageMetadata
	var mu sync.Mutex

	for chunk, err := range stream {
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		mu.Lock()
		// Accumulate text
		if chunk.Text() != "" {
			accumulatedText += chunk.Text()
		}

		// Accumulate function calls
		if fcs := chunk.FunctionCalls(); len(fcs) > 0 {
			accumulatedFunctionCalls = append(accumulatedFunctionCalls, fcs...)
		}

		// Keep track of the latest metadata
		if chunk.UsageMetadata != nil {
			lastUsageMetadata = chunk.UsageMetadata
		}

		// Keep the last chunk as base for accumulated response
		accumulated = chunk
		mu.Unlock()

		// Call the chunk handler
		if chunkHandler != nil {
			chunkHandler(chunk)
		}
	}

	if accumulated == nil {
		return nil, fmt.Errorf("no response received from stream")
	}

	// Build the final accumulated response
	finalResponse := &genai.GenerateContentResponse{
		Candidates:    accumulated.Candidates,
		CreateTime:    accumulated.CreateTime,
		ResponseID:    accumulated.ResponseID,
		ModelVersion:  accumulated.ModelVersion,
		UsageMetadata: lastUsageMetadata,
	}

	// Update the content with accumulated text and function calls
	if len(finalResponse.Candidates) > 0 {
		parts := []*genai.Part{}
		if accumulatedText != "" {
			parts = append(parts, &genai.Part{Text: accumulatedText})
		}
		for _, fc := range accumulatedFunctionCalls {
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: fc.Name,
					Args: fc.Args,
				},
			})
		}
		if len(parts) > 0 {
			finalResponse.Candidates[0].Content = &genai.Content{
				Parts: parts,
				Role:  genai.RoleModel,
			}
		}
	}

	return finalResponse, nil
}

// --- FuncParams getter methods ---

// GetString gets a string value from the parameters
func (fp FuncParams) GetString(key string) (string, bool) {
	if v, ok := fp[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

// GetStringDefault gets a string value with a default
func (fp FuncParams) GetStringDefault(key, defaultValue string) string {
	if s, ok := fp.GetString(key); ok {
		return s
	}
	return defaultValue
}

// GetInt gets an integer value from the parameters
func (fp FuncParams) GetInt(key string) (int, bool) {
	if v, ok := fp[key]; ok {
		switch n := v.(type) {
		case int:
			return n, true
		case int64:
			return int(n), true
		case float64:
			return int(n), true
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return int(i), true
			}
		}
	}
	return 0, false
}

// GetIntDefault gets an integer value with a default
func (fp FuncParams) GetIntDefault(key string, defaultValue int) int {
	if i, ok := fp.GetInt(key); ok {
		return i
	}
	return defaultValue
}

// GetFloat gets a float64 value from the parameters
func (fp FuncParams) GetFloat(key string) (float64, bool) {
	if v, ok := fp[key]; ok {
		switch n := v.(type) {
		case float64:
			return n, true
		case float32:
			return float64(n), true
		case int:
			return float64(n), true
		case int64:
			return float64(n), true
		case json.Number:
			if f, err := n.Float64(); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

// GetFloatDefault gets a float64 value with a default
func (fp FuncParams) GetFloatDefault(key string, defaultValue float64) float64 {
	if f, ok := fp.GetFloat(key); ok {
		return f
	}
	return defaultValue
}

// GetBool gets a boolean value from the parameters
func (fp FuncParams) GetBool(key string) (bool, bool) {
	if v, ok := fp[key]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// GetBoolDefault gets a boolean value with a default
func (fp FuncParams) GetBoolDefault(key string, defaultValue bool) bool {
	if b, ok := fp.GetBool(key); ok {
		return b
	}
	return defaultValue
}

// GetStringArray gets a string array from the parameters
func (fp FuncParams) GetStringArray(key string) ([]string, bool) {
	if v, ok := fp[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result, true
		}
		if arr, ok := v.([]string); ok {
			return arr, true
		}
	}
	return nil, false
}

// GetMap gets a nested map from the parameters
func (fp FuncParams) GetMap(key string) (FuncParams, bool) {
	if v, ok := fp[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return FuncParams(m), true
		}
	}
	return nil, false
}
