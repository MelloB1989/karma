package bedrock_runtime

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	// Debug initial request
	fmt.Println("Making request to:", endpoint)

	// Marshal the request body to JSON
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body to JSON: %v", err)
	}

	// Debug request payload
	fmt.Println("Request payload:", string(payload))

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
	fmt.Println("Sending request...")
	client := &http.Client{
		Timeout: time.Second * 60, // Increase timeout
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-2xx status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Verify content type is event-stream
	contentType := resp.Header.Get("Content-Type")
	fmt.Println("Content-Type:", contentType)

	if !strings.Contains(contentType, "application/vnd.amazon.eventstream") {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	// Read and process raw bytes
	fmt.Println("Reading response body...")

	// Create a tee reader to see the raw data
	// var rawBuffer bytes.Buffer
	// teeReader := io.TeeReader(resp.Body, &rawBuffer)

	// Create a buffer for binary data
	buffer := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if n == 0 {
			continue
		}

		// Process buffer data
		data := buffer[:n]
		var offset int = 0

		for offset < n {
			// Need at least 8 bytes for the prelude
			if offset+8 > n {
				break
			}

			// Read message length (first 4 bytes)
			messageLength := binary.BigEndian.Uint32(data[offset : offset+4])

			// Ensure we have the complete message
			if offset+int(messageLength) > n {
				break
			}

			// Read headers length (next 4 bytes)
			headersLength := binary.BigEndian.Uint32(data[offset+4 : offset+8])

			// Parse headers
			headerEnd := offset + 8 + int(headersLength)

			// Process headers
			// [Header parsing logic here]

			// Payload starts after headers
			payloadStart := headerEnd
			payloadEnd := offset + int(messageLength) - 4 // -4 for the trailing CRC

			payload := data[payloadStart:payloadEnd]

			// Process payload based on headers
			// From your debug output, you want to look for contentBlockDelta events
			// and extract the text from the delta.text field

			var eventData map[string]interface{}
			if err := json.Unmarshal(payload, &eventData); err == nil {
				if delta, ok := eventData["delta"].(map[string]interface{}); ok {
					if text, ok := delta["text"].(string); ok && textCallback != nil {
						if err := textCallback(text); err != nil {
							return err
						}
					}
				}
			}

			// Move to the next message
			offset += int(messageLength)
		}
	}

	return nil
}

// Function for normal stream processing with console output
func InvokeBedrockConverseStreamAPI(modelIdentifier string, requestBody BedrockRequest) error {
	// Text callback function for printing to console
	textCallback := func(text string) error {
		fmt.Print(text) // Removed "ghfgh" prefix that was in the original code
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

type EmbeddingResponse struct {
	Embedding []float32
}

func CreateEmbeddings(text string, modelID string) ([]float32, error) {
	awsAccessKey := config.DefaultConfig().AwsAccessKey
	awsSecretKey := config.DefaultConfig().AwsSecretKey
	region, err := config.GetEnv("AWS_BEDROCK_REGION")
	if err != nil || region == "" {
		return nil, fmt.Errorf("AWS_BEDROCK_REGION is not set or invalid")
	}
	creds := credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, "")
	_, err = session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", region, modelID)
	var payload []byte
	if strings.Contains(modelID, "titan-embed") {
		requestBody := struct {
			InputText string `json:"inputText"`
		}{
			InputText: text,
		}
		payload, err = json.Marshal(requestBody)
	} else if strings.Contains(modelID, "cohere") {
		requestBody := struct {
			Texts     []string `json:"texts"`
			InputType string   `json:"input_type"`
		}{
			Texts:     []string{text},
			InputType: "search_document",
		}
		payload, err = json.Marshal(requestBody)
	} else {
		return nil, fmt.Errorf("unsupported embedding model: %s", modelID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	signer := v4.NewSigner(creds)
	_, err = signer.Sign(req, bytes.NewReader(payload), "bedrock", region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %v", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Received non-2xx status code: %d\nResponse Body: %s", resp.StatusCode, string(responseBody))
		return nil, fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
	}
	var embeddings []float32
	if strings.Contains(modelID, "titan-embed") {
		var titanResp struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.Unmarshal(responseBody, &titanResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Titan response: %v", err)
		}
		embeddings = titanResp.Embedding
	} else if strings.Contains(modelID, "cohere") {
		var cohereResp struct {
			Embeddings [][]float32 `json:"embeddings"`
		}
		if err := json.Unmarshal(responseBody, &cohereResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Cohere response: %v", err)
		}
		if len(cohereResp.Embeddings) > 0 {
			embeddings = cohereResp.Embeddings[0]
		} else {
			return nil, fmt.Errorf("no embeddings returned from Cohere model")
		}
	}

	return embeddings, nil
}
