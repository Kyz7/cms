package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/utils"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

func oauthConfig() *oauth2.Config {
	redirect := os.Getenv("GOOGLE_REDIRECT_URL")
	if redirect == "" {
		redirect = "http://localhost:8080/auth/google/callback"
	}
	return &oauth2.Config{
		RedirectURL:  redirect,
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

var (
	stateStore = make(map[string]time.Time)
	stateMutex sync.RWMutex
)

func generateState() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based state if crypto fails (very unlikely)
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.URLEncoding.EncodeToString(b)
}

func storeState(state string) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	stateStore[state] = time.Now().Add(5 * time.Minute)

	for k, v := range stateStore {
		if time.Now().After(v) {
			delete(stateStore, k)
		}
	}
}

func validateState(state string) bool {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	expiry, exists := stateStore[state]
	if !exists || time.Now().After(expiry) {
		return false
	}
	delete(stateStore, state)
	return true
}

func GoogleLogin(c *fiber.Ctx) error {
	state := generateState()
	storeState(state)
	cfg := oauthConfig()
	url := cfg.AuthCodeURL(state)
	return c.Redirect(url)
}

func GoogleCallback(c *fiber.Ctx) error {
	state := c.Query("state")
	if !validateState(state) {
		return c.Status(400).SendString("Invalid state parameter")
	}

	code := c.Query("code")

	cfg := oauthConfig()
	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return c.Status(500).SendString("Failed to exchange token")
	}

	client := cfg.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return c.Status(500).SendString("Failed to get user info")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(500).SendString("Failed to read user info response")
	}

	var userData map[string]interface{}
	if err := json.Unmarshal(data, &userData); err != nil {
		return c.Status(500).SendString("Failed to parse user info")
	}

	// Extract email and name safely
	emailVal, ok := userData["email"]
	if !ok {
		return c.Status(400).SendString("Email not provided by Google")
	}
	email, ok := emailVal.(string)
	if !ok || email == "" {
		return c.Status(400).SendString("Invalid email from Google")
	}

	name := ""
	if nameVal, ok := userData["name"]; ok {
		if s, ok2 := nameVal.(string); ok2 {
			name = s
		}
	}

	var u models.User
	err = database.DB.Where("email = ?", email).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			viewerRoleID, err2 := utils.GetDefaultViewerRoleID()
			if err2 != nil {
				return c.Status(500).JSON(fiber.Map{"error": "default role not found"})
			}

			u = models.User{
				Name:     name,
				Email:    email,
				Provider: "google",
				Status:   "active",
				RoleID:   viewerRoleID,
			}
			if err := database.DB.Create(&u).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to create user"})
			}
		} else {
			return c.Status(500).JSON(fiber.Map{"error": "database error"})
		}
	}

	if err := database.DB.Preload("Role").First(&u, u.ID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user role"})
	}

	accessToken, err := utils.GenerateJWT(u.ID, u.Role.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate access token"})
	}
	refreshToken, err := utils.GenerateRefreshToken(u.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate refresh token"})
	}

	// Return proper AuthPayload format
	return c.JSON(fiber.Map{
		"token":        accessToken,
		"refreshToken": refreshToken,
		"user":         u,
	})
}
