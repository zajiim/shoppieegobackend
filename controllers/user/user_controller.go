package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fiber-mongo-api/configs"
	"fiber-mongo-api/models"
	"fiber-mongo-api/responses"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = configs.GetCollection(configs.DB, "users")

var jwtSecret = os.Getenv("JWT_SECRET")

// UserSignUp
func UserSignUp(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var reqBody struct {
		Name            string `json:"name" validate:"required"`
		Email           string `json:"email" validate:"required,email"`
		Password        string `json:"password" validate:"required,min=8"`
		ConfirmPassword string `json:"confirmPassword" validate:"required,min=8"`
	}

	// Regular expression for validating email format
	var emailRegex = regexp.MustCompile(`^(([^<>()[\]\.,;:\s@\"]+(\.[^<>()[\]\.,;:\s@\"]+)*)|(\".+\"))@(([^<>()[\]\.,;:\s@\"]+\.)+[^<>()[\]\.,;:\s@\"]{2,})$`)

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Result:  nil,
		})
	}

	//Password validations
	if utf8.RuneCountInString(reqBody.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Passwords must be 8 letters long",
			Result:  nil,
		})
	}

	//Check if the passwords match
	if reqBody.Password != reqBody.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Passwords do not match",
			Result:  nil,
		})
	}

	// Check if email is valid using regex
	if !emailRegex.MatchString(reqBody.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Please enter a valid email address",
			Result:  nil,
		})
	}

	//Checks if the user already exists
	var existingUser models.User
	err := userCollection.FindOne(ctx, bson.M{"email": reqBody.Email}).Decode(&existingUser)

	// if err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
	// 		Status:  fiber.StatusBadRequest,
	// 		Message: "User with same email already exists",
	// 		Result:  nil,
	// 	})
	// }

	if err != nil && err != mongo.ErrNoDocuments {
		// If another error (not 'ErrNoDocuments') occurs, return internal server error
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error checking user existence",
			Result:  nil,
		})
	} else if err == nil {
		// If no error occurred, that means the user exists
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "User with same email already exists",
			Result:  nil,
		})
	}

	//Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(reqBody.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error hashing password",
			Result:  nil,
		})
	}

	//Create user object
	newUser := models.User{
		Id:       primitive.NewObjectID(),
		Name:     reqBody.Name,
		Email:    reqBody.Email,
		ImageUrl: "",
		Password: string(hashedPassword),
		Type:     "user",
		Cart:     []models.CartItem{},
	}

	//Insert into mongodb
	_, err = userCollection.InsertOne(ctx, newUser)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error in saving user, please try again later",
		})
	}

	//Return the created user
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "User created successfully",
		Result:  &fiber.Map{"data": newUser},
	})

}

func UserSignIn(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var reqBody struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Result:  nil,
		})
	}

	//Check if the user exists in the db
	var existingUser models.User

	err := userCollection.FindOne(ctx, bson.M{"email": reqBody.Email}).Decode(&existingUser)
	if err == mongo.ErrNoDocuments {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "User with this account does not exist",
			Result:  nil,
		})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching from database",
			Result:  nil,
		})
	}

	//Compare the password
	err = bcrypt.CompareHashAndPassword([]byte(existingUser.Password), []byte(reqBody.Password))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Incorrect password",
			Result:  nil,
		})
	}

	//Create JWT token
	token, err := createJwt(existingUser.Id.Hex())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error while generating jwt token",
			Result:  nil,
		})
	}

	//Response
	existingUser.Password = ""
	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "User signed in successfully",
		Result: &fiber.Map{
			"data": fiber.Map{
				"id":           existingUser.Id.Hex(),
				"name":         existingUser.Name,
				"profileImage": existingUser.ImageUrl,
				"email":        existingUser.Email,
				"password":     "",
				"type":         existingUser.Type,
				"cart":         existingUser.Cart,
				"token":        token,
			},
		},
	})
}

func createJwt(userId string) (string, error) {
	claims := jwt.MapClaims{
		"id":  userId,
		"exp": time.Now().Add(time.Hour * 720).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func OAuthLogin(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var reqBody struct {
		Provider string `json:"provider" validate:"required"`
		Token    string `json:"token" validate:"required"`
		// Type     string `json:"type" validate:"required,oneof=google facebook"`
	}

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid request format",
			Result:  nil,
		})
	}

	var userInfo map[string]string
	var err error
	switch reqBody.Provider {
	case "google":
		userInfo, err = ValidateGoogleToken(reqBody.Token)
	case "facebook":
		userInfo, err = ValidateFacebookToken(reqBody.Token)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(responses.UserResponse{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid provider",
			Result:  nil,
		})
	}

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid token",
			Result:  nil,
		})
	}

	email := userInfo["email"]
	name := userInfo["name"]
	profileImage := userInfo["profileImage"]

	var existingUser models.User
	err = userCollection.FindOne(ctx, bson.M{"email": email}).Decode(&existingUser)

	var jwtToken string

	if err == mongo.ErrNoDocuments {
		newUser := models.User{
			Id:       primitive.NewObjectID(),
			Name:     name,
			Email:    email,
			ImageUrl: profileImage,
			Cart:     []models.CartItem{},
		}
		_, err := userCollection.InsertOne(ctx, newUser)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
				Status:  fiber.StatusInternalServerError,
				Message: "Error creating user",
				Result:  nil,
			})
		}
		existingUser = newUser
		jwtToken, _ = createJwt(newUser.Id.Hex())
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error fetching user",
			Result:  nil,
		})
	} else {
		if existingUser.ImageUrl != profileImage {
			update := bson.M{"$set": bson.M{"profileImage": profileImage}}
			_, err = userCollection.UpdateOne(ctx, bson.M{"email": email}, update)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
					Status:  fiber.StatusInternalServerError,
					Message: "Error updating user profile image",
					Result:  nil,
				})
			}
			existingUser.ImageUrl = profileImage
		}
		// Existing user, generate token
		jwtToken, err = createJwt(existingUser.Id.Hex())
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(responses.UserResponse{
			Status:  fiber.StatusInternalServerError,
			Message: "Error generating token",
			Result:  nil,
		})

	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "User signed in successfully",
		Result: &fiber.Map{
			"data": fiber.Map{
				"id":           existingUser.Id.Hex(),
				"name":         existingUser.Name,
				"profileImage": existingUser.ImageUrl,
				"email":        existingUser.Email,
				"type":         existingUser.Type,
				"cart":         existingUser.Cart,
				"token":        jwtToken,
			},
		},
	})

}

func ValidateGoogleToken(token string) (map[string]string, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid Google token")
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	email, ok1 := data["email"].(string)
	name, ok2 := data["name"].(string)
	picture := data["picture"].(string)
	if !ok1 || !ok2 {
		return nil, errors.New("missing email or name ")
	}
	return map[string]string{
			"email":        email,
			"name":         name,
			"profileImage": picture,
		},
		nil
}

func ValidateFacebookToken(token string) (map[string]string, error) {
	resp, err := http.Get("https://graph.facebook.com/me?fields=id,name,email&access_token=" + token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid Facebook token")
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	email, ok1 := data["email"].(string)
	name, ok2 := data["name"].(string)
	if !ok1 || !ok2 {
		return nil, errors.New("missing email or name")
	}

	return map[string]string{"email": email, "name": name}, nil
}

func UserSignOut(c *fiber.Ctx) error {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "No auth token, access denied",
		})
	}

	return c.Status(fiber.StatusOK).JSON(responses.UserResponse{
		Status:  fiber.StatusOK,
		Message: "User signed out successfully",
	})

}
