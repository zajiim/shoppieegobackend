package routes

import (
	controllers "fiber-mongo-api/controllers/user"

	"github.com/gofiber/fiber/v2"
)

func UserRoute(app *fiber.App) {
	app.Post("/api/signup", controllers.UserSignUp)
	app.Post("/api/signin", controllers.UserSignIn)
	app.Post("/api/signout", controllers.UserSignOut)
	app.Post("/api/oauth", controllers.OAuthLogin)
}
