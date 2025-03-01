package tests

import (
	"fmt"
	"log"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/ai/parser/codeparser"
)

func TestAICodeParser() {
	// Example project directory
	projectDir := "/Users/mellob/Developer/code/test"

	// Initialize the code parser with Go language
	context, err := codeparser.BuildProjectContext(projectDir, codeparser.Go)
	if err != nil {
		log.Fatalf("Error building project context: %v", err)
	}

	fmt.Println(context.FileTree)

	parser := codeparser.NewCodeParser(
		codeparser.WithModel(ai.Claude3_7Sonnet20250219V1),
		codeparser.WithAIOptions(
			ai.WithTemperature(0.2),
			ai.WithSystemMessage("You are a Go programming expert. Always follow Go best practices and idioms."),
			ai.WithMaxTokens(2000),
			ai.WithTopP(0.9),
		),
		codeparser.WithMaxRetries(2),
		codeparser.WithLanguage(codeparser.Go),
		codeparser.WithProjectContext(context),
	)

	// Load all Go files into memory
	if err := parser.LoadAllFiles(); err != nil {
		log.Fatalf("Error loading files: %v", err)
	}

	// Example 1: Add a new feature
	fmt.Println("\n--- Bug fixes ---")
	prompt := "Fix all bugs in middleware.go"

	changes, err := parser.ParseCodeChanges(prompt)
	if err != nil {
		log.Fatalf("Error parsing code changes: %v", err)
	}

	// Display the proposed changes
	fmt.Printf("AI proposed %d code changes for the authentication feature:\n", len(changes.Changes))
	for i, change := range changes.Changes {
		fmt.Printf("%d. %s %s: %s\n", i+1, change.Operation, change.Path, change.Description)
	}

	// Apply the changes
	if err := parser.ApplyChanges(changes); err != nil {
		log.Fatal(err)
	}
}
