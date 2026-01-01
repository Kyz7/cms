package project

import (
	"errors"
	"fmt"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/globals"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"gorm.io/gorm"
)

type NewRoleMember struct {
	RoleName string
}

func CreateProject(name, description string, createdBy uint) (*models.Project, error) {
	ownerID, ok := globals.GetRoleIDByName(models.ProjectRoleOwner)
	if !ok {

		return nil, response.InternalError(nil, "Project owner role not configured")
	}

	project := models.Project{
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&project).Error; err != nil {
			return err
		}

		member := models.ProjectMember{
			ProjectID: project.ID,
			UserID:    createdBy,
			RoleID:    ownerID,
			Status:    "active",
			InvitedBy: createdBy,
		}

		if err := tx.Create(&member).Error; err != nil {
			return fmt.Errorf("failed to add creator as project owner: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	database.DB.Preload("Creator").Preload("Members.User").First(&project, project.ID)

	return &project, nil
}

func GetProject(projectID uint) (*models.Project, error) {
	var project models.Project
	if err := database.DB.
		Preload("Creator").
		Preload("Members.User").
		Preload("ContentTypes").
		First(&project, projectID).Error; err != nil {
		return nil, fmt.Errorf("project not found")
	}
	return &project, nil
}

func ListProjects(userID uint) ([]models.Project, error) {
	var projects []models.Project

	// Get projects where user is a member
	err := database.DB.
		Joins("JOIN project_members ON projects.id = project_members.project_id").
		Where("project_members.user_id = ? AND project_members.deleted_at IS NULL", userID).
		Preload("Creator").
		Preload("Members.User").
		Group("projects.id").
		Find(&projects).Error

	return projects, err
}

func UpdateProject(projectID uint, name, description string) (*models.Project, error) {
	var project models.Project
	if err := database.DB.First(&project, projectID).Error; err != nil {
		return nil, fmt.Errorf("project not found")
	}

	project.Name = name
	if description != "" {
		project.Description = description
	}

	if err := database.DB.Save(&project).Error; err != nil {
		return nil, err
	}

	database.DB.Preload("Creator").Preload("Members.User").First(&project, project.ID)
	return &project, nil
}

func DeleteProject(projectID uint) error {
	return database.DB.Delete(&models.Project{}, projectID).Error
}

func AddProjectMember(projectID, userID, invitedBy uint, roleName string) (*models.ProjectMember, error) {

	// 1. Validasi Peran & Dapatkan RoleID
	// Kita mengandalkan GetRoleIDByName untuk memvalidasi peran.
	// Jika peranName ada di cache, berarti peran itu valid (ada di DB).
	roleID, ok := globals.GetRoleIDByName(roleName)
	if !ok {
		// Jika peran tidak ditemukan di cache (berarti tidak ada di DB), anggap itu invalid.
		return nil, fmt.Errorf("invalid or unconfigured role: %s", roleName)
	}

	// 2. Periksa Keberadaan Proyek
	var project models.Project
	if err := database.DB.First(&project, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("database error checking project existence: %w", err)
	}

	// 3. Periksa Keanggotaan yang Sudah Ada
	var existingMember models.ProjectMember
	err := database.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&existingMember).Error

	if err == nil {
		// Anggota sudah ditemukan (err == nil)
		return nil, fmt.Errorf("user (ID %d) is already a member of this project", userID)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Error DB lain (bukan karena RecordNotFound)
		return nil, fmt.Errorf("database error during membership check: %w", err)
	}

	// 4. Buat Anggota Proyek Baru

	member := models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		RoleID:    roleID, // Menggunakan RoleID yang didapat dari lookup dinamis
		Status:    "active",
		InvitedBy: invitedBy,
	}

	if err := database.DB.Create(&member).Error; err != nil {
		return nil, fmt.Errorf("failed to create project member: %w", err)
	}

	// 5. Muat (Preload) relasi sebelum mengembalikan hasil

	// Preload semua relasi yang diperlukan (User, Project, Inviter, dan Role)
	database.DB.
		Preload("User").
		Preload("Project").
		Preload("Inviter").
		Preload("Role").
		First(&member, member.ID)

	return &member, nil
}

