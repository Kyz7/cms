package database

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"gorm.io/gorm"
)

type Migration struct {
	ID        uint   `gorm:"primaryKey"`
	Version   string `gorm:"uniqueIndex;size:255"`
	AppliedAt string
}

func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	migrationsPath := "./migrations"
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}

	for _, file := range files {
		filename := filepath.Base(file)

		var existingMigration Migration
		result := db.Where("version = ?", filename).First(&existingMigration)

		if result.Error == nil {
			log.Printf("⏭️  Skipping migration: %s (already applied)", filename)
			continue
		}

		sqlContent, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", filename, err)
		}

		log.Printf("▶️  Applying migration: %s", filename)
		if err := db.Exec(string(sqlContent)).Error; err != nil {
			return fmt.Errorf("failed to execute migration %s: %v", filename, err)
		}

		migration := Migration{
			Version:   filename,
			AppliedAt: fmt.Sprintf("%v", db.NowFunc()),
		}
		if err := db.Create(&migration).Error; err != nil {
			return fmt.Errorf("failed to record migration %s: %v", filename, err)
		}

		log.Printf("✅ Applied migration: %s", filename)
	}

	log.Println("🎉 All migrations completed successfully")
	return nil
}

func RollbackMigration(db *gorm.DB, version string) error {
	var migration Migration
	if err := db.Where("version = ?", version).First(&migration).Error; err != nil {
		return fmt.Errorf("migration not found: %s", version)
	}

	rollbackFile := fmt.Sprintf("./migrations/rollback_%s", version)
	sqlContent, err := ioutil.ReadFile(rollbackFile)
	if err != nil {
		return fmt.Errorf("rollback file not found: %s", rollbackFile)
	}

	if err := db.Exec(string(sqlContent)).Error; err != nil {
		return fmt.Errorf("failed to rollback migration: %v", err)
	}

	if err := db.Delete(&migration).Error; err != nil {
		return fmt.Errorf("failed to delete migration record: %v", err)
	}

	log.Printf("⏪ Rolled back migration: %s", version)
	return nil
}
func GetAppliedMigrations(db *gorm.DB) ([]Migration, error) {
	var migrations []Migration
	if err := db.Order("applied_at DESC").Find(&migrations).Error; err != nil {
		return nil, err
	}
	return migrations, nil
}
