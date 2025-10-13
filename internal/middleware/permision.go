package middleware

import (
	"encoding/json"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
)

func PermissionProtected(module string, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)

		var user models.User
		if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
			return response.Unauthorized(c, "Unauthorized")
		}

		if user.Role == nil {
			return response.Forbidden(c, "User has no role assigned")
		}

		hasPermission := false
		for _, perm := range user.Role.Permissions {
			if perm.Module == module && perm.Action == action {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return response.Forbidden(c, "You don't have permission to perform this action")
		}

		return c.Next()
	}
}

func HasPermission(userID uint, module, action string) bool {
	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return false
	}

	if user.Role == nil {
		return false
	}

	for _, perm := range user.Role.Permissions {
		if perm.Module == module && perm.Action == action {
			return true
		}
	}
	return false
}

func HasAnyPermission(userID uint, permissions []struct{ Module, Action string }) bool {
	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return false
	}

	if user.Role == nil {
		return false
	}

	for _, reqPerm := range permissions {
		for _, userPerm := range user.Role.Permissions {
			if userPerm.Module == reqPerm.Module && userPerm.Action == reqPerm.Action {
				return true
			}
		}
	}
	return false
}

func IsFullAccessRole(user *models.User) bool {
	if user.Role == nil {
		return false
	}

	if user.Role.Name == "admin" {
		return true
	}

	requiredPerms := map[string]map[string]bool{
		"ContentEntry": {"create": false, "read": false, "update": false, "delete": false},
		"Media":        {"create": false, "read": false, "update": false, "delete": false},
		"SEO":          {"create": false, "read": false, "update": false, "delete": false},
	}

	for _, perm := range user.Role.Permissions {
		if actions, exists := requiredPerms[perm.Module]; exists {
			if _, actionExists := actions[perm.Action]; actionExists {
				requiredPerms[perm.Module][perm.Action] = true
			}
		}
	}

	for _, actions := range requiredPerms {
		for _, has := range actions {
			if !has {
				return false
			}
		}
	}

	return true
}

func FilterFieldsByPermission(userID uint, action string, data map[string]interface{}, contentTypeID uint) (map[string]interface{}, error) {
	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return nil, fiber.NewError(401, "Unauthorized")
	}

	if IsFullAccessRole(&user) {
		return data, nil
	}

	var userPermission *models.Permission
	for _, perm := range user.Role.Permissions {
		if perm.Module == "ContentEntry" && perm.Action == action {
			if perm.ContentTypeIDs != nil {
				var contentTypeIDs []uint
				json.Unmarshal(perm.ContentTypeIDs, &contentTypeIDs)

				if len(contentTypeIDs) > 0 {
					hasAccess := false
					for _, id := range contentTypeIDs {
						if id == contentTypeID {
							hasAccess = true
							break
						}
					}
					if !hasAccess {
						continue
					}
				}
			}

			userPermission = &perm
			break
		}
	}

	if userPermission == nil {
		return nil, fiber.NewError(403, "No permission for this action")
	}

	if userPermission.FieldScope == "" {
		userPermission.FieldScope = "all"
	}

	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID).Error; err != nil {
		return nil, err
	}

	allFields := append(ct.Fields, ct.SEOFields...)
	filteredData := make(map[string]interface{})

	if userPermission.FieldScope == "custom" {
		var allowedFields []string
		var deniedFields []string

		if userPermission.AllowedFields != nil {
			json.Unmarshal(userPermission.AllowedFields, &allowedFields)
		}
		if userPermission.DeniedFields != nil {
			json.Unmarshal(userPermission.DeniedFields, &deniedFields)
		}

		if len(allowedFields) > 0 {
			for k, v := range data {
				for _, allowedField := range allowedFields {
					if k == allowedField {
						filteredData[k] = v
						break
					}
				}
			}
			return filteredData, nil
		}

		if len(deniedFields) > 0 {
			for k, v := range data {
				isDenied := false
				for _, deniedField := range deniedFields {
					if k == deniedField {
						isDenied = true
						break
					}
				}
				if !isDenied {
					filteredData[k] = v
				}
			}
			return filteredData, nil
		}
	}

	for k, v := range data {
		for _, field := range allFields {
			if field.Name != k {
				continue
			}

			switch userPermission.FieldScope {
			case "seo_only":
				if field.IsSEO {
					filteredData[k] = v
				}
			case "non_seo_only":
				if !field.IsSEO {
					filteredData[k] = v
				}
			default:
				filteredData[k] = v
			}
			break
		}
	}

	return filteredData, nil
}

func CanAccessField(userID uint, fieldName string, contentTypeID uint) (bool, error) {
	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return false, err
	}

	if user.Role == nil {
		return false, nil
	}

	if IsFullAccessRole(&user) {
		return true, nil
	}

	var ct models.ContentType
	database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID)

	var targetField *models.ContentField
	allFields := append(ct.Fields, ct.SEOFields...)
	for _, f := range allFields {
		if f.Name == fieldName {
			targetField = &f
			break
		}
	}

	if targetField == nil {
		return false, nil
	}

	for _, perm := range user.Role.Permissions {
		if perm.Module != "ContentEntry" {
			continue
		}

		if perm.ContentTypeIDs != nil {
			var contentTypeIDs []uint
			json.Unmarshal(perm.ContentTypeIDs, &contentTypeIDs)

			if len(contentTypeIDs) > 0 {
				hasAccess := false
				for _, id := range contentTypeIDs {
					if id == contentTypeID {
						hasAccess = true
						break
					}
				}
				if !hasAccess {
					continue
				}
			}
		}

		if perm.FieldScope == "custom" {
			var allowedFields []string
			var deniedFields []string

			if perm.AllowedFields != nil {
				json.Unmarshal(perm.AllowedFields, &allowedFields)
			}
			if perm.DeniedFields != nil {
				json.Unmarshal(perm.DeniedFields, &deniedFields)
			}

			if len(allowedFields) > 0 {
				for _, af := range allowedFields {
					if af == fieldName {
						return true, nil
					}
				}
				return false, nil
			}

			if len(deniedFields) > 0 {
				for _, df := range deniedFields {
					if df == fieldName {
						return false, nil
					}
				}
				return true, nil
			}
		}
		switch perm.FieldScope {
		case "", "all":
			return true, nil
		case "seo_only":
			return targetField.IsSEO, nil
		case "non_seo_only":
			return !targetField.IsSEO, nil
		}
	}

	return false, nil
}

type Module string
type Action string

const (
	ContentModule Module = "ContentEntry"
	MediaModule   Module = "Media"
	SEOModule     Module = "SEO"
	CreateAction  Action = "create"
	ReadAction    Action = "read"
	UpdateAction  Action = "update"
	DeleteAction  Action = "delete"
	ApproveAction Action = "approve"
	UploadAction  Action = "upload"
)
