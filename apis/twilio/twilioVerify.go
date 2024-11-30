package twilio

import (
	"fmt"

	"github.com/MelloB1989/karma/config"

	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

func VerifyOTP(code string, phone string) bool {
	// Find your Account SID and Auth Token at twilio.com/console
	// and set the environment variables. See http://twil.io/secure
	accountSid := config.DefaultConfig().TwilioSID
	authToken := config.DefaultConfig().TwilioToken
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(phone)
	params.SetCode(code)

	resp, err := client.VerifyV2.CreateVerificationCheck(config.DefaultConfig().TwilioService, params)
	if err != nil {
		fmt.Println(err.Error())
		return false
	} else {
		if resp.Sid != nil {
			// fmt.Println(*resp.Status, *resp.Valid)
			if *resp.Status == "approved" && *resp.Valid {
				return true
			} else {
				return false
			}
		} else {
			fmt.Println("Not Verified")
			return false
		}
	}
}
