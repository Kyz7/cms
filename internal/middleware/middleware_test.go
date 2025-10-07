package middleware_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/testutils"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// ========== ROLE MANAGEMENT TESTS ==========

func TestCreateRoleHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)

	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Create role with basic permissions", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "moderator",
			"description": "Content moderator",
			"permissions": []map[string]interface{}{
				{
					"module": "ContentEntry",
					"action": "read",
				},
				{
					"module": "ContentEntry",
					"action": "update",
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		// Verify permissions were created
		role := result.Data.(map[string]interface{})
		perms := role["permissions"].([]interface{})
		assert.Equal(t, 2, len(perms))
	})

	t.Run("Success - Create role with field scope", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "seo_manager",
			"description": "SEO content manager",
			"permissions": []map[string]interface{}{
				{
					"module":      "ContentEntry",
					"action":      "read",
					"field_scope": "all",
				},
				{
					"module":      "ContentEntry",
					"action":      "update",
					"field_scope": "seo_only",
				},
				{
					"module":      "SEO",
					"action":      "create",
					"field_scope": "all",
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Create role with allowed fields", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "blog_editor",
			"description": "Blog specific editor",
			"permissions": []map[string]interface{}{
				{
					"module":         "ContentEntry",
					"action":         "update",
					"field_scope":    "custom",
					"allowed_fields": []string{"title", "slug", "content", "featured_image"},
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Create role with denied fields", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "product_editor",
			"description": "Product editor without pricing access",
			"permissions": []map[string]interface{}{
				{
					"module":        "ContentEntry",
					"action":        "update",
					"field_scope":   "custom",
					"denied_fields": []string{"cost_price", "supplier_info", "internal_notes"},
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
		testutils.AssertSuccess(t, resp)
	})

	t.Run("Success - Create role with content type restriction", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "blog_specialist",
			"description": "Only manages blog content",
			"permissions": []map[string]interface{}{
				{
					"module":           "ContentEntry",
					"action":           "create",
					"content_type_ids": []uint{1, 2},
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)
		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Duplicate role name", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "moderator",
			"description": "Duplicate",
			"permissions": []map[string]interface{}{},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)
		testutils.AssertError(t, resp, "CONFLICT")
	})

	t.Run("Error - Missing role name", func(t *testing.T) {
		body := map[string]interface{}{
			"description": "No name",
			"permissions": []map[string]interface{}{},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)
		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Non-admin cannot create role", func(t *testing.T) {
		editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
		editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

		body := map[string]interface{}{
			"name":        "test_role",
			"description": "Test",
			"permissions": []map[string]interface{}{},
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles", body, editorToken)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.Code)
		testutils.AssertError(t, resp, "FORBIDDEN")
	})
}

func TestListRolesHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - List all roles with permissions", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/roles", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		roles := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(roles), 4) // admin, editor, viewer, manager

		// Check that permissions are loaded
		for _, r := range roles {
			role := r.(map[string]interface{})
			assert.NotNil(t, role["permissions"])
		}
	})
}

func TestGetRoleHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	var editorRole models.Role
	db.Where("name = ?", "editor").First(&editorRole)

	t.Run("Success - Get role by ID with permissions", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/roles/"+fmt.Sprint(editorRole.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		role := result.Data.(map[string]interface{})
		assert.Equal(t, "editor", role["name"])
		assert.NotNil(t, role["permissions"])
	})

	t.Run("Error - Role not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/roles/9999", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - Invalid role ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/roles/invalid", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)
		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

func TestUpdateRoleHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Update role name and description", func(t *testing.T) {
		role := &models.Role{Name: "old_name", Description: "Old description"}
		db.Create(role)

		body := map[string]interface{}{
			"name":        "new_name",
			"description": "New description",
			"permissions": []map[string]interface{}{
				{
					"module": "ContentEntry",
					"action": "read",
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/roles/"+fmt.Sprint(role.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		// Verify update
		var updated models.Role
		db.First(&updated, role.ID)
		assert.Equal(t, "new_name", updated.Name)
		assert.Equal(t, "New description", updated.Description)
	})

	t.Run("Success - Update permissions", func(t *testing.T) {
		role := &models.Role{Name: "test_role", Description: "Test"}
		db.Create(role)

		// Create initial permissions
		perm := models.Permission{RoleID: role.ID, Module: "ContentEntry", Action: "read"}
		db.Create(&perm)

		body := map[string]interface{}{
			"name":        "test_role",
			"description": "Test",
			"permissions": []map[string]interface{}{
				{
					"module": "ContentEntry",
					"action": "create",
				},
				{
					"module": "ContentEntry",
					"action": "update",
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/roles/"+fmt.Sprint(role.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		// Verify old permissions deleted and new ones created
		var permissions []models.Permission
		db.Where("role_id = ?", role.ID).Find(&permissions)
		assert.Equal(t, 2, len(permissions))
		assert.Equal(t, "create", permissions[0].Action)
	})

	t.Run("Success - Update with field scope changes", func(t *testing.T) {
		role := &models.Role{Name: "scope_test", Description: "Test"}
		db.Create(role)

		body := map[string]interface{}{
			"name":        "scope_test",
			"description": "Test",
			"permissions": []map[string]interface{}{
				{
					"module":      "ContentEntry",
					"action":      "update",
					"field_scope": "seo_only",
				},
			},
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/roles/"+fmt.Sprint(role.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		// Verify field scope
		var perm models.Permission
		db.Where("role_id = ?", role.ID).First(&perm)
		assert.Equal(t, "seo_only", perm.FieldScope)
	})

	t.Run("Error - Role not found", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "test",
			"description": "test",
			"permissions": []map[string]interface{}{},
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/roles/9999", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

func TestDeleteRoleHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Delete unused role", func(t *testing.T) {
		role := &models.Role{Name: "to_delete", Description: "Will be deleted"}
		db.Create(role)

		// Add some permissions
		perm := models.Permission{RoleID: role.ID, Module: "ContentEntry", Action: "read"}
		db.Create(&perm)

		resp, err := testutils.MakeRequest(app, "DELETE", "/roles/"+fmt.Sprint(role.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)

		// Verify role deleted
		var deleted models.Role
		err = db.First(&deleted, role.ID).Error
		assert.Error(t, err)

		// Verify permissions also deleted
		var perms []models.Permission
		db.Where("role_id = ?", role.ID).Find(&perms)
		assert.Equal(t, 0, len(perms))
	})

	t.Run("Error - Cannot delete role assigned to users", func(t *testing.T) {
		var viewerRole models.Role
		db.Where("name = ?", "viewer").First(&viewerRole)

		// Create user with this role
		testutils.CreateTestUser(t, db, "user_with_role@test.com", "password", "viewer")

		resp, err := testutils.MakeRequest(app, "DELETE", "/roles/"+fmt.Sprint(viewerRole.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)
		testutils.AssertError(t, resp, "CONFLICT")
	})

	t.Run("Error - Role not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE", "/roles/9999", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

func TestAssignRoleToUserHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Assign role to user", func(t *testing.T) {
		user := testutils.CreateTestUser(t, db, "test@test.com", "password", "viewer")

		var editorRole models.Role
		db.Where("name = ?", "editor").First(&editorRole)

		body := map[string]interface{}{
			"user_id": user.ID,
			"role_id": editorRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		// Verify assignment
		var updated models.User
		db.Preload("Role").First(&updated, user.ID)
		assert.Equal(t, editorRole.ID, updated.RoleID)
		assert.Equal(t, "editor", updated.Role.Name)
	})

	t.Run("Success - Reassign user to different role", func(t *testing.T) {
		user := testutils.CreateTestUser(t, db, "reassign@test.com", "password", "editor")

		var viewerRole models.Role
		db.Where("name = ?", "viewer").First(&viewerRole)

		body := map[string]interface{}{
			"user_id": user.ID,
			"role_id": viewerRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		// Verify reassignment
		var updated models.User
		db.Preload("Role").First(&updated, user.ID)
		assert.Equal(t, viewerRole.ID, updated.RoleID)
	})

	t.Run("Error - Missing user_id", func(t *testing.T) {
		var editorRole models.Role
		db.Where("name = ?", "editor").First(&editorRole)

		body := map[string]interface{}{
			"role_id": editorRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)
		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Missing role_id", func(t *testing.T) {
		user := testutils.CreateTestUser(t, db, "test2@test.com", "password", "viewer")

		body := map[string]interface{}{
			"user_id": user.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)
		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Role not found", func(t *testing.T) {
		user := testutils.CreateTestUser(t, db, "test3@test.com", "password", "viewer")

		body := map[string]interface{}{
			"user_id": user.ID,
			"role_id": 9999,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
		testutils.AssertError(t, resp, "NOT_FOUND")
	})

	t.Run("Error - User not found", func(t *testing.T) {
		var editorRole models.Role
		db.Where("name = ?", "editor").First(&editorRole)

		body := map[string]interface{}{
			"user_id": 9999,
			"role_id": editorRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

func TestDuplicateRoleHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Duplicate role with permissions", func(t *testing.T) {
		// Create original role
		role := &models.Role{Name: "original", Description: "Original role"}
		db.Create(role)

		// Add permissions
		perms := []models.Permission{
			{RoleID: role.ID, Module: "ContentEntry", Action: "read"},
			{RoleID: role.ID, Module: "ContentEntry", Action: "update"},
			{RoleID: role.ID, Module: "Media", Action: "read"},
		}
		db.Create(&perms)

		body := map[string]interface{}{
			"name": "duplicated",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/"+fmt.Sprint(role.ID)+"/duplicate", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		newRole := result.Data.(map[string]interface{})

		assert.Equal(t, "duplicated", newRole["name"])
		assert.Equal(t, "Original role (Copy)", newRole["description"])

		// Verify permissions were copied
		newPerms := newRole["permissions"].([]interface{})
		assert.Equal(t, 3, len(newPerms))
	})

	t.Run("Error - Missing new name", func(t *testing.T) {
		var editorRole models.Role
		db.Where("name = ?", "editor").First(&editorRole)

		body := map[string]interface{}{}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/"+fmt.Sprint(editorRole.ID)+"/duplicate", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)
		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Original role not found", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "new_name",
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/9999/duplicate", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)
		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

// ========== PERMISSION FILTERING TESTS ==========

func TestPermissionFieldFiltering(t *testing.T) {
	db := database.DB
	testutils.CreateTestRoles(t, db)
	// Create content type
	ct := &models.ContentType{
		Name: "Article",
		Fields: []models.ContentField{
			{Name: "title", Type: "text", IsSEO: false},
			{Name: "content", Type: "richtext", IsSEO: false},
			{Name: "author", Type: "text", IsSEO: false},
		},
		SEOFields: []models.ContentField{
			{Name: "meta_title", Type: "text", IsSEO: true},
			{Name: "meta_description", Type: "text", IsSEO: true},
		},
	}
	db.Create(ct)

	t.Run("SEO only field scope", func(t *testing.T) {
		// Create role with SEO-only permission
		role := &models.Role{Name: "seo_tester", Description: "Test"}
		db.Create(role)

		perm := models.Permission{
			RoleID:     role.ID,
			Module:     "ContentEntry",
			Action:     "update",
			FieldScope: "seo_only",
		}
		db.Create(&perm)

		user := &models.User{
			Name:   "SEO User",
			Email:  "seo@test.com",
			RoleID: role.ID,
		}
		db.Create(user)

		// Test filtering
		data := map[string]interface{}{
			"title":            "Test",
			"content":          "Content",
			"meta_title":       "Meta Title",
			"meta_description": "Meta Desc",
		}

		filtered, err := middleware.FilterFieldsByPermission(user.ID, "update", data, ct.ID)
		assert.NoError(t, err)
		assert.NotContains(t, filtered, "title")
		assert.NotContains(t, filtered, "content")
		assert.Contains(t, filtered, "meta_title")
		assert.Contains(t, filtered, "meta_description")
	})

	t.Run("Non-SEO only field scope", func(t *testing.T) {
		role := &models.Role{Name: "content_tester", Description: "Test"}
		db.Create(role)

		perm := models.Permission{
			RoleID:     role.ID,
			Module:     "ContentEntry",
			Action:     "update",
			FieldScope: "non_seo_only",
		}
		db.Create(&perm)

		user := &models.User{
			Name:   "Content User",
			Email:  "content@test.com",
			RoleID: role.ID,
		}
		db.Create(user)

		// Similar test for non-SEO fields
	})

	t.Run("Custom allowed fields", func(t *testing.T) {
		role := &models.Role{Name: "custom_tester", Description: "Test"}
		db.Create(role)

		allowedFields := []string{"title", "content"}
		allowedJSON, _ := json.Marshal(allowedFields)

		perm := models.Permission{
			RoleID:        role.ID,
			Module:        "ContentEntry",
			Action:        "update",
			FieldScope:    "custom",
			AllowedFields: datatypes.JSON(allowedJSON),
		}
		db.Create(&perm)

		user := &models.User{
			Name:   "Custom User",
			Email:  "custom@test.com",
			RoleID: role.ID,
		}
		db.Create(user)
	})

	t.Run("Custom denied fields", func(t *testing.T) {
		role := &models.Role{Name: "denied_tester", Description: "Test"}
		db.Create(role)

		deniedFields := []string{"author"}
		deniedJSON, _ := json.Marshal(deniedFields)

		perm := models.Permission{
			RoleID:       role.ID,
			Module:       "ContentEntry",
			Action:       "update",
			FieldScope:   "custom",
			DeniedFields: datatypes.JSON(deniedJSON),
		}
		db.Create(&perm)

		user := &models.User{
			Name:   "Denied User",
			Email:  "denied@test.com",
			RoleID: role.ID,
		}
		db.Create(user)
	})
}

func TestContentTypeRestrictions(t *testing.T) {
	db := database.DB

	testutils.CreateTestRoles(t, db)

	// Create content types dengan slug unique
	ct1 := &models.ContentType{
		Name: "Blog",
		Slug: "blog-test-" + fmt.Sprint(time.Now().UnixNano()), // Unique slug
	}
	ct2 := &models.ContentType{
		Name: "Product",
		Slug: "product-test-" + fmt.Sprint(time.Now().UnixNano()), // Unique slug
	}
	db.Create(ct1)
	db.Create(ct2)

	t.Run("User can only access specific content types", func(t *testing.T) {
		role := &models.Role{Name: "blog_only", Description: "Blog only"}
		db.Create(role)

		contentTypeIDs := []uint{ct1.ID}
		ctJSON, _ := json.Marshal(contentTypeIDs)

		perm := models.Permission{
			RoleID:         role.ID,
			Module:         "ContentEntry",
			Action:         "create",
			ContentTypeIDs: datatypes.JSON(ctJSON),
		}
		db.Create(&perm)

		user := &models.User{
			Name:   "Blog User",
			Email:  "bloguser@test.com",
			RoleID: role.ID,
		}
		db.Create(user)

		// Test should verify user can only access ct1, not ct2
	})
}