func UpdateProjectMemberRole(projectID, memberID, userID uint, newRoleName string) (*models.ProjectMember, error) {

	// 1. Validasi Peran Baru & Dapatkan RoleID
	newRoleID, ok := globals.GetRoleIDByName(newRoleName)
	if !ok {
		return nil, fmt.Errorf("invalid or unconfigured role: %s", newRoleName)
	}

	// 2. Dapatkan Role ID untuk Otorisasi dari Cache
	ownerID, ok := globals.GetRoleIDByName(models.ProjectRoleOwner)
	if !ok {
		return nil, fmt.Errorf("project owner role not found in system configuration")
	}

	adminID, ok := globals.GetRoleIDByName(models.ProjectRoleAdmin)
	if !ok {
		return nil, fmt.Errorf("project admin role not found in system configuration")
	}

	// 3. Cek Anggota Permintaan (Requester) & Otorisasi
	var requesterMember models.ProjectMember
	err := database.DB.Preload("Role").Where("project_id = ? AND user_id = ?", projectID, userID).First(&requesterMember).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("you are not a member of this project or project not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error checking requester membership: %w", err)
	}

	// Otorisasi: Hanya owner dan admin yang dapat mengubah peran
	if requesterMember.RoleID != ownerID && requesterMember.RoleID != adminID {
		return nil, fmt.Errorf("only project owners and admins can change member roles")
	}

	// 4. Dapatkan Anggota yang Diperbarui
	var member models.ProjectMember
	err = database.DB.Preload("Role").Where("id = ? AND project_id = ?", memberID, projectID).First(&member).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("member not found in this project")
	}
	if err != nil {
		return nil, fmt.Errorf("database error retrieving member: %w", err)
	}

	// --- 5. Verifikasi Aturan Perubahan Peran Owner ---

	// Aturan 5.a: Peran Owner tidak dapat diubah oleh Admin
	if member.RoleID == ownerID && requesterMember.RoleID != ownerID {
		return nil, fmt.Errorf("only project owner can change the role of another owner")
	}

	// Aturan 5.b: Owner tidak bisa mengubah perannya sendiri menjadi peran di bawahnya
	if member.UserID == requesterMember.UserID && member.RoleID == ownerID && newRoleID != ownerID {
		return nil, fmt.Errorf("project owner must transfer ownership before changing their own role")
	}

	// 6. Cek Perubahan Role
	oldRoleID := member.RoleID
	if oldRoleID == newRoleID {
		// Tidak ada perubahan peran, langsung kembalikan data anggota yang dimuat (member)
		// Muat ulang dengan Preload yang benar untuk response, meskipun RoleID tidak berubah
		database.DB.
			Preload("User").
			Preload("Project").
			Preload("Inviter").
			Preload("Role").
			First(&member, member.ID)
		return &member, nil
	}

	// --- PERBAIKAN KRITIS: Ganti Save() dengan Updates() ---
	// 7. Simpan Perubahan ke Database menggunakan Updates eksplisit
	if err := database.DB.Model(&models.ProjectMember{}).
		Where("id = ? AND project_id = ?", member.ID, member.ProjectID).
		Update("role_id", newRoleID).Error; err != nil {

		return nil, fmt.Errorf("failed to update member role: %w", err)
	}

	// Perbarui struct member di memori agar konsisten sebelum Preload
	member.RoleID = newRoleID

	// 8. Muat (Preload) relasi setelah perubahan
	// Gunakan First(&member, member.ID) setelah update
	// untuk memastikan semua relasi dimuat ulang berdasarkan RoleID baru (13)
	database.DB.
		Preload("User").
		Preload("Project").
		Preload("Inviter").
		Preload("Role").
		First(&member, member.ID)

	return &member, nil
}

func RemoveProjectMember(projectID, memberID, userID uint) error {

	// 1. Dapatkan Role ID untuk Otorisasi dari Cache
	ownerID, ok := globals.GetRoleIDByName(models.ProjectRoleOwner)
	if !ok {
		return fmt.Errorf("project owner role not found in system configuration")
	}

	adminID, ok := globals.GetRoleIDByName(models.ProjectRoleAdmin)
	if !ok {
		// Jika admin role tidak ada, hanya owner yang bisa menghapus
		// Namun, kita tetap lanjut dan hanya mengizinkan owner.
	}

	// 2. Cek Anggota Permintaan (Requester) & Otorisasi
	var requesterMember models.ProjectMember
	err := database.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&requesterMember).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("you are not a member of this project or project not found")
	}
	if err != nil {
		return fmt.Errorf("database error checking requester membership: %w", err)
	}

	// Otorisasi: Hanya Owner ATAU Admin yang dapat menghapus (Logika De Morgan)
	// Cek: RoleID BUKAN Owner AND BUKAN Admin
	if requesterMember.RoleID != ownerID && requesterMember.RoleID != adminID {
		return fmt.Errorf("only project owners and admins can remove members")
	}

	// 3. Dapatkan Anggota yang Akan Dihapus
	var member models.ProjectMember
	err = database.DB.Where("id = ? AND project_id = ?", memberID, projectID).First(&member).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("member not found in this project")
	}
	if err != nil {
		return fmt.Errorf("database error retrieving member to remove: %w", err)
	}

	// 4. Verifikasi Aturan Khusus Owner (Menggunakan RoleID)

	// Aturan A: Mencegah Owner menghapus dirinya sendiri
	// Cek: Apakah pengguna yang dihapus adalah pengguna yang meminta penghapusan, DAN dia adalah Owner?
	if member.UserID == userID && member.RoleID == ownerID { // <-- PERBAIKAN: Menggunakan RoleID dan OwnerID
		return fmt.Errorf("owner cannot remove themselves")
	}

	// Aturan B: Mencegah penghapusan Owner oleh Admin/Non-Owner
	// Cek: Apakah peran anggota yang dihapus adalah Owner, DAN peran Requester BUKAN Owner?
	if member.RoleID == ownerID && requesterMember.RoleID != ownerID { // <-- PERBAIKAN: Menggunakan RoleID dan OwnerID
		return fmt.Errorf("only another project owner can remove an owner")
	}

	// 5. Cek Pencegahan Penghapusan Owner Terakhir (PENTING)
	// Walaupun owner tidak bisa menghapus dirinya sendiri, kita harus pastikan ada owner lain yang tersisa.
	if member.RoleID == ownerID {
		var ownerCount int64
		// Hitung berapa banyak ProjectMember lain (selain yang akan dihapus) yang berperan sebagai Owner
		if err := database.DB.Model(&models.ProjectMember{}).
			Where("project_id = ? AND role_id = ? AND id != ?", projectID, ownerID, memberID).
			Count(&ownerCount).Error; err != nil {
			return fmt.Errorf("failed to count remaining owners: %w", err)
		}

		if ownerCount == 0 {
			return fmt.Errorf("cannot remove the last project owner. Please assign a new owner first")
		}
	}

	// 6. Lakukan Penghapusan
	// Catatan: Jika Anda menggunakan Soft Delete (DeletedAt di struct), GORM akan melakukan soft delete.
	return database.DB.Delete(&member).Error
}

