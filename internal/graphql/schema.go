package graphql

import (
	"time"

	"github.com/Kyz7/cms/internal/models"
	"github.com/graphql-go/graphql"
)

// Custom scalar types
var TimeType = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "Time",
	Description: "Time scalar type",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case time.Time:
			return v.Format(time.RFC3339)
		case *time.Time:
			if v == nil {
				return nil
			}
			return v.Format(time.RFC3339)
		default:
			return nil
		}
	},
})

var JSONType = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "JSON scalar type",
	Serialize: func(value interface{}) interface{} {
		return value
	},
})

// Enum types
var WorkflowStatusEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "WorkflowStatus",
	Values: graphql.EnumValueConfigMap{
		"DRAFT": &graphql.EnumValueConfig{
			Value: "draft",
		},
		"IN_REVIEW": &graphql.EnumValueConfig{
			Value: "in_review",
		},
		"READY_FOR_APPROVAL": &graphql.EnumValueConfig{
			Value: "ready_for_approval",
		},
		"APPROVED": &graphql.EnumValueConfig{
			Value: "approved",
		},
		"PUBLISHED": &graphql.EnumValueConfig{
			Value: "published",
		},
		"REJECTED": &graphql.EnumValueConfig{
			Value: "rejected",
		},
	},
})

var UserStatusEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "UserStatus",
	Values: graphql.EnumValueConfigMap{
		"ACTIVE": &graphql.EnumValueConfig{
			Value: "active",
		},
		"INACTIVE": &graphql.EnumValueConfig{
			Value: "inactive",
		},
		"SUSPENDED": &graphql.EnumValueConfig{
			Value: "suspended",
		},
	},
})

// Type definitions
var UserType = graphql.NewObject(graphql.ObjectConfig{
	Name: "User",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"name": &graphql.Field{
			Type: graphql.String,
		},
		"email": &graphql.Field{
			Type: graphql.String,
		},
		"provider": &graphql.Field{
			Type: graphql.String,
		},
		"status": &graphql.Field{
			Type: UserStatusEnum,
		},
		"roleId": &graphql.Field{
			Type: graphql.Int,
		},
		"role": &graphql.Field{
			Type: RoleType,
		},
		"profile": &graphql.Field{
			Type: graphql.String,
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var RoleType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Role",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"name": &graphql.Field{
			Type: graphql.String,
		},
		"description": &graphql.Field{
			Type: graphql.String,
		},
		"permissions": &graphql.Field{
			Type: graphql.NewList(PermissionType),
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var PermissionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Permission",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"roleId": &graphql.Field{
			Type: graphql.Int,
		},
		"module": &graphql.Field{
			Type: graphql.String,
		},
		"action": &graphql.Field{
			Type: graphql.String,
		},
		"fieldScope": &graphql.Field{
			Type: graphql.String,
		},
		"allowedFields": &graphql.Field{
			Type: JSONType,
		},
		"deniedFields": &graphql.Field{
			Type: JSONType,
		},
		"contentTypeIds": &graphql.Field{
			Type: JSONType,
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var ContentTypeType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ContentType",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"name": &graphql.Field{
			Type: graphql.String,
		},
		"slug": &graphql.Field{
			Type: graphql.String,
		},
		"enableSeo": &graphql.Field{
			Type: graphql.Boolean,
		},
		"fields": &graphql.Field{
			Type: graphql.NewList(ContentFieldType),
		},
		"seoFields": &graphql.Field{
			Type: graphql.NewList(ContentFieldType),
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var ContentFieldType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ContentField",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"contentTypeId": &graphql.Field{
			Type: graphql.Int,
		},
		"name": &graphql.Field{
			Type: graphql.String,
		},
		"type": &graphql.Field{
			Type: graphql.String,
		},
		"required": &graphql.Field{
			Type: graphql.Boolean,
		},
		"isSeo": &graphql.Field{
			Type: graphql.Boolean,
		},
		"unique": &graphql.Field{
			Type: graphql.Boolean,
		},
		"maxLength": &graphql.Field{
			Type: graphql.Int,
		},
		"minLength": &graphql.Field{
			Type: graphql.Int,
		},
		"pattern": &graphql.Field{
			Type: graphql.String,
		},
		"minValue": &graphql.Field{
			Type: graphql.Float,
		},
		"maxValue": &graphql.Field{
			Type: graphql.Float,
		},
		"defaultValue": &graphql.Field{
			Type: graphql.String,
		},
		"placeholder": &graphql.Field{
			Type: graphql.String,
		},
		"helpText": &graphql.Field{
			Type: graphql.String,
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var ContentEntryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ContentEntry",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"contentTypeId": &graphql.Field{
			Type: graphql.Int,
		},
		"contentType": &graphql.Field{
			Type: ContentTypeType,
		},
		"data": &graphql.Field{
			Type: JSONType,
		},
		"status": &graphql.Field{
			Type: WorkflowStatusEnum,
		},
		"createdBy": &graphql.Field{
			Type: graphql.Int,
		},
		"updatedBy": &graphql.Field{
			Type: graphql.Int,
		},
		"creator": &graphql.Field{
			Type: UserType,
		},
		"updater": &graphql.Field{
			Type: UserType,
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
		"publishedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var MediaFileType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MediaFile",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.ID,
		},
		"fileName": &graphql.Field{
			Type: graphql.String,
		},
		"url": &graphql.Field{
			Type: graphql.String,
		},
		"type": &graphql.Field{
			Type: graphql.String,
		},
		"size": &graphql.Field{
			Type: graphql.Int,
		},
		"width": &graphql.Field{
			Type: graphql.Int,
		},
		"height": &graphql.Field{
			Type: graphql.Int,
		},
		"folder": &graphql.Field{
			Type: graphql.String,
		},
		"tags": &graphql.Field{
			Type: JSONType,
		},
		"alt": &graphql.Field{
			Type: graphql.String,
		},
		"caption": &graphql.Field{
			Type: graphql.String,
		},
		"uploadedBy": &graphql.Field{
			Type: graphql.Int,
		},
		"uploader": &graphql.Field{
			Type: UserType,
		},
		"createdAt": &graphql.Field{
			Type: TimeType,
		},
		"updatedAt": &graphql.Field{
			Type: TimeType,
		},
	},
})

