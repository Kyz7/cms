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
		jwtKey = []byte(secret)
	} else {
		jwtKey = []byte(secret)
	}
}

func ValidateJWTSecret() error {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}

	// Ensure jwtKey is synchronized with the validated secret
	// This fixes issues where init() might have run before .env was loaded
	jwtKey = []byte(secret)

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
		if len(jwtKey) == 0 {
			return nil, fmt.Errorf("jwtKey is empty")
		}
		return jwtKey, nil
	})
	if err != nil {
		fmt.Printf("DEBUG: ParseJWT validation failed: %v\n", err)
		return 0, err
	}
	if !token.Valid {
		fmt.Println("DEBUG: ParseJWT token invalid")
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		fmt.Println("DEBUG: ParseJWT claims type mismatch")
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
	// Case-insensitive lookup for 'viewer'
	if err := database.DB.Where("LOWER(name) = LOWER(?)", "viewer").First(&role).Error; err != nil {
		return 0, err
	}
	if role.ID == 0 {
		return 0, fmt.Errorf("viewer role found but ID is 0")
	}
	// Ensure viewer is marked as global
	if !role.IsGlobal {
		database.DB.Model(&models.Role{}).Where("id = ?", role.ID).Update("is_global", true)
	}
	return role.ID, nil
}
