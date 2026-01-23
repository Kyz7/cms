package project

import (
	"errors"
	"strings"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CreateProjectRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type AddMemberRequest struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role" validate:"required"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required"`
}

func CreateProjectHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var req CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", nil)
	}

	if req.Name == "" {
		return response.ValidationError(c, map[string]string{
			"name": "name is required",
		})
	}

	project, err := CreateProject(req.Name, req.Description, userID)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Created(c, project, "Project created successfully")
}

func GetProjectHandler(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	// Check if user is a member
	if _, err := GetProjectMember(uint(projectID), userID); err != nil {
		return response.Forbidden(c, "You are not a member of this project")
	}

	project, err := GetProject(uint(projectID))
	if err != nil {
		return response.NotFound(c, "Project")
	}

	return response.Success(c, project, "Project retrieved successfully")
}

func ListProjectsHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	projects, err := ListProjects(userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch projects")
	}

	return response.Success(c, projects, "Projects retrieved successfully")
}

func UpdateProjectHandler(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	// Check if user has permission (owner or admin)
	if !HasProjectPermission(uint(projectID), userID, "admin") {
		return response.Forbidden(c, "Only project owners and admins can update project")
	}

	var req UpdateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", nil)
	}

	project, err := UpdateProject(uint(projectID), req.Name, req.Description)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, project, "Project updated successfully")
}

func DeleteProjectHandler(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	// Check if user is owner
	if !HasProjectPermission(uint(projectID), userID, "owner") {
		return response.Forbidden(c, "Only project owner can delete project")
	}

	if err := DeleteProject(uint(projectID)); err != nil {
		return response.InternalError(c, "Failed to delete project")
	}

	return response.Success(c, nil, "Project deleted successfully")
}

func AddProjectMemberHandler(c *fiber.Ctx) error {

	// 1. Validasi ProjectID dari URL
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}
	projectID := uint(id)

	// 2. Ambil UserID dari Locals (Authenticated User)
	// Pastikan user_id di-set oleh middleware otentikasi
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		// Ini adalah error Internal Server jika middleware gagal
		return response.InternalError(c, "User context not available")
	}

	// 3. Otorisasi (Check if user has permission to add members)
	// Menggunakan konstanta ProjectRoleAdmin. Owner secara otomatis diizinkan
	// karena memiliki level yang lebih tinggi dari Admin di hierarki peran.
	if !HasProjectPermission(projectID, userID, models.ProjectRoleAdmin) {
		return response.Forbidden(c, "Only project owners and admins can add members")
	}

	// 4. Parsing Request Body
	var req AddMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body format", nil)
	}

	// 5. Validasi Input Data Tambahan
	if (req.UserID == 0 && req.Email == "") || req.Role == "" {
		return response.BadRequest(c, "UserID/Email and Role are required", nil)
	}

	targetUserID := req.UserID
	if targetUserID == 0 {
		// Lookup user by email
		var user models.User
		if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.NotFound(c, "User with this email not found")
			}
			return response.InternalError(c, "Database error looking up user")
		}
		targetUserID = user.ID
	}

	// Pastikan user tidak mencoba menambahkan dirinya sendiri (walaupun logika AddProjectMember akan menghandle, ini pencegahan di layer handler)
	if targetUserID == userID {
		return response.BadRequest(c, "Cannot add self as a member via this endpoint", nil)
	}

	// 6. Panggil Logika Bisnis
	// userID di sini berfungsi sebagai 'invitedBy'
	member, err := AddProjectMember(projectID, targetUserID, userID, req.Role)
	if err != nil {
		// Menggunakan switch/case untuk membedakan error yang dapat ditampilkan ke user
		switch {
		case strings.Contains(err.Error(), "project not found"):
			return response.NotFound(c, err.Error())
		case strings.Contains(err.Error(), "invalid or unconfigured role"):
			return response.BadRequest(c, err.Error(), "Error in role configuration")
		case strings.Contains(err.Error(), "already a member"):
			return response.Conflict(c, err.Error())
		default:
			// Error DB atau Internal lainnya
			return response.InternalError(c, "Failed to add member: "+err.Error())
		}
	}

	// 7. Respon Berhasil
	return response.Created(c, member, "Member added successfully")
}

func ListProjectMembersHandler(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	// Check if user is a member
	if _, err := GetProjectMember(uint(projectID), userID); err != nil {
		return response.Forbidden(c, "You are not a member of this project")
	}

	members, err := ListProjectMembers(uint(projectID))
	if err != nil {
		return response.InternalError(c, "Failed to fetch members")
	}

	return response.Success(c, members, "Members retrieved successfully")
}

func UpdateProjectMemberRoleHandler(c *fiber.Ctx) error {
	// Ambil ProjectID dari URL
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	// Ambil MemberID dari URL
	memberID, err := c.ParamsInt("member_id")
	if err != nil {
		return response.BadRequest(c, "Invalid member ID", nil)
	}

	// Ambil UserID dari Locals (setelah melewati middleware autentikasi)
	userID := c.Locals("user_id").(uint)

	// Parse Request Body
	var req UpdateMemberRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", nil)
	}

	// Validasi nama role tidak kosong
	if req.Role == "" {
		return response.BadRequest(c, "Role name cannot be empty", nil)
	}

	// Panggil Service Layer
	member, err := UpdateProjectMemberRole(uint(projectID), uint(memberID), userID, req.Role)

	if err != nil {
		// Karena fungsi layanan mengembalikan error spesifik (e.g., "only project owner can..."),
		// kita bisa menangkapnya dan mengembalikannya sebagai Forbidden atau Internal Error.
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, member, "Member role updated successfully")
}

func RemoveProjectMemberHandler(c *fiber.Ctx) error {
	projectID, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid project ID", nil)
	}

	memberID, err := c.ParamsInt("member_id")
	if err != nil {
		return response.BadRequest(c, "Invalid member ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	if err := RemoveProjectMember(uint(projectID), uint(memberID), userID); err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, nil, "Member removed successfully")
}
