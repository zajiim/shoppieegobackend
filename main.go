package main

import (
	"fiber-mongo-api/configs"
	"fiber-mongo-api/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	configs.ConnectDB()

	routes.CartRoutes(app)
	routes.UserRoute(app)
	routes.ProductsRoute(app)
	routes.AccountRoute(app)
	routes.AddressRoutes(app)

	app.Listen(":3000")
}
