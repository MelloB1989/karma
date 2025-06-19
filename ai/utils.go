package ai

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MelloB1989/karma/apis/claude"
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

func (kai *KarmaAI) processMessagesForLlamaBedrockSinglePrompt(prompt string) string {
	return fmt.Sprintf(llama_single_prompt_format, time.Now().String(), kai.SystemMessage, prompt)
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
	if len(kai.MCPConfig.MCPTools) > 0 {
		cc.SetMCPServer(kai.MCPConfig.MCPUrl, kai.MCPConfig.AuthToken)
		for _, tool := range kai.MCPConfig.MCPTools {
			err := cc.AddMCPTool(tool.FriendlyName, tool.Description, tool.ToolName, tool.InputSchema)
			if err != nil {
				log.Printf("Failed to add MCP tool: %v", err)
			}
		}
	}
}
