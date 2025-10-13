package server

import (
	"time"

	"github.com/Kyz7/cms/internal/auth"
	"github.com/Kyz7/cms/internal/content"
	"github.com/Kyz7/cms/internal/media"
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/Kyz7/cms/internal/role"
	"github.com/Kyz7/cms/internal/search"
	"github.com/Kyz7/cms/internal/user"
	"github.com/Kyz7/cms/internal/workflow"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRoutes(app *fiber.App) {
	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS, PATCH",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "CMS API is running",
		})
	})

	// ==========================================
	// AUTH ROUTES (No authentication required)
	// ==========================================
	authGroup := app.Group("/auth")
	app.Use("/auth", limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
	}))
	authGroup.Post("/register", auth.RegisterHandler)
	authGroup.Post("/login", limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	}), auth.LoginHandler)
	authGroup.Get("/google/login", auth.GoogleLogin)
	authGroup.Get("/google/callback", auth.GoogleCallback)
	authGroup.Post("/forgot-password", auth.ForgotPasswordHandler)
	authGroup.Post("/reset-password", auth.ResetPasswordHandler)
	authGroup.Post("/refresh", limiter.New(limiter.Config{
		Max:        3,
		Expiration: 5 * time.Minute,
	}), auth.RefreshHandler)
	authGroup.Post("/logout", auth.JWTProtected(), auth.LogoutHandler)
	// ==========================================
	// USER MANAGEMENT (Admin only)
	// ==========================================
	userGroup := app.Group("/users")
	userGroup.Use(auth.JWTProtected())
	userGroup.Use(auth.RoleProtected("admin"))
	userGroup.Post("/", user.CreateUserHandler)
	userGroup.Get("/", user.ListUsersHandler)
	userGroup.Get("/:id", user.GetUserHandler)
	userGroup.Put("/:id", user.UpdateUserHandler)
	userGroup.Delete("/:id", user.DeleteUserHandler)

	// ==========================================
	// ROLE MANAGEMENT (Admin only)
	// ==========================================
	roleGroup := app.Group("/roles")
	roleGroup.Use(auth.JWTProtected())
	roleGroup.Use(auth.RoleProtected("admin"))
	roleGroup.Post("/", role.CreateRoleHandler)
	roleGroup.Get("/", role.ListRolesHandler)
	roleGroup.Get("/:id", role.GetRoleHandler)
	roleGroup.Put("/:id", role.UpdateRoleHandler)
	roleGroup.Delete("/:id", role.DeleteRoleHandler)
	roleGroup.Post("/:id/duplicate", role.DuplicateRoleHandler)
	roleGroup.Post("/assign", role.AssignRoleToUserHandler)

	// ==========================================
	// CONTENT MANAGEMENT
	// ==========================================
	contentGroup := app.Group("/content")
	contentGroup.Use(auth.JWTProtected())

	// Content Types
	contentGroup.Post("/types",
		middleware.PermissionProtected("ContentEntry", "create"),
		content.CreateContentTypeHandler)
	contentGroup.Get("/types",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.ListContentTypesHandler)
	contentGroup.Get("/types/:id",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GetContentTypeHandler)
	contentGroup.Put("/types/:id",
		middleware.PermissionProtected("ContentEntry", "update"),
		content.UpdateContentTypeHandler)
	contentGroup.Delete("/types/:id",
		middleware.PermissionProtected("ContentEntry", "delete"),
		content.DeleteContentTypeHandler)

	// Content Fields
	contentGroup.Post("/types/:content_type_id/fields",
		middleware.PermissionProtected("ContentEntry", "update"),
		content.AddFieldHandler)
	contentGroup.Put("/fields/:field_id",
		middleware.PermissionProtected("ContentEntry", "update"),
		content.UpdateFieldHandler)
	contentGroup.Delete("/fields/:field_id",
		middleware.PermissionProtected("ContentEntry", "delete"),
		content.DeleteFieldHandler)

	// Content Entries - List by Content Type
	contentGroup.Post("/:content_type_id/entries",
		middleware.PermissionProtected("ContentEntry", "create"),
		content.CreateEntryHandler)
	contentGroup.Get("/:content_type_id/entries",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.ListEntriesHandler)
	contentGroup.Post("/:content_type_id/entries/json",
		middleware.PermissionProtected("ContentEntry", "create"),
		content.CreateEntryHandlerJSON)

	// Content Entries - Single Entry Operations
	contentGroup.Get("/entries/:entry_id",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GetEntryHandler)
	contentGroup.Put("/entries/:entry_id",
		middleware.PermissionProtected("ContentEntry", "update"),
		content.UpdateEntryHandler)
	contentGroup.Delete("/entries/:entry_id",
		middleware.PermissionProtected("ContentEntry", "delete"),
		content.DeleteEntryHandler)

	// SEO
	contentGroup.Get("/entries/:entry_id/seo-preview",
		middleware.PermissionProtected("SEO", "read"),
		content.SEOPreviewHandler)

	// Relations
	contentGroup.Post("/:from_content_id/relations",
		middleware.PermissionProtected("ContentEntry", "update"),
		content.CreateRelationHandler)
	contentGroup.Get("/:from_content_id/relations",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.ListRelationsHandler)
	contentGroup.Delete("/relations/:relation_id",
		middleware.PermissionProtected("ContentEntry", "delete"),
		content.DeleteRelationHandler)

	// API Documentation & Reference
	contentGroup.Get("/types/:id/api-reference",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GenerateAPIReferenceHandler)

	contentGroup.Get("/types/:id/openapi",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GenerateOpenAPISpecHandler)

	contentGroup.Get("/types/:id/docs/markdown",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GenerateMarkdownDocsHandler)

	// Field Validation Rules
	contentGroup.Get("/fields/:field_id/validation",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GetFieldValidationHandler)

	// ==========================================
	// MEDIA LIBRARY
	// ==========================================
	mediaGroup := app.Group("/media")
	mediaGroup.Use(auth.JWTProtected())

	// Media Folders
	mediaGroup.Get("/folders",
		middleware.PermissionProtected("Media", "read"),
		media.ListFoldersHandler)
	mediaGroup.Post("/folders",
		middleware.PermissionProtected("Media", "create"),
		media.CreateFolderHandler)

	mediaGroup.Post("/upload",
		middleware.PermissionProtected("Media", "create"),
		media.UploadMediaHandler)
	mediaGroup.Post("/bulk-upload",
		middleware.PermissionProtected("Media", "create"),
		media.BulkUploadMediaHandler)
	mediaGroup.Get("/",
		middleware.PermissionProtected("Media", "read"),
		media.ListMediaHandler)
	mediaGroup.Get("/search",
		middleware.PermissionProtected("Media", "read"),
		media.SearchMediaHandler)
	mediaGroup.Get("/stats",
		middleware.PermissionProtected("Media", "read"),
		media.GetMediaStatsHandler)
	mediaGroup.Get("/:id",
		middleware.PermissionProtected("Media", "read"),
		media.GetMediaHandler)
	mediaGroup.Put("/:id",
		middleware.PermissionProtected("Media", "update"),
		media.UpdateMediaHandler)
	mediaGroup.Delete("/:id",
		middleware.PermissionProtected("Media", "delete"),
		media.DeleteMediaHandler)

	// ==========================================
	// WORKFLOW
	// ==========================================
	workflowGroup := app.Group("/workflow")
	workflowGroup.Use(auth.JWTProtected())

	// Status Changes
	workflowGroup.Post("/entries/:entry_id/status",
		middleware.PermissionProtected("ContentEntry", "update"),
		workflow.ChangeStatusHandler)
	workflowGroup.Post("/entries/:entry_id/request-review",
		middleware.PermissionProtected("ContentEntry", "update"),
		workflow.RequestReviewHandler)
	workflowGroup.Post("/entries/:entry_id/approve",
		middleware.PermissionProtected("ContentEntry", "approve"),
		workflow.ApproveEntryHandler)
	workflowGroup.Post("/entries/:entry_id/reject",
		middleware.PermissionProtected("ContentEntry", "approve"),
		workflow.RejectEntryHandler)
	workflowGroup.Post("/entries/:entry_id/publish",
		middleware.PermissionProtected("ContentEntry", "approve"),
		workflow.PublishEntryHandler)

	// History & Comments
	workflowGroup.Get("/entries/:entry_id/history",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetHistoryHandler)
	workflowGroup.Post("/entries/:entry_id/comments",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.AddCommentHandler)
	workflowGroup.Get("/entries/:entry_id/comments",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetCommentsHandler)

	// Assignment
	workflowGroup.Post("/entries/:entry_id/assign",
		middleware.PermissionProtected("ContentEntry", "approve"),
		workflow.AssignEntryHandler)
	workflowGroup.Get("/assignments",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetMyAssignmentsHandler)
	workflowGroup.Put("/assignments/:assignment_id/complete",
		middleware.PermissionProtected("ContentEntry", "update"),
		workflow.CompleteAssignmentHandler)

	// Statistics & Filtering
	workflowGroup.Get("/content-types/:content_type_id/entries",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetEntriesByStatusHandler)
	workflowGroup.Get("/content-types/:content_type_id/stats",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetWorkflowStatsHandler)

	// ==========================================
	// SEARCH & FILTERING
	// ==========================================
	searchGroup := app.Group("/search")
	searchGroup.Use(auth.JWTProtected())

	// Full-text search
	searchGroup.Get("/entries",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.SearchEntriesHandler)

	// Advanced search with filters
	searchGroup.Post("/advanced",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.AdvancedSearchHandler)

	// Search facets for filtering UI
	searchGroup.Get("/facets",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.GetSearchFacetsHandler)

	// Autocomplete
	searchGroup.Get("/autocomplete",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.AutoCompleteHandler)

	// Search by relation
	searchGroup.Get("/entries/:entry_id/related",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.SearchByRelationHandler)

	// Bulk search across multiple content types
	searchGroup.Post("/bulk",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.BulkSearchHandler)

	// Export search results
	searchGroup.Get("/export",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.ExportSearchResultsHandler)

	// Search statistics
	searchGroup.Get("/stats",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.SearchStatsHandler)

	// Search suggestions
	searchGroup.Get("/suggestions",
		middleware.PermissionProtected("ContentEntry", "read"),
		search.SearchSuggestionsHandler)
}
