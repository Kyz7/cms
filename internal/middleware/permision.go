package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ProjectIDOnly struct {
	ProjectID *uint `json:"project_id"`
}

const (
	ContentModule        Module = "ContentEntry"
	ProjectContentModule Module = "ProjectContent"
	ProjectSchemaModule  Module = "ProjectSchema"
	MediaModule          Module = "Media"
	SEOModule            Module = "SEO"
	CreateAction         Action = "create"
	ReadAction           Action = "read"
	UpdateAction         Action = "update"
	DeleteAction         Action = "delete"
	ApproveAction        Action = "approve"
	UploadAction         Action = "upload"
)

func GetProjectIDByContentTypeID(contentTypeID uint) (uint, error) {
	var ct models.ContentType
	// Hanya ambil ProjectID-nya saja
	result := database.DB.Select("project_id").First(&ct, contentTypeID)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return 0, errors.New("content type not found")
		}
		return 0, result.Error
	}

	// Jika ProjectID adalah pointer yang nil (Global Content Type), kembalikan 0
	if ct.ProjectID == nil {
		return 0, nil
	}

	// Kembalikan ProjectID
	return *ct.ProjectID, nil
}

func checkGlobalPermission(user *models.User, module string, action string) bool {
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

func PermissionProtected(module string, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)
		projectID := detectProjectID(c) // ⭐️ Langkah 1: Deteksi dari Request/URL
		contentTypeID := uint(0)

		// --- Langkah 2: Deteksi ProjectID melalui Lookup DB (hanya jika ProjectID masih 0) ---

		if projectID == 0 {
			// A. Deteksi dari Content Type ID
			ctIDInt, err := c.ParamsInt("id")
			if err != nil {
				ctIDInt, err = c.ParamsInt("content_type_id")
			}

			// B. Deteksi dari Field ID
			fieldIDInt, fieldIDErr := c.ParamsInt("field_id")
			if fieldIDErr == nil && fieldIDInt > 0 {
				var field models.ContentField
				if dbErr := database.DB.Select("content_type_id").First(&field, fieldIDInt).Error; dbErr == nil {
					ctIDInt = int(field.ContentTypeID)
				}
			}

			if ctIDInt > 0 {
				ctID := uint(ctIDInt)
				contentTypeID = ctID

				retrievedProjectID, dbErr := GetProjectIDByContentTypeID(ctID)

				if dbErr == nil && retrievedProjectID > 0 {
					projectID = retrievedProjectID
					log.Printf("AUTH CHECK SUCCESS: ProjectID %d retrieved from ContentTypeID %d", projectID, ctID)
				}
			}
		}

		// C. Deteksi ProjectID dari Entry ID di URL (jika projectID masih 0)
		if projectID == 0 {
			entryIDInt, err := c.ParamsInt("entry_id")
			if err == nil && entryIDInt > 0 {
				var entry models.ContentEntry
				// Preload ContentType untuk mendapatkan ProjectID
				if err := database.DB.Preload("ContentType").First(&entry, entryIDInt).Error; err == nil {
					contentTypeID = entry.ContentTypeID
					if entry.ContentType.ProjectID != nil && *entry.ContentType.ProjectID > 0 {
						projectID = *entry.ContentType.ProjectID
						log.Printf("AUTH CHECK SUCCESS: ProjectID %d retrieved from EntryID %d", projectID, entryIDInt)
					}
				} else {
					log.Printf("AUTH CHECK WARNING: ContentEntry ID %d not found or DB error: %v", entryIDInt, err)
				}
			}
		}

		// Simpan ProjectID dan ContentTypeID ke Locals
		c.Locals("project_id", projectID)
		if contentTypeID > 0 {
			c.Locals("content_type_id", contentTypeID)
		}

		// --- Langkah 3: Tentukan Module yang Diperiksa (Global vs. Project Module) ---

		currentModule := module
		if projectID > 0 {
			// Jika ada Project ID, override module untuk Project Role
			if module == string(ContentModule) {
				currentModule = string(ProjectContentModule)
			} else if module == "ContentType" || module == "ContentField" {
				currentModule = string(ProjectSchemaModule)
			}
		}
		c.Locals("module", currentModule)

		log.Printf("AUTH CHECK: UserID=%d, Module=%s, Action=%s, ProjectID=%d", userID, currentModule, action, projectID)

		// --- Langkah 4: Cek Izin ---

		var user models.User
		if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
			return response.Unauthorized(c, "Unauthorized: User account not found")
		}

		// Bypass Admin
		if IsFullAccessRole(&user) {
			return c.Next()
		}

		hasPermission := false

		if projectID == 0 {
			// A. KONTEKS GLOBAL
			hasPermission = HasGlobalPermission(userID, currentModule, action)
		} else {
			// B. KONTEKS PROYEK

			// 4.1 Prioritas 1: Cek Izin Proyek (Project Role)
			if CheckProjectPermissionByModuleAction(projectID, userID, currentModule, action) {
				hasPermission = true
			}

			// 4.2 Prioritas 2: Peran Global Fallback (misalnya Global Schema Editor)
			if !hasPermission && (currentModule == string(ProjectSchemaModule) || currentModule == string(ProjectContentModule)) {
				// Cek apakah user memiliki izin Global yang setara (misalnya ContentType:read)
				globalEquivalentModule := strings.TrimPrefix(currentModule, "Project")
				if HasGlobalPermission(userID, globalEquivalentModule, action) {
					hasPermission = true
				}
			}
		}

		// --- FINAL CHECK ---
		if !hasPermission {
			return response.Forbidden(c, "You don't have permission to perform this action")
		}

		return c.Next()
	}
}

