package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	SupabaseURL    string
	SupabaseKey    string
	SupabaseBucket string
)

func InitSupabase(url, key, bucket string) {
	SupabaseURL = url
	SupabaseKey = key
	SupabaseBucket = bucket
	SetSupabaseMode(true)
}

func UploadToSupabase(file *multipart.FileHeader) (string, error) {
	if SupabaseURL == "" || SupabaseKey == "" || SupabaseBucket == "" {
		return "", fmt.Errorf("supabase storage not initialized")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Read content
	content, err := io.ReadAll(src)
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s/%s%s",
		time.Now().Format("2006/01"),
		uuid.New().String(),
		ext,
	)

	// Remove leading slash if present in filename to match Supabase path expectation
	if filename[0] == '/' {
		filename = filename[1:]
	}

	apiURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", SupabaseURL, SupabaseBucket, filename)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(content))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+SupabaseKey)
	req.Header.Set("Content-Type", file.Header.Get("Content-Type"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload to supabase: status %d, body %s", resp.StatusCode, string(body))
	}

	// Construct public URL
	// Format: https://<project>.supabase.co/storage/v1/object/public/<bucket>/<filename>
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", SupabaseURL, SupabaseBucket, filename)
	return publicURL, nil
}

func DeleteFromSupabase(url string) error {
	// TODO: Implement deletion if needed.
	// For now, we focus on upload.
	// To implement delete, we need to extract the path from the URL.
	return nil
}
