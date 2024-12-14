package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Product struct {
	// ProductID   string   `bson:"productId" json:"productId" validate:"required,uuid4"`
	ID            primitive.ObjectID `json:"productId,omitempty" bson:"_id,omitempty"`
	Name          string             `bson:"name" json:"name" validate:"required"`
	Brand         string             `bson:"brand" json:"brand" validate:"required"`
	Description   string             `bson:"description" json:"description" validate:"required"`
	Quantity      int                `bson:"quantity" json:"quantity" validate:"required,min=1"`
	Price         float64            `bson:"price" json:"price" validate:"required,gt=0"`
	Category      string             `bson:"category" json:"category" validate:"required"`
	Images        []string           `bson:"images" json:"images" validate:"required,min=1,dive"`
	InCart        bool               `bson:"inCart" json:"inCart"`
	CartItemCount int                `bson:"cartItemCount,omitempty" json:"cartItemCount,omitempty"`
	Size          string             `bson:"size,omitempty" json:"size,omitempty"`
}