func CheckProjectPermissionByModuleAction(projectID, userID uint, module, action string) bool {
	var member models.ProjectMember

	// 1. Ambil ProjectMember (hanya butuh RoleID)
	err := database.DB.
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, "active").
		First(&member).Error

	if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
		log.Printf("DEBUG PERM: Project Member Check Failed. P:%d, U:%d, Err: %v", projectID, userID, err)
		return false
	}

	// 2. Cek Izin di Tabel Permission
	var count int64
	// Cari apakah ada baris Permission yang cocok dengan RoleID ProjectMember, Module, dan Action
	err = database.DB.Model(&models.Permission{}).
		Where("role_id = ? AND module = ? AND action = ?", member.RoleID, module, action).
		Count(&count).Error

	if err != nil {
		log.Printf("DEBUG PERM: Permission Count DB Error for RoleID %d: %v", member.RoleID, err)
		return false
	}

	log.Printf("DEBUG PERM: Project Check Result: RoleID %d, M:%s, A:%s, Count: %d", member.RoleID, module, action, count)

	return count > 0 // Jika count > 0, izin ditemukan
}

func HasPermission(userID uint, module, action string) bool {
	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return false
	}

	if user.Role == nil {
		return false
	}

	// Admin bypass - admin has full access to all resources
	if IsFullAccessRole(&user) {
		return true
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

	// Admin bypass - admin has full access to all resources
	if IsFullAccessRole(&user) {
		return true
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

	// Logika pemeriksaan izin penuh lainnya (tetap sama)
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

func FilterFieldsByPermission(userID uint, action string, data map[string]interface{}, contentTypeID uint, projectID uint) (map[string]interface{}, error) {

	// --- 1. Ambil Pengguna Global & Bypass Admin ---
	var user models.User
	if err := database.DB.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return nil, fiber.NewError(401, "Unauthorized: User account not found")
	}

	if IsFullAccessRole(&user) {
		fmt.Printf("[PERM DEBUG] User %d is Full Access. Bypassing filter.\n", userID)
		return data, nil
	}

	var userPermission *models.Permission
	var memberRoleID uint

	fmt.Printf("[PERM DEBUG] Starting Filter. UserID: %d, Action: %s, CTID: %d, ProjectID: %d\n", userID, action, contentTypeID, projectID)

	// --- 2. Tentukan Izin Utama (Global vs. Proyek) ---

	if projectID > 0 {
		// KONTEKS PROYEK
		projectModule := "ProjectContent"

		var err error
		memberRoleID, err = GetProjectMemberRoleID(userID, projectID)

		if err != nil {
			fmt.Printf("[PERM DEBUG] Failed to get Project Member Role ID for Project %d: %v\n", projectID, err)
		}

		if memberRoleID > 0 {
			fmt.Printf("[PERM DEBUG] Found Project Member Role ID: %d\n", memberRoleID)

			// Mengambil izin Project Content
			userPermission, err = GetPermissionByRoleID(memberRoleID, projectModule, action)

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Printf("[PERM DEBUG] ❌ Project Permission (%s:%s) NOT FOUND. Checking Global Role...\n", projectModule, action)
				} else {
					return nil, fmt.Errorf("database error checking project permission: %w", err)
				}
			} else {
				fmt.Printf("[PERM DEBUG] ✅ Project Permission FOUND. Scope: %s\n", userPermission.FieldScope)
			}
		} else {
			fmt.Println("[PERM DEBUG] ❌ User is not an ACTIVE member of the project (RoleID 0). Checking Global Role...")
		}
	}

	// Jika belum ada izin dari Proyek, cek Global Role
	if userPermission == nil {
		// KONTEKS GLOBAL
		for _, perm := range user.Role.Permissions {
			if perm.Module == "ContentEntry" && perm.Action == action {

				// Pengecekan ContentTypeIDs
				if perm.ContentTypeIDs != nil {
					var contentTypeIDs []uint
					json.Unmarshal(perm.ContentTypeIDs, &contentTypeIDs)

					hasAccess := false
					for _, id := range contentTypeIDs {
						if id == contentTypeID {
							hasAccess = true
							break
						}
					}
					if !hasAccess {
						fmt.Printf("[PERM DEBUG] Global Role found but CTID %d is NOT listed in Allowed IDs.\n", contentTypeID)
						continue
					}
				}

				userPermission = &perm
				fmt.Printf("[PERM DEBUG] ✅ Global Permission FOUND. Scope: %s\n", userPermission.FieldScope)
				break
			}
		}
	}

	// --- 3. FINAL CHECK OTORISASI ---

	if userPermission == nil {
		fmt.Println("[PERM DEBUG] 🛑 Final Decision: NO WRITE PERMISSION FOUND (Project/Global).")
		return nil, fiber.NewError(403, "No write permission for this content type (Global or Project)")
	}

	// --- 4. Load Content Type & Fields ---

	if userPermission.FieldScope == "" {
		userPermission.FieldScope = "all"
	}

	fmt.Printf("[PERM DEBUG] Proceeding to field filtering with Scope: %s\n", userPermission.FieldScope)

	var ct models.ContentType
	if err := database.DB.Preload("Fields").Preload("SEOFields").First(&ct, contentTypeID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve content type for field filtering: %w", err)
	}

	allFields := append(ct.Fields, ct.SEOFields...)
	filteredData := make(map[string]interface{})

	// --- 5. Implementasi Field-Scope Filtering (Logika tidak berubah) ---

	if userPermission.FieldScope == "custom" {

		var allowedFields []string
		var deniedFields []string

		// ... (Logika unmarshal dan filtering custom fields tetap sama)
		if userPermission.AllowedFields != nil {
			json.Unmarshal(userPermission.AllowedFields, &allowedFields)
		}
		if userPermission.DeniedFields != nil {
			json.Unmarshal(userPermission.DeniedFields, &deniedFields)
		}

		// Prioritas 1: Allowed Fields
		if len(allowedFields) > 0 {
			for k, v := range data {
				for _, allowedField := range allowedFields {
					if k == allowedField || strings.HasPrefix(k, allowedField+"_media_id") {
						filteredData[k] = v
						break
					}
				}
			}
		} else if len(deniedFields) > 0 {
			// Prioritas 2: Denied Fields
			for k, v := range data {
				isDenied := false
				for _, deniedField := range deniedFields {
					if k == deniedField || strings.HasPrefix(k, deniedField+"_media_id") {
						isDenied = true
						break
					}
				}
				if !isDenied {
					filteredData[k] = v
				}
			}
		}

	} else {
		// Scope 'all', 'seo_only', atau 'non_seo_only'
		for k, v := range data {
			for _, field := range allFields {
				if field.Name == k {

					switch userPermission.FieldScope {
					case "seo_only":
						if field.IsSEO {
							filteredData[k] = v
						}
					case "non_seo_only":
						if !field.IsSEO {
							filteredData[k] = v
						}
					default: // "all"
						filteredData[k] = v
					}
					break
				}
			}

			// Masukkan kembali media ID
			if strings.HasSuffix(k, "_media_id") {
				baseFieldName := strings.TrimSuffix(k, "_media_id")
				if _, ok := filteredData[baseFieldName]; ok {
					filteredData[k] = v
				}
			}
		}
	}

	// 6. Final Check
	if len(filteredData) == 0 && len(data) > 0 {
		fmt.Printf("[PERM DEBUG] 🛑 Final Check FAILED. Input fields (%d) were all filtered out.\n", len(data))
		return nil, fiber.NewError(403, "Your role is restricted from writing to all provided fields.")
	}

	fmt.Printf("[PERM DEBUG] ✅ Filtering successful. %d fields passed the scope check.\n", len(filteredData))
	return filteredData, nil
}

