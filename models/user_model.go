package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	Id       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name     string             `bson:"name" json:"name" validate:"required"`
	Email    string             `bson:"email" json:"email" validate:"required,email"`
	ImageUrl string             `bson:"profileImage" json:"profileImage,omitempty"`
	Password string             `bson:"password" json:"password" validate:"required,min=8"`
	Address  string             `bson:"address,omitempty" json:"address,omitempty"`
	Type     string             `bson:"type,omitempty" json:"type,omitempty" validate:"required,oneof=user admin"`
	Cart     []CartItem         `bson:"cart" json:"cart"`
}

type CartItem struct {
	Product  Product `bson:"product" json:"product" validate:"required"`
	Quantity int     `bson:"quantity" json:"quantity" validate:"required,min=1"`
}
