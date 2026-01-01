package graphql

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Kyz7/cms/internal/content"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/project"
	"github.com/Kyz7/cms/internal/search"
	"github.com/Kyz7/cms/internal/translation"
	"github.com/Kyz7/cms/internal/workflow"
	"github.com/graphql-go/graphql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

func (r *Resolvers) MeResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := r.DB.Preload("Role").First(&user, userID).Error; err != nil {
		return nil, err
	}

	return ConvertUserToGraphQL(&user), nil
}

func (r *Resolvers) LoginResolver(p graphql.ResolveParams) (interface{}, error) {
	email := p.Args["email"].(string)
	password := p.Args["password"].(string)

	var user models.User
	if err := r.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	token := "jwt_token_here"
	refreshToken := "refresh_token_here"

	return map[string]interface{}{
		"token":        token,
		"refreshToken": refreshToken,
		"user":         ConvertUserToGraphQL(&user),
	}, nil
}

func (r *Resolvers) RegisterResolver(p graphql.ResolveParams) (interface{}, error) {
	name := p.Args["name"].(string)
	email := p.Args["email"].(string)
	password := p.Args["password"].(string)

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
		RoleID:   2,
		Status:   "active",
		Provider: "local",
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Password = string(hash)

	if err := r.DB.Create(user).Error; err != nil {
		return nil, err
	}

	token := "jwt_token_here"
	refreshToken := "refresh_token_here"

	return map[string]interface{}{
		"token":        token,
		"refreshToken": refreshToken,
		"user":         ConvertUserToGraphQL(user),
	}, nil
}

func (r *Resolvers) UsersResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "User", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	var users []models.User
	query := r.DB.Preload("Role")

	if limit, ok := p.Args["limit"].(int); ok {
		query = query.Limit(limit)
	}
	if offset, ok := p.Args["offset"].(int); ok {
		query = query.Offset(offset)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(users))
	for i, user := range users {
		result[i] = ConvertUserToGraphQL(&user)
	}

	return result, nil
}

func (r *Resolvers) UserResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "User", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := r.DB.Preload("Role").First(&user, id).Error; err != nil {
		return nil, err
	}

	return ConvertUserToGraphQL(&user), nil
}

func (r *Resolvers) CreateUserResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "User", "create") {
		return nil, fmt.Errorf("permission denied")
	}

	name := p.Args["name"].(string)
	email := p.Args["email"].(string)
	password := p.Args["password"].(string)
	roleID := uint(p.Args["roleId"].(int))

	newUser := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
		RoleID:   roleID,
		Status:   "active",
		Provider: "local",
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	newUser.Password = string(hash)

	if err := r.DB.Create(newUser).Error; err != nil {
		return nil, err
	}

	return ConvertUserToGraphQL(newUser), nil
}

func (r *Resolvers) UpdateUserResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "User", "update") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := r.DB.First(&user, id).Error; err != nil {
		return nil, err
	}

	if name, ok := p.Args["name"].(string); ok {
		user.Name = name
	}
	if email, ok := p.Args["email"].(string); ok {
		user.Email = email
	}
	if password, ok := p.Args["password"].(string); ok {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hash)
	}
	if roleID, ok := p.Args["roleId"].(int); ok {
		user.RoleID = uint(roleID)
	}
	if status, ok := p.Args["status"].(string); ok {
		user.Status = status
	}

	if err := r.DB.Save(&user).Error; err != nil {
		return nil, err
	}

	return ConvertUserToGraphQL(&user), nil
}

func (r *Resolvers) DeleteUserResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "User", "delete") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	if err := r.DB.Delete(&models.User{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

func (r *Resolvers) RolesResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Role", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	var roles []models.Role
	if err := r.DB.Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(roles))
	for i, role := range roles {
		result[i] = ConvertRoleToGraphQL(&role)
	}

	return result, nil
}

func (r *Resolvers) RoleResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Role", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var role models.Role
	if err := r.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		return nil, err
	}

	return ConvertRoleToGraphQL(&role), nil
}

func (r *Resolvers) CreateRoleResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Role", "create") {
		return nil, fmt.Errorf("permission denied")
	}

	name := p.Args["name"].(string)
	description := ""
	if desc, ok := p.Args["description"].(string); ok {
		description = desc
	}

	role := &models.Role{
		Name:        name,
		Description: description,
	}

	if err := r.DB.Create(role).Error; err != nil {
		return nil, err
	}

	return ConvertRoleToGraphQL(role), nil
}