// Mengambil RoleID pengguna dalam proyek tertentu
func GetProjectMemberRoleID(userID, projectID uint) (uint, error) {
	var member models.ProjectMember
	err := database.DB.Select("role_id").
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, "active").
		First(&member).Error

	if err != nil {
		return 0, err // Mengembalikan error (termasuk gorm.ErrRecordNotFound)
	}
	return member.RoleID, nil
}

// Mengambil satu permission spesifik berdasarkan RoleID, Module, dan Action
func GetPermissionByRoleID(roleID uint, module, action string) (*models.Permission, error) {
	var perm models.Permission
	err := database.DB.
		Where("role_id = ? AND module = ? AND action = ?", roleID, module, action).
		First(&perm).Error

	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func detectProjectID(c *fiber.Ctx) uint {
	// 1. Cek dari URL Parameters (Paling Prioritas)
	if idParamInt, err := c.ParamsInt("project_id"); err == nil && idParamInt > 0 {
		log.Printf("AUTH DETECT: Found ProjectID %d from URL Path Param 'project_id'.", idParamInt)
		return uint(idParamInt)
	}

	// 2. Cek dari Query Parameters
	if qProjectID := c.Query("project_id"); qProjectID != "" {
		if id, err := strconv.ParseUint(qProjectID, 10, 64); err == nil && id > 0 {
			log.Printf("AUTH DETECT: Found ProjectID %d from Query Param 'project_id'.", id)
			return uint(id)
		}
	}

	// 3. Cek dari Request Body (Hanya untuk POST/PUT/PATCH)
	if c.Method() == fiber.MethodPost || c.Method() == fiber.MethodPut || c.Method() == fiber.MethodPatch {
		bodyBytes := c.Body()
		if len(bodyBytes) > 0 {
			var body ProjectIDOnly
			// Unmarshal body secara non-invasif (hanya untuk ProjectID)
			if err := json.Unmarshal(bodyBytes, &body); err == nil && body.ProjectID != nil && *body.ProjectID > 0 {
				log.Printf("AUTH DETECT: Found ProjectID %d from Request Body 'project_id'.", *body.ProjectID)
				return *body.ProjectID
			}
		}
	}

	return 0
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

func HasGlobalPermission(userID uint, module, action string) bool {
	var count int64

	// Perbaikan: Menggunakan tabel users dan kolom users.role_id
	database.DB.Raw(`
        SELECT COUNT(p.id)
        FROM users u                           
        JOIN roles r ON r.id = u.role_id       
        JOIN permissions p ON p.role_id = r.id
        WHERE u.id = ? 
          AND r.is_global = TRUE               -- Filter Role Global
          AND p.module = ? 
          AND p.action = ?
    `, userID, module, action).Scan(&count)

	return count > 0
}
func GetAccessibleProjectIDs(userID uint, module, action string) []uint {
	var accessibleProjects []uint

	// Perbaikan: JOIN roles r ON r.id = pm.role_id
	database.DB.Raw(`
        SELECT DISTINCT pm.project_id 
        FROM project_members pm
        JOIN roles r ON r.id = pm.role_id      -- Menggunakan kolom pm.role_id yang benar
        JOIN permissions p ON p.role_id = r.id
        WHERE pm.user_id = ? 
          AND r.is_global = FALSE              -- Filter Project Role
          AND p.module = ? 
          AND p.action = ?
    `, userID, module, action).Scan(&accessibleProjects)

	return accessibleProjects
}

type Module string
type Action string
