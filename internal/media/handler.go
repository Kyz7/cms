package media

import (
	"database/sql"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/Kyz7/cms/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
)

var policy = bluemonday.UGCPolicy()

func sanitizeInput(input string) string {
	return policy.Sanitize(input)
}

func UploadMediaHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "File is required", nil)
	}
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if strings.HasPrefix(file.Header.Get("Content-Type"), "video/") {
		maxSize = int64(100 * 1024 * 1024) // 100MB for videos
	}

	if file.Size > maxSize {
		return response.BadRequest(c, "File too large", map[string]interface{}{
			"max_size_mb":  maxSize / (1024 * 1024),
			"file_size_mb": file.Size / (1024 * 1024),
		})
	}

	folder := c.FormValue("folder", "")
	alt := c.FormValue("alt", "")
	caption := c.FormValue("caption", "")
	tagsStr := c.FormValue("tags", "[]")

	var tags []string
	json.Unmarshal([]byte(tagsStr), &tags)

	url, err := utils.UploadFile(file)
	if err != nil {
		return response.InternalError(c, "Failed to upload file: "+err.Error())
	}

	mediaFile := models.MediaFile{
		FileName:   file.Filename,
		URL:        url,
		Type:       file.Header.Get("Content-Type"),
		Size:       file.Size,
		Folder:     folder,
		Alt:        alt,
		Caption:    caption,
		UploadedBy: userID,
	}

	if strings.HasPrefix(mediaFile.Type, "image/") {
		if width, height, err := getImageDimensions(file); err == nil {
			mediaFile.Width = &width
			mediaFile.Height = &height
		}
	}

	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		mediaFile.Tags = tagsJSON
	}

	if err := database.DB.Create(&mediaFile).Error; err != nil {
		utils.DeleteFile(url)
		return response.InternalError(c, "Failed to save media metadata")
	}

	database.DB.Preload("Uploader").First(&mediaFile, mediaFile.ID)

	return response.Created(c, mediaFile, "Media uploaded successfully")
}

func BulkUploadMediaHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	folder := c.FormValue("folder", "")

	form, err := c.MultipartForm()
	if err != nil {
		return response.BadRequest(c, "Invalid form data", err.Error())
	}

	files := form.File["files"]
	if len(files) == 0 {
		return response.BadRequest(c, "No files provided", nil)
	}

	uploadedFiles := []models.MediaFile{}
	errors := []map[string]string{}

	for _, file := range files {
		maxSize := int64(10 * 1024 * 1024) // 10MB
		if strings.HasPrefix(file.Header.Get("Content-Type"), "video/") {
			maxSize = int64(100 * 1024 * 1024) // 100MB for videos
		}

		if file.Size > maxSize {
			errors = append(errors, map[string]string{
				"filename": file.Filename,
				"error":    "file too large",
			})
			continue
		}
		url, err := utils.UploadFile(file)
		if err != nil {
			errors = append(errors, map[string]string{
				"filename": file.Filename,
				"error":    err.Error(),
			})
			continue
		}

		mediaFile := models.MediaFile{
			FileName:   file.Filename,
			URL:        url,
			Type:       file.Header.Get("Content-Type"),
			Size:       file.Size,
			Folder:     folder,
			UploadedBy: userID,
		}

		if strings.HasPrefix(mediaFile.Type, "image/") {
			if width, height, err := getImageDimensions(file); err == nil {
				mediaFile.Width = &width
				mediaFile.Height = &height
			}
		}

		if err := database.DB.Create(&mediaFile).Error; err != nil {
			errors = append(errors, map[string]string{
				"filename": file.Filename,
				"error":    "failed to save metadata",
			})
			continue
		}

		uploadedFiles = append(uploadedFiles, mediaFile)
	}

	result := fiber.Map{
		"uploaded": len(uploadedFiles),
		"failed":   len(errors),
		"files":    uploadedFiles,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	return response.Created(c, result, "Bulk upload completed")
}

func ListMediaHandler(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	mediaType := c.Query("type", "")
	folder := c.Query("folder", "")
	search := c.Query("search", "")

	offset := (page - 1) * limit

	var mediaFiles []models.MediaFile
	var total int64

	query := database.DB.Model(&models.MediaFile{})

	if mediaType != "" {
		query = query.Where("type LIKE ?", mediaType+"%")
	}

	if folder != "" {
		query = query.Where("folder = ?", folder)
	}

	if search != "" {
		query = query.Where("file_name LIKE ? OR alt LIKE ? OR caption LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	query.Preload("Uploader").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&mediaFiles)

	meta := response.CalculateMeta(page, limit, total)
	return response.SuccessWithMeta(c, mediaFiles, meta, "Media files retrieved successfully")
}

func GetMediaHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid media ID", nil)
	}

	var mediaFile models.MediaFile
	if err := database.DB.Preload("Uploader").First(&mediaFile, id).Error; err != nil {
		return response.NotFound(c, "Media")
	}

	return response.Success(c, mediaFile, "Media retrieved successfully")
}

func UpdateMediaHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid media ID", nil)
	}

	var mediaFile models.MediaFile
	if err := database.DB.First(&mediaFile, id).Error; err != nil {
		return response.NotFound(c, "Media")
	}

	var body struct {
		Alt     string   `json:"alt"`
		Caption string   `json:"caption"`
		Folder  string   `json:"folder"`
		Tags    []string `json:"tags"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	mediaFile.Alt = body.Alt
	mediaFile.Caption = sanitizeInput(body.Caption)
	mediaFile.Folder = body.Folder

	if len(body.Tags) > 0 {
		tagsJSON, _ := json.Marshal(body.Tags)
		mediaFile.Tags = tagsJSON
	}

	if err := database.DB.Save(&mediaFile).Error; err != nil {
		return response.InternalError(c, "Failed to update media")
	}

	return response.Success(c, mediaFile, "Media updated successfully")
}

func DeleteMediaHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid media ID", nil)
	}

	var mediaFile models.MediaFile
	if err := database.DB.First(&mediaFile, id).Error; err != nil {
		return response.NotFound(c, "Media")
	}

	if err := utils.DeleteFile(mediaFile.URL); err != nil {
		c.Append("X-Warning", "File deleted from database but may still exist in storage")
	}

	if err := database.DB.Delete(&mediaFile).Error; err != nil {
		return response.InternalError(c, "Failed to delete media")
	}

	return c.Status(204).JSON(fiber.Map{})
}

func SearchMediaHandler(c *fiber.Ctx) error {
	query := c.Query("q", "")
	if query == "" {
		return response.BadRequest(c, "Search query is required", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := (page - 1) * limit

	var mediaFiles []models.MediaFile
	var total int64

	dbQuery := database.DB.Model(&models.MediaFile{}).
		Where("file_name LIKE ? OR alt LIKE ? OR caption LIKE ?",
			"%"+query+"%", "%"+query+"%", "%"+query+"%")

	dbQuery.Count(&total)
	dbQuery.Preload("Uploader").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&mediaFiles)

	meta := response.CalculateMeta(page, limit, total)
	return response.SuccessWithMeta(c, mediaFiles, meta, "Search results retrieved successfully")
}

func GetMediaStatsHandler(c *fiber.Ctx) error {
	var stats struct {
		TotalFiles    int64            `json:"total_files"`
		TotalSize     int64            `json:"total_size_bytes"`
		ByType        map[string]int64 `json:"by_type"`
		RecentUploads int64            `json:"recent_uploads_24h"`
		StorageMode   string           `json:"storage_mode"`
	}

	database.DB.Model(&models.MediaFile{}).Count(&stats.TotalFiles)

	database.DB.Model(&models.MediaFile{}).
		Select("COALESCE(SUM(size), 0)").
		Row().Scan(&stats.TotalSize)

	stats.ByType = make(map[string]int64)

	dbName := database.DB.Dialector.Name()

	var rows *sql.Rows
	var err error

	if dbName == "sqlite" {
		var mediaFiles []models.MediaFile
		database.DB.Model(&models.MediaFile{}).Select("type").Find(&mediaFiles)

		for _, media := range mediaFiles {
			mediaType := strings.Split(media.Type, "/")[0]
			stats.ByType[mediaType]++
		}
	} else {
		rows, err = database.DB.Model(&models.MediaFile{}).
			Select("split_part(type, '/', 1) as media_type, COUNT(*) as count").
			Group("media_type").Rows()

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var mediaType string
				var count int64
				rows.Scan(&mediaType, &count)
				stats.ByType[mediaType] = count
			}
		}
	}

	database.DB.Model(&models.MediaFile{}).
		Where("created_at > ?", time.Now().Add(-24*time.Hour)).
		Count(&stats.RecentUploads)

	stats.StorageMode = utils.GetStorageMode()

	return response.Success(c, stats, "Media statistics retrieved successfully")
}

func CreateFolderHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var body struct {
		Name     string `json:"name"`
		ParentID *uint  `json:"parent_id,omitempty"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Name == "" {
		return response.ValidationError(c, map[string]string{
			"name": "folder name is required",
		})
	}

	path := "/" + body.Name
	if body.ParentID != nil {
		var parent models.MediaFolder
		if err := database.DB.First(&parent, *body.ParentID).Error; err != nil {
			return response.NotFound(c, "Parent folder")
		}
		path = parent.Path + "/" + body.Name
	}

	folder := models.MediaFolder{
		Name:      body.Name,
		Path:      path,
		ParentID:  body.ParentID,
		CreatedBy: userID,
	}

	if err := database.DB.Create(&folder).Error; err != nil {
		return response.InternalError(c, "Failed to create folder")
	}

	return response.Created(c, folder, "Folder created successfully")
}

func ListFoldersHandler(c *fiber.Ctx) error {
	var folders []models.MediaFolder
	if err := database.DB.Preload("Parent").Order("path").Find(&folders).Error; err != nil {
		return response.InternalError(c, "Failed to fetch folders")
	}

	return response.Success(c, folders, "Folders retrieved successfully")
}

func getImageDimensions(file *multipart.FileHeader) (int, int, error) {
	src, err := file.Open()
	if err != nil {
		return 0, 0, err
	}
	defer src.Close()

	img, _, err := image.DecodeConfig(src)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}
