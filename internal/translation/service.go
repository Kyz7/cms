package translation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type translateRequest struct {
	Q      []string `json:"q"`
	Target string   `json:"target"`
	Source string   `json:"source,omitempty"`
}

type translateResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
}

func NewClient() (*Client, error) {
	apiKey := os.Getenv("GOOGLE_TRANSLATE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("missing GOOGLE_TRANSLATE_API_KEY")
	}
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

func (c *Client) TranslateTexts(texts []string, targetLang, sourceLang string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}
	reqBody := translateRequest{Q: texts, Target: targetLang}
	if sourceLang != "" {
		reqBody.Source = sourceLang
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://translation.googleapis.com/language/translate/v2?key=%s", c.apiKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("translate API status %d", resp.StatusCode)
	}
	var tr translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(tr.Data.Translations))
	for _, t := range tr.Data.Translations {
		out = append(out, t.TranslatedText)
	}
	return out, nil
}
