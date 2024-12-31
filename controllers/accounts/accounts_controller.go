package controllers

import (
	"context"
	"fiber-mongo-api/configs"
	"fiber-mongo-api/models"
	"fiber-mongo-api/responses"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var userCollection = configs.GetCollection(configs.ConnectDB(), "users")

func UpdateUserProfile(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
			Result:  nil,
		})
	}

	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid UserID format",
			Result:  nil,
		})
	}

	//Parse request body
	var reqBody struct {
		Name     string `json:"name" validate:"required"`
		ImageUrl string `json:"profileImage"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Result:  nil,
		})
	}
	// Check if the user exists
	var existingUser models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&existingUser)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
				Status:  fiber.StatusNotFound,
				Message: "User not found",
				Result:  nil,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching user data",
			Result:  nil,
		})
	}

	//update the user data
	update := bson.M{
		"name":         reqBody.Name,
		"profileImage": reqBody.ImageUrl,
	}

	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": userObjectID}, bson.M{"$set": update})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error updating user profile",
			Result:  nil,
		})
	}

	// Return success response
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Profile updated successfully",
		Result:  &fiber.Map{"data": update},
	})

}
