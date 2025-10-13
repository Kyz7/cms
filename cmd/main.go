package main

import (
	"log"
	"os"
	"time"

	"github.com/Kyz7/cms/internal/config"
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/role"
	"github.com/Kyz7/cms/internal/server"
	"github.com/Kyz7/cms/internal/utils"
)

func main() {
	cfg := config.Load()

	if err := utils.ValidateJWTSecret(); err != nil {
		log.Fatal("❌ JWT Configuration Error: ", err)
	}
	log.Println("✅ JWT secret validated")

	requiredEnvVars := map[string]string{
		"DB_HOST":     os.Getenv("DB_HOST"),
		"DB_NAME":     os.Getenv("DB_NAME"),
		"DB_USER":     os.Getenv("DB_USER"),
		"DB_PASSWORD": os.Getenv("DB_PASSWORD"),
	}

	for key, value := range requiredEnvVars {
		if value == "" {
			log.Fatalf("❌ Required environment variable %s is not set", key)
		}
	}
	log.Println("✅ Required environment variables validated")

	// ========== DATABASE SETUP ==========
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("❌ Database connection failed:", err)
	}
	database.DB = db

	// Run GORM AutoMigrate first
	if err := database.Migrate(db); err != nil {
		log.Fatal("❌ Migration failed: ", err)
	}
	log.Println("✅ Database migrated successfully")

	// ========== RUN SQL MIGRATIONS (FOR SEARCH INDEXES) ==========
	log.Println("🔍 Running SQL migrations for search indexes...")
	if err := database.RunMigrations(db); err != nil {
		log.Printf("⚠️  SQL migrations failed: %v", err)
		log.Println("⚠️  Search features may not work optimally")
		log.Println("💡 Run migrations manually: psql -U user -d dbname -f migrations/001_add_search_indexes.sql")
	} else {
		log.Println("✅ SQL migrations completed successfully")
	}

	// ========== STORAGE SETUP ==========
	if err := utils.InitLocalStorage(); err != nil {
		log.Fatal("❌ Failed to initialize local storage:", err)
	}
	log.Println("✅ Local storage initialized at ./uploads/")

	useS3 := os.Getenv("USE_S3")
	if useS3 == "true" {
		s3Bucket := os.Getenv("S3_BUCKET")
		s3Region := os.Getenv("S3_REGION")
		cloudfrontURL := os.Getenv("CLOUDFRONT_URL")

		if s3Bucket != "" && s3Region != "" {
			if err := utils.InitS3(s3Bucket, s3Region, cloudfrontURL); err != nil {
				log.Println("⚠️  S3 initialization failed:", err)
				log.Println("⚠️  Falling back to local storage")
				utils.SetStorageMode(true)
			} else {
				log.Println("✅ S3 initialized successfully")
				log.Printf("☁️  Using S3: %s (region: %s)", s3Bucket, s3Region)
			}
		} else {
			log.Println("⚠️  USE_S3=true but S3_BUCKET or S3_REGION not configured")
			log.Println("⚠️  Falling back to local storage")
		}
	} else {
		log.Println("💾 Using LOCAL storage mode (./uploads/)")
		utils.SetStorageMode(true)
	}

	// ========== SEED DEFAULT DATA ==========
	if err := role.SeedDefaultRoles(); err != nil {
		log.Println("⚠️  Failed to seed roles (may already exist):", err)
	} else {
		log.Println("✅ Default roles seeded")
	}

	if err := role.SeedWorkflowTransitions(database.DB); err != nil {
		log.Println("⚠️  Failed to seed workflow transitions:", err)
	} else {
		log.Println("✅ Workflow transitions seeded")
	}

	// ========== BACKGROUND JOBS ==========
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			result := database.DB.Where("expires_at < ?", time.Now()).Delete(&models.ResetToken{})
			if result.RowsAffected > 0 {
				log.Printf("🧹 Cleaned up %d expired reset tokens", result.RowsAffected)
			}

			result = database.DB.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{})
			if result.RowsAffected > 0 {
				log.Printf("🧹 Cleaned up %d expired refresh tokens", result.RowsAffected)
			}
		}
	}()

	// ========== START SERVER ==========
	app := server.New(db)

	log.Printf("🚀 CMS Server starting on %s", cfg.ServerAddr)
	log.Printf("📚 API Documentation: %s/health", cfg.ServerAddr)
	log.Printf("💾 Storage Mode: %s", utils.GetStorageMode())
	log.Printf("🔐 JWT Authentication: Enabled")
	log.Printf("🔍 Full-Text Search: Enabled")

	if err := app.Listen(cfg.ServerAddr); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}
