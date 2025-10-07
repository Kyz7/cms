package models

import (
	"time"

	"gorm.io/gorm"
)

type WorkflowStatus string

const (
	StatusDraft            WorkflowStatus = "draft"
	StatusInReview         WorkflowStatus = "in_review"
	StatusReadyForApproval WorkflowStatus = "ready_for_approval"
	StatusApproved         WorkflowStatus = "approved"
	StatusPublished        WorkflowStatus = "published"
	StatusRejected         WorkflowStatus = "rejected"
)

func EnsureEnum(db *gorm.DB) error {
	return db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'workflow_status') THEN
				CREATE TYPE workflow_status AS ENUM (
					'draft',
					'in_review',
					'ready_for_approval',
					'approved',
					'published',
					'rejected'
				);
			END IF;
		END
		$$;
	`).Error
}

type WorkflowTransition struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	FromStatus   WorkflowStatus `gorm:"type:workflow_status" json:"from_status"`
	ToStatus     WorkflowStatus `gorm:"type:workflow_status" json:"to_status"`
	RequiredRole string         `gorm:"size:50" json:"required_role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type WorkflowHistory struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EntryID    uint           `json:"entry_id"`
	Entry      *ContentEntry  `gorm:"foreignKey:EntryID" json:"entry,omitempty"`
	FromStatus WorkflowStatus `gorm:"type:workflow_status" json:"from_status"`
	ToStatus   WorkflowStatus `gorm:"type:workflow_status" json:"to_status"`
	ChangedBy  uint           `json:"changed_by"`
	User       *User          `gorm:"foreignKey:ChangedBy" json:"user,omitempty"`
	Comment    string         `gorm:"type:text" json:"comment"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type WorkflowComment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	EntryID   uint           `json:"entry_id"`
	Entry     *ContentEntry  `gorm:"foreignKey:EntryID" json:"entry,omitempty"`
	UserID    uint           `json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Comment   string         `gorm:"type:text" json:"comment"`
	IsPrivate bool           `json:"is_private"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type WorkflowAssignment struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	EntryID    uint           `json:"entry_id"`
	Entry      *ContentEntry  `gorm:"foreignKey:EntryID" json:"entry,omitempty"`
	AssignedTo uint           `json:"assigned_to"`
	User       *User          `gorm:"foreignKey:AssignedTo" json:"user,omitempty"`
	AssignedBy uint           `json:"assigned_by"`
	Assigner   *User          `gorm:"foreignKey:AssignedBy" json:"assigner,omitempty"`
	Status     string         `gorm:"size:50;default:'pending'" json:"status"`
	DueDate    *time.Time     `json:"due_date,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
