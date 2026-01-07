package gemini

import (
	"context"
	"fmt"

	"github.com/MelloB1989/karma/config"
	"google.golang.org/genai"
)

func RunGemini(prompt, model, system_prompt string, temp, topP, topK float64, maxOutputTokens int64, response_type ...string) (*genai.GenerateContentResponse, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		// APIKey: config.GetEnvRaw("GOOGLE_API_KEY"),
		Backend:  genai.BackendVertexAI,
		Project:  config.GetEnvRaw("GOOGLE_PROJECT_ID"),
		Location: config.GetEnvRaw("GOOGLE_LOCATION"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	var res string

	if len(response_type) > 0 {
		res = response_type[0]
	} else {
		res = "text/plain"
	}

	// Float64 to Float32 conversion
	temp32 := float32(temp)
	topP32 := float32(topP)
	topK32 := float32(topK)
	// int64 to int32 conversion
	maxOutputTokens32 := int32(maxOutputTokens)

	config := &genai.GenerateContentConfig{
		Temperature:       &temp32,
		TopP:              &topP32,
		TopK:              &topK32,
		MaxOutputTokens:   maxOutputTokens32,
		ResponseMIMEType:  res,
		SystemInstruction: genai.NewContentFromText(system_prompt, genai.RoleModel),
	}

	result, err := client.Models.GenerateContent(
		ctx,
		model,
		genai.Text(prompt),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// fmt.Println(model, result.Text())
	return result, nil
}
