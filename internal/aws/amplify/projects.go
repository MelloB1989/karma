package amplify

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/aws/aws-sdk-go-v2/service/amplify/types"
)

// AmplifyClient wraps the AWS Amplify client with additional functionality
type AmplifyClient struct {
	client *amplify.Client
	ctx    context.Context
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
		client: client,
		ctx:    ctx,
	}, nil
}

// ProjectConfig holds configuration for creating a new Amplify project
type ProjectConfig struct {
	Name        string
	Repository  string
	AccessToken string
	Platform    string
	Framework   string
}

// CreateProject creates a new Amplify project
func (a *AmplifyClient) CreateProject(cfg ProjectConfig) (*types.App, error) {
	input := &amplify.CreateAppInput{
		Name: &cfg.Name,
		Repository: &cfg.Repository,
		OauthToken: &cfg.AccessToken,
	}

	if cfg.Platform != "" {
		input.Platform = types.Platform(cfg.Platform)
	}

	if cfg.Framework != "" {
		input.Framework = types.Framework(cfg.Framework)
	}

	result, err := a.client.CreateApp(a.ctx, input)
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

	result, err := a.client.GetApp(a.ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get Amplify project details: %w", err)
	}

	return result.App, nil
}

// ListProjects retrieves all Amplify projects
func (a *AmplifyClient) ListProjects() ([]types.App, error) {
	input := &amplify.ListAppsInput{}

	result, err := a.client.ListApps(a.ctx, input)
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

	result, err := a.client.StartJob(a.ctx, input)
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

	result, err := a.client.ListJobs(a.ctx, input)
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

	result, err := a.client.GetJob(a.ctx, input)
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

	_, err := a.client.DeleteApp(a.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete Amplify project: %w", err)
	}

	return nil
}
