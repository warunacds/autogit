package openai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/warunacds/autogit/internal/provider/openai"
)

func TestGenerateMessage_EmptyDiff(t *testing.T) {
	client := openai.New("fake-key", "http://localhost:1234/v1", "gpt-4o")
	_, err := client.GenerateMessage("")
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
}

func TestGenerateMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", body["model"])
		}
		if body["max_tokens"] != float64(1024) {
			t.Errorf("expected max_tokens 1024, got %v", body["max_tokens"])
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "feat: add new feature",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := openai.New("test-key", server.URL, "test-model")
	msg, err := client.GenerateMessage("diff content here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: add new feature" {
		t.Fatalf("expected 'feat: add new feature', got %q", msg)
	}
}

func TestGenerateMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "server error"}}`))
	}))
	defer server.Close()

	client := openai.New("test-key", server.URL, "test-model")
	_, err := client.GenerateMessage("diff content")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestGenerateMessage_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := openai.New("test-key", server.URL, "test-model")
	_, err := client.GenerateMessage("diff content")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestGenerateMessage_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header, got %s", r.Header.Get("Authorization"))
		}
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "feat: local model"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := openai.New("", server.URL, "llama3")
	msg, err := client.GenerateMessage("diff content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: local model" {
		t.Fatalf("expected 'feat: local model', got %q", msg)
	}
}
