package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/globals"
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/project"
	"github.com/Kyz7/cms/internal/response"
	"github.com/Kyz7/cms/internal/translation"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CreateContentTypeRequest struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ProjectID *uint  `json:"project_id,omitempty"`
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

type TranslateEntryRequest struct {
	TargetLang string   `json:"target_lang"`
	SourceLang string   `json:"source_lang"`
	Fields     []string `json:"fields"`
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

	userID := c.Locals("user_id").(uint)

	ownerID, ok := globals.GetRoleIDByName(models.ProjectRoleOwner)
	if !ok {
		return response.InternalError(c, "Project owner role not found")
	}

	adminID, ok := globals.GetRoleIDByName(models.ProjectRoleAdmin)
	if !ok {
		return response.InternalError(c, "Project admin role not found")
	}

	editorID, ok := globals.GetRoleIDByName(models.ProjectRoleEditor)
	if !ok {
		return response.InternalError(c, "Project editor role not found")
	}

	// If project_id is provided, verify user is a member
	if body.ProjectID != nil {
		projectMember, err := project.GetProjectMember(*body.ProjectID, userID)
		if err != nil {
			return response.Forbidden(c, "You are not a member of this project")
		}
		// Only owner, admin, and editor can create content types
		if projectMember.RoleID != ownerID && projectMember.RoleID != adminID && projectMember.RoleID != editorID {
			return response.Forbidden(c, "Only project owners can create content types")
		}

	}

	ct, err := CreateContentType(body.Name, body.Slug, body.ProjectID)
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
	contentTypeID, err := c.ParamsInt("content_type_id")
	if err != nil || contentTypeID <= 0 {
		return response.BadRequest(c, "Invalid Content Type ID", nil)
	}

	userID := c.Locals("user_id").(uint)
	ctID := uint(contentTypeID)

	// 1. Ambil Content Type dan Field Terkait
	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, ctID).Error; err != nil {
		return response.NotFound(c, "Content type not found")
	}

	allFields := append(ct.Fields, ct.SEOFields...)
	data := make(map[string]interface{})

	// 2. Parse Payload (JSON atau Multipart Form)
	contentType := c.Get("Content-Type", "")

	if strings.Contains(contentType, "application/json") {
		// --- Parsing JSON ---
		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			return response.BadRequest(c, "Invalid JSON payload", err.Error())
		}
		data = mapJSONToData(payload, allFields, ctID)

	} else if strings.Contains(contentType, "multipart/form-data") {
		// --- Parsing Multipart Form ---
		form, err := c.MultipartForm()
		if err != nil {
			return response.BadRequest(c, "Invalid multipart form", err.Error())
		}

		var errParse error
		data, errParse = mapFormToData(c, form, allFields, userID)
		if errParse != nil {
			return response.BadRequest(c, errParse.Error(), nil)
		}

	} else {
		return response.BadRequest(c, "Unsupported Content-Type", nil)
	}

	// 3. Tentukan ProjectID untuk Otorisasi Filter dan Penyimpanan
	// Variabel ini akan digunakan untuk FilterFieldsByPermission (uint) dan CreateContentEntry (*uint)
	var projectID uint = 0
	var projectIDPtr *uint

	if ct.ProjectID != nil && *ct.ProjectID > 0 {
		projectID = *ct.ProjectID
		projectIDPtr = ct.ProjectID // Pointer untuk CreateContentEntry
	}

	// 4. Filter Field Berdasarkan Izin (Field-Level Permission)
	// ⭐️ PERBAIKAN KRITIS: Menambahkan projectID sebagai argumen kelima (uint)
	filteredData, err := middleware.FilterFieldsByPermission(userID, "create", data, ctID, projectID)

	if err != nil {
		// Error terjadi selama filter (misalnya, required field tidak diizinkan atau izin tidak ditemukan)
		return response.Forbidden(c, err.Error())
	}

	// Periksa apakah ada data yang tersisa untuk disimpan
	if len(filteredData) == 0 {
		return response.Forbidden(c, "You do not have permission to write to any allowed fields, or no data was provided.")
	}

	// 5. Buat Entry Konten
	// Menggunakan projectIDPtr (*uint) yang sesuai untuk fungsi CreateContentEntry
	entry, err := CreateContentEntry(ctID, userID, filteredData, projectIDPtr)
	if err != nil {
		// Error mungkin datang dari validasi data atau DB
		return response.BadRequest(c, fmt.Sprintf("Failed to create entry: %s", err.Error()), nil)
	}

	return response.Created(c, entry, "Entry created successfully")
}

