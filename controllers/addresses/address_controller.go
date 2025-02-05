package addressController

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

var addressCollection *mongo.Collection = configs.GetCollection(configs.DB, "addresses")

func AddAddress(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var reqBody struct {
		StreetAddress string `json:"streetAddress" validate:"required"`
		City          string `json:"city" validate:"required"`
		State         string `json:"state" validate:"required"`
		ZipCode       string `json:"zipCode" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Result:  nil,
		})
	}

	userId := c.Locals("userId").(string)
	userObjId, err := primitive.ObjectIDFromHex(userId)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID",
			Result:  nil,
		})
	}

	newAddress := models.Address{
		Id:            primitive.NewObjectID(),
		UserId:        userObjId,
		StreetAddress: reqBody.StreetAddress,
		City:          reqBody.City,
		State:         reqBody.State,
		ZipCode:       reqBody.ZipCode,
	}

	_, err = addressCollection.InsertOne(ctx, newAddress)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error adding address",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Address added successfully",
		Result:  nil,
	})

}

func GetAddresses(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userId := c.Locals("userId").(string)
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID",
			Result:  nil,
		})
	}

	// var addresses []models.Address
	addresses := make([]models.Address, 0)
	cursor, err := addressCollection.Find(ctx, primitive.M{"userId": userObjId})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching addresses",
			Result: &fiber.Map{
				"addresses": addresses,
			},
		})
	}
	defer cursor.Close(ctx)
	// Decode addresses
	for cursor.Next(ctx) {
		var address models.Address
		if err := cursor.Decode(&address); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
				Status:  fiber.StatusInternalServerError,
				Message: "Error decoding addresses",
				Result: &fiber.Map{
					"addresses": addresses,
				},
			})
		}
		addresses = append(addresses, address)
	}

	// Return addresses
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Addresses fetched successfully",
		Result: &fiber.Map{
			"addresses": addresses,
		},
	})
}

func DeleteAddress(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get address ID from query parameter
	addressId := c.Query("id")
	if addressId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Address ID is required",
			Result:  nil,
		})
	}

	// Convert string ID to ObjectID
	objId, err := primitive.ObjectIDFromHex(addressId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid address ID format",
			Result:  nil,
		})
	}

	// Get user ID from JWT token
	userId := c.Locals("userId").(string)
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID",
			Result:  nil,
		})
	}

	// Find and delete address ensuring it belongs to the user
	result, err := addressCollection.DeleteOne(ctx, bson.M{
		"_id":    objId,
		"userId": userObjId,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error deleting address",
			Result:  nil,
		})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
			Status:  fiber.StatusNotFound,
			Message: "Address not found or you don't have permission to delete it",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Address deleted successfully",
		Result:  nil,
	})
}

func EditAddress(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addressId := c.Query("id")
	addressObjId, err := primitive.ObjectIDFromHex(addressId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid address ID format",
			Result:  nil,
		})
	}

	var reqBody struct {
		StreetAddress string `json:"streetAddress" validate:"required"`
		City          string `json:"city" validate:"required"`
		State         string `json:"state" validate:"required"`
		ZipCode       string `json:"zipCode" validate:"required"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Result:  nil,
		})
	}

	userId := c.Locals("userId").(string)
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID",
			Result:  nil,
		})
	}

	update := bson.M{
		"streetAddress": reqBody.StreetAddress,
		"city":          reqBody.City,
		"state":         reqBody.State,
		"zipCode":       reqBody.ZipCode,
	}

	filter := bson.M{"_id": addressObjId, "userId": userObjId}
	result, err := addressCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error updating address",
			Result:  nil,
		})
	}

	if result.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
			Status:  fiber.StatusNotFound,
			Message: "Address not found or you don't have permission to update it",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Address updated successfully",
		Result:  nil,
	})
}
