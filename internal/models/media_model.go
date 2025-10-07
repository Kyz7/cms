package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MediaFile struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	FileName   string         `gorm:"size:255" json:"file_name"`
	URL        string         `gorm:"size:500" json:"url"`
	Type       string         `gorm:"size:100;index" json:"type"`
	Size       int64          `json:"size"`
	Width      *int           `json:"width,omitempty"`
	Height     *int           `json:"height,omitempty"`
	Folder     string         `gorm:"size:255;index" json:"folder"`
	Tags       datatypes.JSON `json:"tags,omitempty"`
	Alt        string         `gorm:"size:255" json:"alt"`
	Caption    string         `gorm:"type:text" json:"caption"`
	UploadedBy uint           `gorm:"index" json:"uploaded_by"`
	Uploader   *User          `gorm:"foreignKey:UploadedBy" json:"uploader,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type MediaFolder struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100" json:"name"`
	Path      string         `gorm:"size:255;uniqueIndex" json:"path"`
	ParentID  *uint          `json:"parent_id,omitempty"`
	Parent    *MediaFolder   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	CreatedBy uint           `json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
