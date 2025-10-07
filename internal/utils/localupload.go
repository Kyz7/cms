package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	UploadBasePath = "./uploads"
	PhotosPath     = "./uploads/photos"
	VideosPath     = "./uploads/videos"
	DocumentsPath  = "./uploads/documents"
	OthersPath     = "./uploads/others"
)

func InitLocalStorage() error {
	directories := []string{
		UploadBasePath,
		PhotosPath,
		VideosPath,
		DocumentsPath,
		OthersPath,
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}

func UploadToLocal(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	folder := determineFolder(contentType)

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s-%s%s",
		time.Now().Format("20060102-150405"),
		uuid.New().String()[:8],
		ext,
	)

	fullPath := filepath.Join(folder, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	relativePath := strings.TrimPrefix(fullPath, "./")
	return "/" + relativePath, nil
}

func determineFolder(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return PhotosPath
	case strings.HasPrefix(contentType, "video/"):
		return VideosPath
	case strings.HasPrefix(contentType, "application/pdf"),
		strings.HasPrefix(contentType, "application/msword"),
		strings.HasPrefix(contentType, "application/vnd.openxmlformats"):
		return DocumentsPath
	default:
		return OthersPath
	}
}

func DeleteFromLocal(filePath string) error {
	filePath = strings.TrimPrefix(filePath, "/")

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %v", err)
	}

	baseAbs, err := filepath.Abs(UploadBasePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %v", err)
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = realPath
	}
	if !strings.HasPrefix(absPath, baseAbs) {
		return fmt.Errorf("file path outside uploads directory")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return nil
}

func GetFileSize(filePath string) (int64, error) {
	filePath = strings.TrimPrefix(filePath, "/")

	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func FileExists(filePath string) bool {
	filePath = strings.TrimPrefix(filePath, "/")

	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
