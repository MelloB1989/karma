package bedrock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MelloB1989/karma/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// ConverseParams describes a single Converse / ConverseStream invocation. It is
// model-agnostic — the Bedrock Converse API normalizes inference parameters
// across providers (Anthropic, Meta, Amazon, Cohere, Mistral, …).
type ConverseParams struct {
	// ModelID is a model ID, ARN, or inference-profile ARN.
	ModelID string
	// System is the system prompt. Empty means no system block.
	System string
	// History is the conversation. User and assistant turns are mapped to
	// Converse messages; other roles are ignored.
	History models.AIChatHistory

	MaxTokens     int
	Temperature   float32
	TopP          float32
	TopK          int
	StopSequences []string

	// APIKey optionally overrides the Bedrock API key (bearer token). When
	// empty, AWS_BEARER_TOKEN_BEDROCK / AWS credentials are used.
	APIKey string
	// Region optionally overrides the AWS region.
	Region string
}

// ConverseResult is the normalized outcome of a Converse / ConverseStream call.
type ConverseResult struct {
	Text         string
	StopReason   string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	LatencyMs    int
}

// Converse performs a non-streaming Bedrock Converse request using the official
// AWS SDK v2.
func Converse(ctx context.Context, params ConverseParams) (*ConverseResult, error) {
	client, err := NewRuntimeClient(ctx, ClientOptions{Region: params.Region, APIKey: params.APIKey})
	if err != nil {
		return nil, err
	}

	input, err := buildConverseInput(params)
	if err != nil {
		return nil, err
	}

	out, err := client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId:                      input.ModelId,
		Messages:                     input.Messages,
		System:                       input.System,
		InferenceConfig:              input.InferenceConfig,
		AdditionalModelRequestFields: input.AdditionalModelRequestFields,
	})
	if err != nil {
		return nil, fmt.Errorf("bedrock converse failed: %w", err)
	}

	result := &ConverseResult{StopReason: string(out.StopReason)}

	if msg, ok := out.Output.(*types.ConverseOutputMemberMessage); ok {
		result.Text = extractText(msg.Value.Content)
	}
	if out.Usage != nil {
		result.InputTokens = int(aws.ToInt32(out.Usage.InputTokens))
		result.OutputTokens = int(aws.ToInt32(out.Usage.OutputTokens))
		result.TotalTokens = int(aws.ToInt32(out.Usage.TotalTokens))
	}
	if out.Metrics != nil {
		result.LatencyMs = int(aws.ToInt64(out.Metrics.LatencyMs))
	}

	return result, nil
}

// ConverseStream performs a streaming Bedrock Converse request. onText is
// invoked for each text delta as it arrives. The returned ConverseResult carries
// the aggregated text plus final usage / stop reason from the metadata event.
func ConverseStream(ctx context.Context, params ConverseParams, onText func(text string) error) (*ConverseResult, error) {
	client, err := NewRuntimeClient(ctx, ClientOptions{Region: params.Region, APIKey: params.APIKey})
	if err != nil {
		return nil, err
	}

	input, err := buildConverseInput(params)
	if err != nil {
		return nil, err
	}

	out, err := client.ConverseStream(ctx, &bedrockruntime.ConverseStreamInput{
		ModelId:                      input.ModelId,
		Messages:                     input.Messages,
		System:                       input.System,
		InferenceConfig:              input.InferenceConfig,
		AdditionalModelRequestFields: input.AdditionalModelRequestFields,
	})
	if err != nil {
		return nil, fmt.Errorf("bedrock converse stream failed: %w", err)
	}

	stream := out.GetStream()
	defer stream.Close()

	result := &ConverseResult{}
	for event := range stream.Events() {
		switch e := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockDelta:
			if delta, ok := e.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
				result.Text += delta.Value
				if onText != nil {
					if err := onText(delta.Value); err != nil {
						return result, err
					}
				}
			}
		case *types.ConverseStreamOutputMemberMessageStop:
			result.StopReason = string(e.Value.StopReason)
		case *types.ConverseStreamOutputMemberMetadata:
			if e.Value.Usage != nil {
				result.InputTokens = int(aws.ToInt32(e.Value.Usage.InputTokens))
				result.OutputTokens = int(aws.ToInt32(e.Value.Usage.OutputTokens))
				result.TotalTokens = int(aws.ToInt32(e.Value.Usage.TotalTokens))
			}
			if e.Value.Metrics != nil {
				result.LatencyMs = int(aws.ToInt64(e.Value.Metrics.LatencyMs))
			}
		}
	}

	if err := stream.Err(); err != nil {
		return result, fmt.Errorf("bedrock converse stream error: %w", err)
	}

	return result, nil
}

