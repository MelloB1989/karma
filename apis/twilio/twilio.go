package twilio

import (
	"fmt"

	"github.com/MelloB1989/karma/config"

	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

func SendOTP(phone string) bool {
	// Find your Account SID and Auth Token at twilio.com/console
	// and set the environment variables. See http://twil.io/secure
	accountSid := config.DefaultConfig().TwilioSID
	authToken := config.DefaultConfig().TwilioToken
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &verify.CreateVerificationParams{}
	params.SetTo(phone)
	params.SetChannel("sms")

	resp, err := client.VerifyV2.CreateVerification(config.DefaultConfig().TwilioService, params)
	if err != nil {
		fmt.Println(err.Error())
		return false
	} else {
		if resp.Sid != nil {
			// fmt.Println(*resp.Sid)
			return true
		} else {
			// fmt.Println(resp.Sid)
			return false
		}
	}
}
