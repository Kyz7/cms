package database

import (
	"fmt"
	"log"

	"github.com/Kyz7/cms/internal/config"
	"github.com/Kyz7/cms/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	DB = db

	return db, nil
}

func Migrate(db *gorm.DB) error {

	if err := models.EnsureEnum(db); err != nil {
		log.Fatal("failed to create enum:", err)
	}
	err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.ContentType{},
		&models.ContentField{},
		&models.ContentEntry{},
		&models.ContentRelation{},
		&models.PasswordResetToken{},
		&models.ResetToken{},
		&models.RefreshToken{},
		&models.WorkflowTransition{},
		&models.WorkflowHistory{},
		&models.WorkflowComment{},
		&models.WorkflowAssignment{},
		&models.MediaFile{},
		&models.MediaFolder{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}
	log.Println("Database migrated successfully!")
	return nil
}
