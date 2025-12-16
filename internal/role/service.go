package role

import (
	"errors"
	"log"

	"github.com/Kyz7/cms/internal/models"
	"gorm.io/gorm"
)

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleAlreadyExists = errors.New("role already exists")
)

// =========================================================================
// OPERASI PERMISSION (Helper)
// =========================================================================

// SavePermissions menghapus semua izin yang ada untuk RoleID ini dan membuat izin baru.
// Fungsi ini harus dipanggil di dalam transaksi GORM.
func SavePermissions(tx *gorm.DB, roleID uint, perms []models.Permission) error {
	// 1. Hapus semua Permission yang ada untuk Role ini
	if err := tx.Where("role_id = ?", roleID).Delete(&models.Permission{}).Error; err != nil {
		return err
	}

	// 2. Buat Permission baru
	for i := range perms {
		// Pastikan RoleID terikat sebelum dibuat
		perms[i].RoleID = roleID
		perms[i].ID = 0 // Pastikan ID direset jika objek lama digunakan kembali

		if err := tx.Create(&perms[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

// =========================================================================
// OPERASI ROLE UTAMA
// =========================================================================

// CreateRole membuat Role baru. Jika Role sudah ada, fungsi ini akan mengembalikan ErrRoleAlreadyExists.
func CreateRole(db *gorm.DB, name string, description string, isGlobal bool, permissions []models.Permission) (*models.Role, error) {
	var role models.Role

	// 1. Cek duplikasi Role
	if err := db.Where("name = ?", name).First(&role).Error; err == nil {
		return nil, ErrRoleAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err // Error database lainnya
	}

	// 2. Buat Role dan Permissions dalam satu transaksi
	err := db.Transaction(func(tx *gorm.DB) error {
		role = models.Role{
			Name:        name,
			Description: description,
			IsGlobal:    isGlobal,
		}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		// Simpan Permissions
		if err := SavePermissions(tx, role.ID, permissions); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Printf("Role '%s' created successfully.", name)

	// Muat ulang Role dengan Permissions
	db.Preload("Permissions").First(&role, role.ID)
	return &role, nil
}

// GetRoleByID mengambil Role berdasarkan ID dan memuat Permissions-nya.
func GetRoleByID(db *gorm.DB, roleID uint) (*models.Role, error) {
	var role models.Role
	if err := db.Preload("Permissions").First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

// UpdateRole memperbarui detail Role dan mengganti Permissions-nya.
func UpdateRole(db *gorm.DB, roleID uint, name string, description string, isGlobal bool, permissions []models.Permission) (*models.Role, error) {
	var role models.Role
	if err := db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	// Cek duplikasi nama (kecuali untuk dirinya sendiri)
	var existing models.Role
	if err := db.Where("name = ? AND id != ?", name, roleID).First(&existing).Error; err == nil {
		return nil, ErrRoleAlreadyExists
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Update detail Role
		role.Name = name
		role.Description = description
		role.IsGlobal = isGlobal
		if err := tx.Save(&role).Error; err != nil {
			return err
		}

		// 2. Simpan Permissions baru (menghapus yang lama)
		return SavePermissions(tx, role.ID, permissions)
	})

	if err != nil {
		return nil, err
	}

	// Muat ulang Role dengan Permissions
	return GetRoleByID(db, roleID)
}

// DeleteRole menghapus Role dan Permissions terkait.
func DeleteRole(db *gorm.DB, roleID uint) error {
	var role models.Role
	if err := db.First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	// Cek Ketergantungan (Sudah dilakukan di Role Handler, tetapi lebih aman untuk mengulang di sini)

	// 1. Cek penggunaan oleh User (Role Global)
	var userCount int64
	if err := db.Model(&models.User{}).Where("role_id = ?", roleID).Count(&userCount).Error; err != nil {
		return errors.New("failed to check user role usage")
	}
	if userCount > 0 {
		return errors.New("cannot delete role assigned to users")
	}

	// 2. Cek penggunaan oleh ProjectMember (Role Proyek)
	var memberCount int64
	if err := db.Model(&models.ProjectMember{}).Where("role_id = ? AND status = ?", roleID, "active").Count(&memberCount).Error; err != nil {
		return errors.New("failed to check project member role usage")
	}
	if memberCount > 0 {
		return errors.New("cannot delete role assigned to active project members")
	}

	// Hapus dalam transaksi
	return db.Transaction(func(tx *gorm.DB) error {
		// Hapus Permissions
		if err := tx.Where("role_id = ?", roleID).Delete(&models.Permission{}).Error; err != nil {
			return err
		}
		// Hapus Role
		if err := tx.Delete(&role).Error; err != nil {
			return err
		}
		return nil
	})
}

// ListRoles mengambil semua Role (Global dan Project Role).
func ListRoles(db *gorm.DB) ([]models.Role, error) {
	var roles []models.Role
	// Menggunakan DB yang diterima sebagai parameter, bukan database.DB global (Best Practice)
	if err := db.Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ListGlobalRoles mengambil hanya Role yang bersifat Global.
func ListGlobalRoles(db *gorm.DB) ([]models.Role, error) {
	var roles []models.Role
	if err := db.Preload("Permissions").Where("is_global = ?", true).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ListProjectRoles mengambil hanya Role yang bersifat Project.
func ListProjectRoles(db *gorm.DB) ([]models.Role, error) {
	var roles []models.Role
	if err := db.Preload("Permissions").Where("is_global = ?", false).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
