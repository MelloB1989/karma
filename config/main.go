package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	Port                     string
	JWTSecret                string
	AdminKey                 string
	AdministratorKey         string
	BACKEND_URL              string
	DatabaseURL              string
	DatabaseName             string
	DatabaseHost             string
	DatabasePort             string
	DatabaseUser             string
	DatabasePassword         string
	DatabaseSSLMode          string
	RedisURL                 string
	RedisToken               string
	ErrorsDefinationFile     string
	TwilioSID                string
	TwilioToken              string
	TwilioService            string
	TestPhoneNumbers         []string
	LogLevel                 string
	ApiKey                   string
	ClientID                 string
	ClientSecret             string
	AwsAccessKey             string
	AwsSecretKey             string
	AwsRegion                string
	AwsBucketName            string
	S3BucketRegion           string
	AwsBedrockRegion         string
	SendGridAPIKey           string
	StripeSecretKey          string
	MailgunAPIKey            string
	Environment              string
	WebhookSecret            string
	KARMAPAY_WEBHOOK         string
	KARMAPAY_PG_ENUM         string
	KARMAPAY_API_KEY         string
	KARMAPAY_IDENTIFIER      string
	KARMAPAY_APP_DOMAIN      string
	KARMAPAY_VERIFY_URL      string
	GOOGLE_AUTH_CALLBACK_URL string
	GOOGLE_CLIENT_ID         string
	GOOGLE_CLIENT_SECRET     string
	OPENAI_KEY               string
}

func DefaultConfig() *Config {
	err := godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()
	if err != nil {
		sugar.Error("unable to load .env")
	}
	// sugar.Info("loaded .env file")

	return &Config{
		Port:                 os.Getenv("PORT"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		AdminKey:             os.Getenv("ADMIN_KEY"),
		AdministratorKey:     os.Getenv("ADMINISTRATOR_KEY"),
		BACKEND_URL:          os.Getenv("BACKEND_URL"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseName:         os.Getenv("DATABASE_NAME"),
		DatabaseHost:         os.Getenv("DATABASE_HOST"),
		DatabasePort:         os.Getenv("DATABASE_PORT"),
		DatabaseUser:         os.Getenv("DATABASE_USER"),
		DatabasePassword:     os.Getenv("DATABASE_PASSWORD"),
		DatabaseSSLMode:      os.Getenv("DATABASE_SSLMODE"),
		RedisURL:             os.Getenv("REDIS_URL"),
		RedisToken:           os.Getenv("REDIS_TOKEN"),
		ErrorsDefinationFile: os.Getenv("ERRORS_DEFINATION_FILE"),
		TwilioSID:            os.Getenv("TWILIO_SID"),
		TwilioToken:          os.Getenv("TWILIO_TOKEN"),
		TwilioService:        os.Getenv("TWILIO_SERVICE"),
		TestPhoneNumbers: []string{"+919812940706",
			"+917398394041",
			"+919662105710",
			"+918266187862",
			"+919923104801",
			"+917740674090",
			"+913175523534",
			"+912082265688",
			"+916892770133",
			"+911937243659",
			"+912958198971",
			"+917842248874",
			"+916149247480",
			"+912090090825",
			"+910104982716",
			"+911566326784",
			"+912376627951",
			"+912362790556",
			"+913963969678",
			"+914579253395"},
		LogLevel:                 os.Getenv("LOG_LEVEL"),
		ApiKey:                   os.Getenv("API_KEY"),
		ClientID:                 os.Getenv("CLIENT_ID"),
		ClientSecret:             os.Getenv("CLIENT_SECRET"),
		AwsAccessKey:             os.Getenv("AWS_ACCESS_KEY_ID"),
		AwsSecretKey:             os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AwsRegion:                os.Getenv("AWS_REGION"),
		AwsBucketName:            os.Getenv("BUCKET_NAME"),
		S3BucketRegion:           os.Getenv("BUCKET_REGION"),
		AwsBedrockRegion:         os.Getenv("BEDROCK_REGION"),
		SendGridAPIKey:           os.Getenv("SENDGRID_API_KEY"),
		StripeSecretKey:          os.Getenv("STRIPE_SECRET_KEY"),
		MailgunAPIKey:            os.Getenv("MAILGUN_API_KEY"),
		Environment:              os.Getenv("ENVIRONMENT"),
		WebhookSecret:            os.Getenv("WEBHOOK_SECRET"),
		KARMAPAY_WEBHOOK:         os.Getenv("KARMAPAY_WEBHOOK"),
		KARMAPAY_PG_ENUM:         os.Getenv("KARMAPAY_PG_ENUM"),
		KARMAPAY_API_KEY:         os.Getenv("KARMAPAY_API_KEY"),
		KARMAPAY_IDENTIFIER:      os.Getenv("KARMAPAY_IDENTIFIER"),
		KARMAPAY_APP_DOMAIN:      os.Getenv("KARMAPAY_APP_DOMAIN"),
		KARMAPAY_VERIFY_URL:      os.Getenv("KARMAPAY_VERIFY_URL"),
		GOOGLE_AUTH_CALLBACK_URL: os.Getenv("GOOGLE_AUTH_CALLBACK_URL"),
		GOOGLE_CLIENT_ID:         os.Getenv("GOOGLE_CLIENT_ID"),
		GOOGLE_CLIENT_SECRET:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		OPENAI_KEY:               os.Getenv("OPENAI_KEY"),
	}
}

/*
* Example usage:
package main

import (
	"fmt"
	"github.com/MelloB1989/karma/config"
)

func main() {
	fmt.Println("Default Config:", config.DefaultConfig().Port)
}
*/

// If you want to use a custom configuration, you can use the CustomConfig function.
func CustomConfig(cfg interface{}) error {
	err := godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()
	if err != nil {
		sugar.Error("unable to load .env")
	}
	// sugar.Info("loaded .env file")

	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("CustomConfig: expected a pointer to a struct, got %T", cfg)
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		envVar := field.Tag.Get("env")
		if envVar == "" {
			envVar = field.Name
		}
		value := os.Getenv(envVar)
		if value != "" {
			v.Field(i).SetString(value)
		}
	}

	return nil
}

/*
* Example usage:
package main

import (

	"fmt"
	"github.com/MelloB1989/karma/config"

)

	type MyCustomConfig struct {
		AppName     string `env:"APP_NAME"`
		CustomField string `env:"CUSTOM_FIELD"`
		DatabaseURL string `env:"DATABASE_URL"`
	}

	func main() {
		customConfig := &MyCustomConfig{}
		err := config.CustomConfig(customConfig)
		if err != nil {
			panic(err)
		}

		fmt.Println("Custom Config:", customConfig)
	}
*/
func GetEnv(key string) (string, error) {
	err := godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()
	if err != nil {
		sugar.Error("unable to load a ENV variable")
		return "", err
	}
	// sugar.Info("loaded .env file")
	return os.Getenv(key), nil
}

func GetEnvOrDefault(key string, defaultValue string) string {
	value, err := GetEnv(key)
	if err != nil {
		return defaultValue
	}
	if value == "" {
		return defaultValue
	}
	return value
}

func GetEnvRaw(key string) string {
	value, err := GetEnv(key)
	if err != nil {
		return ""
	}
	return value
}
