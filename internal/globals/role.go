package globals

import (
	"log"

	"github.com/Kyz7/cms/internal/models"
	"gorm.io/gorm"
)

var RoleIDCache = make(map[string]uint)

func InitRoleCache(db *gorm.DB) error {
	var roles []models.Role
	// Ambil semua peran yang relevan
	if err := db.Find(&roles).Error; err != nil {
		return err
	}

	for _, role := range roles {
		RoleIDCache[role.Name] = role.ID
	}
	log.Println("Role cache initialized successfully.")
	return nil
}

// Fungsi untuk mendapatkan ID
func GetRoleIDByName(name string) (uint, bool) {
	id, ok := RoleIDCache[name]
	return id, ok
}