func mapJSONToData(payload map[string]interface{}, allFields []models.ContentField, ctID uint) map[string]interface{} {
	data := make(map[string]interface{})
	for _, field := range allFields {
		if field.Type == "media" {
			// Media ID harus dikirim sebagai media_id: X.X
			if mediaID, ok := payload[field.Name+"_media_id"].(float64); ok {
				data[field.Name+"_media_id"] = uint(mediaID)
			}
		} else {
			if val, ok := payload[field.Name]; ok {
				// Tambahkan field yang ada ke data
				data[field.Name] = val
			}
		}
	}
	return data
}

func mapFormToData(c *fiber.Ctx, form *multipart.Form, allFields []models.ContentField, userID uint) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	for _, field := range allFields {
		if field.Type == "media" {
			mediaIDStr := c.FormValue(field.Name + "_media_id")
			fileHeader, fileExists := form.File[field.Name]

			if mediaIDStr != "" {
				// Case 1: Media ID sudah ada (existing file)
				mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid media ID format for field %s", field.Name)
				}
				data[field.Name+"_media_id"] = uint(mediaID)

			} else if fileExists && len(fileHeader) > 0 {
				// Case 2: Upload file baru
				url, err := utils.UploadFile(fileHeader[0])
				if err != nil {
					return nil, fmt.Errorf("failed to upload file for field %s: %w", field.Name, err)
				}

				// Simpan metadata ke MediaFile
				mediaFile := models.MediaFile{
					FileName:   fileHeader[0].Filename,
					URL:        url,
					Type:       fileHeader[0].Header.Get("Content-Type"),
					UploadedBy: userID,
				}

				if err := database.DB.Create(&mediaFile).Error; err != nil {
					utils.DeleteFile(url) // Clean up file jika DB gagal
					return nil, errors.New("failed to save media metadata")
				}

				data[field.Name+"_media_id"] = mediaFile.ID
			}
			// Jika tidak ada media ID dan tidak ada file, lewati field ini.

		} else {
			// Case 3: Field standar
			if val := c.FormValue(field.Name); val != "" {
				data[field.Name] = val
			}
		}
	}
	return data, nil
}

