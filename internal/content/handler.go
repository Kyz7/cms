package content

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

type CreateContentTypeRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AddFieldRequest struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	IsSEO        bool     `json:"is_seo"`
	Unique       bool     `json:"unique"`
	MaxLength    *int     `json:"max_length,omitempty"`
	MinLength    *int     `json:"min_length,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	MinValue     *float64 `json:"min_value,omitempty"`
	MaxValue     *float64 `json:"max_value,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
	Placeholder  string   `json:"placeholder,omitempty"`
	HelpText     string   `json:"help_text,omitempty"`
}

type CreateEntryRequest struct {
	Data map[string]interface{} `json:"data"`
}

type CreateRelationRequest struct {
	ToContentID  uint   `json:"to_content_id"`
	RelationType string `json:"relation_type"`
}

func CreateContentTypeHandler(c *fiber.Ctx) error {
	var body CreateContentTypeRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" || body.Slug == "" {
		return response.ValidationError(c, map[string]string{
			"name": "name is required",
			"slug": "slug is required",
		})
	}

	ct, err := CreateContentType(body.Name, body.Slug)
	if err != nil {
		return response.InternalError(c, "Failed to create content type")
	}

	return response.Created(c, ct, "Content type created successfully")
}

func AddFieldHandler(c *fiber.Ctx) error {
	contentTypeID, err := c.ParamsInt("content_type_id")
	if err != nil {
		return response.BadRequest(c, "Invalid content type ID", nil)
	}

	var body AddFieldRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" || body.Type == "" {
		return response.ValidationError(c, map[string]string{
			"name": "name is required",
			"type": "type is required",
		})
	}

	field, err := AddFieldToContentType(uint(contentTypeID), body.Name, body.Type, body.Required, body.IsSEO)
	if err != nil {
		return response.InternalError(c, "Failed to add field")
	}

	return response.Created(c, field, "Field added successfully")
}

func CreateEntryHandler(c *fiber.Ctx) error {
	contentTypeID, _ := c.ParamsInt("content_type_id")
	userID := c.Locals("user_id").(uint)

	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	allFields := append(ct.Fields, ct.SEOFields...)
	data := make(map[string]interface{})

	contentType := c.Get("Content-Type", "")
	if strings.Contains(contentType, "application/json") {

		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			return response.BadRequest(c, "Invalid JSON payload", err.Error())
		}
		for _, field := range allFields {
			if field.Type == "media" {
				if mediaID, ok := payload[field.Name+"_media_id"].(float64); ok {
					var mediaFile models.MediaFile
					if err := database.DB.First(&mediaFile, uint(mediaID)).Error; err != nil {
						return response.NotFound(c, "Media for field "+field.Name)
					}
					data[field.Name] = mediaFile.URL
					data[field.Name+"_media_id"] = mediaFile.ID
				}
			} else {
				if val, ok := payload[field.Name]; ok {
					data[field.Name] = val
				}
			}
		}
	} else {
		form, err := c.MultipartForm()
		if err != nil {
			return response.BadRequest(c, "Invalid multipart form", err.Error())
		}
		for _, field := range allFields {
			if field.Type == "media" {
				mediaIDStr := c.FormValue(field.Name + "_media_id")

				if mediaIDStr != "" {
					mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
					if err != nil {
						return response.BadRequest(c, "Invalid media ID for field "+field.Name, nil)
					}

					var mediaFile models.MediaFile
					if err := database.DB.First(&mediaFile, uint(mediaID)).Error; err != nil {
						return response.NotFound(c, "Media for field "+field.Name)
					}
					data[field.Name] = mediaFile.URL
					data[field.Name+"_media_id"] = mediaFile.ID
				} else {
					fileHeader, ok := form.File[field.Name]
					if ok && len(fileHeader) > 0 {
						url, err := utils.UploadFile(fileHeader[0])
						if err != nil {
							return response.BadRequest(c, "Failed to upload file", err.Error())
						}
						mediaFile := models.MediaFile{
							FileName:   fileHeader[0].Filename,
							URL:        url,
							Type:       fileHeader[0].Header.Get("Content-Type"),
							UploadedBy: userID,
						}

						if err := database.DB.Create(&mediaFile).Error; err != nil {
							utils.DeleteFile(url)
							return response.InternalError(c, "Failed to save media metadata")
						}

						data[field.Name] = url
						data[field.Name+"_media_id"] = mediaFile.ID
					}
				}
			} else {
				data[field.Name] = c.FormValue(field.Name)
			}
		}
	}

	filteredData, err := middleware.FilterFieldsByPermission(userID, "create", data, uint(contentTypeID))
	if err != nil {
		return response.Forbidden(c, err.Error())
	}

	if len(filteredData) == 0 {
		return response.Forbidden(c, "No permission to create content with provided fields")
	}

	for k, v := range data {
		if strings.HasSuffix(k, "_media_id") {
			filteredData[k] = v
		}
	}

	entry, err := CreateContentEntry(uint(contentTypeID), userID, filteredData)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Created(c, entry, "Entry created successfully")
}

