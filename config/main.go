package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

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

func GetEnv(key string) (string, error) {
	err := godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	if err != nil {
		logger.Error("unable to load .env")
	}
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

// AppConfig is the generic singleton wrapper around any user-defined config struct.
// T must be a pointer to a struct.
type AppConfig[T any] struct {
	mu       sync.RWMutex
	instance T
	loaded   bool
	optional map[string]bool   // field names that are optional
	defaults map[string]string // field name -> default value
}

var (
	// globalInstances stores one *AppConfig per concrete struct type name.
	globalInstances   = map[string]any{}
	globalInstancesMu sync.RWMutex
)

// NewAppConfig creates (or returns the existing) singleton AppConfig for type T.
// Call this once at startup (e.g., in init() or main()).
//
//	type MyConfig struct {
//	    AppName  string `env:"APP_NAME"`
//	    Debug    string `env:"DEBUG"    optional:"true"  default:"false"`
//	    DBUrl    string `env:"DATABASE_URL"`
//	}
//
//	var Cfg = config.NewAppConfig[*MyConfig]()
func NewAppConfig[T any]() *AppConfig[T] {
	var zero T
	typeName := fmt.Sprintf("%T", zero)

	globalInstancesMu.Lock()
	defer globalInstancesMu.Unlock()

	if existing, ok := globalInstances[typeName]; ok {
		return existing.(*AppConfig[T])
	}

	ac := &AppConfig[T]{
		optional: make(map[string]bool),
		defaults: make(map[string]string),
	}
	globalInstances[typeName] = ac
	return ac
}

// Load reads environment variables into T.
// It respects `env`, `optional`, and `default` struct tags.
// Fields tagged `optional:"true"` are not required to be present.
// Fields tagged `default:"value"` get that value when the env var is empty.
//
// Load is idempotent — calling it again reloads values from the environment.
func (ac *AppConfig[T]) Load() error {
	if err := godotenv.Load(); err != nil {
		// Non-fatal: variables may already be in the environment.
		logger, _ := zap.NewProduction()
		defer logger.Sync()
		logger.Warn("unable to load .env file", zap.Error(err))
	}

	var zero T
	rv := reflect.ValueOf(zero)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("AppConfig: T must be a pointer to a struct, got %T", zero)
	}

	// Allocate a new concrete value of T's elem type.
	elem := reflect.New(rv.Type().Elem())
	v := elem.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		// Only handle string fields.
		if fv.Kind() != reflect.String {
			continue
		}

		envKey := field.Tag.Get("env")
		if envKey == "" {
			envKey = field.Name
		}

		isOptional := field.Tag.Get("optional") == "true"
		defaultVal := field.Tag.Get("default")

		// Merge with programmatic optional/default registrations.
		if ac.optional[field.Name] {
			isOptional = true
		}
		if d, ok := ac.defaults[field.Name]; ok {
			defaultVal = d
		}

		value := os.Getenv(envKey)
		if value == "" {
			value = defaultVal
		}

		fv.SetString(value)

		// Track optional status derived from tags for Validate().
		if isOptional {
			ac.optional[field.Name] = true
		}
	}

	ac.mu.Lock()
	ac.instance = elem.Interface().(T)
	ac.loaded = true
	ac.mu.Unlock()

	return nil
}

// MustLoad calls Load and panics on error. Useful for fail-fast startup.
func (ac *AppConfig[T]) MustLoad() *AppConfig[T] {
	if err := ac.Load(); err != nil {
		panic(fmt.Sprintf("config: failed to load: %v", err))
	}
	return ac
}

// Get returns the loaded config instance.
// Panics if Load has not been called yet.
func (ac *AppConfig[T]) Get() T {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	if !ac.loaded {
		panic("config: Get() called before Load(). Call MustLoad() or Load() first.")
	}
	return ac.instance
}