func CreateEntryHandlerJSON(c *fiber.Ctx) error {
	contentTypeIDInt, _ := c.ParamsInt("content_type_id")
	contentTypeID := uint(contentTypeIDInt)
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

	// ⭐️ Tentukan ProjectID untuk Filter
	var projectID uint = 0
	var projectIDPtr *uint

	if ct.ProjectID != nil && *ct.ProjectID > 0 {
		projectID = *ct.ProjectID
		projectIDPtr = ct.ProjectID
	}

	// ⭐️ PERBAIKAN PANGGILAN: Menambahkan projectID sebagai argumen kelima
	filteredData, err := middleware.FilterFieldsByPermission(userID, "create", data, contentTypeID, projectID)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	if len(filteredData) == 0 {
		return c.Status(403).JSON(fiber.Map{ // Mengubah 400 menjadi 403 karena ini masalah izin field
			"error": "You do not have permission to write to any allowed fields, or no data was provided.",
		})
	}

	// Pastikan media_id dari data awal ditambahkan kembali ke filteredData jika fieldnya lolos filter
	for k, v := range data {
		if strings.HasSuffix(k, "_media_id") {
			if _, ok := filteredData[strings.TrimSuffix(k, "_media_id")]; ok {
				filteredData[k] = v
			}
		}
	}

	// ⭐️ Hapus semua logika 'Get project_id from query' dan 'Verify user is a member'
	// Logika ini sudah ditangani oleh middleware.PermissionProtected sebelumnya.

	entry, err := CreateContentEntry(contentTypeID, userID, filteredData, projectIDPtr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(entry)
}

func ListEntriesHandler(c *fiber.Ctx) error {
	contentTypeID, _ := c.ParamsInt("content_type_id")
	userID := c.Locals("user_id").(uint)

	// Get content type to check if it belongs to a project
	var ct models.ContentType
	if err := database.DB.First(&ct, contentTypeID).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	query := database.DB.Model(&models.ContentEntry{}).Where("content_type_id = ?", contentTypeID)

	// Filter by project if content type belongs to a project
	if ct.ProjectID != nil {
		// Verify user is a member of the project
		if _, err := project.GetProjectMember(*ct.ProjectID, userID); err != nil {
			return response.Forbidden(c, "You are not a member of this project")
		}
		query = query.Where("project_id = ?", *ct.ProjectID)
	} else {
		// For global content types, only show global entries (project_id IS NULL)
		query = query.Where("project_id IS NULL")
	}

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

	// debug logging removed

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
	if err != nil || entryID <= 0 {
		return response.BadRequest(c, "Invalid or missing Content Entry ID", nil)
	}

	seoData, err := GenerateSEOPreview(uint(entryID))

	if err != nil {
		// Tangani error spesifik dari service layer
		if fiberErr, ok := err.(*fiber.Error); ok {
			return fiberErr // Kembalikan error Fiber yang sudah dibuat (misal 404 Not Found)
		}
		// Tangani error internal database atau unmarshal
		return response.InternalError(c, fmt.Sprintf("Failed to generate SEO preview: %s", err.Error()))
	}

	if len(seoData) == 0 {
		return response.NotFound(c, "No SEO data or related fields found for this content entry.")
	}

	return response.Success(c, seoData, "SEO preview generated successfully")
}

// GeneratePreviewTokenHandler issues a signed preview token and URL for live preview.
// Requires authenticated user with ContentEntry:read permission.
func GeneratePreviewTokenHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil || entryID <= 0 {
		return response.BadRequest(c, "Invalid or missing Content Entry ID", nil)
	}

	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "Entry")
		}
		return response.InternalError(c, "Failed to fetch entry")
	}

	token, expiresAt, err := GenerateEntryPreviewToken(entry)
	if err != nil {
		return response.InternalError(c, fmt.Sprintf("Failed to generate preview token: %s", err.Error()))
	}

	backendURL := BuildPreviewURL(c.BaseURL(), entry.ID, token)
	frontendURL := os.Getenv("PREVIEW_FRONTEND_URL")
	if frontendURL != "" {
		frontendURL = BuildPreviewURL(frontendURL, entry.ID, token)
	}

	return response.Success(c, fiber.Map{
		"token":                token,
		"expires_at":           expiresAt.Format(time.RFC3339),
		"preview_url":          backendURL,
		"frontend_preview_url": frontendURL,
	}, "Preview token generated")
}

