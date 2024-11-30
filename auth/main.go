package auth

import (
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
)

type LoginWithEmailAndPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthUser interface {
	GetEmail() string
	GetPassword() string
	GetID() string
}

func LoginWithEmailAndPassword(getUserByEmail func(email string) (AuthUser, error)) func(c *fiber.Ctx) error {
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