func (r *Resolvers) UpdateRoleResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Role", "update") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var role models.Role
	if err := r.DB.First(&role, id).Error; err != nil {
		return nil, err
	}
	if name, ok := p.Args["name"].(string); ok {
		role.Name = name
	}
	if description, ok := p.Args["description"].(string); ok {
		role.Description = description
	}

	if err := r.DB.Save(&role).Error; err != nil {
		return nil, err
	}

	return ConvertRoleToGraphQL(&role), nil
}

func (r *Resolvers) DeleteRoleResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Role", "delete") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	if err := r.DB.Delete(&models.Role{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

func (r *Resolvers) ContentTypesResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	var contentTypes []models.ContentType
	if err := r.DB.Preload("Fields").Find(&contentTypes).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(contentTypes))
	for i, ct := range contentTypes {
		result[i] = ConvertContentTypeToGraphQL(&ct)
	}

	return result, nil
}

func (r *Resolvers) ContentTypeResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var contentType models.ContentType
	if err := r.DB.Preload("Fields").First(&contentType, id).Error; err != nil {
		return nil, err
	}

	return ConvertContentTypeToGraphQL(&contentType), nil
}

func (r *Resolvers) CreateContentTypeResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "create") {
		return nil, fmt.Errorf("permission denied")
	}

	name := p.Args["name"].(string)
	slug := p.Args["slug"].(string)
	enableSeo := false
	if seo, ok := p.Args["enableSeo"].(bool); ok {
		enableSeo = seo
	}

	contentType := &models.ContentType{
		Name:      name,
		Slug:      slug,
		EnableSEO: enableSeo,
	}

	// Handle projectId if provided
	if projectIDVal, ok := p.Args["projectId"]; ok && projectIDVal != nil {
		projectID := uint(projectIDVal.(int))
		contentType.ProjectID = &projectID
	}

	if err := r.DB.Create(contentType).Error; err != nil {
		return nil, err
	}

	return ConvertContentTypeToGraphQL(contentType), nil
}

func (r *Resolvers) UpdateContentTypeResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	updateData := make(map[string]interface{})
	if name, ok := p.Args["name"].(string); ok {
		updateData["name"] = name
	}
	if slug, ok := p.Args["slug"].(string); ok {
		updateData["slug"] = slug
	}
	if enableSeo, ok := p.Args["enableSeo"].(bool); ok {
		updateData["enable_seo"] = enableSeo
	}

	var contentType models.ContentType
	if err := r.DB.First(&contentType, id).Error; err != nil {
		return nil, err
	}

	if name, ok := p.Args["name"].(string); ok {
		contentType.Name = name
	}
	if slug, ok := p.Args["slug"].(string); ok {
		contentType.Slug = slug
	}
	if enableSeo, ok := p.Args["enableSeo"].(bool); ok {
		contentType.EnableSEO = enableSeo
	}

	if err := r.DB.Save(&contentType).Error; err != nil {
		return nil, err
	}

	return ConvertContentTypeToGraphQL(&contentType), nil
}

func (r *Resolvers) DeleteContentTypeResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "delete") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	if err := r.DB.Delete(&models.ContentType{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

func (r *Resolvers) ContentEntriesResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	contentTypeID := p.Args["contentTypeId"].(int)

	var entries []models.ContentEntry
	query := r.DB.Where("content_type_id = ?", contentTypeID).Preload("Creator").Preload("Updater")

	if limit, ok := p.Args["limit"].(int); ok {
		query = query.Limit(limit)
	}
	if offset, ok := p.Args["offset"].(int); ok {
		query = query.Offset(offset)
	}

	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		result[i] = ConvertContentEntryToGraphQL(&entry)
	}

	return result, nil
}

func (r *Resolvers) ContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var entry models.ContentEntry
	if err := r.DB.Preload("Creator").Preload("Updater").First(&entry, id).Error; err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG ContentEntry: ID=%d, Status=%s (type: %T)\n", entry.ID, entry.Status, entry.Status)
	return ConvertContentEntryToGraphQL(&entry), nil
}

