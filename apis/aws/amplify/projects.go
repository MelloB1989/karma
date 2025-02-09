package amplify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/aws/aws-sdk-go-v2/service/amplify/types"
	"github.com/aws/aws-sdk-go/aws"
)

type RepositoryProvider string

const (
	GitHub        RepositoryProvider = "github"
	GitLab        RepositoryProvider = "gitlab"
	BitBucket     RepositoryProvider = "bitbucket"
	SelfHostedGit RepositoryProvider = "self-hosted"
)

// ProjectConfig holds configuration for creating a new Amplify project
type ProjectConfig struct {
	Name        string
	Repository  string
	AccessToken string
	Platform    string
	Framework   string
	// Optional fields for self-hosted repositories
	CustomBaseURL string // For self-hosted GitLab/GitHub Enterprise
}

// AmplifyClient wraps the AWS Amplify client with additional functionality
type AmplifyClient struct {
	Client *amplify.Client
	Ctx    context.Context
}

// ClientConfig holds configuration options for the Amplify client
type ClientConfig struct {
	Region string
}

// NewAmplifyClient creates a new instance of AmplifyClient
func NewAmplifyClient(cfg *ClientConfig) (*AmplifyClient, error) {
	ctx := context.Background()

	// Load AWS configuration
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Override region if specified
	if cfg != nil && cfg.Region != "" {
		sdkConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to load AWS config with custom region: %w", err)
		}
	}

	// Create Amplify client
	client := amplify.NewFromConfig(sdkConfig)

	return &AmplifyClient{
		Client: client,
		Ctx:    ctx,
	}, nil
}

func (a *AmplifyClient) CreateProject(cfg ProjectConfig) (*types.App, error) {
	// Validate repository URL and determine provider
	repoURL, provider, err := validateRepository(cfg.Repository, cfg.CustomBaseURL)
	if err != nil {
		return nil, err
	}

	// Base input for creating app
	input := &amplify.CreateAppInput{
		Name:       aws.String(cfg.Name),
		Repository: aws.String(repoURL),
		OauthToken: aws.String(cfg.AccessToken),
	}

	// Add platform if specified
	if cfg.Platform != "" {
		input.Platform = types.Platform(cfg.Platform)
	}

	// Handle custom repository configurations
	if provider == "SelfHostedGit" {
		// For self-hosted GitLab, we need to ensure the custom base URL is set
		if cfg.CustomBaseURL != "" {
			// Add any provider-specific configurations here
			// Note: AWS Amplify supports custom repository providers through additional configuration
			headers := map[string]string{
				"Custom-Base-URL": cfg.CustomBaseURL,
			}
			headersJSON, err := json.Marshal(headers)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal custom headers: %w", err)
			}
			headersStr := string(headersJSON)
			input.CustomHeaders = &headersStr
		}
	}

	result, err := a.Client.CreateApp(a.Ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create Amplify project: %w", err)
	}

	return result.App, nil
}

// GetProject retrieves details of an Amplify project
func (a *AmplifyClient) GetProject(appID string) (*types.App, error) {
	input := &amplify.GetAppInput{
		AppId: &appID,
	}

	result, err := a.Client.GetApp(a.Ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get Amplify project details: %w", err)
	}

	return result.App, nil
}

// ListProjects retrieves all Amplify projects
func (a *AmplifyClient) ListProjects() ([]types.App, error) {
	input := &amplify.ListAppsInput{}

	result, err := a.Client.ListApps(a.Ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list Amplify projects: %w", err)
	}

	return result.Apps, nil
}

// DeploymentConfig holds configuration for creating a deployment
type DeploymentConfig struct {
	AppID      string
	BranchName string
	JobType    string
}

// CreateDeployment creates a new deployment for a branch
func (a *AmplifyClient) CreateDeployment(cfg DeploymentConfig) (*types.JobSummary, error) {
	input := &amplify.StartJobInput{
		AppId:      &cfg.AppID,
		BranchName: &cfg.BranchName,
		JobType:    types.JobType(cfg.JobType),
	}

	result, err := a.Client.StartJob(a.Ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	return result.JobSummary, nil
}

// ListDeployments retrieves all deployments for a project
func (a *AmplifyClient) ListDeployments(appID, branchName string) ([]types.JobSummary, error) {
	input := &amplify.ListJobsInput{
		AppId:      &appID,
		BranchName: &branchName,
	}

	result, err := a.Client.ListJobs(a.Ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	return result.JobSummaries, nil
}

// GetDeployment retrieves details of a specific deployment
func (a *AmplifyClient) GetDeployment(appID, branchName, jobID string) (*types.Job, error) {
	input := &amplify.GetJobInput{
		AppId:      &appID,
		BranchName: &branchName,
		JobId:      &jobID,
	}

	result, err := a.Client.GetJob(a.Ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment details: %w", err)
	}

	return result.Job, nil
}

// DeleteProject deletes an Amplify project
func (a *AmplifyClient) DeleteProject(appID string) error {
	input := &amplify.DeleteAppInput{
		AppId: &appID,
	}

	_, err := a.Client.DeleteApp(a.Ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete Amplify project: %w", err)
	}

	return nil
}

// validateRepository checks if the repository URL is valid and determines the provider
func validateRepository(repository, customBaseURL string) (string, RepositoryProvider, error) {
	repoURL, err := url.Parse(repository)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Handle self-hosted GitLab/GitHub Enterprise
	if customBaseURL != "" {
		customURL, err := url.Parse(customBaseURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid custom base URL: %w", err)
		}
		if repoURL.Host == customURL.Host {
			return repository, SelfHostedGit, nil
		}
	}

	// Determine provider based on hostname
	switch {
	case repoURL.Host == "github.com":
		return repository, GitHub, nil
	case repoURL.Host == "gitlab.com":
		return repository, GitLab, nil
	case repoURL.Host == "bitbucket.org":
		return repository, BitBucket, nil
	default:
		return repository, SelfHostedGit, nil
	}
}
