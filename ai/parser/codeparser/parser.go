package codeparser

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MelloB1989/karma/ai"
)

// CodeOperation represents the type of code operation
type CodeOperation string

const (
	Create CodeOperation = "CREATE"
	Read   CodeOperation = "READ"
	Update CodeOperation = "UPDATE"
	Delete CodeOperation = "DELETE"
)

// CodeLanguage defines supported programming languages
type CodeLanguage string

const (
	Go         CodeLanguage = "go"
	Python     CodeLanguage = "python"
	Java       CodeLanguage = "java"
	JavaScript CodeLanguage = "javascript"
	TypeScript CodeLanguage = "typescript"
)

// FileContent represents a file's content to be modified
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// CodeChange represents a specific code change
type CodeChange struct {
	Operation   CodeOperation `json:"operation"`
	Path        string        `json:"path"`
	OldCode     string        `json:"oldCode,omitempty"`
	NewCode     string        `json:"newCode,omitempty"`
	LineStart   int           `json:"lineStart,omitempty"`
	LineEnd     int           `json:"lineEnd,omitempty"`
	Description string        `json:"description,omitempty"`
}

// CodeChanges represents multiple code changes
type CodeChanges struct {
	Changes     []CodeChange `json:"changes"`
	Description string       `json:"description,omitempty"`
}

// ProjectContext contains information about the project structure
type ProjectContext struct {
	RootDir    string                 `json:"-"`
	Files      map[string]FileContent `json:"-"`
	FileTree   []FileNode             `json:"fileTree"`
	Language   CodeLanguage           `json:"language"`
	EntryPoint string                 `json:"entryPoint,omitempty"`
}

// FileNode represents a node in the file tree
type FileNode struct {
	Path     string     `json:"path"`
	IsDir    bool       `json:"isDir"`
	Children []FileNode `json:"children,omitempty"`
}

// CodeParser represents the main parser configuration
type CodeParser struct {
	model      ai.Models
	options    []ai.Option
	client     *ai.KarmaAI
	maxRetries int
	language   CodeLanguage
	context    *ProjectContext
}

// CodeParserOption defines functional options for CodeParser
type CodeParserOption func(*CodeParser)

// WithMaxRetries sets the maximum number of retries for parsing
func WithMaxRetries(retries int) CodeParserOption {
	return func(p *CodeParser) {
		p.maxRetries = retries
	}
}

// WithAIClient directly sets the KarmaAI client
func WithAIClient(client *ai.KarmaAI) CodeParserOption {
	return func(p *CodeParser) {
		p.client = client
	}
}

// WithModel sets the AI model to use
func WithModel(model ai.Models) CodeParserOption {
	return func(p *CodeParser) {
		p.model = model
	}
}

// WithAIOptions sets additional options for the KarmaAI client
func WithAIOptions(options ...ai.Option) CodeParserOption {
	return func(p *CodeParser) {
		p.options = options
	}
}

// WithLanguage sets the programming language
func WithLanguage(language CodeLanguage) CodeParserOption {
	return func(p *CodeParser) {
		p.language = language
	}
}

// WithProjectContext sets the project context
func WithProjectContext(context *ProjectContext) CodeParserOption {
	return func(p *CodeParser) {
		p.context = context
	}
}

// NewCodeParser creates a new parser instance
func NewCodeParser(opts ...CodeParserOption) *CodeParser {
	// Default configuration
	p := &CodeParser{
		model:      (ai.ApacClaude3_5Sonnet20240620V1),
		maxRetries: 3,
		language:   Go,
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}

	// Initialize client if not provided
	if p.client == nil {
		p.client = ai.NewKarmaAI(p.model, p.options...)
	}

	return p
}

// BuildProjectContext builds a project context from a directory
func BuildProjectContext(rootDir string, language CodeLanguage) (*ProjectContext, error) {
	context := &ProjectContext{
		RootDir:  rootDir,
		Files:    make(map[string]FileContent),
		Language: language,
	}

	// Build file tree and collect file contents
	fileTree, err := buildFileTree(rootDir, rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to build file tree: %w", err)
	}
	context.FileTree = fileTree

	// Find possible entry points based on language
	context.EntryPoint = findEntryPoint(rootDir, language)

	return context, nil
}

