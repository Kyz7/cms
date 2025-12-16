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

type SwaggerHandler struct {
	openAPIPath string
}

func NewSwaggerHandler(openAPIPath string) *SwaggerHandler {
	return &SwaggerHandler{
		openAPIPath: openAPIPath,
	}
}

func (h *SwaggerHandler) ServeSwaggerUI(c *fiber.Ctx) error {
	fullPath := c.Path()

	if fullPath == "/swagger" || fullPath == "/swagger/" {
		return h.serveEmbeddedFile(c, "index.html")
	}

	requestPath := strings.TrimPrefix(fullPath, "/swagger/")

	return h.serveEmbeddedFile(c, requestPath)
}

func (h *SwaggerHandler) ServeOpenAPISpec(c *fiber.Ctx) error {
	return c.SendFile(h.openAPIPath)
}

func (h *SwaggerHandler) serveEmbeddedFile(c *fiber.Ctx, reqPath string) error {
	reqPath = path.Clean(reqPath)
	if reqPath == "." || reqPath == "" {
		reqPath = "index.html"
	}
	filePath := path.Join("swagger-ui", reqPath)

	data, err := swaggerUIFiles.ReadFile(filePath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("File not found: " + filePath)
	}

	contentType := h.getContentType(reqPath)
	c.Set("Content-Type", contentType)

	if strings.HasSuffix(reqPath, "index.html") {
		content := string(data)
		content = strings.ReplaceAll(content, "{{OPENAPI_URL}}", "/swagger/openapi.yaml")
		return c.SendString(content)
	}

	return c.Send(data)
}

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

func SetupSwaggerRoutes(app *fiber.App, openAPIPath string) {
	handler := NewSwaggerHandler(openAPIPath)

	app.Get("/swagger/openapi.yaml", handler.ServeOpenAPISpec)
	app.Get("/swagger/openapi.json", handler.ServeOpenAPISpec)

	app.Get("/swagger", handler.ServeSwaggerUI)
	app.Get("/swagger/", handler.ServeSwaggerUI)
	app.Get("/swagger/*", handler.ServeSwaggerUI)
}
