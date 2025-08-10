package ai

import (
	"maps"
	"strconv"
	"strings"

	"github.com/MelloB1989/karma/models"
	"github.com/posthog/posthog-go"
)

const AIGenerationEvent string = "$ai_generation"

// AIProperty represents the keys used for AI event tracking
type AIProperty string

const (
	// Posthog LLM observability properties
	AITraceID       AIProperty = "$ai_trace_id"       // The trace ID (UUID to group AI events), similar to conversation_id. Must contain only letters, numbers, and special characters: -, _, ~, ., @, (, ), !, ', :, |
	AIModel         AIProperty = "$ai_model"          // The model used (e.g., gpt-3.5-turbo)
	AIProvider      AIProperty = "$ai_provider"       // The LLM provider name
	AIInput         AIProperty = "$ai_input"          // List of messages sent to the LLM (e.g., [{"role": "user", "content": "..."}])
	AIInputTokens   AIProperty = "$ai_input_tokens"   // The number of tokens in the input (from response.usage)
	AIOutputChoices AIProperty = "$ai_output_choices" // List of choices returned by the LLM (e.g., [{"role": "assistant", "content": "..."}])
	AIOutputTokens  AIProperty = "$ai_output_tokens"  // The number of tokens in the output (from response.usage)
	AILatency       AIProperty = "$ai_latency"        // The latency of the LLM call in seconds
	AIHTTPStatus    AIProperty = "$ai_http_status"    // The HTTP status code of the LLM response
	AIBaseURL       AIProperty = "$ai_base_url"       // The base URL of the LLM provider
	AIIsError       AIProperty = "$ai_is_error"       // Boolean indicating whether the request resulted in an error
	AIError         AIProperty = "$ai_error"          // The error message or object if the request failed

	// Custom properties
	SystemPrompt    AIProperty = "$kai_system_prompt"
	ToolCallEnabled AIProperty = "$kai_tool_call_enabled"
	McpServerUrls   AIProperty = "$kai_mcp_server_urls"
	Temperature     AIProperty = "$kai_temperature"
	TopP            AIProperty = "$kai_top_p"
	TopK            AIProperty = "$kai_top_k"
	MaxTokens       AIProperty = "$kai_max_tokens"
)

func (kai *KarmaAI) captureResponse(mgs models.AIChatHistory, res models.AIChatResponse) {
	// Run analytics capture in a goroutine to avoid blocking the response
	go func() {
		// Check if analytics is enabled
		if !kai.Analytics.on || kai.Analytics.client == nil {
			return
		}

		kai.SetAnalyticProperty(AIInputTokens, res.InputTokens)
		kai.SetAnalyticProperty(AIOutputTokens, res.OutputTokens)
		kai.SetAnalyticProperty(AILatency, res.TimeTaken)
		kai.SetAnalyticProperty(AIIsError, false) // Mark as successful response

		if kai.Analytics.CaptureUserPrompts && len(mgs.Messages) >= 1 {
			kai.SetAnalyticProperty(AIInput, mgs.Messages[len(mgs.Messages)-1].Message)
		}
		if kai.Analytics.CaptureAIResponses {
			kai.SetAnalyticProperty(AIOutputChoices, res.AIResponse)
		}

		// Send the event after setting all properties
		kai.SendEvent()
	}()
}

func (kai *KarmaAI) setBasicProperties() {
	// Check if analytics is enabled
	if !kai.Analytics.on || kai.Analytics.client == nil {
		return
	}

	kai.SetAnalyticProperty(AITraceID, kai.Analytics.TraceId)
	kai.SetAnalyticProperty(AIModel, kai.Model)
	kai.SetAnalyticProperty(AIProvider, kai.Model.GetModelProvider())
	kai.SetAnalyticProperty(SystemPrompt, kai.SystemMessage)

	if kai.ToolsEnabled {
		kai.SetAnalyticProperty(ToolCallEnabled, strconv.FormatBool(kai.ToolsEnabled))
		server_urls := []string{}
		server_urls = append(server_urls, kai.MCPConfig.MCPUrl)
		for _, server := range kai.MCPServers {
			server_urls = append(server_urls, server.URL)
		}
		kai.SetAnalyticProperty(McpServerUrls, strings.Join(server_urls, ","))
	}
	kai.SetAnalyticProperty(Temperature, strconv.FormatFloat(kai.Temperature, 'f', -1, 64))
	kai.SetAnalyticProperty(TopP, strconv.FormatFloat(kai.TopP, 'f', -1, 64))
	kai.SetAnalyticProperty(TopK, strconv.Itoa(int(kai.TopK)))
	kai.SetAnalyticProperty(MaxTokens, strconv.Itoa(int(kai.MaxTokens)))
}

func (kai *KarmaAI) SendEvent() {
	kai.Analytics.mu.RLock()
	propertiesCopy := make(map[string]any)
	if kai.Analytics.properties != nil {
		maps.Copy(propertiesCopy, kai.Analytics.properties)
	}
	kai.Analytics.mu.RUnlock()

	if kai.Analytics.client != nil {
		kai.Analytics.client.Enqueue(posthog.Capture{
			DistinctId: kai.Analytics.DistinctID,
			Event:      AIGenerationEvent,
			Properties: propertiesCopy,
		})
	}
}

func (kai *KarmaAI) SendErrorEvent(err error) {
	// Run error event capture in a goroutine to avoid blocking the response
	go func() {
		// Check if analytics is enabled
		if !kai.Analytics.on || kai.Analytics.client == nil {
			return
		}

		kai.SetAnalyticProperty(AIError, err)
		kai.SetAnalyticProperty(AIIsError, true)

		kai.Analytics.mu.RLock()
		propertiesCopy := make(map[string]any)
		if kai.Analytics.properties != nil {
			maps.Copy(propertiesCopy, kai.Analytics.properties)
		}
		kai.Analytics.mu.RUnlock()

		if kai.Analytics.client != nil {
			kai.Analytics.client.Enqueue(posthog.Capture{
				DistinctId: kai.Analytics.DistinctID,
				Event:      AIGenerationEvent,
				Properties: propertiesCopy,
			})
		}

		kai.DeleteAnalyticProperty(AIError)
		kai.DeleteAnalyticProperty(AIIsError)
	}()
}

func (kai *KarmaAI) SetAnalyticProperty(property AIProperty, val any) {
	kai.Analytics.mu.Lock()
	defer kai.Analytics.mu.Unlock()
	if kai.Analytics.properties == nil {
		kai.Analytics.properties = make(map[string]any)
	}
	kai.Analytics.properties[string(property)] = val
}

func (kai *KarmaAI) DeleteAnalyticProperty(property AIProperty) {
	kai.Analytics.mu.Lock()
	defer kai.Analytics.mu.Unlock()
	if kai.Analytics.properties != nil {
		delete(kai.Analytics.properties, string(property))
	}
}