// buildFileTree recursively builds a file tree starting from a directory
func buildFileTree(rootDir, currentDir string) ([]FileNode, error) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, err
	}

	var nodes []FileNode
	for _, entry := range entries {
		path, err := filepath.Rel(rootDir, filepath.Join(currentDir, entry.Name()))
		if err != nil {
			continue
		}

		// Skip hidden files and directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.IsDir() {
			children, err := buildFileTree(rootDir, filepath.Join(currentDir, entry.Name()))
			if err != nil {
				continue
			}
			nodes = append(nodes, FileNode{
				Path:     path,
				IsDir:    true,
				Children: children,
			})
		} else {
			nodes = append(nodes, FileNode{
				Path:  path,
				IsDir: false,
			})
		}
	}

	return nodes, nil
}

// findEntryPoint tries to find the main entry point of the project
func findEntryPoint(rootDir string, language CodeLanguage) string {
	switch language {
	case Go:
		// Look for main.go files
		var mainFiles []string
		filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == "main.go" {
				relPath, err := filepath.Rel(rootDir, path)
				if err == nil {
					mainFiles = append(mainFiles, relPath)
				}
			}
			return nil
		})
		if len(mainFiles) > 0 {
			return mainFiles[0]
		}
	case Python:
		// Look for __main__.py or app.py or main.py
		candidates := []string{"__main__.py", "app.py", "main.py"}
		for _, candidate := range candidates {
			var found string
			filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && info.Name() == candidate {
					relPath, err := filepath.Rel(rootDir, path)
					if err == nil {
						found = relPath
						return filepath.SkipAll
					}
				}
				return nil
			})
			if found != "" {
				return found
			}
		}
	}
	return ""
}

// LoadFileContent loads the content of a file
func (p *CodeParser) LoadFileContent(path string) (string, error) {
	fullPath := filepath.Join(p.context.RootDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	p.context.Files[path] = FileContent{
		Path:    path,
		Content: string(content),
	}
	return string(content), nil
}

// LoadAllFiles loads all files of a specific type into memory
func (p *CodeParser) LoadAllFiles() error {
	extensions := getExtensionsForLanguage(p.language)
	return filepath.Walk(p.context.RootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		for _, validExt := range extensions {
			if ext == validExt {
				relPath, err := filepath.Rel(p.context.RootDir, path)
				if err != nil {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				p.context.Files[relPath] = FileContent{
					Path:    relPath,
					Content: string(content),
				}
				break
			}
		}
		return nil
	})
}

// getExtensionsForLanguage returns file extensions for a language
func getExtensionsForLanguage(language CodeLanguage) []string {
	switch language {
	case Go:
		return []string{".go"}
	case Python:
		return []string{".py"}
	case Java:
		return []string{".java"}
	case JavaScript:
		return []string{".js", ".jsx"}
	case TypeScript:
		return []string{".ts", ".tsx"}
	default:
		return []string{}
	}
}

// createPromptForCodeChanges generates a prompt for code changes
func (p *CodeParser) createPromptForCodeChanges(prompt string) string {
	var sb strings.Builder

	// Add general instructions
	sb.WriteString("# Code Modification Request\n\n")
	sb.WriteString("You are a coding assistant specialized in modifying code. ")
	sb.WriteString("I'll provide you with a project structure and some code files, ")
	sb.WriteString("and I need you to make specific changes based on my request.\n\n")

	// Add context about the project structure
	sb.WriteString("## Project Structure\n\n")
	sb.WriteString("```\n")
	p.writeFileTree(&sb, p.context.FileTree, 0)
	sb.WriteString("```\n\n")

	// Add information about key files
	sb.WriteString("## Key Files\n\n")

	if p.context.EntryPoint != "" {
		sb.WriteString(fmt.Sprintf("Entry point: `%s`\n\n", p.context.EntryPoint))

		// Include content of entry point
		if content, ok := p.context.Files[p.context.EntryPoint]; ok {
			sb.WriteString("```")
			sb.WriteString(string(p.language))
			sb.WriteString("\n// " + p.context.EntryPoint + "\n")
			sb.WriteString(content.Content)
			sb.WriteString("\n```\n\n")
		} else {
			content, err := p.LoadFileContent(p.context.EntryPoint)
			if err == nil {
				sb.WriteString("```")
				sb.WriteString(string(p.language))
				sb.WriteString("\n// " + p.context.EntryPoint + "\n")
				sb.WriteString(content)
				sb.WriteString("\n```\n\n")
			}
		}
	}

	// Add other relevant files
	relevantFiles := p.findRelevantFiles(prompt)
	if len(relevantFiles) > 0 {
		sb.WriteString("## Relevant Files\n\n")
		for _, file := range relevantFiles {
			if content, ok := p.context.Files[file]; ok {
				sb.WriteString("```")
				sb.WriteString(string(p.language))
				sb.WriteString("\n// " + file + "\n")
				sb.WriteString(content.Content)
				sb.WriteString("\n```\n\n")
			} else {
				content, err := p.LoadFileContent(file)
				if err == nil {
					sb.WriteString("```")
					sb.WriteString(string(p.language))
					sb.WriteString("\n// " + file + "\n")
					sb.WriteString(content)
					sb.WriteString("\n```\n\n")
				}
			}
		}
	}

	// Add response format instructions
	sb.WriteString("## Required Changes\n\n")
	sb.WriteString(prompt + "\n\n")

	sb.WriteString("## Response Format\n\n")
	sb.WriteString("Respond ONLY with a JSON object in the following format:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "changes": [
    {
      "operation": "CREATE|UPDATE|DELETE",
      "path": "relative/path/to/file",
      "newCode": "full content for CREATE; new content for UPDATE",
      "oldCode": "for UPDATE, the code being replaced",
      "lineStart": 0,
      "lineEnd": 0,
      "description": "Description of the change"
    }
  ],
  "description": "Overall description of changes"
}
` + "\n```\n\n")

	sb.WriteString("Guidelines for each operation:\n")
	sb.WriteString("- CREATE: Provide the full file content in 'newCode'\n")
	sb.WriteString("- UPDATE: You can update a full file by providing the entire content, or update a specific part by providing 'oldCode' and 'newCode'\n")
	sb.WriteString("- DELETE: Just specify the path to delete\n\n")
	sb.WriteString("IMPORTANT: Do not include any markdown formatting or explanations outside the JSON.\n")

	return sb.String()
}

