package examples

import (
	"fmt"
	"log"
	"time"

	"github.com/MelloB1989/karma/apis/claude"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
)

// Define input schemas for MCP tools
type CalculatorInput struct {
	Operation string  `json:"operation" jsonschema_description:"The arithmetic operation to perform (add, subtract, multiply, divide)"`
	X         float64 `json:"x" jsonschema_description:"First number"`
	Y         float64 `json:"y" jsonschema_description:"Second number"`
}

type AuthenticationInput struct {
	Email    string `json:"email" jsonschema_description:"User email address"`
	Password string `json:"password" jsonschema_description:"User password"`
}

type UserInfoInput struct {
	// No parameters needed for getting user info
}

func main() {
	// Create Claude client
	claudeClient := claude.NewClaudeClient(
		1024,                                 // maxTokens
		anthropic.ModelClaude3_5SonnetLatest, // model
		0.7,                                  // temperature
		0.9,                                  // topP
		40,                                   // topK
		"You are a helpful AI assistant with access to a calculator and user management system.", // system prompt
	)

	// Configure MCP server
	claudeClient.SetMCPServer("http://localhost:8080/mcp", "")

	// Add MCP tools
	err := claudeClient.AddMCPTool(
		"calculator",
		"Perform basic arithmetic operations (add, subtract, multiply, divide). Requires authentication.",
		"calculate", // MCP tool name
		CalculatorInput{},
	)
	if err != nil {
		log.Fatalf("Failed to add calculator tool: %v", err)
	}

	err = claudeClient.AddMCPTool(
		"authenticate",
		"Authenticate user and get access token for protected operations.",
		"authenticate", // MCP tool name
		AuthenticationInput{},
	)
	if err != nil {
		log.Fatalf("Failed to add authentication tool: %v", err)
	}

	err = claudeClient.AddMCPTool(
		"get_user_info",
		"Get information about the current authenticated user.",
		"get_user_info", // MCP tool name
		UserInfoInput{},
	)
	if err != nil {
		log.Fatalf("Failed to add user info tool: %v", err)
	}

	// Example 1: Simple conversation
	fmt.Println("=== Example 1: Simple Conversation ===")
	messages := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   "Hello! Can you help me with some calculations?",
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "example-1",
			},
		},
		ChatId:    "example-chat-1",
		CreatedAt: time.Now(),
		Title:     "Simple Conversation",
	}

	response, err := claudeClient.ClaudeChatCompletion(messages, false)
	if err != nil {
		log.Printf("Error in simple conversation: %v", err)
	} else {
		fmt.Printf("Claude: %s\n\n", response)
	}

	// Example 2: Using tools - Authentication and Calculator
	fmt.Println("=== Example 2: Using MCP Tools ===")
	messages = models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   "I need to authenticate first. Please use email 'admin@example.com' and password 'password123', then calculate 15 * 8.",
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "example-2",
			},
		},
		ChatId:    "example-chat-2",
		CreatedAt: time.Now(),
		Title:     "Using MCP Tools",
	}

	response, err = claudeClient.ClaudeChatCompletion(messages, true)
	if err != nil {
		log.Printf("Error using tools: %v", err)
	} else {
		fmt.Printf("Claude: %s\n\n", response)
	}

	// Example 3: Multi-turn conversation with tools
	fmt.Println("=== Example 3: Multi-turn Conversation ===")
	conversation := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   "First authenticate with admin@example.com and password123",
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "example-3-1",
			},
		},
		ChatId:    "example-chat-3",
		CreatedAt: time.Now(),
		Title:     "Multi-turn Conversation",
	}

	// First turn - authentication
	response, err = claudeClient.ClaudeChatCompletion(conversation, true)
	if err != nil {
		log.Printf("Error in multi-turn conversation: %v", err)
		return
	}

	conversation.Messages = append(conversation.Messages, models.AIMessage{
		Message:   response,
		Role:      models.Assistant,
		Timestamp: time.Now(),
		UniqueId:  "example-3-2",
	})

	fmt.Printf("Claude: %s\n", response)

	// Second turn - calculator
	conversation.Messages = append(conversation.Messages, models.AIMessage{
		Message:   "Now calculate 123 + 456",
		Role:      models.User,
		Timestamp: time.Now(),
		UniqueId:  "example-3-3",
	})

	response, err = claudeClient.ClaudeChatCompletion(conversation, true)
	if err != nil {
		log.Printf("Error in calculator request: %v", err)
	} else {
		fmt.Printf("Claude: %s\n\n", response)
	}

	// Example 4: Streaming with tools
	fmt.Println("=== Example 4: Streaming with Tools ===")
	streamMessages := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   "Please authenticate with admin@example.com and password123, then get my user info, and finally calculate what 25% of 200 is.",
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "example-4",
			},
		},
		ChatId:    "example-chat-4",
		CreatedAt: time.Now(),
		Title:     "Streaming with Tools",
	}

	streamResponse := ""
	callback := func(chunk models.StreamedResponse) error {
		fmt.Print(chunk.AIResponse)
		streamResponse += chunk.AIResponse
		return nil
	}

	finalResponse, err := claudeClient.ClaudeStreamCompletionWithTools(streamMessages, callback, true)
	if err != nil {
		log.Printf("Error in streaming: %v", err)
	} else {
		fmt.Printf("\n\nFinal response: %s\n\n", finalResponse)
	}

	// Example 5: Error handling - trying to use calculator without auth
	fmt.Println("=== Example 5: Error Handling ===")
	errorMessages := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   "Calculate 10 + 5 without authenticating first.",
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "example-5",
			},
		},
		ChatId:    "example-chat-5",
		CreatedAt: time.Now(),
		Title:     "Error Handling",
	}

	response, err = claudeClient.ClaudeChatCompletion(errorMessages, true)
	if err != nil {
		log.Printf("Error (expected): %v", err)
	} else {
		fmt.Printf("Claude: %s\n\n", response)
	}

	// Example 6: Complex calculation workflow
	fmt.Println("=== Example 6: Complex Workflow ===")
	workflowMessages := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   "I need to do some financial calculations. First authenticate as admin@example.com with password123, then help me calculate the total cost: I'm buying 3 items at $12.50 each, plus 8.5% tax. Show me step by step.",
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "example-6",
			},
		},
		ChatId:    "example-chat-6",
		CreatedAt: time.Now(),
		Title:     "Complex Workflow",
	}

	response, err = claudeClient.ClaudeChatCompletion(workflowMessages, true)
	if err != nil {
		log.Printf("Error in workflow: %v", err)
	} else {
		fmt.Printf("Claude: %s\n", response)
	}
}
