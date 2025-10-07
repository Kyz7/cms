package content

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"gorm.io/datatypes"
)

func CreateContentType(name, slug string) (*models.ContentType, error) {
	ct := models.ContentType{Name: name, Slug: slug}
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

func ListContentEntries(contentTypeID uint) ([]models.ContentEntry, error) {
	var entries []models.ContentEntry
	if err := database.DB.Where("content_type_id = ?", contentTypeID).Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

func CreateContentRelation(fromID, toID uint, relationType string) (*models.ContentRelation, error) {
	relation := models.ContentRelation{
		FromContentID: fromID,
		ToContentID:   toID,
		RelationType:  relationType,
	}
	if err := database.DB.Create(&relation).Error; err != nil {
		return nil, err
	}
	return &relation, nil
}

func ListContentRelations(fromID uint) ([]models.ContentRelation, error) {
	var relations []models.ContentRelation
	if err := database.DB.Where("from_content_id = ?", fromID).Find(&relations).Error; err != nil {
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

	query := database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND data @> ?", contentTypeID,
			fmt.Sprintf(`{"%s": %s}`, fieldName, jsonValue))

	if excludeEntryID != nil {
		query = query.Where("id != ?", *excludeEntryID)
	}

	err := query.Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check uniqueness for field '%s'", fieldName)
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

func CreateContentEntry(contentTypeID, createdBy uint, data map[string]interface{}) (*models.ContentEntry, error) {
	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID).Error; err != nil {
		return nil, err
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
		Data:          datatypes.JSON(jsonData),
		Status:        models.StatusDraft,
		CreatedBy:     createdBy,
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		return nil, err
	}

	return &entry, nil
}

func GenerateSEOPreview(entryID uint) (map[string]interface{}, error) {
	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(entry.Data), &data); err != nil {
		return nil, err
	}

	seoData := make(map[string]interface{})
	for k, v := range data {
		if strings.HasPrefix(k, "seo_") || k == "slug" || k == "meta_title" || k == "meta_description" || k == "meta_image" {
			seoData[k] = v
		}
	}

	return seoData, nil
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
