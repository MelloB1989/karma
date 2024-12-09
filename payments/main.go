package payments

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/razorpay/razorpay-go"
)

type Order struct {
	UID                   string `json:"uid"`
	OrderID               string `json:"order_id"`
	OrderAmount           string `json:"order_amount"`
	OrderStatus           string `json:"order_status"`
	OrderCID              string `json:"order_cid"`
	OrderCurrency         string `json:"order_currency"`
	OrderDescription      string `json:"order_description"`
	OrderTimeStamp        string `json:"order_timestamp"`
	OrderUpiTransactionID string `json:"order_upi_transaction_id"`
}

type RedisOrder struct {
	OrderID          string          `json:"order_id"`
	OrderStatus      string          `json:"order_status"`
	UID              string          `json:"uid"`
	Email            string          `json:"email"`
	KPAPI            string          `json:"kpapi"`
	API_KEY          string          `json:"api_key"`
	OrderAmount      string          `json:"order_amt"`
	OrderCurrency    string          `json:"order_currency"`
	OrderDescription string          `json:"order_description"`
	Subdomain        string          `json:"subdomain"`
	OrderMode        string          `json:"order_mode"`
	WebhookURL       string          `json:"webhook_url"`
	RedirectURL      string          `json:"redirect_url"`
	VerifyURL        string          `json:"verify_url"`
	Registration     string          `json:"registration"`
	OrderCID         string          `json:"order_cid"`
	PGOrder          json.RawMessage `json:"PGOrder"`
	Timestamp        string          `json:"timestamp"`
}

type CreatePaymentOrder struct {
	OrderAmount      int32  `json:"order_amt"`
	OrderCurrency    string `json:"order_currency"`
	OrderDescription string `json:"order_description"`
	OrderMode        string `json:"order_mode"`
	RedirectURL      string `json:"redirect_url"`
	WebhookURL       string `json:"webhook_url"`
	Registration     string `json:"registration"`
}

func CreateOrder(order CreatePaymentOrder) string {
	oid := utils.GenerateID(25)
	var orderData RedisOrder = RedisOrder{
		OrderID:          oid,
		OrderStatus:      "PENDING",
		UID:              config.DefaultConfig().KARMAPAY_IDENTIFIER, //Use as app name
		Email:            config.DefaultConfig().KARMAPAY_IDENTIFIER,
		API_KEY:          config.DefaultConfig().KARMAPAY_API_KEY,
		VerifyURL:        config.DefaultConfig().KARMAPAY_VERIFY_URL,
		OrderAmount:      fmt.Sprintf("%d", order.OrderAmount),
		OrderCurrency:    order.OrderCurrency,
		OrderDescription: order.OrderDescription,
		Subdomain:        config.DefaultConfig().KARMAPAY_APP_DOMAIN,
		OrderMode:        order.OrderMode,
		OrderCID:         "",
		PGOrder:          json.RawMessage(`{}`),
		KPAPI:            "",
		Registration:     order.Registration,
		RedirectURL:      order.RedirectURL,
		WebhookURL:       fmt.Sprintf("%s&webhook_key=%s&koid=%s", order.WebhookURL, config.DefaultConfig().WebhookSecret, oid),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
	}
	PushOrderToRedis(orderData)
	return oid
}

type VerifyPaymentRequest struct {
	OID       string `json:"oid"`
	CID       string `json:"cid"`
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
	Signature string `json:"signature"`
	RZKey     string `json:"RZKey"`
}

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func VerifyPaymentAPI() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		req := new(VerifyPaymentRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}
		order, err := GetOrderFromRedis(req.OID)
		if err != nil {
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to get order from Redis.",
				Data:    nil,
			})
		}

		// var orderData Order = Order{
		// 	OrderID:               order.OrderID,
		// 	OrderAmount:           order.OrderAmount,
		// 	OrderCurrency:         order.OrderCurrency,
		// 	OrderDescription:      order.OrderDescription,
		// 	OrderStatus:           "COMPLETED",
		// 	OrderCID:              req.CID,
		// 	OrderTimeStamp:        order.Timestamp,
		// 	OrderUpiTransactionID: "",
		// 	UID:                   order.UID,
		// }
		api, err := DecodeAPI(order.API_KEY)
		if err != nil {
			fmt.Printf("Error decoding API: %v\n", err)
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}
		id, ok := api["key"].(string)
		if !ok {
			fmt.Println("Error: 'id' field is not a string")
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}
		secret, ok := api["secret"].(string)
		if !ok {
			fmt.Println("Error: 'id' field is not a string")
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}
		client := razorpay.NewClient(id, secret)
		data := map[string]interface{}{
			"expand[]": "emi",
		}

		body, err := client.Payment.Fetch(req.PaymentID, data, nil)

		if len(body) == 0 {
			log.Println("The body map is empty.")
			return c.JSON(ResponseHTTP{
				Success: false,
				Message: "Payment not done.",
				Data:    nil,
			})
		} else {
			if body["captured"].(bool) {
				err := TriggerWebhook(order.WebhookURL)
				if err != nil {
					return c.JSON(ResponseHTTP{
						Success: false,
						Message: "Failed to trigger webhook.",
						Data:    nil,
					})
				}
				return c.JSON(ResponseHTTP{
					Success: true,
					Message: "Payment verified",
					Data:    nil,
				})
			}
		}
		return c.JSON(ResponseHTTP{
			Success: false,
			Message: "Payment not done.",
			Data:    nil,
		})
	}
}

func KarmaPayWebhook(action func(data map[string]string) error) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		webhookKey := c.Query("webhook_key")
		if webhookKey != config.DefaultConfig().WebhookSecret {
			return c.JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid webhook key.",
				Data:    nil,
			})
		}
		queries := c.Queries()
		err := action(queries)
		if err != nil {
			return c.JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to process webhook.",
				Data:    nil,
			})
		}
		return c.JSON(ResponseHTTP{
			Success: true,
			Message: "Webhook verified.",
			Data:    nil,
		})
	}
}
