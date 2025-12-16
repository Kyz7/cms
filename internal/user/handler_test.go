package user_test

import (
	"fmt"
	"testing"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// ========== USER TESTS ==========

func TestCreateUserHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	var viewerRole models.Role
	db.Where("name = ?", "viewer").First(&viewerRole)

	t.Run("Success - Create user", func(t *testing.T) {
		body := map[string]interface{}{
			"name":     "New User",
			"email":    "newuser@test.com",
			"password": "password123",
			"role_id":  viewerRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/users", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Duplicate email", func(t *testing.T) {
		body := map[string]interface{}{
			"name":     "Duplicate",
			"email":    "newuser@test.com", // Already exists
			"password": "password123",
			"role_id":  viewerRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/users", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)

		testutils.AssertError(t, resp, "CONFLICT")
	})

	t.Run("Error - Missing required fields", func(t *testing.T) {
		body := map[string]interface{}{
			"email": "incomplete@test.com",
			// Missing name and password
		}

		resp, err := testutils.MakeRequest(app, "POST", "/users", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 422, resp.Code)

		testutils.AssertError(t, resp, "VALIDATION_ERROR")
	})

	t.Run("Error - Non-admin cannot create user", func(t *testing.T) {
		viewer := testutils.CreateTestUser(t, db, "viewer@test.com", "password", "viewer")
		viewerToken := testutils.GetAuthToken(t, viewer.ID, viewer.Role.Name)

		body := map[string]interface{}{
			"name":     "Test",
			"email":    "test@test.com",
			"password": "password123",
			"role_id":  viewerRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/users", body, viewerToken)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.Code)

		testutils.AssertError(t, resp, "FORBIDDEN")
	})
}

func TestListUsersHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	// Create additional users
	testutils.CreateTestUser(t, db, "user1@test.com", "password", "viewer")
	testutils.CreateTestUser(t, db, "user2@test.com", "password", "editor")

	t.Run("Success - List all users", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/users", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		users := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(users), 3)
	})
}

func TestGetUserHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	user := testutils.CreateTestUser(t, db, "test@test.com", "password", "viewer")

	t.Run("Success - Get user by ID", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/users/"+fmt.Sprint(user.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - User not found", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/users/9999", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.Code)

		testutils.AssertError(t, resp, "NOT_FOUND")
	})
}

func TestUpdateUserHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	user := testutils.CreateTestUser(t, db, "test@test.com", "password", "viewer")

	var editorRole models.Role
	db.Where("name = ?", "editor").First(&editorRole)

	t.Run("Success - Update user", func(t *testing.T) {
		body := map[string]interface{}{
			"name":    "Updated Name",
			"email":   "updated@test.com",
			"role_id": editorRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/users/"+fmt.Sprint(user.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)

		// Verify update
		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "updated@test.com", updated.Email)
	})

	t.Run("Error - Email already taken", func(t *testing.T) {
		another := testutils.CreateTestUser(t, db, "another@test.com", "password", "viewer")

		body := map[string]interface{}{
			"email": "updated@test.com", // Already taken
		}

		resp, err := testutils.MakeRequest(app, "PUT", "/users/"+fmt.Sprint(another.ID), body, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)

		testutils.AssertError(t, resp, "CONFLICT")
	})
}

func TestDeleteUserHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Delete user", func(t *testing.T) {
		user := testutils.CreateTestUser(t, db, "todelete@test.com", "password", "viewer")

		resp, err := testutils.MakeRequest(app, "DELETE", "/users/"+fmt.Sprint(user.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)
	})

	t.Run("Error - Cannot delete own account", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "DELETE", "/users/"+fmt.Sprint(admin.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.Code)

		testutils.AssertError(t, resp, "BAD_REQUEST")
	})
}

// ========== ROLE TESTS ==========

func TestCreateRoleHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - Create role with permissions", func(t *testing.T) {
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

		testutils.AssertSuccess(t, resp)
	})

	t.Run("Error - Duplicate role name", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "moderator", // Already exists
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
}

func TestListRolesHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	t.Run("Success - List all roles", func(t *testing.T) {
		resp, err := testutils.MakeRequest(app, "GET", "/roles", nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		var result testutils.StandardResponse
		testutils.ParseResponse(t, resp, &result)
		assert.True(t, result.Success)

		roles := result.Data.([]interface{})
		assert.GreaterOrEqual(t, len(roles), 3) // admin, editor, viewer
	})
}

func TestAssignRoleToUserHandler(t *testing.T) {
	app := testutils.SetupTestApp(t)
	db := database.DB

	admin := testutils.CreateTestUser(t, db, "admin@test.com", "password", "admin")
	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

	user := testutils.CreateTestUser(t, db, "test@test.com", "password", "viewer")

	var editorRole models.Role
	db.Where("name = ?", "editor").First(&editorRole)

	t.Run("Success - Assign role to user", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id": user.ID,
			"role_id": editorRole.ID,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		testutils.AssertSuccess(t, resp)

		// Verify assignment
		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, editorRole.ID, updated.RoleID)
	})

	t.Run("Error - Role not found", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id": user.ID,
			"role_id": 9999,
		}

		resp, err := testutils.MakeRequest(app, "POST", "/roles/assign", body, token)
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
		role := &models.Role{Name: "todelete", Description: "To be deleted"}
		db.Create(role)

		resp, err := testutils.MakeRequest(app, "DELETE", "/roles/"+fmt.Sprint(role.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 204, resp.Code)
	})

	t.Run("Error - Cannot delete role assigned to users", func(t *testing.T) {
		var viewerRole models.Role
		db.Where("name = ?", "viewer").First(&viewerRole)

		testutils.CreateTestUser(t, db, "viewer@test.com", "password", "viewer")

		resp, err := testutils.MakeRequest(app, "DELETE", "/roles/"+fmt.Sprint(viewerRole.ID), nil, token)
		assert.NoError(t, err)
		assert.Equal(t, 409, resp.Code)

		testutils.AssertError(t, resp, "CONFLICT")
	})
}