func CreateEntryHandlerJSON(c *fiber.Ctx) error {
	contentTypeID, _ := c.ParamsInt("content_type_id")
	userID := c.Locals("user_id").(uint)

	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "content type not found"})
	}

	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON payload"})
	}

	allFields := append(ct.Fields, ct.SEOFields...)
	data := make(map[string]interface{})

	for _, field := range allFields {
		if field.Type == "media" {
			if mediaID, ok := payload[field.Name+"_media_id"].(float64); ok {
				var mediaFile models.MediaFile
				if err := database.DB.First(&mediaFile, uint(mediaID)).Error; err != nil {
					return c.Status(404).JSON(fiber.Map{"error": "Media not found for field " + field.Name})
				}

				data[field.Name] = mediaFile.URL
				data[field.Name+"_media_id"] = mediaFile.ID
			}
		} else {
			if val, ok := payload[field.Name]; ok {
				data[field.Name] = val
			}
		}
	}

	filteredData, err := middleware.FilterFieldsByPermission(userID, "create", data, uint(contentTypeID))
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	if len(filteredData) == 0 {
		return c.Status(403).JSON(fiber.Map{
			"error": "No permission to create content with the provided fields",
		})
	}
	for k, v := range data {
		if strings.HasSuffix(k, "_media_id") {
			filteredData[k] = v
		}
	}

	entry, err := CreateContentEntry(uint(contentTypeID), userID, filteredData)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(entry)
}

func ListEntriesHandler(c *fiber.Ctx) error {
	contentTypeID, _ := c.ParamsInt("content_type_id")

	query := database.DB.Where("content_type_id = ?", contentTypeID)

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if createdBy := c.Query("created_by"); createdBy != "" {
		query = query.Where("created_by = ?", createdBy)
	}

	if from := c.Query("from"); from != "" {
		query = query.Where("created_at >= ?", from)
	}
	if to := c.Query("to"); to != "" {
		query = query.Where("created_at <= ?", to)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	var entries []models.ContentEntry
	var total int64

	query.Count(&total)
	query.Offset(offset).Limit(limit).Find(&entries)

	meta := response.CalculateMeta(page, limit, total)
	return response.SuccessWithMeta(c, entries, meta, "Entries retrieved successfully")
}

func CreateRelationHandler(c *fiber.Ctx) error {
	fromID, err := c.ParamsInt("from_content_id")
	if err != nil {
		return response.BadRequest(c, "Invalid from_content_id", nil)
	}

	var body CreateRelationRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.ToContentID == 0 {
		return response.ValidationError(c, map[string]string{
			"to_content_id": "to_content_id is required",
		})
	}

	if body.RelationType == "" {
		return response.ValidationError(c, map[string]string{
			"relation_type": "relation_type is required",
		})
	}

	relation, err := CreateContentRelation(uint(fromID), body.ToContentID, body.RelationType)
	if err != nil {
		return response.InternalError(c, "Failed to create relation")
	}

	return response.Created(c, relation, "Relation created successfully")
}

func ListRelationsHandler(c *fiber.Ctx) error {
	fromID, err := c.ParamsInt("from_content_id")
	if err != nil {
		return response.BadRequest(c, "Invalid from_content_id", nil)
	}

	relations, err := ListContentRelations(uint(fromID))
	if err != nil {
		return response.InternalError(c, "Failed to fetch relations")
	}

	return response.Success(c, relations, "Relations retrieved successfully")
}

func SEOPreviewHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	seoData, err := GenerateSEOPreview(uint(entryID))
	if err != nil {
		return response.InternalError(c, "Failed to generate SEO preview")
	}

	return response.Success(c, seoData, "SEO preview generated successfully")
}

