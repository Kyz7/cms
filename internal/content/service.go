package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/Kyz7/cms/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func CreateContentType(name, slug string, projectID *uint) (*models.ContentType, error) {
	ct := models.ContentType{
		Name:      name,
		Slug:      slug,
		ProjectID: projectID,
	}
	if err := database.DB.Create(&ct).Error; err != nil {
		return nil, err
	}
	return &ct, nil
}

func AddFieldToContentType(contentTypeID uint, name, fieldType string, required bool, isSEO bool) (*models.ContentField, error) {
	field := models.ContentField{
		ContentTypeID: contentTypeID,
		Name:          name,
		Type:          fieldType,
		Required:      required,
		IsSEO:         isSEO,
	}

	if err := database.DB.Create(&field).Error; err != nil {
		return nil, err
	}

	if isSEO {
		database.DB.Model(&models.ContentType{}).
			Where("id = ?", contentTypeID).
			Update("enable_seo", true)
	}

	return &field, nil
}

func CreateContentRelation(fromID, toID uint, relationType string, projectID *uint) (*models.ContentRelation, error) {
	// Validate that both entries belong to the same project (if projectID is set)
	if projectID != nil {
		var fromEntry, toEntry models.ContentEntry
		if err := database.DB.First(&fromEntry, fromID).Error; err != nil {
			return nil, fmt.Errorf("source entry not found")
		}
		if err := database.DB.First(&toEntry, toID).Error; err != nil {
			return nil, fmt.Errorf("target entry not found")
		}

		// Allow linking to global entries (ProjectID is nil) or same project entries
		if fromEntry.ProjectID != nil && *fromEntry.ProjectID != *projectID {
			return nil, fmt.Errorf("source entry does not belong to the current project")
		}
		if toEntry.ProjectID != nil && *toEntry.ProjectID != *projectID {
			return nil, fmt.Errorf("target entry does not belong to the current project")
		}
	}

	relation := models.ContentRelation{
		FromContentID: fromID,
		ToContentID:   toID,
		RelationType:  relationType,
		ProjectID:     projectID,
	}
	if err := database.DB.Create(&relation).Error; err != nil {
		return nil, err
	}
	return &relation, nil
}

func ListContentRelations(fromID uint, projectID *uint) ([]models.ContentRelation, error) {
	var relations []models.ContentRelation
	query := database.DB.Where("from_content_id = ?", fromID)

	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}

	if err := query.Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

