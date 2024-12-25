package auth

import (
	"log"
	"time"

	"github.com/MelloB1989/karma/apis/twilio"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/google"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/golang-jwt/jwt"
)

type User struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginWithEmailAndPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type JWTClaimsProvider interface {
	GetJWTClaims() map[string]interface{}
	AdditionalClaims() map[string]interface{}
}

type AuthUserEmail interface {
	GetEmail() string
	GetPassword() string
	GetID() string
	JWTClaimsProvider
}

type AuthUserPhone interface {
	GetPhone() string
	GetPassword() string
	GetID() string
	JWTClaimsProvider
}

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetPhone() string {
	return u.Phone
}

func (u *User) GetPassword() string {
	return ""
}

func (u *User) GetID() string {
	return u.Id
}

func (u *User) GetJWTClaims() map[string]interface{} {
	return map[string]interface{}{
		"id":    u.Id,
		"phone": u.Phone,
	}
}

func (u *User) AdditionalClaims() map[string]interface{} {
	return map[string]interface{}{
		"role": "user",
	}
}

func NewAuthUserPhone(phone, password, id string) AuthUserPhone {
	return &User{
		Phone:    phone,
		Password: password,
		Id:       id,
		Email:    "",
	}
}

func NewAuthUserEmail(email, password, id string) AuthUserPhone {
	return &User{
		Phone:    "",
		Password: password,
		Id:       id,
		Email:    email,
	}
}

func LoginWithEmailAndPasswordHandler(getUserByEmail func(email string) (AuthUserEmail, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Parse and validate request body
		req := new(LoginWithEmailAndPasswordRequest)
		if err := c.BodyParser(req); err != nil {
			log.Printf("Body parsing error: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid request payload.",
			})
		}

		// Basic validation for email and password presence
		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Email and password are required.",
			})
		}

		// Retrieve user by email
		user, err := getUserByEmail(req.Email)
		if err != nil {
			log.Printf("Error retrieving user by email (%s): %v", req.Email, err)
			// To prevent user enumeration, return a generic message
			return c.Status(fiber.StatusUnauthorized).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid email or password.",
			})
		}

		// If user is nil (not found), return generic message
		if user == nil {
			log.Printf("User not found for email: %s", req.Email)
			return c.Status(fiber.StatusUnauthorized).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid email or password.",
			})
		}

		// Check if the provided password matches the stored hash
		if !utils.CheckPasswordHash(req.Password, user.GetPassword()) {
			log.Printf("Invalid password attempt for email: %s", req.Email)
			return c.Status(fiber.StatusUnauthorized).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid email or password.",
			})
		}

		// Load configuration and ensure it's valid
		cfg := config.DefaultConfig()
		if cfg == nil {
			log.Println("Configuration retrieval failed: config is nil.")
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Internal server error.",
			})
		}

		// Create JWT token with claims
		claims := jwt.MapClaims{
			"email": user.GetEmail(),
			"uid":   user.GetID(),
			"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
		}

		// Add additional claims if the user implements JWTClaimsProvider
		if claimsProvider, ok := user.(JWTClaimsProvider); ok {
			for key, value := range claimsProvider.AdditionalClaims() {
				claims[key] = value
			}
		}

		// Create the token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		// Sign the token with the secret
		jwtSecret := []byte(cfg.JWTSecret)
		if len(jwtSecret) == 0 {
			log.Println("JWT secret is not set in the configuration.")
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Internal server error.",
			})
		}

		signedToken, err := token.SignedString(jwtSecret)
		if err != nil {
			log.Printf("Error signing JWT token for email (%s): %v", req.Email, err)
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to generate authentication token.",
			})
		}

		// Respond with the signed token
		return c.Status(fiber.StatusOK).JSON(ResponseHTTP{
			Success: true,
			Message: "Login successful.",
			Data:    map[string]string{"token": signedToken},
		})
	}
}

type LoginWithPhoneOTPRequest struct {
	Phone string `json:"phone"`
}

func LoginWithPhoneOTPHandler(getUserByPhone func(phone string) (AuthUserPhone, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Parse the request body into LoginWithPhoneOTPRequest
		req := new(LoginWithPhoneOTPRequest)
		if err := c.BodyParser(req); err != nil {
			log.Printf("Body parsing error: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}

		// Validate the phone number format
		if !utils.VerifyPhoneNumber(req.Phone) {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid phone number format.",
				Data:    nil,
			})
		}

		// Load the configuration once
		cfg := config.DefaultConfig()
		if cfg == nil {
			log.Println("Configuration is nil.")
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Configuration error.",
				Data:    nil,
			})
		}

		// Check if the phone number is a test number
		isTestNumber := utils.Contains(cfg.TestPhoneNumbers, req.Phone)

		// Attempt to retrieve the user by phone number
		user, err := getUserByPhone(req.Phone)
		if err != nil || user == nil {
			log.Printf("Error retrieving user by phone (%s): %v", req.Phone, err)
		}

		// Determine if the account exists
		accountExists := false
		if err == nil && user != nil && user.GetPhone() != "" {
			accountExists = true
		}

		// Send OTP only if it's not a test number
		if !isTestNumber {
			otpSent := twilio.SendOTP(req.Phone)
			if !otpSent {
				log.Printf("Failed to send OTP to phone (%s)", req.Phone)
				return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
					Success: false,
					Message: "Failed to send OTP.",
					Data:    nil,
				})
			}
		}

		// Prepare the response data
		responseData := map[string]bool{
			"account_exists": accountExists,
		}

		// Include test_phone flag if it's a test number
		if isTestNumber {
			responseData["test_phone"] = true
		}

		return c.Status(fiber.StatusOK).JSON(ResponseHTTP{
			Success: true,
			Message: "OTP sent to phone number.",
			Data:    responseData,
		})
	}
}

type VerifyPhoneOTPRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

func VerifyPhoneOTPHandler(getUserByPhone func(phone string) (AuthUserPhone, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Parse the request body into VerifyPhoneOTPRequest
		req := new(VerifyPhoneOTPRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}

		// Validate the phone number format
		if !utils.VerifyPhoneNumber(req.Phone) {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid phone number format.",
				Data:    nil,
			})
		}

		// Load the configuration
		config := config.DefaultConfig()
		if config == nil {
			log.Println("Configuration is nil.")
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Configuration error.",
				Data:    nil,
			})
		}

		// Verify OTP or check if the phone number is a test number
		validOTP := twilio.VerifyOTP(req.OTP, req.Phone)
		isTestNumber := utils.Contains(config.TestPhoneNumbers, req.Phone)

		if !validOTP && !isTestNumber {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid OTP.",
				Data:    nil,
			})
		}

		// Initialize account existence as false
		accountExists := false

		// Attempt to retrieve the user by phone number
		user, err := getUserByPhone(req.Phone)
		if err != nil || user == nil {
			accountExists = false
		} else {
			// User exists
			accountExists = true
		}

		// Initialize JWT claims
		claims := jwt.MapClaims{
			"phone": req.Phone,
			"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days expiration
		}

		// If user exists, add additional claims
		if accountExists && user != nil {
			claims["uid"] = user.GetID()
			claims["phone"] = user.GetPhone() // This might be redundant if already set above
			if claimsProvider, ok := user.(JWTClaimsProvider); ok {
				for key, value := range claimsProvider.AdditionalClaims() {
					claims[key] = value
				}
			}
		}

		// Create and sign the JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(config.JWTSecret))
		if err != nil {
			log.Printf("Failed to create token for phone (%s): %v", req.Phone, err)
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to create token.",
				Data:    nil,
			})
		}

		// Respond with the token and account existence status
		return c.JSON(ResponseHTTP{
			Success: true,
			Message: "OTP verified.",
			Data: map[string]interface{}{
				"account_exists": accountExists,
				"token":          tokenString,
			},
		})
	}
}

type GoogleConfig struct {
	CookieExpiration time.Duration
	CookieDomain     string
	CookieHTTPSOnly  bool
	OAuthStateString string
}

type GoogleAuth struct {
	config GoogleConfig
}

type GoogleCallbackData struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	ID            string `json:"id"`
}

func NewGoogleAuth(config GoogleConfig) *GoogleAuth {
	return &GoogleAuth{
		config: config,
	}
}

func InitializeGoogleAuth(config GoogleConfig) *GoogleAuth {
	google.InitializeStore(config.CookieExpiration, config.CookieDomain, config.CookieHTTPSOnly, config.OAuthStateString)
	return NewGoogleAuth(config)
}

func (ga *GoogleAuth) GetGoogleOauthURL() string {
	return google.GetGoogleOauthURL()
}

func (ga *GoogleAuth) GoogleLoginHandler() func(c *fiber.Ctx) error {
	return google.HandleGoogleLogin
}

func (ga *GoogleAuth) GoogleLoginBuilder(authHandler func(c *fiber.Ctx) error) func(c *fiber.Ctx) error {
	return google.AuthBuilder(authHandler)
}

func (ga *GoogleAuth) GoogleCallbackBuilder(callbackHandler func(c *fiber.Ctx, user *models.GoogleCallbackData, tokenSess *session.Session) error) func(c *fiber.Ctx) error {
	return google.GoogleCallbackBuilder(callbackHandler)
}

func (ga *GoogleAuth) GoogleHandleCallback() func(c *fiber.Ctx) error {
	return google.HandleGoogleCallback
}

func (ga *GoogleAuth) GetSessionData() func(c *fiber.Ctx) error {
	return google.GetSessionData
}

func (ga *GoogleAuth) RequireGoogleAuth() func(c *fiber.Ctx) error {
	return google.RequireGoogleAuth
}

func (ga *GoogleAuth) IsGoogleAuthenticated(c *fiber.Ctx) bool {
	return google.IsGoogleAuthenticated(c)
}
