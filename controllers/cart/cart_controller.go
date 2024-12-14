package cartController

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
)

var userCollection = configs.GetCollection(configs.ConnectDB(), "users")

var productCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "products")

type AddToCartRequest struct {
	ProductID string `json:"id" validate:"required"`
	Region    string `json:"region" validate:"required,oneof=EU US UK"`
	Size      int    `json:"size" validate:"required"`
}

func AddtoCart(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request AddToCartRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request",
			Result:  nil,
		})
	}

	productID, err := primitive.ObjectIDFromHex(request.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid product Id",
			Result:  nil,
		})
	}

	//Validate region and size of the product
	sizeMap := map[string]map[int]string{
		"EU": {38: "S", 39: "M", 40: "L", 41: "XL", 42: "XXL", 43: "XXXL"},
		"US": {5: "S", 6: "M", 7: "L", 8: "XL", 9: "XXL", 10: "XXXL"},
		"UK": {4: "S", 5: "M", 6: "L", 7: "XL", 8: "XXL", 9: "XXXL"},
	}

	sizeCategory, valid := sizeMap[request.Region][request.Size]
	if !valid {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid size and region",
			Result:  nil,
		})
	}

	var product models.Product
	if err := productCollection.FindOne(ctx, bson.M{"_id": productID}).Decode(&product); err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
				Status:  fiber.StatusNotFound,
				Message: "Product not found",
				Result:  nil,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching product details",
			Result:  nil,
		})
	}

	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
		})
	}

	// Convert userId to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid User ID format",
		})
	}

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "User not found",
			Result:  nil,
		})
	}

	// Check if the product is already in the cart
	found := false
	for i, cartItem := range user.Cart {
		if cartItem.Product.ID == productID && cartItem.Product.Size == sizeCategory {
			// Product is found, increment quantity
			user.Cart[i].Quantity += 1
			found = true
			break
		}
	}
	// If the product was not found, add it to the cart with a quantity of 1
	if !found {
		product.InCart = true
		product.Size = sizeCategory
		user.Cart = append(user.Cart, models.CartItem{
			Product:  product,
			Quantity: 1,
		})
	}

	// Update the user in the database
	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": userObjectID}, bson.M{"$set": bson.M{"cart": user.Cart}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update cart",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully added to cart",
		Result: &fiber.Map{
			"status":    "success",
			"cartCount": len(user.Cart),
		},
	})

}

type AddToCartFromCartRequest struct {
	ProductID string `json:"id" validate:"required"`
	Size      string `json:"size" validate:"required"`
}

func AddToCartFromCart(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request AddToCartFromCartRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request",
			Result:  nil,
		})
	}

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
			Message: "Invalid User ID format",
		})
	}

	productID, err := primitive.ObjectIDFromHex(request.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid product Id",
			Result:  nil,
		})
	}

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "User not found",
			Result:  nil,
		})
	}

	found := false
	for i, cartItem := range user.Cart {
		if cartItem.Product.ID == productID && cartItem.Product.Size == request.Size {
			user.Cart[i].Quantity += 1
			found = true
			break
		}
	}

	if !found {
		return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
			Status:  fiber.StatusNotFound,
			Message: "Product with specified size not found in cart",
			Result:  nil,
		})
	}

	// Update the user's cart in the database
	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": userObjectID}, bson.M{"$set": bson.M{"cart": user.Cart}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update cart",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully added to cart",
		Result: &fiber.Map{
			"status":    "success",
			"cartCount": len(user.Cart),
		},
	})

}

type RemoveFromCartRequest struct {
	ProductID string `json:"id" validate:"required"`
}

