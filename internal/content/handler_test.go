package content_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/testutils"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// ============================================
// CONTENT TYPE MANAGEMENT TESTS
// ============================================

func TestCreateContentTypeHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Create content type", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "Blog",
			"slug": "blog",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/content/types", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Missing required fields", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "Product",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/content/types", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Unauthorized", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "Page",
			"slug": "page",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/content/types", body, "")
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.Code)

		testutils.AssertError(t, resp, "UNAUTHORIZED")
	})
}

func TestListContentTypesHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	database.DB.Create(&models.ContentType{Name: "Blog", Slug: "blog"})
	database.DB.Create(&models.ContentType{Name: "Product", Slug: "product"})

	t.Run("Success - List all content types", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		data := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(data), 2)
	})
}

func TestGetContentTypeHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	t.Run("Success - Get content type", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/"+fmt.Sprint(ct.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/9999", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - Invalid ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/invalid", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestUpdateContentTypeHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	t.Run("Success - Update content type", func(t *testing.T) {
		body := map[string]interface{}{
			"name":       "Updated Blog",
			"slug":       "updated-blog",
			"enable_seo": true,
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/content/types/"+fmt.Sprint(ct.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)

		var updated models.ContentType
		database.DB.First(&updated, ct.ID)
		assert.Equal(t, "Updated Blog", updated.Name)
		assert.True(t, updated.EnableSEO)
	})

	t.Run("Error - Not found", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "Updated",
			"slug": "updated",
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/content/types/9999", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

func TestDeleteContentTypeHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Delete content type", func(t *testing.T) {
		ct := &models.ContentType{Name: "ToDelete", Slug: "to-delete"}
		database.DB.Create(ct)

		resp, err := testutils.MakeRequest(app, "DELETE", "/content/types/"+fmt.Sprint(ct.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)

		var deleted models.ContentType
		result := database.DB.First(&deleted, ct.ID)
		assert.Error(t, result.Error)
	})

	t.Run("Error - Cannot delete with entries", func(t *testing.T) {
		ct := &models.ContentType{Name: "HasEntries", Slug: "has-entries"}
		database.DB.Create(ct)

		entry := &models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     admin.ID,
			Status:        models.StatusDraft,
		}
		database.DB.Create(entry)

		resp, err := testutils.MakeRequest(app, "DELETE", "/content/types/"+fmt.Sprint(ct.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)

		testutils.AssertError(t, resp, "CONFLICT")
	})
}

// ============================================
// FIELD MANAGEMENT TESTS
// ============================================

func TestAddFieldHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	t.Run("Success - Add field", func(t *testing.T) {
		body := map[string]interface{}{
			"name":     "title",
			"type":     "string",
			"required": true,
			"is_seo":   false,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/content/types/"+fmt.Sprint(ct.ID)+"/fields", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Add field with validation rules", func(t *testing.T) {
		maxLen := 100
		minLen := 5
		body := map[string]interface{}{
			"name":       "slug",
			"type":       "string",
			"required":   true,
			"unique":     true,
			"max_length": maxLen,
			"min_length": minLen,
			"pattern":    "^[a-z0-9-]+$",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/content/types/"+fmt.Sprint(ct.ID)+"/fields", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Missing required fields", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "description",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/content/types/"+fmt.Sprint(ct.ID)+"/fields", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})
}

func TestUpdateFieldHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	field := &models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "title",
		Type:          "string",
		Required:      true,
	}
	database.DB.Create(field)

	t.Run("Success - Update field", func(t *testing.T) {
		maxLen := 200
		body := map[string]interface{}{
			"name":       "title",
			"type":       "string",
			"required":   true,
			"max_length": maxLen,
			"help_text":  "Enter the blog title",
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/content/fields/"+fmt.Sprint(field.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})
}

func TestDeleteFieldHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	field := &models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "temporary",
		Type:          "string",
	}
	database.DB.Create(field)

	t.Run("Success - Delete field", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE", "/content/fields/"+fmt.Sprint(field.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)
	})
}

// ============================================
// CONTENT ENTRY MANAGEMENT TESTS
// ============================================

func TestCreateEntryWithMedia(t *testing.T) {
	app := testutils.SetupTestApp(t)

	utils.InitLocalStorage()

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Article", Slug: "article"}
	database.DB.Create(ct)

	titleField := &models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "title",
		Type:          "string",
		Required:      true,
	}
	database.DB.Create(titleField)

	mediaField := &models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "featured_image",
		Type:          "media",
		Required:      false,
	}
	database.DB.Create(mediaField)

	t.Run("Success - Upload new media", func(t *testing.T) {
		fields := map[string]string{
			"title": "Article with Image",
		}
		files := map[string][]byte{
			"featured_image": []byte("fake image content"),
		}

		resp, err := testutils.MakeMultipartRequestWithFile(app, "POST",
			"/content/"+fmt.Sprint(ct.ID)+"/entries", fields, files, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Reuse existing media", func(t *testing.T) {
		existingMedia := &models.MediaFile{
			FileName:   "existing.jpg",
			URL:        "/uploads/photos/existing.jpg",
			Type:       "image/jpeg",
			Size:       1024,
			UploadedBy: editor.ID,
		}
		database.DB.Create(existingMedia)

		fields := map[string]string{
			"title":                   "Article with Existing Image",
			"featured_image_media_id": fmt.Sprint(existingMedia.ID),
		}

		resp, err := testutils.MakeMultipartRequest(app, "POST",
			"/content/"+fmt.Sprint(ct.ID)+"/entries", fields, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		entryData := result.Data.(map[string]interface{})
		data := entryData["data"].(map[string]interface{})

		assert.Equal(t, existingMedia.URL, data["featured_image"])
	})
}

func TestCreateEntryHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	field := &models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "title",
		Type:          "string",
		Required:      true,
	}
	database.DB.Create(field)

	t.Run("Success - Create entry", func(t *testing.T) {
		fields := map[string]string{
			"title": "My First Blog Post",
		}

		resp, err := testutils.MakeMultipartRequest(app, "POST", "/content/"+fmt.Sprint(ct.ID)+"/entries", fields, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Missing required field", func(t *testing.T) {
		fields := map[string]string{}

		resp, err := testutils.MakeMultipartRequest(app, "POST", "/content/"+fmt.Sprint(ct.ID)+"/entries", fields, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestListEntriesHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	for i := 0; i < 5; i++ {
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     editor.ID,
			Status:        models.StatusDraft,
			Data:          datatypes.JSON([]byte(`{}`)),
		})
	}

	t.Run("Success - List entries with pagination", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/"+fmt.Sprint(ct.ID)+"/entries?page=1&limit=10", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Meta)
		assert.Equal(t, int64(5), result.Meta.Total)
	})

	t.Run("Success - Filter by status", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/"+fmt.Sprint(ct.ID)+"/entries?status=draft", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})
}

func TestUpdateEntryHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	utils.InitLocalStorage()

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	titleField := &models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "title",
		Type:          "string",
		Required:      false,
	}
	database.DB.Create(titleField)

	initialData := map[string]interface{}{
		"title": "Original Title",
	}
	jsonData, _ := json.Marshal(initialData)
	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON(jsonData),
	}
	database.DB.Create(entry)

	t.Run("Success - Update text only (JSON)", func(t *testing.T) {
		body := map[string]interface{}{
			"title": "Updated Title",
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/content/entries/"+fmt.Sprint(entry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Cannot update published entry", func(t *testing.T) {
		publishedEntry := &models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     editor.ID,
			Status:        models.StatusPublished,
			Data:          datatypes.JSON([]byte(`{}`)),
		}
		database.DB.Create(publishedEntry)

		body := map[string]interface{}{
			"title": "Updated",
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/content/entries/"+fmt.Sprint(publishedEntry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)

		testutils.AssertError(t, resp, "CONFLICT")
	})
}

func TestDeleteEntryHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	t.Run("Success - Delete draft entry", func(t *testing.T) {
		entry := &models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     admin.ID,
			Status:        models.StatusDraft,
		}
		database.DB.Create(entry)

		resp, err := testutils.MakeRequest(app, "DELETE", "/content/entries/"+fmt.Sprint(entry.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)
	})

	t.Run("Error - Cannot delete published entry", func(t *testing.T) {
		publishedEntry := &models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     admin.ID,
			Status:        models.StatusPublished,
		}
		database.DB.Create(publishedEntry)

		resp, err := testutils.MakeRequest(app, "DELETE", "/content/entries/"+fmt.Sprint(publishedEntry.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)

		testutils.AssertError(t, resp, "CONFLICT")
	})
}