func (r *Resolvers) CreateContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "create") {
		return nil, fmt.Errorf("permission denied")
	}

	contentTypeID, err := strconv.ParseUint(p.Args["contentTypeId"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	data := p.Args["data"]

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	entry := &models.ContentEntry{
		ContentTypeID: uint(contentTypeID),
		Data:          datatypes.JSON(jsonData),
		Status:        "draft",
		CreatedBy:     userID,
		UpdatedBy:     userID,
	}

	if err := r.DB.Create(entry).Error; err != nil {
		return nil, err
	}

	// Fetch entry with relationships
	if err := r.DB.Preload("Creator").Preload("Updater").First(entry, entry.ID).Error; err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG CreateContentEntry: ID=%d, Status=%s (type: %T)\n", entry.ID, entry.Status, entry.Status)
	return ConvertContentEntryToGraphQL(entry), nil
}

func (r *Resolvers) UpdateContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	updateData := make(map[string]interface{})
	if data, ok := p.Args["data"]; ok {
		updateData["data"] = data
	}
	if status, ok := p.Args["status"].(string); ok {
		updateData["status"] = status
	}

	var entry models.ContentEntry
	if err := r.DB.First(&entry, id).Error; err != nil {
		return nil, err
	}

	if data, ok := p.Args["data"]; ok {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		entry.Data = datatypes.JSON(jsonData)
	}
	if status, ok := p.Args["status"].(string); ok {
		entry.Status = models.WorkflowStatus(status)
	}
	entry.UpdatedBy = userID

	if err := r.DB.Save(&entry).Error; err != nil {
		return nil, err
	}

	// Fetch entry with relationships
	if err := r.DB.Preload("Creator").Preload("Updater").First(&entry, entry.ID).Error; err != nil {
		return nil, err
	}

	return ConvertContentEntryToGraphQL(&entry), nil
}

func (r *Resolvers) DeleteContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "ContentEntry", "delete") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	if err := r.DB.Delete(&models.ContentEntry{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

func (r *Resolvers) TranslateContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}
	targetLang := p.Args["targetLang"].(string)
	sourceLang := ""
	if v, ok := p.Args["sourceLang"].(string); ok {
		sourceLang = v
	}

	var fieldsFilter []string
	if v, ok := p.Args["fields"].([]interface{}); ok {
		for _, f := range v {
			if s, ok := f.(string); ok {
				fieldsFilter = append(fieldsFilter, s)
			}
		}
	}

	var entry models.ContentEntry
	if err := r.DB.First(&entry, id).Error; err != nil {
		return nil, err
	}

	data := map[string]interface{}{}
	if err := json.Unmarshal([]byte(entry.Data), &data); err != nil {
		return nil, err
	}

	texts := []string{}
	keys := []string{}
	includeAll := len(fieldsFilter) == 0
	shouldInclude := func(k string) bool {
		if includeAll {
			return true
		}
		for _, f := range fieldsFilter {
			if f == k {
				return true
			}
		}
		return false
	}

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
		return nil, err
	}
	translated, err := client.TranslateTexts(texts, targetLang, sourceLang)
	if err != nil {
		return nil, err
	}

	i18nAny, ok := data["_i18n"].(map[string]interface{})
	if !ok || i18nAny == nil {
		i18nAny = map[string]interface{}{}
	}
	langMapAny, _ := i18nAny[targetLang].(map[string]interface{})
	if langMapAny == nil {
		langMapAny = map[string]interface{}{}
	}
	for i, key := range keys {
		if i < len(translated) {
			langMapAny[key] = translated[i]
		}
	}
	i18nAny[targetLang] = langMapAny
	data["_i18n"] = i18nAny

	buf, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	entry.Data = datatypes.JSON(buf)
	entry.UpdatedBy = userID
	if err := r.DB.Save(&entry).Error; err != nil {
		return nil, err
	}

	return ConvertContentEntryToGraphQL(&entry), nil
}

func (r *Resolvers) MediaFilesResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Media", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	var mediaFiles []models.MediaFile
	query := r.DB.Preload("Uploader")

	if folder, ok := p.Args["folder"].(string); ok {
		query = query.Where("folder = ?", folder)
	}

	if limit, ok := p.Args["limit"].(int); ok {
		query = query.Limit(limit)
	}
	if offset, ok := p.Args["offset"].(int); ok {
		query = query.Offset(offset)
	}

	if err := query.Find(&mediaFiles).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(mediaFiles))
	for i, media := range mediaFiles {
		result[i] = ConvertMediaFileToGraphQL(&media)
	}

	return result, nil
}

func (r *Resolvers) MediaFileResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Media", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	id, err := strconv.ParseUint(p.Args["id"].(string), 10, 32)
	if err != nil {
		return nil, err
	}

	var mediaFile models.MediaFile
	if err := r.DB.Preload("Uploader").First(&mediaFile, id).Error; err != nil {
		return nil, err
	}

	return ConvertMediaFileToGraphQL(&mediaFile), nil
}

func (r *Resolvers) MediaFoldersResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "Media", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	var folders []models.MediaFolder
	if err := r.DB.Preload("Parent").Order("path").Find(&folders).Error; err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(folders))
	for i, f := range folders {
		m := map[string]interface{}{
			"id":        f.ID,
			"name":      f.Name,
			"path":      f.Path,
			"createdBy": f.CreatedBy,
			"createdAt": f.CreatedAt,
			"updatedAt": f.UpdatedAt,
		}
		if f.ParentID != nil {
			m["parentId"] = *f.ParentID
		}
		if f.Parent != nil {
			m["parent"] = map[string]interface{}{
				"id":   f.Parent.ID,
				"name": f.Parent.Name,
				"path": f.Parent.Path,
			}
		}
		result[i] = m
	}
	return result, nil
}

