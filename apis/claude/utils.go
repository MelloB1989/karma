package claude

import (
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
)

func processMessages(messages models.AIChatHistory) []anthropic.MessageParam {
	var processedMessages []anthropic.MessageParam
	for _, msg := range messages.Messages {
		var role anthropic.MessageParamRole
		if msg.Role == models.User {
			role = anthropic.MessageParamRoleUser
		} else {
			role = anthropic.MessageParamRoleAssistant
		}
		processedMessages = append(processedMessages, anthropic.MessageParam{
			Role: role,
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: msg.Message},
			}},
		})
	}
	return processedMessages
}