func ValidateContentEntryEnhanced(ct models.ContentType, data map[string]interface{}) error {
	allFields := append(ct.Fields, ct.SEOFields...)

	for _, field := range allFields {
		value, exists := data[field.Name]

		if !exists || value == nil || value == "" {
			if field.Required {
				return fmt.Errorf("field '%s' is required", field.Name)
			}
			if field.DefaultValue != "" {
				data[field.Name] = field.DefaultValue
			}
			continue
		}

		switch field.Type {
		case "string", "text":
			if err := validateString(field, value); err != nil {
				return err
			}

		case "email":
			if err := validateEmail(field, value); err != nil {
				return err
			}

		case "url":
			if err := validateURL(field, value); err != nil {
				return err
			}

		case "number":
			if err := validateNumber(field, value); err != nil {
				return err
			}

		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("field '%s' must be boolean", field.Name)
			}

		case "date":
			if err := validateDate(field, value); err != nil {
				return err
			}

		case "media":
			if err := validateMedia(field, value); err != nil {
				return err
			}
		}

		if field.Unique {
			if err := checkUniqueness(ct.ID, field.Name, value, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateString(field models.ContentField, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be string", field.Name)
	}

	if field.MinLength != nil && len(strVal) < *field.MinLength {
		return fmt.Errorf("field '%s' must be at least %d characters", field.Name, *field.MinLength)
	}

	if field.MaxLength != nil && len(strVal) > *field.MaxLength {
		return fmt.Errorf("field '%s' must not exceed %d characters", field.Name, *field.MaxLength)
	}

	// Custom pattern validation Exmp : "^[a-z0-9]+(?:-[a-z0-9]+)*$"
	if field.Pattern != "" {
		matched, err := regexp.MatchString(field.Pattern, strVal)
		if err != nil {
			return fmt.Errorf("invalid pattern for field '%s'", field.Name)
		}
		if !matched {
			return fmt.Errorf("field '%s' does not match required pattern", field.Name)
		}
	}

	if field.Name == "slug" {
		slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
		if !slugRegex.MatchString(strVal) {
			return fmt.Errorf("field '%s' must be a valid slug (lowercase, numbers, hyphens only)", field.Name)
		}
	}

	return nil
}

func validateEmail(field models.ContentField, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be string", field.Name)
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(strVal) {
		return fmt.Errorf("field '%s' must be a valid email address", field.Name)
	}

	return nil
}

func validateURL(field models.ContentField, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be string", field.Name)
	}

	if _, err := url.ParseRequestURI(strVal); err != nil {
		return fmt.Errorf("field '%s' must be a valid URL", field.Name)
	}

	return nil
}

func validateNumber(field models.ContentField, value interface{}) error {
	var numVal float64

	switch v := value.(type) {
	case float64:
		numVal = v
	case float32:
		numVal = float64(v)
	case int:
		numVal = float64(v)
	case int64:
		numVal = float64(v)
	default:
		return fmt.Errorf("field '%s' must be a number", field.Name)
	}

	// Min value check
	if field.MinValue != nil && numVal < *field.MinValue {
		return fmt.Errorf("field '%s' must be at least %.2f", field.Name, *field.MinValue)
	}

	// Max value check
	if field.MaxValue != nil && numVal > *field.MaxValue {
		return fmt.Errorf("field '%s' must not exceed %.2f", field.Name, *field.MaxValue)
	}

	return nil
}

func validateDate(field models.ContentField, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be date string", field.Name)
	}

	if _, err := time.Parse("2006-01-02", strVal); err != nil {
		return fmt.Errorf("field '%s' must be in format YYYY-MM-DD", field.Name)
	}

	return nil
}

func validateMedia(field models.ContentField, value interface{}) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be a valid media URL", field.Name)
	}

	if _, err := url.ParseRequestURI(strVal); err != nil {
		return fmt.Errorf("field '%s' must be a valid URL", field.Name)
	}

	return nil
}

