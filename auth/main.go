package auth

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MelloB1989/karma/apis/twilio"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/google"
	"github.com/MelloB1989/karma/mails"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/golang-jwt/jwt"
)

type User struct {
	Id          string                 `json:"id"`
	Email       string                 `json:"email"`
	Phone       string                 `json:"phone"`
	Password    string                 `json:"password"`
	AddedClaims map[string]interface{} `json:"additional_claims"`
}

type LoginWithEmailAndPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type JWTClaimsProvider interface {
	AdditionalClaims() map[string]interface{}
	SetAdditionalClaims(claims map[string]interface{})
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
	return u.Password
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
	return u.AddedClaims
}

func (u *User) SetAdditionalClaims(claims map[string]interface{}) {
	u.AddedClaims = claims
}

func NewAuthUserPhone(phone, password, id string) AuthUserPhone {
	return &User{
		Phone:    phone,
		Password: password,
		Id:       id,
		Email:    "",
	}
}

func NewAuthUserEmail(email, password, id string) AuthUserEmail {
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
		} else {
			responseData["test_phone"] = false
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

// Email OTP request payloads
type LoginWithEmailOTPRequest struct {
	Email string `json:"email"`
}

type VerifyEmailOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

// LoginWithEmailOTPHandler initiates an OTP flow by generating an OTP,
// storing it in Redis (5 min expiration), and emailing it to the user.
func LoginWithEmailOTPHandler(getUserByEmail func(email string) (AuthUserEmail, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		req := new(LoginWithEmailOTPRequest)
		if err := c.BodyParser(req); err != nil {
			log.Printf("Body parsing error (email otp send): %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}

		if !utils.IsValidEmail(req.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid email format.",
				Data:    nil,
			})
		}

		cfg := config.DefaultConfig()
		if cfg == nil {
			log.Println("Configuration is nil (email otp send).")
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Configuration error.",
				Data:    nil,
			})
		}

		// Determine if account exists
		accountExists := false
		user, err := getUserByEmail(req.Email)
		if err == nil && user != nil && user.GetEmail() != "" {
			accountExists = true
		} else if err != nil {
			log.Printf("Error retrieving user by email (%s): %v", req.Email, err)
		}

		// Generate and store OTP
		otp := utils.GenerateOTP()
		key := "email_otp:" + strings.ToLower(req.Email)

		redisClient := utils.RedisConnect()
		if err := redisClient.Set(context.Background(), key, otp, 5*time.Minute).Err(); err != nil {
			log.Printf("Failed storing OTP in Redis for email (%s): %v", req.Email, err)
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to process OTP.",
				Data:    nil,
			})
		}

		// Send OTP email
		mailer := mails.NewKarmaMail(config.GetEnvRaw("KARMA_AUTH_MAILER"), "AWS_SES")
		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<title>Karma Auth Verification Code</title>
