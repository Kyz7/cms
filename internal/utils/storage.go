package utils

import (
	"fmt"
	"mime/multipart"
)

var (
	UseLocalStorage bool = true
	UseSupabase     bool = false
)

// UploadFile handles file upload based on the configured storage mode
func UploadFile(file *multipart.FileHeader) (string, error) {
	if UseLocalStorage {
		return UploadToLocal(file)
	}
	if UseSupabase {
		return UploadToSupabase(file)
	}
	return "", fmt.Errorf("no storage provider configured (S3 support removed)")
}

// DeleteFile handles file deletion based on the configured storage mode
func DeleteFile(url string) error {
	if UseLocalStorage {
		return DeleteFromLocal(url)
	}
	if UseSupabase {
		return DeleteFromSupabase(url)
	}
	return fmt.Errorf("no storage provider configured (S3 support removed)")
}

func GetStorageMode() string {
	if UseLocalStorage {
		return "local"
	}
	if UseSupabase {
		return "supabase"
	}
	return "none"
}

func SetStorageMode(useLocal bool) {
	UseLocalStorage = useLocal
	if useLocal {
		UseSupabase = false
	}
}

func SetSupabaseMode(useSupabase bool) {
	UseSupabase = useSupabase
	if useSupabase {
		UseLocalStorage = false
	}
}