var AuthPayloadType = graphql.NewObject(graphql.ObjectConfig{
	Name: "AuthPayload",
	Fields: graphql.Fields{
		"token": &graphql.Field{
			Type: graphql.String,
		},
		"refreshToken": &graphql.Field{
			Type: graphql.String,
		},
		"user": &graphql.Field{
			Type: UserType,
		},
	},
})

// Helper function to convert models to GraphQL types
func ConvertUserToGraphQL(user *models.User) map[string]interface{} {
	result := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"provider":  user.Provider,
		"status":    user.Status,
		"roleId":    user.RoleID,
		"profile":   user.Profile,
		"createdAt": user.CreatedAt,
		"updatedAt": user.UpdatedAt,
	}

	if user.Role != nil {
		result["role"] = ConvertRoleToGraphQL(user.Role)
	}

	return result
}

func ConvertRoleToGraphQL(role *models.Role) map[string]interface{} {
	result := map[string]interface{}{
		"id":          role.ID,
		"name":        role.Name,
		"description": role.Description,
		"createdAt":   role.CreatedAt,
		"updatedAt":   role.UpdatedAt,
	}

	if len(role.Permissions) > 0 {
		permissions := make([]map[string]interface{}, len(role.Permissions))
		for i, perm := range role.Permissions {
			permissions[i] = ConvertPermissionToGraphQL(&perm)
		}
		result["permissions"] = permissions
	}

	return result
}

func ConvertPermissionToGraphQL(permission *models.Permission) map[string]interface{} {
	return map[string]interface{}{
		"id":             permission.ID,
		"roleId":         permission.RoleID,
		"module":         permission.Module,
		"action":         permission.Action,
		"fieldScope":     permission.FieldScope,
		"allowedFields":  permission.AllowedFields,
		"deniedFields":   permission.DeniedFields,
		"contentTypeIds": permission.ContentTypeIDs,
		"createdAt":      permission.CreatedAt,
		"updatedAt":      permission.UpdatedAt,
	}
}

func ConvertContentTypeToGraphQL(contentType *models.ContentType) map[string]interface{} {
	result := map[string]interface{}{
		"id":        contentType.ID,
		"name":      contentType.Name,
		"slug":      contentType.Slug,
		"enableSeo": contentType.EnableSEO,
		"createdAt": contentType.CreatedAt,
		"updatedAt": contentType.UpdatedAt,
	}

	if len(contentType.Fields) > 0 {
		fields := make([]map[string]interface{}, len(contentType.Fields))
		for i, field := range contentType.Fields {
			fields[i] = ConvertContentFieldToGraphQL(&field)
		}
		result["fields"] = fields
	}

	return result
}

func ConvertContentFieldToGraphQL(field *models.ContentField) map[string]interface{} {
	return map[string]interface{}{
		"id":            field.ID,
		"contentTypeId": field.ContentTypeID,
		"name":          field.Name,
		"type":          field.Type,
		"required":      field.Required,
		"isSeo":         field.IsSEO,
		"unique":        field.Unique,
		"maxLength":     field.MaxLength,
		"minLength":     field.MinLength,
		"pattern":       field.Pattern,
		"minValue":      field.MinValue,
		"maxValue":      field.MaxValue,
		"defaultValue":  field.DefaultValue,
		"placeholder":   field.Placeholder,
		"helpText":      field.HelpText,
		"createdAt":     field.CreatedAt,
		"updatedAt":     field.UpdatedAt,
	}
}

func ConvertContentEntryToGraphQL(entry *models.ContentEntry) map[string]interface{} {
	result := map[string]interface{}{
		"id":            entry.ID,
		"contentTypeId": entry.ContentTypeID,
		"data":          entry.Data,
		"status":        entry.Status,
		"createdBy":     entry.CreatedBy,
		"updatedBy":     entry.UpdatedBy,
		"createdAt":     entry.CreatedAt,
		"updatedAt":     entry.UpdatedAt,
	}

	if entry.Creator != nil {
		result["creator"] = ConvertUserToGraphQL(entry.Creator)
	}

	if entry.Updater != nil {
		result["updater"] = ConvertUserToGraphQL(entry.Updater)
	}

	if entry.PublishedAt != nil {
		result["publishedAt"] = *entry.PublishedAt
	}

	return result
}

func ConvertMediaFileToGraphQL(media *models.MediaFile) map[string]interface{} {
	result := map[string]interface{}{
		"id":         media.ID,
		"fileName":   media.FileName,
		"url":        media.URL,
		"type":       media.Type,
		"size":       media.Size,
		"width":      media.Width,
		"height":     media.Height,
		"folder":     media.Folder,
		"tags":       media.Tags,
		"alt":        media.Alt,
		"caption":    media.Caption,
		"uploadedBy": media.UploadedBy,
		"createdAt":  media.CreatedAt,
		"updatedAt":  media.UpdatedAt,
	}

	if media.Uploader != nil {
		result["uploader"] = ConvertUserToGraphQL(media.Uploader)
	}

	return result
}
