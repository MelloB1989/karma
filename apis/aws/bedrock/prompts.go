package bedrock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	c "github.com/MelloB1989/karma/config"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-sdk-go/aws"
)

type Generation struct {
	Generation           string `json:"generation"`
	PromptTokenCount     int    `json:"prompt_token_count"`
	GenerationTokenCount int    `json:"generation_token_count"`
	StopReason           string `json:"stop_reason"`
}

func createClient() *bedrockruntime.Client {
	overrideRegion, _ := c.GetEnv("AWS_BEDROCK_REGION")
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if overrideRegion != "" {
		sdkConfig, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(overrideRegion),
		)
	}
	if err != nil {
		log.Println("Couldn't load default configuration. Have you set up your AWS account?")
		log.Println(err)
	}
	bedrockClient := bedrockruntime.NewFromConfig(sdkConfig)
	return bedrockClient
}

func PromptModel(prompt string, temp float32, top_p float32, max_len int, model string) string {
	bedrockClient := createClient()
	// Set the input values without using pointers directly
	input := &bedrockruntime.InvokeModelInput{
		Accept:      aws.String("application/json"),
		ModelId:     aws.String(model),
		ContentType: aws.String("application/json"),
	}

	body := map[string]interface{}{
		"prompt":      prompt,
		"temperature": temp,
		"top_p":       top_p,
		"max_gen_len": max_len,
	}
	bytes, err := json.Marshal(body)
	if err != nil {
		log.Println("Error converting interface to []byte:", err)
		return ""
	}
	input.Body = bytes

	result, err := bedrockClient.InvokeModel(context.TODO(), input)
	if err != nil {
		log.Println("Error invoking model:", err)
		return ""
	}

	var generated Generation
	err = json.Unmarshal(result.Body, &generated)
	if err != nil {
		log.Println("Error unmarshaling response:", err)
		return ""
	}
	log.Println(generated)
	return generated.Generation
}

func PromptModelStream(prompt string, temp float32, top_p float32, max_len int, model string) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	bedrockClient := createClient()
	input := &bedrockruntime.InvokeModelWithResponseStreamInput{
		Accept:      aws.String("application/json"),
		ModelId:     aws.String(model),
		ContentType: aws.String("application/json"),
	}

	body := map[string]interface{}{
		"prompt":      prompt,
		"temperature": temp,
		"top_p":       top_p,
		"max_gen_len": max_len,
	}
	bytes, err := json.Marshal(body)
	if err != nil {
		log.Println("Error converting interface to []byte:", err)
		return nil, err
	}
	input.Body = bytes

	// Begin streaming from the model
	stream, err := bedrockClient.InvokeModelWithResponseStream(context.TODO(), input)
	if err != nil {
		log.Println("Error invoking model:", err)
		return nil, err
	}

	// _, err = ProcessStreamingOutput(stream, func(ctx context.Context, part Generation) error {
	// 	fmt.Print(string(part.Generation))
	// 	return nil
	// })

	// if err != nil {
	// 	log.Fatal("streaming output processing error: ", err)
	// }

	// You can collect or return the response based on specific needs
	return stream, nil
}

type StreamingOutputHandler func(ctx context.Context, part Generation) error

func ProcessStreamingOutput(output *bedrockruntime.InvokeModelWithResponseStreamOutput, handler StreamingOutputHandler) (Generation, error) {

	var combinedResult string
	resp := Generation{}

	for event := range output.GetStream().Events() {
		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:

			//fmt.Println("payload", string(v.Value.Bytes))

			var resp Generation
			err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(&resp)
			if err != nil {
				return resp, err
			}

			handler(context.Background(), resp)
			combinedResult += resp.Generation

		case *types.UnknownUnionMember:
			fmt.Println("unknown tag:", v.Tag)

		default:
			fmt.Println("union is nil or unknown type")
		}
	}

	resp.Generation = combinedResult

	return resp, nil
}
