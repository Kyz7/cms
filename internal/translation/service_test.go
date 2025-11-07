package translation

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewClientMissingAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_TRANSLATE_API_KEY", "")

	client, err := NewClient()
	if err == nil {
		t.Fatalf("expected error, got client: %#v", client)
	}
	if err.Error() != "missing GOOGLE_TRANSLATE_API_KEY" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewClientSuccess(t *testing.T) {
	t.Setenv("GOOGLE_TRANSLATE_API_KEY", "dummy-key")

	client, err := NewClient()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestTranslateTextsSuccess(t *testing.T) {
	expectedReq := translateRequest{}
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content-type: %s", req.Header.Get("Content-Type"))
		}
		defer req.Body.Close()
		if err := json.NewDecoder(req.Body).Decode(&expectedReq); err != nil {
			t.Fatalf("failed decoding request: %v", err)
		}
		body := `{"data":{"translations":[{"translatedText":"Hola"},{"translatedText":"Mundo"}]}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	client := &Client{apiKey: "dummy", httpClient: &http.Client{Transport: rt}}

	texts := []string{"Hello", "World"}
	translated, err := client.TranslateTexts(texts, "es", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(translated) != 2 || translated[0] != "Hola" || translated[1] != "Mundo" {
		t.Fatalf("unexpected translations: %#v", translated)
	}

	if expectedReq.Target != "es" {
		t.Fatalf("unexpected target: %s", expectedReq.Target)
	}
	if expectedReq.Source != "en" {
		t.Fatalf("unexpected source: %s", expectedReq.Source)
	}
	if len(expectedReq.Q) != len(texts) {
		t.Fatalf("unexpected request texts: %#v", expectedReq.Q)
	}
}

func TestTranslateTextsEmptyInput(t *testing.T) {
	client := &Client{apiKey: "dummy", httpClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected HTTP call: %v", req)
		return nil, nil
	})}}

	translated, err := client.TranslateTexts(nil, "es", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(translated) != 0 {
		t.Fatalf("expected empty result, got %#v", translated)
	}
}

func TestTranslateTextsAPIErrors(t *testing.T) {
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	client := &Client{apiKey: "dummy", httpClient: &http.Client{Transport: rt}}

	_, err := client.TranslateTexts([]string{"hola"}, "en", "es")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
