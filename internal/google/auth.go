package google

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	// Configure OAuth2 config
	googleOauthConfig = &oauth2.Config{
		ClientID:     config.DefaultConfig().GOOGLE_CLIENT_ID,
		ClientSecret: config.DefaultConfig().GOOGLE_CLIENT_SECRET,
		RedirectURL:  config.DefaultConfig().GOOGLE_AUTH_CALLBACK_URL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	oauthStateString = "DfknsCaCoffeeCodesdsanlnjn"
	Store            *session.Store
)

type UserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	ID            string `json:"id"`
}

func init() {
	// Register types for gob encoding
	gob.Register(map[string]interface{}{})
	gob.Register(&UserInfo{})
}

func InitializeStore(cookieExp time.Duration, cookieDomain string, cookieSecure bool, oauthState string) {
	Store = session.New(session.Config{
		KeyLookup:    "cookie:karma_google_session",
		Expiration:   cookieExp,
		CookiePath:   "/",
		CookieSecure: cookieSecure,
		CookieName:   "karma_google_session",
		CookieDomain: cookieDomain,
	})
	oauthStateString = oauthState
}

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func GetGoogleOauthURL() string {
	return googleOauthConfig.AuthCodeURL(oauthStateString)
}

func HandleGoogleLogin(c *fiber.Ctx) error {
	return c.Status(fiber.StatusTemporaryRedirect).JSON(ResponseHTTP{
		Success: true,
		Data:    GetGoogleOauthURL(),
		Message: "Redirecting to Google OAuth",
	})
}

func AuthBuilder(authHandler func(c *fiber.Ctx) error) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Verify state
		state := c.Query("state")
		if state != oauthStateString {
			return c.Status(fiber.StatusUnauthorized).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid OAuth state",
				Data:    nil,
			})
		}

		// Get authorization code
		code := c.Query("code")

		// Exchange authorization code for token
		token, err := googleOauthConfig.Exchange(c.Context(), code)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Code exchange failed",
				Data:    nil,
			})
		}

		// Get user info
		userInfo, err := getUserInfo(token.AccessToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to get user info",
				Data:    nil,
			})
		}

		sess, err := Store.Get(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Session error",
				Data:    nil,
			})
		}

		sess.Set("user", userInfo)
		if err := sess.Save(); err != nil {
			fmt.Printf("Save error: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
				Success: false,
				Message: "Failed to save session",
				Data:    nil,
			})
		}

		return authHandler(c)
	}
}

func GoogleCallbackBuilder(callbackHandler func(c *fiber.Ctx) error) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Verify state
		state := c.Query("state")
		if state != oauthStateString {
			return c.Status(fiber.StatusUnauthorized).JSON(ResponseHTTP{
				Success: false,
				Message: "Invalid OAuth state",
				Data:    nil,
			})
		}

		// Get authorization code
		code := c.Query("code")

		// Exchange authorization code for token
		token, err := googleOauthConfig.Exchange(c.Context(), code)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Code exchange failed")
		}

		// Get user info
		userInfo, err := getUserInfo(token.AccessToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to get user info")
		}

		sess, err := Store.Get(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Session error")
		}

		sess.Set("user", userInfo)
		if err := sess.Save(); err != nil {
			fmt.Printf("Save error: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to save session")
		}

		return callbackHandler(c)
	}
}

func HandleGoogleCallback(c *fiber.Ctx) error {
	// Verify state
	state := c.Query("state")
	if state != oauthStateString {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid OAuth state")
	}

	// Get authorization code
	code := c.Query("code")

	// Exchange authorization code for token
	token, err := googleOauthConfig.Exchange(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
			Success: false,
			Message: "Code exchange failed",
			Data:    nil,
		})
	}

	// Get user info
	userInfo, err := getUserInfo(token.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
			Success: false,
			Message: "Failed to get user info",
			Data:    nil,
		})
	}

	sess, err := Store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
			Success: false,
			Message: "Session error",
			Data:    nil,
		})
	}

	sess.Set("user", userInfo)
	if err := sess.Save(); err != nil {
		fmt.Printf("Save error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseHTTP{
			Success: false,
			Message: "Failed to save session",
			Data:    nil,
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(ResponseHTTP{
		Success: true,
		Message: "Authenticated",
		Data:    nil,
	})
}

// Helper function to get user info from Google
func getUserInfo(accessToken string) (*UserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo UserInfo
	if err = json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func IsGoogleAuthenticated(c *fiber.Ctx) bool {
	sess, err := Store.Get(c)
	if err != nil {
		return false
	}

	user := sess.Get("user")
	if user == nil {
		return false
	}

	return true
}

func RequireGoogleAuth(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/auth/google")
	}

	user := sess.Get("user")
	if user == nil {
		return c.Redirect("/auth/google")
	}

	return c.Next()
}
