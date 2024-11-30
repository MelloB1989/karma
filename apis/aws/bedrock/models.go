package bedrock

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	b "github.com/aws/aws-sdk-go-v2/service/bedrock"
)

type Models struct {
	LLAMA3_8B  string
	LLAMA3_70B string
}

func OurModels() *Models {
	return &Models{
		LLAMA3_8B:  "meta.llama3-8b-instruct-v1:0",
		LLAMA3_70B: "meta.llama3-70b-instruct-v1:0",
	}
}

func GetModels() []string {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("Couldn't load default configuration. Have you set up your AWS account?")
		log.Println(err)
	}
	bedrockClient := b.NewFromConfig(sdkConfig)

	models, err := bedrockClient.ListFoundationModels(context.TODO(), &b.ListFoundationModelsInput{})
	if err != nil {
		log.Println("Error listing models")
	}
	var model_names []string
	for _, model := range models.ModelSummaries {
		model_names = append(model_names, *model.ModelId)
	}
	// log.Println(model_names)
	return model_names
}
