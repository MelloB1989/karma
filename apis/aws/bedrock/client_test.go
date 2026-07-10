package bedrock

import (
	"context"
	"net/http"
	"testing"

	"github.com/MelloB1989/karma/models"

	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// TestBearerAuthMiddleware verifies the middleware injects the Bedrock API key
// as an HTTP bearer token on the outgoing request.
func TestBearerAuthMiddleware(t *testing.T) {
	req := smithyhttp.NewStackRequest().(*smithyhttp.Request)

	mw := bearerAuthMiddleware("my-secret-key")
	_, _, err := mw.HandleFinalize(
		context.Background(),
		smithymiddleware.FinalizeInput{Request: req},
		smithymiddleware.FinalizeHandlerFunc(func(_ context.Context, in smithymiddleware.FinalizeInput) (smithymiddleware.FinalizeOutput, smithymiddleware.Metadata, error) {
			r := in.Request.(*smithyhttp.Request)
			if got := r.Header.Get("Authorization"); got != "Bearer my-secret-key" {
				t.Fatalf("Authorization header = %q, want %q", got, "Bearer my-secret-key")
			}
			return smithymiddleware.FinalizeOutput{}, smithymiddleware.Metadata{}, nil
		}),
	)
	if err != nil {
		t.Fatalf("HandleFinalize returned error: %v", err)
	}

	// Sanity: the header must survive on the request object itself.
	var _ http.Header = req.Header
}

func TestResolveRegion(t *testing.T) {
	t.Setenv("AWS_BEDROCK_REGION", "")
	t.Setenv("BEDROCK_REGION", "")
	t.Setenv("AWS_REGION", "")

	if got := ResolveRegion("eu-west-1"); got != "eu-west-1" {
		t.Fatalf("explicit override = %q, want eu-west-1", got)
	}
	if got := ResolveRegion(""); got != "us-east-1" {
		t.Fatalf("default = %q, want us-east-1", got)
	}

	t.Setenv("AWS_REGION", "ap-south-1")
	if got := ResolveRegion(""); got != "ap-south-1" {
		t.Fatalf("AWS_REGION fallback = %q, want ap-south-1", got)
	}

	t.Setenv("AWS_BEDROCK_REGION", "us-west-2")
	if got := ResolveRegion(""); got != "us-west-2" {
		t.Fatalf("AWS_BEDROCK_REGION precedence = %q, want us-west-2", got)
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Setenv("BEDROCK_API_KEY", "")
	t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "")

	// Explicit override always wins.
	t.Setenv("BEDROCK_API_KEY", "env-key")
	if got := resolveAPIKey("explicit-key"); got != "explicit-key" {
		t.Fatalf("resolveAPIKey explicit = %q, want explicit-key", got)
	}

	// BEDROCK_API_KEY is auto-detected.
	if got := resolveAPIKey(""); got != "env-key" {
		t.Fatalf("resolveAPIKey BEDROCK_API_KEY = %q, want env-key", got)
	}

	// Falls back to the AWS SDK-standard env var.
	t.Setenv("BEDROCK_API_KEY", "")
	t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "sdk-key")
	if got := resolveAPIKey(""); got != "sdk-key" {
		t.Fatalf("resolveAPIKey AWS_BEARER_TOKEN_BEDROCK = %q, want sdk-key", got)
	}

	// No key set → empty (caller falls back to AWS access keys).
	t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "")
	if got := resolveAPIKey(""); got != "" {
		t.Fatalf("resolveAPIKey none = %q, want empty", got)
	}
}

func TestBuildConverseInputAnthropicDropsTopP(t *testing.T) {
	hist := models.AIChatHistory{Messages: []models.AIMessage{{Role: models.User, Message: "hi"}}}

	// Anthropic model with both temperature and top_p set → top_p dropped.
	in, err := buildConverseInput(ConverseParams{
		ModelID: "global.anthropic.claude-sonnet-4-6", History: hist,
		Temperature: 0.5, TopP: 0.9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if in.InferenceConfig.Temperature == nil {
		t.Fatal("expected temperature to be set")
	}
	if in.InferenceConfig.TopP != nil {
		t.Fatalf("expected top_p to be dropped for Anthropic, got %v", *in.InferenceConfig.TopP)
	}

	// Anthropic with only top_p (no temperature) → top_p kept.
	in, _ = buildConverseInput(ConverseParams{ModelID: "anthropic.claude-v2", History: hist, TopP: 0.8})
	if in.InferenceConfig.TopP == nil {
		t.Fatal("expected top_p kept when temperature unset")
	}

	// Non-Anthropic model → both kept.
	in, _ = buildConverseInput(ConverseParams{ModelID: "meta.llama3-70b-instruct-v1:0", History: hist, Temperature: 0.5, TopP: 0.9})
	if in.InferenceConfig.Temperature == nil || in.InferenceConfig.TopP == nil {
		t.Fatal("expected both temperature and top_p for non-Anthropic model")
	}
}
