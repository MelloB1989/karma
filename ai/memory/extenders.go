/*
 * This file adds extenders to KarmaAI package, adding managed memory functionality.
 */
package memory

import (
	"time"

	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
)

func (km *KarmaMemory) ChatCompletion() (*models.AIChatResponse, error) {
	res, err := km.kai.ChatCompletion(km.messagesHistory)
	if err != nil {
		return nil, err
	}
	m := append(km.messagesHistory.Messages, models.AIMessage{
		Message:   res.AIResponse,
		Role:      models.Assistant,
		Timestamp: time.Now(),
		UniqueId:  utils.GenerateID(6),
	})
	// Update to trigger memory updates
	km.UpdateMessageHistory(m)
	return res, nil
}

func (km *KarmaMemory) ChatCompletionStream(callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	res, err := km.kai.ChatCompletionStream(km.messagesHistory, callback)
	if err != nil {
		return nil, err
	}
	m := append(km.messagesHistory.Messages, models.AIMessage{
		Message:   res.AIResponse,
		Role:      models.Assistant,
		Timestamp: time.Now(),
		UniqueId:  utils.GenerateID(6),
	})
	// Update to trigger memory updates
	km.UpdateMessageHistory(m)
	return res, nil
}