func UpdateEntryHandler(c *fiber.Ctx) error {
	entryID, _ := c.ParamsInt("entry_id")
	userID := c.Locals("user_id").(uint)

	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return response.NotFound(c, "Entry")
	}

	if entry.Status == models.StatusPublished {
		return response.Conflict(c, "Cannot edit published content. Please unpublish first")
	}

	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, entry.ContentTypeID).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	allFields := append(ct.Fields, ct.SEOFields...)
	data := make(map[string]interface{})

	if form, err := c.MultipartForm(); err == nil {
		for _, field := range allFields {
			if field.Type == "media" {
				mediaIDStr := c.FormValue(field.Name + "_media_id")

				if mediaIDStr != "" {
					mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
					if err != nil {
						return response.BadRequest(c, "Invalid media ID for field "+field.Name, nil)
					}

					var mediaFile models.MediaFile
					if err := database.DB.First(&mediaFile, uint(mediaID)).Error; err != nil {
						return response.NotFound(c, "Media for field "+field.Name)
					}

					data[field.Name] = mediaFile.URL
					data[field.Name+"_media_id"] = mediaFile.ID
				} else if fileHeaders, ok := form.File[field.Name]; ok && len(fileHeaders) > 0 {
					url, err := utils.UploadFile(fileHeaders[0])
					if err != nil {
						return response.BadRequest(c, "Failed to upload file", err.Error())
					}

					mediaFile := models.MediaFile{
						FileName:   fileHeaders[0].Filename,
						URL:        url,
						Type:       fileHeaders[0].Header.Get("Content-Type"),
						UploadedBy: userID,
					}

					if err := database.DB.Create(&mediaFile).Error; err != nil {
						utils.DeleteFile(url)
						return response.InternalError(c, "Failed to save media metadata")
					}

					data[field.Name] = url
					data[field.Name+"_media_id"] = mediaFile.ID
				}
			} else {
				if _, exists := form.Value[field.Name]; exists {
					data[field.Name] = c.FormValue(field.Name)
				}
			}
		}
	} else {
		if err := c.BodyParser(&data); err != nil {
			return response.BadRequest(c, "Invalid request body", err.Error())
		}
	}

	if len(data) == 0 {
		return response.BadRequest(c, "No data provided for update", nil)
	}

	filteredData, err := middleware.FilterFieldsByPermission(userID, "update", data, entry.ContentTypeID)
	if err != nil {
		return response.Forbidden(c, err.Error())
	}

	if len(filteredData) == 0 {
		return response.Forbidden(c, "No permission to edit provided fields")
	}

	for k, v := range data {
		if strings.HasSuffix(k, "_media_id") {
			filteredData[k] = v
		}
	}

	if err := ValidatePartialUpdate(ct, filteredData, uint(entryID)); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	var existingData map[string]interface{}
	json.Unmarshal([]byte(entry.Data), &existingData)

	for k, v := range filteredData {
		existingData[k] = v
	}

	jsonData, err := json.Marshal(existingData)
	if err != nil {
		return response.InternalError(c, "Failed to serialize data")
	}

	entry.Data = datatypes.JSON(jsonData)
	entry.Status = models.StatusDraft
	entry.UpdatedBy = userID

	if err := database.DB.Save(&entry).Error; err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	database.DB.Preload("Creator").Preload("Updater").First(&entry, entryID)

	return response.Success(c, entry, "Entry updated successfully")
}

func ListContentTypesHandler(c *fiber.Ctx) error {
	var cts []models.ContentType
	if err := database.DB.
		Preload("Fields", "is_seo = ?", false).
		Preload("SEOFields", "is_seo = ?", true).
		Find(&cts).Error; err != nil {
		return response.InternalError(c, "Failed to fetch content types")
	}

	return response.Success(c, cts, "Content types retrieved successfully")
}

func GetContentTypeHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid content type ID", nil)
	}

	var ct models.ContentType
	if err := database.DB.
		Preload("Fields", "is_seo = ?", false).
		Preload("SEOFields", "is_seo = ?", true).
		First(&ct, id).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	return response.Success(c, ct, "Content type retrieved successfully")
}

func GetEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	var entry models.ContentEntry
	if err := database.DB.
		Preload("Creator").
		Preload("Updater").
		First(&entry, entryID).Error; err != nil {
		return response.NotFound(c, "Entry")
	}

	return response.Success(c, entry, "Entry retrieved successfully")
}

func DeleteEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return response.NotFound(c, "Entry")
	}

	if entry.Status == models.StatusPublished {
		return response.Conflict(c, "Cannot delete published content. Please unpublish first")
	}

	if err := database.DB.Delete(&entry).Error; err != nil {
		return response.InternalError(c, "Failed to delete entry")
	}

	return response.NoContent(c)
}

func UpdateContentTypeHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid content type ID", nil)
	}

	var body struct {
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		EnableSEO bool   `json:"enable_seo"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	var ct models.ContentType
	if err := database.DB.First(&ct, id).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	ct.Name = body.Name
	ct.Slug = body.Slug
	ct.EnableSEO = body.EnableSEO

	if err := database.DB.Save(&ct).Error; err != nil {
		return response.InternalError(c, "Failed to update content type")
	}

	return response.Success(c, ct, "Content type updated successfully")
}

func DeleteContentTypeHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid content type ID", nil)
	}

	var entryCount int64
	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ?", id).
		Count(&entryCount)

	if entryCount > 0 {
		return response.Conflict(c, "Cannot delete content type with existing entries")
	}

	var ct models.ContentType
	if err := database.DB.First(&ct, id).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	database.DB.Where("content_type_id = ?", id).Delete(&models.ContentField{})

	if err := database.DB.Delete(&ct).Error; err != nil {
		return response.InternalError(c, "Failed to delete content type")
	}

	return response.NoContent(c)
}

func UpdateFieldHandler(c *fiber.Ctx) error {
	fieldID, err := c.ParamsInt("field_id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid field_id"})
	}

	var body AddFieldRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	var field models.ContentField
	if err := database.DB.First(&field, fieldID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "field not found"})
	}

	field.Name = body.Name
	field.Type = body.Type
	field.Required = body.Required
	field.IsSEO = body.IsSEO
	field.Unique = body.Unique
	field.MaxLength = body.MaxLength
	field.MinLength = body.MinLength
	field.Pattern = body.Pattern
	field.MinValue = body.MinValue
	field.MaxValue = body.MaxValue
	field.DefaultValue = body.DefaultValue
	field.Placeholder = body.Placeholder
	field.HelpText = body.HelpText

	if err := database.DB.Save(&field).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":          "field updated successfully",
		"field":            field,
		"validation_rules": GetFieldValidationRules(field),
	})
}

func DeleteRelationHandler(c *fiber.Ctx) error {
	relationID, err := c.ParamsInt("relation_id")
	if err != nil {
		return response.BadRequest(c, "Invalid relation ID", nil)
	}

	var relation models.ContentRelation
	if err := database.DB.First(&relation, relationID).Error; err != nil {
		return response.NotFound(c, "Relation")
	}

	if err := database.DB.Delete(&relation).Error; err != nil {
		return response.InternalError(c, "Failed to delete relation")
	}

	return response.NoContent(c)
}

func DeleteFieldHandler(c *fiber.Ctx) error {
	fieldID, err := c.ParamsInt("field_id")
	if err != nil {
		return response.BadRequest(c, "Invalid field ID", nil)
	}

	var field models.ContentField
	if err := database.DB.First(&field, fieldID).Error; err != nil {
		return response.NotFound(c, "Field")
	}

	if err := database.DB.Delete(&field).Error; err != nil {
		return response.InternalError(c, "Failed to delete field")
	}

	return response.NoContent(c)
}

func GetFieldValidationHandler(c *fiber.Ctx) error {
	fieldID, err := c.ParamsInt("field_id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid field_id"})
	}

	var field models.ContentField
	if err := database.DB.First(&field, fieldID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "field not found"})
	}

	return c.JSON(GetFieldValidationRules(field))
}