// ============================================
// WORKFLOW TESTS
// ============================================

func TestWorkflowChangeStatus(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
	admin := testutils.CreateTestUser(t, database.DB, "admin_wf@test.com", "password", "admin")
	editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)
	adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Test"}`)),
	}
	database.DB.Create(entry)

	t.Run("Success - Editor requests review", func(t *testing.T) {
		body := map[string]interface{}{
			"status":  "in_review",
			"comment": "Ready for review",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/status", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)
	})

	t.Run("Success - Editor marks ready for approval", func(t *testing.T) {
		body := map[string]interface{}{
			"status":  "ready_for_approval",
			"comment": "Ready for approval",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/status", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("Success - Admin approves", func(t *testing.T) {
		body := map[string]interface{}{
			"status":  "approved",
			"comment": "Approved",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/status", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("Success - Admin publishes", func(t *testing.T) {
		body := map[string]interface{}{
			"status":  "published",
			"comment": "Publishing",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/status", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var updated models.ContentEntry
		database.DB.First(&updated, entry.ID)
		assert.Equal(t, models.StatusPublished, updated.Status)
		assert.NotNil(t, updated.PublishedAt)
	})
}

func TestWorkflowRejectFlow(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor2@test.com", "password", "editor")
	admin := testutils.CreateTestUser(t, database.DB, "admin2@test.com", "password", "admin")
	editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)
	adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Article", Slug: "article"}
	database.DB.Create(ct)

	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusReadyForApproval,
		Data:          datatypes.JSON([]byte(`{"title":"Test Article"}`)),
	}
	database.DB.Create(entry)

	t.Run("Success - Admin rejects entry", func(t *testing.T) {
		body := map[string]interface{}{
			"comment": "Needs more work",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/reject", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var updated models.ContentEntry
		database.DB.First(&updated, entry.ID)
		assert.Equal(t, models.StatusRejected, updated.Status)
	})

	t.Run("Success - Editor returns to draft", func(t *testing.T) {
		body := map[string]interface{}{
			"status":  "draft",
			"comment": "Making revisions",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/status", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})
}

func TestWorkflowHistory(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor3@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "News", Slug: "news"}
	database.DB.Create(ct)

	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"News"}`)),
	}
	database.DB.Create(entry)

	database.DB.Create(&models.WorkflowHistory{
		EntryID:    entry.ID,
		FromStatus: models.StatusDraft,
		ToStatus:   models.StatusInReview,
		ChangedBy:  editor.ID,
		Comment:    "Requesting review",
	})

	t.Run("Success - Get workflow history", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/history", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		history := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(history), 1)
	})
}

func TestWorkflowComments(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor4@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Post", Slug: "post"}
	database.DB.Create(ct)

	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusInReview,
		Data:          datatypes.JSON([]byte(`{"title":"Post"}`)),
	}
	database.DB.Create(entry)

	t.Run("Success - Add public comment", func(t *testing.T) {
		body := map[string]interface{}{
			"comment":    "This looks good",
			"is_private": false,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/comments", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Add private comment", func(t *testing.T) {
		body := map[string]interface{}{
			"comment":    "Internal note",
			"is_private": true,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/comments", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
	})

	t.Run("Success - Get all comments", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/comments?include_private=true", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		comments := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(comments), 2)
	})

	t.Run("Error - Missing comment text", func(t *testing.T) {
		body := map[string]interface{}{
			"is_private": false,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/comments", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)
	})
}

func TestWorkflowAssignments(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin_assign@test.com", "password", "admin")
	reviewer := testutils.CreateTestUser(t, database.DB, "reviewer@test.com", "password", "editor")
	adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)
	reviewerToken := testutils.GetAuthToken(t, reviewer.ID, reviewer.Role.Name)

	ct := &models.ContentType{Name: "Article", Slug: "article"}
	database.DB.Create(ct)

	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     admin.ID,
		Status:        models.StatusInReview,
		Data:          datatypes.JSON([]byte(`{"title":"Article"}`)),
	}
	database.DB.Create(entry)

	t.Run("Success - Assign entry", func(t *testing.T) {
		body := map[string]interface{}{
			"assigned_to": reviewer.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/assign", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Get my assignments", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/workflow/assignments", nil, reviewerToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		if result.Data != nil {
			assignments := result.Data.([]interface{})
			assert.GreaterOrEqual(t, len(assignments), 1)
		}
	})

	t.Run("Error - Missing assigned_to", func(t *testing.T) {
		body := map[string]interface{}{}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/assign", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)
	})
}

func TestWorkflowRequestReview(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor6@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(ct)

	entry := &models.ContentEntry{
		ContentTypeID: ct.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Blog Post"}`)),
	}
	database.DB.Create(entry)

	t.Run("Success - Request review", func(t *testing.T) {
		body := map[string]interface{}{
			"comment": "Please review this",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entry.ID)+"/request-review", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var updated models.ContentEntry
		database.DB.First(&updated, entry.ID)
		assert.Equal(t, models.StatusInReview, updated.Status)
	})
}