func GetProjectMember(projectID, userID uint) (*models.ProjectMember, error) {
	var member models.ProjectMember
	if err := database.DB.
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Preload("User").
		Preload("Project").
		First(&member).Error; err != nil {
		return nil, fmt.Errorf("user is not a member of this project")
	}
	return &member, nil
}

func ListProjectMembers(projectID uint) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := database.DB.
		Where("project_id = ?", projectID).
		Preload("User").
		Preload("Role").
		Preload("Inviter").
		Order("created_at DESC").
		Find(&members).Error
	return members, err
}

func HasProjectPermission(projectID, userID uint, requiredRoleName string) bool {

	// 1. Ambil Anggota Proyek dan Muat Peran (Role)
	var member models.ProjectMember
	// Kita harus Preload("Role") agar member.Role.Name dapat diakses
	err := database.DB.
		Preload("Role").
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, "active").
		First(&member).Error

	if err != nil {
		// Jika anggota tidak ditemukan atau error DB
		return false
	}

	// Pastikan Role dimuat dan tidak nil
	if member.Role == nil {
		return false
	}

	// 2. Definisikan Hierarki (menggunakan Nama Peran)
	roleHierarchy := map[string]int{
		models.ProjectRoleViewer: 1,
		models.ProjectRoleEditor: 2,
		models.ProjectRoleAdmin:  3,
		models.ProjectRoleOwner:  4,
	}

	// 3. Ambil Level Anggota (menggunakan member.Role.Name)
	memberLevel, memberOk := roleHierarchy[member.Role.Name] // <-- PERBAIKAN: Menggunakan member.Role.Name
	requiredLevel, requiredOk := roleHierarchy[requiredRoleName]

	if !memberOk || !requiredOk {
		// Jika salah satu peran tidak ada dalam hierarki yang ditentukan
		return false
	}

	// 4. Perbandingan Hierarki
	return memberLevel >= requiredLevel
}

func GetUserProjectRole(projectID, userID uint) (string, error) {

	// 1. Definisikan struct untuk menampung data
	var member models.ProjectMember

	// 2. Query ke Database
	// PENTING: Gunakan Preload("Role") agar data dari tabel Role ikut dimuat
	err := database.DB.
		Preload("Role").
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, "active").
		First(&member).Error

	// 3. Penanganan Error GORM
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Jika record tidak ditemukan, artinya user bukan anggota aktif
			return "", fmt.Errorf("user is not an active member of this project")
		}
		// Error database lainnya
		return "", fmt.Errorf("database error retrieving project member: %w", err)
	}

	// 4. Verifikasi dan Kembalikan Nama Peran

	// Periksa apakah relasi Role berhasil dimuat
	if member.Role == nil {
		// Ini bisa terjadi jika ProjectMember.RoleID tidak valid (ada di member, tapi tidak ada di tabel Role)
		return "", fmt.Errorf("project role not found for member (RoleID: %d)", member.RoleID)
	}

	// Kembalikan nama peran (string) dari struct Role yang dimuat
	return member.Role.Name, nil // <-- PERBAIKAN: Mengembalikan member.Role.Name
}
