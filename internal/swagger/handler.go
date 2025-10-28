package swagger

import (
	"embed"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

//go:embed swagger-ui
var swaggerUIFiles embed.FS

// SwaggerHandler handles serving Swagger UI and OpenAPI spec
type SwaggerHandler struct {
	openAPIPath string
}

// NewSwaggerHandler creates a new Swagger handler
func NewSwaggerHandler(openAPIPath string) *SwaggerHandler {
	return &SwaggerHandler{
		openAPIPath: openAPIPath,
	}
}

// ServeSwaggerUI serves the Swagger UI interface
func (h *SwaggerHandler) ServeSwaggerUI(c *fiber.Ctx) error {
	fullPath := c.Path()

	// If accessing root, serve index.html
	if fullPath == "/swagger" || fullPath == "/swagger/" {
		return h.serveEmbeddedFile(c, "index.html")
	}

	// Remove /swagger/ prefix from path
	requestPath := strings.TrimPrefix(fullPath, "/swagger/")

	// Serve files from embedded filesystem
	return h.serveEmbeddedFile(c, requestPath)
}

// ServeOpenAPISpec serves the OpenAPI specification
func (h *SwaggerHandler) ServeOpenAPISpec(c *fiber.Ctx) error {
	return c.SendFile(h.openAPIPath)
}

func (h *SwaggerHandler) serveEmbeddedFile(c *fiber.Ctx, reqPath string) error {
	// Clean the path using path.Clean (for forward slashes)
	reqPath = path.Clean(reqPath)
	if reqPath == "." || reqPath == "" {
		reqPath = "index.html"
	}

	// Build file path using path.Join (NOT filepath.Join!)
	// This ensures forward slashes for embed.FS
	filePath := path.Join("swagger-ui", reqPath)

	// Read file from embedded FS
	data, err := swaggerUIFiles.ReadFile(filePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("File not found: " + filePath)
	}

	// Set appropriate content type
	contentType := h.getContentType(reqPath)
	c.Set("Content-Type", contentType)

	// If index.html, inject OpenAPI URL
	if strings.HasSuffix(reqPath, "index.html") {
		content := string(data)
		content = strings.ReplaceAll(content, "{{OPENAPI_URL}}", "/swagger/openapi.yaml")
		return c.SendString(content)
	}

	return c.Send(data)
}

// getContentType returns the appropriate content type for the file
func (h *SwaggerHandler) getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".svg":
		return "image/svg+xml"
	default:
		return "text/plain"
	}
}

// SetupSwaggerRoutes sets up Swagger-related routes
func SetupSwaggerRoutes(app *fiber.App, openAPIPath string) {
	handler := NewSwaggerHandler(openAPIPath)

	// Serve OpenAPI spec (specific routes first)
	app.Get("/swagger/openapi.yaml", handler.ServeOpenAPISpec)
	app.Get("/swagger/openapi.json", handler.ServeOpenAPISpec)

	// Serve Swagger UI files (wildcard last)
	app.Get("/swagger", handler.ServeSwaggerUI)
	app.Get("/swagger/", handler.ServeSwaggerUI)
	app.Get("/swagger/*", handler.ServeSwaggerUI)
}