func TestWorkflowStatistics(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin_stats@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "Product", Slug: "product"}
	database.DB.Create(ct)

	statuses := []models.WorkflowStatus{
		models.StatusDraft,
		models.StatusInReview,
		models.StatusApproved,
		models.StatusPublished,
	}

	for _, status := range statuses {
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     admin.ID,
			Status:        status,
			Data:          datatypes.JSON([]byte(`{"name":"Product"}`)),
		})
	}

	t.Run("Success - Get workflow statistics", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/workflow/content-types/"+fmt.Sprint(ct.ID)+"/stats", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		if result.Data != nil {
			stats := result.Data.(map[string]interface{})
			assert.NotNil(t, stats["total"])
			assert.NotNil(t, stats["draft"])
			assert.NotNil(t, stats["published"])
		}
	})
}

func TestWorkflowGetEntriesByStatus(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor7@test.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	ct := &models.ContentType{Name: "News", Slug: "news"}
	database.DB.Create(ct)

	for i := 0; i < 3; i++ {
		database.DB.Create(&models.ContentEntry{
			ContentTypeID: ct.ID,
			CreatedBy:     editor.ID,
			Status:        models.StatusInReview,
			Data:          datatypes.JSON([]byte(`{"title":"News"}`)),
		})
	}

	t.Run("Success - Get entries by status", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/workflow/content-types/"+fmt.Sprint(ct.ID)+"/entries?status=in_review", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		if result.Data != nil {
			entries := result.Data.([]interface{})
			assert.GreaterOrEqual(t, len(entries), 3)
		}
	})
}

// ============================================
// API REFERENCE TESTS
// ============================================

func TestGenerateAPIReference(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin3@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{
		Name:      "Product",
		Slug:      "product",
		EnableSEO: true,
	}
	database.DB.Create(ct)

	fields := []models.ContentField{
		{ContentTypeID: ct.ID, Name: "name", Type: "string", Required: true},
		{ContentTypeID: ct.ID, Name: "price", Type: "number", Required: true},
		{ContentTypeID: ct.ID, Name: "description", Type: "text", Required: false},
		{ContentTypeID: ct.ID, Name: "meta_title", Type: "string", Required: false, IsSEO: true},
	}

	for _, field := range fields {
		database.DB.Create(&field)
	}

	t.Run("Success - Generate API reference", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/"+fmt.Sprint(ct.ID)+"/api-reference", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &result)

		assert.Equal(t, "Product", result["content_type"])
		assert.Equal(t, "product", result["slug"])
		assert.True(t, result["seo_enabled"].(bool))
		assert.NotNil(t, result["endpoints"])
		assert.NotNil(t, result["fields"])

		endpoints := result["endpoints"].([]interface{})
		assert.GreaterOrEqual(t, len(endpoints), 5)

		fields := result["fields"].([]interface{})
		assert.Equal(t, 4, len(fields))
	})

	t.Run("Error - Content type not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/9999/api-reference", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
	})
}

