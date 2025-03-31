package bedrock_runtime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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

// Generic base function that handles common logic for stream processing
func processBedrockStream(
	modelIdentifier string,
	requestBody BedrockRequest,
	textCallback func(string) error,
	metadataCallback func(map[string]interface{}) error,
) error {
	// Fetch configuration
	awsAccessKey := config.DefaultConfig().AwsAccessKey
	awsSecretKey := config.DefaultConfig().AwsSecretKey
	region, err := config.GetEnv("AWS_BEDROCK_REGION")
	if err != nil || region == "" {
		return fmt.Errorf("AWS_BEDROCK_REGION is not set or invalid")
	}

	// Create AWS credentials
	creds := credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, "")

	// Initialize a session with the correct region
	_, err = session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Construct the endpoint URL for streaming
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/converse-stream", region, modelIdentifier)

	// Marshal the request body to JSON
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body to JSON: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign the request using SigV4 with the correct service name
	signer := v4.NewSigner(creds)
	_, err = signer.Sign(req, bytes.NewReader(payload), "bedrock", region, time.Now())
	if err != nil {
		return fmt.Errorf("failed to sign request: %v", err)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-2xx status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Verify content type is event-stream
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/vnd.amazon.eventstream") {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	// Process the event stream response
	reader := bufio.NewReader(resp.Body)

	for {
		// Read the headers
		headers, err := readEventStreamHeaders(reader)
		if err != nil {
			if err == io.EOF {
				break // End of stream
			}
			return fmt.Errorf("error reading event stream headers: %v", err)
		}

		// Read the event payload
		contentLength, ok := headers["content-length"]
		if !ok {
			return fmt.Errorf("missing content-length header")
		}

		length, err := strconv.Atoi(contentLength)
		if err != nil {
			return fmt.Errorf("invalid content-length: %v", err)
		}

		payload := make([]byte, length)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			return fmt.Errorf("error reading event payload: %v", err)
		}

		// Process the event based on its type
		eventType, ok := headers["event-type"]
		if !ok {
			return fmt.Errorf("missing event-type header")
		}

		messageType, _ := headers["message-type"]

		if messageType == "event" {
			switch eventType {
			case "contentBlockDelta":
				// Parse the delta content
				var delta struct {
					ContentBlockIndex int `json:"contentBlockIndex"`
					Delta             struct {
						Text string `json:"text"`
					} `json:"delta"`
					P string `json:"p"`
				}

				if err := json.Unmarshal(payload, &delta); err != nil {
					log.Printf("Error parsing delta: %v", err)
					continue
				}

				// Call the text callback if provided and text is not empty
				if textCallback != nil && delta.Delta.Text != "" {
					if err := textCallback(delta.Delta.Text); err != nil {
						return fmt.Errorf("text callback error: %v", err)
					}
				}

			case "metadata":
				// Parse metadata for token usage, etc.
				var metadata map[string]interface{}
				if err := json.Unmarshal(payload, &metadata); err != nil {
					log.Printf("Error parsing metadata: %v", err)
					continue
				}

				// Call the metadata callback if provided
				if metadataCallback != nil {
					if err := metadataCallback(metadata); err != nil {
						return fmt.Errorf("metadata callback error: %v", err)
					}
				}
			}
		}

		// Read the trailing newline after each event
		_, err = reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading trailing newline: %v", err)
		}
	}

	return nil
}

// Function for normal stream processing with console output
func InvokeBedrockConverseStreamAPI(modelIdentifier string, requestBody BedrockRequest) error {
	// Text callback function for printing to console
	textCallback := func(text string) error {
		fmt.Print("ghfgh", text)
		return nil
	}

	// Metadata callback function for printing to console
	metadataCallback := func(metadata map[string]interface{}) error {
		// Optionally print metadata information
		if usage, ok := metadata["usage"].(map[string]interface{}); ok {
			log.Printf("Usage: Input: %v, Output: %v, Total: %v tokens",
				usage["inputTokens"],
				usage["outputTokens"],
				usage["totalTokens"])
		}
		return nil
	}

	// Use the generic function with console output callbacks
	return processBedrockStream(modelIdentifier, requestBody, textCallback, metadataCallback)
}

// Function for stream processing with custom callback
func InvokeBedrockConverseStreamAPIWithCallback(
	modelIdentifier string,
	requestBody BedrockRequest,
	callback func(string),
) error {
	// Text callback function that calls the user-provided callback
	textCallback := func(text string) error {
		if callback != nil {
			callback(text)
		}
		return nil
	}

	// Use the generic function with the custom text callback and no metadata callback
	return processBedrockStream(modelIdentifier, requestBody, textCallback, nil)
}

// Helper function to read event stream headers (unchanged)
func readEventStreamHeaders(reader *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		// Trim the newline
		line = strings.TrimSuffix(line, "\n")

		// Empty line marks the end of headers
		if line == "" {
			break
		}

		// Parse the header
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header format: %s", line)
		}

		headers[parts[0]] = parts[1]
	}

	return headers, nil
}
