package routes

import (
	orderController "fiber-mongo-api/controllers/orders"
	"fiber-mongo-api/middlewares"

	"github.com/gofiber/fiber/v2"
)

func OrderRoutes(app *fiber.App) {
	app.Post("/api/create-order", middlewares.AuthMiddleware, orderController.CreateOrder)
	app.Post("/api/verify-payment", middlewares.AuthMiddleware, orderController.VerifyPayment)
}
