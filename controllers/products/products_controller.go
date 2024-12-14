package controllers

import (
	"context"
	"fiber-mongo-api/configs"
	"fiber-mongo-api/models"
	"fiber-mongo-api/responses"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var productCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "products")

// getProducts
func GetAllProducts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	// page, err := strconv.Atoi(pageStr)
	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil {
		page = 1
	}

	// limit, err := strconv.Atoi(limitStr)
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 10
	}

	skip := (page - 1) * limit

	totalProducts, err := productCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error counting products",
			Result:  nil,
		})
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))

	//Find paginated products
	var products []models.Product
	cursor, err := productCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching products",
			Result:  nil,
		})
	}
	if err = cursor.All(ctx, &products); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error parsing products",
			Result:  nil,
		})
	}
	totalPages := (totalProducts + int64(limit) - 1) / int64(limit)

	//Determine response status
	status := "success"
	if len(products) == 0 {
		status = "no more products"
	}

	//Return response
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Fetched Products",
		Result: &fiber.Map{
			"status":        status,
			"currentPage":   page,
			"totalPages":    totalPages,
			"totalProducts": totalProducts,
			"products":      products,
		},
	})

}

// Only for admin
func AddProduct(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var product models.Product
	if err := c.BodyParser(&product); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Error parsing product data",
			Result:  nil,
		})
	}

	result, err := productCollection.InsertOne(ctx, product)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error inserting product",
			Result:  nil,
		})
	}

	insertedId := result.InsertedID.(primitive.ObjectID)
	product.ID = insertedId

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Product added successfully",
		Result: &fiber.Map{
			"product": product,
		},
	})
}

// Search a Product
func SearchProducts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	name := c.Query("name")
	//Todo: implement filter functionality
	// brand := c.Query("brand")
	// category := c.Query("category")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil {
		page = 1
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 10
	}

	skip := (page - 1) * limit

	filter := bson.M{}

	if name != "" {
		filter["name"] = bson.M{"$regex": name, "$options": "i"}
	}

	//Get total num of products available
	totalProducts, err := productCollection.CountDocuments(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error in counting products",
			Result:  nil,
		})
	}

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)

	var products []models.Product

	cursor, err := productCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error in fetching products",
			Result:  nil,
		})
	}

	//Parse the cursor to get all products with name
	if err = cursor.All(ctx, &products); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error parsing products",
			Result:  nil,
		})
	}

	totalPages := (totalProducts + limit - 1) / limit

	//If no products found
	if len(products) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
			Status:  fiber.StatusNotFound,
			Message: "No products found",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Products found",
		Result: &fiber.Map{
			"currentPage":   page,
			"totalPages":    totalPages,
			"totalProducts": totalProducts,
			"products":      products,
		},
	})
}

func GetPopularProducts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	brand := c.Query("brand")
	if brand == "" {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "brand is required",
			Result:  nil,
		})
	}

	// Filter products by brand and Popular Brands
	popularFilter := bson.M{"brand": brand, "category": "Popular Brands"}

	// Filter products by brand and New Arrivals
	newArrivalFilter := bson.M{"brand": brand, "category": "New Arrivals"}

	// Set a limit of 10 items
	findOptions := options.Find()
	findOptions.SetLimit(10)

	// Find popularProducts
	var popularProducts []models.Product
	popularCursor, err := productCollection.Find(ctx, popularFilter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching data",
			Result:  nil,
		})
	}

	if err = popularCursor.All(ctx, &popularProducts); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error parsing products",
			Result:  nil,
		})
	}

	// Find newArrivalProducts
	var newArrivalProducts []models.Product
	newArrivalCursor, err := productCollection.Find(ctx, newArrivalFilter, findOptions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching data",
			Result:  nil,
		})
	}

	if err = newArrivalCursor.All(ctx, &newArrivalProducts); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error parsing products",
			Result:  nil,
		})
	}

	//Return the success data
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully fetched data",
		Result: &fiber.Map{
			"popular":     popularProducts,
			"newArrivals": newArrivalProducts,
		},
	})

}
