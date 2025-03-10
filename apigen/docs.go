package apigen

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// APIDefinition represents a complete API with base URL and endpoints
type APIDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	BaseURL     string     `json:"baseUrl"`
	Endpoints   []Endpoint `json:"endpoints"`
}

// Endpoint represents a single API endpoint
type Endpoint struct {
	Path           string            `json:"path"`
	Method         string            `json:"method"`
	Summary        string            `json:"summary"`
	Description    string            `json:"description"`
	Headers        map[string]string `json:"headers,omitempty"`
	QueryParams    []Parameter       `json:"queryParams,omitempty"`
	RequestBody    *RequestBody      `json:"requestBody,omitempty"`
	Responses      []Response        `json:"responses"`
	Authentication *Auth             `json:"authentication,omitempty"`
}

// Parameter defines a request parameter (query or path param)
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

// RequestBody defines the structure of a request body
type RequestBody struct {
	ContentType string          `json:"contentType"`
	Required    bool            `json:"required"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Example     json.RawMessage `json:"example,omitempty"`
}

// Response defines a possible API response
type Response struct {
	StatusCode  int               `json:"statusCode"`
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Schema      json.RawMessage   `json:"schema,omitempty"`
	Example     json.RawMessage   `json:"example,omitempty"`
}

// Auth defines authentication details
type Auth struct {
	Type        string            `json:"type"` // bearer, basic, apiKey, oauth2
	Description string            `json:"description,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// NewAPIDefinition creates a new API definition with the given name and base URL
func NewAPIDefinition(name, description, baseURL string) *APIDefinition {
	return &APIDefinition{
		Name:        name,
		Description: description,
		BaseURL:     baseURL,
		Endpoints:   []Endpoint{},
	}
}

// AddEndpoint adds a new endpoint to the API definition
func (api *APIDefinition) AddEndpoint(endpoint Endpoint) *APIDefinition {
	api.Endpoints = append(api.Endpoints, endpoint)
	return api
}

// SaveToFile saves the API definition to a JSON file
func (api *APIDefinition) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(api, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling API definition: %w", err)
	}

	dir := filepath.Dir(filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// LoadFromFile loads an API definition from a JSON file
func LoadFromFile(filename string) (*APIDefinition, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var api APIDefinition
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("error unmarshaling API definition: %w", err)
	}

	return &api, nil
}

// ExportToPostman exports the API definition to a Postman collection
func (api *APIDefinition) ExportToPostman(filename string) error {
	collection := generatePostmanCollection(api)

	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling Postman collection: %w", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// ExportToOpenAPI exports the API definition to an OpenAPI JSON file
func (api *APIDefinition) ExportToOpenAPI(filename string) error {
	openapi := generateOpenAPI(api)

	data, err := json.MarshalIndent(openapi, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling OpenAPI specification: %w", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// ExportToMarkdown exports the API definition to a Markdown file
// This format is particularly LLM-friendly
func (api *APIDefinition) ExportToMarkdown(filename string) error {
	markdown := generateMarkdown(api)
	return ioutil.WriteFile(filename, []byte(markdown), 0644)
}

// Helper functions to generate different export formats

func generatePostmanCollection(api *APIDefinition) map[string]any {
	collection := map[string]any{
		"info": map[string]any{
			"name":        api.Name,
			"description": api.Description,
			"schema":      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"item": []any{},
	}

	items := []any{}
	for _, endpoint := range api.Endpoints {
		item := map[string]any{
			"name": endpoint.Summary,
			"request": map[string]any{
				"method": endpoint.Method,
				"url": map[string]any{
					"raw":  api.BaseURL + endpoint.Path,
					"host": strings.Split(strings.TrimPrefix(strings.TrimPrefix(api.BaseURL, "http://"), "https://"), "/")[0],
					"path": strings.Split(strings.Trim(endpoint.Path, "/"), "/"),
				},
				"description": endpoint.Description,
			},
			"response": []any{},
		}

		// Add headers
		if len(endpoint.Headers) > 0 {
			headers := []any{}
			for key, value := range endpoint.Headers {
				headers = append(headers, map[string]string{
					"key":   key,
					"value": value,
				})
			}
			item["request"].(map[string]any)["header"] = headers
		}

		// Add request body if present
		if endpoint.RequestBody != nil {
			item["request"].(map[string]any)["body"] = map[string]any{
				"mode": "raw",
				"raw":  string(endpoint.RequestBody.Example),
				"options": map[string]any{
					"raw": map[string]any{
						"language": "json",
					},
				},
			}
		}

		// Add responses
		responses := []any{}
		for _, resp := range endpoint.Responses {
			response := map[string]any{
				"name":        fmt.Sprintf("%d %s", resp.StatusCode, resp.Description),
				"code":        resp.StatusCode,
				"description": resp.Description,
			}

			if resp.Example != nil {
				response["body"] = string(resp.Example)
			}

			responses = append(responses, response)
		}
		item["response"] = responses

		items = append(items, item)
	}

	collection["item"] = items
	return collection
}

func generateOpenAPI(api *APIDefinition) map[string]any {
	openapi := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]string{
			"title":       api.Name,
			"description": api.Description,
			"version":     "1.0.0",
		},
		"servers": []map[string]string{
			{
				"url": api.BaseURL,
			},
		},
		"paths": map[string]any{},
	}

	paths := map[string]any{}
	for _, endpoint := range api.Endpoints {
		method := strings.ToLower(endpoint.Method)

		pathData := map[string]any{
			"summary":     endpoint.Summary,
			"description": endpoint.Description,
			"responses":   map[string]any{},
		}

		// Add parameters
		parameters := []any{}
		for _, param := range endpoint.QueryParams {
			parameters = append(parameters, map[string]any{
				"name":        param.Name,
				"in":          "query",
				"description": param.Description,
				"required":    param.Required,
				"schema": map[string]string{
					"type": param.Type,
				},
				"example": param.Example,
			})
		}

		for key := range endpoint.Headers {
			parameters = append(parameters, map[string]any{
				"name":     key,
				"in":       "header",
				"required": true,
				"schema": map[string]string{
					"type": "string",
				},
			})
		}

		if len(parameters) > 0 {
			pathData["parameters"] = parameters
		}

		// Add request body
		if endpoint.RequestBody != nil {
			pathData["requestBody"] = map[string]any{
				"required": endpoint.RequestBody.Required,
				"content": map[string]any{
					endpoint.RequestBody.ContentType: map[string]any{
						"schema":  endpoint.RequestBody.Schema,
						"example": endpoint.RequestBody.Example,
					},
				},
			}
		}

		// Add responses
		responses := map[string]any{}
		for _, resp := range endpoint.Responses {
			respData := map[string]any{
				"description": resp.Description,
			}

			if resp.Schema != nil {
				respData["content"] = map[string]any{
					resp.ContentType: map[string]any{
						"schema":  resp.Schema,
						"example": resp.Example,
					},
				}
			}

			responses[fmt.Sprintf("%d", resp.StatusCode)] = respData
		}
		pathData["responses"] = responses

		// Add path to paths object
		if _, ok := paths[endpoint.Path]; !ok {
			paths[endpoint.Path] = map[string]any{}
		}
		paths[endpoint.Path].(map[string]any)[method] = pathData
	}

	openapi["paths"] = paths
	return openapi
}

func generateMarkdown(api *APIDefinition) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", api.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", api.Description))
	sb.WriteString(fmt.Sprintf("Base URL: `%s`\n\n", api.BaseURL))

	sb.WriteString("## Endpoints\n\n")
	for _, endpoint := range api.Endpoints {
		sb.WriteString(fmt.Sprintf("### %s\n\n", endpoint.Summary))
		sb.WriteString(fmt.Sprintf("**Path:** `%s`\n\n", endpoint.Path))
		sb.WriteString(fmt.Sprintf("**Method:** `%s`\n\n", endpoint.Method))

		if endpoint.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", endpoint.Description))
		}

		if len(endpoint.Headers) > 0 {
			sb.WriteString("#### Headers\n\n")
			for key, value := range endpoint.Headers {
				sb.WriteString(fmt.Sprintf("- `%s`: %s\n", key, value))
			}
			sb.WriteString("\n")
		}

		if len(endpoint.QueryParams) > 0 {
			sb.WriteString("#### Query Parameters\n\n")
			for _, param := range endpoint.QueryParams {
				required := ""
				if param.Required {
					required = " (required)"
				}
				sb.WriteString(fmt.Sprintf("- `%s`: %s%s", param.Name, param.Description, required))
				if param.Example != "" {
					sb.WriteString(fmt.Sprintf(" (Example: `%s`)", param.Example))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		if endpoint.RequestBody != nil {
			sb.WriteString("#### Request Body\n\n")
			sb.WriteString(fmt.Sprintf("Content-Type: `%s`\n\n", endpoint.RequestBody.ContentType))

			if endpoint.RequestBody.Example != nil {
				sb.WriteString("Example:\n\n```json\n")
				sb.WriteString(string(endpoint.RequestBody.Example))
				sb.WriteString("\n```\n\n")
			}
		}

		sb.WriteString("#### Responses\n\n")
		for _, resp := range endpoint.Responses {
			sb.WriteString(fmt.Sprintf("**%d**: %s\n\n", resp.StatusCode, resp.Description))

			if resp.Example != nil {
				sb.WriteString("Example:\n\n```json\n")
				sb.WriteString(string(resp.Example))
				sb.WriteString("\n```\n\n")
			}
		}
	}

	return sb.String()
}
