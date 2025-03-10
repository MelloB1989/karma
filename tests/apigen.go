package tests

import (
	"fmt"
	"log"

	"github.com/MelloB1989/karma/apigen"
)

type GitLabIssue struct {
	Name        string   `json:"name" description:"Issue name"`
	Description string   `json:"description" description:"Issue description"`
	Labels      []string `json:"labels" description:"Issue labels"`
	Assignees   []string `json:"assignees,omitempty" description:"Users assigned to this issue"`
	DueDate     string   `json:"due_date,omitempty" description:"Issue due date (YYYY-MM-DD)"`
}

type GitLabIssueResponse struct {
	ID          int      `json:"id" description:"Issue ID"`
	Name        string   `json:"name" description:"Issue name"`
	Description string   `json:"description" description:"Issue description"`
	Labels      []string `json:"labels" description:"Issue labels"`
	Assignees   []string `json:"assignees" description:"Users assigned to this issue"`
	DueDate     string   `json:"due_date" description:"Issue due date (YYYY-MM-DD)"`
	CreatedAt   string   `json:"created_at" description:"Creation timestamp"`
	UpdatedAt   string   `json:"updated_at" description:"Last update timestamp"`
	WebURL      string   `json:"web_url" description:"URL to view the issue in browser"`
}

func TestAPIGen() {
	// Initialize API definition with output details
	api := apigen.NewAPIDefinition(
		"GitLab Issues API",
		"API for managing issues in GitLab projects",
		[]string{
			"https://gitlab.com/api/v4",
			"https://gitlab.example.com/api/v4",
		},
		"./docstest",    // Output folder
		"gitlab_issues", // Base filename for exports
	)

	// Add global variables
	api.AddGlobalVariable("api_version", "v4")
	api.AddGlobalVariable("default_per_page", "20")

	// Create request body from struct with field overrides
	requestBody, err := apigen.RequestBodyFromStruct(
		GitLabIssue{},
		"application/json",
		true,
		[]apigen.FieldOverride{
			{
				Name:        "DueDate",
				Description: "Due date in ISO format (YYYY-MM-DD)",
				Example:     "2025-06-30",
			},
			{
				Name:     "Assignees",
				Example:  []string{"user1", "user2"},
				Required: new(bool), // false
			},
		},
	)
	if err != nil {
		log.Fatalf("Error creating request body: %v", err)
	}

	// Create response from struct with field overrides
	successResponse, err := apigen.ResponseFromStruct(
		200,
		"Issue created successfully",
		GitLabIssueResponse{},
		"application/json",
		[]apigen.FieldOverride{
			{
				Name:    "ID",
				Example: 42,
			},
			{
				Name:    "WebURL",
				Example: "https://gitlab.com/mygroup/myproject/-/issues/42",
			},
		},
	)
	if err != nil {
		log.Fatalf("Error creating response: %v", err)
	}

	// Add endpoints with dynamic path parameters
	api.AddEndpoint(apigen.Endpoint{
		Path:        "/projects/{project_id}/issues",
		Method:      "GET",
		Summary:     "List project issues",
		Description: "Get a list of issues for a specific project",
		PathParams: []apigen.Parameter{
			{
				Name:        "project_id",
				Type:        "integer",
				Required:    true,
				Description: "The ID or URL-encoded path of the project",
				Example:     "12345",
			},
		},
		QueryParams: []apigen.Parameter{
			{
				Name:        "state",
				Type:        "string",
				Required:    false,
				Description: "Return issues with the specified state (opened, closed, or all)",
				Example:     "opened",
			},
			{
				Name:        "labels",
				Type:        "string",
				Required:    false,
				Description: "Comma-separated list of label names",
				Example:     "bug,critical",
			},
		},
		Headers: apigen.KarmaHeaders{
			apigen.HeaderPrivateToken: "YOUR_GITLAB_TOKEN",
			apigen.HeaderContentType:  "application/json",
			apigen.HeaderAccept:       "application/json",
		},
		Responses: []apigen.Response{
			{
				StatusCode:  200,
				Description: "List of issues",
				ContentType: "application/json",
				Example:     []byte(`[{"id": 1, "name": "Bug report", "description": "App crashes on startup"}, {"id": 2, "name": "Feature request", "description": "Add dark mode"}]`),
			},
			{
				StatusCode:  401,
				Description: "Unauthorized",
				Example:     []byte(`{"message": "401 Unauthorized"}`),
			},
		},
	})

	api.AddEndpoint(apigen.Endpoint{
		Path:        "/projects/{project_id}/issues",
		Method:      "POST",
		Summary:     "Create new issue",
		Description: "Creates a new issue in the specified project",
		PathParams: []apigen.Parameter{
			{
				Name:        "project_id",
				Type:        "integer",
				Required:    true,
				Description: "The ID or URL-encoded path of the project",
				Example:     "12345",
			},
		},
		Headers: apigen.KarmaHeaders{
			apigen.HeaderPrivateToken: "YOUR_GITLAB_TOKEN",
			apigen.HeaderContentType:  "application/json",
			apigen.HeaderAccept:       "application/json",
		},
		RequestBody: requestBody,
		Responses: []apigen.Response{
			*successResponse,
			{
				StatusCode:  400,
				Description: "Bad request",
				ContentType: "application/json",
				Example:     []byte(`{"message": "Required fields missing or invalid"}`),
			},
		},
	})

	api.AddEndpoint(apigen.Endpoint{
		Path:        "/projects/{project_id}/issues/{issue_id}",
		Method:      "GET",
		Summary:     "Get issue details",
		Description: "Get details of a specific issue",
		// Path parameters will be automatically detected from the URL pattern
		Headers: apigen.KarmaHeaders{
			apigen.HeaderPrivateToken: "YOUR_GITLAB_TOKEN",
			apigen.HeaderContentType:  "application/json",
			apigen.HeaderAccept:       "application/json",
		},
		Responses: []apigen.Response{
			{
				StatusCode:  200,
				Description: "Issue details",
				ContentType: "application/json",
				Example:     []byte(`{"id": 42, "name": "Bug report", "description": "App crashes on startup", "labels": ["bug", "critical"], "created_at": "2025-03-11T10:00:00Z"}`),
			},
			{
				StatusCode:  404,
				Description: "Issue not found",
				Example:     []byte(`{"message": "404 Issue Not Found"}`),
			},
		},
	})

	// Alternative URL pattern syntax example
	api.AddEndpoint(apigen.Endpoint{
		Path:        "/groups/{group_id}/issues",
		Method:      "GET",
		Summary:     "List group issues",
		Description: "Get a list of issues for a specific group",
		// Path parameters will be automatically detected
		QueryParams: []apigen.Parameter{
			{
				Name:        "state",
				Type:        "string",
				Required:    false,
				Description: "Return issues with the specified state",
				Example:     "all",
			},
		},
		Headers: apigen.KarmaHeaders{
			apigen.HeaderPrivateToken: "YOUR_GITLAB_TOKEN",
			apigen.HeaderContentType:  "application/json",
			apigen.HeaderAccept:       "application/json",
		},
		Responses: []apigen.Response{
			{
				StatusCode:  200,
				Description: "List of issues",
				ContentType: "application/json",
				Example:     []byte(`[{"id": 1, "name": "Bug report"}, {"id": 2, "name": "Feature request"}]`),
			},
		},
	})

	// Export to all formats
	if err := api.ExportAll(); err != nil {
		log.Fatalf("Error exporting API definition: %v", err)
	}

	fmt.Println("API definition exported successfully to ./output directory")
	fmt.Println("Files generated:")
	fmt.Println("- gitlab_issues.json (Raw API definition)")
	fmt.Println("- gitlab_issues_postman.json (Postman collection)")
	fmt.Println("- gitlab_issues_openapi.json (OpenAPI specification)")
	fmt.Println("- gitlab_issues_docs.md (Markdown documentation)")
}
