package routes

import (
	controllers "fiber-mongo-api/controllers/products"
	"fiber-mongo-api/middlewares"

	"github.com/gofiber/fiber/v2"
)

func ProductsRoute(app *fiber.App) {
	app.Get("/api/get-all-products", middlewares.AuthMiddleware, controllers.GetAllProducts)

	//For admin add-product
	app.Post("/api/admin/add-product", middlewares.AuthMiddleware, controllers.AddProduct)

	//Search products with name
	app.Get("/api/search", controllers.SearchProducts)

	//Get popular products based on brandName
	app.Get("api/popularBrand", controllers.GetPopularProducts)

	//Fetch productDetails
	app.Get("api/details", controllers.FetchProductDetails)
}
