package apigen

import (
	"encoding/json"
	"fmt"
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

	// Create a map for our example instead of using reflection directly
	exampleMap := make(map[string]interface{})

	// Add any completely new fields specified in overrides
	for _, override := range overrides {
		if override.IsNewField {
			newField := RequestBodyField{
				Name:        override.Name,
				JsonName:    override.JsonName,
				Type:        override.Type,
				Description: override.Description,
				Example:     override.Example,
			}

			if override.JsonName == "" {
				newField.JsonName = override.Name
			}

			if override.Required != nil {
				newField.Required = *override.Required
			} else {
				newField.Required = true // Default to required
			}

			fields = append(fields, newField)

			// Also add to example map
			exampleMap[newField.JsonName] = override.Example
		}
	}

	// Populate the example map with fields that aren't excluded
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		jsonName := fieldName

		// Parse the json tag to get the field name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		// Skip excluded fields
		if override, exists := overrideMap[fieldName]; exists && override.Exclude {
			continue
		}

		// Add field to example with its zero value or override example
		if override, exists := overrideMap[fieldName]; exists && override.Example != nil {
			exampleMap[jsonName] = override.Example
		} else {
			// Use zero value
			exampleMap[jsonName] = getZeroValue(field.Type)
		}
	}

	// Marshal the map to JSON
	exampleJSON, err := json.MarshalIndent(exampleMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error creating example JSON: %w", err)
	}

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

	// Create a map for our example
	exampleMap := make(map[string]interface{})

	// Populate the example map with fields that aren't excluded
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		// jsonTag := field.Tag.Get("json")
		jsonName := getJSONFieldName(field)

		// Skip fields with no JSON name
		if jsonName == "" {
			continue
		}

		// Skip excluded fields
		if override, exists := overrideMap[fieldName]; exists && override.Exclude {
			continue
		}

		// Handle field with override
		if override, exists := overrideMap[fieldName]; exists && override.Example != nil {
			switch ex := override.Example.(type) {
			case []byte:
				// Try to parse the bytes as JSON first
				var jsonData interface{}
				if err := json.Unmarshal(ex, &jsonData); err == nil {
					exampleMap[jsonName] = jsonData
				} else {
					// If not valid JSON, use as string
					exampleMap[jsonName] = string(ex)
				}
			case string:
				// Try to parse the string as JSON if it looks like JSON
				if len(ex) > 0 && (ex[0] == '{' || ex[0] == '[') {
					var jsonData interface{}
					if err := json.Unmarshal([]byte(ex), &jsonData); err == nil {
						exampleMap[jsonName] = jsonData
					} else {
						exampleMap[jsonName] = ex
					}
				} else {
					exampleMap[jsonName] = ex
				}
			case json.RawMessage:
				// Parse raw JSON
				var jsonData interface{}
				if err := json.Unmarshal(ex, &jsonData); err == nil {
					exampleMap[jsonName] = jsonData
				} else {
					exampleMap[jsonName] = string(ex)
				}
			default:
				exampleMap[jsonName] = override.Example
			}
		} else {
			// Use zero value for fields without overrides
			exampleMap[jsonName] = getZeroValue(field.Type)
		}
	}

	// Marshal the example map
	example, err := json.MarshalIndent(exampleMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error creating example JSON: %w", err)
	}

	return &Response{
		StatusCode:  statusCode,
		Description: description,
		ContentType: contentType,
		Fields:      fields,
		Example:     example,
	}, nil
}
