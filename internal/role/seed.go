package role

import (
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"gorm.io/gorm"
)

// =========================================================================
// SEEDING ROLES
// =========================================================================

func SeedDefaultRoles() error {
	// Editor Role (Global)
	editorPerms := []models.Permission{
		{Module: "ContentEntry", Action: "create", FieldScope: "all"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "ContentEntry", Action: "update", FieldScope: "all"},
		{Module: "Media", Action: "create"},
		{Module: "Media", Action: "read"},
		{Module: "Media", Action: "update"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "editor", "Can create/edit content, upload media, and view SEO", true, editorPerms)

	// Manager Role (Global)
	managerPerms := []models.Permission{
		{Module: "ContentEntry", Action: "approve"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "Media", Action: "read"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "manager", "Can approve content", true, managerPerms)

	// Viewer Role (Global)
	viewerPerms := []models.Permission{
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "Media", Action: "read"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "viewer", "Can view content only", true, viewerPerms)

	// Admin Role (Global - Full access)
	adminPerms := []models.Permission{
		{Module: "ContentEntry", Action: "create", FieldScope: "all"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "ContentEntry", Action: "update", FieldScope: "all"},
		{Module: "ContentEntry", Action: "delete"},
		{Module: "ContentEntry", Action: "approve"},

		// Tambahkan Izin Skema Global (ContentType) untuk Admin Global
		{Module: "ContentType", Action: "create"},
		{Module: "ContentType", Action: "read"},
		{Module: "ContentType", Action: "update"},
		{Module: "ContentType", Action: "delete"},

		{Module: "Media", Action: "create"},
		{Module: "Media", Action: "read"},
		{Module: "Media", Action: "update"},
		{Module: "Media", Action: "delete"},
		{Module: "SEO", Action: "create", FieldScope: "all"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
		{Module: "SEO", Action: "update", FieldScope: "all"},
		{Module: "SEO", Action: "delete"},
	}
	_, _ = CreateRole(database.DB, "admin", "Full access to all resources", true, adminPerms)

	// SEO Specialist (Global)
	seoPerms := []models.Permission{
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "ContentEntry", Action: "update", FieldScope: "seo_only"},
		{Module: "Media", Action: "read"},
		{Module: "SEO", Action: "create", FieldScope: "all"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
		{Module: "SEO", Action: "update", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "seo_specialist", "Can edit SEO fields only", true, seoPerms)

	// ====================================================================
	// PROJECT ROLES (Menggunakan konstanta ProjectRoleViewer, dll. dari models)
	// ====================================================================

	// Project Viewer Role
	viewerProjectPerms := []models.Permission{
		{Module: "ProjectContent", Action: "read", FieldScope: "all"},
		{Module: "ProjectMedia", Action: "read"},
	}
	_, _ = CreateRole(database.DB, models.ProjectRoleViewer, "Can view all content and media within a project", false, viewerProjectPerms)

	// Project Editor Role
	editorProjectPerms := []models.Permission{
		{Module: "ProjectContent", Action: "create", FieldScope: "all"},
		{Module: "ProjectContent", Action: "read", FieldScope: "all"},
		{Module: "ProjectContent", Action: "update", FieldScope: "all"},
		{Module: "ProjectMedia", Action: "create"},
		{Module: "ProjectMedia", Action: "read"},
		{Module: "ProjectMedia", Action: "update"},

		// Tambahkan Izin SEO Project (untuk editor yang ingin preview)
		{Module: "SEO", Action: "read", FieldScope: "all"},

		// Tambahkan Izin Skema Proyek agar Editor bisa membuat/mengelola tipe konten di project
		{Module: "ProjectSchema", Action: "create"},
		{Module: "ProjectSchema", Action: "read"},
		{Module: "ProjectSchema", Action: "update"},
	}
	_, _ = CreateRole(database.DB, models.ProjectRoleEditor, "Can manage content and media within a project", false, editorProjectPerms)

	// Project Admin Role
	adminProjectPerms := []models.Permission{
		{Module: "ProjectContent", Action: "create", FieldScope: "all"},
		{Module: "ProjectContent", Action: "read", FieldScope: "all"},
		{Module: "ProjectContent", Action: "update", FieldScope: "all"},
		{Module: "ProjectContent", Action: "delete"},
		{Module: "ProjectMedia", Action: "create"},
		{Module: "ProjectMedia", Action: "read"},
		{Module: "ProjectMedia", Action: "update"},
		{Module: "ProjectMedia", Action: "delete"},
		{Module: "ProjectSettings", Action: "read"},
		{Module: "ProjectMembers", Action: "invite"},
		{Module: "ProjectMembers", Action: "read"},
		{Module: "ProjectMembers", Action: "update"},
		{Module: "ProjectMembers", Action: "remove"},
		{Module: "SEO", Action: "read", FieldScope: "all"},

		// Tambahkan Izin Skema Proyek agar Admin bisa membuat/mengelola tipe konten di project
		{Module: "ProjectSchema", Action: "create"},
		{Module: "ProjectSchema", Action: "read"},
		{Module: "ProjectSchema", Action: "update"},
		{Module: "ProjectSchema", Action: "delete"},
	}
	_, _ = CreateRole(database.DB, models.ProjectRoleAdmin, "Full content access and manage project members", false, adminProjectPerms)

	// Project Owner Role (KRITIS: Menambahkan izin Skema yang Hilang)
	ownerPerms := []models.Permission{
		// Izin Konten dan Media
		{Module: "ProjectContent", Action: "create", FieldScope: "all"},
		{Module: "ProjectContent", Action: "read", FieldScope: "all"},
		{Module: "ProjectContent", Action: "update", FieldScope: "all"},
		{Module: "ProjectContent", Action: "delete"},
		{Module: "ProjectMedia", Action: "create"},
		{Module: "ProjectMedia", Action: "read"},
		{Module: "ProjectMedia", Action: "update"},
		{Module: "ProjectMedia", Action: "delete"},
		{Module: "ProjectContent", Action: "approve"},
		{Module: "ProjectContent", Action: "publish"},

		// ⭐️ Izin Skema Proyek (Untuk mengatasi 403 Update Field)
		{Module: "ProjectSchema", Action: "create"},
		{Module: "ProjectSchema", Action: "read"},
		{Module: "ProjectSchema", Action: "update"},
		{Module: "ProjectSchema", Action: "delete"},

		// Izin SEO (Untuk mengatasi 403 SEO Preview)
		{Module: "SEO", Action: "read", FieldScope: "all"},

		// Izin Administrasi Penuh
		{Module: "ProjectSettings", Action: "read"},
		{Module: "ProjectSettings", Action: "update"},
		{Module: "ProjectSettings", Action: "delete"},
		{Module: "ProjectMembers", Action: "invite"},
		{Module: "ProjectMembers", Action: "read"},
		{Module: "ProjectMembers", Action: "update"},
		{Module: "ProjectMembers", Action: "remove"},
		{Module: "ProjectOwnership", Action: "transfer"},
	}
	_, _ = CreateRole(database.DB, models.ProjectRoleOwner, "Full administrative control, including project deletion and ownership transfer", false, ownerPerms)

	projectWriterPerms := []models.Permission{
		{Module: "ProjectContent", Action: "create", FieldScope: "non_seo_only"},
		{Module: "ProjectContent", Action: "read", FieldScope: "all"},
		{Module: "ProjectContent", Action: "update", FieldScope: "non_seo_only"},
		{Module: "ProjectMedia", Action: "create"},
		{Module: "ProjectMedia", Action: "read"},
		{Module: "ProjectSchema", Action: "create"},
		{Module: "ProjectSchema", Action: "read"},
		{Module: "ProjectSchema", Action: "update"},
	}
	_, _ = CreateRole(database.DB, models.ProjectRoleContentWriter, "Project-scoped content writer (non-SEO fields)", false, projectWriterPerms)

	// Content Writer (Global)
	writerPerms := []models.Permission{
		{Module: "ContentEntry", Action: "create", FieldScope: "non_seo_only"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "ContentEntry", Action: "update", FieldScope: "non_seo_only"},
		{Module: "Media", Action: "create"},
		{Module: "Media", Action: "read"},
	}
	_, _ = CreateRole(database.DB, "content_writer", "Can create/edit content (non-SEO fields)", true, writerPerms)

	return nil
}

// NormalizeRoleScopes memastikan flag is_global konsisten untuk role bawaan.
// Berguna untuk memperbaiki database lama yang belum memiliki penandaan is_global yang benar.
func NormalizeRoleScopes(db *gorm.DB) error {
	type pair struct {
		Name     string
		IsGlobal bool
	}
	roles := []pair{
		// Global roles
		{Name: "admin", IsGlobal: true},
		{Name: "editor", IsGlobal: true},
		{Name: "manager", IsGlobal: true},
		{Name: "viewer", IsGlobal: true},
		{Name: "content_writer", IsGlobal: true},
		{Name: "seo_specialist", IsGlobal: true},
		// Project roles
		{Name: models.ProjectRoleViewer, IsGlobal: false},
		{Name: models.ProjectRoleEditor, IsGlobal: false},
		{Name: models.ProjectRoleAdmin, IsGlobal: false},
		{Name: models.ProjectRoleOwner, IsGlobal: false},
		{Name: models.ProjectRoleContentWriter, IsGlobal: false},
	}

	for _, r := range roles {
		db.Model(&models.Role{}).
			Where("name = ?", r.Name).
			Update("is_global", r.IsGlobal)
	}
	return nil
}

// EnsureGlobalSchemaPermissions memastikan izin ContentType sesuai kebijakan:
// - editor (global): create, read, update
// - content_writer (global): read, update (tanpa create)
func EnsureGlobalSchemaPermissions(db *gorm.DB) error {
	// Definisikan aksi yang diizinkan per role
	allowedActions := map[string][]string{
		"editor":         {"create", "read", "update"},
		"content_writer": {"read", "update"},
	}

	for roleName, actions := range allowedActions {
		var role models.Role
		if err := db.Preload("Permissions").Where("name = ? AND is_global = ?", roleName, true).First(&role).Error; err != nil {
			// Jika role belum ada, lewati tanpa error (akan dibuat oleh seeding di awal)
			continue
		}

		exists := map[string]bool{"create": false, "read": false, "update": false}
		for _, p := range role.Permissions {
			if p.Module == "ContentType" && (p.Action == "create" || p.Action == "read" || p.Action == "update") {
				exists[p.Action] = true
			}
		}

		for _, act := range actions {
			if exists[act] {
				continue
			}
			perm := models.Permission{
				RoleID: role.ID,
				Module: "ContentType",
				Action: act,
			}
			if err := db.Create(&perm).Error; err != nil {
				return err
			}
		}

		// Jika role adalah content_writer dan memiliki izin create, hapus izin tersebut
		if roleName == "content_writer" && exists["create"] {
			if err := db.Where("role_id = ? AND module = ? AND action = ?", role.ID, "ContentType", "create").Delete(&models.Permission{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// EnsureEditorContentEntryUnrestricted memastikan role editor global
// dapat create/update entries di semua ContentType (ContentTypeIDs NULL, FieldScope 'all').
func EnsureEditorContentEntryUnrestricted(db *gorm.DB) error {
	var role models.Role
	if err := db.Preload("Permissions").Where("name = ? AND is_global = ?", "editor", true).First(&role).Error; err != nil {
		return nil // jika tidak ada, abaikan
	}

	for _, p := range role.Permissions {
		if p.Module == "ContentEntry" && (p.Action == "create" || p.Action == "update") {
			updates := map[string]interface{}{}
			// Set FieldScope ke 'all' jika kosong
			if p.FieldScope == "" {
				updates["field_scope"] = "all"
			}
			// Hapus pembatasan ContentTypeIDs jika ada
			if p.ContentTypeIDs != nil {
				updates["content_type_ids"] = nil
			}
			if len(updates) > 0 {
				if err := db.Model(&models.Permission{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func EnsureProjectSchemaPermissions(db *gorm.DB) error {
	type ensureSpec struct {
		RoleName string
		Actions  []string
	}
	specs := []ensureSpec{
		{RoleName: models.ProjectRoleEditor, Actions: []string{"create", "read", "update"}},
		{RoleName: models.ProjectRoleAdmin, Actions: []string{"create", "read", "update", "delete"}},
		{RoleName: models.ProjectRoleOwner, Actions: []string{"create", "read", "update", "delete"}},
		{RoleName: models.ProjectRoleContentWriter, Actions: []string{"create", "read", "update"}},
	}
	for _, s := range specs {
		var role models.Role
		if err := db.Preload("Permissions").Where("name = ? AND is_global = ?", s.RoleName, false).First(&role).Error; err != nil {
			continue
		}
		exists := map[string]bool{"create": false, "read": false, "update": false, "delete": false}
		for _, p := range role.Permissions {
			if p.Module == "ProjectSchema" && (p.Action == "create" || p.Action == "read" || p.Action == "update" || p.Action == "delete") {
				exists[p.Action] = true
			}
		}
		for _, act := range s.Actions {
			if exists[act] {
				continue
			}
			perm := models.Permission{
				RoleID: role.ID,
				Module: "ProjectSchema",
				Action: act,
			}
			if err := db.Create(&perm).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// =========================================================================
// SEEDING WORKFLOW
// =========================================================================

func SeedWorkflowTransitions(db *gorm.DB) error {
	transitions := []models.WorkflowTransition{
		// From Draft
		{FromStatus: "draft", ToStatus: "in_review", RequiredRole: "editor"},
		{FromStatus: "draft", ToStatus: "in_review", RequiredRole: "admin"},
		{FromStatus: "draft", ToStatus: "in_review", RequiredRole: models.ProjectRoleEditor},
		{FromStatus: "draft", ToStatus: "in_review", RequiredRole: models.ProjectRoleAdmin},

		// From In Review
		{FromStatus: "in_review", ToStatus: "ready_for_approval", RequiredRole: "editor"},
		{FromStatus: "in_review", ToStatus: "ready_for_approval", RequiredRole: "admin"},
		{FromStatus: "in_review", ToStatus: "rejected", RequiredRole: "editor"},
		{FromStatus: "in_review", ToStatus: "rejected", RequiredRole: "admin"},
		{FromStatus: "in_review", ToStatus: "draft", RequiredRole: "editor"},
		{FromStatus: "in_review", ToStatus: "draft", RequiredRole: "admin"},

		// From Ready for Approval (Hanya Manager/Admin Global yang Approve/Reject)
		{FromStatus: "ready_for_approval", ToStatus: "approved", RequiredRole: "manager"},
		{FromStatus: "ready_for_approval", ToStatus: "approved", RequiredRole: "admin"},
		{FromStatus: "ready_for_approval", ToStatus: "rejected", RequiredRole: "manager"},
		{FromStatus: "ready_for_approval", ToStatus: "rejected", RequiredRole: "admin"},
		{FromStatus: "ready_for_approval", ToStatus: "approved", RequiredRole: models.ProjectRoleOwner},
		{FromStatus: "ready_for_approval", ToStatus: "approved", RequiredRole: models.ProjectRoleAdmin},
		{FromStatus: "ready_for_approval", ToStatus: "rejected", RequiredRole: models.ProjectRoleOwner},
		{FromStatus: "ready_for_approval", ToStatus: "rejected", RequiredRole: models.ProjectRoleAdmin},

		// From Approved
		{FromStatus: "approved", ToStatus: "published", RequiredRole: "manager"},
		{FromStatus: "approved", ToStatus: "published", RequiredRole: "admin"},
		{FromStatus: "approved", ToStatus: "published", RequiredRole: models.ProjectRoleOwner},
		{FromStatus: "approved", ToStatus: "published", RequiredRole: models.ProjectRoleAdmin},
		// From Rejected
		{FromStatus: "rejected", ToStatus: "draft", RequiredRole: "editor"},
		{FromStatus: "rejected", ToStatus: "draft", RequiredRole: "admin"},
		{FromStatus: "rejected", ToStatus: "draft", RequiredRole: models.ProjectRoleEditor},
		{FromStatus: "rejected", ToStatus: "draft", RequiredRole: models.ProjectRoleAdmin},
	}

	for _, transition := range transitions {
		// Check if transition already exists
		var existing models.WorkflowTransition
		result := db.Where("from_status = ? AND to_status = ? AND required_role = ?",
			transition.FromStatus, transition.ToStatus, transition.RequiredRole).
			First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&transition).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
