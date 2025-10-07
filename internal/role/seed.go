package role

import (
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"gorm.io/gorm"
)

func SeedDefaultRoles() error {
	// Editor Role - Bisa create/edit content, upload media, dan read SEO
	editorPerms := []models.Permission{
		{Module: "ContentEntry", Action: "create", FieldScope: "all"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "ContentEntry", Action: "update", FieldScope: "all"},
		{Module: "Media", Action: "create"},
		{Module: "Media", Action: "read"},
		{Module: "Media", Action: "update"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "editor", "Can create/edit content, upload media, and view SEO", editorPerms)

	// Manager Role - Bisa approve content
	managerPerms := []models.Permission{
		{Module: "ContentEntry", Action: "approve"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "Media", Action: "read"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "manager", "Can approve content", managerPerms)

	// Viewer Role - Read only
	viewerPerms := []models.Permission{
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "Media", Action: "read"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "viewer", "Can view content only", viewerPerms)

	// Admin Role - Full access
	adminPerms := []models.Permission{
		{Module: "ContentEntry", Action: "create", FieldScope: "all"},
		{Module: "ContentEntry", Action: "read", FieldScope: "all"},
		{Module: "ContentEntry", Action: "update", FieldScope: "all"},
		{Module: "ContentEntry", Action: "delete"},
		{Module: "ContentEntry", Action: "approve"},
		{Module: "Media", Action: "create"},
		{Module: "Media", Action: "read"},
		{Module: "Media", Action: "update"},
		{Module: "Media", Action: "delete"},
		{Module: "SEO", Action: "create", FieldScope: "all"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
		{Module: "SEO", Action: "update", FieldScope: "all"},
		{Module: "SEO", Action: "delete"},
	}
	_, _ = CreateRole(database.DB, "admin", "Full access to all resources", adminPerms)

	// SEO Specialist - Hanya bisa edit SEO fields (ENHANCED)
	seoPerms := []models.Permission{
		{
			Module:     "ContentEntry",
			Action:     "read",
			FieldScope: "all",
		},
		{
			Module:     "ContentEntry",
			Action:     "update",
			FieldScope: "seo_only",
		},
		{Module: "Media", Action: "read"},
		{Module: "SEO", Action: "create", FieldScope: "all"},
		{Module: "SEO", Action: "read", FieldScope: "all"},
		{Module: "SEO", Action: "update", FieldScope: "all"},
	}
	_, _ = CreateRole(database.DB, "seo_specialist", "Can edit SEO fields only", seoPerms)

	// Content Writer - Hanya bisa edit non-SEO fields (ENHANCED)
	writerPerms := []models.Permission{
		{
			Module:     "ContentEntry",
			Action:     "create",
			FieldScope: "non_seo_only",
		},
		{
			Module:     "ContentEntry",
			Action:     "read",
			FieldScope: "all",
		},
		{
			Module:     "ContentEntry",
			Action:     "update",
			FieldScope: "non_seo_only",
		},
		{Module: "Media", Action: "create"},
		{Module: "Media", Action: "read"},
		// Content writer TIDAK perlu SEO access
	}
	_, _ = CreateRole(database.DB, "content_writer", "Can create/edit content (non-SEO fields)", writerPerms)

	return nil
}

func SeedWorkflowTransitions(db *gorm.DB) error {
	transitions := []models.WorkflowTransition{
		// From Draft
		{FromStatus: "draft", ToStatus: "in_review", RequiredRole: "editor"},
		{FromStatus: "draft", ToStatus: "in_review", RequiredRole: "admin"},

		// From In Review
		{FromStatus: "in_review", ToStatus: "ready_for_approval", RequiredRole: "editor"},
		{FromStatus: "in_review", ToStatus: "ready_for_approval", RequiredRole: "admin"},
		{FromStatus: "in_review", ToStatus: "rejected", RequiredRole: "editor"},
		{FromStatus: "in_review", ToStatus: "rejected", RequiredRole: "admin"},
		{FromStatus: "in_review", ToStatus: "draft", RequiredRole: "editor"},
		{FromStatus: "in_review", ToStatus: "draft", RequiredRole: "admin"},

		// From Ready for Approval
		{FromStatus: "ready_for_approval", ToStatus: "approved", RequiredRole: "manager"},
		{FromStatus: "ready_for_approval", ToStatus: "approved", RequiredRole: "admin"},
		{FromStatus: "ready_for_approval", ToStatus: "rejected", RequiredRole: "manager"},
		{FromStatus: "ready_for_approval", ToStatus: "rejected", RequiredRole: "admin"},

		// From Approved
		{FromStatus: "approved", ToStatus: "published", RequiredRole: "manager"},
		{FromStatus: "approved", ToStatus: "published", RequiredRole: "admin"},

		// From Rejected
		{FromStatus: "rejected", ToStatus: "draft", RequiredRole: "editor"},
		{FromStatus: "rejected", ToStatus: "draft", RequiredRole: "admin"},
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