// buildConverseInput assembles the shared Converse request fields from params.
func buildConverseInput(params ConverseParams) (*bedrockruntime.ConverseInput, error) {
	if params.ModelID == "" {
		return nil, errors.New("bedrock: model ID is required")
	}

	messages := mapMessages(params.History)
	if len(messages) == 0 {
		return nil, errors.New("bedrock: no user/assistant messages to send")
	}

	inference := &types.InferenceConfiguration{}
	if params.MaxTokens > 0 {
		inference.MaxTokens = aws.Int32(int32(params.MaxTokens))
	}
	if params.Temperature > 0 {
		inference.Temperature = aws.Float32(params.Temperature)
	}
	// Anthropic (Claude) models on Bedrock reject temperature and top_p being
	// specified together. Since karma sets non-zero defaults for both, prefer
	// temperature and drop top_p for those models to avoid a ValidationException.
	if params.TopP > 0 && !(isAnthropicModel(params.ModelID) && inference.Temperature != nil) {
		inference.TopP = aws.Float32(params.TopP)
	}
	if len(params.StopSequences) > 0 {
		inference.StopSequences = params.StopSequences
	}

	input := &bedrockruntime.ConverseInput{
		ModelId:         aws.String(params.ModelID),
		Messages:        messages,
		InferenceConfig: inference,
	}

	if params.System != "" {
		input.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: params.System},
		}
	}

	// top_k is not part of the normalized inferenceConfig; it is passed through
	// additionalModelRequestFields (as in the AWS console Converse examples).
	if params.TopK > 0 {
		input.AdditionalModelRequestFields = document.NewLazyDocument(map[string]interface{}{
			"top_k": params.TopK,
		})
	}

	return input, nil
}

// isAnthropicModel reports whether the model ID refers to an Anthropic (Claude)
// model, including inference-profile ARNs and cross-region prefixes such as
// "us.anthropic.*" or "global.anthropic.*".
func isAnthropicModel(modelID string) bool {
	id := strings.ToLower(modelID)
	return strings.Contains(id, "anthropic") || strings.Contains(id, "claude")
}

// mapMessages converts karma chat history into Converse messages, merging
// consecutive turns with the same role into a single message (Converse requires
// strictly alternating user/assistant turns).
func mapMessages(history models.AIChatHistory) []types.Message {
	var out []types.Message
	var lastRole types.ConversationRole
	var current []types.ContentBlock

	flush := func() {
		if len(current) > 0 {
			out = append(out, types.Message{Role: lastRole, Content: current})
			current = nil
		}
	}

	for _, msg := range history.Messages {
		var role types.ConversationRole
		switch msg.Role {
		case models.User:
			role = types.ConversationRoleUser
		case models.Assistant:
			role = types.ConversationRoleAssistant
		default:
			continue // system/tool/function roles are handled elsewhere or unsupported here
		}

		if msg.Message == "" {
			continue
		}

		if role != lastRole {
			flush()
			lastRole = role
		}
		current = append(current, &types.ContentBlockMemberText{Value: msg.Message})
	}
	flush()

	return out
}

// extractText concatenates the text content blocks of a Converse message.
func extractText(blocks []types.ContentBlock) string {
	var text string
	for _, block := range blocks {
		if t, ok := block.(*types.ContentBlockMemberText); ok {
			text += t.Value
		}
	}
	return text
}
