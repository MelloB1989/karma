package amplify

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

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

type ManualDeploymentType string

const (
	S3Deploy     ManualDeploymentType = "S3"
	ZipDeploy    ManualDeploymentType = "ZIP"
	URLDeploy    ManualDeploymentType = "URL"
)

// ManualDeploymentConfig holds configuration for manual deployments
type ManualDeploymentConfig struct {
	AppName     string
	BranchName  string
	DeployType  ManualDeploymentType
	ZipPath     string   // Path to zip file for ZipDeploy
	S3URL       string   // S3 URL for S3Deploy
	PublicURL   string   // Public URL for URLDeploy
	SourceDir   string   // Directory to zip for ZipDeploy
	IgnorePaths []string // Paths to ignore when creating zip
}

// FileInfo represents a file to be deployed
type FileInfo struct {
	Path     string
	MD5Hash  string
	Content  []byte
}

// CreateManualDeployment creates a new manual deployment without Git
func (a *AmplifyClient) CreateManualDeployment(cfg ManualDeploymentConfig) (*types.App, error) {
	// Create app if it doesn't exist
	app, err := a.createManualApp(cfg.AppName, cfg.BranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to create manual app: %w", err)
	}

	// Handle different deployment types
	switch cfg.DeployType {
	case ZipDeploy:
		err = a.handleZipDeployment(app.AppId, cfg)
	case S3Deploy:
		err = fmt.Errorf("S3 deployment not implemented yet")
	case URLDeploy:
		err = fmt.Errorf("URL deployment not implemented yet")
	default:
		err = fmt.Errorf("invalid deployment type")
	}

	if err != nil {
		return nil, err
	}

	return app, nil
}

// createManualApp creates a new Amplify app for manual deployment
func (a *AmplifyClient) createManualApp(appName, branchName string) (*types.App, error) {
	input := &amplify.CreateAppInput{
		Name:       &appName,
		Platform:   types.PlatformWEB,
	}

	result, err := a.client.CreateApp(a.ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	// Create branch
	_, err = a.client.CreateBranch(a.ctx, &amplify.CreateBranchInput{
		AppId:      result.App.AppId,
		BranchName: &branchName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	return result.App, nil
}

// handleZipDeployment handles deployment from a zip file or directory
func (a *AmplifyClient) handleZipDeployment(appID *string, cfg ManualDeploymentConfig) error {
	var files []FileInfo
	var err error

	if cfg.ZipPath != "" {
		files, err = a.processExistingZip(cfg.ZipPath)
	} else if cfg.SourceDir != "" {
		files, err = a.createZipFromDirectory(cfg.SourceDir, cfg.IgnorePaths)
	} else {
		return fmt.Errorf("either ZipPath or SourceDir must be provided")
	}

	if err != nil {
		return err
	}

	// Create file map for deployment
	fileMap := make(map[string]string)
	for _, file := range files {
		fileMap[file.Path] = file.MD5Hash
	}

	// Start deployment
	input := &amplify.CreateDeploymentInput{
		AppId:      appID,
		BranchName: &cfg.BranchName,
		FileMap:    fileMap,
	}

	result, err := a.client.CreateDeployment(a.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Upload files using the provided URLs
	jobId := result.JobId
	for filePath, uploadURL := range result.FileUploadUrls {
		var fileContent []byte
		for _, file := range files {
			if file.Path == filePath {
				fileContent = file.Content
				break
			}
		}

		err = a.uploadFile(uploadURL, fileContent)
		if err != nil {
			return fmt.Errorf("failed to upload file %s: %w", filePath, err)
		}
	}

	// Start the job
	_, err = a.client.StartJob(a.ctx, &amplify.StartJobInput{
		AppId:      appID,
		BranchName: &cfg.BranchName,
		JobId:      jobId,
	})

	return err
}

// processExistingZip processes an existing zip file
func (a *AmplifyClient) processExistingZip(zipPath string) ([]FileInfo, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zipReader.Close()

	var files []FileInfo
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file in zip: %w", err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file content: %w", err)
		}

		hash := md5.Sum(content)
		files = append(files, FileInfo{
			Path:    file.Name,
			MD5Hash: hex.EncodeToString(hash[:]),
			Content: content,
		})
	}

	return files, nil
}

// createZipFromDirectory creates a zip file from a directory
func (a *AmplifyClient) createZipFromDirectory(sourceDir string, ignorePaths []string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and ignored paths
		if info.IsDir() {
			return nil
		}
		for _, ignorePath := range ignorePaths {
			if matched, _ := filepath.Match(ignorePath, path); matched {
				return nil
			}
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Calculate relative path and MD5 hash
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		hash := md5.Sum(content)
		files = append(files, FileInfo{
			Path:    relPath,
			MD5Hash: hex.EncodeToString(hash[:]),
			Content: content,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process directory: %w", err)
	}

	return files, nil
}

// uploadFile uploads a file to the provided URL
func (a *AmplifyClient) uploadFile(url string, content []byte) error {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status: %s", resp.Status)
	}

	return nil
}
