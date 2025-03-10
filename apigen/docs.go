package apigen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// NewAPIDefinition creates a new API definition with the given name and base URLs
func NewAPIDefinition(name, description string, baseURLs []string, outputFolder, outputFileName string) *APIDefinition {
	return &APIDefinition{
		Name:               name,
		Description:        description,
		BaseURLs:           baseURLs,
		GlobalVariables:    make(map[string]string),
		Endpoints:          []Endpoint{},
		OutputFileBaseName: outputFileName,
		OutputFolder:       outputFolder,
	}
}

// AddGlobalVariable adds a global variable to the API definition
func (api *APIDefinition) AddGlobalVariable(name, value string) *APIDefinition {
	api.GlobalVariables[name] = value
	return api
}

// AddEndpoint adds a new endpoint to the API definition
func (api *APIDefinition) AddEndpoint(endpoint Endpoint) *APIDefinition {
	// Extract path parameters from the path
	pathParams := extractPathParams(endpoint.Path)

	// If path parameters are found but not defined in the endpoint, create them
	if len(pathParams) > 0 {
		paramMap := make(map[string]bool)
		for _, param := range endpoint.PathParams {
			paramMap[param.Name] = true
		}

		for _, paramName := range pathParams {
			if !paramMap[paramName] {
				endpoint.PathParams = append(endpoint.PathParams, Parameter{
					Name:        paramName,
					Type:        "string",
					Required:    true,
					Description: fmt.Sprintf("Path parameter: %s", paramName),
				})
			}
		}
	}

	api.Endpoints = append(api.Endpoints, endpoint)
	return api
}

// RequestBodyFromStruct creates a RequestBody from a struct type
func RequestBodyFromStruct(structPtr any, contentType ContentType, required bool, overrides []FieldOverride) (*RequestBody, error) {
	t := reflect.TypeOf(structPtr)

	// Ensure we're dealing with a struct
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or pointer to struct, got %v", t.Kind())
	}

	// Create overrides map for quick lookup
	overrideMap := make(map[string]FieldOverride)
	for _, override := range overrides {
		overrideMap[override.Name] = override
	}

	fields := extractStructFields(t, overrideMap, "")

	// Create example JSON
	exampleValue := reflect.New(t).Interface()
	exampleJSON, err := json.MarshalIndent(exampleValue, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error creating example JSON: %w", err)
	}

	// // Create JSON schema (simplified)
	// schemaMap := map[string]any{
	// 	"type":       "object",
	// 	"properties": map[string]any{},
	// 	"required":   []string{},
	// }

	return &RequestBody{
		ContentType: contentType,
		Required:    required,
		Fields:      fields,
		Example:     exampleJSON,
		Schema:      nil, // Advanced schema generation would go here
	}, nil
}

// ResponseFromStruct creates a Response from a struct type
func ResponseFromStruct(statusCode int, description string, structPtr any, contentType ContentType, overrides []FieldOverride) (*Response, error) {
	t := reflect.TypeOf(structPtr)

	// Ensure we're dealing with a struct
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or pointer to struct, got %v", t.Kind())
	}

	// Create overrides map for quick lookup
	overrideMap := make(map[string]FieldOverride)
	for _, override := range overrides {
		overrideMap[override.Name] = override
	}

	fields := extractStructFields(t, overrideMap, "")

	// Create example JSON
	exampleValue := reflect.New(t).Interface()
	exampleJSON, err := json.MarshalIndent(exampleValue, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error creating example JSON: %w", err)
	}

	return &Response{
		StatusCode:  statusCode,
		Description: description,
		ContentType: contentType,
		Fields:      fields,
		Example:     exampleJSON,
	}, nil
}

// SaveToFile saves the API definition to a JSON file
func (api *APIDefinition) SaveToFile() error {
	if api.OutputFolder == "" {
		api.OutputFolder = "."
	}

	if api.OutputFileBaseName == "" {
		api.OutputFileBaseName = strings.ToLower(strings.ReplaceAll(api.Name, " ", "_"))
	}

	jsonFilename := filepath.Join(api.OutputFolder, api.OutputFileBaseName+".json")

	data, err := json.MarshalIndent(api, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling API definition: %w", err)
	}

	if err := os.MkdirAll(api.OutputFolder, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	return os.WriteFile(jsonFilename, data, 0644)
}

// LoadFromFile loads an API definition from a JSON file
func LoadFromFile(filename string) (*APIDefinition, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var api APIDefinition
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("error unmarshaling API definition: %w", err)
	}

	// Set output file information based on input file
	api.OutputFolder = filepath.Dir(filename)
	api.OutputFileBaseName = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	return &api, nil
}

// ExportToPostman exports the API definition to a Postman collection
func (api *APIDefinition) ExportToPostman() error {
	collection := generatePostmanCollection(api)

	filename := filepath.Join(api.OutputFolder, api.OutputFileBaseName+"_postman.json")
	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling Postman collection: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// ExportToOpenAPI exports the API definition to an OpenAPI JSON file
func (api *APIDefinition) ExportToOpenAPI() error {
	openapi := generateOpenAPI(api)

	filename := filepath.Join(api.OutputFolder, api.OutputFileBaseName+"_openapi.json")
	data, err := json.MarshalIndent(openapi, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling OpenAPI specification: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// ExportToMarkdown exports the API definition to a Markdown file
func (api *APIDefinition) ExportToMarkdown() error {
	markdown := generateMarkdown(api)

	filename := filepath.Join(api.OutputFolder, api.OutputFileBaseName+"_docs.md")
	return os.WriteFile(filename, []byte(markdown), 0644)
}

// ExportAll exports to all supported formats
func (api *APIDefinition) ExportAll() error {
	if err := api.SaveToFile(); err != nil {
		return err
	}

	if err := api.ExportToPostman(); err != nil {
		return err
	}

	if err := api.ExportToOpenAPI(); err != nil {
		return err
	}

	if err := api.ExportToMarkdown(); err != nil {
		return err
	}

	return nil
}
