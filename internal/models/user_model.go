package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100" json:"name"`
	Email     string         `gorm:"uniqueIndex;size:100" json:"email"`
	Password  string         `gorm:"size:255" json:"-"`
	Provider  string         `gorm:"size:50" json:"provider"`
	Status    string         `gorm:"size:20;default:'active'" json:"status"`
	RoleID    uint           `json:"role_id"`
	Role      *Role          `gorm:"foreignKey:RoleID;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE" json:"role,omitempty"`
	Profile   string         `json:"profile,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