func TestGenerateOpenAPISpec(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin4@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{
		Name:      "Article",
		Slug:      "article",
		EnableSEO: false,
	}
	database.DB.Create(ct)

	maxLen := 200
	minLen := 10
	fields := []models.ContentField{
		{
			ContentTypeID: ct.ID,
			Name:          "title",
			Type:          "string",
			Required:      true,
			MaxLength:     &maxLen,
			MinLength:     &minLen,
		},
		{
			ContentTypeID: ct.ID,
			Name:          "slug",
			Type:          "string",
			Required:      true,
			Unique:        true,
			Pattern:       "^[a-z0-9-]+$",
		},
	}

	for _, field := range fields {
		database.DB.Create(&field)
	}

	t.Run("Success - Generate OpenAPI spec", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/"+fmt.Sprint(ct.ID)+"/openapi", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var spec map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &spec)

		assert.Equal(t, "3.0.0", spec["openapi"])

		info := spec["info"].(map[string]interface{})
		assert.Equal(t, "Article API", info["title"])

		assert.NotNil(t, spec["paths"])
		assert.NotNil(t, spec["components"])

		components := spec["components"].(map[string]interface{})
		schemas := components["schemas"].(map[string]interface{})
		assert.NotNil(t, schemas["ArticleRequest"])
		assert.NotNil(t, schemas["ArticleResponse"])
	})
}

func TestGenerateMarkdownDocs(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin5@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{
		Name:      "BlogPost",
		Slug:      "blog-post",
		EnableSEO: true,
	}
	database.DB.Create(ct)

	maxLen := 150
	fields := []models.ContentField{
		{
			ContentTypeID: ct.ID,
			Name:          "title",
			Type:          "string",
			Required:      true,
			MaxLength:     &maxLen,
		},
		{
			ContentTypeID: ct.ID,
			Name:          "published_at",
			Type:          "date",
			Required:      false,
		},
	}

	for _, field := range fields {
		database.DB.Create(&field)
	}

	t.Run("Success - Generate Markdown documentation", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/"+fmt.Sprint(ct.ID)+"/docs/markdown", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		markdown := resp.Body.String()
		assert.Contains(t, markdown, "# BlogPost API Documentation")
		assert.Contains(t, markdown, "## Fields")
		assert.Contains(t, markdown, "## Endpoints")
		assert.Contains(t, markdown, "### Create Entry")
		assert.Contains(t, markdown, "### List Entries")
		assert.Contains(t, markdown, "### Update Entry")
		assert.Contains(t, markdown, "### Delete Entry")
		assert.Contains(t, markdown, "**SEO Enabled:** ✓")
	})
}

