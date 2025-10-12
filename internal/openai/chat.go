package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go/v2"
)

type OpenAI struct {
	Client          openai.Client
	Model           string
	Temperature     float64
	MaxTokens       int64
	SystemMessage   string
	ExtraFields     map[string]any
	MCPManager      *mcp.Manager
	MultiMCPManager *mcp.MultiManager
}

func NewOpenAI(model, sysmgs string, temperature float64, maxTokens int64) *OpenAI {
	return &OpenAI{
		Client:        createClient(),
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: sysmgs,
		MCPManager:    nil,
	}
}

func NewOpenAICompatible(model, sysmgs string, temperature float64, maxTokens int64, base_url, apikey string) *OpenAI {
	return &OpenAI{
		Client: createClient(CompatibleOptions{
			BaseURL: base_url,
			API_Key: apikey,
		}),
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: sysmgs,
		MCPManager:    nil,
	}
}

// SetMCPServer configures the MCP server and creates a tool manager
func (o *OpenAI) SetMCPServer(serverURL string, authToken string) {
	mcpClient := mcp.NewClient(serverURL, authToken)
	o.MCPManager = mcp.NewManager(mcpClient)
}

// SetMultiMCPManager configures multiple MCP servers
func (o *OpenAI) SetMultiMCPManager(multiManager *mcp.MultiManager) {
	o.MultiMCPManager = multiManager
}

// AddMCPTool adds an MCP tool that Claude can use
func (o *OpenAI) AddMCPTool(name, description, mcpToolName string, inputSchema any) error {
	if o.MCPManager == nil {
		return fmt.Errorf("MCP server not configured. Call SetMCPServer first")
	}
	return o.MCPManager.AddToolFromSchema(name, description, mcpToolName, inputSchema)
}

// GetMCPManager returns the MCP manager for advanced tool management
func (o *OpenAI) GetMCPManager() *mcp.Manager {
	return o.MCPManager
}

func (o *OpenAI) CreateChat(messages models.AIChatHistory, enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	mgs := formatMessages(messages, o.SystemMessage)

	params := openai.ChatCompletionNewParams{
		Model:    o.Model,
		Messages: mgs,
		Seed:     openai.Int(69),
	}

	params.SetExtraFields(o.ExtraFields)

	if o.Temperature > 0 {
		params.Temperature = openai.Float(o.Temperature)
	}
	if o.MaxTokens > 0 {
		if strings.Contains(o.Model, "gpt-5") { //Special handling for GPT-5 models
			params.MaxCompletionTokens = openai.Int(o.MaxTokens)
		} else {
			params.MaxTokens = openai.Int(o.MaxTokens)
		}
	}

	// Add MCP tools if enabled and available
	if enableTools && o.hasMCPTools() {
		params.Tools = o.convertMCPToolsToOpenAI()
	}

	ctx := context.TODO()
	for {
		chatCompletion, err := o.Client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, err
		}

		// Check if OpenAI wants to use tools
		if len(chatCompletion.Choices) == 0 || len(chatCompletion.Choices[0].Message.ToolCalls) == 0 {
			return chatCompletion, nil
		}

		// If not using MCP execution, return immediately with tool calls for external handling
		if !useMCPExecution {
			return chatCompletion, nil
		}

		// Follow v2 API pattern - add assistant message with tool calls
		params.Messages = append(params.Messages, chatCompletion.Choices[0].Message.ToParam())

		// Create mapping of original to short IDs for tool responses
		idMapping := make(map[string]string)
		for _, toolCall := range chatCompletion.Choices[0].Message.ToolCalls {
			shortID := generateShortToolCallID(toolCall.ID)
			idMapping[toolCall.ID] = shortID
		}

		for _, toolCall := range chatCompletion.Choices[0].Message.ToolCalls {
			if enableTools {
				shortID := idMapping[toolCall.ID]

				// Parse tool call arguments
				var arguments map[string]any
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments)
				if err != nil {
					params.Messages = append(params.Messages, openai.ToolMessage(
						fmt.Sprintf("Error parsing arguments: %v", err),
						shortID,
					))
					continue
				}

				// Call the MCP tool
				result, err := o.callMCPTool(ctx, toolCall.Function.Name, arguments)
				if err != nil {
					fmt.Printf("MCP tool error: %v\n", err)
					params.Messages = append(params.Messages, openai.ToolMessage(
						fmt.Sprintf("Error calling tool: %v", err),
						shortID,
					))
				} else {
					params.Messages = append(params.Messages, openai.ToolMessage(result, shortID))
				}
			}
		}

		// Remove tools for the follow-up to avoid loops
		// params.Tools = []openai.ChatCompletionToolUnionParam{}
	}
}

func (o *OpenAI) CreateChatStream(messages models.AIChatHistory, chunkHandler func(chuck openai.ChatCompletionChunk), enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	mgs := formatMessages(messages, o.SystemMessage)

	params := openai.ChatCompletionNewParams{
		Model:    o.Model,
		Messages: mgs,
	}

	params.SetExtraFields(o.ExtraFields)

	if o.Temperature > 0 {
		params.Temperature = openai.Float(o.Temperature)
	}
	if o.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(o.MaxTokens)
	}

	// Add MCP tools if enabled and available
	if enableTools && o.hasMCPTools() {
		params.Tools = o.convertMCPToolsToOpenAI()
	}

	ctx := context.TODO()
	stream := o.Client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		chunkHandler(chunk)
	}

	if err := stream.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	// Handle tool calls if any - follow v2 API pattern
	if enableTools && useMCPExecution && len(acc.Choices) > 0 && len(acc.Choices[0].Message.ToolCalls) > 0 {
		followUpParams := params
		followUpParams.Messages = append(followUpParams.Messages, acc.Choices[0].Message.ToParam())

		// Create mapping of original to short IDs for tool responses
		idMapping := make(map[string]string)
		for _, toolCall := range acc.Choices[0].Message.ToolCalls {
			shortID := generateShortToolCallID(toolCall.ID)
			idMapping[toolCall.ID] = shortID
		}

		for _, toolCall := range acc.Choices[0].Message.ToolCalls {
			shortID := idMapping[toolCall.ID]

			// Parse tool call arguments
			var arguments map[string]any
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments)
			if err != nil {
				followUpParams.Messages = append(followUpParams.Messages, openai.ToolMessage(
					fmt.Sprintf("Error parsing arguments: %v", err),
					shortID,
				))
				continue
			}

			// Call the MCP tool
			result, err := o.callMCPTool(ctx, toolCall.Function.Name, arguments)
			if err != nil {
				fmt.Printf("MCP tool error: %v\n", err)
				followUpParams.Messages = append(followUpParams.Messages, openai.ToolMessage(
					fmt.Sprintf("Error calling tool: %v", err),
					shortID,
				))
			} else {
				followUpParams.Messages = append(followUpParams.Messages, openai.ToolMessage(result, shortID))
			}
		}

		// Disable tools for follow-up to avoid loops
		// followUpParams.Tools = []openai.ChatCompletionToolUnionParam{}

		followUpStream := o.Client.Chat.Completions.NewStreaming(ctx, followUpParams)
		for followUpStream.Next() {
			chunk := followUpStream.Current()
			chunkHandler(chunk)
		}
		if followUpStream.Err() != nil {
			return nil, followUpStream.Err()
		}
	}

	// After the stream is finished, acc can be used like a ChatCompletion
	return &acc.ChatCompletion, nil
}
