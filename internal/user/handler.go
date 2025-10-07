package user

import (
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func CreateUserHandler(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		RoleID   uint   `json:"role_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.RoleID != 0 {
		var role models.Role
		if err := database.DB.First(&role, body.RoleID).Error; err != nil {
			return response.NotFound(c, "Role")
		}
	}
	if body.Email == "" || body.Password == "" || body.Name == "" {
		return response.ValidationError(c, map[string]string{
			"email":    "email is required",
			"password": "password is required",
			"name":     "name is required",
		})
	}

	var existing models.User
	if err := database.DB.Where("email = ?", body.Email).First(&existing).Error; err == nil {
		return response.Conflict(c, "User with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.InternalError(c, "Failed to hash password")
	}

	user := models.User{
		Email:    body.Email,
		Password: string(hashedPassword),
		Name:     body.Name,
		RoleID:   body.RoleID,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return response.InternalError(c, "Failed to create user")
	}

	database.DB.Preload("Role.Permissions").First(&user, user.ID)
	user.Password = ""

	return response.Created(c, user, "User created successfully")
}

func ListUsersHandler(c *fiber.Ctx) error {
	var users []models.User

	if err := database.DB.Preload("Role").Find(&users).Error; err != nil {
		return response.InternalError(c, "Failed to fetch users")
	}

	for i := range users {
		users[i].Password = ""
	}

	return response.Success(c, users, "Users retrieved successfully")
}

func GetUserHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid user ID", nil)
	}

	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, id).Error; err != nil {
		return response.NotFound(c, "User")
	}

	user.Password = ""

	return response.Success(c, user, "User retrieved successfully")
}

func UpdateUserHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid user ID", nil)
	}

	var body struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		RoleID uint   `json:"role_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return response.NotFound(c, "User")
	}

	if body.Email != "" && body.Email != user.Email {
		var existing models.User
		if err := database.DB.Where("email = ? AND id != ?", body.Email, id).First(&existing).Error; err == nil {
			return response.Conflict(c, "Email already taken")
		}
		user.Email = body.Email
	}

	if body.Name != "" {
		user.Name = body.Name
	}

	if body.RoleID != 0 {
		var role models.Role
		if err := database.DB.First(&role, body.RoleID).Error; err != nil {
			return response.NotFound(c, "Role")
		}
		user.RoleID = body.RoleID
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return response.InternalError(c, "Failed to update user")
	}

	database.DB.Preload("Role.Permissions").First(&user, user.ID)
	user.Password = ""

	return response.Success(c, user, "User updated successfully")
}

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

	if err := database.DB.Delete(&user).Error; err != nil {
		return response.InternalError(c, "Failed to delete user")
	}

	return response.NoContent(c)
}
