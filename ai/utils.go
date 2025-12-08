package ai

import (
	"fmt"
	"log"
	"strings"
	"time"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/apis/claude"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
)

const llama_single_prompt_format = `
	<|begin_of_text|><|start_header_id|>system<|end_header_id|>

Cutting Knowledge Date: December 2023
Today Date: %s

%s<|eot_id|><|start_header_id|>user<|end_header_id|>

%s<|eot_id|><|start_header_id|>assistant<|end_header_id|>
`

const llama_system_prompt_format = `
<|begin_of_text|><|start_header_id|>system<|end_header_id|>

Cutting Knowledge Date: December 2023
Today Date: %s

%s<|eot_id|>
`

const user_message_format = `
<|start_header_id|>user<|end_header_id|>

%s<|eot_id|>
`

const assistant_message_format = `
<|start_header_id|>assistant<|end_header_id|>

%s<|eot_id|>
`

const role_message_format = `
<|start_header_id|>%s<|end_header_id|>

%s<|eot_id|>
`

const assitant_end = `
<|start_header_id|>assistant<|end_header_id|>
`

func (kai *KarmaAI) addUserPreprompt(chat models.AIChatHistory) models.AIChatHistory {
	if len(chat.Messages) == 0 {
		return chat
	}
	chat.Messages[len(chat.Messages)-1].Message = kai.UserPrePrompt + "\n" + chat.Messages[len(chat.Messages)-1].Message
	return chat
}

func (kai *KarmaAI) processMessagesForLlamaBedrockSystemPrompt(chat models.AIChatHistory) string {
	var finalPrompt strings.Builder
	finalPrompt.WriteString(fmt.Sprintf(llama_system_prompt_format, time.Now().String(), kai.SystemMessage))
	for _, message := range chat.Messages {
		if message.Role == models.User {
			finalPrompt.WriteString(fmt.Sprintf(user_message_format, message.Message))
		} else if message.Role == models.Assistant {
			finalPrompt.WriteString(fmt.Sprintf(assistant_message_format, message.Message))
		} else {
			finalPrompt.WriteString(fmt.Sprintf(role_message_format, message.Role, message.Message))
		}
	}
	finalPrompt.WriteString(assitant_end)

	return finalPrompt.String()
}

func (kai *KarmaAI) configureClaudeClientForMCP(cc *claude.ClaudeClient) {
	if len(kai.MCPServers) > 0 {
		kai.configureMultiMCPForClaude(cc)
	} else if len(kai.MCPTools) > 0 {
		cc.SetMCPServer(kai.MCPUrl, kai.AuthToken)
		for _, tool := range kai.MCPTools {
			err := cc.AddMCPTool(tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema) // Claude requires tool names to match ^[a-zA-Z0-9_-]{1,128}$ (letters, numbers, underscore, hyphen only)
			if err != nil {
				log.Printf("Failed to add MCP tool: %v", err)
			}
		}
	}
}

func (kai *KarmaAI) configureOpenaiClientForMCP(o *openai.OpenAI) {
	if len(kai.MCPServers) > 0 {
		kai.configureMultiMCPForOpenAI(o)
	} else if len(kai.MCPTools) > 0 {
		o.SetMCPServer(kai.MCPUrl, kai.AuthToken)
		for _, tool := range kai.MCPTools {
			err := o.AddMCPTool(tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema) // Claude requires tool names to match ^[a-zA-Z0-9_-]{1,128}$ (letters, numbers, underscore, hyphen only)
			if err != nil {
				log.Printf("Failed to add MCP tool: %v", err)
			}
		}
	}
}

func (kai *KarmaAI) configureMultiMCPForOpenAI(o *openai.OpenAI) {
	multiManager := mcp.NewMultiManager()

	for i, server := range kai.MCPServers {
		serverID := fmt.Sprintf("server_%d", i)
		multiManager.AddServer(serverID, server.URL, server.AuthToken)

		for _, tool := range server.Tools {
			err := multiManager.AddToolToServer(serverID, tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema)
			if err != nil {
				log.Printf("Failed to add MCP tool %s to server %s: %v", tool.FriendlyName, serverID, err)
			}
		}
	}

	o.SetMultiMCPManager(multiManager)
}

func (kai *KarmaAI) configureMultiMCPForClaude(cc *claude.ClaudeClient) {
	multiManager := mcp.NewMultiManager()

	for i, server := range kai.MCPServers {
		serverID := fmt.Sprintf("server_%d", i)
		multiManager.AddServer(serverID, server.URL, server.AuthToken)

		for _, tool := range server.Tools {
			err := multiManager.AddToolToServer(serverID, tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema)
			if err != nil {
				log.Printf("Failed to add MCP tool %s to server %s: %v", tool.FriendlyName, serverID, err)
			}
		}
	}

	cc.SetMultiMCPManager(multiManager)
}
