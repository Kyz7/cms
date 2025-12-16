package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var jwtKey []byte

func init() {
	if err := godotenv.Load(); err != nil {
		log.Default().Println("No .env file found")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test_secret_key_minimum_32_characters_long_for_testing_only"
	}

	jwtKey = []byte(secret)
}

func ValidateJWTSecret() error {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long (current: %d)", len(secret))
	}

	if secret == "test_secret_key_minimum_32_characters_long_for_testing_only" {
		return fmt.Errorf("cannot use default test secret in production")
	}

	return nil
}

func GenerateJWT(userID uint, roleName string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  strconv.Itoa(int(userID)),
		"role": roleName,
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseJWT(tokenStr string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}

func GetDefaultViewerRoleID() (uint, error) {
	var role models.Role
	if err := database.DB.Where("name = ?", "viewer").First(&role).Error; err != nil {
		return 0, err
	}
	if role.ID == 0 {
		return 0, fmt.Errorf("viewer role found but ID is 0")
	}
	return role.ID, nil
}
