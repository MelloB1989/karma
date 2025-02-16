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

// ECSConfig holds all configuration for an ECS deployment
type ECSConfig struct {
	// Cluster and Service details
	ClusterName  string
	ServiceName  string
	DesiredCount int32

	// Task Definition details
	Family          string
	CPU             string
	Memory          string
	TaskRole        string
	ExecutionRole   string
	Architecture    types.CPUArchitecture
	OperatingSystem types.OSFamily

	// Container configurations
	ContainerDefinitions []types.ContainerDefinition

	// Network configuration
	NetworkMode    types.NetworkMode
	AssignPublicIP types.AssignPublicIp
	Subnets        []string
	SecurityGroups []string

	// Launch configuration
	LaunchType types.LaunchType

	// Load balancing
	LoadBalancers     []types.LoadBalancer
	ServiceRegistries []types.ServiceRegistry

	// Additional settings
	EnableECSManagedTags bool
	EnableExecuteCommand bool
	Tags                 []types.Tag
}

// ECSClient wraps the AWS ECS client with additional functionality
type ECSClient struct {
	Client *ecs.Client
	Ctx    context.Context
	Config *ECSConfig
}

// NewECSClient creates a new ECS client with the given configuration
func NewECSClient(clientCfg *ClientConfig, ecsCfg *ECSConfig) (*ECSClient, error) {
	ctx := context.Background()

	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	if clientCfg != nil && clientCfg.Region != "" {
		sdkConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(clientCfg.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to load AWS config with custom region: %w", err)
		}
	}

	client := ecs.NewFromConfig(sdkConfig)
	return &ECSClient{
		Client: client,
		Ctx:    ctx,
		Config: ecsCfg,
	}, nil
}

// RegisterTaskDefinition creates or updates a task definition using the client config
func (c *ECSClient) RegisterTaskDefinition() (*ecs.RegisterTaskDefinitionOutput, error) {
	input := &ecs.RegisterTaskDefinitionInput{
		Family:                  &c.Config.Family,
		ContainerDefinitions:    c.Config.ContainerDefinitions,
		Cpu:                     &c.Config.CPU,
		Memory:                  &c.Config.Memory,
		NetworkMode:             c.Config.NetworkMode,
		TaskRoleArn:             &c.Config.TaskRole,
		ExecutionRoleArn:        &c.Config.ExecutionRole,
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityFargate},
		Tags:                    c.Config.Tags,
		RuntimePlatform: &types.RuntimePlatform{
			CpuArchitecture:       c.Config.Architecture,
			OperatingSystemFamily: c.Config.OperatingSystem,
		},
	}

	return c.Client.RegisterTaskDefinition(c.Ctx, input)
}

// CreateService creates a new ECS service using the client config
func (c *ECSClient) CreateService(taskDefinitionARN string) (*ecs.CreateServiceOutput, error) {
	input := &ecs.CreateServiceInput{
		Cluster:              &c.Config.ClusterName,
		ServiceName:          &c.Config.ServiceName,
		TaskDefinition:       &taskDefinitionARN,
		DesiredCount:         &c.Config.DesiredCount,
		LaunchType:           c.Config.LaunchType,
		LoadBalancers:        c.Config.LoadBalancers,
		ServiceRegistries:    c.Config.ServiceRegistries,
		EnableECSManagedTags: c.Config.EnableECSManagedTags,
		EnableExecuteCommand: c.Config.EnableExecuteCommand,
		Tags:                 c.Config.Tags,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: c.Config.AssignPublicIP,
				Subnets:        c.Config.Subnets,
				SecurityGroups: c.Config.SecurityGroups,
			},
		},
	}

	return c.Client.CreateService(c.Ctx, input)
}

// UpdateService updates an existing ECS service using the client config
func (c *ECSClient) UpdateService(taskDefinitionARN string) (*ecs.UpdateServiceOutput, error) {
	input := &ecs.UpdateServiceInput{
		Cluster:        &c.Config.ClusterName,
		Service:        &c.Config.ServiceName,
		TaskDefinition: &taskDefinitionARN,
		DesiredCount:   &c.Config.DesiredCount,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: c.Config.AssignPublicIP,
				Subnets:        c.Config.Subnets,
				SecurityGroups: c.Config.SecurityGroups,
			},
		},
	}

	return c.Client.UpdateService(c.Ctx, input)
}

// ForceNewDeployment forces a new deployment of the service using the client config
func (c *ECSClient) ForceNewDeployment() (*ecs.UpdateServiceOutput, error) {
	input := &ecs.UpdateServiceInput{
		Cluster:            &c.Config.ClusterName,
		Service:            &c.Config.ServiceName,
		ForceNewDeployment: true,
	}

	return c.Client.UpdateService(c.Ctx, input)
}