// PreviewEntryHandler returns full content entry using preview token (no Bearer auth required).
func PreviewEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil || entryID <= 0 {
		return response.BadRequest(c, "Invalid or missing Content Entry ID", nil)
	}

	token := c.Query("token")
	if token == "" {
		token = c.Get("X-Preview-Token")
	}
	if token == "" {
		return response.BadRequest(c, "Preview token is required", nil)
	}

	claims, err := ParsePreviewToken(token)
	if err != nil {
		return response.Unauthorized(c, fmt.Sprintf("Invalid preview token: %s", err.Error()))
	}

	if claims.EntryID != uint(entryID) {
		return response.Unauthorized(c, "Preview token does not match entry")
	}

	var entry models.ContentEntry
	if err := database.DB.Preload("ContentType").First(&entry, entryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "Entry")
		}
		return response.InternalError(c, "Failed to fetch entry")
	}

	return response.Success(c, entry, "Preview entry retrieved")
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

	// --- 1. Parsing Payload (Multipart atau JSON) ---

	// Coba parsing Multipart Form
	if form, err := c.MultipartForm(); err == nil {
		// Logika parsing form dan upload file (Sudah Anda sediakan dan biarkan)
		for _, field := range allFields {
			if field.Type == "media" {
				// ... (Logika penanganan media ID dan upload file) ...
				mediaIDStr := c.FormValue(field.Name + "_media_id")

				if mediaIDStr != "" {
					// Logic to handle existing media ID
					mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
					if err != nil {
						return response.BadRequest(c, "Invalid media ID for field "+field.Name, nil)
					}
					// Cek ketersediaan mediaFile
					var mediaFile models.MediaFile
					if err := database.DB.First(&mediaFile, uint(mediaID)).Error; err != nil {
						return response.NotFound(c, "Media for field "+field.Name)
					}
					data[field.Name] = mediaFile.URL
					data[field.Name+"_media_id"] = mediaFile.ID
				} else if fileHeaders, ok := form.File[field.Name]; ok && len(fileHeaders) > 0 {
					// Logic to handle new file upload
					url, err := utils.UploadFile(fileHeaders[0])
					if err != nil {
						return response.BadRequest(c, "Failed to upload file", err.Error())
					}
					// Simpan metadata
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
		// Fallback ke BodyParser jika bukan Multipart (misalnya JSON)
		if err := c.BodyParser(&data); err != nil {
			return response.BadRequest(c, "Invalid request body", err.Error())
		}
	}

	if len(data) == 0 {
		return response.BadRequest(c, "No data provided for update", nil)
	}

	// --- 2. Tentukan ProjectID untuk Filter ---
	var projectID uint = 0
	if ct.ProjectID != nil && *ct.ProjectID > 0 {
		projectID = *ct.ProjectID
	}

	// ⭐️ PERBAIKAN PANGGILAN: Menambahkan projectID sebagai argumen kelima
	filteredData, err := middleware.FilterFieldsByPermission(userID, "update", data, entry.ContentTypeID, projectID)
	if err != nil {
		return response.Forbidden(c, err.Error())
	}

	if len(filteredData) == 0 {
		return response.BadRequest(c, "No valid fields to update. All provided fields were filtered out by permissions or don't exist", nil)
	}

	// --- 3. Tambahkan kembali media_id yang lolos filter ---
	// Logika ini dipindahkan ke sini untuk memastikan media_id ditambahkan SETELAH filter sukses
	for k, v := range data {
		if strings.HasSuffix(k, "_media_id") {
			if _, ok := filteredData[strings.TrimSuffix(k, "_media_id")]; ok {
				filteredData[k] = v
			}
		}
	}

	// --- 4. Validasi dan Simpan ---

	// Asumsi ValidatePartialUpdate sudah didefinisikan
	if err := ValidatePartialUpdate(ct, filteredData, uint(entryID)); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	var existingData map[string]interface{}
	// Masukkan existingData ke map
	json.Unmarshal([]byte(entry.Data), &existingData)

	// Gabungkan data yang difilter ke existingData
	for k, v := range filteredData {
		existingData[k] = v
	}

	jsonData, err := json.Marshal(existingData)
	if err != nil {
		return response.InternalError(c, "Failed to serialize data")
	}

	entry.Data = datatypes.JSON(jsonData)
	entry.UpdatedBy = userID

	if err := database.DB.Save(&entry).Error; err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	database.DB.Preload("Creator").Preload("Updater").First(&entry, entryID)

	return response.Success(c, entry, "Entry updated successfully")
}

func ListContentTypesHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var cts []models.ContentType
	query := database.DB

	// Filter by project if project_id is provided
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if pid, err := strconv.ParseUint(projectIDStr, 10, 32); err == nil {
			projectID := uint(pid)
			// Verify user is a member
			if _, err := project.GetProjectMember(projectID, userID); err != nil {
				return response.Forbidden(c, "You are not a member of this project")
			}
			query = query.Where("project_id = ?", projectID)
		}
	} else {
		// If no project_id, show only global content types (project_id IS NULL)
		// and project content types where user is a member
		query = query.Where("project_id IS NULL OR project_id IN (SELECT project_id FROM project_members WHERE user_id = ? AND status = 'active' AND deleted_at IS NULL)", userID)
	}

	if err := query.
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

func TranslateEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body TranslateEntryRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}
	if body.TargetLang == "" {
		al := c.Get("Accept-Language")
		inferred := inferLangFromAcceptLanguage(al)
		if inferred == "" {
			return response.ValidationError(c, map[string]string{"target_lang": "target_lang is required or inferable from Accept-Language"})
		}
		body.TargetLang = inferred
	}

	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return response.NotFound(c, "Entry")
	}

	data := map[string]interface{}{}
	if err := json.Unmarshal([]byte(entry.Data), &data); err != nil {
		return response.InternalError(c, "Failed to parse entry data")
	}

	includeAll := len(body.Fields) == 0
	shouldInclude := func(k string) bool {
		if includeAll {
			return true
		}
		for _, f := range body.Fields {
			if f == k {
				return true
			}
		}
		return false
	}

	texts := []string{}
	keys := []string{}
	for k, v := range data {
		if !shouldInclude(k) {
			continue
		}
		if s, ok := v.(string); ok && s != "" {
			texts = append(texts, s)
			keys = append(keys, k)
		}
	}

	client, err := translation.NewClient()
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	translated, err := client.TranslateTexts(texts, body.TargetLang, body.SourceLang)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	// put into _i18n[targetLang]
	i18n, _ := data["_i18n"].(map[string]interface{})
	if i18n == nil {
		i18n = map[string]interface{}{}
	}
	langMap, _ := i18n[body.TargetLang].(map[string]interface{})
	if langMap == nil {
		langMap = map[string]interface{}{}
	}
	for i, key := range keys {
		if i < len(translated) {
			langMap[key] = translated[i]
		}
	}
	i18n[body.TargetLang] = langMap
	data["_i18n"] = i18n

	buf, err := json.Marshal(data)
	if err != nil {
		return response.InternalError(c, "Failed to serialize data")
	}
	entry.Data = datatypes.JSON(buf)
	entry.UpdatedBy = userID
	if err := database.DB.Save(&entry).Error; err != nil {
		return response.InternalError(c, "Failed to save translation")
	}

	return response.Success(c, entry, "Entry translated successfully")
}

