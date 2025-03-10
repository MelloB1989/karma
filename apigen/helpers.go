package apigen

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func writeFieldToMarkdown(sb *strings.Builder, field RequestBodyField, indent int) {
	indentation := strings.Repeat("  ", indent)
	required := ""
	if field.Required {
		required = " (required)"
	}

	sb.WriteString(fmt.Sprintf("%s- `%s` (%s): %s%s", indentation, field.JsonName, field.Type, field.Description, required))

	if field.Example != nil {
		exampleStr, _ := json.Marshal(field.Example)
		sb.WriteString(fmt.Sprintf(" (Example: `%s`)", string(exampleStr)))
	}

	sb.WriteString("\n")

	// If this field has nested fields, write them with increased indentation
	if len(field.Fields) > 0 {
		for _, nestedField := range field.Fields {
			writeFieldToMarkdown(sb, nestedField, indent+1)
		}
	}
}

// extractStructFields recursively extracts field definitions from a struct
func extractStructFields(t reflect.Type, overrides map[string]FieldOverride, prefix string) []RequestBodyField {
	var fields []RequestBodyField
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		fieldName := field.Name
		qualifiedName := prefix + fieldName
		// Check if this field should be excluded via override
		if override, exists := overrides[qualifiedName]; exists && override.Exclude {
			continue
		}
		// Extract JSON tag
		jsonTag := field.Tag.Get("json")
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}
		if jsonName == "-" {
			continue // Skip fields explicitly excluded with json:"-"
		}
		// Extract description
		description := field.Tag.Get("description")
		// Determine if required (assume required by default)
		required := !strings.Contains(jsonTag, "omitempty")
		// Create base field
		requestField := RequestBodyField{
			Name:        fieldName,
			JsonName:    jsonName,
			Type:        field.Type.String(),
			Required:    required,
			Description: description,
		}
		// Apply overrides if they exist
		if override, exists := overrides[qualifiedName]; exists {
			if override.JsonName != "" {
				requestField.JsonName = override.JsonName
			}
			if override.Type != "" {
				requestField.Type = override.Type
			}
			if override.Required != nil {
				requestField.Required = *override.Required
			}
			if override.Description != "" {
				requestField.Description = override.Description
			}
			if override.Example != nil {
				requestField.Example = override.Example
			}
		}

		// Handle nested structs - both named and anonymous
		if field.Type.Kind() == reflect.Struct {
			nestedPrefix := qualifiedName + "."
			requestField.Fields = extractStructFields(field.Type, overrides, nestedPrefix)
		} else if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			// Handle pointer to struct
			nestedPrefix := qualifiedName + "."
			requestField.Fields = extractStructFields(field.Type.Elem(), overrides, nestedPrefix)
		} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			// Handle slice of structs - we extract the fields from the element type
			// Note: This will only show the structure, not array-specific info
			nestedPrefix := qualifiedName + "[]."
			requestField.Fields = extractStructFields(field.Type.Elem(), overrides, nestedPrefix)
		} else if field.Type.Kind() == reflect.Map && field.Type.Elem().Kind() == reflect.Struct {
			// Handle maps with struct values - we extract fields from the value type
			// Note: This will only show the structure of values, not map-specific info
			nestedPrefix := qualifiedName + "{}."
			requestField.Fields = extractStructFields(field.Type.Elem(), overrides, nestedPrefix)
		}

		// Special handling for anonymous structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// For anonymous embedded structs, we want to flatten the fields
			// rather than nesting them under the embedded struct name
			embeddedFields := extractStructFields(field.Type, overrides, prefix)
			fields = append(fields, embeddedFields...)
		} else {
			fields = append(fields, requestField)
		}
	}
	return fields
}

// extractPathParams extracts parameter names from a path like "/users/{userId}/posts/{postId}"
func extractPathParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			// Extract the parameter name without braces
			paramName := part[1 : len(part)-1]
			params = append(params, paramName)
		} else if strings.HasPrefix(part, "${") && strings.HasSuffix(part, "}") {
			// Handle ${parameter} format
			paramName := part[2 : len(part)-1]
			params = append(params, paramName)
		} else if strings.HasPrefix(part, ":") {
			// Handle :parameter format
			paramName := part[1:]
			params = append(params, paramName)
		}
	}

	return params
}
