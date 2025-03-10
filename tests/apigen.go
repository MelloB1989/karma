package tests

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/MelloB1989/karma/apigen"
)

func TestAPIGen() {
	// Create a new API definition
	api := apigen.NewAPIDefinition(
		"User Management API",
		"API for managing users in the system",
		"https://api.example.com/v1",
	)

	// Define the Get Users endpoint
	getUsersEndpoint := apigen.Endpoint{
		Path:        "/users",
		Method:      "GET",
		Summary:     "Get all users",
		Description: "Retrieves a list of all users in the system with pagination",
		Headers: map[string]string{
			"Authorization": "Bearer {token}",
		},
		QueryParams: []apigen.Parameter{
			{
				Name:        "page",
				Type:        "integer",
				Required:    false,
				Description: "Page number for pagination",
				Example:     "1",
			},
			{
				Name:        "limit",
				Type:        "integer",
				Required:    false,
				Description: "Number of items per page",
				Example:     "10",
			},
		},
		Authentication: &apigen.Auth{
			Type:        "bearer",
			Description: "JWT token for authentication",
		},
		Responses: []apigen.Response{
			{
				StatusCode:  200,
				Description: "Successful response",
				ContentType: "application/json",
				Schema:      json.RawMessage(`{"type":"object","properties":{"users":{"type":"array","items":{"$ref":"#/components/schemas/User"}},"total":{"type":"integer"}}}`),
				Example:     json.RawMessage(`{"users":[{"id":1,"name":"John Doe","email":"john@example.com"},{"id":2,"name":"Jane Smith","email":"jane@example.com"}],"total":2}`),
			},
			{
				StatusCode:  401,
				Description: "Unauthorized",
				Example:     json.RawMessage(`{"error":"Invalid or missing authentication token"}`),
			},
		},
	}

	// Define the Create User endpoint
	createUserEndpoint := apigen.Endpoint{
		Path:        "/users",
		Method:      "POST",
		Summary:     "Create a new user",
		Description: "Creates a new user in the system",
		Headers: map[string]string{
			"Authorization": "Bearer {token}",
			"Content-Type":  "application/json",
		},
		RequestBody: &apigen.RequestBody{
			ContentType: "application/json",
			Required:    true,
			Schema:      json.RawMessage(`{"type":"object","required":["name","email"],"properties":{"name":{"type":"string"},"email":{"type":"string","format":"email"},"role":{"type":"string","enum":["admin","user"]}}}`),
			Example:     json.RawMessage(`{"name":"John Doe","email":"john@example.com","role":"user"}`),
		},
		Responses: []apigen.Response{
			{
				StatusCode:  201,
				Description: "User created successfully",
				ContentType: "application/json",
				Example:     json.RawMessage(`{"id":3,"name":"John Doe","email":"john@example.com","role":"user"}`),
			},
			{
				StatusCode:  400,
				Description: "Invalid request",
				ContentType: "application/json",
				Example:     json.RawMessage(`{"error":"Invalid email format"}`),
			},
			{
				StatusCode:  401,
				Description: "Unauthorized",
				ContentType: "application/json",
				Example:     json.RawMessage(`{"error":"Invalid or missing authentication token"}`),
			},
		},
	}

	// Add endpoints to the API definition
	api.AddEndpoint(getUsersEndpoint)
	api.AddEndpoint(createUserEndpoint)

	// Save API definition to a file
	if err := api.SaveToFile("./docstest/api-definition.json"); err != nil {
		log.Fatalf("Error saving API definition: %v", err)
	}
	fmt.Println("API definition saved to api-definition.json")

	// Export to Postman collection
	if err := api.ExportToPostman("./docstest/postman-collection.json"); err != nil {
		log.Fatalf("Error exporting to Postman: %v", err)
	}
	fmt.Println("Postman collection exported to postman-collection.json")

	// Export to OpenAPI JSON
	if err := api.ExportToOpenAPI("./docstest/openapi.json"); err != nil {
		log.Fatalf("Error exporting to OpenAPI: %v", err)
	}
	fmt.Println("OpenAPI specification exported to openapi.json")

	// Export to Markdown (LLM-friendly format)
	if err := api.ExportToMarkdown("./docstest/api-docs.md"); err != nil {
		log.Fatalf("Error exporting to Markdown: %v", err)
	}
	fmt.Println("Markdown documentation exported to api-docs.md")

	// Load API definition from file
	loadedAPI, err := apigen.LoadFromFile("./docstest/api-definition.json")
	if err != nil {
		log.Fatalf("Error loading API definition: %v", err)
	}
	fmt.Printf("Loaded API: %s with %d endpoints\n", loadedAPI.Name, len(loadedAPI.Endpoints))
}
