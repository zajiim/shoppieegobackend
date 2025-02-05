package routes

import (
	addressController "fiber-mongo-api/controllers/addresses"
	"fiber-mongo-api/middlewares"

	"github.com/gofiber/fiber/v2"
)

func AddressRoutes(app *fiber.App) {
	app.Post("/api/add-address", middlewares.AuthMiddleware, addressController.AddAddress)
	app.Get("/api/get-addresses", middlewares.AuthMiddleware, addressController.GetAddresses)
	app.Delete("api/address", middlewares.AuthMiddleware, addressController.DeleteAddress)
	app.Put("api/edit-address", middlewares.AuthMiddleware, addressController.EditAddress)
}