// writeFileTree recursively writes file tree to string builder
func (p *CodeParser) writeFileTree(sb *strings.Builder, nodes []FileNode, indent int) {
	for _, node := range nodes {
		for i := 0; i < indent; i++ {
			sb.WriteString("  ")
		}

		if node.IsDir {
			sb.WriteString("📁 " + node.Path + "/\n")
			p.writeFileTree(sb, node.Children, indent+1)
		} else {
			sb.WriteString("📄 " + node.Path + "\n")
		}
	}
}

// findRelevantFiles finds files relevant to the prompt
func (p *CodeParser) findRelevantFiles(prompt string) []string {
	var relevantFiles []string

	// Extract file paths mentioned in the prompt
	fileRegex := regexp.MustCompile(`(?i)['"]([\w\-./]+\.\w+)['"]`)
	matches := fileRegex.FindAllStringSubmatch(prompt, -1)

	pathMap := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			path := match[1]
			pathMap[path] = true
		}
	}

	// Add files explicitly mentioned
	for path := range pathMap {
		relevantFiles = append(relevantFiles, path)
	}

	// For Go, look for imports mentioned in the prompt
	if p.language == Go {
		// Simple heuristic: look for package names
		words := strings.Fields(prompt)
		for _, word := range words {
			word = strings.Trim(word, ",.;()\"'")

			// Check if this word appears as a directory in the project
			for _, node := range p.context.FileTree {
				if node.IsDir && (node.Path == word || strings.HasSuffix(node.Path, "/"+word)) {
					// Find main file in this package
					for filePath := range p.context.Files {
						if strings.HasPrefix(filePath, node.Path+"/") && strings.HasSuffix(filePath, ".go") {
							if !pathMap[filePath] {
								relevantFiles = append(relevantFiles, filePath)
								pathMap[filePath] = true
							}
							break
						}
					}
				}
			}
		}
	}

	return relevantFiles
}

