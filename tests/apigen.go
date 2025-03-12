package tests

import (
	"log"
	"time"

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

type UsersB struct {
	TableName    struct{}          `karma_table:"users"`
	Id           string            `json:"id" karma:"primary"`
	Username     string            `json:"username"`
	Email        string            `json:"email"`
	Name         string            `json:"name"`
	Phone        string            `json:"phone"`
	Bio          string            `json:"bio"`
	ProfileImage string            `json:"profile_image"`
	Socials      map[string]string `json:"socials" db:"socials"`
	DateOfBirth  time.Time         `json:"date_of_birth"`
	Gender       string            `json:"gender"`
	CreatedAt    time.Time         `json:"created_at"`
	DeviceId     string            `json:"device_id"`
	PasswordHash string            `json:"password_hash"`
}

type LoginBody struct {
	Phone string `json:"phone" description:"Phone number of the user"`
}

type LoginSuccessData struct {
	AccountExists bool `json:"account_exists" description:"If Account exists"`
	TestPhone     bool `json:"test_phone" description:"Is it a Test phone"`
}

type LoginSuccess struct {
	Success bool             `json:"success" description:"Success status"`
	Message string           `json:"message" description:"Message"`
	Data    LoginSuccessData `json:"data" description:"Data"`
}

type VerifyOTPBody struct {
	Phone string `json:"phone" description:"Phone number of the user"`
	OTP   string `json:"otp" description:"OTP sent to the user"`
}

type VerifySuccessData struct {
	AccountExists bool   `json:"account_exists" description:"If Account exists"`
	Token         string `json:"token" description:"JWT Token"`
}

type VerifyOTPSuccess struct {
	Success bool              `json:"success" description:"Success status"`
	Message string            `json:"message" description:"Message"`
	Data    VerifySuccessData `json:"data" description:"Data"`
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

	auth := apigen.NewAPIDefinition("Auth APIs", "Authentication apis for users", []string{"http://localhost:9000"}, "./docstest/auth", "auth")
	auth.AddGlobalVariable("api_version", "v1")

	invalidBodyResponse := apigen.Response{
		StatusCode:  400,
		Description: "Invalid request body",
		ContentType: apigen.ContentTypeJSON,
		Example:     []byte(`{"success": false, "message": "Failed to parse request body.", "data": null}`),
	}

	invalidPhoneResponse := apigen.Response{
		StatusCode:  400,
		Description: "Invalid phone number",
		ContentType: apigen.ContentTypeJSON,
		Example:     []byte(`{"success": false, "message": "Invalid phone number", "data": null}`),
	}

	configurationErrorResponse := apigen.Response{
		StatusCode:  500,
		Description: "Configuration error. This is a server error.",
		ContentType: apigen.ContentTypeJSON,
		Example:     []byte(`{"success": false, "message": "Configuration error.", "data": null}`),
	}

	loginBody, _ := apigen.RequestBodyFromStruct(LoginBody{}, apigen.ContentTypeJSON, true, []apigen.FieldOverride{})
	successLoginResponse, _ := apigen.ResponseFromStruct(200, "OTP sent to the phone", LoginSuccess{}, apigen.ContentTypeJSON, []apigen.FieldOverride{})

	auth.AddEndpoint(apigen.Endpoint{
		Path:        "/auth/login",
		Method:      "POST",
		Summary:     "Login with phone number",
		Description: "This endpoint is used to login with phone number.",
		RequestBody: loginBody,
		Responses: []apigen.Response{
			*successLoginResponse,
			invalidBodyResponse,
			invalidPhoneResponse,
			configurationErrorResponse,
			{
				StatusCode:  500,
				Description: "Failed to send OTP. This is a server error.",
				ContentType: apigen.ContentTypeJSON,
				Example:     []byte(`{"success": false, "message": "Failed to send OTP.", "data": null}`),
			},
		},
	})

	verifyOTPBody, _ := apigen.RequestBodyFromStruct(VerifyOTPBody{}, apigen.ContentTypeJSON, true, []apigen.FieldOverride{})
	verifySuccessResponse, _ := apigen.ResponseFromStruct(200, "OTP verified successfully", VerifyOTPSuccess{}, apigen.ContentTypeJSON, []apigen.FieldOverride{
		{
			Name:        "Data",
			Description: "Data containing account information and authentication token",
			Example: map[string]interface{}{
				"account_exists": true,
				"token":          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkb2IiOiIxOTkwLTAxLTAxVDAwOjAwOjAwWiIsImVtYWlsIjoiam9obi5kb2VAZXhhbXBsZS5jb20iLCJleHAiOjE3NDQyMzI1NTYsImdlbmRlciI6Im1hbGUiLCJwaG9uZSI6Iis5MTk4MTI5NDA3MDYiLCJ1aWQiOiIzeDIyXzJkIn0.fVRaF88T3xVXHX3-i4EY3utSqcSIxlfc45EVCr8byNM",
				"test_phone":     false,
			},
		},
	})

	auth.AddEndpoint(apigen.Endpoint{
		Path:        "/auth/verify_otp",
		Method:      "POST",
		Summary:     "Verify OTP",
		Description: "This endpoint is used to verify OTP sent to the phone number.",
		RequestBody: verifyOTPBody,
		Responses: []apigen.Response{
			*verifySuccessResponse,
			invalidBodyResponse,
			invalidPhoneResponse,
			configurationErrorResponse,
			{
				StatusCode:  500,
				Description: "Failed to send OTP. This is a server error.",
				ContentType: apigen.ContentTypeJSON,
				Example:     []byte(`{"success": false, "message": "Failed to send OTP.", "data": null}`),
			},
			{
				StatusCode:  500,
				Description: "Failed to create JWT token. This is a server error.",
				ContentType: apigen.ContentTypeJSON,
				Example:     []byte(`{"success": false, "message": "Failed to create JWT token.", "data": null}`),
			},
			{
				StatusCode:  400,
				Description: "Invalid OTP. Please check the OTP and try again.",
				ContentType: apigen.ContentTypeJSON,
				Example:     []byte(`{"success": false, "message": "Invalid OTP.", "data": null}`),
			},
		},
	})

	registerReqBody, err := apigen.RequestBodyFromStruct(
		UsersB{},
		apigen.ContentTypeJSON,
		true, []apigen.FieldOverride{
			{
				Name:    "TableName",
				Exclude: true,
			},
			{
				Name:    "Id",
				Exclude: true,
			},
			{
				Name:    "CreatedAt",
				Exclude: true,
			},
			{
				Name:    "DeviceId",
				Exclude: true,
			},
			{
				Name:    "Phone",
				Exclude: true,
			},
		})

	if err != nil {
		println(err)
	}

	auth.AddEndpoint(apigen.Endpoint{
		Path:        "/auth/register",
		Method:      "POST",
		Summary:     "Register user",
		Description: "This endpoint is used to register a new user.",
		RequestBody: registerReqBody,
		Responses: []apigen.Response{
			invalidBodyResponse,
			{
				StatusCode:  500,
				Description: "Failed to register. This is a server error.",
				ContentType: apigen.ContentTypeJSON,
				Example:     []byte(`{"success": false, "message": "Failed to register user.", "data": null}`),
			},
			{
				StatusCode:  201,
				Description: "User registered successfully.",
				ContentType: apigen.ContentTypeJSON,
				Example:     []byte(`{"success": false, "message": "User registered successfully.", "data": {}}`),
			},
		},
	})

	// Generate API documentation
	err = auth.ExportAll()
	if err != nil {
		log.Fatalf("Error exporting API documentation: %v", err)
	}

	// err = api.ExportAll()
	// if err != nil {
	// 	log.Fatalf("Error exporting API documentation: %v", err)
	// }
}
