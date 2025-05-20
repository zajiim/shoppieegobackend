package routes

import (
	orderController "fiber-mongo-api/controllers/orders"
	"fiber-mongo-api/middlewares"

	"github.com/gofiber/fiber/v2"
)

func OrderRoutes(app *fiber.App) {
	app.Post("/api/create-order", middlewares.AuthMiddleware, orderController.CreateOrder)
	app.Post("/api/verify-payment", middlewares.AuthMiddleware, orderController.VerifyPayment)
	// app.Get("/api/get-orders-processing", middlewares.AuthMiddleware, orderController.GetProcessingOrders)
	// app.Get("/api/get-orders-delivered", middlewares.AuthMiddleware, orderController.GetDeliveredOrders)
	// app.Get("/api/get-orders-cancelled", middlewares.AuthMiddleware, orderController.GetCancelledOrders)
	app.Get("/api/get-orders", middlewares.AuthMiddleware, orderController.GetOrders)
	app.Get("/api/get-order", middlewares.AuthMiddleware, orderController.GetOrderById)
}
