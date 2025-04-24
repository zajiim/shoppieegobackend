package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OrderItem represents a single item in an order
type OrderItem struct {
	ProductID primitive.ObjectID `json:"productId" bson:"productId"`
	Product   Product     `json:"product" bson:"product"`
	Quantity  int                `json:"quantity" bson:"quantity"`
	Size      string             `json:"size" bson:"size,omitempty"`
}

// Order represents a customer order
type Order struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	AddressID     primitive.ObjectID `json:"addressId" bson:"addressId"`
	Items         []OrderItem        `json:"items" bson:"items"`
	TotalAmount   float64            `json:"totalAmount" bson:"totalAmount"`
	Status        string             `json:"status" bson:"status"`               // pending, processing, shipped, delivered, cancelled
	PaymentStatus string             `json:"paymentStatus" bson:"paymentStatus"` // pending, completed, failed
	RazorpayID    string             `json:"razorpayId" bson:"razorpayId"`
	PaymentID     string             `json:"paymentId,omitempty" bson:"paymentId,omitempty"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
}
