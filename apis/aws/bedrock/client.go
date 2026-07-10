package bedrock

import (
	"context"
	"fmt"
	"os"
	"strings"

	c "github.com/MelloB1989/karma/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// bedrockAPIKeyEnvs are the environment variables consulted (in order) for an
// Amazon Bedrock API key (bearer token). Both short-term and long-term Bedrock
// API keys are passed the same way — as a bearer token in the HTTP
// Authorization header — so no distinction is needed here.
// BEDROCK_API_KEY is the karma-preferred name; AWS_BEARER_TOKEN_BEDROCK is the
// name the official AWS SDKs recognize. See:
// https://docs.aws.amazon.com/bedrock/latest/userguide/api-keys-use.html
var bedrockAPIKeyEnvs = []string{"BEDROCK_API_KEY", "AWS_BEARER_TOKEN_BEDROCK"}

// ClientOptions controls how the Bedrock runtime client authenticates and which
// region it targets. All fields are optional; zero values fall back to
// environment / shared-config defaults.
type ClientOptions struct {
	// Region overrides the resolved AWS region.
	Region string
	// APIKey is an Amazon Bedrock API key (bearer token). When set, the client
	// authenticates with the token instead of SigV4-signed AWS credentials. If
	// empty, the AWS_BEARER_TOKEN_BEDROCK environment variable is consulted.
	APIKey string
}

// ResolveRegion determines the Bedrock region, checking (in order):
// explicit override, AWS_BEDROCK_REGION, BEDROCK_REGION, AWS_REGION, then
// falling back to us-east-1.
func ResolveRegion(override string) string {
	if override != "" {
		return override
	}
	for _, key := range []string{"AWS_BEDROCK_REGION", "BEDROCK_REGION", "AWS_REGION"} {
		if v := strings.TrimSpace(c.GetEnvRaw(key)); v != "" {
			return v
		}
	}
	return "us-east-1"
}

// resolveAPIKey returns the Bedrock API key to use. An explicit value (e.g. from
// ai.WithBedrockAPIKey) always wins; otherwise the environment variables in
// bedrockAPIKeyEnvs are auto-detected in order. Returns "" when none is set, in
// which case the caller falls back to AWS access-key / default-chain auth.
func resolveAPIKey(explicit string) string {
	if v := strings.TrimSpace(explicit); v != "" {
		return v
	}
	for _, env := range bedrockAPIKeyEnvs {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	return ""
}

// NewRuntimeClient builds a Bedrock runtime client using the official AWS SDK
// v2. It supports three authentication modes, resolved in this order:
//
//  1. Bedrock API key (bearer token) — from opts.APIKey or AWS_BEARER_TOKEN_BEDROCK.
//  2. Static AWS credentials — from the karma config (AWS_ACCESS_KEY_ID / secret).
//  3. The default AWS credential chain (shared config, env, IMDS, etc.).
func NewRuntimeClient(ctx context.Context, opts ClientOptions) (*bedrockruntime.Client, error) {
	region := ResolveRegion(opts.Region)

	if apiKey := resolveAPIKey(opts.APIKey); apiKey != "" {
		return newBearerTokenClient(ctx, region, apiKey)
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}

	// Prefer explicit static credentials from the karma config when present so
	// callers that inject keys programmatically keep working.
	cfg := c.DefaultConfig()
	if cfg.AwsAccessKey != "" && cfg.AwsSecretKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AwsAccessKey, cfg.AwsSecretKey, ""),
		))
	}

	sdkConfig, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return bedrockruntime.NewFromConfig(sdkConfig), nil
}

// newBearerTokenClient builds a client that authenticates with an Amazon
// Bedrock API key. The SDK version pinned by this module predates native bearer
// support for Bedrock, so we replicate what newer SDKs do internally: skip SigV4
// signing (anonymous credentials) and inject an `Authorization: Bearer <token>`
// header on every request via a finalize-step middleware.
func newBearerTokenClient(ctx context.Context, region, apiKey string) (*bedrockruntime.Client, error) {
	sdkConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return bedrockruntime.NewFromConfig(sdkConfig, func(o *bedrockruntime.Options) {
		o.APIOptions = append(o.APIOptions, func(stack *smithymiddleware.Stack) error {
			return stack.Finalize.Add(bearerAuthMiddleware(apiKey), smithymiddleware.After)
		})
	}), nil
}

// bearerAuthMiddleware sets the Authorization header to the Bedrock API key.
// It is registered at the end of the Finalize step so it runs after (and is not
// overwritten by) the SDK's signing middleware.
func bearerAuthMiddleware(apiKey string) smithymiddleware.FinalizeMiddleware {
	return smithymiddleware.FinalizeMiddlewareFunc(
		"KarmaBedrockBearerAuth",
		func(ctx context.Context, in smithymiddleware.FinalizeInput, next smithymiddleware.FinalizeHandler) (smithymiddleware.FinalizeOutput, smithymiddleware.Metadata, error) {
			if req, ok := in.Request.(*smithyhttp.Request); ok {
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}
			return next.HandleFinalize(ctx, in)
		},
	)
}