<meta name="viewport" content="width=device-width,initial-scale=1" />
<style>
body {margin:0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;background:#f5f7fb;color:#222;}
.container {max-width:520px;margin:40px auto;background:#ffffff;border-radius:12px;box-shadow:0 4px 18px rgba(0,0,0,0.08);overflow:hidden;}
.header {background:linear-gradient(135deg,#6a4dfd,#8f73ff);padding:28px 32px;color:#fff;}
.header h1 {margin:0;font-size:24px;letter-spacing:.5px;}
.brand-badge {display:inline-block;margin-top:8px;padding:4px 10px;border:1px solid rgba(255,255,255,0.4);border-radius:20px;font-size:12px;letter-spacing:1px;text-transform:uppercase;}
.content {padding:32px;}
.code-box {text-align:center;background:#f2f7ff;border:1px dashed #b2c4ff;border-radius:10px;padding:28px 10px;margin:28px 0;}
.code {font-size:40px;letter-spacing:6px;font-weight:600;font-family:'SFMono-Regular',Menlo,monospace;color:#4b32d3;}
.meta {font-size:13px;color:#555;}
.footer {padding:22px 32px;background:#fafbfe;font-size:12px;color:#6b7280;text-align:center;line-height:1.5;}
a {color:#6a4dfd;text-decoration:none;}
</style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Karma Auth</h1>
      <div class="brand-badge">Secure Login</div>
    </div>
    <div class="content">
      <p style="margin-top:0;">Hi,</p>
      <p>Use the verification code below to complete your sign in. This code is valid for <strong>5 minutes</strong>.</p>
      <div class="code-box">
        <div class="code">%s</div>
      </div>
      <p class="meta">If you did not request this code you can ignore this email — your account is still safe.</p>
      <p class="meta">For security reasons, never share this code with anyone.</p>
    </div>
    <div class="footer">
      Sent by Karma Auth • Empowering secure experiences<br/>
      Need help? <a href="mailto:support@karmaauth.example">Contact Support</a>
    </div>
  </div>
</body>
</html>`, otp)
		textBody := "Karma Auth verification code: " + otp + " (valid for 5 minutes). If you did not request this, ignore this email."
		if err := mailer.SendSingleMail(models.SingleEmailRequest{
			To: req.Email,
			Email: models.Email{
				Subject: "Your Login OTP",
				Body: models.EmailBody{
					Text: textBody,
					HTML: htmlBody,
				},
			},
		}); err != nil {
			log.Printf("Failed sending OTP email to (%s): %v", req.Email, err)
			// Clean up Redis key on failure to send
			_ = redisClient.Del(context.Background(), key).Err()
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to send OTP.",
				Data:    nil,
			})
		}

		return c.Status(fiber.StatusOK).JSON(ResponseHTTP{
			Success: true,
			Message: "OTP sent to email.",
			Data: map[string]bool{
				"account_exists": accountExists,
			},
		})
	}
}

// VerifyEmailOTPHandler verifies the OTP sent to email and returns a JWT.
// If the user exists, user-specific claims are added.
func VerifyEmailOTPHandler(getUserByEmail func(email string) (AuthUserEmail, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		req := new(VerifyEmailOTPRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}

		if !utils.IsValidEmail(req.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid email format.",
				Data:    nil,
			})
		}

		if len(req.OTP) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "OTP is required.",
				Data:    nil,
			})
		}

		cfg := config.DefaultConfig()
		if cfg == nil {
			log.Println("Configuration is nil (email otp verify).")
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Configuration error.",
				Data:    nil,
			})
		}

		key := "email_otp:" + strings.ToLower(req.Email)
		redisClient := utils.RedisConnect()
		storedOTP, err := redisClient.Get(context.Background(), key).Result()
		if err != nil || storedOTP == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid or expired OTP.",
				Data:    nil,
			})
		}

		if storedOTP != req.OTP {
			return c.Status(fiber.StatusBadRequest).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid OTP.",
				Data:    nil,
			})
		}

		// OTP valid - delete key to enforce one-time use
		_ = redisClient.Del(context.Background(), key).Err()

		accountExists := false
		var user AuthUserEmail
		user, err = getUserByEmail(req.Email)
		if err == nil && user != nil && user.GetEmail() != "" {
			accountExists = true
		}

		claims := jwt.MapClaims{
			"email": req.Email,
			"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		}

		if accountExists && user != nil {
			claims["uid"] = user.GetID()
			if claimsProvider, ok := user.(JWTClaimsProvider); ok {
				for k, v := range claimsProvider.AdditionalClaims() {
					claims[k] = v
				}
			}
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			log.Printf("Failed to sign JWT (email otp verify) for email (%s): %v", req.Email, err)
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to create token.",
				Data:    nil,
			})
		}

		return c.Status(fiber.StatusOK).JSON(ResponseHTTP{
			Success: true,
			Message: "OTP verified.",
			Data: map[string]interface{}{
				"account_exists": accountExists,
				"token":          tokenString,
			},
		})
	}
}

type GoogleAuth struct {
	config models.GoogleConfig
}

type GoogleCallbackData struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	ID            string `json:"id"`
}

func NewGoogleAuth(config models.GoogleConfig) *GoogleAuth {
	return &GoogleAuth{
		config: config,
	}
}

func InitializeGoogleAuth(config models.GoogleConfig) *GoogleAuth {
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
	return google.HandleGoogleCallback(&ga.config)
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