func RemoveFromCart(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request RemoveFromCartRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request",
			Result:  nil,
		})
	}

	productID, err := primitive.ObjectIDFromHex(request.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid product Id",
			Result:  nil,
		})
	}

	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
		})
	}

	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid User ID format",
		})
	}

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "User not found",
			Result:  nil,
		})
	}

	for i, cartItem := range user.Cart {
		if cartItem.Product.ID == productID {
			user.Cart = append(user.Cart[:i], user.Cart[i+1:]...)
			break
		}
	}

	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": userObjectID}, bson.M{"$set": bson.M{"cart": user.Cart}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update cart",
			Result:  nil,
		})
	}

	// Set the InCart flag to false in the product collection
	_, err = productCollection.UpdateOne(ctx, bson.M{"_id": productID}, bson.M{"$set": bson.M{"inCart": false}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update product status",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully removed from cart",
		Result: &fiber.Map{
			"status":    "success",
			"cartCount": len(user.Cart),
		},
	})

}

//Decrement an item from cart

type DecrementFromCartRequest struct {
	ProductID string `json:"id" validate:"required"`
}

func DecrementFromCart(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var request DecrementFromCartRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request",
			Result:  nil,
		})
	}

	productID, err := primitive.ObjectIDFromHex(request.ProductID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid product id",
			Result:  nil,
		})
	}

	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "UserId is not found in the token",
			Result:  nil,
		})
	}

	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid User ID format",
			Result:  nil,
		})
	}

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "User not found",
			Result:  nil,
		})
	}

	for i, cartItem := range user.Cart {
		if cartItem.Product.ID == productID {
			if cartItem.Quantity > 1 {
				user.Cart[i].Quantity -= 1
			} else {
				user.Cart = append(user.Cart[:i], user.Cart[i+1:]...)
			}
			break
		}
	}

	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": userObjectID}, bson.M{"$set": bson.M{"cart": user.Cart}})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update cart",
			Result:  nil,
		})
	}

	// Optionally, update the product's inCart flag if it's completely removed
	productStillInCart := false
	for _, cartItem := range user.Cart {
		if cartItem.Product.ID == productID {
			productStillInCart = true
			break
		}
	}

	if !productStillInCart {
		_, err = productCollection.UpdateOne(ctx, bson.M{"_id": productID}, bson.M{"$set": bson.M{"inCart": false}})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
				Status:  fiber.StatusInternalServerError,
				Message: "Failed to update product status",
				Result:  nil,
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully removed 1 item from cart",
		Result: &fiber.Map{
			"status":    "success",
			"cartCount": len(user.Cart),
		},
	})

}

func GetAllCarts(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve and validate the user ID from Locals
	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
			Result:  nil,
		})
	}

	// Convert userId to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid User ID format",
		})
	}

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

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "User not found",
			Result:  nil,
		})
	}

	totalCartItems := int64(len(user.Cart))
	totalPages := (totalCartItems + limit - 1) / limit

	// Calculate start and end indexes for the slice
	start := (page - 1) * limit
	end := start + limit
	if start > totalCartItems {
		start = totalCartItems
	}
	if end > totalCartItems {
		end = totalCartItems
	}

	// Paginate the cart items
	paginatedCartItems := user.Cart[start:end]
	// Modify the product object to include cartItemCount
	for i := range paginatedCartItems {
		paginatedCartItems[i].Product.CartItemCount = paginatedCartItems[i].Quantity
	}

	status := "success"
	if len(paginatedCartItems) == 0 {
		status = "no more items"
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully fetched cart items",
		Result: &fiber.Map{
			"status":         status,
			"currentPage":    page,
			"totalPages":     totalPages,
			"totalCartItems": totalCartItems,
			"cartItems":      paginatedCartItems,
		},
	})

}

// Get cart total
func GetCartTotals(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve and validate the user ID from Locals
	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
			Result:  nil,
		})
	}

	// Convert userId to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid User ID format",
		})
	}

	// Fetch the user
	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "User not found",
			Result:  nil,
		})
	}

	// Calculate the total price
	var totalPrice float64
	for _, cartItem := range user.Cart {
		totalPrice += float64(cartItem.Product.Price) * float64(cartItem.Quantity)
	}

	// Calculate platform fee (2% of the total item)
	platformFee := float64(totalPrice) * 0.002

	// Calculate the grand total
	grandTotal := totalPrice + platformFee

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Successfully calculated cart totals",
		Result: &fiber.Map{
			"totalPrice":  totalPrice,
			"platformFee": platformFee,
			"grandTotal":  grandTotal,
		},
	})
}
