package controllers

import (
	"context"
	"fiber-mongo-api/models"
	"time"

	"fiber-mongo-api/responses"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func FetchProductDetails(c *fiber.Ctx) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productId := c.Query("productId")

	// Convert the productId string to an ObjectID
	objectId, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid product ID format",
			Result:  nil,
		})
	}

	var product models.Product
	err = productCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&product)

	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
			Status:  fiber.StatusNotFound,
			Message: "Product not found",
			Result:  nil,
		})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching product details",
			Result:  nil,
		})
	}

	// If product is found, return the product details
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Product fetched successfully",
		Result: &fiber.Map{
			"status":  "success",
			"product": product,
		},
	})

}
