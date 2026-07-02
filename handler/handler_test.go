package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"image_sync/config"
)

func setupRouter(cfg config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, nil, cfg)
	return r
}

func TestSyncRequestValidation(t *testing.T) {
	cfg := config.Config{
		Registries: map[string]config.RegistryConfig{
			"harbor": {URL: "reg.example.com", Username: "u", Password: "p"},
		},
	}
	r := setupRouter(cfg)

	tests := []struct {
		name       string
		body       SyncRequest
		wantStatus int
	}{
		{"missing src", SyncRequest{Dst: "harbor", Image: "nginx", Tag: "1.0"}, http.StatusBadRequest},
		{"missing dst", SyncRequest{Src: "harbor", Image: "nginx", Tag: "1.0"}, http.StatusBadRequest},
		{"missing image", SyncRequest{Src: "harbor", Dst: "harbor", Tag: "1.0"}, http.StatusBadRequest},
		{"missing tag", SyncRequest{Src: "harbor", Dst: "harbor", Image: "nginx"}, http.StatusBadRequest},
		{"src not configured", SyncRequest{Src: "unknown", Dst: "harbor", Image: "nginx", Tag: "1.0"}, http.StatusBadRequest},
		{"dst not configured", SyncRequest{Src: "harbor", Dst: "unknown", Image: "nginx", Tag: "1.0"}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/sync", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestTaskStatusNotFound(t *testing.T) {
	cfg := config.Config{}
	r := setupRouter(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tasks/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
