package bedrock_runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

type Metrics struct {
	LatencyMs float64 `json:"latencyMs"`
}

type Output struct {
	Message Message `json:"message"`
}

type Usage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type BedrockResponse struct {
	Metrics    Metrics `json:"metrics"`
	Output     Output  `json:"output"`
	StopReason string  `json:"stopReason"`
	Usage      Usage   `json:"usage"`
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

func ProcessChatMessages(history models.AIChatHistory) []Message {
	processedMessages := []Message{}

	var lastRole string
	var currentContent []Content

	for _, msg := range history.Messages {
		if msg.Role != models.User && msg.Role != models.Assistant {
			// Skip unknown roles or handle accordingly
			continue
		}

		if string(msg.Role) != lastRole {
			// If role changes, append the previous group to processedMessages
			if len(currentContent) > 0 {
				processedMessages = append(processedMessages, Message{
					Content: currentContent,
					Role:    string(lastRole),
				})
				// Reset currentContent for the new role
				currentContent = []Content{}
			}
			lastRole = string(msg.Role)
		}

		// Append the current message's text to the currentContent
		currentContent = append(currentContent, Content{
			Text: msg.Message,
		})
	}

	// Append any remaining messages after the loop
	if len(currentContent) > 0 {
		processedMessages = append(processedMessages, Message{
			Content: currentContent,
			Role:    string(lastRole),
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

func InvokeBedrockConverseAPI(modelIdentifier string, requestBody BedrockRequest) (*BedrockResponse, error) {
	// Fetch configuration
	awsAccessKey := config.DefaultConfig().AwsAccessKey
	awsSecretKey := config.DefaultConfig().AwsSecretKey
	region, err := config.GetEnv("AWS_BEDROCK_REGION")
	if err != nil || region == "" {
		return nil, fmt.Errorf("AWS_BEDROCK_REGION is not set or invalid")
	}

	// Create AWS credentials
	creds := credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, "")

	// Initialize a session with the correct region
	_, err = session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Construct the endpoint URL
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/converse", region, modelIdentifier)
	// fmt.Println(endpoint)
	// Marshal the request body to JSON
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body to JSON: %v", err)
	}

	// log.Printf("Request Payload: %s", string(payload)) // Debugging

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign the request using SigV4 with the correct service name
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

	// Check for non-2xx status codes and log detailed error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Received non-2xx status code: %d\nResponse Body: %s", resp.StatusCode, string(responseBody))
		return nil, fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
	}

	// Unmarshal the successful response
	var bedrockResp BedrockResponse
	if err := json.Unmarshal(responseBody, &bedrockResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	return &bedrockResp, nil
}
