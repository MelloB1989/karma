/*
 * This file adds extenders to KarmaAI package, adding managed memory functionality.
 */
package memory

import (
	"time"

	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"go.uber.org/zap"
)

func (km *KarmaMemory) ChatCompletion(prompt string) (*models.AIChatResponse, error) {
	history := &km.messagesHistory
	memoryContext, err := km.GetContext(prompt)
	if err != nil {
		km.logger.Error("Memory retrieval failed, continuing without memory context", zap.Error(err))
		memoryContext = ""
	}

	userContent := prompt
	if memoryContext != "" {
		userContent = memoryContext + "\n" + prompt
	}

	userMsgIndex := len(history.Messages)
	history.Messages = append(history.Messages, models.AIMessage{
		Message:   userContent,
		Role:      models.User,
		Timestamp: time.Now(),
		UniqueId:  utils.GenerateID(6),
	})

	res, err := km.kai.ChatCompletionManaged(history)
	if err != nil {
		return nil, err
	}

	// Restore original prompt without memory context
	history.Messages[userMsgIndex].Message = prompt
	km.UpdateMessageHistory(history.Messages)

	return res, nil
}

func (km *KarmaMemory) ChatCompletionStream(prompt string, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	history := &km.messagesHistory
	// Add memory context
	memoryContext, err := km.GetContext(prompt)
	if err != nil {
		km.logger.Error("Memory retrieval failed, continuing without memory context", zap.Error(err))
		memoryContext = ""
	}

	userContent := prompt
	if memoryContext != "" {
		userContent = memoryContext + "\n" + prompt
	}

	userMsgIndex := len(history.Messages)

	history.Messages = append(history.Messages, models.AIMessage{
		Message:   userContent,
		Role:      models.User,
		Timestamp: time.Now(),
		UniqueId:  utils.GenerateID(6),
	})

	res, err := km.kai.ChatCompletionStreamManaged(history, callback)
	if err != nil {
		return nil, err
	}

	// Restore original prompt without memory context
	history.Messages[userMsgIndex].Message = prompt
	// Update to trigger memory updates
	km.UpdateMessageHistory(history.Messages)
	return res, nil
}
