package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;uniqueIndex" json:"name"`
	Description string         `json:"description"`
	Permissions []Permission   `gorm:"foreignKey:RoleID" json:"permissions"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Permission struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	RoleID         uint           `gorm:"index:idx_role_module_action" json:"role_id"`
	Module         string         `gorm:"size:50;index:idx_role_module_action" json:"module"`
	Action         string         `gorm:"size:50;index:idx_role_module_action" json:"action"`
	FieldScope     string         `json:"field_scope,omitempty"`      // "all", "seo_only", "non_seo_only", "custom"
	AllowedFields  datatypes.JSON `json:"allowed_fields,omitempty"`   // ["title", "slug", "description"]
	DeniedFields   datatypes.JSON `json:"denied_fields,omitempty"`    // ["internal_notes", "pricing"]
	ContentTypeIDs datatypes.JSON `json:"content_type_ids,omitempty"` // [1, 2, 3] - limit to specific content types
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
