package role

import (
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"gorm.io/gorm"
)

func CreateRole(db *gorm.DB, name string, description string, perms []models.Permission) (*models.Role, error) {
	role := models.Role{Name: name, Description: description, Permissions: perms}
	if err := db.Create(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func AssignPermissions(db *gorm.DB, roleID uint, perms []models.Permission) error {
	for _, p := range perms {
		p.RoleID = roleID
		if err := db.Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func ListRoles() ([]models.Role, error) {
	var roles []models.Role
	if err := database.DB.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