// ParseCodeChanges sends a prompt to the AI and parses the code changes
func (p *CodeParser) ParseCodeChanges(prompt string) (*CodeChanges, error) {
	// Create a prompt for code changes
	codePrompt := p.createPromptForCodeChanges(prompt)

	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Send prompt to the AI
		resp, err := p.client.GenerateFromSinglePrompt(codePrompt)
		if err != nil {
			lastErr = fmt.Errorf("AI request failed: %w", err)
			continue
		}
		log.Println("Total token burn: ", resp.Tokens)

		// Clean the response to extract just the JSON
		cleanedJSON := cleanResponse(resp.AIResponse)

		// Try to parse the JSON
		var changes CodeChanges
		err = json.Unmarshal([]byte(cleanedJSON), &changes)
		if err == nil {
			// Validate the changes
			if err := p.validateCodeChanges(&changes); err != nil {
				if attempt == p.maxRetries-1 {
					return &changes, fmt.Errorf("invalid code changes: %w", err)
				}

				// Try to fix the issues
				fixedChanges, fixErr := p.fixInvalidChanges(&changes, err)
				if fixErr == nil {
					return fixedChanges, nil
				}

				lastErr = fmt.Errorf("failed to fix invalid changes: %w", fixErr)
				continue
			}

			// Success!
			return &changes, nil
		}

		lastErr = fmt.Errorf("JSON parsing failed: %w, Response: %s", err, cleanedJSON)

		// For retries, add more explicit instructions about the failure
		codePrompt = fmt.Sprintf(
			"Your previous response could not be parsed correctly. Error: %v\n\n"+
				"Please provide a response in STRICTLY valid JSON format, with NO additional text:\n\n%s",
			err, codePrompt)
	}

	return nil, lastErr
}

// cleanResponse extracts JSON from an AI response, handling various formats
func cleanResponse(response string) string {
	// Try to find JSON between code blocks
	re := regexp.MustCompile("```(?:json)?\n?(.*?)```")
	matches := re.FindAllStringSubmatch(response, -1)
	if len(matches) > 0 {
		// Use the last match (in case there are multiple code blocks)
		return strings.TrimSpace(matches[len(matches)-1][1])
	}

	// Try to find JSON-like content (starting with { and ending with })
	re = regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(response)
	if match != "" {
		return strings.TrimSpace(match)
	}

	// If all else fails, return the cleaned response
	return strings.TrimSpace(response)
}

// validateCodeChanges validates the code changes
func (p *CodeParser) validateCodeChanges(changes *CodeChanges) error {
	if len(changes.Changes) == 0 {
		return fmt.Errorf("no changes provided")
	}

	for i, change := range changes.Changes {
		// Basic validation
		if change.Operation == "" {
			return fmt.Errorf("change %d has no operation", i)
		}
		if change.Path == "" {
			return fmt.Errorf("change %d has no path", i)
		}

		// Operation-specific validation
		switch change.Operation {
		case Create:
			if change.NewCode == "" {
				return fmt.Errorf("CREATE operation %d has no newCode", i)
			}
			if p.language == Go {
				if err := validateGoCode(change.NewCode); err != nil {
					return fmt.Errorf("invalid Go code in CREATE operation %d: %w", i, err)
				}
			}
		case Update:
			if change.NewCode == "" {
				return fmt.Errorf("UPDATE operation %d has no newCode", i)
			}
			if change.OldCode == "" && (change.LineStart == 0 && change.LineEnd == 0) {
				// Load the file to get oldCode
				content, err := p.LoadFileContent(change.Path)
				if err != nil {
					return fmt.Errorf("UPDATE operation %d references non-existent file: %s", i, change.Path)
				}
				changes.Changes[i].OldCode = content
			}
			if p.language == Go {
				if err := validateGoCode(change.NewCode); err != nil {
					return fmt.Errorf("invalid Go code in UPDATE operation %d: %w", i, err)
				}
			}
		case Delete:
			// No additional validation needed
		default:
			return fmt.Errorf("invalid operation %s in change %d", change.Operation, i)
		}
	}

	return nil
}

// fixInvalidChanges attempts to fix invalid code changes
func (p *CodeParser) fixInvalidChanges(changes *CodeChanges, validationErr error) (*CodeChanges, error) {
	// Create a prompt to fix the issues
	var sb strings.Builder
	sb.WriteString("I received the following code changes but they have validation issues.\n\n")
	sb.WriteString("## Code Changes\n\n")

	changesJSON, err := json.MarshalIndent(changes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal changes: %w", err)
	}
	sb.WriteString("```json\n")
	sb.WriteString(string(changesJSON))
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Validation Issues\n\n")
	sb.WriteString(validationErr.Error())
	sb.WriteString("\n\n")

	sb.WriteString("Please fix these issues and provide the corrected JSON response. ")
	sb.WriteString("Ensure all code is valid " + string(p.language) + " code and that the response follows the correct format:")
	sb.WriteString("\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "changes": [
    {
      "operation": "CREATE|UPDATE|DELETE",
      "path": "relative/path/to/file",
      "newCode": "full content for CREATE; new content for UPDATE",
      "oldCode": "for UPDATE, the code being replaced",
      "lineStart": 0,
      "lineEnd": 0,
      "description": "Description of the change"
    }
  ],
  "description": "Overall description of changes"
}` + "\n```\n\n")
	sb.WriteString("IMPORTANT: Do not include any markdown formatting or explanations outside the JSON.\n")

	prompt := sb.String()

	// Try to fix the changes
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Send prompt to the AI
		resp, err := p.client.GenerateFromSinglePrompt(prompt)
		if err != nil {
			continue
		}

		// Clean the response to extract just the JSON
		cleanedJSON := cleanResponse(resp.AIResponse)

		// Try to parse the JSON
		var fixedChanges CodeChanges
		err = json.Unmarshal([]byte(cleanedJSON), &fixedChanges)
		if err == nil {
			// Validate the fixed changes
			if err := p.validateCodeChanges(&fixedChanges); err != nil {
				continue
			}

			// Success!
			return &fixedChanges, nil
		}
	}

	return nil, fmt.Errorf("failed to fix invalid changes after %d attempts", p.maxRetries)
}

