package routes

import (
	controllers "fiber-mongo-api/controllers/accounts"
	"fiber-mongo-api/middlewares"

	"github.com/gofiber/fiber/v2"
)

func AccountRoute(app *fiber.App) {

	app.Post("/api/update-profile", middlewares.AuthMiddleware, controllers.UpdateUserProfile)
	app.Get("/api/get-user-profile", middlewares.AuthMiddleware, controllers.GetUserProfile)
}
