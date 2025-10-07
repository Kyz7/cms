package media_test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/testutils"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/stretchr/testify/assert"
)

// Helper function to create multipart form with file
func createMultipartForm(filename string, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(content)

	// Add additional fields
	for key, val := range fields {
		writer.WriteField(key, val)
	}

	contentType := writer.FormDataContentType()
	writer.Close()

	return body, contentType
}

func TestUploadMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	// Initialize local storage
	err := utils.InitLocalStorage()
	assert.NoError(t, err, "Failed to initialize local storage")

	utils.SetStorageMode(true)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Success - Upload image with metadata", func(t *testing.T) {
		body, contentType := createMultipartForm("test.jpg", []byte("fake image content"), map[string]string{
			"alt":     "Test Image",
			"caption": "Test Caption",
			"folder":  "test-folder",
			"tags":    `["tag1", "tag2"]`,
		})

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		// FIX: Create recorder properly and copy response body
		rec := httptest.NewRecorder()
		rec.Code = resp.StatusCode
		io.Copy(rec.Body, resp.Body)
		resp.Body.Close()

		var result testutils.StandardResponse
		testutils.ParseResponse(t, rec, &result)
		assert.True(t, result.Success)
		assert.Equal(t, "Media uploaded successfully", result.Message)

		// Verify in database
		var media models.MediaFile
		database.DB.First(&media)
		assert.Equal(t, "test.jpg", media.FileName)
		assert.Equal(t, "Test Image", media.Alt)
		assert.Equal(t, editor.ID, media.UploadedBy)
	})

	t.Run("Success - Upload video", func(t *testing.T) {
		body, contentType := createMultipartForm("video.mp4", []byte("fake video content"), map[string]string{
			"alt": "Test Video",
		})

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
	})

	t.Run("Error - No file provided", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("alt", "No file")
		contentType := writer.FormDataContentType()
		writer.Close()

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)

		// FIX THIS:
		rec := httptest.NewRecorder()
		rec.Code = resp.StatusCode
		io.Copy(rec.Body, resp.Body)
		resp.Body.Close()

		var result testutils.StandardResponse
		testutils.ParseResponse(t, rec, &result)
		assert.False(t, result.Success)
		assert.Equal(t, "BAD_REQUEST", result.Error.Code)
	})

	t.Run("Error - File too large (image)", func(t *testing.T) {
		// Create 11MB file (exceeds 10MB limit for images)
		largeContent := make([]byte, 11*1024*1024)
		body, contentType := createMultipartForm("large.jpg", largeContent, map[string]string{})

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)

		rec := httptest.NewRecorder()
		rec.Code = resp.StatusCode
		io.Copy(rec.Body, resp.Body)
		resp.Body.Close()

		var result testutils.StandardResponse
		testutils.ParseResponse(t, rec, &result)
		assert.False(t, result.Success)
		assert.Equal(t, "BAD_REQUEST", result.Error.Code)
		assert.Contains(t, result.Error.Message, "too large")
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		body, contentType := createMultipartForm("test.jpg", []byte("test"), map[string]string{})

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		// No token

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

func TestBulkUploadMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	utils.InitLocalStorage()

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Success - Upload multiple files", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add multiple files
		for i := 1; i <= 3; i++ {
			part, _ := writer.CreateFormFile("files", fmt.Sprintf("file%d.jpg", i))
			part.Write([]byte(fmt.Sprintf("content %d", i)))
		}

		writer.WriteField("folder", "bulk-folder")
		contentType := writer.FormDataContentType()
		writer.Close()

		req := httptest.NewRequest("POST", "/media/bulk-upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		rec := httptest.NewRecorder()
		rec.Code = resp.StatusCode
		io.Copy(rec.Body, resp.Body)
		resp.Body.Close()

		var result testutils.StandardResponse
		testutils.ParseResponse(t, rec, &result)

		data := result.Data.(map[string]interface{})
		assert.Equal(t, float64(3), data["uploaded"])
		assert.Equal(t, float64(0), data["failed"])
	})

	t.Run("Error - No files provided", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		contentType := writer.FormDataContentType()
		writer.Close()

		req := httptest.NewRequest("POST", "/media/bulk-upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("Partial Success - Some files fail", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add valid file
		part1, _ := writer.CreateFormFile("files", "valid.jpg")
		part1.Write([]byte("valid content"))

		// Add oversized file
		part2, _ := writer.CreateFormFile("files", "toolarge.jpg")
		part2.Write(make([]byte, 11*1024*1024)) // 11MB

		contentType := writer.FormDataContentType()
		writer.Close()

		req := httptest.NewRequest("POST", "/media/bulk-upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		rec := httptest.NewRecorder()
		rec.Code = resp.StatusCode
		io.Copy(rec.Body, resp.Body)
		resp.Body.Close()

		var result testutils.StandardResponse
		testutils.ParseResponse(t, rec, &result)

		data := result.Data.(map[string]interface{})
		assert.Greater(t, int(data["failed"].(float64)), 0)
	})
}

func TestListMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create test media files
	for i := 1; i <= 15; i++ {
		database.DB.Create(&models.MediaFile{
			FileName:   fmt.Sprintf("test%d.jpg", i),
			URL:        fmt.Sprintf("/uploads/photos/test%d.jpg", i),
			Type:       "image/jpeg",
			Size:       int64(1024 * i),
			Folder:     "test-folder",
			UploadedBy: editor.ID,
		})
	}

	t.Run("Success - List with default pagination", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Meta)
		assert.Equal(t, int64(15), result.Meta.Total)
	})

	t.Run("Success - List with custom pagination", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media?page=2&limit=5", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.Equal(t, 2, result.Meta.Page)
		assert.Equal(t, 5, result.Meta.Limit)
	})

	t.Run("Success - Filter by type", func(t *testing.T) {
		// Add video file
		database.DB.Create(&models.MediaFile{
			FileName:   "video.mp4",
			URL:        "/uploads/videos/video.mp4",
			Type:       "video/mp4",
			Size:       10240,
			UploadedBy: editor.ID,
		})

		resp, err := testutils.MakeRequest(app, "GET", "/media?type=video", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)
	})

	t.Run("Success - Filter by folder", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media?folder=test-folder", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Search by filename", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media?search=test5", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})
}

func TestGetMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	media := &models.MediaFile{
		FileName:   "test.jpg",
		URL:        "/uploads/photos/test.jpg",
		Type:       "image/jpeg",
		Size:       1024,
		Alt:        "Test Alt",
		Caption:    "Test Caption",
		UploadedBy: editor.ID,
	}
	database.DB.Create(media)

	t.Run("Success - Get media by ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/"+fmt.Sprint(media.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.(map[string]interface{})
		assert.Equal(t, "test.jpg", data["file_name"])
		assert.Equal(t, "Test Alt", data["alt"])
	})

	t.Run("Error - Media not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/9999", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - Invalid ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/invalid", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestUpdateMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	media := &models.MediaFile{
		FileName:   "test.jpg",
		URL:        "/uploads/photos/test.jpg",
		Type:       "image/jpeg",
		Size:       1024,
		Alt:        "Original Alt",
		Caption:    "Original Caption",
		Folder:     "original-folder",
		UploadedBy: editor.ID,
	}
	database.DB.Create(media)

	t.Run("Success - Update all metadata", func(t *testing.T) {
		body := map[string]interface{}{
			"alt":     "Updated Alt",
			"caption": "Updated Caption",
			"folder":  "updated-folder",
			"tags":    []string{"tag1", "tag2", "tag3"},
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/media/"+fmt.Sprint(media.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)

		// Verify updates
		var updated models.MediaFile
		database.DB.First(&updated, media.ID)
		assert.Equal(t, "Updated Alt", updated.Alt)
		assert.Equal(t, "Updated Caption", updated.Caption)
		assert.Equal(t, "updated-folder", updated.Folder)
		assert.NotNil(t, updated.Tags)
	})

	t.Run("Success - Partial update", func(t *testing.T) {
		body := map[string]interface{}{
			"alt": "Only Alt Updated",
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/media/"+fmt.Sprint(media.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Media not found", func(t *testing.T) {
		body := map[string]interface{}{
			"alt": "Test",
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/media/9999", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - Invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/media/"+fmt.Sprint(media.ID), bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestDeleteMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)
	editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Success - Delete media", func(t *testing.T) {
		media := &models.MediaFile{
			FileName:   "todelete.jpg",
			URL:        "/uploads/photos/todelete.jpg",
			Type:       "image/jpeg",
			Size:       1024,
			UploadedBy: editor.ID,
		}
		database.DB.Create(media)

		resp, err := testutils.MakeRequest(app, "DELETE", "/media/"+fmt.Sprint(media.ID), nil, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)

		// Verify deletion
		var deleted models.MediaFile
		result := database.DB.First(&deleted, media.ID)
		assert.Error(t, result.Error)
	})

	t.Run("Error - Media not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE", "/media/9999", nil, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - Forbidden (editor trying to delete)", func(t *testing.T) {
		media := &models.MediaFile{
			FileName:   "protected.jpg",
			URL:        "/uploads/photos/protected.jpg",
			Type:       "image/jpeg",
			Size:       1024,
			UploadedBy: admin.ID,
		}
		database.DB.Create(media)

		resp, err := testutils.MakeRequest(app, "DELETE", "/media/"+fmt.Sprint(media.ID), nil, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.Code)

		testutils.AssertError(t, resp, "FORBIDDEN")
	})
}

func TestSearchMediaHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create searchable media
	database.DB.Create(&models.MediaFile{
		FileName:   "sunset.jpg",
		URL:        "/uploads/photos/sunset.jpg",
		Type:       "image/jpeg",
		Size:       2048,
		Alt:        "Beautiful Sunset",
		Caption:    "Sunset at the beach",
		UploadedBy: editor.ID,
	})

	database.DB.Create(&models.MediaFile{
		FileName:   "sunrise.jpg",
		URL:        "/uploads/photos/sunrise.jpg",
		Type:       "image/jpeg",
		Size:       2048,
		Alt:        "Morning Sunrise",
		Caption:    "Sunrise in the mountains",
		UploadedBy: editor.ID,
	})

	t.Run("Success - Search by filename", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/search?q=sunset", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)
	})

	t.Run("Success - Search by alt text", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/search?q=Beautiful", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Search by caption", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/search?q=beach", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Search with pagination", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/search?q=sun&page=1&limit=5", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.NotNil(t, result.Meta)
	})

	t.Run("Error - Missing search query", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/search", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})

	t.Run("Success - No results", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/search?q=nonexistent", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.Equal(t, int64(0), result.Meta.Total)
	})
}

func TestGetMediaStatsHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create media files with different types
	database.DB.Create(&models.MediaFile{FileName: "img1.jpg", URL: "/uploads/photos/img1.jpg", Type: "image/jpeg", Size: 2048, UploadedBy: editor.ID})
	database.DB.Create(&models.MediaFile{FileName: "img2.png", URL: "/uploads/photos/img2.png", Type: "image/png", Size: 3072, UploadedBy: editor.ID})
	database.DB.Create(&models.MediaFile{FileName: "vid1.mp4", URL: "/uploads/videos/vid1.mp4", Type: "video/mp4", Size: 10240, UploadedBy: editor.ID})
	database.DB.Create(&models.MediaFile{FileName: "doc1.pdf", URL: "/uploads/documents/doc1.pdf", Type: "application/pdf", Size: 5120, UploadedBy: editor.ID})

	// Create recent upload (within 24h)
	recentTime := time.Now().Add(-12 * time.Hour)
	database.DB.Create(&models.MediaFile{FileName: "recent.jpg", URL: "/uploads/photos/recent.jpg", Type: "image/jpeg", Size: 1024, UploadedBy: editor.ID, CreatedAt: recentTime})

	t.Run("Success - Get comprehensive statistics", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/stats", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		stats := result.Data.(map[string]interface{})

		// Check total files
		assert.NotNil(t, stats["total_files"])
		assert.GreaterOrEqual(t, int(stats["total_files"].(float64)), 5)

		// Check total size
		assert.NotNil(t, stats["total_size_bytes"])
		assert.Greater(t, int(stats["total_size_bytes"].(float64)), 0)

		// Check by_type breakdown
		assert.NotNil(t, stats["by_type"])
		byType := stats["by_type"].(map[string]interface{})
		assert.Contains(t, byType, "image")
		assert.Contains(t, byType, "video")
		assert.Contains(t, byType, "application")

		// Check recent uploads
		assert.NotNil(t, stats["recent_uploads_24h"])
		assert.GreaterOrEqual(t, int(stats["recent_uploads_24h"].(float64)), 1)

		// Check storage mode
		assert.NotNil(t, stats["storage_mode"])
		assert.Contains(t, []string{"local", "s3"}, stats["storage_mode"].(string))
	})
}

func TestCreateFolderHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Success - Create root folder", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "Images",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/media/folders", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.(map[string]interface{})
		assert.Equal(t, "Images", data["name"])
		assert.Equal(t, "/Images", data["path"])
	})

	t.Run("Success - Create nested folder", func(t *testing.T) {
		// Create parent folder
		parent := &models.MediaFolder{
			Name:      "Photos",
			Path:      "/Photos",
			CreatedBy: editor.ID,
		}
		database.DB.Create(parent)

		body := map[string]interface{}{
			"name":      "Vacation",
			"parent_id": parent.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/media/folders", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)

		data := result.Data.(map[string]interface{})
		assert.Equal(t, "/Photos/Vacation", data["path"])
	})

	t.Run("Error - Missing folder name", func(t *testing.T) {
		body := map[string]interface{}{}

		resp, err := testutils.MakeRequest(app, "POST", "/media/folders", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Parent folder not found", func(t *testing.T) {
		body := map[string]interface{}{
			"name":      "Subfolder",
			"parent_id": 9999,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/media/folders", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

func TestListFoldersHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create folder hierarchy
	root := &models.MediaFolder{Name: "Root", Path: "/Root", CreatedBy: editor.ID}
	database.DB.Create(root)

	child := &models.MediaFolder{Name: "Child", Path: "/Root/Child", ParentID: &root.ID, CreatedBy: editor.ID}
	database.DB.Create(child)

	t.Run("Success - List all folders", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/folders", nil, token)
		assert.NoError(t, err)

		// Debug: print response body
		t.Logf("Response status: %d", resp.Code)
		t.Logf("Response body: %s", resp.Body.String())

		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		// Check if Data is nil before type assertion
		if result.Data != nil {
			folders := result.Data.([]interface{})
			assert.GreaterOrEqual(t, len(folders), 2)
		}
	})

	t.Run("Verify folder hierarchy", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/media/folders", nil, token)
		assert.NoError(t, err)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)

		folders := result.Data.([]interface{})

		// Check if parent-child relationship exists
		hasRoot := false
		hasChild := false

		for _, f := range folders {
			folder := f.(map[string]interface{})
			if folder["name"] == "Root" {
				hasRoot = true
			}
			if folder["name"] == "Child" && folder["path"] == "/Root/Child" {
				hasChild = true
			}
		}

		assert.True(t, hasRoot, "Root folder should exist")
		assert.True(t, hasChild, "Child folder should exist with correct path")
	})
}

// Edge Cases & Integration Tests

func TestMediaUploadIntegration(t *testing.T) {
	app := testutils.SetupTestApp(t)

	utils.InitLocalStorage()

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Complete workflow - Upload, Update, Delete", func(t *testing.T) {
		// Upload dengan editor
		body, contentType := createMultipartForm("workflow.jpg", []byte("test content"), map[string]string{
			"alt": "Initial Alt",
		})

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, _ := app.Test(req, -1)
		assert.Equal(t, 201, resp.StatusCode)

		rec := httptest.NewRecorder()
		rec.Code = resp.StatusCode
		io.Copy(rec.Body, resp.Body)
		resp.Body.Close()

		var uploadResult testutils.StandardResponse
		testutils.ParseResponse(t, rec, &uploadResult)

		mediaData := uploadResult.Data.(map[string]interface{})
		mediaID := int(mediaData["id"].(float64))

		// Update dengan editor
		updateBody := map[string]interface{}{
			"alt":     "Updated Alt",
			"caption": "New Caption",
		}

		resp2, _ := testutils.MakeRequest(app, "PUT", "/media/"+fmt.Sprint(mediaID), updateBody, token)
		assert.Equal(t, 200, resp2.Code)

		// Verify update
		resp3, _ := testutils.MakeRequest(app, "GET", "/media/"+fmt.Sprint(mediaID), nil, token)
		assert.Equal(t, 200, resp3.Code)

		var getResult testutils.StandardResponse
		testutils.ParseResponse(t, resp3, &getResult)
		getData := getResult.Data.(map[string]interface{})
		assert.Equal(t, "Updated Alt", getData["alt"])

		// Delete dengan ADMIN (bukan editor)
		admin := testutils.CreateTestUser(t, database.DB, "admin-workflow@test.com", "password", "admin")
		adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

		resp4, _ := testutils.MakeRequest(app, "DELETE", "/media/"+fmt.Sprint(mediaID), nil, adminToken)
		assert.Equal(t, 204, resp4.Code)

		// Verify deletion
		resp5, _ := testutils.MakeRequest(app, "GET", "/media/"+fmt.Sprint(mediaID), nil, token)
		assert.Equal(t, 404, resp5.Code)
	})
}

