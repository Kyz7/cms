package models

import (
	"time"

	"gorm.io/gorm"
)

type PasswordResetToken struct {
	ID        uint      `gorm:"primarykey"`
	UserID    uint      `gorm:"not null"`
	Token     string    `gorm:"unique;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

type ResetToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	TokenHash string    `gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

type RefreshToken struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index" json:"user_id"`
	TokenHash string         `gorm:"uniqueIndex" json:"-"` // hashed, bukan plain
	ExpiresAt time.Time      `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	Revoked   bool           `gorm:"default:false" json:"revoked"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
