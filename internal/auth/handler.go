package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(c *fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" || body.Email == "" || body.Password == "" {
		return response.ValidationError(c, map[string]string{
			"name":     "name is required",
			"email":    "email is required",
			"password": "password is required",
		})
	}

	var existing models.User
	if err := database.DB.Where("email = ?", body.Email).First(&existing).Error; err == nil {
		return response.Conflict(c, "Email already registered")
	}

	hashedPassword, _ := utils.HashPassword(body.Password)

	viewerRoleID, err := utils.GetDefaultViewerRoleID()
	if err != nil {
		return response.InternalError(c, "Failed to assign default role")
	}

	u := models.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: hashedPassword,
		Status:   "active",
		RoleID:   viewerRoleID,
	}

	if err := database.DB.Create(&u).Error; err != nil {
		return response.InternalError(c, "Failed to create user")
	}

	database.DB.Preload("Role").First(&u, u.ID)

	accessToken, _ := utils.GenerateJWT(u.ID, u.Role.Name)
	refreshToken, _ := utils.GenerateRefreshToken(u.ID)

	return response.Created(c, fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          u,
	}, "Registration successful")
}

func LoginHandler(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Email == "" || body.Password == "" {
		return response.ValidationError(c, map[string]string{
			"email":    "email is required",
			"password": "password is required",
		})
	}

	accessToken, refreshToken, err := LoginUser(body.Email, body.Password)
	if err != nil {
		return response.Unauthorized(c, "Invalid email or password")
	}

	return response.Success(c, fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    900,
	}, "Login successful")
}

func RefreshHandler(c *fiber.Ctx) error {
	var body struct {
		UserID       uint   `json:"user_id"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.UserID == 0 || body.RefreshToken == "" {
		return response.ValidationError(c, map[string]string{
			"user_id":       "user_id is required",
			"refresh_token": "refresh_token is required",
		})
	}

	accessToken, newRefreshToken, err := utils.RefreshTokenPair(body.UserID, body.RefreshToken)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}

	var user models.User
	database.DB.Preload("Role").First(&user, body.UserID)
	user.Password = ""

	return response.Success(c, fiber.Map{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"user":          user,
		"expires_in":    900,
	}, "Token refreshed successfully")
}

func LogoutHandler(c *fiber.Ctx) error {
	userIDInterface := c.Locals("user_id")
	if userIDInterface == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Invalid user ID format",
			},
		})
	}
	log.Printf("User %d logged out", userID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Logout successful",
		"data": fiber.Map{
			"user_id": userID,
		},
	})
}

func ForgotPasswordHandler(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Email == "" {
		return response.ValidationError(c, map[string]string{
			"email": "email is required",
		})
	}

	var user models.User
	if err := database.DB.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return response.Success(c, nil, "If account exists, reset link has been sent")
	}

	plainToken, tokenHash, err := generateSecureToken(32)
	if err != nil {
		return response.InternalError(c, "Failed to generate reset token")
	}

	reset := models.ResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := database.DB.Create(&reset).Error; err != nil {
		return response.InternalError(c, "Failed to save reset token")
	}

	resetURL := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", plainToken)
	msg := fmt.Sprintf("Subject: Password Reset\n\nClick here to reset: %s", resetURL)
	_ = smtp.SendMail("smtp.example.com:587",
		smtp.PlainAuth("", "your@email.com", "password", "smtp.example.com"),
		"your@email.com", []string{user.Email}, []byte(msg))

	return response.Success(c, nil, "If account exists, reset link has been sent")
}

func ResetPasswordHandler(c *fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Token == "" || body.NewPassword == "" {
		return response.ValidationError(c, map[string]string{
			"token":        "token is required",
			"new_password": "new_password is required",
		})
	}

	hash := sha256.Sum256([]byte(body.Token))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	var reset models.ResetToken
	if err := database.DB.Where("token_hash = ?", tokenHash).First(&reset).Error; err != nil {
		return response.BadRequest(c, "Invalid or expired token", nil)
	}

	if reset.ExpiresAt.Before(time.Now()) {
		database.DB.Delete(&reset)
		return response.BadRequest(c, "Token expired", nil)
	}

	var user models.User
	if err := database.DB.First(&user, reset.UserID).Error; err != nil {
		return response.NotFound(c, "User")
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	user.Password = string(hashedPassword)
	database.DB.Save(&user)

	database.DB.Delete(&reset)

	return response.Success(c, nil, "Password reset successful")
}

func generateSecureToken(n int) (string, string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := base64.URLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	return token, base64.URLEncoding.EncodeToString(hash[:]), nil
}
