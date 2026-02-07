package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email) && len(email) <= 254
}

func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hasUpper := false
	hasLower := false
	hasNumber := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return fmt.Errorf("password must contain uppercase, lowercase, and numbers")
	}

	return nil
}

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

	if !isValidEmail(body.Email) {
		return response.ValidationError(c, map[string]string{
			"email": "invalid email format",
		})
	}

	if err := validatePasswordStrength(body.Password); err != nil {
		return response.ValidationError(c, map[string]string{
			"password": err.Error(),
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

	if !isValidEmail(body.Email) {
		return response.ValidationError(c, map[string]string{
			"email": "invalid email format",
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

	// Always return success message for security to prevent email enumeration
	successResponse := func() error {
		return response.Success(c, nil, "If account exists, reset link has been sent")
	}

	var user models.User
	if err := database.DB.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return successResponse()
	}

	plainToken, tokenHash, err := generateSecureToken(32)
	if err != nil {
		log.Printf("Failed to generate reset token: %v", err)
		return successResponse()
	}

	reset := models.ResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := database.DB.Create(&reset).Error; err != nil {
		log.Printf("Failed to save reset token: %v", err)
		return successResponse()
	}

	FRONTEND_URL := os.Getenv("FRONTEND_URL")
	if FRONTEND_URL == "" {
		FRONTEND_URL = "http://localhost:3000" // Fallback default
	}

	resetURL := fmt.Sprintf("%s/auth/new-password?token=%s", FRONTEND_URL, url.QueryEscape(plainToken))
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPassword == "" {
		// Log error but don't expose to user or break security
		log.Printf("SMTP configuration missing, cannot send reset email")
		return successResponse()
	}

	if smtpFrom == "" {
		smtpFrom = smtpUser
	}

	// Send email in background to avoid blocking the response
	go func() {
		msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: Password Reset\nMIME-Version: 1.0\nContent-Type: text/plain; charset=UTF-8\n\nClick here to reset your password: %s", smtpFrom, user.Email, resetURL)

		err := smtp.SendMail(
			fmt.Sprintf("%s:%s", smtpHost, smtpPort),
			smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost),
			smtpFrom,
			[]string{user.Email},
			[]byte(msg),
		)
		if err != nil {
			log.Printf("Failed to send reset email to %s: %v", user.Email, err)
		}
	}()

	return successResponse()
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

// MeHandler returns the currently authenticated user using JWT token
func MeHandler(c *fiber.Ctx) error {
	userIDInterface := c.Locals("user_id")
	if userIDInterface == nil {
		return response.Unauthorized(c, "User not authenticated")
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		return response.InternalError(c, "Invalid user ID format")
	}

	var u models.User
	if err := database.DB.Preload("Role.Permissions").First(&u, userID).Error; err != nil {
		return response.NotFound(c, "User")
	}
	u.Password = ""
	return response.Success(c, u, "Current user retrieved")
}

// UpdateMeHandler allows authenticated users to update their own profile (name, email, password)
func UpdateMeHandler(c *fiber.Ctx) error {
	userIDInterface := c.Locals("user_id")
	if userIDInterface == nil {
		return response.Unauthorized(c, "User not authenticated")
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		return response.InternalError(c, "Invalid user ID format")
	}

	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return response.NotFound(c, "User")
	}

	if body.Email != "" && !isValidEmail(body.Email) {
		return response.ValidationError(c, map[string]string{
			"email": "invalid email format",
		})
	}

	// Update Email (ensure not taken by others)
	if body.Email != "" && body.Email != user.Email {
		var existing models.User
		if err := database.DB.Where("email = ? AND id != ?", body.Email, userID).First(&existing).Error; err == nil {
			return response.Conflict(c, "Email already taken")
		}
		user.Email = body.Email
	}

	// Update Name
	if body.Name != "" {
		user.Name = body.Name
	}

	// Update Password if provided
	if body.Password != "" {
		if err := validatePasswordStrength(body.Password); err != nil {
			return response.ValidationError(c, map[string]string{
				"password": err.Error(),
			})
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			return response.InternalError(c, "Failed to hash password")
		}
		user.Password = string(hashedPassword)
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return response.InternalError(c, "Failed to update user")
	}

	database.DB.Preload("Role.Permissions").First(&user, user.ID)
	user.Password = ""
	return response.Success(c, user, "Profile updated successfully")
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
