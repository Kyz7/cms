package role

import (
	"encoding/json"
	"errors"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PermissionBody merepresentasikan struktur izin dari body request
type PermissionBody struct {
	Module         string   `json:"module"`
	Action         string   `json:"action"`
	FieldScope     string   `json:"field_scope,omitempty"`
	AllowedFields  []string `json:"allowed_fields,omitempty"`
	DeniedFields   []string `json:"denied_fields,omitempty"`
	ContentTypeIDs []uint   `json:"content_type_ids,omitempty"`
}

// createUpdateRoleBody mendefinisikan struktur input untuk pembuatan/pembaruan Role.
type createUpdateRoleBody struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	IsGlobal    bool             `json:"is_global"` // ⭐️ PERUBAHAN KRITIS
	Permissions []PermissionBody `json:"permissions"`
}

// savePermissions menghapus izin lama dan membuat izin baru dalam transaksi.
func savePermissions(tx *gorm.DB, roleID uint, permissions []PermissionBody) error {
	// Hapus semua izin lama
	if err := tx.Where("role_id = ?", roleID).Delete(&models.Permission{}).Error; err != nil {
		return err
	}

	// Buat izin baru
	for _, p := range permissions {
		perm := models.Permission{
			RoleID:     roleID,
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
}

// CreateRoleHandler menangani pembuatan Role baru (Global atau Project Role).
func CreateRoleHandler(c *fiber.Ctx) error {
	var body createUpdateRoleBody

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" {
		return response.ValidationError(c, map[string]string{
			"name": "role name is required",
		})
	}

	// Pengecekan Duplikasi Nama Role
	var existing models.Role
	if err := database.DB.Where("name = ?", body.Name).First(&existing).Error; err == nil {
		return response.Conflict(c, "Role with this name already exists")
	}

	var role models.Role
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		role = models.Role{
			Name:        body.Name,
			Description: body.Description,
			IsGlobal:    body.IsGlobal, // ⭐️ Simpan nilai IsGlobal
		}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		// Simpan Permissions
		return savePermissions(tx, role.ID, body.Permissions)
	})

	if err != nil {
		return response.InternalError(c, "Failed to create role")
	}

	database.DB.Preload("Permissions").First(&role, role.ID)

	return response.Created(c, role, "Role created successfully")
}

// ListRolesHandler mengambil daftar semua Role.
func ListRolesHandler(c *fiber.Ctx) error {
	var roles []models.Role
	query := database.DB.Preload("Permissions")

	// Optional filter: is_global=true/false atau scope=global|project
	if v := c.Query("is_global"); v != "" {
		if v == "true" {
			query = query.Where("is_global = ?", true)
		} else if v == "false" {
			query = query.Where("is_global = ?", false)
		}
	} else if scope := c.Query("scope"); scope != "" {
		if scope == "global" {
			query = query.Where("is_global = ?", true)
		} else if scope == "project" {
			query = query.Where("is_global = ?", false)
		}
	}

	if err := query.Find(&roles).Error; err != nil {
		return response.InternalError(c, "Failed to fetch roles")
	}

	return response.Success(c, roles, "Roles retrieved successfully")
}

// NormalizeRoleScopesHandler: admin endpoint untuk memperbaiki flag is_global di database lama
func NormalizeRoleScopesHandler(c *fiber.Ctx) error {
	if err := NormalizeRoleScopes(database.DB); err != nil {
		return response.InternalError(c, "Failed to normalize role scopes")
	}
	return response.Success(c, fiber.Map{"normalized": true}, "Role scopes normalized")
}

// GetRoleHandler mengambil Role berdasarkan ID.
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