func TestCompleteWorkflow(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor_complete@test.com", "password", "editor")
	admin := testutils.CreateTestUser(t, database.DB, "admin_complete@test.com", "password", "admin")

	editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)
	adminToken := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	ct := &models.ContentType{Name: "CompleteTest", Slug: "complete-test", EnableSEO: true}
	database.DB.Create(ct)

	database.DB.Create(&models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "title",
		Type:          "string",
		Required:      true,
	})

	database.DB.Create(&models.ContentField{
		ContentTypeID: ct.ID,
		Name:          "meta_title",
		Type:          "string",
		Required:      false,
		IsSEO:         true,
	})

	var entryID uint

	t.Run("Step 1 - Editor creates entry", func(t *testing.T) {
		fields := map[string]string{
			"title":      "Complete Workflow Test",
			"meta_title": "SEO Title",
		}

		resp, err := testutils.MakeMultipartRequest(app, "POST", "/content/"+fmt.Sprint(ct.ID)+"/entries", fields, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		entryData := result.Data.(map[string]interface{})
		entryID = uint(entryData["id"].(float64))
	})

	t.Run("Step 2 - Editor requests review", func(t *testing.T) {
		body := map[string]interface{}{
			"comment": "Please review",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entryID)+"/request-review", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("Step 3 - Editor adds comment", func(t *testing.T) {
		body := map[string]interface{}{
			"comment":    "Ready for review",
			"is_private": false,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entryID)+"/comments", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
	})

	t.Run("Step 4 - Admin assigns to reviewer", func(t *testing.T) {
		body := map[string]interface{}{
			"assigned_to": editor.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entryID)+"/assign", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
	})

	t.Run("Step 5 - Editor marks ready for approval", func(t *testing.T) {
		body := map[string]interface{}{
			"status":  "ready_for_approval",
			"comment": "Ready for approval",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entryID)+"/status", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("Step 6 - Admin approves", func(t *testing.T) {
		body := map[string]interface{}{
			"comment": "Approved",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entryID)+"/approve", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("Step 7 - Admin publishes", func(t *testing.T) {
		body := map[string]interface{}{
			"comment": "Publishing now",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/workflow/entries/"+fmt.Sprint(entryID)+"/publish", body, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var entry models.ContentEntry
		database.DB.First(&entry, entryID)
		assert.Equal(t, models.StatusPublished, entry.Status)
		assert.NotNil(t, entry.PublishedAt)
	})

	t.Run("Step 8 - Get workflow history", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/workflow/entries/"+fmt.Sprint(entryID)+"/history", nil, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)

		if result.Data != nil {
			history := result.Data.([]interface{})
			assert.GreaterOrEqual(t, len(history), 3)
		}
	})

	t.Run("Step 9 - Generate API reference", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/types/"+fmt.Sprint(ct.ID)+"/api-reference", nil, adminToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var apiRef map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &apiRef)
		assert.Equal(t, "CompleteTest", apiRef["content_type"])
		assert.True(t, apiRef["seo_enabled"].(bool))
	})

	t.Run("Step 10 - Get SEO preview", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/content/entries/"+fmt.Sprint(entryID)+"/seo-preview", nil, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)

		if result.Data != nil {
			seoData := result.Data.(map[string]interface{})
			assert.Equal(t, "SEO Title", seoData["meta_title"])
		}
	})
}

// ============================================
// CONTENT RELATION TESTS
// ============================================

func TestCreateRelationHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@relation.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create two content types
	blogCT := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(blogCT)

	categoryCT := &models.ContentType{Name: "Category", Slug: "category"}
	database.DB.Create(categoryCT)

	// Create entries for both
	blogEntry := &models.ContentEntry{
		ContentTypeID: blogCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Blog Post"}`)),
	}
	database.DB.Create(blogEntry)

	categoryEntry := &models.ContentEntry{
		ContentTypeID: categoryCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"name":"Technology"}`)),
	}
	database.DB.Create(categoryEntry)

	t.Run("Success - Create relation", func(t *testing.T) {
		body := map[string]interface{}{
			"to_content_id": categoryEntry.ID,
			"relation_type": "belongs_to",
		}

		resp, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		// Check if response body is not empty
		if resp.Body.Len() > 0 {
			var result testutils.StandardResponse
			testutils.ParseResponse(t, resp, &result)
			assert.True(t, result.Success)

			if result.Data != nil {
				relationData := result.Data.(map[string]interface{})
				assert.Equal(t, float64(blogEntry.ID), relationData["from_content_id"])
				assert.Equal(t, float64(categoryEntry.ID), relationData["to_content_id"])
				assert.Equal(t, "belongs_to", relationData["relation_type"])
			}
		}

		// Verify relation was actually created in database
		var relation models.ContentRelation
		err = database.DB.Where("from_content_id = ? AND to_content_id = ?",
			blogEntry.ID, categoryEntry.ID).First(&relation).Error
		assert.NoError(t, err)
		assert.Equal(t, "belongs_to", relation.RelationType)
	})

	t.Run("Success - Create multiple relations", func(t *testing.T) {
		// Create another category
		categoryEntry2 := &models.ContentEntry{
			ContentTypeID: categoryCT.ID,
			CreatedBy:     editor.ID,
			Status:        models.StatusDraft,
			Data:          datatypes.JSON([]byte(`{"name":"Programming"}`)),
		}
		database.DB.Create(categoryEntry2)

		body := map[string]interface{}{
			"to_content_id": categoryEntry2.ID,
			"relation_type": "has_many",
		}

		resp, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		// Verify in database
		var relation models.ContentRelation
		err = database.DB.Where("from_content_id = ? AND to_content_id = ?",
			blogEntry.ID, categoryEntry2.ID).First(&relation).Error
		assert.NoError(t, err)
		assert.Equal(t, "has_many", relation.RelationType)
	})

	t.Run("Error - Missing to_content_id", func(t *testing.T) {
		body := map[string]interface{}{
			"relation_type": "belongs_to",
			// to_content_id missing - akan jadi 0
		}

		resp, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Missing relation_type", func(t *testing.T) {
		body := map[string]interface{}{
			"to_content_id": categoryEntry.ID,
			// relation_type missing
		}

		resp, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Invalid from_content_id", func(t *testing.T) {
		body := map[string]interface{}{
			"to_content_id": categoryEntry.ID,
			"relation_type": "belongs_to",
		}

		resp, err := testutils.MakeRequest(app, "POST",
			"/content/invalid/relations", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestListRelationsHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@list.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Setup content types and entries
	blogCT := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(blogCT)

	categoryCT := &models.ContentType{Name: "Category", Slug: "category"}
	database.DB.Create(categoryCT)

	blogEntry := &models.ContentEntry{
		ContentTypeID: blogCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Blog with Relations"}`)),
	}
	database.DB.Create(blogEntry)

	// Create multiple categories and relations
	for i := 1; i <= 3; i++ {
		categoryEntry := &models.ContentEntry{
			ContentTypeID: categoryCT.ID,
			CreatedBy:     editor.ID,
			Status:        models.StatusDraft,
			Data:          datatypes.JSON([]byte(fmt.Sprintf(`{"name":"Category %d"}`, i))),
		}
		database.DB.Create(categoryEntry)

		relation := &models.ContentRelation{
			FromContentID: blogEntry.ID,
			ToContentID:   categoryEntry.ID,
			RelationType:  "has_many",
		}
		database.DB.Create(relation)
	}

	t.Run("Success - List all relations", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		relations := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(relations), 3)
	})

	t.Run("Success - Empty relations list", func(t *testing.T) {
		emptyEntry := &models.ContentEntry{
			ContentTypeID: blogCT.ID,
			CreatedBy:     editor.ID,
			Status:        models.StatusDraft,
			Data:          datatypes.JSON([]byte(`{"title":"No Relations"}`)),
		}
		database.DB.Create(emptyEntry)

		resp, err := testutils.MakeRequest(app, "GET",
			fmt.Sprintf("/content/%d/relations", emptyEntry.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		relations := result.Data.([]interface{})
		assert.Equal(t, 0, len(relations))
	})

	t.Run("Error - Invalid content ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET",
			"/content/invalid/relations", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestDeleteRelationHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@delete.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	// Setup
	blogCT := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(blogCT)

	categoryCT := &models.ContentType{Name: "Category", Slug: "category"}
	database.DB.Create(categoryCT)

	blogEntry := &models.ContentEntry{
		ContentTypeID: blogCT.ID,
		CreatedBy:     admin.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Blog"}`)),
	}
	database.DB.Create(blogEntry)

	categoryEntry := &models.ContentEntry{
		ContentTypeID: categoryCT.ID,
		CreatedBy:     admin.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"name":"Tech"}`)),
	}
	database.DB.Create(categoryEntry)

	relation := &models.ContentRelation{
		FromContentID: blogEntry.ID,
		ToContentID:   categoryEntry.ID,
		RelationType:  "belongs_to",
	}
	database.DB.Create(relation)

	t.Run("Success - Delete relation", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE",
			fmt.Sprintf("/content/relations/%d", relation.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)

		// Verify deletion
		var deleted models.ContentRelation
		result := database.DB.First(&deleted, relation.ID)
		assert.Error(t, result.Error)
	})

	t.Run("Error - Relation not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE",
			"/content/relations/9999", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - Invalid relation ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE",
			"/content/relations/invalid", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestRelationTypes(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@types.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Setup content types
	postCT := &models.ContentType{Name: "Post", Slug: "post"}
	database.DB.Create(postCT)

	authorCT := &models.ContentType{Name: "Author", Slug: "author"}
	database.DB.Create(authorCT)

	tagCT := &models.ContentType{Name: "Tag", Slug: "tag"}
	database.DB.Create(tagCT)

	// Create entries
	postEntry := &models.ContentEntry{
		ContentTypeID: postCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Post"}`)),
	}
	database.DB.Create(postEntry)

	authorEntry := &models.ContentEntry{
		ContentTypeID: authorCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"name":"John Doe"}`)),
	}
	database.DB.Create(authorEntry)

	tagEntry := &models.ContentEntry{
		ContentTypeID: tagCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"name":"Technology"}`)),
	}
	database.DB.Create(tagEntry)

	relationTypes := []string{"belongs_to", "has_many", "has_one", "many_to_many"}

	t.Run("Success - Test different relation types", func(t *testing.T) {
		for _, relType := range relationTypes {
			body := map[string]interface{}{
				"to_content_id": authorEntry.ID,
				"relation_type": relType,
			}

			resp, err := testutils.MakeRequest(app, "POST",
				fmt.Sprintf("/content/%d/relations", postEntry.ID), body, token)
			assert.NoError(t, err)
			assert.Equal(t, 201, resp.Code)

			var result testutils.StandardResponse
			testutils.ParseResponse(t, resp, &result)
			relationData := result.Data.(map[string]interface{})
			assert.Equal(t, relType, relationData["relation_type"])
		}
	})

	t.Run("Success - Bidirectional relation", func(t *testing.T) {
		// Post belongs_to Author
		body1 := map[string]interface{}{
			"to_content_id": authorEntry.ID,
			"relation_type": "belongs_to",
		}
		resp1, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", postEntry.ID), body1, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp1.Code)

		// Author has_many Posts
		body2 := map[string]interface{}{
			"to_content_id": postEntry.ID,
			"relation_type": "has_many",
		}
		resp2, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", authorEntry.ID), body2, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp2.Code)
	})
}

