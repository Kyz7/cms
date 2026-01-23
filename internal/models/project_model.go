package models

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	Name         string          `gorm:"size:255;not null" json:"name"`
	Description  string          `gorm:"type:text" json:"description,omitempty"`
	CreatedBy    uint            `json:"created_by"`
	Creator      *User           `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Members      []ProjectMember `gorm:"foreignKey:ProjectID" json:"members,omitempty"`
	ContentTypes []ContentType   `gorm:"foreignKey:ProjectID" json:"content_types,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `gorm:"index" json:"-"`
}

type ProjectMember struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProjectID uint           `gorm:"index;not null" json:"project_id"`
	Project   *Project       `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role      *Role          `gorm:"foreignKey:RoleID;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE" json:"role,omitempty"`
	RoleID    uint           `json:"role_id"`
	InvitedBy uint           `json:"invited_by,omitempty"`
	Inviter   *User          `gorm:"foreignKey:InvitedBy" json:"inviter,omitempty"`
	Status    string         `gorm:"size:20;default:'active'" json:"status"` // active, pending, inactive
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProjectMemberRole constants
const (
	ProjectRoleOwner         = "ProjectOwner"
	ProjectRoleAdmin         = "ProjectAdmin"
	ProjectRoleEditor        = "ProjectEditor"
	ProjectRoleViewer        = "ProjectViewer"
	ProjectRoleContentWriter = "ProjectContentWriter"
)
