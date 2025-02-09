package tests

import (
	"context"
	"fmt"

	"github.com/MelloB1989/karma/apis/aws/amplify"
)

func TestAmplifyFuncs() {
	cfg := amplify.ProjectConfig{
		Name:          "MyAmplifyApp",
		Repository:    "https://example.com/repo.git",
		AccessToken:   "my-access-token",
		Platform:      "WEB",
		CustomBaseURL: "https://example.com",
	}

	client, _ := amplify.NewAmplifyClient(&amplify.ClientConfig{Region: "ap-south-1"})
	amplifyClient := &amplify.AmplifyClient{
		Client: client.Client,
		Ctx:    context.Background(),
	}

	app, err := amplifyClient.CreateProject(cfg)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Created app:", app)
}