// GetField returns the string value of a named field.
// Returns ("", false) if the field does not exist or is not a string.
func (ac *AppConfig[T]) GetField(fieldName string) (string, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	v := reflect.ValueOf(ac.instance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() || f.Kind() != reflect.String {
		return "", false
	}
	return f.String(), true
}

// MustGetField returns the value of a named field or panics if missing/empty.
func (ac *AppConfig[T]) MustGetField(fieldName string) string {
	val, ok := ac.GetField(fieldName)
	if !ok || val == "" {
		panic(fmt.Sprintf("config: required field %q is missing or empty", fieldName))
	}
	return val
}

// SetOptional marks field names as optional so Validate() does not flag them.
//
//	cfg.SetOptional("Debug", "FeatureFlag")
func (ac *AppConfig[T]) SetOptional(fieldNames ...string) *AppConfig[T] {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	for _, name := range fieldNames {
		ac.optional[name] = true
	}
	return ac
}

// SetDefault registers a fallback value for a field used during the next Load().
//
//	cfg.SetDefault("LogLevel", "info").SetDefault("Port", "8080")
func (ac *AppConfig[T]) SetDefault(fieldName, value string) *AppConfig[T] {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.defaults[fieldName] = value
	return ac
}

// Validate checks that all non-optional string fields are non-empty.
// Returns a descriptive error listing every missing field.
func (ac *AppConfig[T]) Validate() error {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if !ac.loaded {
		return fmt.Errorf("config: Validate() called before Load()")
	}

	v := reflect.ValueOf(ac.instance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	var missing []string
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if fv.Kind() != reflect.String {
			continue
		}

		isOptional := ac.optional[field.Name] ||
			field.Tag.Get("optional") == "true"

		if !isOptional && fv.String() == "" {
			envKey := field.Tag.Get("env")
			if envKey == "" {
				envKey = field.Name
			}
			missing = append(missing, fmt.Sprintf("%s (env: %s)", field.Name, envKey))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("config: missing required fields:\n  - %s",
			strings.Join(missing, "\n  - "))
	}
	return nil
}

// MustValidate calls Validate and panics on error.
func (ac *AppConfig[T]) MustValidate() *AppConfig[T] {
	if err := ac.Validate(); err != nil {
		panic(err.Error())
	}
	return ac
}

// Missing returns the names of all non-optional empty fields, or nil if all present.
func (ac *AppConfig[T]) Missing() []string {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if !ac.loaded {
		return nil
	}

	v := reflect.ValueOf(ac.instance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	var out []string
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if fv.Kind() != reflect.String {
			continue
		}
		isOptional := ac.optional[field.Name] || field.Tag.Get("optional") == "true"
		if !isOptional && fv.String() == "" {
			out = append(out, field.Name)
		}
	}
	return out
}

// IsLoaded reports whether Load has been called at least once.
func (ac *AppConfig[T]) IsLoaded() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.loaded
}

// Override sets a field's value at runtime without reloading from the environment.
// Useful in tests or for dynamic overrides.
func (ac *AppConfig[T]) Override(fieldName, value string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	v := reflect.ValueOf(ac.instance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	} else {
		// For pointer types we need addressability; rebuild through pointer.
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		v = ptr.Elem()
	}

	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return fmt.Errorf("config: field %q not found", fieldName)
	}
	if f.Kind() != reflect.String {
		return fmt.Errorf("config: field %q is not a string", fieldName)
	}
	if !f.CanSet() {
		return fmt.Errorf("config: field %q is not settable", fieldName)
	}
	f.SetString(value)
	return nil
}

/*
──────────────────────────────────────────────────────────────────────────────
USAGE GUIDE
──────────────────────────────────────────────────────────────────────────────

1. Define your config struct with tags:

	type MyConfig struct {
	    AppName  string `env:"APP_NAME"`
	    Port     string `env:"PORT"      default:"8080"  optional:"true"`
	    DBUrl    string `env:"DATABASE_URL"`
	    Debug    string `env:"DEBUG"     optional:"true" default:"false"`
	    LogLevel string `env:"LOG_LEVEL" optional:"true" default:"info"`
	    Secret   string `env:"JWT_SECRET"`
	}

2. Declare a package-level singleton (one per config type):

	// config/app.go  (or anywhere in your package)
	var AppCfg = config.NewAppConfig[*MyConfig]().
	    SetDefault("Port", "9090").   // programmatic defaults (override tags)
	    SetOptional("Debug").         // programmatic optional marking
	    MustLoad().                   // load + panic on error
	    MustValidate()                // validate + panic on missing required fields

3. Access from anywhere in your app — no re-initialisation needed:

	func main() {
	    db := connectDB(AppCfg.Get().DBUrl)
	    port := AppCfg.Get().Port

	    // Or by field name (useful for dynamic lookups):
	    secret, ok := AppCfg.GetField("Secret")
	    _ = AppCfg.MustGetField("AppName") // panics if empty
	}

4. Validate at startup:

	if err := AppCfg.Validate(); err != nil {
	    log.Fatal(err)
	}
	// or use MustValidate() to panic immediately.

5. Check what's missing without erroring:

	if missing := AppCfg.Missing(); len(missing) > 0 {
	    log.Printf("optional fields not set: %v", missing)
	}

6. Override in tests without touching .env:

	AppCfg.Override("Secret", "test-secret")

──────────────────────────────────────────────────────────────────────────────
*/
