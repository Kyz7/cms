package user

import (
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func CreateUser(db *gorm.DB, u *models.User) (*models.User, error) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	u.Password = string(hash)
	if err := db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func AssignRole(db *gorm.DB, userID uint, roleID uint) error {
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	u.RoleID = roleID
	return db.Save(&u).Error
}

func HasPermission(db *gorm.DB, userID uint, module string, action string) (bool, error) {
	var u models.User
	if err := db.Preload("Role.Permissions").First(&u, userID).Error; err != nil {
		return false, err
	}
	for _, perm := range u.Role.Permissions {
		if perm.Module == module && perm.Action == action {
			return true, nil
		}
	}
	return false, nil
}

func ListUsers() ([]models.User, error) {
	var users []models.User
	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
