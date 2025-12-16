package project_test

// import (
// 	"fmt"
// 	"testing"

// 	"github.com/Kyz7/cms/internal/database"
// 	"github.com/Kyz7/cms/internal/models"
// 	"github.com/Kyz7/cms/internal/testutils"
// 	"github.com/stretchr/testify/assert"
// )

// func TestCreateProjectHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	t.Run("Success - Create project", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"name":        "Test Project",
// 			"description": "Test project description",
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", "/projects", body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 201, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		data := result.Data.(map[string]interface{})
// 		assert.Equal(t, "Test Project", data["name"])
// 		assert.NotNil(t, data["id"])
// 	})

// 	t.Run("Error - Missing required fields", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"description": "Test project description",
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", "/projects", body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 422, resp.Code)

// 		testutils.AssertError(t, resp, "VALIDATION_ERROR")
// 	})

// 	t.Run("Error - Unauthorized", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"name": "Test Project",
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", "/projects", body, "")
// 		assert.NoError(t, err)
// 		assert.Equal(t, 401, resp.Code)
// 	})
// }

// func TestGetProjectHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	// Add admin as owner
// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	t.Run("Success - Get project", func(t *testing.T) {
// 		resp, err := testutils.MakeRequest(app, "GET", fmt.Sprintf("/projects/%d", project.ID), nil, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		data := result.Data.(map[string]interface{})
// 		assert.Equal(t, "Test Project", data["name"])
// 	})

// 	t.Run("Error - Not a member", func(t *testing.T) {
// 		otherUser := testutils.CreateTestUser(t, database.DB, "other@test.com", "password", "editor")
// 		otherToken := testutils.GetAuthToken(t, otherUser.ID, otherUser.Role.Name)

// 		resp, err := testutils.MakeRequest(app, "GET", fmt.Sprintf("/projects/%d", project.ID), nil, otherToken)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 403, resp.Code)
// 	})
// }

// func TestListProjectsHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create projects
// 	project1 := &models.Project{
// 		Name:      "Project 1",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project1)

// 	member1 := &models.ProjectMember{
// 		ProjectID: project1.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member1)

// 	project2 := &models.Project{
// 		Name:      "Project 2",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project2)

// 	member2 := &models.ProjectMember{
// 		ProjectID: project2.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleAdmin,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member2)

// 	t.Run("Success - List projects", func(t *testing.T) {
// 		resp, err := testutils.MakeRequest(app, "GET", "/projects", nil, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		data := result.Data.([]interface{})
// 		assert.GreaterOrEqual(t, len(data), 2)
// 	})
// }

// func TestUpdateProjectHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	t.Run("Success - Update project", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"name":        "Updated Project",
// 			"description": "Updated description",
// 		}

// 		resp, err := testutils.MakeRequest(app, "PUT", fmt.Sprintf("/projects/%d", project.ID), body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		data := result.Data.(map[string]interface{})
// 		assert.Equal(t, "Updated Project", data["name"])
// 	})

// 	t.Run("Error - Not admin or owner", func(t *testing.T) {
// 		viewer := testutils.CreateTestUser(t, database.DB, "viewer@test.com", "password", "viewer")
// 		viewerToken := testutils.GetAuthToken(t, viewer.ID, viewer.Role.Name)

// 		// Add viewer as member with viewer role
// 		viewerMember := &models.ProjectMember{
// 			ProjectID: project.ID,
// 			UserID:    viewer.ID,
// 			Role:      models.ProjectRoleViewer,
// 			Status:    "active",
// 		}
// 		database.DB.Create(viewerMember)

// 		body := map[string]interface{}{
// 			"name": "Updated Project",
// 		}

// 		resp, err := testutils.MakeRequest(app, "PUT", fmt.Sprintf("/projects/%d", project.ID), body, viewerToken)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 403, resp.Code)
// 	})
// }

// func TestDeleteProjectHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	t.Run("Success - Delete project", func(t *testing.T) {
// 		resp, err := testutils.MakeRequest(app, "DELETE", fmt.Sprintf("/projects/%d", project.ID), nil, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		// Verify project is deleted
// 		var deletedProject models.Project
// 		err = database.DB.First(&deletedProject, project.ID).Error
// 		assert.Error(t, err) // Should not be found
// 	})

// 	t.Run("Error - Not owner", func(t *testing.T) {
// 		// Create another project
// 		project2 := &models.Project{
// 			Name:      "Test Project 2",
// 			CreatedBy: admin.ID,
// 		}
// 		database.DB.Create(project2)

// 		editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
// 		editorToken := testutils.GetAuthToken(t, editor.ID, editor.Role.Name)

// 		editorMember := &models.ProjectMember{
// 			ProjectID: project2.ID,
// 			UserID:    editor.ID,
// 			Role:      models.ProjectRoleEditor,
// 			Status:    "active",
// 		}
// 		database.DB.Create(editorMember)

// 		resp, err := testutils.MakeRequest(app, "DELETE", fmt.Sprintf("/projects/%d", project2.ID), nil, editorToken)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 403, resp.Code)
// 	})
// }

// func TestAddProjectMemberHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	// Create another user to add as member
// 	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")