func TestMediaFolderHierarchy(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Create deep folder hierarchy", func(t *testing.T) {
		// Level 1
		body1 := map[string]interface{}{"name": "Level1"}
		resp1, _ := testutils.MakeRequest(app, "POST", "/media/folders", body1, token)
		assert.Equal(t, 201, resp1.Code)

		var result1 testutils.StandardResponse
		testutils.ParseResponse(t, resp1, &result1)
		level1ID := int(result1.Data.(map[string]interface{})["id"].(float64))

		// Level 2
		body2 := map[string]interface{}{
			"name":      "Level2",
			"parent_id": level1ID,
		}
		resp2, _ := testutils.MakeRequest(app, "POST", "/media/folders", body2, token)
		assert.Equal(t, 201, resp2.Code)

		var result2 testutils.StandardResponse
		testutils.ParseResponse(t, resp2, &result2)
		assert.Equal(t, "/Level1/Level2", result2.Data.(map[string]interface{})["path"])

		// Level 3
		level2ID := int(result2.Data.(map[string]interface{})["id"].(float64))
		body3 := map[string]interface{}{
			"name":      "Level3",
			"parent_id": level2ID,
		}
		resp3, _ := testutils.MakeRequest(app, "POST", "/media/folders", body3, token)
		assert.Equal(t, 201, resp3.Code)

		var result3 testutils.StandardResponse
		testutils.ParseResponse(t, resp3, &result3)
		assert.Equal(t, "/Level1/Level2/Level3", result3.Data.(map[string]interface{})["path"])
	})
}

