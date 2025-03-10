package apigen

import (
	"fmt"
	"strings"
)

// Helper functions to generate different export formats

func generatePostmanCollection(api *APIDefinition) map[string]any {
	collection := map[string]any{
		"info": map[string]any{
			"name":        api.Name,
			"description": api.Description,
			"schema":      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"item":     []any{},
		"variable": []any{},
	}

	// Add global variables
	variables := []any{}
	for key, value := range api.GlobalVariables {
		variables = append(variables, map[string]string{
			"key":   key,
			"value": value,
		})
	}
	collection["variable"] = variables

	items := []any{}
	for _, endpoint := range api.Endpoints {
		// Use first base URL if available
		baseURL := ""
		if len(api.BaseURLs) > 0 {
			baseURL = api.BaseURLs[0]
		}

		path := endpoint.Path
		// Replace path parameters in Postman format
		for _, param := range endpoint.PathParams {
			path = strings.ReplaceAll(path, "{"+param.Name+"}", ":"+param.Name)
			path = strings.ReplaceAll(path, "${"+param.Name+"}", ":"+param.Name)
		}

		item := map[string]any{
			"name": endpoint.Summary,
			"request": map[string]any{
				"method": endpoint.Method,
				"url": map[string]any{
					"raw":  baseURL + path,
					"host": strings.Split(strings.TrimPrefix(strings.TrimPrefix(baseURL, "http://"), "https://"), "/")[0],
					"path": strings.Split(strings.Trim(path, "/"), "/"),
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
					"key":   string(key),
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

		// Add path parameters
		if len(endpoint.PathParams) > 0 {
			pathVars := []any{}
			for _, param := range endpoint.PathParams {
				pathVars = append(pathVars, map[string]any{
					"key":         param.Name,
					"value":       param.Example,
					"description": param.Description,
				})
			}

			if urlData, ok := item["request"].(map[string]any)["url"].(map[string]any); ok {
				urlData["variable"] = pathVars
			}
		}

		// Add query parameters
		if len(endpoint.QueryParams) > 0 {
			queryParams := []any{}
			for _, param := range endpoint.QueryParams {
				queryParams = append(queryParams, map[string]any{
					"key":         param.Name,
					"value":       param.Example,
					"description": param.Description,
					"disabled":    !param.Required,
				})
			}

			if urlData, ok := item["request"].(map[string]any)["url"].(map[string]any); ok {
				urlData["query"] = queryParams
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

			if len(resp.Headers) > 0 {
				headersList := []any{}
				for key, value := range resp.Headers {
					headersList = append(headersList, map[string]string{
						"key":   string(key),
						"value": value,
					})
				}
				response["header"] = headersList
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
		"servers": []map[string]string{},
		"paths":   map[string]any{},
	}

	// Add base URLs as servers
	servers := []map[string]string{}
	for _, url := range api.BaseURLs {
		servers = append(servers, map[string]string{
			"url": url,
		})
	}
	openapi["servers"] = servers

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

		// Add path parameters
		for _, param := range endpoint.PathParams {
			parameters = append(parameters, map[string]any{
				"name":        param.Name,
				"in":          "path",
				"description": param.Description,
				"required":    true, // Path parameters are always required
				"schema": map[string]string{
					"type": param.Type,
				},
				"example": param.Example,
			})
		}

		// Add query parameters
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

		// Add header parameters
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
					string(endpoint.RequestBody.ContentType): map[string]any{
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

			if resp.Schema != nil || resp.Example != nil {
				content := map[string]any{}

				if resp.ContentType != "" {
					contentType := resp.ContentType
					contentData := map[string]any{}

					if resp.Schema != nil {
						contentData["schema"] = resp.Schema
					}

					if resp.Example != nil {
						contentData["example"] = resp.Example
					}

					content[string(contentType)] = contentData
					respData["content"] = content
				}
			}

			if len(resp.Headers) > 0 {
				headers := map[string]any{}
				for name, value := range resp.Headers {
					headers[string(name)] = map[string]any{
						"schema": map[string]string{
							"type": "string",
						},
						"example": value,
					}
				}
				respData["headers"] = headers
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

	// Add components section for schemas from structured fields
	components := map[string]any{
		"schemas": map[string]any{},
	}

	// TODO: Generate schemas from structured fields

	openapi["components"] = components

	return openapi
}

func generateMarkdown(api *APIDefinition) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", api.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", api.Description))

	sb.WriteString("## Base URLs\n\n")
	for _, url := range api.BaseURLs {
		sb.WriteString(fmt.Sprintf("- `%s`\n", url))
	}
	sb.WriteString("\n")

	if len(api.GlobalVariables) > 0 {
		sb.WriteString("## Global Variables\n\n")
		for key, value := range api.GlobalVariables {
			sb.WriteString(fmt.Sprintf("- `%s`: %s\n", key, value))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Endpoints\n\n")
	for _, endpoint := range api.Endpoints {
		sb.WriteString(fmt.Sprintf("### %s\n\n", endpoint.Summary))
		sb.WriteString(fmt.Sprintf("**Path:** `%s`\n\n", endpoint.Path))
		sb.WriteString(fmt.Sprintf("**Method:** `%s`\n\n", endpoint.Method))

		if endpoint.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", endpoint.Description))
		}

		if len(endpoint.PathParams) > 0 {
			sb.WriteString("#### Path Parameters\n\n")
			for _, param := range endpoint.PathParams {
				sb.WriteString(fmt.Sprintf("- `%s`: %s (required)", param.Name, param.Description))
				if param.Example != "" {
					sb.WriteString(fmt.Sprintf(" (Example: `%s`)", param.Example))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
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

			if len(endpoint.RequestBody.Fields) > 0 {
				sb.WriteString("Fields:\n\n")
				for _, field := range endpoint.RequestBody.Fields {
					writeFieldToMarkdown(&sb, field, 0)
				}
				sb.WriteString("\n")
			}

			if endpoint.RequestBody.Example != nil {
				sb.WriteString("Example:\n\n```json\n")
				sb.WriteString(string(endpoint.RequestBody.Example))
				sb.WriteString("\n```\n\n")
			}
		}

		sb.WriteString("#### Responses\n\n")
		for _, resp := range endpoint.Responses {
			sb.WriteString(fmt.Sprintf("**%d**: %s\n\n", resp.StatusCode, resp.Description))

			if len(resp.Fields) > 0 {
				sb.WriteString("Fields:\n\n")
				for _, field := range resp.Fields {
					writeFieldToMarkdown(&sb, field, 0)
				}
				sb.WriteString("\n")
			}

			if resp.Example != nil {
				sb.WriteString("Example:\n\n```json\n")
				sb.WriteString(string(resp.Example))
				sb.WriteString("\n```\n\n")
			}
		}
	}

	return sb.String()
}