func (r *Resolvers) CreateMediaFolderResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "Media", "create") {
		return nil, fmt.Errorf("permission denied")
	}
	name := p.Args["name"].(string)
	var parentIDPtr *uint
	if v, ok := p.Args["parentId"].(int); ok {
		vv := uint(v)
		parentIDPtr = &vv
	}
	path := "/" + name
	if parentIDPtr != nil {
		var parent models.MediaFolder
		if err := r.DB.First(&parent, *parentIDPtr).Error; err != nil {
			return nil, err
		}
		path = parent.Path + "/" + name
	}
	folder := &models.MediaFolder{
		Name:      name,
		Path:      path,
		ParentID:  parentIDPtr,
		CreatedBy: userID,
	}
	if err := r.DB.Create(folder).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":        folder.ID,
		"name":      folder.Name,
		"path":      folder.Path,
		"parentId":  folder.ParentID,
		"createdBy": folder.CreatedBy,
		"createdAt": folder.CreatedAt,
		"updatedAt": folder.UpdatedAt,
	}, nil
}

func (r *Resolvers) MediaStatsResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "Media", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	type counts struct {
		TotalFiles    int64
		TotalSize     int64
		RecentUploads int64
	}
	var c counts
	if err := r.DB.Model(&models.MediaFile{}).Count(&c.TotalFiles).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Model(&models.MediaFile{}).Select("COALESCE(SUM(size),0)").Row().Scan(&c.TotalSize); err != nil {
		return nil, err
	}
	if err := r.DB.Model(&models.MediaFile{}).Where("created_at > NOW() - INTERVAL '24 HOURS'").Count(&c.RecentUploads).Error; err != nil {
		c.RecentUploads = 0
	}
	var mediaFiles []models.MediaFile
	if err := r.DB.Select("type").Find(&mediaFiles).Error; err != nil {
		return nil, err
	}
	byType := map[string]int{}
	for _, m := range mediaFiles {
		t := m.Type
		if idx := len(t); idx >= 0 {
			// split by '/'
		}
		v := t
		if i := indexRune(t, '/'); i >= 0 {
			v = t[:i]
		}
		byType[v] = byType[v] + 1
	}
	return map[string]interface{}{
		"totalFiles":    c.TotalFiles,
		"totalSize":     c.TotalSize,
		"byType":        byType,
		"recentUploads": c.RecentUploads,
		"storageMode":   "local",
	}, nil
}

func indexRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

func (r *Resolvers) SearchEntriesResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	params := search.SearchParams{
		Query:    getStringArg(p, "query"),
		Status:   getStringArg(p, "status"),
		FromDate: getStringArg(p, "fromDate"),
		ToDate:   getStringArg(p, "toDate"),
		Page:     getIntArg(p, "page", 1),
		Limit:    getIntArg(p, "limit", 10),
		SortBy:   getStringArg(p, "sortBy"),
		OrderBy:  getStringArg(p, "orderBy"),
	}
	if arr, ok := p.Args["contentTypeIds"].([]interface{}); ok {
		for _, v := range arr {
			if n, ok := v.(int); ok {
				params.ContentTypeIDs = append(params.ContentTypeIDs, uint(n))
			}
		}
	}
	if arr, ok := p.Args["fields"].([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				params.Fields = append(params.Fields, s)
			}
		}
	}
	if arr, ok := p.Args["tags"].([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				params.Tags = append(params.Tags, s)
			}
		}
	}
	if v, ok := p.Args["createdBy"].(int); ok && v > 0 {
		params.CreatedBy = uint(v)
	}
	res, err := search.FullTextSearch(params)
	if err != nil {
		return nil, err
	}
	return ConvertSearchResultToGraphQL(res), nil
}

func (r *Resolvers) AdvancedSearchResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	params := search.SearchParams{
		Query:    getStringArg(p, "query"),
		Status:   getStringArg(p, "status"),
		FromDate: getStringArg(p, "fromDate"),
		ToDate:   getStringArg(p, "toDate"),
		Page:     getIntArg(p, "page", 1),
		Limit:    getIntArg(p, "limit", 10),
		SortBy:   getStringArg(p, "sortBy"),
		OrderBy:  getStringArg(p, "orderBy"),
	}
	if arr, ok := p.Args["contentTypeIds"].([]interface{}); ok {
		for _, v := range arr {
			if n, ok := v.(int); ok {
				params.ContentTypeIDs = append(params.ContentTypeIDs, uint(n))
			}
		}
	}
	if arr, ok := p.Args["fields"].([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				params.Fields = append(params.Fields, s)
			}
		}
	}
	if arr, ok := p.Args["tags"].([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				params.Tags = append(params.Tags, s)
			}
		}
	}
	var result *search.SearchResult
	if filters, ok := p.Args["filters"]; ok && filters != nil {
		if m, ok := filters.(map[string]interface{}); ok {
			r1, err := search.AdvancedFilter(m, params)
			if err != nil {
				return nil, err
			}
			result = r1
		}
	}
	if result == nil {
		r1, err := search.FullTextSearch(params)
		if err != nil {
			return nil, err
		}
		result = r1
	}
	return ConvertSearchResultToGraphQL(result), nil
}