// inferLangFromAcceptLanguage returns the first language tag (lowercased, 2-letter when possible)
func inferLangFromAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return ""
	}
	// e.g., "en-US;q=0.9" -> "en"
	primary := strings.TrimSpace(parts[0])
	if idx := strings.Index(primary, ";"); idx >= 0 {
		primary = primary[:idx]
	}
	primary = strings.ToLower(primary)
	if dash := strings.Index(primary, "-"); dash > 0 {
		return primary[:dash]
	}
	return primary
}

func GetEntriesSemuaHandler(c *fiber.Ctx) error {
	// Ambil User ID dari Fiber Locals
	userID := c.Locals("user_id").(uint)

	// Ambil ContentTypeID dari query parameter untuk filtering opsional
	contentTypeIDStr := c.Query("content_type_id")
	var contentTypeID uint = 0

	if contentTypeIDStr != "" {
		if id, err := strconv.ParseUint(contentTypeIDStr, 10, 32); err == nil {
			contentTypeID = uint(id)
		} else {
			return response.BadRequest(c, "Invalid content_type_id in query parameter", nil)
		}
	}

	// Panggil service layer
	entries, err := GetAccessibleEntries(userID, contentTypeID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch accessible entries")
	}

	// Kembalikan 200 OK dengan list entries
	return response.Success(c, entries, "Entries retrieved successfully")
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
	// 1. Ambil Parameter dari URL
	// Gunakan := untuk variabel baru agar tidak menimpa error dari c.ParamsInt
	fieldID, err := c.ParamsInt("field_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid field ID provided."})
	}

	contentTypeID, err := c.ParamsInt("content_type_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid content type ID provided."})
	}

	// 2. Binding Request Body
	var body AddFieldRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("Invalid request body: %v", err)})
	}

	// ⭐️ 3. VALIDASI INPUT (Minimal)
	if body.Name == "" || body.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Field name and type are required."})
	}

	// 4. Cari Field yang Akan Diupdate
	var field models.ContentField

	// ⭐️ PENTING: Pastikan Field ID milik ContentType ID yang benar
	if err := database.DB.Where("content_type_id = ?", contentTypeID).First(&field, fieldID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Mengembalikan 404 jika field tidak ditemukan ATAU tidak termasuk ContentType ID yang diberikan
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Field not found or does not belong to the specified content type."})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("Database error: %v", err)})
	}

	// 5. Update Properti Field (Menggunakan Struct untuk kebersihan)

	// Perhatikan: Kita tidak mengizinkan perubahan ContentTypeID setelah field dibuat

	field.Name = body.Name
	field.Type = body.Type
	field.Required = body.Required
	field.IsSEO = body.IsSEO

	// Update Validation Rules
	field.Unique = body.Unique
	field.MaxLength = body.MaxLength
	field.MinLength = body.MinLength
	field.Pattern = body.Pattern
	field.MinValue = body.MinValue
	field.MaxValue = body.MaxValue
	field.DefaultValue = body.DefaultValue
	field.Placeholder = body.Placeholder
	field.HelpText = body.HelpText

	// 6. Simpan Perubahan
	if err := database.DB.Save(&field).Error; err != nil {
		// Cek jika error GORM adalah violation (misalnya, Name/Unique Constraint)
		// Ini sering terjadi jika Anda mencoba mengganti nama field menjadi yang sudah ada.
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "unique constraint") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "A field with this name already exists in this content type."})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("Failed to save field changes: %v", err)})
	}

	// 7. Respon Sukses
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Field updated successfully.",
		"field":   field,
		// Asumsi GetFieldValidationRules ada dan mengembalikan aturan yang dapat dibaca
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
