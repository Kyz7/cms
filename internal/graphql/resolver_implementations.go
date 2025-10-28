package graphql

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Kyz7/cms/internal/models"
	"github.com/graphql-go/graphql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

// Authentication Resolvers
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

	// Find user by email
	var user models.User
	if err := r.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens (simplified)
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

	// Create user
	user := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
		RoleID:   2, // Default role
		Status:   "active",
		Provider: "local",
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Password = string(hash)

	// Save user
	if err := r.DB.Create(user).Error; err != nil {
		return nil, err
	}

	// Generate tokens (simplified)
	token := "jwt_token_here"
	refreshToken := "refresh_token_here"

	return map[string]interface{}{
		"token":        token,
		"refreshToken": refreshToken,
		"user":         ConvertUserToGraphQL(user),
	}, nil
}

// User Resolvers
func (r *Resolvers) UsersResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "User", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	var users []models.User
	query := r.DB.Preload("Role")

	// Apply pagination
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
	// Check permissions
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
	// Check permissions
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

	// Create user
	newUser := &models.User{
		Name:     name,
		Email:    email,
		Password: password,
		RoleID:   roleID,
		Status:   "active",
		Provider: "local",
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	newUser.Password = string(hash)

	// Save user
	if err := r.DB.Create(newUser).Error; err != nil {
		return nil, err
	}

	return ConvertUserToGraphQL(newUser), nil
}

func (r *Resolvers) UpdateUserResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Find user
	var user models.User
	if err := r.DB.First(&user, id).Error; err != nil {
		return nil, err
	}

	// Update fields
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

	// Save user
	if err := r.DB.Save(&user).Error; err != nil {
		return nil, err
	}

	return ConvertUserToGraphQL(&user), nil
}

func (r *Resolvers) DeleteUserResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Delete user
	if err := r.DB.Delete(&models.User{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

// Role Resolvers
func (r *Resolvers) RolesResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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
	// Check permissions
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
	// Check permissions
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

	// Create role
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
	// Check permissions
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

	// Find role
	var role models.Role
	if err := r.DB.First(&role, id).Error; err != nil {
		return nil, err
	}

	// Update fields
	if name, ok := p.Args["name"].(string); ok {
		role.Name = name
	}
	if description, ok := p.Args["description"].(string); ok {
		role.Description = description
	}

	// Save role
	if err := r.DB.Save(&role).Error; err != nil {
		return nil, err
	}

	return ConvertRoleToGraphQL(&role), nil
}

func (r *Resolvers) DeleteRoleResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Delete role
	if err := r.DB.Delete(&models.Role{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

// Content Type Resolvers
func (r *Resolvers) ContentTypesResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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
	// Check permissions
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
	// Check permissions
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

	// Create content type
	contentType := &models.ContentType{
		Name:      name,
		Slug:      slug,
		EnableSEO: enableSeo,
	}

	if err := r.DB.Create(contentType).Error; err != nil {
		return nil, err
	}

	return ConvertContentTypeToGraphQL(contentType), nil
}

func (r *Resolvers) UpdateContentTypeResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Build update data
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

	// Find content type
	var contentType models.ContentType
	if err := r.DB.First(&contentType, id).Error; err != nil {
		return nil, err
	}

	// Update fields
	if name, ok := p.Args["name"].(string); ok {
		contentType.Name = name
	}
	if slug, ok := p.Args["slug"].(string); ok {
		contentType.Slug = slug
	}
	if enableSeo, ok := p.Args["enableSeo"].(bool); ok {
		contentType.EnableSEO = enableSeo
	}

	// Save content type
	if err := r.DB.Save(&contentType).Error; err != nil {
		return nil, err
	}

	return ConvertContentTypeToGraphQL(&contentType), nil
}

func (r *Resolvers) DeleteContentTypeResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Delete content type
	if err := r.DB.Delete(&models.ContentType{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

// Content Entry Resolvers
func (r *Resolvers) ContentEntriesResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Apply pagination
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
	// Check permissions
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

	return ConvertContentEntryToGraphQL(&entry), nil
}

func (r *Resolvers) CreateContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create content entry
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

	return ConvertContentEntryToGraphQL(entry), nil
}

func (r *Resolvers) UpdateContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Build update data
	updateData := make(map[string]interface{})
	if data, ok := p.Args["data"]; ok {
		updateData["data"] = data
	}
	if status, ok := p.Args["status"].(string); ok {
		updateData["status"] = status
	}

	// Find content entry
	var entry models.ContentEntry
	if err := r.DB.First(&entry, id).Error; err != nil {
		return nil, err
	}

	// Update fields
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

	// Save content entry
	if err := r.DB.Save(&entry).Error; err != nil {
		return nil, err
	}

	return ConvertContentEntryToGraphQL(&entry), nil
}

func (r *Resolvers) DeleteContentEntryResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
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

	// Delete content entry
	if err := r.DB.Delete(&models.ContentEntry{}, id).Error; err != nil {
		return nil, err
	}

	return true, nil
}

// Media Resolvers
func (r *Resolvers) MediaFilesResolver(p graphql.ResolveParams) (interface{}, error) {
	// Check permissions
	userID, err := r.getUserIDFromContext(p)
	if err != nil {
		return nil, err
	}

	if !r.checkPermission(userID, "Media", "read") {
		return nil, fmt.Errorf("permission denied")
	}

	var mediaFiles []models.MediaFile
	query := r.DB.Preload("Uploader")

	// Apply filters
	if folder, ok := p.Args["folder"].(string); ok {
		query = query.Where("folder = ?", folder)
	}

	// Apply pagination
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
	// Check permissions
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