func getStringArg(p graphql.ResolveParams, key string) string {
	if v, ok := p.Args[key].(string); ok {
		return v
	}
	return ""
}
func getIntArg(p graphql.ResolveParams, key string, def int) int {
	if v, ok := p.Args[key].(int); ok {
		return v
	}
	return def
}

func ConvertSearchResultToGraphQL(sr *search.SearchResult) map[string]interface{} {
	out := map[string]interface{}{
		"total":           sr.Total,
		"page":            sr.Page,
		"limit":           sr.Limit,
		"totalPages":      sr.TotalPages,
		"hasNextPage":     sr.Page < int(sr.TotalPages),
		"hasPreviousPage": sr.Page > 1,
		"query":           sr.Query,
	}
	entries := make([]map[string]interface{}, len(sr.Entries))
	for i, e := range sr.Entries {
		entries[i] = ConvertContentEntryToGraphQL(&e)
	}
	out["entries"] = entries
	return out
}
func (r *Resolvers) ContentRelationsResolver(p graphql.ResolveParams) (interface{}, error) {

	userID, err := r.getUserIDFromContext(p)

	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	fromID := uint(p.Args["fromContentId"].(int))
	var relations []models.ContentRelation
	if err := r.DB.Where("from_content_id = ?", fromID).Find(&relations).Error; err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(relations))
	for i, rel := range relations {
		result[i] = ConvertContentRelationToGraphQL(&rel)
	}
	return result, nil
}

func (r *Resolvers) CreateContentRelationResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}
	fromIDStr := p.Args["fromContentId"].(string)
	fromID64, err := strconv.ParseUint(fromIDStr, 10, 32)
	if err != nil {
		return nil, err
	}
	toID := uint(p.Args["toContentId"].(int))
	relationType := p.Args["relationType"].(string)
	rel := &models.ContentRelation{
		FromContentID: uint(fromID64),
		ToContentID:   toID,
		RelationType:  relationType,
	}
	if err := r.DB.Create(rel).Error; err != nil {
		return nil, err
	}
	return ConvertContentRelationToGraphQL(rel), nil
}

func (r *Resolvers) DeleteContentRelationResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "delete") {
		return nil, fmt.Errorf("permission denied")
	}
	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, err
	}
	if err := r.DB.Delete(&models.ContentRelation{}, uint(id64)).Error; err != nil {
		return nil, err
	}
	return true, nil
}

func (r *Resolvers) DuplicateRoleResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "Role", "create") {
		return nil, fmt.Errorf("permission denied")
	}
	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, err
	}
	var original models.Role
	if err := r.DB.Preload("Permissions").First(&original, uint(id64)).Error; err != nil {
		return nil, err
	}
	newName := original.Name + " Copy"
	if v, ok := p.Args["name"].(string); ok && v != "" {
		newName = v
	}
	newRole := models.Role{Name: newName, Description: original.Description + " (Copy)"}
	if err := r.DB.Create(&newRole).Error; err != nil {
		return nil, err
	}
	for _, perm := range original.Permissions {
		np := models.Permission{
			RoleID:         newRole.ID,
			Module:         perm.Module,
			Action:         perm.Action,
			FieldScope:     perm.FieldScope,
			AllowedFields:  perm.AllowedFields,
			DeniedFields:   perm.DeniedFields,
			ContentTypeIDs: perm.ContentTypeIDs,
		}
		if err := r.DB.Create(&np).Error; err != nil {
			return nil, err
		}
	}
	if err := r.DB.Preload("Permissions").First(&newRole, newRole.ID).Error; err != nil {
		return nil, err
	}
	return ConvertRoleToGraphQL(&newRole), nil
}

