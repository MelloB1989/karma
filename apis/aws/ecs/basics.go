package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// ClientConfig holds configuration for the ECS client
type ClientConfig struct {
	Region string
}

// ECSClient wraps the AWS ECS client with additional functionality
type ECSClient struct {
	Client *ecs.Client
	Ctx    context.Context
}

// NewECSClient creates a new ECS client with the given configuration
func NewECSClient(cfg *ClientConfig) (*ECSClient, error) {
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

	// Create ECS client
	client := ecs.NewFromConfig(sdkConfig)
	return &ECSClient{
		Client: client,
		Ctx:    ctx,
	}, nil
}

// TaskDefinitionInput represents the input for creating/updating a task definition
type TaskDefinitionInput struct {
	Family               string
	ContainerDefinitions []types.ContainerDefinition
	CPU                  string
	Memory               string
	NetworkMode          string
	TaskRole             string
	ExecutionRole        string
	Tags                 []types.Tag
}

// ServiceInput represents the input for creating/updating a service
type ServiceInput struct {
	ClusterName          string
	ServiceName          string
	TaskDefinitionARN    string
	DesiredCount         int32
	LaunchType           types.LaunchType
	NetworkConfiguration *types.NetworkConfiguration
	LoadBalancers        []types.LoadBalancer
	ServiceRegistries    []types.ServiceRegistry
	EnableECSManagedTags bool
	EnableExecuteCommand bool
	Tags                 []types.Tag
}

// RegisterTaskDefinition creates or updates a task definition
func (c *ECSClient) RegisterTaskDefinition(input *TaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	return c.Client.RegisterTaskDefinition(c.Ctx, &ecs.RegisterTaskDefinitionInput{
		Family:                  &input.Family,
		ContainerDefinitions:    input.ContainerDefinitions,
		Cpu:                     &input.CPU,
		Memory:                  &input.Memory,
		NetworkMode:             types.NetworkMode(input.NetworkMode),
		TaskRoleArn:             &input.TaskRole,
		ExecutionRoleArn:        &input.ExecutionRole,
		Tags:                    input.Tags,
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
	})
}

// CreateService creates a new ECS service
func (c *ECSClient) CreateService(input *ServiceInput) (*ecs.CreateServiceOutput, error) {
	return c.Client.CreateService(c.Ctx, &ecs.CreateServiceInput{
		Cluster:              &input.ClusterName,
		ServiceName:          &input.ServiceName,
		TaskDefinition:       &input.TaskDefinitionARN,
		DesiredCount:         &input.DesiredCount,
		LaunchType:           input.LaunchType,
		NetworkConfiguration: input.NetworkConfiguration,
		LoadBalancers:        input.LoadBalancers,
		ServiceRegistries:    input.ServiceRegistries,
		EnableECSManagedTags: input.EnableECSManagedTags,
		EnableExecuteCommand: input.EnableExecuteCommand,
		Tags:                 input.Tags,
	})
}

// UpdateService updates an existing ECS service
func (c *ECSClient) UpdateService(input *ServiceInput) (*ecs.UpdateServiceOutput, error) {
	return c.Client.UpdateService(c.Ctx, &ecs.UpdateServiceInput{
		Cluster:              &input.ClusterName,
		Service:              &input.ServiceName,
		TaskDefinition:       &input.TaskDefinitionARN,
		DesiredCount:         &input.DesiredCount,
		NetworkConfiguration: input.NetworkConfiguration,
		ForceNewDeployment:   true,
	})
}

// ForceNewDeployment forces a new deployment of the service
func (c *ECSClient) ForceNewDeployment(clusterName, serviceName string) (*ecs.UpdateServiceOutput, error) {
	return c.Client.UpdateService(c.Ctx, &ecs.UpdateServiceInput{
		Cluster:            &clusterName,
		Service:            &serviceName,
		ForceNewDeployment: true,
	})
}