// 	t.Run("Success - Add project member", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"user_id": editor.ID,
// 			"role":    models.ProjectRoleEditor,
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", fmt.Sprintf("/projects/%d/members", project.ID), body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 201, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		// Verify member was added
// 		var projectMember models.ProjectMember
// 		err = database.DB.Where("project_id = ? AND user_id = ?", project.ID, editor.ID).First(&projectMember).Error
// 		assert.NoError(t, err)
// 		assert.Equal(t, models.ProjectRoleEditor, projectMember.Role)
// 	})

// 	t.Run("Error - User already a member", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"user_id": editor.ID,
// 			"role":    models.ProjectRoleViewer,
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", fmt.Sprintf("/projects/%d/members", project.ID), body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 500, resp.Code) // Should return error
// 	})
// }

// func TestListProjectMembersHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	// Add multiple members
// 	member1 := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member1)

// 	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
// 	member2 := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    editor.ID,
// 		Role:      models.ProjectRoleEditor,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member2)

// 	t.Run("Success - List project members", func(t *testing.T) {
// 		resp, err := testutils.MakeRequest(app, "GET", fmt.Sprintf("/projects/%d/members", project.ID), nil, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		data := result.Data.([]interface{})
// 		assert.GreaterOrEqual(t, len(data), 2)
// 	})
// }

// func TestUpdateProjectMemberRoleHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	// Create another user and add as member
// 	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
// 	editorMember := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    editor.ID,
// 		Role:      models.ProjectRoleEditor,
// 		Status:    "active",
// 	}
// 	database.DB.Create(editorMember)

// 	t.Run("Success - Update member role", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"role": models.ProjectRoleAdmin,
// 		}

// 		resp, err := testutils.MakeRequest(app, "PUT", fmt.Sprintf("/projects/%d/members/%d", project.ID, editorMember.ID), body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		// Verify role was updated
// 		var updatedMember models.ProjectMember
// 		database.DB.First(&updatedMember, editorMember.ID)
// 		assert.Equal(t, models.ProjectRoleAdmin, updatedMember.Role)
// 	})
// }

// func TestRemoveProjectMemberHandler(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	// Create another user and add as member
// 	editor := testutils.CreateTestUser(t, database.DB, "editor@test.com", "password", "editor")
// 	editorMember := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    editor.ID,
// 		Role:      models.ProjectRoleEditor,
// 		Status:    "active",
// 	}
// 	database.DB.Create(editorMember)

// 	t.Run("Success - Remove project member", func(t *testing.T) {
// 		resp, err := testutils.MakeRequest(app, "DELETE", fmt.Sprintf("/projects/%d/members/%d", project.ID, editorMember.ID), nil, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 200, resp.Code)

// 		// Verify member was removed
// 		var deletedMember models.ProjectMember
// 		err = database.DB.First(&deletedMember, editorMember.ID).Error
// 		assert.Error(t, err) // Should not be found
// 	})
// }

// func TestProjectContentTypeIntegration(t *testing.T) {
// 	app := testutils.SetupTestApp(t)

// 	admin := testutils.CreateTestUser(t, database.DB, "admin@test.com", "password", "admin")
// 	token := testutils.GetAuthToken(t, admin.ID, admin.Role.Name)

// 	// Create project
// 	project := &models.Project{
// 		Name:      "Test Project",
// 		CreatedBy: admin.ID,
// 	}
// 	database.DB.Create(project)

// 	member := &models.ProjectMember{
// 		ProjectID: project.ID,
// 		UserID:    admin.ID,
// 		Role:      models.ProjectRoleOwner,
// 		Status:    "active",
// 	}
// 	database.DB.Create(member)

// 	t.Run("Success - Create content type for project", func(t *testing.T) {
// 		body := map[string]interface{}{
// 			"name":       "Project Blog",
// 			"slug":       "project-blog",
// 			"project_id": project.ID,
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", "/content/types", body, token)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 201, resp.Code)

// 		var result testutils.StandardResponse
// 		testutils.ParseResponse(t, resp, &result)
// 		assert.True(t, result.Success)

// 		data := result.Data.(map[string]interface{})
// 		assert.Equal(t, "Project Blog", data["name"])
// 		assert.NotNil(t, data["project_id"])

// 		// Verify content type was created with project_id
// 		var ct models.ContentType
// 		database.DB.Where("slug = ? AND project_id = ?", "project-blog", project.ID).First(&ct)
// 		assert.Equal(t, project.ID, *ct.ProjectID)
// 	})

// 	t.Run("Error - Non-member cannot create content type", func(t *testing.T) {
// 		nonMember := testutils.CreateTestUser(t, database.DB, "nonmember@test.com", "password", "editor")
// 		nonMemberToken := testutils.GetAuthToken(t, nonMember.ID, nonMember.Role.Name)

// 		body := map[string]interface{}{
// 			"name":       "Project Blog 2",
// 			"slug":       "project-blog-2",
// 			"project_id": project.ID,
// 		}

// 		resp, err := testutils.MakeRequest(app, "POST", "/content/types", body, nonMemberToken)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 403, resp.Code)
// 	})
// }
