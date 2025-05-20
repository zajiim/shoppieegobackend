package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fiber-mongo-api/configs"
	"fiber-mongo-api/models"
	"fiber-mongo-api/responses"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/razorpay/razorpay-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var orderCollection *mongo.Collection = configs.GetCollection(configs.DB, "orders")
var userCollection = configs.GetCollection(configs.ConnectDB(), "users")
var addressCollection *mongo.Collection = configs.GetCollection(configs.DB, "addresses")

var razorpayKeyID = configs.EnvRazorpayKeyId()
var razorpayKeySecret = configs.EnvRazorpayKeySecret()

// CreateOrderRequest holds the data required to create an order
type CreateOrderRequest struct {
	AddressID string  `json:"addressId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

// VerifyPaymentRequest holds the data for payment verification
type VerifyPaymentRequest struct {
	OrderID    string `json:"orderId"`
	PaymentID  string `json:"paymentId"`
	Signature  string `json:"signature"`
	RazorpayID string `json:"razorpayId"`
}

func CreateOrder(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
			Result:  nil,
		})
	}

	// Parse request body
	var orderReq CreateOrderRequest
	if err := c.BodyParser(&orderReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Result:  nil,
		})
	}

	// Convert user ID to ObjectID
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID format",
			Result:  nil,
		})
	}

	// Convert address ID to ObjectID
	addressObjectID, err := primitive.ObjectIDFromHex(orderReq.AddressID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid address ID format",
			Result:  nil,
		})
	}

	// Fetch user's cart items
	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching user details",
			Result:  nil,
		})
	}

	// Ensure user has items in cart
	if len(user.Cart) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Cart is empty",
			Result:  nil,
		})
	}

	// Validate address belongs to user
	var address models.Address
	if err := addressCollection.FindOne(ctx, bson.M{
		"_id":    addressObjectID,
		"userId": userObjectID,
	}).Decode(&address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Address not found or doesn't belong to user",
			Result:  nil,
		})
	}

	// Convert cart items to order items
	var orderItems []models.OrderItem
	for _, cartItem := range user.Cart {
		orderItems = append(orderItems, models.OrderItem{
			ProductID: cartItem.Product.ID,
			Product:   cartItem.Product,
			Quantity:  cartItem.Quantity,
			Size:      cartItem.Product.Size,
		})
	}

	// Initialize Razorpay client
	client := razorpay.NewClient(razorpayKeyID, razorpayKeySecret)

	// Create order in Razorpay
	orderAmount := int64(orderReq.Amount * 100) // Convert to paise (Razorpay uses smallest currency unit)
	currency := "INR"
	if orderReq.Currency != "" {
		currency = orderReq.Currency
	}

	data := map[string]interface{}{
		"amount":   orderAmount,
		"currency": currency,
		"receipt":  "receipt_" + primitive.NewObjectID().Hex(),
	}

	razorpayOrder, err := client.Order.Create(data, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to create Razorpay order: " + err.Error(),
			Result:  nil,
		})
	}

	// Create order in database
	now := time.Now()
	order := models.Order{
		ID:            primitive.NewObjectID(),
		UserID:        userObjectID,
		AddressID:     addressObjectID,
		Items:         orderItems,
		TotalAmount:   orderReq.Amount,
		Status:        "pending",
		PaymentStatus: "pending",
		RazorpayID:    razorpayOrder["id"].(string),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err = orderCollection.InsertOne(ctx, order)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to create order in database",
			Result:  nil,
		})
	}

	// Return order details to client
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Order created successfully",
		Result: &fiber.Map{
			"orderId":    order.ID.Hex(),
			"razorpayId": razorpayOrder["id"],
			"amount":     razorpayOrder["amount"],
			"currency":   razorpayOrder["currency"],
			"key_id":     razorpayKeyID,
		},
	})
}

// VerifyPayment verifies the payment signature from Razorpay and updates order status
func VerifyPayment(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user ID from context
	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
			Result:  nil,
		})
	}

	// Parse request body
	var verifyReq VerifyPaymentRequest
	if err := c.BodyParser(&verifyReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request body",
			Result:  nil,
		})
	}

	// Verify the signature
	signature := verifyReq.Signature
	data := verifyReq.RazorpayID + "|" + verifyReq.PaymentID

	// Generate HMAC SHA256 signature
	h := hmac.New(sha256.New, []byte(razorpayKeySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Verify signature
	if signature != expectedSignature {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid payment signature",
			Result:  nil,
		})
	}

	// Update order status in database
	orderObjectID, err := primitive.ObjectIDFromHex(verifyReq.OrderID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid order ID format",
			Result:  nil,
		})
	}

	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID format",
			Result:  nil,
		})
	}

	// Update order with payment details
	update := bson.M{
		"$set": bson.M{
			"paymentStatus": "completed",
			"status":        "processing", // Change order status to processing after payment
			"paymentId":     verifyReq.PaymentID,
			"updatedAt":     time.Now(),
		},
	}

	result, err := orderCollection.UpdateOne(
		ctx,
		bson.M{"_id": orderObjectID, "userId": userObjectID},
		update,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to update order: " + err.Error(),
			Result:  nil,
		})
	}

	if result.MatchedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
			Status:  fiber.StatusNotFound,
			Message: "Order not found or doesn't belong to user",
			Result:  nil,
		})
	}

	// Clear user's cart after successful payment
	_, err = userCollection.UpdateOne(
		ctx,
		bson.M{"_id": userObjectID},
		bson.M{"$set": bson.M{"cart": []interface{}{}}},
	)

	if err != nil {
		// Log error but don't fail the request
		// The payment is successful, but cart couldn't be cleared
		// You might want to handle this in a background job
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Payment verified successfully",
		Result: &fiber.Map{
			"orderId":    verifyReq.OrderID,
			"paymentId":  verifyReq.PaymentID,
			"razorpayId": verifyReq.RazorpayID,
		},
	})
}

func GetOrders(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
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
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID format",
			Result:  nil,
		})
	}

	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")
	status := c.Query("status", "") // Optional status filter

	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil {
		page = 1
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 10
	}

	skip := (page - 1) * limit

	// Build query filter
	filter := bson.M{"userId": userObjectID}
	if status != "" {
		filter["status"] = status
	}

	// Count total orders for the user
	totalOrders, err := orderCollection.CountDocuments(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to count orders",
			Result:  nil,
		})
	}

	// Fetch paginated orders
	cursor, err := orderCollection.Find(ctx, filter, &options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to fetch orders",
			Result:  nil,
		})
	}
	defer cursor.Close(ctx)

	var orders []fiber.Map
	for cursor.Next(ctx) {
		var order models.Order
		if err := cursor.Decode(&order); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
				Status:  fiber.StatusInternalServerError,
				Message: "Failed to decode order",
				Result:  nil,
			})
		}

		// Simplify order items
		var simplifiedItems []fiber.Map
		for _, item := range order.Items {
			simplifiedItems = append(simplifiedItems, fiber.Map{
				"name":     item.Product.Name,
				"price":    item.Product.Price,
				"size":     item.Size,
				"quantity": item.Quantity,
				"image":    item.Product.Images[0], // Assuming the first image is required
			})
		}

		orders = append(orders, fiber.Map{
			"id":        order.ID.Hex(),
			"items":     simplifiedItems,
			"status":    order.Status,
			"total":     order.TotalAmount,
			"createdAt": order.CreatedAt,
		})
	}

	if err := cursor.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Cursor error",
			Result:  nil,
		})
	}

	totalPages := (totalOrders + limit - 1) / limit

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Orders fetched successfully",
		Result: &fiber.Map{
			"orders":      orders,
			"currentPage": page,
			"totalPages":  totalPages,
			"totalOrders": totalOrders,
		},
	})
}

func GetOrderById(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	userId, ok := c.Locals("userId").(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
			Result:  nil,
		})
	}

	orderId := c.Query("id")
	if orderId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Order ID is required",
			Result:  nil,
		})
	}

	orderObjectID, err := primitive.ObjectIDFromHex(orderId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid order ID format",
			Result:  nil,
		})
	}

	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid user ID format",
			Result:  nil,
		})
	}

	var order models.Order
	err = orderCollection.FindOne(ctx, bson.M{"_id": orderObjectID, "userId": userObjectID}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(responses.UserResponse{
				Status:  fiber.StatusNotFound,
				Message: "Order not found",
				Result:  nil,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Failed to fetch order",
			Result:  nil,
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "Order fetched successfully",
		Result: &fiber.Map{
			"order": order,
		},
	})
}
