package middlewares

import (
	"fiber-mongo-api/responses"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = os.Getenv("JWT_SECRET")

// AuthMiddleware
func AuthMiddleware(c *fiber.Ctx) error {
	// Extract the token from the Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "No auth token, access denied",
		})
	}

	// Check if the Authorization header starts with "Bearer "
	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid authorization header format",
		})
	}

	tokenString := bearerToken[1]

	// Parse and validate the token
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "Token verification failed, access denied",
		})
	}

	// Extract the user ID from the token claims
	userId, ok := (*claims)["id"].(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(responses.UserResponse{
			Status:  fiber.StatusUnauthorized,
			Message: "User ID not found in token",
		})
	}

	// Save the user ID to Locals for later use
	c.Locals("userId", userId)

	return c.Next()
}
