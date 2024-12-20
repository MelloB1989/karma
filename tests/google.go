package tests

import (
	"github.com/MelloB1989/karma/auth"
	"github.com/MelloB1989/karma/internal/google"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func GoogleAuth() {
	app := fiber.New()

	// Initialize the store
	googleAuth := auth.InitializeGoogleAuth(auth.GoogleConfig{
		CookieExpiration: 60 * 60 * 24 * 7,
		CookieDomain:     "localhost",
		CookieHTTPSOnly:  false,
		OAuthStateString: "dskkjdbskdjcjkn",
	})

	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOrigins:     "http://localhost:3000",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
	}))

	// Route to initiate Google OAuth login
	app.Get("/auth/google", googleAuth.GoogleLoginHandler())

	// Callback route after Google OAuth
	app.Get("/auth/callbacks/google", googleAuth.GoogleHandleCallback())

	// app.Get("/dashboard", google.RequireGoogleAuth, func(c *fiber.Ctx) error {
	// 	sess, _ := google.Store.Get(c) // Use auth.Store
	// 	user := sess.Get("user")
	// 	return c.JSON(user)
	// })
	app.Get("/dashboard", google.RequireGoogleAuth, googleAuth.GetSessionData())

	app.Listen(":3000")
}