// validateGoCode validates Go code syntax
// This is a simple validation; a more thorough one would use go/parser
func validateGoCode(code string) error {
	// Check for basic syntax
	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("unbalanced braces")
	}
	if strings.Count(code, "(") != strings.Count(code, ")") {
		return fmt.Errorf("unbalanced parentheses")
	}

	// Check for missing imports
	if strings.Contains(code, "fmt.") && !strings.Contains(code, `"fmt"`) && !strings.Contains(code, "`fmt`") {
		return fmt.Errorf("using fmt package without importing it")
	}

	return nil
}

// ApplyChanges applies the code changes to the filesystem
func (p *CodeParser) ApplyChanges(changes *CodeChanges) error {
	for _, change := range changes.Changes {
		switch change.Operation {
		case Create:
			if err := p.createFile(change.Path, change.NewCode); err != nil {
				return fmt.Errorf("failed to create file %s: %w", change.Path, err)
			}
		case Update:
			if err := p.updateFile(change); err != nil {
				return fmt.Errorf("failed to update file %s: %w", change.Path, err)
			}
		case Delete:
			if err := p.deleteFile(change.Path); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", change.Path, err)
			}
		}
	}

	return nil
}

// createFile creates a new file with the given content
func (p *CodeParser) createFile(path string, content string) error {
	fullPath := filepath.Join(p.context.RootDir, path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create the file
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// updateFile updates an existing file
func (p *CodeParser) updateFile(change CodeChange) error {
	fullPath := filepath.Join(p.context.RootDir, change.Path)

	// Check if file exists
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", change.Path)
	}

	// If we're updating the entire file
	if change.OldCode != "" && change.LineStart == 0 && change.LineEnd == 0 {
		// Read the current content
		currentContent, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		// If oldCode doesn't match, try to find it
		if change.OldCode != string(currentContent) {
			newContent := strings.Replace(string(currentContent), change.OldCode, change.NewCode, 1)
			if newContent == string(currentContent) {
				// If oldCode wasn't found, replace the entire file
				return os.WriteFile(fullPath, []byte(change.NewCode), 0644)
			}
			return os.WriteFile(fullPath, []byte(newContent), 0644)
		}

		// Replace the entire file
		return os.WriteFile(fullPath, []byte(change.NewCode), 0644)
	}

	// If we're updating a specific part
	if change.LineStart > 0 && change.LineEnd >= change.LineStart {
		// Read the current content
		currentContent, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		lines := strings.Split(string(currentContent), "\n")

		// Check if line numbers are valid
		if change.LineStart > len(lines) || change.LineEnd > len(lines) {
			return fmt.Errorf("invalid line numbers: start=%d, end=%d, total=%d",
				change.LineStart, change.LineEnd, len(lines))
		}

		// Replace the lines
		newLines := append(
			lines[:change.LineStart-1],
			append(
				strings.Split(change.NewCode, "\n"),
				lines[change.LineEnd:]...,
			)...,
		)

		// Write the new content
		return os.WriteFile(fullPath, []byte(strings.Join(newLines, "\n")), 0644)
	}

	// If no specific part is provided, replace the entire file
	return os.WriteFile(fullPath, []byte(change.NewCode), 0644)
}

// deleteFile deletes a file
func (p *CodeParser) deleteFile(path string) error {
	fullPath := filepath.Join(p.context.RootDir, path)
	return os.Remove(fullPath)
}
