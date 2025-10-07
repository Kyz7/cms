package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContentType struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;uniqueIndex" json:"name"`
	Slug      string         `gorm:"size:100;uniqueIndex" json:"slug"`
	EnableSEO bool           `json:"enable_seo"`
	Fields    []ContentField `gorm:"foreignKey:ContentTypeID" json:"fields"`
	SEOFields []ContentField `gorm:"foreignKey:ContentTypeID" json:"seo_fields"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ContentField struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	ContentTypeID uint   `json:"content_type_id"`
	Name          string `gorm:"size:100" json:"name"`
	Type          string `gorm:"size:50" json:"type"` // string, number, boolean, date, media, text, email, url
	Required      bool   `json:"required"`
	IsSEO         bool   `json:"is_seo"`

	// Enhanced Validation Fields
	Unique       bool     `json:"unique" gorm:"default:false"`
	MaxLength    *int     `json:"max_length,omitempty"`
	MinLength    *int     `json:"min_length,omitempty"`
	Pattern      string   `gorm:"size:255" json:"pattern,omitempty"`
	MinValue     *float64 `json:"min_value,omitempty"`
	MaxValue     *float64 `json:"max_value,omitempty"`
	DefaultValue string   `gorm:"size:500" json:"default_value,omitempty"`
	Placeholder  string   `gorm:"size:255" json:"placeholder,omitempty"`
	HelpText     string   `gorm:"size:500" json:"help_text,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ContentEntry struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	ContentTypeID uint           `json:"content_type_id"`
	Data          datatypes.JSON `json:"data"`
	Status        WorkflowStatus `gorm:"type:workflow_status;default:'draft';index" json:"status"`
	CreatedBy     uint           `gorm:"index" json:"created_by,omitempty"`
	UpdatedBy     uint           `gorm:"index" json:"updated_by,omitempty"`
	Creator       *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Updater       *User          `gorm:"foreignKey:UpdatedBy" json:"updater,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type ContentRelation struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	FromContentID uint           `json:"from_content_id"`
	ToContentID   uint           `json:"to_content_id"`
	RelationType  string         `gorm:"size:50" json:"relation_type"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
