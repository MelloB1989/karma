package aws

import (
	"context"
	"log"

	c "github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewProduction()
	sugar     = logger.Sugar()
)

func createClient() *ses.Client {
	overrideRegion, _ := c.GetEnv("AWS_SES_REGION")
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
	sesClient := ses.NewFromConfig(sdkConfig)
	return sesClient
}

func SendEmailToSingleRecipient(email models.SingleEmailRequest, from string) error {
	sesClient := createClient()
	EmailInput := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{email.To},
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(email.Email.Body.HTML),
				},
				Text: &types.Content{
					Data: aws.String(email.Email.Body.Text),
				},
			},
			Subject: &types.Content{
				Data: aws.String(email.Email.Subject),
			},
		},
		Source: &from,
	}
	out, err := sesClient.SendEmail(context.TODO(), EmailInput)
	if err != nil {
		log.Println("Error sending email")
		log.Println(err)
		return err
	}
	sugar.Info("Email sent!", out)
	return nil
}
