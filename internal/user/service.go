package user

import (
	"errors"

	"github.com/Kyz7/cms/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrRoleNotFound = errors.New("role not found")
)

// CreateUser membuat pengguna baru, meng-hash password, dan menyimpannya ke DB.
func CreateUser(db *gorm.DB, u *models.User) (*models.User, error) {
	// Pengecekan Duplikasi Email (penting dilakukan di service layer)
	var existing models.User
	if err := db.Where("email = ?", u.Email).First(&existing).Error; err == nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash Password
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}
	u.Password = string(hash)

	// Validasi Role ID harus Global Role (sebelum disimpan)
	if u.RoleID != 0 {
		var role models.Role
		// Pastikan Role tersebut ada dan bersifat Global
		if err := db.Where("is_global = ?", true).First(&role, u.RoleID).Error; err != nil {
			return nil, ErrRoleNotFound
		}
	} else {
		// Asumsi RoleID tidak boleh 0 jika sistem otorisasi global diperlukan.
		// Anda bisa menambahkan logika default role di sini.
	}

	if err := db.Create(u).Error; err != nil {
		return nil, err
	}
	// Muat ulang Role dan Permissions untuk respons
	db.Preload("Role.Permissions").First(u, u.ID)
	u.Password = ""
	return u, nil
}

// GetUserByID mengambil satu pengguna berdasarkan ID dan memuat Role Globalnya.
func GetUserByID(db *gorm.DB, id uint) (*models.User, error) {
	var user models.User
	if err := db.Preload("Role.Permissions").First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	user.Password = ""
	return &user, nil
}

// UpdateUser memperbarui detail pengguna (Nama, Email, Role Global).
func UpdateUser(db *gorm.DB, userID uint, updates map[string]interface{}) (*models.User, error) {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Jika RoleID diupdate, validasi harus Global Role
	if newRoleID, ok := updates["RoleID"]; ok && newRoleID.(uint) != 0 {
		var role models.Role
		if err := db.Where("is_global = ?", true).First(&role, newRoleID).Error; err != nil {
			return nil, ErrRoleNotFound
		}
	}

	// Cek Duplikasi Email jika Email diupdate
	if newEmail, ok := updates["Email"]; ok && newEmail.(string) != user.Email {
		var existing models.User
		if err := db.Where("email = ? AND id != ?", newEmail, userID).First(&existing).Error; err == nil {
			return nil, errors.New("email already taken")
		}
	}

	if err := db.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Muat ulang data terbaru
	return GetUserByID(db, userID)
}

// DeleteUser menghapus pengguna secara soft delete.
func DeleteUser(db *gorm.DB, userID uint) error {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if err := db.Delete(&user).Error; err != nil {
		return err
	}
	return nil
}

// AssignRole menetapkan Role Global kepada pengguna (menggunakan kolom RoleID di tabel users).
func AssignRole(db *gorm.DB, userID uint, roleID uint) error {
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}

	// Validasi Role Global
	var role models.Role
	if err := db.Where("is_global = ?", true).First(&role, roleID).Error; err != nil {
		return ErrRoleNotFound
	}

	u.RoleID = roleID
	return db.Save(&u).Error
}

// HasPermission memeriksa apakah Role Global pengguna memiliki izin tertentu.
// Catatan: Fungsi ini TIDAK memeriksa Project Roles.
func HasPermission(db *gorm.DB, userID uint, module string, action string) (bool, error) {
	var u models.User
	if err := db.Preload("Role.Permissions").First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // Jika user tidak ditemukan, anggap tidak punya izin
		}
		return false, err
	}

	// Periksa Permissions di Global Role
	for _, perm := range u.Role.Permissions {
		if perm.Module == module && perm.Action == action {
			return true, nil
		}
	}
	return false, nil
}

// ListUsers mengambil semua pengguna dan memuat Role Global mereka.
func ListUsers(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	if err := db.Preload("Role").Find(&users).Error; err != nil {
		return nil, err
	}

	// Sembunyikan Password
	for i := range users {
		users[i].Password = ""
	}

	return users, nil
}
