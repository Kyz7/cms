package content

import (
	"fmt"
	"strings"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/gofiber/fiber/v2"
)

type APIEndpoint struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Description string                 `json:"description"`
	Auth        bool                   `json:"auth_required"`
	Permission  string                 `json:"permission,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	RequestBody map[string]interface{} `json:"request_body,omitempty"`
	Response    map[string]interface{} `json:"response_example,omitempty"`
}

type APIReference struct {
	ContentType string                   `json:"content_type"`
	ContentID   uint                     `json:"content_type_id"`
	Slug        string                   `json:"slug"`
	Description string                   `json:"description"`
	BaseURL     string                   `json:"base_url"`
	Endpoints   []APIEndpoint            `json:"endpoints"`
	Fields      []map[string]interface{} `json:"fields"`
	SEOEnabled  bool                     `json:"seo_enabled"`
	CreatedAt   string                   `json:"created_at"`
}

func GenerateAPIReferenceHandler(c *fiber.Ctx) error {
	contentTypeID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid content_type_id"})
	}

	var ct models.ContentType
	if err := database.DB.
		Preload("Fields", "is_seo = ?", false).
		Preload("SEOFields", "is_seo = ?", true).
		First(&ct, contentTypeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "content type not found"})
	}

	baseURL := c.BaseURL()
	apiRef := generateAPIReference(ct, baseURL)

	return c.JSON(apiRef)
}

func generateAPIReference(ct models.ContentType, baseURL string) APIReference {
	apiRef := APIReference{
		ContentType: ct.Name,
		ContentID:   ct.ID,
		Slug:        ct.Slug,
		Description: fmt.Sprintf("API Reference for %s content type", ct.Name),
		BaseURL:     baseURL,
		SEOEnabled:  ct.EnableSEO,
		CreatedAt:   ct.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// Build fields documentation
	allFields := append(ct.Fields, ct.SEOFields...)
	for _, field := range allFields {
		fieldDoc := map[string]interface{}{
			"name":        field.Name,
			"type":        field.Type,
			"required":    field.Required,
			"is_seo":      field.IsSEO,
			"unique":      field.Unique,
			"description": getFieldDescription(field),
		}

		if field.MaxLength != nil {
			fieldDoc["max_length"] = *field.MaxLength
		}
		if field.MinLength != nil {
			fieldDoc["min_length"] = *field.MinLength
		}
		if field.Pattern != "" {
			fieldDoc["pattern"] = field.Pattern
		}
		if field.MinValue != nil {
			fieldDoc["min_value"] = *field.MinValue
		}
		if field.MaxValue != nil {
			fieldDoc["max_value"] = *field.MaxValue
		}
		if field.DefaultValue != "" {
			fieldDoc["default"] = field.DefaultValue
		}
		if field.Placeholder != "" {
			fieldDoc["placeholder"] = field.Placeholder
		}
		if field.HelpText != "" {
			fieldDoc["help_text"] = field.HelpText
		}

		fieldDoc["example"] = generateFieldExample(field)

		apiRef.Fields = append(apiRef.Fields, fieldDoc)
	}

	// Generate endpoints
	apiRef.Endpoints = []APIEndpoint{
		{
			Method:      "POST",
			Path:        fmt.Sprintf("/content/%d/entries", ct.ID),
			Description: fmt.Sprintf("Create a new %s entry", ct.Name),
			Auth:        true,
			Permission:  "ContentEntry:create",
			RequestBody: generateRequestBodyExample(ct),
			Response:    generateResponseExample(ct, "create"),
		},
		{
			Method:      "GET",
			Path:        fmt.Sprintf("/content/%d/entries", ct.ID),
			Description: fmt.Sprintf("List all %s entries", ct.Name),
			Auth:        true,
			Permission:  "ContentEntry:read",
			Parameters: map[string]interface{}{
				"page":       map[string]interface{}{"type": "integer", "default": 1, "description": "Page number"},
				"limit":      map[string]interface{}{"type": "integer", "default": 10, "description": "Items per page"},
				"status":     map[string]interface{}{"type": "string", "enum": []string{"draft", "in_review", "approved", "published"}, "description": "Filter by status"},
				"created_by": map[string]interface{}{"type": "integer", "description": "Filter by creator user ID"},
				"from":       map[string]interface{}{"type": "string", "format": "date", "description": "Filter from date (YYYY-MM-DD)"},
				"to":         map[string]interface{}{"type": "string", "format": "date", "description": "Filter to date (YYYY-MM-DD)"},
			},
			Response: generateResponseExample(ct, "list"),
		},
		{
			Method:      "GET",
			Path:        "/content/entries/{entry_id}",
			Description: fmt.Sprintf("Get a specific %s entry", ct.Name),
			Auth:        true,
			Permission:  "ContentEntry:read",
			Parameters: map[string]interface{}{
				"entry_id": map[string]interface{}{"type": "integer", "required": true, "in": "path"},
			},
			Response: generateResponseExample(ct, "single"),
		},
		{
			Method:      "PUT",
			Path:        "/content/entries/{entry_id}",
			Description: fmt.Sprintf("Update a %s entry", ct.Name),
			Auth:        true,
			Permission:  "ContentEntry:update",
			Parameters: map[string]interface{}{
				"entry_id": map[string]interface{}{"type": "integer", "required": true, "in": "path"},
			},
			RequestBody: generateRequestBodyExample(ct),
			Response:    generateResponseExample(ct, "update"),
		},
		{
			Method:      "DELETE",
			Path:        "/content/entries/{entry_id}",
			Description: fmt.Sprintf("Delete a %s entry (must not be published)", ct.Name),
			Auth:        true,
			Permission:  "ContentEntry:delete",
			Parameters: map[string]interface{}{
				"entry_id": map[string]interface{}{"type": "integer", "required": true, "in": "path"},
			},
			Response: map[string]interface{}{
				"status": 204,
				"body":   nil,
			},
		},
	}

	if ct.EnableSEO {
		apiRef.Endpoints = append(apiRef.Endpoints, APIEndpoint{
			Method:      "GET",
			Path:        "/content/entries/{entry_id}/seo-preview",
			Description: fmt.Sprintf("Get SEO preview for a %s entry", ct.Name),
			Auth:        true,
			Permission:  "SEO:read",
			Parameters: map[string]interface{}{
				"entry_id": map[string]interface{}{"type": "integer", "required": true, "in": "path"},
			},
			Response: generateSEOPreviewExample(ct),
		})
	}

	return apiRef
}

func generateRequestBodyExample(ct models.ContentType) map[string]interface{} {
	example := make(map[string]interface{})

	allFields := append(ct.Fields, ct.SEOFields...)
	for _, field := range allFields {
		example[field.Name] = generateFieldExample(field)
	}

	return example
}

func generateResponseExample(ct models.ContentType, responseType string) map[string]interface{} {
	switch responseType {
	case "create", "update", "single":
		data := make(map[string]interface{})
		allFields := append(ct.Fields, ct.SEOFields...)
		for _, field := range allFields {
			data[field.Name] = generateFieldExample(field)
		}

		return map[string]interface{}{
			"id":              1,
			"content_type_id": ct.ID,
			"data":            data,
			"status":          "draft",
			"created_by":      1,
			"created_at":      "2024-01-15T10:30:00Z",
			"updated_at":      "2024-01-15T10:30:00Z",
		}

	case "list":
		singleExample := generateResponseExample(ct, "single")
		return map[string]interface{}{
			"data":  []interface{}{singleExample},
			"total": 100,
			"page":  1,
			"limit": 10,
		}
	}

	return map[string]interface{}{}
}

func generateSEOPreviewExample(ct models.ContentType) map[string]interface{} {
	example := make(map[string]interface{})

	for _, field := range ct.SEOFields {
		example[field.Name] = generateFieldExample(field)
	}

	return example
}

func generateFieldExample(field models.ContentField) interface{} {
	if field.DefaultValue != "" {
		return field.DefaultValue
	}

	switch field.Type {
	case "string", "text":
		if field.Name == "slug" {
			return "example-slug"
		}
		if strings.Contains(strings.ToLower(field.Name), "title") {
			return fmt.Sprintf("Example %s", field.Name)
		}
		if field.Placeholder != "" {
			return field.Placeholder
		}
		return fmt.Sprintf("example_%s", field.Name)

	case "email":
		return "user@example.com"

	case "url":
		return "https://example.com"

	case "number":
		if field.MinValue != nil {
			return *field.MinValue
		}
		return 100

	case "boolean":
		return true

	case "date":
		return "2024-01-15"

	case "media":
		return map[string]interface{}{
			"url":  "https://cdn.example.com/image.jpg",
			"note": "Send either: 1) file upload in this field, OR 2) {fieldname}_media_id with existing media ID",
		}

	default:
		return fmt.Sprintf("example_%s", field.Name)
	}
}

func getFieldDescription(field models.ContentField) string {
	if field.HelpText != "" {
		return field.HelpText
	}

	descriptions := map[string]string{
		"title":            "The title of the content",
		"slug":             "URL-friendly identifier (lowercase, hyphens only)",
		"description":      "Brief description of the content",
		"content":          "Main content body",
		"meta_title":       "SEO meta title (recommended: 50-60 characters)",
		"meta_description": "SEO meta description (recommended: 150-160 characters)",
		"featured_image":   "Main image for the content",
		"published_at":     "Publication date",
	}

	if desc, exists := descriptions[field.Name]; exists {
		return desc
	}

	return fmt.Sprintf("%s field", strings.Title(field.Name))
}

func GenerateOpenAPISpecHandler(c *fiber.Ctx) error {
	contentTypeID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid content_type_id"})
	}

	var ct models.ContentType
	if err := database.DB.
		Preload("Fields", "is_seo = ?", false).
		Preload("SEOFields", "is_seo = ?", true).
		First(&ct, contentTypeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "content type not found"})
	}

	spec := generateOpenAPISpec(ct, c.BaseURL())
	return c.JSON(spec)
}

func generateOpenAPISpec(ct models.ContentType, baseURL string) map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       fmt.Sprintf("%s API", ct.Name),
			"description": fmt.Sprintf("API documentation for %s content type", ct.Name),
			"version":     "1.0.0",
		},
		"servers": []map[string]string{
			{"url": baseURL, "description": "API Server"},
		},
		"paths": generateOpenAPIPaths(ct),
		"components": map[string]interface{}{
			"schemas": generateOpenAPISchemas(ct),
			"securitySchemes": map[string]interface{}{
				"BearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"security": []map[string]interface{}{
			{"BearerAuth": []string{}},
		},
	}

	return spec
}

func generateOpenAPIPaths(ct models.ContentType) map[string]interface{} {
	return map[string]interface{}{
		fmt.Sprintf("/content/%d/entries", ct.ID): map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     fmt.Sprintf("Create %s entry", ct.Name),
				"operationId": fmt.Sprintf("create%sEntry", ct.Name),
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]string{
								"$ref": fmt.Sprintf("#/components/schemas/%sRequest", ct.Name),
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Success",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]string{
									"$ref": fmt.Sprintf("#/components/schemas/%sResponse", ct.Name),
								},
							},
						},
					},
				},
			},
			"get": map[string]interface{}{
				"summary":     fmt.Sprintf("List %s entries", ct.Name),
				"operationId": fmt.Sprintf("list%sEntries", ct.Name),
				"parameters": []map[string]interface{}{
					{"name": "page", "in": "query", "schema": map[string]string{"type": "integer"}},
					{"name": "limit", "in": "query", "schema": map[string]string{"type": "integer"}},
					{"name": "status", "in": "query", "schema": map[string]string{"type": "string"}},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Success",
					},
				},
			},
		},
	}
}

func generateOpenAPISchemas(ct models.ContentType) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	allFields := append(ct.Fields, ct.SEOFields...)
	for _, field := range allFields {
		fieldSchema := map[string]interface{}{
			"type":        mapFieldTypeToOpenAPI(field.Type),
			"description": getFieldDescription(field),
		}

		if field.MaxLength != nil {
			fieldSchema["maxLength"] = *field.MaxLength
		}
		if field.MinLength != nil {
			fieldSchema["minLength"] = *field.MinLength
		}
		if field.Pattern != "" {
			fieldSchema["pattern"] = field.Pattern
		}
		if field.MinValue != nil {
			fieldSchema["minimum"] = *field.MinValue
		}
		if field.MaxValue != nil {
			fieldSchema["maximum"] = *field.MaxValue
		}
		if field.DefaultValue != "" {
			fieldSchema["default"] = field.DefaultValue
		}
		if field.Placeholder != "" {
			fieldSchema["example"] = field.Placeholder
		}

		properties[field.Name] = fieldSchema

		if field.Required {
			required = append(required, field.Name)
		}
	}

	schemas := map[string]interface{}{
		fmt.Sprintf("%sRequest", ct.Name): map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
		fmt.Sprintf("%sResponse", ct.Name): map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":              map[string]string{"type": "integer"},
				"content_type_id": map[string]string{"type": "integer"},
				"data":            map[string]interface{}{"type": "object", "properties": properties},
				"status":          map[string]string{"type": "string"},
				"created_at":      map[string]string{"type": "string", "format": "date-time"},
				"updated_at":      map[string]string{"type": "string", "format": "date-time"},
			},
		},
	}

	return schemas
}

func mapFieldTypeToOpenAPI(fieldType string) string {
	typeMap := map[string]string{
		"string":  "string",
		"text":    "string",
		"email":   "string",
		"url":     "string",
		"number":  "number",
		"boolean": "boolean",
		"date":    "string",
		"media":   "string",
	}

	if t, exists := typeMap[fieldType]; exists {
		return t
	}
	return "string"
}

func GenerateMarkdownDocsHandler(c *fiber.Ctx) error {
	contentTypeID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid content_type_id"})
	}

	var ct models.ContentType
	if err := database.DB.
		Preload("Fields", "is_seo = ?", false).
		Preload("SEOFields", "is_seo = ?", true).
		First(&ct, contentTypeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "content type not found"})
	}

	markdown := generateMarkdownDocs(ct, c.BaseURL())

	c.Set("Content-Type", "text/markdown")
	return c.SendString(markdown)
}

func generateMarkdownDocs(ct models.ContentType, baseURL string) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# %s API Documentation\n\n", ct.Name))
	md.WriteString(fmt.Sprintf("**Base URL:** `%s`\n\n", baseURL))
	md.WriteString(fmt.Sprintf("**Content Type ID:** `%d`\n\n", ct.ID))
	md.WriteString(fmt.Sprintf("**Slug:** `%s`\n\n", ct.Slug))

	if ct.EnableSEO {
		md.WriteString("**SEO Enabled:** ✓\n\n")
	}

	md.WriteString("---\n\n")

	md.WriteString("## Fields\n\n")
	md.WriteString("| Field | Type | Required | Unique | Validation Rules |\n")
	md.WriteString("|-------|------|----------|--------|------------------|\n")

	allFields := append(ct.Fields, ct.SEOFields...)
	for _, field := range allFields {
		required := "✗"
		if field.Required {
			required = "✓"
		}
		unique := "✗"
		if field.Unique {
			unique = "✓"
		}

		validation := []string{}
		if field.MinLength != nil {
			validation = append(validation, fmt.Sprintf("min: %d", *field.MinLength))
		}
		if field.MaxLength != nil {
			validation = append(validation, fmt.Sprintf("max: %d", *field.MaxLength))
		}
		if field.MinValue != nil {
			validation = append(validation, fmt.Sprintf("min: %.2f", *field.MinValue))
		}
		if field.MaxValue != nil {
			validation = append(validation, fmt.Sprintf("max: %.2f", *field.MaxValue))
		}
		if field.Pattern != "" {
			validation = append(validation, fmt.Sprintf("pattern: `%s`", field.Pattern))
		}

		validationStr := "-"
		if len(validation) > 0 {
			validationStr = strings.Join(validation, ", ")
		}

		fieldType := field.Type
		if field.IsSEO {
			fieldType += " (SEO)"
		}

		md.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
			field.Name, fieldType, required, unique, validationStr))
	}

	md.WriteString("\n")

	md.WriteString("## Endpoints\n\n")

	// CREATE
	md.WriteString("### Create Entry\n\n")
	md.WriteString(fmt.Sprintf("**POST** `/content/%d/entries`\n\n", ct.ID))
	md.WriteString("**Authentication:** Required (Bearer Token)\n\n")
	md.WriteString("**Permission:** `ContentEntry:create`\n\n")
	md.WriteString("**Request Body:**\n\n")
	md.WriteString("```json\n")
	md.WriteString(formatJSON(generateRequestBodyExample(ct)))
	md.WriteString("```\n\n")
	md.WriteString("**Response:**\n\n")
	md.WriteString("```json\n")
	md.WriteString(formatJSON(generateResponseExample(ct, "create")))
	md.WriteString("```\n\n")

	// LIST
	md.WriteString("### List Entries\n\n")
	md.WriteString(fmt.Sprintf("**GET** `/content/%d/entries`\n\n", ct.ID))
	md.WriteString("**Authentication:** Required (Bearer Token)\n\n")
	md.WriteString("**Permission:** `ContentEntry:read`\n\n")
	md.WriteString("**Query Parameters:**\n\n")
	md.WriteString("- `page` (integer, default: 1) - Page number\n")
	md.WriteString("- `limit` (integer, default: 10) - Items per page\n")
	md.WriteString("- `status` (string) - Filter by status (draft, in_review, approved, published)\n")
	md.WriteString("- `created_by` (integer) - Filter by creator user ID\n")
	md.WriteString("- `from` (date) - Filter from date (YYYY-MM-DD)\n")
	md.WriteString("- `to` (date) - Filter to date (YYYY-MM-DD)\n\n")

	// GET ONE
	md.WriteString("### Get Entry\n\n")
	md.WriteString("**GET** `/content/entries/{entry_id}`\n\n")
	md.WriteString("**Authentication:** Required (Bearer Token)\n\n")
	md.WriteString("**Permission:** `ContentEntry:read`\n\n")

	// UPDATE
	md.WriteString("### Update Entry\n\n")
	md.WriteString("**PUT** `/content/entries/{entry_id}`\n\n")
	md.WriteString("**Authentication:** Required (Bearer Token)\n\n")
	md.WriteString("**Permission:** `ContentEntry:update`\n\n")

	// DELETE
	md.WriteString("### Delete Entry\n\n")
	md.WriteString("**DELETE** `/content/entries/{entry_id}`\n\n")
	md.WriteString("**Authentication:** Required (Bearer Token)\n\n")
	md.WriteString("**Permission:** `ContentEntry:delete`\n\n")
	md.WriteString("**Note:** Cannot delete published entries. Unpublish first.\n\n")

	if ct.EnableSEO {
		md.WriteString("### SEO Preview\n\n")
		md.WriteString("**GET** `/content/entries/{entry_id}/seo-preview`\n\n")
		md.WriteString("**Authentication:** Required (Bearer Token)\n\n")
		md.WriteString("**Permission:** `SEO:read`\n\n")
	}

	md.WriteString("---\n\n")
	md.WriteString("## Authentication\n\n")
	md.WriteString("All endpoints require a Bearer token in the Authorization header:\n\n")
	md.WriteString("```\nAuthorization: Bearer <your_jwt_token>\n```\n\n")
	md.WriteString("Obtain a token by authenticating at `/auth/login`\n\n")

	return md.String()
}

func formatJSON(data interface{}) string {
	result := ""
	switch v := data.(type) {
	case map[string]interface{}:
		result = "{\n"
		for key, val := range v {
			result += fmt.Sprintf("  \"%s\": %v,\n", key, formatValue(val))
		}
		result = strings.TrimSuffix(result, ",\n") + "\n}"
	default:
		result = fmt.Sprintf("%v", v)
	}
	return result
}

func formatValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case map[string]interface{}:
		return formatJSON(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
