package mails

import (
	"errors"

	"github.com/MelloB1989/karma/internal/aws"
	"github.com/MelloB1989/karma/models"
)

type MailServices string

const (
	AWS_SES  MailServices = "AWS_SES"
	MAILGUN  MailServices = "MAILGUN"
	SENDGRID MailServices = "SENDGRID"
)

type MailClient struct {
	FromMail string
	Service  MailServices
}

func NewKarmaMail(fromMail string, service MailServices) *MailClient {
	return &MailClient{
		FromMail: fromMail,
		Service:  service,
	}
}

func (m *MailClient) SendSingleMail(email models.SingleEmailRequest) error {
	if m.Service == AWS_SES {
		return aws.SendEmailToSingleRecipient(email, m.FromMail)
	} else if m.Service == MAILGUN {
		return errors.New("Mailgun not supported yet")
	} else if m.Service == SENDGRID {
		return errors.New("Sendgrid not supported yet")
	} else {
		return errors.New("Mail service not supported")
	}
}
