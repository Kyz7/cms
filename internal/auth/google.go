package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/utils"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost:8080/auth/google/callback",
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	Endpoint:     google.Endpoint,
}

var (
	stateStore = make(map[string]time.Time)
	stateMutex sync.RWMutex
)

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
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
	url := googleOauthConfig.AuthCodeURL(state)
	return c.Redirect(url)
}

func GoogleCallback(c *fiber.Ctx) error {
	state := c.Query("state")
	if !validateState(state) {
		return c.Status(400).SendString("Invalid state parameter")
	}

	code := c.Query("code")

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return c.Status(500).SendString("Failed to exchange token")
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return c.Status(500).SendString("Failed to get user info")
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	var userData map[string]interface{}
	json.Unmarshal(data, &userData)

	email := userData["email"].(string)
	name := userData["name"].(string)

	var u models.User
	err = database.DB.Where("email = ?", email).First(&u).Error
	if err != nil {
		viewerRoleID, err := utils.GetDefaultViewerRoleID()
		if err != nil {
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
	}

	database.DB.Preload("Role").First(&u, u.ID)

	accessToken, _ := utils.GenerateJWT(u.ID, u.Role.Name)
	refreshToken, _ := utils.GenerateRefreshToken(u.ID)

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          u,
	})
}
