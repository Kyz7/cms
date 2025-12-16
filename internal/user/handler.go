package user

import (
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserHandler menangani pembuatan pengguna baru dan memastikan Role yang diberikan adalah Role Global.
func CreateUserHandler(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		RoleID   uint   `json:"role_id"` // Diasumsikan untuk Global Role
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	// 1. Validasi Wajib
	if body.Email == "" || body.Password == "" || body.Name == "" {
		return response.ValidationError(c, map[string]string{
			"email":    "email is required",
			"password": "password is required",
			"name":     "name is required",
		})
	}

	// 2. Validasi Role Global
	if body.RoleID != 0 {
		var role models.Role
		// Cari Role berdasarkan ID DAN pastikan is_global = TRUE
		if err := database.DB.Where("is_global = ?", true).First(&role, body.RoleID).Error; err != nil {
			// GORM akan mengembalikan record not found jika ID tidak ada ATAU jika is_global=false
			return response.NotFound(c, "Global Role not found or is a Project Role")
		}
	}

	// 3. Cek Konflik Email
	var existing models.User
	if err := database.DB.Where("email = ?", body.Email).First(&existing).Error; err == nil {
		return response.Conflict(c, "User with this email already exists")
	}

	// 4. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.InternalError(c, "Failed to hash password")
	}

	// 5. Buat Pengguna
	user := models.User{
		Email:    body.Email,
		Password: string(hashedPassword),
		Name:     body.Name,
		RoleID:   body.RoleID, // RoleID yang sudah divalidasi sebagai Global Role
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return response.InternalError(c, "Failed to create user")
	}

	// 6. Respon
	// Preload Role dan Permissions untuk dikembalikan (asumsi relasi Role di model User)
	database.DB.Preload("Role.Permissions").First(&user, user.ID)
	user.Password = ""

	return response.Created(c, user, "User created successfully")
}

// ListUsersHandler mengambil semua pengguna beserta Role Global mereka.
func ListUsersHandler(c *fiber.Ctx) error {
	var users []models.User

	// Preload Role (Global Role)
	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
		return response.InternalError(c, "Failed to fetch users")
	}

	for i := range users {
		users[i].Password = ""
	}

	return response.Success(c, users, "Users retrieved successfully")
}

// GetUserHandler mengambil satu pengguna berdasarkan ID.
func GetUserHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid user ID", nil)
	}

	var user models.User
	// Preload Role (Global Role) dan Permissions
	if err := database.DB.Preload("Role.Permissions").First(&user, id).Error; err != nil {
		return response.NotFound(c, "User")
	}

	user.Password = ""

	return response.Success(c, user, "User retrieved successfully")
}

// UpdateUserHandler menangani pembaruan pengguna dan memvalidasi RoleID sebagai Role Global.
func UpdateUserHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid user ID", nil)
	}

	var body struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		RoleID uint   `json:"role_id"` // Diasumsikan untuk Global Role
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return response.NotFound(c, "User")
	}

	// 1. Update Email
	if body.Email != "" && body.Email != user.Email {
		var existing models.User
		if err := database.DB.Where("email = ? AND id != ?", body.Email, id).First(&existing).Error; err == nil {
			return response.Conflict(c, "Email already taken")
		}
		user.Email = body.Email
	}

	// 2. Update Nama
	if body.Name != "" {
		user.Name = body.Name
	}

	// 3. Validasi dan Update Role Global
	if body.RoleID != 0 {
		var role models.Role
		// Cari Role berdasarkan ID DAN pastikan is_global = TRUE
		if err := database.DB.Where("is_global = ?", true).First(&role, body.RoleID).Error; err != nil {
			return response.NotFound(c, "Global Role not found or is a Project Role")
		}
		user.RoleID = body.RoleID
	}

	// 4. Simpan Perubahan
	if err := database.DB.Save(&user).Error; err != nil {
		return response.InternalError(c, "Failed to update user")
	}

	// 5. Respon
	database.DB.Preload("Role.Permissions").First(&user, user.ID)
	user.Password = ""

	return response.Success(c, user, "User updated successfully")
}

// DeleteUserHandler menghapus pengguna.
func DeleteUserHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid user ID", nil)
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return response.NotFound(c, "User")
	}

	currentUserID := c.Locals("user_id").(uint)
	if uint(id) == currentUserID {
		return response.BadRequest(c, "Cannot delete your own account", nil)
	}

	// Catatan: Jika user memiliki project_members, Anda mungkin perlu menghapus entri project_members
	// terlebih dahulu untuk menjaga integritas relasi.

	if err := database.DB.Delete(&user).Error; err != nil {
		return response.InternalError(c, "Failed to delete user")
	}

	return response.NoContent(c)
}
