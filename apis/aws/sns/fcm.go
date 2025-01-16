package sns

import (
	"context"
	"log"

	"github.com/MelloB1989/karma/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type KarmaFCMSNS struct {
	Client         *sns.Client
	ApplicationArn string
}

func NewFCM(arn ...string) *KarmaFCMSNS {
	ap, _ := config.GetEnv("AWS_SNS_APPLICATION_ARN")
	if len(arn) > 0 {
		ap = arn[0]
	}
	return &KarmaFCMSNS{
		Client:         createClient(),
		ApplicationArn: ap,
	}
}

func (k *KarmaFCMSNS) CreateApplicationPlatformEndpoint(user_data, token string) string {
	snsClient := k.Client
	input := &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: &k.ApplicationArn,
		Token:                  &token,
		CustomUserData:         &user_data,
	}
	result, err := snsClient.CreatePlatformEndpoint(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't create platform endpoint. Here's why: %v\n", err)
		return ""
	}
	return *result.EndpointArn
}

func (k *KarmaFCMSNS) DeleteApplicationPlatformEndpoint(endpoint_arn string) {
	snsClient := k.Client
	input := &sns.DeleteEndpointInput{
		EndpointArn: &endpoint_arn,
	}
	_, err := snsClient.DeleteEndpoint(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't delete platform endpoint. Here's why: %v\n", err)
	}
}

func (k *KarmaFCMSNS) GetEndpointAttributes(endpoint_arn string) {
	snsClient := k.Client
	input := &sns.GetEndpointAttributesInput{
		EndpointArn: &endpoint_arn,
	}
	_, err := snsClient.GetEndpointAttributes(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't get endpoint attributes. Here's why: %v\n", err)
	}
}

func (k *KarmaFCMSNS) GetEndpointARNByUserData(user_data string) string {
	snsClient := k.Client
	input := &sns.ListEndpointsByPlatformApplicationInput{
		PlatformApplicationArn: aws.String(k.ApplicationArn),
	}

	paginator := sns.NewListEndpointsByPlatformApplicationPaginator(snsClient, input)

	for paginator.HasMorePages() {
		result, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Printf("Couldn't get endpoint attributes. Here's why: %v\n", err)
			return ""
		}

		for _, endpoint := range result.Endpoints {
			if userData, ok := endpoint.Attributes["CustomUserData"]; ok {
				if userData == user_data {
					return *endpoint.EndpointArn
				}
			}
		}
	}
	return ""
}

func (k *KarmaFCMSNS) PublishGCMMessage(user_data, message string) {
	endpoint_arn := k.GetEndpointARNByUserData(user_data)
	snsClient := k.Client
	input := &sns.PublishInput{
		Message:          &message,
		TargetArn:        &endpoint_arn,
		MessageStructure: aws.String("json"),
	}
	_, err := snsClient.Publish(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't publish message. Here's why: %v\n", err)
	}
}
