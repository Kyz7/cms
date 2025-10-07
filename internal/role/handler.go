package role

import (
	"encoding/json"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func CreateRoleHandler(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Permissions []struct {
			Module         string   `json:"module"`
			Action         string   `json:"action"`
			FieldScope     string   `json:"field_scope,omitempty"`
			AllowedFields  []string `json:"allowed_fields,omitempty"`
			DeniedFields   []string `json:"denied_fields,omitempty"`
			ContentTypeIDs []uint   `json:"content_type_ids,omitempty"`
		} `json:"permissions"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" {
		return response.ValidationError(c, map[string]string{
			"name": "role name is required",
		})
	}

	var existing models.Role
	if err := database.DB.Where("name = ?", body.Name).First(&existing).Error; err == nil {
		return response.Conflict(c, "Role with this name already exists")
	}

	var role models.Role
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		role = models.Role{
			Name:        body.Name,
			Description: body.Description,
		}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		for _, p := range body.Permissions {
			perm := models.Permission{
				RoleID:     role.ID,
				Module:     p.Module,
				Action:     p.Action,
				FieldScope: p.FieldScope,
			}

			if len(p.AllowedFields) > 0 {
				allowedJSON, _ := json.Marshal(p.AllowedFields)
				perm.AllowedFields = allowedJSON
			}

			if len(p.DeniedFields) > 0 {
				deniedJSON, _ := json.Marshal(p.DeniedFields)
				perm.DeniedFields = deniedJSON
			}

			if len(p.ContentTypeIDs) > 0 {
				contentTypeJSON, _ := json.Marshal(p.ContentTypeIDs)
				perm.ContentTypeIDs = contentTypeJSON
			}

			if err := tx.Create(&perm).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return response.InternalError(c, "Failed to create role")
	}

	database.DB.Preload("Permissions").First(&role, role.ID)

	return response.Created(c, role, "Role created successfully")
}

func ListRolesHandler(c *fiber.Ctx) error {
	var roles []models.Role
	if err := database.DB.Preload("Permissions").Find(&roles).Error; err != nil {
		return response.InternalError(c, "Failed to fetch roles")
	}

	return response.Success(c, roles, "Roles retrieved successfully")
}

func GetRoleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid role ID", nil)
	}

	var role models.Role
	if err := database.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	return response.Success(c, role, "Role retrieved successfully")
}

func UpdateRoleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid role ID", nil)
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Permissions []struct {
			Module         string   `json:"module"`
			Action         string   `json:"action"`
			FieldScope     string   `json:"field_scope,omitempty"`
			AllowedFields  []string `json:"allowed_fields,omitempty"`
			DeniedFields   []string `json:"denied_fields,omitempty"`
			ContentTypeIDs []uint   `json:"content_type_ids,omitempty"`
		} `json:"permissions"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		role.Name = body.Name
		role.Description = body.Description
		if err := tx.Save(&role).Error; err != nil {
			return err
		}

		if err := tx.Where("role_id = ?", role.ID).Delete(&models.Permission{}).Error; err != nil {
			return err
		}

		for _, p := range body.Permissions {
			newPerm := models.Permission{
				RoleID:     role.ID,
				Module:     p.Module,
				Action:     p.Action,
				FieldScope: p.FieldScope,
			}

			if len(p.AllowedFields) > 0 {
				allowedJSON, _ := json.Marshal(p.AllowedFields)
				newPerm.AllowedFields = allowedJSON
			}

			if len(p.DeniedFields) > 0 {
				deniedJSON, _ := json.Marshal(p.DeniedFields)
				newPerm.DeniedFields = deniedJSON
			}

			if len(p.ContentTypeIDs) > 0 {
				contentTypeJSON, _ := json.Marshal(p.ContentTypeIDs)
				newPerm.ContentTypeIDs = contentTypeJSON
			}

			if err := tx.Create(&newPerm).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return response.InternalError(c, "Failed to update role")
	}

	database.DB.Preload("Permissions").First(&role, role.ID)

	return response.Success(c, role, "Role updated successfully")
}

func DeleteRoleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid role ID", nil)
	}

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	var userCount int64
	if err := database.DB.Model(&models.User{}).Where("role_id = ?", id).Count(&userCount).Error; err != nil {
		return response.InternalError(c, "Failed to check role usage")
	}

	if userCount > 0 {
		return response.Conflict(c, "Cannot delete role that is assigned to users")
	}

	if err := database.DB.Where("role_id = ?", id).Delete(&models.Permission{}).Error; err != nil {
		return response.InternalError(c, "Failed to delete role permissions")
	}

	if err := database.DB.Delete(&role).Error; err != nil {
		return response.InternalError(c, "Failed to delete role")
	}

	return response.NoContent(c)
}

func AssignRoleToUserHandler(c *fiber.Ctx) error {
	var body struct {
		UserID uint `json:"user_id"`
		RoleID uint `json:"role_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.UserID == 0 || body.RoleID == 0 {
		return response.ValidationError(c, map[string]string{
			"user_id": "user_id is required",
			"role_id": "role_id is required",
		})
	}

	var role models.Role
	if err := database.DB.First(&role, body.RoleID).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	var user models.User
	if err := database.DB.First(&user, body.UserID).Error; err != nil {
		return response.NotFound(c, "User")
	}

	user.RoleID = body.RoleID
	if err := database.DB.Save(&user).Error; err != nil {
		return response.InternalError(c, "Failed to assign role")
	}

	database.DB.Preload("Role.Permissions").First(&user, user.ID)

	return response.Success(c, user, "Role assigned successfully")
}

func DuplicateRoleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid role ID", nil)
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" {
		return response.ValidationError(c, map[string]string{
			"name": "new role name is required",
		})
	}

	var originalRole models.Role
	if err := database.DB.Preload("Permissions").First(&originalRole, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	var newRole models.Role
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		newRole = models.Role{
			Name:        body.Name,
			Description: originalRole.Description + " (Copy)",
		}
		if err := tx.Create(&newRole).Error; err != nil {
			return err
		}

		for _, perm := range originalRole.Permissions {
			newPerm := models.Permission{
				RoleID:     newRole.ID,
				Module:     perm.Module,
				Action:     perm.Action,
				FieldScope: perm.FieldScope,
			}
			if err := tx.Create(&newPerm).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return response.InternalError(c, "Failed to duplicate role")
	}

	database.DB.Preload("Permissions").First(&newRole, newRole.ID)

	return response.Created(c, newRole, "Role duplicated successfully")
}
