package sns

import (
	"context"
	"fmt"
	"log"

	c "github.com/MelloB1989/karma/config"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewProduction()
	sugar     = logger.Sugar()
)

func createClient() *sns.Client {
	overrideRegion, _ := c.GetEnv("AWS_SNS_REGION")
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
	snsClient := sns.NewFromConfig(sdkConfig)
	return snsClient
}

type KarmaSNS struct {
	Client *sns.Client
}

func New() *KarmaSNS {
	return &KarmaSNS{
		Client: createClient(),
	}
}

func (k *KarmaSNS) GetAllTopic() []string {
	snsClient := k.Client
	var topics []types.Topic
	var topicArns []string
	paginator := sns.NewListTopicsPaginator(snsClient, &sns.ListTopicsInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Printf("Couldn't get topics. Here's why: %v\n", err)
			break
		} else {
			topics = append(topics, output.Topics...)
		}
	}
	if len(topics) == 0 {
		fmt.Println("You don't have any topics!")
	} else {
		for _, topic := range topics {
			topicArns = append(topicArns, *topic.TopicArn)
		}
	}
	return topicArns
}

func (k *KarmaSNS) CreateTopic(topicName string) string {
	snsClient := k.Client
	input := &sns.CreateTopicInput{
		Name: &topicName,
	}
	result, err := snsClient.CreateTopic(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't create topic. Here's why: %v\n", err)
		return ""
	}
	return *result.TopicArn
}

func (k *KarmaSNS) DeleteTopic(topicArn string) {
	snsClient := k.Client
	input := &sns.DeleteTopicInput{
		TopicArn: &topicArn,
	}
	_, err := snsClient.DeleteTopic(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't delete topic. Here's why: %v\n", err)
	}
}

func (k *KarmaSNS) SubscribeToTopic(topicArn, protocol, endpoint string) string {
	snsClient := k.Client
	input := &sns.SubscribeInput{
		Protocol: &protocol,
		TopicArn: &topicArn,
		Endpoint: &endpoint,
	}
	result, err := snsClient.Subscribe(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't subscribe to topic. Here's why: %v\n", err)
		return ""
	}
	return *result.SubscriptionArn
}

func (k *KarmaSNS) UnsubscribeFromTopic(subscriptionArn string) {
	snsClient := k.Client
	input := &sns.UnsubscribeInput{
		SubscriptionArn: &subscriptionArn,
	}
	_, err := snsClient.Unsubscribe(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't unsubscribe from topic. Here's why: %v\n", err)
	}
}

func (k *KarmaSNS) PublishToTopic(topicArn, message string) {
	snsClient := k.Client
	input := &sns.PublishInput{
		Message:  &message,
		TopicArn: &topicArn,
	}
	_, err := snsClient.Publish(context.TODO(), input)
	if err != nil {
		log.Printf("Couldn't publish message. Here's why: %v\n", err)
	}
}