func ValidatePartialUpdate(ct models.ContentType, updatedFields map[string]interface{}, entryID uint) error {
	allFields := append(ct.Fields, ct.SEOFields...)
	fieldMap := make(map[string]models.ContentField)
	for _, field := range allFields {
		fieldMap[field.Name] = field
	}

	for fieldName, value := range updatedFields {
		if strings.HasSuffix(fieldName, "_media_id") {
			continue
		}

		field, exists := fieldMap[fieldName]
		if !exists {
			return fmt.Errorf("field '%s' does not exist in content type", fieldName)
		}
		if value == nil || value == "" {
			if field.Required {
				return fmt.Errorf("field '%s' is required and cannot be empty", field.Name)
			}
			continue
		}
		if err := validateFieldByType(field, value); err != nil {
			return err
		}
		if field.Unique {
			if err := checkUniqueness(ct.ID, field.Name, value, &entryID); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateFieldByType(field models.ContentField, value interface{}) error {
	switch field.Type {
	case "string", "text":
		return validateString(field, value)
	case "email":
		return validateEmail(field, value)
	case "url":
		return validateURL(field, value)
	case "number":
		return validateNumber(field, value)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be boolean", field.Name)
		}
	case "date":
		return validateDate(field, value)
	case "media":
		return validateMedia(field, value)
	}
	return nil
}

func checkUniqueness(contentTypeID uint, fieldName string, value interface{}, excludeEntryID *uint) error {
	var count int64
	jsonValue, _ := json.Marshal(value)

	// Use raw SQL for JSONB comparison to ensure compatibility with PostgreSQL
	// data->'field' = 'value'::jsonb
	query := database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ?", contentTypeID).
		Where(fmt.Sprintf("data->'%s' = ?::jsonb", fieldName), string(jsonValue))

	if excludeEntryID != nil {
		query = query.Where("id != ?", *excludeEntryID)
	}

	err := query.Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check uniqueness for field '%s': %w", fieldName, err)
	}

	if count > 0 {
		return fmt.Errorf("field '%s' must be unique, value '%v' already exists", fieldName, value)
	}

	return nil
}

func GetFieldValidationRules(field models.ContentField) map[string]interface{} {
	rules := make(map[string]interface{})

	rules["name"] = field.Name
	rules["type"] = field.Type
	rules["required"] = field.Required
	rules["unique"] = field.Unique

	if field.MinLength != nil {
		rules["min_length"] = *field.MinLength
	}
	if field.MaxLength != nil {
		rules["max_length"] = *field.MaxLength
	}
	if field.Pattern != "" {
		rules["pattern"] = field.Pattern
	}
	if field.MinValue != nil {
		rules["min_value"] = *field.MinValue
	}
	if field.MaxValue != nil {
		rules["max_value"] = *field.MaxValue
	}
	if field.DefaultValue != "" {
		rules["default"] = field.DefaultValue
	}
	if field.Placeholder != "" {
		rules["placeholder"] = field.Placeholder
	}
	if field.HelpText != "" {
		rules["help_text"] = field.HelpText
	}

	return rules
}

func CreateContentEntry(contentTypeID, createdBy uint, data map[string]interface{}, projectID *uint) (*models.ContentEntry, error) {
	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID).Error; err != nil {
		return nil, err
	}

	// If content type has project_id, ensure it matches the entry's project_id
	if ct.ProjectID != nil {
		if projectID == nil || *ct.ProjectID != *projectID {
			return nil, fmt.Errorf("entry project_id must match content type project_id")
		}
	} else if projectID != nil {
		return nil, fmt.Errorf("cannot create project entry for global content type")
	}

	if err := ValidateContentEntryEnhanced(ct, data); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	entry := models.ContentEntry{
		ContentTypeID: contentTypeID,
		ProjectID:     projectID,
		Data:          datatypes.JSON(jsonData),
		Status:        models.StatusDraft,
		CreatedBy:     createdBy,
		UpdatedBy:     createdBy,
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		return nil, err
	}

	return &entry, nil
}