func (r *Resolvers) AssignRoleToUserResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "Role", "update") {
		return nil, fmt.Errorf("permission denied")
	}
	userIdStr := p.Args["userId"].(string)
	roleIdStr := p.Args["roleId"].(string)
	uid64, err := strconv.ParseUint(userIdStr, 10, 32)
	if err != nil {
		return nil, err
	}
	rid64, err := strconv.ParseUint(roleIdStr, 10, 32)
	if err != nil {
		return nil, err
	}
	var role models.Role
	if err := r.DB.First(&role, uint(rid64)).Error; err != nil {
		return nil, err
	}
	var user models.User
	if err := r.DB.First(&user, uint(uid64)).Error; err != nil {
		return nil, err
	}
	user.RoleID = uint(rid64)
	if err := r.DB.Save(&user).Error; err != nil {
		return nil, err
	}
	return true, nil
}

// =====================
// Workflow Resolvers
// =====================
func (r *Resolvers) WorkflowHistoryResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	hist, err := workflow.GetWorkflowHistory(entryId)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(hist))
	for i := range hist {
		out[i] = ConvertWorkflowHistoryToGraphQL(&hist[i])
	}
	return out, nil
}

func (r *Resolvers) WorkflowCommentsResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	includePrivate := false
	if v, ok := p.Args["includePrivate"].(bool); ok {
		includePrivate = v
	}
	comments, err := workflow.GetWorkflowComments(entryId, includePrivate)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(comments))
	for i := range comments {
		out[i] = ConvertWorkflowCommentToGraphQL(&comments[i])
	}
	return out, nil
}

func (r *Resolvers) WorkflowAssignmentsResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	status := ""
	if v, ok := p.Args["status"].(string); ok {
		status = v
	}
	list, err := workflow.GetMyAssignments(userID, status)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, len(list))
	for i := range list {
		out[i] = ConvertWorkflowAssignmentToGraphQL(&list[i])
	}
	return out, nil
}

func (r *Resolvers) WorkflowStatsResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	ctID := uint(p.Args["contentTypeId"].(int))
	stats, err := workflow.GetWorkflowStatistics(ctID)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *Resolvers) ChangeContentStatusResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	status := p.Args["status"].(string)
	comment := getStringArg(p, "comment")
	_, err = workflow.ChangeWorkflowStatus(entryId, userID, status, comment)
	if err != nil {
		return nil, err
	}
	hist, err := workflow.GetWorkflowHistory(entryId)
	if err != nil || len(hist) == 0 {
		return nil, fmt.Errorf("failed to fetch workflow history")
	}
	return ConvertWorkflowHistoryToGraphQL(&hist[0]), nil
}

func (r *Resolvers) RequestReviewResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	comment := getStringArg(p, "comment")
	_, err = workflow.RequestReview(entryId, userID, comment)
	if err != nil {
		return nil, err
	}
	hist, err := workflow.GetWorkflowHistory(entryId)
	if err != nil || len(hist) == 0 {
		return nil, fmt.Errorf("failed to fetch workflow history")
	}
	return ConvertWorkflowHistoryToGraphQL(&hist[0]), nil
}

func (r *Resolvers) ApproveEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "approve") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	comment := getStringArg(p, "comment")
	_, err = workflow.ApproveEntry(entryId, userID, comment)
	if err != nil {
		return nil, err
	}
	hist, err := workflow.GetWorkflowHistory(entryId)
	if err != nil || len(hist) == 0 {
		return nil, fmt.Errorf("failed to fetch workflow history")
	}
	return ConvertWorkflowHistoryToGraphQL(&hist[0]), nil
}

func (r *Resolvers) RejectEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "approve") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	comment := getStringArg(p, "comment")
	_, err = workflow.RejectEntry(entryId, userID, comment)
	if err != nil {
		return nil, err
	}
	hist, err := workflow.GetWorkflowHistory(entryId)
	if err != nil || len(hist) == 0 {
		return nil, fmt.Errorf("failed to fetch workflow history")
	}
	return ConvertWorkflowHistoryToGraphQL(&hist[0]), nil
}

func (r *Resolvers) PublishEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "approve") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	comment := getStringArg(p, "comment")
	_, err = workflow.PublishEntry(entryId, userID, comment)
	if err != nil {
		return nil, err
	}
	hist, err := workflow.GetWorkflowHistory(entryId)
	if err != nil || len(hist) == 0 {
		return nil, fmt.Errorf("failed to fetch workflow history")
	}
	return ConvertWorkflowHistoryToGraphQL(&hist[0]), nil
}

func (r *Resolvers) AddWorkflowCommentResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	comment := p.Args["comment"].(string)
	isPrivate := false
	if v, ok := p.Args["isPrivate"].(bool); ok {
		isPrivate = v
	}
	wc, err := workflow.AddWorkflowComment(entryId, userID, comment, isPrivate)
	if err != nil {
		return nil, err
	}
	return ConvertWorkflowCommentToGraphQL(wc), nil
}

