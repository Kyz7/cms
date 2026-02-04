package main

import (
	"log"
	"os"
	"time"

	"github.com/Kyz7/cms/internal/config"
	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/globals"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/role"
	"github.com/Kyz7/cms/internal/server"
	"github.com/Kyz7/cms/internal/utils"
)

func main() {
	cfg := config.Load()

	if err := utils.ValidateJWTSecret(); err != nil {
		log.Fatal("JWT Configuration Error: ", err)
	}
	log.Println("JWT secret validated")

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

	useSupabase := os.Getenv("USE_SUPABASE")

	if useSupabase == "true" {
		params := []string{
			os.Getenv("SUPABASE_URL"),
			os.Getenv("SUPABASE_KEY"),
			os.Getenv("SUPABASE_BUCKET"),
		}

		valid := true
		for _, p := range params {
			if p == "" {
				valid = false
				break
			}
		}

		if valid {
			utils.InitSupabase(params[0], params[1], params[2])
			log.Println("✅ Supabase storage initialized")
			log.Printf("☁️  Using Supabase: %s", params[2])
		} else {
			log.Println("⚠️  USE_SUPABASE=true but missing configuration")
			log.Println("⚠️  Falling back to local storage")
			utils.SetStorageMode(true)
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
	// Normalize legacy databases: ensure is_global flags are correct
	if err := role.NormalizeRoleScopes(database.DB); err != nil {
		log.Println("⚠️  Failed to normalize role scopes:", err)
	} else {
		log.Println("✅ Role scopes normalized (is_global flags updated)")
	}
	// Ensure global roles have schema permissions for Content Builder & Fields
	if err := role.EnsureGlobalSchemaPermissions(database.DB); err != nil {
		log.Println("⚠️  Failed to ensure global schema permissions:", err)
	} else {
		log.Println("✅ Global schema permissions ensured for editor/content_writer")
	}
	// Ensure editor can create/update entries across all content types
	if err := role.EnsureEditorContentEntryUnrestricted(database.DB); err != nil {
		log.Println("⚠️  Failed to ensure editor entry permissions:", err)
	} else {
		log.Println("✅ Editor entry permissions normalized (unrestricted ContentTypeIDs, FieldScope=all)")
	}
	if err := role.EnsureProjectSchemaPermissions(database.DB); err != nil {
		log.Println("⚠️  Failed to ensure project schema permissions:", err)
	} else {
		log.Println("✅ Project schema permissions ensured for project roles")
	}

	if err := role.SeedWorkflowTransitions(database.DB); err != nil {
		log.Println("⚠️  Failed to seed workflow transitions:", err)
	} else {
		log.Println("✅ Workflow transitions seeded")
	}

	if err := globals.InitRoleCache(database.DB); err != nil {
		log.Println("⚠️  Failed to initialize role cache:", err)
	} else {
		log.Println("✅ Role cache initialized")
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
