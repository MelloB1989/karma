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
		req := new(LoginWithEmailAndPasswordRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Invalid request",
			})
		}

		user, err := getUserByEmail(req.Email)
		if err != nil || user == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Email does not exist",
			})
		}

		if utils.CheckPasswordHash(req.Password, user.GetPassword()) {
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["email"] = user.GetEmail()
			claims["uid"] = user.GetID()
			claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix()
			if claimsProvider, ok := user.(JWTClaimsProvider); ok {
				for key, value := range claimsProvider.AdditionalClaims() {
					claims[key] = value
				}
			}
			t, err := token.SignedString([]byte(config.DefaultConfig().JWTSecret))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"message": "Error signing token",
				})
			}
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"token": t,
			})
		}

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid credentials",
		})
	}
}

type LoginWithPhoneOTPRequest struct {
	Phone string `json:"phone"`
}

func LoginWithPhoneOTPHandler(getUserByPhone func(phone string) (AuthUserPhone, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		req := new(LoginWithPhoneOTPRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}
		if !utils.VerifyPhoneNumber(req.Phone) {
			return c.JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid Phone number format",
				Data:    nil,
			})
		}
		user, err := getUserByPhone(req.Phone)
		if err != nil || user.GetPhone() == "" {
			if utils.Contains(config.DefaultConfig().TestPhoneNumbers, req.Phone) {
				return c.JSON(ResponseHTTP{
					Success: true,
					Message: "OTP sent to phone number.",
					Data:    map[string]bool{"account_exists": false, "test_phone": true},
				})
			}
			twilio.SendOTP(req.Phone)
			// User does not exist, send back OTP
			return c.JSON(ResponseHTTP{
				Success: true,
				Message: "OTP sent to phone number.",
				Data:    map[string]bool{"account_exists": false},
			})
		} else {
			if utils.Contains(config.DefaultConfig().TestPhoneNumbers, req.Phone) {
				return c.JSON(ResponseHTTP{
					Success: true,
					Message: "OTP sent to phone number.",
					Data:    map[string]bool{"account_exists": false, "test_phone": true},
				})
			}
			twilio.SendOTP(req.Phone)
			// User exists, send back OTP
			return c.JSON(ResponseHTTP{
				Success: true,
				Message: "OTP sent to phone number.",
				Data:    map[string]bool{"account_exists": true},
			})
		}
	}
}

type VerifyPhoneOTPRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

func VerifyPhoneOTPHandler(getUserByPhone func(phone string) (AuthUserPhone, error)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		req := new(VerifyPhoneOTPRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to parse request body.",
				Data:    nil,
			})
		}
		if !utils.VerifyPhoneNumber(req.Phone) {
			return c.JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid Phone number format",
				Data:    nil,
			})
		}

		config := config.DefaultConfig()
		if config == nil {
			// Handle case where config is nil
			return c.Status(500).JSON(ResponseHTTP{
				Success: false,
				Message: "Configuration error.",
				Data:    nil,
			})
		}

		if twilio.VerifyOTP(req.OTP, req.Phone) || utils.Contains(config.TestPhoneNumbers, req.Phone) {
			exist := true
			user, err := getUserByPhone(req.Phone)
			if err != nil || user == nil {
				exist = false
				// Log or handle user retrieval error
			}

			claims := jwt.MapClaims{
				"phone": req.Phone,
				"exp":   time.Now().Add(time.Hour * 24 * 30).Unix(),
			}

			if user != nil {
				claims["uid"] = user.GetID()
				claims["phone"] = user.GetPhone()
				claims["exp"] = time.Now().Add(time.Hour * 24 * 30).Unix()
				if claimsProvider, ok := user.(JWTClaimsProvider); ok {
					for key, value := range claimsProvider.AdditionalClaims() {
						claims[key] = value
					}
				}
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			jwtSecret := []byte(config.JWTSecret)
			tokenString, err := token.SignedString(jwtSecret)
			if err != nil {
				log.Println("Failed to create token:", err)
				return c.JSON(ResponseHTTP{
					Success: false,
					Message: "Failed to create token.",
					Data:    nil,
				})
			}

			return c.JSON(ResponseHTTP{
				Success: true,
				Message: "OTP verified.",
				Data: map[string]interface{}{
					"account_exists": exist,
					"token":          tokenString,
				},
			})
		} else {
			return c.JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid OTP.",
				Data:    nil,
			})
		}
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