// UpdateRoleHandler menangani pembaruan detail Role dan izin.
func UpdateRoleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid role ID", nil)
	}

	var body createUpdateRoleBody
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	// Pengecekan Duplikasi Nama Role (jika nama diubah)
	if body.Name != "" && body.Name != role.Name {
		var existing models.Role
		if err := database.DB.Where("name = ? AND id != ?", body.Name, id).First(&existing).Error; err == nil {
			return response.Conflict(c, "Role with this name already exists")
		}
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Update detail Role
		role.Name = body.Name
		role.Description = body.Description
		role.IsGlobal = body.IsGlobal // ⭐️ Update nilai IsGlobal

		if err := tx.Save(&role).Error; err != nil {
			return err
		}

		// Hapus dan Buat ulang Permissions
		return savePermissions(tx, role.ID, body.Permissions)
	})

	if err != nil {
		return response.InternalError(c, "Failed to update role")
	}

	database.DB.Preload("Permissions").First(&role, role.ID)

	return response.Success(c, role, "Role updated successfully")
}

// DeleteRoleHandler menghapus Role jika tidak digunakan.
func DeleteRoleHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid role ID", nil)
	}

	var role models.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	// 1. Cek penggunaan oleh User (Role Global)
	var userCount int64
	if err := database.DB.Model(&models.User{}).Where("role_id = ?", id).Count(&userCount).Error; err != nil {
		return response.InternalError(c, "Failed to check user role usage")
	}
	if userCount > 0 {
		return response.Conflict(c, "Cannot delete role that is assigned to users")
	}

	// 2. Cek penggunaan oleh ProjectMember (Role Proyek) ⭐️ PERUBAHAN TAMBAHAN
	var memberCount int64
	if err := database.DB.Model(&models.ProjectMember{}).Where("role_id = ? AND status = ?", id, "active").Count(&memberCount).Error; err != nil {
		return response.InternalError(c, "Failed to check project member role usage")
	}
	if memberCount > 0 {
		return response.Conflict(c, "Cannot delete role that is assigned to active project members")
	}

	// Hapus dalam transaksi
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Hapus Permissions
		if err := tx.Where("role_id = ?", id).Delete(&models.Permission{}).Error; err != nil {
			return err
		}
		// Hapus Role
		if err := tx.Delete(&role).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return response.InternalError(c, "Failed to delete role")
	}

	return response.NoContent(c)
}

// AssignRoleToUserHandler menetapkan Role Global kepada pengguna.
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

	// 1. Validasi Role: Harus Role Global
	var role models.Role
	if err := database.DB.Where("is_global = ?", true).First(&role, body.RoleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "Role is not a Global Role or does not exist") // ⭐️ Penekanan pada Global Role
		}
		return response.InternalError(c, "Failed to find role")
	}

	// 2. Cek User
	var user models.User
	if err := database.DB.First(&user, body.UserID).Error; err != nil {
		return response.NotFound(c, "User")
	}

	// 3. Simpan
	user.RoleID = body.RoleID
	if err := database.DB.Save(&user).Error; err != nil {
		return response.InternalError(c, "Failed to assign role")
	}

	database.DB.Preload("Role.Permissions").First(&user, user.ID)
	user.Password = "" // Jangan kembalikan password

	return response.Success(c, user, "Role assigned successfully")
}

// DuplicateRoleHandler menangani duplikasi Role.
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

	// Pengecekan Duplikasi Nama Role Baru
	var existing models.Role
	if err := database.DB.Where("name = ?", body.Name).First(&existing).Error; err == nil {
		return response.Conflict(c, "New role name already exists")
	}

	var originalRole models.Role
	if err := database.DB.Preload("Permissions").First(&originalRole, id).Error; err != nil {
		return response.NotFound(c, "Role")
	}

	var newRole models.Role
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Duplikasi Role (termasuk IsGlobal)
		newRole = models.Role{
			Name:        body.Name,
			Description: originalRole.Description + " (Copy)",
			IsGlobal:    originalRole.IsGlobal, // ⭐️ Salin nilai IsGlobal
		}
		if err := tx.Create(&newRole).Error; err != nil {
			return err
		}

		// Duplikasi Permissions
		for _, perm := range originalRole.Permissions {
			// Menggunakan GORM Copy (untuk byte array seperti AllowedFields/DeniedFields/ContentTypeIDs)
			newPerm := perm
			newPerm.ID = 0 // Reset ID untuk insert baru
			newPerm.RoleID = newRole.ID

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