func GenerateSEOPreview(entryID uint) (map[string]interface{}, error) {
	var entry models.ContentEntry

	// 1. Ambil Content Entry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Content entry not found")
		}
		return nil, err
	}

	var data map[string]interface{}
	// 2. Unmarshal data JSONB
	if err := json.Unmarshal([]byte(entry.Data), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content data: %w", err)
	}

	// 3. Ambil Fields yang terkait dengan ContentType dari Entry ini
	var seoFields []models.ContentField

	// Query hanya mengambil fields yang ditandai is_seo=TRUE
	err := database.DB.
		Where("content_type_id = ? AND is_seo = TRUE", entry.ContentTypeID).
		Find(&seoFields).Error

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve SEO fields from schema: %w", err)
	}

	// 4. Inisialisasi daftar nama field SEO yang diizinkan (dari skema & hardcode)
	seoFieldNames := make(map[string]bool)

	// A. Tambahkan dari Skema Database (is_seo = TRUE)
	for _, field := range seoFields {
		seoFieldNames[field.Name] = true
	}

	// B. Tambahkan Hardcode/Konvensi standar (Meta, Slug, dll.)
	// Ini menangani field seperti "meta_title" yang mungkin tidak didesain sebagai ContentField
	// tetapi selalu diakui sebagai SEO.
	seoFieldNames["slug"] = true
	seoFieldNames["meta_title"] = true
	seoFieldNames["meta_description"] = true
	seoFieldNames["meta_image"] = true

	// 5. Filter data berdasarkan nama field SEO yang ditemukan
	seoData := make(map[string]interface{})

	for k, v := range data {
		kLower := strings.ToLower(k) // Gunakan kLower untuk pengecekan "meta" / "seo"

		// Cek 1: Berdasarkan Nama Field yang ada di Skema/Hardcode
		if seoFieldNames[k] {
			seoData[k] = v
			continue
		}

		// Cek 2: Logika Tambahan jika Anda ingin menyertakan field *lain* yang mengandung "meta" atau "seo"
		// Misalnya, jika ada field "seo_rating" yang tidak terdaftar di skema.
		if strings.Contains(kLower, "meta") || strings.Contains(kLower, "seo") {
			seoData[k] = v
			continue
		}

		// Cek 3: Pasangan kunci media ID-nya
		if strings.HasSuffix(kLower, "_media_id") {
			baseFieldName := strings.TrimSuffix(k, "_media_id")
			// Cek apakah baseFieldName adalah SEO (baik dari skema maupun hardcode)
			if seoFieldNames[baseFieldName] || strings.Contains(baseFieldName, "meta") || strings.Contains(baseFieldName, "seo") {
				seoData[k] = v
			}
		}
	}

	// 6. Validasi Akhir
	if len(seoData) == 0 {
		return nil, nil
	}

	return seoData, nil
}
func getPlaceholderString(count int) string {
	if count <= 0 {
		return ""
	}
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ", ")
}

func GetAccessibleEntries(userID uint, contentTypeID uint) ([]models.ContentEntry, error) {
	// 1. Ambil Project ID yang dapat diakses (MENGGUNAKAN FUNGSI ANDA YANG SUDAH ADA)
	const Module = "ProjectContent"
	const Action = "read"

	accessibleProjectIDs := middleware.GetAccessibleProjectIDs(userID, Module, Action) // ⭐️ Menggunakan helper Anda

	// 2. Tentukan kriteria query dasar
	query := database.DB.Preload("ContentType").Model(&models.ContentEntry{})

	// Filter berdasarkan ContentTypeID jika disediakan
	if contentTypeID > 0 {
		query = query.Where("content_type_id = ?", contentTypeID)
	}

	// 3. Tentukan kriteria otorisasi (ProjectID IS NULL OR ProjectID IN (...))

	// Mencari entri di project yang diizinkan ATAU entri yang bersifat global (ProjectID IS NULL)
	if len(accessibleProjectIDs) > 0 {
		// Logika: Tampilkan entri ProjectID yang diizinkan ATAU yang Global
		query = query.Where(
			database.DB.Where("project_id IN (?)", accessibleProjectIDs).Or("project_id IS NULL"),
		)
	} else {
		// Logika: Jika user tidak memiliki akses Project (tidak ada accessibleProjectIDs),
		// hanya tampilkan yang Global
		query = query.Where("project_id IS NULL")
	}

	var entries []models.ContentEntry
	// Urutkan (opsional)
	if err := query.Order("project_id ASC").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return entries, nil
}

func UpdateEntry(entryID, updatedBy uint, data map[string]interface{}) (*models.ContentEntry, error) {
	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return nil, err
	}

	if entry.Status == models.StatusPublished {
		return nil, fmt.Errorf("cannot edit published content directly, please create a new version or unpublish first")
	}

	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, entry.ContentTypeID).Error; err != nil {
		return nil, err
	}

	if err := ValidateContentEntryEnhanced(ct, data); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	entry.Data = datatypes.JSON(jsonData)
	entry.Status = models.StatusDraft
	entry.UpdatedBy = updatedBy

	if err := database.DB.Save(&entry).Error; err != nil {
		return nil, err
	}

	return &entry, nil
}
