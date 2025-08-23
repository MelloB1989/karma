package tests

import (
	"fmt"
	"log"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/ai/parser"
	"github.com/MelloB1989/karma/models"
)

// Sample response structure
type ProductRecommendation struct {
	Name        string   `json:"name" description:"Product name"`
	Description string   `json:"description" description:"Brief product description"`
	Price       float64  `json:"price" description:"Product price in USD"`
	Rating      float64  `json:"rating" description:"User rating from 1.0 to 5.0"`
	Features    []string `json:"features" description:"Key product features"`
	Pros        []string `json:"pros" description:"Product advantages"`
	Cons        []string `json:"cons" description:"Product disadvantages"`
}

type GitLabIssues struct {
	Name        string   `json:"name" description:"Issue name"`
	Description string   `json:"description" description:"Issue description"`
	Labels      []string `json:"labels" description:"Issue labels"`
}

type GitLabIssuesResponse struct {
	Issues []GitLabIssues `json:"issues"`
}

func TestKarmaParser() {

	// Example with chat history
	chatExample()
	productsGeneration()
	issuesGeneration()
}

func issuesGeneration() {
	// Initialize the AI parser
	p := parser.NewParser(
		parser.WithModel((ai.Gemini20Flash)),
		parser.WithAIOptions(
			ai.WithTemperature(0.5),
			ai.WithSystemMessage("You are a helpful AI Project manager that specializes in issue generation."),
			ai.WithMaxTokens(500),
			ai.WithTopP(0.9),
			ai.WithResponseType("application/json"),
		),
		parser.WithMaxRetries(2),
	)

	// Define a prompt
	prompt := `
	Generate a list of issues for Gitlab based on the following issue list, refine and make them detailed:
	- Default country India ___HIGH
- Need help, onboarding screens (UI/UX) ___LOW
	`

	// Context information (optional)
	context := ""

	// Initialize the output structure
	var recommendation GitLabIssuesResponse

	// Parse the response
	timeTaken, tokens, err := p.Parse(prompt, context, &recommendation)
	if err != nil {
		log.Fatalf("Error parsing AI response: %v", err)
	}

	// Display the structured result
	fmt.Println("Generated Issues:")
	fmt.Println("Time taken: ", timeTaken)
	fmt.Println("Tokens: ", tokens)
	for _, issue := range recommendation.Issues {
		fmt.Printf("Name: %s\n", issue.Name)
		fmt.Printf("Description: %s\n", issue.Description)
		fmt.Printf("Labels: %v\n", issue.Labels)
		fmt.Println()
	}
}

func productsGeneration() {
	// Initialize the AI parser
	p := parser.NewParser(
		parser.WithModel((ai.Llama3_70B)),
		parser.WithAIOptions(
			ai.WithTemperature(0.1),
			ai.WithSystemMessage("You are a helpful assistant that specializes in product recommendations."),
			ai.WithMaxTokens(500),
			ai.WithTopP(0.9),
		),
		parser.WithMaxRetries(2),
	)

	// Define a prompt
	prompt := "Recommend a smartphone for a college student who needs good battery life and camera quality, budget around $500."

	// Context information (optional)
	context := "The user is looking for a mid-range smartphone with emphasis on battery life and camera quality."

	// Initialize the output structure
	var recommendation ProductRecommendation

	// Parse the response
	timeTaken, tokens, err := p.Parse(prompt, context, &recommendation)
	if err != nil {
		log.Fatalf("Error parsing AI response: %v", err)
	}

	// Display the structured result
	fmt.Println("Time taken: ", timeTaken)
	fmt.Println("Tokens: ", tokens)
	fmt.Printf("Recommended Product: %s\n", recommendation.Name)
	fmt.Printf("Description: %s\n", recommendation.Description)
	fmt.Printf("Price: $%.2f\n", recommendation.Price)
	fmt.Printf("Rating: %.1f/5.0\n", recommendation.Rating)

	fmt.Println("\nFeatures:")
	for _, feature := range recommendation.Features {
		fmt.Printf("- %s\n", feature)
	}

	fmt.Println("\nPros:")
	for _, pro := range recommendation.Pros {
		fmt.Printf("- %s\n", pro)
	}

	fmt.Println("\nCons:")
	for _, con := range recommendation.Cons {
		fmt.Printf("- %s\n", con)
	}
}

func chatExample() {
	fmt.Println("\n--- Chat Example ---")

	// Initialize the AI parser
	p := parser.NewParser()

	// Create a chat history
	chatHistory := []models.AIMessage{
		{
			Role:    models.System,
			Message: "You are a helpful assistant that specializes in weather forecasts.",
		},
		{
			Role:    models.User,
			Message: "What's the weather like in New York?",
		},
		{
			Role:    models.Assistant,
			Message: "I don't have access to real-time weather data. To provide an accurate forecast, I would need the current date and access to weather services.",
		},
		{
			Role:    models.User,
			Message: "Let's assume it's spring. Give me a typical forecast for New York in spring.",
		},
	}

	// Define the output structure
	type WeatherForecast struct {
		Location    string `json:"location" description:"City name"`
		Season      string `json:"season" description:"Current season"`
		Temperature struct {
			High float64 `json:"high" description:"High temperature in Celsius"`
			Low  float64 `json:"low" description:"Low temperature in Celsius"`
		} `json:"temperature" description:"Temperature range"`
		Conditions []string `json:"conditions" description:"Possible weather conditions"`
		Activities []string `json:"recommendedActivities" description:"Recommended activities for this weather"`
	}

	var forecast WeatherForecast

	// Parse the chat response
	err := p.ParseChatResponse(chatHistory, &forecast)
	if err != nil {
		log.Fatalf("Error parsing chat response: %v", err)
	}

	// Display the structured result
	fmt.Printf("Weather Forecast for %s in %s\n", forecast.Location, forecast.Season)
	fmt.Printf("Temperature: %.1f°C to %.1f°C\n", forecast.Temperature.Low, forecast.Temperature.High)

	fmt.Println("Possible Conditions:")
	for _, condition := range forecast.Conditions {
		fmt.Printf("- %s\n", condition)
	}

	fmt.Println("\nRecommended Activities:")
	for _, activity := range forecast.Activities {
		fmt.Printf("- %s\n", activity)
	}
}
