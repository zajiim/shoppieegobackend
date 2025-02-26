package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Address struct {
	Id             primitive.ObjectID `json:"id" bson:"_id"`
	UserId         primitive.ObjectID `json:"userId" bson:"userId"`
	StreetAddress  string             `json:"streetAddress" bson:"streetAddress"`
	City           string             `json:"city" bson:"city"`
	State          string             `json:"state" bson:"state"`
	ZipCode        string             `json:"zipCode" bson:"zipCode"`
	IsUserSelected bool               `json:"isUserSelected" bson:"isUserSelected"`
}
