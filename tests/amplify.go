package tests

import (
	"context"
	"fmt"

	"github.com/MelloB1989/karma/apis/aws/amplify"
)

func TestAmplifyFuncs() {

	client, _ := amplify.NewAmplifyClient(&amplify.ClientConfig{Region: "ap-south-1"})
	amplifyClient := &amplify.AmplifyClient{
		Client: client.Client,
		Ctx:    context.Background(),
	}

	// app, err := amplifyClient.CreateProject(amplify.ProjectConfig{
	// 	Name:     "Test",
	// 	Platform: "WEB",
	// })
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }
	//
	apps, err := amplifyClient.ListProjects()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, app := range apps {
		fmt.Println(*app.AppId, *app.CustomHeaders, app.Platform.Values())
	}

	deployment, err := client.CreateDeployment(amplify.DeploymentConfig{
		AppID:      "dpsvq4nwkea6o",
		BranchName: "main",
		JobType:    "RELEASE",
	})
}
