package content

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// PreviewClaims defines the payload for entry preview tokens.
type PreviewClaims struct {
	EntryID       uint `json:"entry_id"`
	ContentTypeID uint `json:"content_type_id"`
	jwt.RegisteredClaims
}

func getPreviewSecret() ([]byte, error) {
	secret := os.Getenv("PREVIEW_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return nil, fmt.Errorf("preview secret is not configured (set PREVIEW_SECRET or JWT_SECRET)")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("preview secret must be at least 32 characters")
	}
	return []byte(secret), nil
}

func getPreviewTTL() time.Duration {
	if ttlStr := os.Getenv("PREVIEW_TOKEN_TTL_MINUTES"); ttlStr != "" {
		if v, err := strconv.Atoi(ttlStr); err == nil && v > 0 {
			return time.Duration(v) * time.Minute
		}
	}
	return 60 * time.Minute
}

// GenerateEntryPreviewToken issues a signed token for previewing a content entry.
func GenerateEntryPreviewToken(entry models.ContentEntry) (string, time.Time, error) {
	secret, err := getPreviewSecret()
		if err != nil {
			return "", time.Time{}, err
		}

		expiresAt := time.Now().Add(getPreviewTTL())
		claims := PreviewClaims{
			EntryID:       entry.ID,
			ContentTypeID: entry.ContentTypeID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expiresAt),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString(secret)
		if err != nil {
			return "", time.Time{}, err
		}

		return signed, expiresAt, nil
	}

// ParsePreviewToken validates and parses a preview token.
func ParsePreviewToken(tokenStr string) (*PreviewClaims, error) {
	secret, err := getPreviewSecret()
		if err != nil {
			return nil, err
		}

		claims := &PreviewClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		if err != nil {
			return nil, err
		}
		if !token.Valid {
			return nil, fmt.Errorf("invalid preview token")
		}
		return claims, nil
	}

// BuildPreviewURL builds a URL that front-end can open for live preview.
// If PREVIEW_FRONTEND_URL is set, it will be used; otherwise backend preview endpoint is returned.
func BuildPreviewURL(baseBackendURL string, entryID uint, token string) string {
	frontend := strings.TrimSuffix(os.Getenv("PREVIEW_FRONTEND_URL"), "/")
	escapedToken := url.QueryEscape(token)

	if frontend != "" {
		return fmt.Sprintf("%s?entry_id=%d&token=%s", frontend, entryID, escapedToken)
	}

	base := strings.TrimSuffix(baseBackendURL, "/")
	return fmt.Sprintf("%s/content/entries/%d/preview?token=%s", base, entryID, escapedToken)
}

