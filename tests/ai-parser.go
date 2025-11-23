package tests

import (
	"fmt"
	"log"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/ai/parser"
	"github.com/MelloB1989/karma/models"
)

// Example 1: Simple structured data extraction
type Product struct {
	Name     string   `json:"name" description:"Product name"`
	Price    float64  `json:"price" description:"Price in USD"`
	Category string   `json:"category" description:"Product category"`
	Features []string `json:"features" description:"List of key features"`
	InStock  bool     `json:"in_stock" description:"Availability status"`
	Rating   float64  `json:"rating,omitempty" description:"User rating out of 5"`
}

// Example 2: Complex nested structure
type Article struct {
	Title    string    `json:"title" description:"Article title"`
	Summary  string    `json:"summary" description:"Brief summary"`
	Author   Author    `json:"author" description:"Author information"`
	Tags     []string  `json:"tags" description:"Article tags"`
	Sections []Section `json:"sections" description:"Article sections"`
}

type Author struct {
	Name  string `json:"name" description:"Author name"`
	Email string `json:"email,omitempty" description:"Author email"`
}

type Section struct {
	Heading string `json:"heading" description:"Section heading"`
	Content string `json:"content" description:"Section content"`
}

// Example 3: Data analysis result
type SentimentAnalysis struct {
	OverallSentiment string             `json:"overall_sentiment" description:"positive, negative, or neutral"`
	Score            float64            `json:"score" description:"Sentiment score from -1 to 1"`
	KeyPhrases       []string           `json:"key_phrases" description:"Important phrases"`
	Emotions         map[string]float64 `json:"emotions" description:"Emotion scores"`
}

func TestParser() {
	// Initialize parser with options
	p := parser.NewParser(
		parser.WithMaxRetries(3),
		parser.WithDebug(true),
	)

	// Example 1: Extract product information
	fmt.Println("=== Example 1: Product Extraction ===")
	productExample()

	// Example 2: Generate structured article
	fmt.Println("\n=== Example 2: Article Generation ===")
	articleExample(p)

	// Example 3: Sentiment analysis
	fmt.Println("\n=== Example 3: Sentiment Analysis ===")
	sentimentExample(p)

	// Example 4: Chat-based parsing
	fmt.Println("\n=== Example 4: Chat Completion ===")
	chatExample(p)
}

func productExample() {
	// Custom AI client setup
	client := ai.NewKarmaAI(
		ai.GPT4oMini,
		ai.OpenAI,
	)

	p := parser.NewParser(
		parser.WithAIClient(client),
		parser.WithMaxRetries(2),
	)

	var product Product
	prompt := "Extract information about the iPhone 15 Pro"
	context := "Latest Apple flagship smartphone released in 2023"

	duration, tokens, err := p.Parse(prompt, context, &product)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Duration: %v, Tokens: %d\n", duration, tokens)
	fmt.Printf("Product: %+v\n", product)
}

func articleExample(p *parser.Parser) {
	var article Article
	prompt := "Write an article about the benefits of meditation"
	context := "Target audience: beginners interested in wellness"

	duration, tokens, err := p.Parse(prompt, context, &article)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Duration: %v, Tokens: %d\n", duration, tokens)
	fmt.Printf("Title: %s\n", article.Title)
	fmt.Printf("Author: %s\n", article.Author.Name)
	fmt.Printf("Sections: %d\n", len(article.Sections))
}

func sentimentExample(p *parser.Parser) {
	var sentiment SentimentAnalysis
	prompt := "Analyze the sentiment of this review"
	context := `"This product exceeded my expectations! The quality is outstanding,
	and customer service was incredibly helpful. However, shipping took longer than expected."`

	duration, tokens, err := p.Parse(prompt, context, &sentiment)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Duration: %v, Tokens: %d\n", duration, tokens)
	fmt.Printf("Sentiment: %s (%.2f)\n", sentiment.OverallSentiment, sentiment.Score)
	fmt.Printf("Key Phrases: %v\n", sentiment.KeyPhrases)
	fmt.Printf("Emotions: %v\n", sentiment.Emotions)
}

func chatExample(p *parser.Parser) {
	var product Product

	messages := []models.AIMessage{
		{
			Role:    models.User,
			Message: "I need information about wireless headphones",
		},
		{
			Role:    models.Assistant,
			Message: "I can help with that. Which brand are you interested in?",
		},
		{
			Role:    models.User,
			Message: "Sony WH-1000XM5",
		},
	}

	err := p.ParseChat(messages, &product)
	if err != nil {
		log.Fatalf("Chat parse error: %v", err)
	}

	fmt.Printf("Product: %+v\n", product)
}

// Advanced Example: Batch processing
func batchProcessingExample() {
	p := parser.NewParser(parser.WithMaxRetries(2))

	reviews := []string{
		"Great product, highly recommend!",
		"Terrible experience, would not buy again.",
		"It's okay, nothing special.",
	}

	type BatchResult struct {
		Reviews []SentimentAnalysis `json:"reviews"`
	}

	var result BatchResult
	prompt := "Analyze sentiment for each review"
	context := fmt.Sprintf("Reviews: %v", reviews)

	_, _, err := p.Parse(prompt, context, &result)
	if err != nil {
		log.Fatalf("Batch parse error: %v", err)
	}

	for i, analysis := range result.Reviews {
		fmt.Printf("Review %d: %s (%.2f)\n", i+1, analysis.OverallSentiment, analysis.Score)
	}
}

// Error handling example
func errorHandlingExample() {
	p := parser.NewParser(parser.WithMaxRetries(1))

	type InvalidStruct struct {
		unexported string // Won't work - unexported field
	}

	var invalid InvalidStruct
	_, _, err := p.Parse("test", "", &invalid)
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	// Correct usage
	type ValidStruct struct {
		Field string `json:"field"`
	}

	var valid ValidStruct
	duration, tokens, err := p.Parse("Generate a test value", "", &valid)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Success! Duration: %v, Tokens: %d, Value: %+v\n", duration, tokens, valid)
	}
}
