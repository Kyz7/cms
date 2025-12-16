package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
)

func GenerateRefreshToken(userID uint) (string, error) {
	rawToken := RandomString(64)
	hash := HashToken(rawToken)

	rt := models.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}

	if err := database.DB.Create(&rt).Error; err != nil {
		return "", err
	}

	return rawToken, nil
}

func ValidateRefreshToken(userID uint, token string) bool {
	hash := HashToken(token)

	result := database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND token_hash = ? AND revoked = false", userID, hash).
		Update("revoked", true)

	return result.RowsAffected == 1
}

func RefreshTokenPair(userID uint, oldToken string) (string, string, error) {
	if !ValidateRefreshToken(userID, oldToken) {
		return "", "", fmt.Errorf("invalid or expired refresh token")
	}

	var user models.User
	if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
		return "", "", fmt.Errorf("user not found")
	}

	accessToken, err := GenerateJWT(user.ID, user.Role.Name)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %v", err)
	}

	newRefreshToken, err := GenerateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %v", err)
	}

	return accessToken, newRefreshToken, nil
}

func RandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[num.Int64()]
	}
	return string(result)
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