func TestMediaPermissions(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	editor := testutils.CreateTestUser(t, db, "editor@test.com", "password", "editor")
	viewer := testutils.CreateTestUser(t, db, "viewer@test.com", "password", "viewer")

	adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)
	// editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name) // HAPUS INI
	viewerToken := testutils.GetAuthToken(t, viewer.ID, viewer.Role.Name)

	media := &models.MediaFile{
		FileName:   "test.jpg",
		URL:        "/uploads/photos/test.jpg",
		Type:       "image/jpeg",
		Size:       1024,
		UploadedBy: editor.ID,
	}
	db.Create(media)

	t.Run("Admin can delete any media", func(t *testing.T) {
		resp, _ := testutils.MakeRequest(app, "DELETE", "/media/"+fmt.Sprint(media.ID), nil, adminToken)
		assert.Equal(t, 204, resp.Code)
	})

	// TAMBAHKAN media2 DI SINI agar bisa dipakai di test selanjutnya
	media2 := &models.MediaFile{
		FileName:   "test2.jpg",
		URL:        "/uploads/photos/test2.jpg",
		Type:       "image/jpeg",
		Size:       1024,
		UploadedBy: editor.ID,
	}
	db.Create(media2)

	t.Run("Viewer cannot delete media", func(t *testing.T) {
		resp, _ := testutils.MakeRequest(app, "DELETE", "/media/"+fmt.Sprint(media2.ID), nil, viewerToken)
		assert.Equal(t, 403, resp.Code)
		testutils.AssertError(t, resp, "FORBIDDEN")
	})

	t.Run("Viewer can read media", func(t *testing.T) {
		resp, _ := testutils.MakeRequest(app, "GET", "/media/"+fmt.Sprint(media2.ID), nil, viewerToken)
		assert.Equal(t, 200, resp.Code)
	})
}

func TestMediaWithTags(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	t.Run("Upload with tags", func(t *testing.T) {
		body, contentType := createMultipartForm("tagged.jpg", []byte("test"), map[string]string{
			"tags": `["nature", "landscape", "sunset"]`,
		})

		req := httptest.NewRequest("POST", "/media/upload", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, _ := app.Test(req, -1)
		assert.Equal(t, 201, resp.StatusCode)

		// Verify tags saved
		var media models.MediaFile
		database.DB.Where("file_name = ?", "tagged.jpg").First(&media)
		assert.NotNil(t, media.Tags)
	})

	t.Run("Update tags", func(t *testing.T) {
		media := &models.MediaFile{
			FileName:   "updatetags.jpg",
			URL:        "/uploads/photos/updatetags.jpg",
			Type:       "image/jpeg",
			Size:       1024,
			UploadedBy: editor.ID,
		}
		database.DB.Create(media)

		body := map[string]interface{}{
			"tags": []string{"new", "updated", "tags"},
		}

		resp, _ := testutils.MakeRequest(app, "PUT", "/media/"+fmt.Sprint(media.ID), body, token)
		assert.Equal(t, 200, resp.Code)
	})
}

func TestMediaStatsByType(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create diverse media types
	mediaTypes := []struct {
		filename string
		mimeType string
		size     int64
	}{
		{"img1.jpg", "image/jpeg", 2048},
		{"img2.png", "image/png", 3072},
		{"img3.gif", "image/gif", 1024},
		{"video1.mp4", "video/mp4", 10240},
		{"video2.avi", "video/avi", 15360},
		{"doc1.pdf", "application/pdf", 5120},
		{"doc2.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", 4096},
	}

	for _, mt := range mediaTypes {
		database.DB.Create(&models.MediaFile{
			FileName:   mt.filename,
			URL:        "/uploads/" + mt.filename,
			Type:       mt.mimeType,
			Size:       mt.size,
			UploadedBy: editor.ID,
		})
	}

	t.Run("Stats show correct type distribution", func(t *testing.T) {
		resp, _ := testutils.MakeRequest(app, "GET", "/media/stats", nil, token)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)

		stats := result.Data.(map[string]interface{})
		byType := stats["by_type"].(map[string]interface{})

		// Should have 3 images, 2 videos, 2 documents
		assert.Equal(t, float64(3), byType["image"])
		assert.Equal(t, float64(2), byType["video"])
		assert.Equal(t, float64(2), byType["application"])

		// Total size should be sum of all
		expectedSize := int64(2048 + 3072 + 1024 + 10240 + 15360 + 5120 + 4096)
		assert.Equal(t, float64(expectedSize), stats["total_size_bytes"])
	})
}
