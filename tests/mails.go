package tests

import (
	"github.com/MelloB1989/karma/mails"
	"github.com/MelloB1989/karma/models"
)

func TestSendingSingleMail() {
	km := mails.NewKarmaMail("internal@mails.coffeecodes.in", "AWS_SES")
	err := km.SendSingleMail(models.SingleEmailRequest{
		To: "kartik.mellob@coffeecodes.in",
		Email: models.Email{
			Subject: "Testing!",
			Body: models.EmailBody{
				Text: "Hello!!",
				HTML: "<h1>Hello HTML!</h1>",
			},
		},
	})
	if err != nil {
		panic(err)
	}
}
