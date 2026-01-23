package server

import (
	"os"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/auth"
	"github.com/Kyz7/cms/internal/content"
	"github.com/Kyz7/cms/internal/graphql"
	"github.com/Kyz7/cms/internal/media"
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/Kyz7/cms/internal/project"
	"github.com/Kyz7/cms/internal/role"
	"github.com/Kyz7/cms/internal/search"
	"github.com/Kyz7/cms/internal/swagger"
	"github.com/Kyz7/cms/internal/user"
	"github.com/Kyz7/cms/internal/workflow"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	// Middleware
	app.Use(logger.New())

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000,http://localhost:8080"
	}
	originsList := strings.Split(corsOrigins, ",")
	for i, origin := range originsList {
		originsList[i] = strings.TrimSpace(origin)
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(originsList, ","),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-Token",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	app.Use(csrf.New(csrf.Config{
		KeyLookup:      "header:X-CSRF-Token",
		CookieName:     "csrf_",
		CookieSameSite: "Lax",
		CookieHTTPOnly: true,
		CookieSecure:   os.Getenv("APP_ENV") == "production",
		Expiration:     1 * time.Hour,
		Next: func(c *fiber.Ctx) bool {
			path := c.Path()
			if path == "/health" || strings.HasPrefix(path, "/swagger") || path == "/openapi.yaml" {
				return true
			}
			// Skip CSRF when using JWT Authorization for API calls
			if authHeader := c.Get("Authorization"); authHeader != "" {
				return true
			}
			// Skip CSRF for auth endpoints (except logout which requires JWT)
			if strings.HasPrefix(path, "/auth/") && path != "/auth/logout" {
				return true
			}
			// Skip CSRF for GraphQL endpoints (they use JWT authentication)
			if strings.HasPrefix(path, "/graphql") {
				return true
			}
			return false
		},
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "CMS API is running",
		})
	})

	app.Get("/csrf-token", func(c *fiber.Ctx) error {
		token, ok := c.Locals("csrf").(string)
		if !ok {
			token = c.GetRespHeader("X-CSRF-Token")
		}

		if token == "" {
			return c.Status(500).JSON(fiber.Map{"error": "Check middleware order"})
		}

		return c.JSON(fiber.Map{"csrf_token": token})
	})

	// ==========================================
	// SWAGGER DOCUMENTATION
	// ==========================================
	// Setup Swagger routes (serves OpenAPI spec and Swagger UI)
	swagger.SetupSwaggerRoutes(app, "./openapi.yaml")

	// ==========================================
	// GRAPHQL ROUTES
	// ==========================================
	// Initialize GraphQL resolvers
	graphqlResolvers := graphql.NewResolvers(db)

	// GraphQL endpoint
	app.Post("/graphql", graphqlResolvers.GraphQLHandler())

	// GraphiQL playground (development only)
	app.Get("/graphql", graphqlResolvers.GraphiQLHandler())

	// Batch GraphQL endpoint
	app.Post("/graphql/batch", graphqlResolvers.BatchGraphQLHandler())

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
	authGroup.Get("/me", auth.JWTProtected(), auth.MeHandler)
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
	roleGroup.Post("/normalize", role.NormalizeRoleScopesHandler)

	// ==========================================
	// PROJECT MANAGEMENT
	// ==========================================
	projectGroup := app.Group("/projects")
	projectGroup.Use(auth.JWTProtected())

	// Project CRUD
	projectGroup.Post("/", project.CreateProjectHandler)
	projectGroup.Get("/", project.ListProjectsHandler)
	projectGroup.Get("/:id", project.GetProjectHandler)
	projectGroup.Put("/:id", project.UpdateProjectHandler)
	projectGroup.Delete("/:id", project.DeleteProjectHandler)

	// Project Members
	projectGroup.Post("/:id/members", project.AddProjectMemberHandler)
	projectGroup.Get("/:id/members", project.ListProjectMembersHandler)
	projectGroup.Put("/:id/members/:member_id", project.UpdateProjectMemberRoleHandler)
	projectGroup.Delete("/:id/members/:member_id", project.RemoveProjectMemberHandler)

	// ==========================================
	// CONTENT MANAGEMENT
	// ==========================================
	// Public preview endpoint (no JWT, only requires preview token)
	app.Get("/content/entries/:entry_id/preview", content.PreviewEntryHandler)

	contentGroup := app.Group("/content")
	contentGroup.Use(auth.JWTProtected())

	// Content Types
	contentGroup.Post("/types",
		middleware.PermissionProtected("ContentType", "create"),
		content.CreateContentTypeHandler)
	contentGroup.Get("/types",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.ListContentTypesHandler)
	contentGroup.Get("/types/:id",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GetContentTypeHandler)
	contentGroup.Put("/types/:id",
		middleware.PermissionProtected("ContentType", "update"),
		content.UpdateContentTypeHandler)
	contentGroup.Delete("/types/:id",
		middleware.PermissionProtected("ContentType", "delete"),
		content.DeleteContentTypeHandler)

	// Content Fields
	contentGroup.Post("/types/:content_type_id/fields",
		middleware.PermissionProtected("ContentType", "update"),
		content.AddFieldHandler)
	// Rute Baru (Lebih Kontekstual)
	contentGroup.Put("/:content_type_id/fields/:field_id",
		middleware.PermissionProtected("ContentType", "update"),
		content.UpdateFieldHandler)
	contentGroup.Delete("/:content_type_id/fields/:field_id",
		middleware.PermissionProtected("ContentType", "delete"),
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
	contentGroup.Get("/entries", middleware.PermissionProtected("ContentEntry", "read"),
		content.GetEntriesSemuaHandler)

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

	// Translation
	contentGroup.Post("/entries/:entry_id/translate",
		middleware.PermissionProtected("ContentEntry", "update"),
		content.TranslateEntryHandler)

	// SEO
	contentGroup.Get("/entries/:entry_id/seo-preview",
		middleware.PermissionProtected("SEO", "read"),
		content.SEOPreviewHandler)
	contentGroup.Post("/entries/:entry_id/preview-token",
		middleware.PermissionProtected("ContentEntry", "read"),
		content.GeneratePreviewTokenHandler)

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
		middleware.PermissionProtected("ContentEntry", "update"), // Menggunakan update permission agar editor bisa assign
		workflow.AssignEntryHandler)
	workflowGroup.Get("/entries/:entry_id/active-assignment",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetActiveAssignmentHandler)
	workflowGroup.Get("/assignees",
		middleware.PermissionProtected("ContentEntry", "read"),
		workflow.GetAssigneesHandler)
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