func (r *Resolvers) AssignEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "approve") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	assignedTo := uint(p.Args["assignedTo"].(int))
	var dueDatePtr *time.Time
	if v, ok := p.Args["dueDate"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dueDatePtr = &t
		}
	}
	wa, err := workflow.AssignEntry(entryId, assignedTo, userID, dueDatePtr)
	if err != nil {
		return nil, err
	}
	return ConvertWorkflowAssignmentToGraphQL(wa), nil
}

// =====================
// Content Field Resolvers
// =====================
func (r *Resolvers) AddContentFieldResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}
	ctID := uint(p.Args["contentTypeId"].(int))
	name := p.Args["name"].(string)
	ftype := p.Args["type"].(string)
	required := false
	if v, ok := p.Args["required"].(bool); ok {
		required = v
	}
	isSEO := false
	if v, ok := p.Args["isSeo"].(bool); ok {
		isSEO = v
	}
	field, err := content.AddFieldToContentType(ctID, name, ftype, required, isSEO)
	if err != nil {
		return nil, err
	}
	if v, ok := p.Args["unique"].(bool); ok {
		field.Unique = v
	}
	if v, ok := p.Args["maxLength"].(int); ok {
		field.MaxLength = &v
	}
	if v, ok := p.Args["minLength"].(int); ok {
		field.MinLength = &v
	}
	if v, ok := p.Args["pattern"].(string); ok {
		field.Pattern = v
	}
	if v, ok := p.Args["minValue"].(float64); ok {
		field.MinValue = &v
	}
	if v, ok := p.Args["maxValue"].(float64); ok {
		field.MaxValue = &v
	}
	if v, ok := p.Args["defaultValue"].(string); ok {
		field.DefaultValue = v
	}
	if v, ok := p.Args["placeholder"].(string); ok {
		field.Placeholder = v
	}
	if v, ok := p.Args["helpText"].(string); ok {
		field.HelpText = v
	}
	if err := r.DB.Save(field).Error; err != nil {
		return nil, err
	}
	return ConvertContentFieldToGraphQL(field), nil
}

func (r *Resolvers) UpdateContentFieldResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "update") {
		return nil, fmt.Errorf("permission denied")
	}
	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, err
	}
	var field models.ContentField
	if err := r.DB.First(&field, uint(id64)).Error; err != nil {
		return nil, err
	}
	if v, ok := p.Args["name"].(string); ok {
		field.Name = v
	}
	if v, ok := p.Args["type"].(string); ok {
		field.Type = v
	}
	if v, ok := p.Args["required"].(bool); ok {
		field.Required = v
	}
	if v, ok := p.Args["isSeo"].(bool); ok {
		field.IsSEO = v
	}
	if v, ok := p.Args["unique"].(bool); ok {
		field.Unique = v
	}
	if v, ok := p.Args["maxLength"].(int); ok {
		field.MaxLength = &v
	}
	if v, ok := p.Args["minLength"].(int); ok {
		field.MinLength = &v
	}
	if v, ok := p.Args["pattern"].(string); ok {
		field.Pattern = v
	}
	if v, ok := p.Args["minValue"].(float64); ok {
		field.MinValue = &v
	}
	if v, ok := p.Args["maxValue"].(float64); ok {
		field.MaxValue = &v
	}
	if v, ok := p.Args["defaultValue"].(string); ok {
		field.DefaultValue = v
	}
	if v, ok := p.Args["placeholder"].(string); ok {
		field.Placeholder = v
	}
	if v, ok := p.Args["helpText"].(string); ok {
		field.HelpText = v
	}
	if err := r.DB.Save(&field).Error; err != nil {
		return nil, err
	}
	return ConvertContentFieldToGraphQL(&field), nil
}

func (r *Resolvers) DeleteContentFieldResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "delete") {
		return nil, fmt.Errorf("permission denied")
	}
	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, err
	}
	if err := r.DB.Delete(&models.ContentField{}, uint(id64)).Error; err != nil {
		return nil, err
	}
	return true, nil
}

// =====================
// SEO Preview & Search Helpers
// =====================
func (r *Resolvers) SEOPreviewResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "SEO", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	entryId := uint(p.Args["entryId"].(int))
	data, err := content.GenerateSEOPreview(entryId)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *Resolvers) SearchFacetsResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	params := search.SearchParams{
		Query: getStringArg(p, "query"),
	}
	if arr, ok := p.Args["contentTypeIds"].([]interface{}); ok {
		for _, v := range arr {
			if n, ok := v.(int); ok {
				params.ContentTypeIDs = append(params.ContentTypeIDs, uint(n))
			}
		}
	}
	facets, err := search.GetSearchFacets(params)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"content_types": facets.ContentTypes,
		"statuses":      facets.Statuses,
		"date_range":    facets.DateRange,
	}
	return out, nil
}

