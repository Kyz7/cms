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
