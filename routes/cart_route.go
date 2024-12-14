package routes

import (
	cartController "fiber-mongo-api/controllers/cart"
	"fiber-mongo-api/middlewares"

	"github.com/gofiber/fiber/v2"
)

func CartRoutes(app *fiber.App) {
	app.Post("/api/add-to-cart", middlewares.AuthMiddleware, cartController.AddtoCart)

	app.Post("api/add-to-cart-from-cart", middlewares.AuthMiddleware, cartController.AddToCartFromCart)

	app.Post("/api/remove-from-cart", middlewares.AuthMiddleware, cartController.RemoveFromCart)

	app.Post("/api/decrement-from-cart", middlewares.AuthMiddleware, cartController.DecrementFromCart)

	app.Get("/api/fetchCartItems", middlewares.AuthMiddleware, cartController.GetAllCarts)

	app.Get("/api/getCartTotal", middlewares.AuthMiddleware, cartController.GetCartTotals)

}
