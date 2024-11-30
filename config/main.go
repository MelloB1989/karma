package config

import (
	"fmt"
	"os"
	"reflect"

	"github.com/joho/godotenv"
	"golang.org/x/exp/slog"
)

type Config struct {
	Port             string
	JWTSecret        string
	AdminKey         string
	AdministratorKey string
	BACKEND_URL      string
	RedisURL         string
	DatabaseURL      string
	LogLevel         string
	ApiKey           string
	ClientID         string
	ClientSecret     string
	AwsAccessKey     string
	AwsSecretKey     string
	AwsRegion        string
	AwsBucketName    string
	S3BucketRegion   string
	AwsBedrockRegion string
	SendGridAPIKey   string
	StripeSecretKey  string
	MailgunAPIKey    string
	Environment      string
}

func DefaultConfig() *Config {
	err := godotenv.Load()
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	if err != nil {
		logger.Error("unable to load .env")
	}

	return &Config{
		Port:             os.Getenv("PORT"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AdminKey:         os.Getenv("ADMIN_KEY"),
		AdministratorKey: os.Getenv("ADMINISTRATOR_KEY"),
		BACKEND_URL:      os.Getenv("BACKEND_URL"),
		RedisURL:         os.Getenv("REDIS_URL"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		LogLevel:         os.Getenv("LOG_LEVEL"),
		ApiKey:           os.Getenv("API_KEY"),
		ClientID:         os.Getenv("CLIENT_ID"),
		ClientSecret:     os.Getenv("CLIENT_SECRET"),
		AwsAccessKey:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AwsSecretKey:     os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AwsRegion:        os.Getenv("AWS_REGION"),
		AwsBucketName:    os.Getenv("BUCKET_NAME"),
		S3BucketRegion:   os.Getenv("BUCKET_REGION"),
		AwsBedrockRegion: os.Getenv("BEDROCK_REGION"),
		SendGridAPIKey:   os.Getenv("SENDGRID_API_KEY"),
		StripeSecretKey:  os.Getenv("STRIPE_SECRET_KEY"),
		MailgunAPIKey:    os.Getenv("MAILGUN_API_KEY"),
		Environment:      os.Getenv("ENVIRONMENT"),
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
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	if err != nil {
		logger.Error("unable to load .env")
	}

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
