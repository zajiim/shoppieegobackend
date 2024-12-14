package routes

import (
	controllers "fiber-mongo-api/controllers/user"

	"github.com/gofiber/fiber/v2"
)

func UserRoute(app *fiber.App) {
	app.Post("/api/signup", controllers.UserSignUp)
	app.Post("/api/signin", controllers.UserSignIn)

}
