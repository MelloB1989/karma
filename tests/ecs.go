package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/apis/aws/ecs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestECSIntegration() {
	// Create unified configuration
	ecsConfig := &ecs.ECSConfig{
		ClusterName:     "karma",
		ServiceName:     "my-service",
		Family:          "my-task-family",
		DesiredCount:    1,
		CPU:             "256",
		Memory:          "512",
		NetworkMode:     types.NetworkModeAwsvpc,
		LaunchType:      types.LaunchTypeFargate,
		Architecture:    types.CPUArchitectureX8664,
		AssignPublicIP:  types.AssignPublicIpEnabled,
		Subnets:         []string{"subnet-01b09fd5796041dd0", "subnet-08c71e697fa058c17", "subnet-0b49551d797d3fe5e"},
		SecurityGroups:  []string{"sg-052b52a837f190658"},
		OperatingSystem: types.OSFamilyLinux,
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Image: aws.String("registry.coffeecodes.in/wedzing/backend:latest"),
				Name:  aws.String("my-container"),
				PortMappings: []types.PortMapping{
					{
						ContainerPort: aws.Int32(9000),
						HostPort:      aws.Int32(9000),
						AppProtocol:   "http",
						Protocol:      "tcp",
						Name:          aws.String("endpoint"),
					},
				},
				RepositoryCredentials: &types.RepositoryCredentials{
					CredentialsParameter: aws.String("arn:aws:secretsmanager:ap-south-1:022499029734:secret:CC-Registry-hwDOIs"),
				},
				Essential: aws.Bool(true),
			},
		},
	}

	// Create client with configuration
	client, _ := ecs.NewECSClient(&ecs.ClientConfig{
		Region: "ap-south-1",
	}, ecsConfig)

	// Create task definition
	taskDef, _ := client.RegisterTaskDefinition()

	fmt.Println(taskDef)

	// Create service using the task definition ARN
	service, _ := client.CreateService(*taskDef.TaskDefinition.TaskDefinitionArn)
	fmt.Println(service)

	// Update service later
	client.UpdateService(*taskDef.TaskDefinition.TaskDefinitionArn)

	// Force new deployment
	client.ForceNewDeployment()
}
