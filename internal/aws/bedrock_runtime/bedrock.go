package bedrock_runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

// BedrockRequest represents the structure of the Bedrock API request body.
type BedrockRequest struct {
	AdditionalModelRequestFields map[string]interface{} `json:"additionalModelRequestFields"`
	InferenceConfig              InferenceConfig        `json:"inferenceConfig"`
	Messages                     []Message              `json:"messages"`
	System                       []SystemMessage        `json:"system"`
}

// InferenceConfig defines the inference configurations.
type InferenceConfig struct {
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"topP"`
}

// Message represents a single message in the messages array.
type Message struct {
	Content []Content `json:"content"`
	Role    string    `json:"role"`
}

// Content represents the content of a message.
type Content struct {
	Text string `json:"text"`
}

// SystemMessage represents a system-level message.
type SystemMessage struct {
	Text string `json:"text"`
}

func ProcessChatMessages(messages models.AIChatHistory) []Message {
	processedMessages := []Message{}
	for _, message := range messages.Messages {
		processedMessages = append(processedMessages, Message{
			Content: []Content{
				{
					Text: message.Message,
				},
			},
			Role: string(message.Role),
		})
	}
	return processedMessages
}

func CreateBedrockRequest(maxTokens int, Temperature, TopP float64, messages models.AIChatHistory, systemMessage string) BedrockRequest {
	return BedrockRequest{
		AdditionalModelRequestFields: map[string]interface{}{},
		InferenceConfig: InferenceConfig{
			MaxTokens:   maxTokens,
			Temperature: Temperature,
			TopP:        TopP,
		},
		Messages: ProcessChatMessages(messages),
		System: []SystemMessage{
			{
				Text: systemMessage,
			},
		},
	}
}

func InvokeBedrockConverseAPI(modelIdentifier string, requestBody BedrockRequest) ([]byte, error) {
	// Create AWS credentials
	creds := credentials.NewStaticCredentials(config.DefaultConfig().AwsAccessKey, config.DefaultConfig().AwsSecretKey, "")
	region, _ := config.GetEnv("AWS_BEDROCK_REGION")
	// Initialize a session
	if region != "" {
		_, err := session.NewSession(&aws.Config{
			Credentials: creds,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS session: %v", err)
		}
	} else {
		_, err := session.NewSession(&aws.Config{
			Region:      aws.String(region),
			Credentials: creds,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS session: %v", err)
		}
	}

	// Construct the endpoint URL
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/converse", region, modelIdentifier)

	// Marshal the request body to JSON
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body to JSON: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign the request using SigV4
	signer := v4.NewSigner(creds)
	_, err = signer.Sign(req, bytes.NewReader(payload), "bedrock", region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %v", err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseBody, fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
	}

	return responseBody, nil
}
