package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/server"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err, "Failed to create test database")

	err = db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.ContentType{},
		&models.ContentField{},
		&models.ContentEntry{},
		&models.ContentRelation{},
		&models.ResetToken{},
		&models.RefreshToken{},
		&models.WorkflowTransition{},
		&models.WorkflowHistory{},
		&models.WorkflowComment{},
		&models.WorkflowAssignment{},
		&models.MediaFile{},
		&models.MediaFolder{},
	)
	assert.NoError(t, err, "Failed to migrate test database")

	return db
}

func SetupTestApp(t *testing.T) *fiber.App {
	db := TestDB(t)
	database.DB = db

	CreateTestRoles(t, db)

	err := utils.InitLocalStorage()
	assert.NoError(t, err, "Failed to initialize storage")
	utils.SetStorageMode(true) //

	app := server.New(db)
	return app
}

func CreateTestRoles(t *testing.T, db *gorm.DB) {
	adminRole := models.Role{
		Name:        "admin",
		Description: "Administrator with full access",
	}
	db.Create(&adminRole)

	adminPerms := []models.Permission{
		{RoleID: adminRole.ID, Module: "ContentEntry", Action: "create", FieldScope: "all"},
		{RoleID: adminRole.ID, Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{RoleID: adminRole.ID, Module: "ContentEntry", Action: "update", FieldScope: "all"},
		{RoleID: adminRole.ID, Module: "ContentEntry", Action: "delete"},
		{RoleID: adminRole.ID, Module: "ContentEntry", Action: "approve"},
		{RoleID: adminRole.ID, Module: "Media", Action: "create"},
		{RoleID: adminRole.ID, Module: "Media", Action: "read"},
		{RoleID: adminRole.ID, Module: "Media", Action: "update"},
		{RoleID: adminRole.ID, Module: "Media", Action: "delete"},
		{RoleID: adminRole.ID, Module: "SEO", Action: "create", FieldScope: "all"},
		{RoleID: adminRole.ID, Module: "SEO", Action: "read", FieldScope: "all"},
		{RoleID: adminRole.ID, Module: "SEO", Action: "update", FieldScope: "all"},
		{RoleID: adminRole.ID, Module: "SEO", Action: "delete"},
	}
	for _, perm := range adminPerms {
		db.Create(&perm)
	}

	// Editor role - IMPORTANT: Add FieldScope
	editorRole := models.Role{
		Name:        "editor",
		Description: "Editor - can create and edit content",
	}
	db.Create(&editorRole)

	editorPerms := []models.Permission{
		{RoleID: editorRole.ID, Module: "ContentEntry", Action: "create", FieldScope: "all"},
		{RoleID: editorRole.ID, Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{RoleID: editorRole.ID, Module: "ContentEntry", Action: "update", FieldScope: "all"},
		{RoleID: editorRole.ID, Module: "Media", Action: "create"},
		{RoleID: editorRole.ID, Module: "Media", Action: "read"},
		{RoleID: editorRole.ID, Module: "Media", Action: "update"},
		{RoleID: editorRole.ID, Module: "SEO", Action: "read", FieldScope: "all"},
	}
	for _, perm := range editorPerms {
		db.Create(&perm)
	}

	// Viewer role
	viewerRole := models.Role{
		Name:        "viewer",
		Description: "Viewer - read-only access",
	}
	db.Create(&viewerRole)

	viewerPerms := []models.Permission{
		{RoleID: viewerRole.ID, Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{RoleID: viewerRole.ID, Module: "Media", Action: "read"},
	}
	for _, perm := range viewerPerms {
		db.Create(&perm)
	}

	managerRole := models.Role{
		Name:        "manager",
		Description: "Manager - can approve content",
	}
	db.Create(&managerRole)

	managerPerms := []models.Permission{
		{RoleID: managerRole.ID, Module: "ContentEntry", Action: "approve"},
		{RoleID: managerRole.ID, Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{RoleID: managerRole.ID, Module: "Media", Action: "read"},
		{RoleID: managerRole.ID, Module: "SEO", Action: "read", FieldScope: "all"},
	}
	for _, perm := range managerPerms {
		db.Create(&perm)
	}
}

func CreateTestUser(t *testing.T, db *gorm.DB, email, password, roleName string) *models.User {
	hashedPassword, _ := utils.HashPassword(password)

	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		t.Fatalf("Failed to find role '%s': %v. Make sure CreateTestRoles was called.", roleName, err)
	}

	user := &models.User{
		Name:     "Test User",
		Email:    email,
		Password: hashedPassword,
		Status:   "active",
		RoleID:   role.ID,
	}

	err := db.Create(user).Error
	assert.NoError(t, err, "Failed to create test user")

	// Preload role with permissions
	db.Preload("Role.Permissions").First(user, user.ID)

	if user.Role == nil {
		t.Fatal("Role not loaded for user")
	}

	return user
}

func GetAuthToken(t *testing.T, userID uint, roleName string) string {
	token, err := utils.GenerateJWT(userID, roleName)
	assert.NoError(t, err, "Failed to generate test token")
	return token
}

func MakeRequest(app *fiber.App, method, url string, body interface{}, token string) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBody)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()

	resp, err := app.Test(req, -1)
	if err != nil {
		return rec, err
	}

	rec.Code = resp.StatusCode

	io.Copy(rec.Body, resp.Body)
	resp.Body.Close()

	return rec, nil
}

func ParseResponse(t *testing.T, resp *httptest.ResponseRecorder, v interface{}) {
	if resp.Body.Len() == 0 {
		t.Log("Warning: Response body is empty")
		return
	}

	err := json.NewDecoder(resp.Body).Decode(v)
	if err != nil && err != io.EOF {
		t.Logf("Response body: %s", resp.Body.String())
		assert.NoError(t, err, "Failed to parse response")
	}
}

type StandardResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    interface{}  `json:"data"`
	Error   *ErrorDetail `json:"error"`
	Meta    *Meta        `json:"meta"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func AssertSuccess(t *testing.T, resp *httptest.ResponseRecorder) {
	var result StandardResponse
	ParseResponse(t, resp, &result)
	assert.True(t, result.Success, "Expected success response")
	assert.Empty(t, result.Error, "Expected no error")
}

func AssertError(t *testing.T, resp *httptest.ResponseRecorder, expectedCode string) {
	var result StandardResponse
	ParseResponse(t, resp, &result)
	assert.False(t, result.Success, "Expected error response")
	assert.NotNil(t, result.Error, "Expected error object")
	assert.Equal(t, expectedCode, result.Error.Code, "Error code mismatch")
}

func MakeMultipartRequest(app *fiber.App, method, url string, fields map[string]string, token string) (*httptest.ResponseRecorder, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, val := range fields {
		writer.WriteField(key, val) // Pastikan ini ada
	}

	contentType := writer.FormDataContentType()
	writer.Close() // PENTING: Close sebelum membaca body

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", contentType) // PENTING: Set content type

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	resp, err := app.Test(req, -1)
	if err != nil {
		return rec, err
	}

	rec.Code = resp.StatusCode
	io.Copy(rec.Body, resp.Body)
	resp.Body.Close()

	return rec, nil
}

func MakeMultipartRequestWithFile(app *fiber.App, method, url string, fields map[string]string, files map[string][]byte, token string) (*httptest.ResponseRecorder, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add text fields
	for key, val := range fields {
		writer.WriteField(key, val)
	}

	// Add file fields
	for fieldName, fileContent := range files {
		part, err := writer.CreateFormFile(fieldName, fieldName+".jpg")
		if err != nil {
			return nil, err
		}
		part.Write(fileContent)
	}

	contentType := writer.FormDataContentType()
	writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", contentType)

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	resp, err := app.Test(req, -1)
	if err != nil {
		return rec, err
	}

	rec.Code = resp.StatusCode
	io.Copy(rec.Body, resp.Body)
	resp.Body.Close()

	return rec, nil
}

func MakeRedirectRequest(app *fiber.App, method, url string, token string) (*httptest.ResponseRecorder, error) {
	// Buat http.Request biasa dengan nil body untuk GET
	req := httptest.NewRequest(method, url, nil)

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()

	resp, err := app.Test(req, -1)
	if err != nil {
		return rec, err
	}

	rec.Code = resp.StatusCode

	// Copy headers - ini yang penting untuk redirect
	for k, v := range resp.Header {
		for _, val := range v {
			rec.Header().Add(k, val)
		}
	}

	io.Copy(rec.Body, resp.Body)
	resp.Body.Close()

	return rec, nil
}
