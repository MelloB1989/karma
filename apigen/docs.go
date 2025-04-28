package apigen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