func (r *Resolvers) AutocompleteResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}
	if !r.checkPermission(userID, "ContentEntry", "read") {
		return nil, fmt.Errorf("permission denied")
	}
	field := p.Args["field"].(string)
	prefix := p.Args["prefix"].(string)
	ctID := uint(p.Args["contentTypeId"].(int))
	limit := 10
	if v, ok := p.Args["limit"].(int); ok {
		limit = v
	}
	return search.AutoComplete(field, prefix, ctID, limit)
}

// =====================
// Project Resolvers
// =====================

func (r *Resolvers) ProjectsResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	projects, err := project.ListProjects(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	result := make([]map[string]interface{}, len(projects))
	for i, proj := range projects {
		result[i] = ConvertProjectToGraphQL(&proj)
	}
	return result, nil
}

func (r *Resolvers) ProjectResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	proj, err := project.GetProject(uint(id64))
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Check if user is a member
	if _, err := project.GetProjectMember(uint(id64), userID); err != nil {
		return nil, fmt.Errorf("permission denied: you are not a member of this project")
	}

	return ConvertProjectToGraphQL(proj), nil
}

func (r *Resolvers) ProjectMembersResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	projectIDStr := p.Args["projectId"].(string)
	projectID64, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	projectID := uint(projectID64)

	// Check if user is a member
	if _, err := project.GetProjectMember(projectID, userID); err != nil {
		return nil, fmt.Errorf("permission denied: you are not a member of this project")
	}

	members, err := project.ListProjectMembers(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project members: %w", err)
	}

	result := make([]map[string]interface{}, len(members))
	for i, member := range members {
		result[i] = ConvertProjectMemberToGraphQL(&member)
	}
	return result, nil
}

func (r *Resolvers) CreateProjectResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	name := p.Args["name"].(string)
	description := ""
	if v, ok := p.Args["description"].(string); ok {
		description = v
	}

	proj, err := project.CreateProject(name, description, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return ConvertProjectToGraphQL(proj), nil
}

func (r *Resolvers) UpdateProjectResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	projectID := uint(id64)

	// Check if user has permission (owner or admin)
	if !project.HasProjectPermission(projectID, userID, models.ProjectRoleAdmin) {
		return nil, fmt.Errorf("permission denied: only project owners and admins can update project")
	}

	name := p.Args["name"].(string)
	description := ""
	if v, ok := p.Args["description"].(string); ok {
		description = v
	}

	proj, err := project.UpdateProject(projectID, name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return ConvertProjectToGraphQL(proj), nil
}

func (r *Resolvers) DeleteProjectResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	idStr := p.Args["id"].(string)
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	projectID := uint(id64)

	// Check if user is owner
	if !project.HasProjectPermission(projectID, userID, "owner") {
		return nil, fmt.Errorf("permission denied: only project owner can delete project")
	}

	if err := project.DeleteProject(projectID); err != nil {
		return nil, fmt.Errorf("failed to delete project: %w", err)
	}

	return true, nil
}

func (r *Resolvers) AddProjectMemberResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	projectIDStr := p.Args["projectId"].(string)
	projectID64, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	projectID := uint(projectID64)

	userIDStr := p.Args["userId"].(string)
	userID64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	memberUserID := uint(userID64)

	role := p.Args["role"].(string)

	member, err := project.AddProjectMember(projectID, memberUserID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to add project member: %w", err)
	}

	// Member already has Role preloaded from AddProjectMember
	return ConvertProjectMemberToGraphQL(member), nil
}

func (r *Resolvers) UpdateProjectMemberRoleResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	projectIDStr := p.Args["projectId"].(string)
	projectID64, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	projectID := uint(projectID64)

	memberIDStr := p.Args["memberId"].(string)
	memberID64, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid member ID: %w", err)
	}
	memberID := uint(memberID64)

	role := p.Args["role"].(string)

	member, err := project.UpdateProjectMemberRole(projectID, memberID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to update project member role: %w", err)
	}

	// Member already has Role preloaded from UpdateProjectMemberRole
	return ConvertProjectMemberToGraphQL(member), nil
}

func (r *Resolvers) RemoveProjectMemberResolver(p graphql.ResolveParams) (interface{}, error) {
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	projectIDStr := p.Args["projectId"].(string)
	projectID64, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	projectID := uint(projectID64)

	memberIDStr := p.Args["memberId"].(string)
	memberID64, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid member ID: %w", err)
	}
	memberID := uint(memberID64)

	if err := project.RemoveProjectMember(projectID, memberID, userID); err != nil {
		return nil, fmt.Errorf("failed to remove project member: %w", err)
	}

	return true, nil
}