func TestComplexRelations(t *testing.T) {
	app := testutils.SetupTestApp(t)

	editor := testutils.CreateTestUser(t, database.DB, "editor@complex.com", "password", "editor")
	token := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

	// Create a complex content structure
	// Blog -> Categories (many_to_many)
	// Blog -> Author (belongs_to)
	// Blog -> Comments (has_many)

	blogCT := &models.ContentType{Name: "Blog", Slug: "blog"}
	database.DB.Create(blogCT)

	categoryCT := &models.ContentType{Name: "Category", Slug: "category"}
	database.DB.Create(categoryCT)

	authorCT := &models.ContentType{Name: "Author", Slug: "author"}
	database.DB.Create(authorCT)

	commentCT := &models.ContentType{Name: "Comment", Slug: "comment"}
	database.DB.Create(commentCT)

	// Create main blog entry
	blogEntry := &models.ContentEntry{
		ContentTypeID: blogCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"title":"Complex Blog Post"}`)),
	}
	database.DB.Create(blogEntry)

	// Create author
	authorEntry := &models.ContentEntry{
		ContentTypeID: authorCT.ID,
		CreatedBy:     editor.ID,
		Status:        models.StatusDraft,
		Data:          datatypes.JSON([]byte(`{"name":"Jane Doe"}`)),
	}
	database.DB.Create(authorEntry)

	t.Run("Success - Create complex relations", func(t *testing.T) {
		// Blog belongs to Author
		body := map[string]interface{}{
			"to_content_id": authorEntry.ID,
			"relation_type": "belongs_to",
		}
		resp, err := testutils.MakeRequest(app, "POST",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		// Blog has many Categories
		for i := 1; i <= 3; i++ {
			categoryEntry := &models.ContentEntry{
				ContentTypeID: categoryCT.ID,
				CreatedBy:     editor.ID,
				Status:        models.StatusDraft,
				Data:          datatypes.JSON([]byte(fmt.Sprintf(`{"name":"Category %d"}`, i))),
			}
			database.DB.Create(categoryEntry)

			body := map[string]interface{}{
				"to_content_id": categoryEntry.ID,
				"relation_type": "many_to_many",
			}
			resp, err := testutils.MakeRequest(app, "POST",
				fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
			assert.NoError(t, err)
			assert.Equal(t, 201, resp.Code)
		}

		// Blog has many Comments
		for i := 1; i <= 5; i++ {
			commentEntry := &models.ContentEntry{
				ContentTypeID: commentCT.ID,
				CreatedBy:     editor.ID,
				Status:        models.StatusDraft,
				Data:          datatypes.JSON([]byte(fmt.Sprintf(`{"text":"Comment %d"}`, i))),
			}
			database.DB.Create(commentEntry)

			body := map[string]interface{}{
				"to_content_id": commentEntry.ID,
				"relation_type": "has_many",
			}
			resp, err := testutils.MakeRequest(app, "POST",
				fmt.Sprintf("/content/%d/relations", blogEntry.ID), body, token)
			assert.NoError(t, err)
			assert.Equal(t, 201, resp.Code)
		}
	})

	t.Run("Success - List all relations of blog", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET",
			fmt.Sprintf("/content/%d/relations", blogEntry.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		relations := result.Data.([]interface{})

		// Should have: 1 author + 3 categories + 5 comments = 9 relations
		assert.GreaterOrEqual(t, len(relations), 9)
	})
}
