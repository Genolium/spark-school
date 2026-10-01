package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPromoValidate_BadRequestOnEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPromoHandler(nil)
	router := gin.New()
	router.POST("/api/v1/promo/validate", handler.Validate)

	// Missing code field
	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/promo/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty body, got %d", w.Code)
	}
}

func TestPromoCreate_BadRequestOnEmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPromoHandler(nil)
	router := gin.New()
	router.POST("/api/v1/promo/create", handler.Create)

	body, _ := json.Marshal(map[string]string{"code": ""})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/promo/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty promo code, got %d", w.Code)
	}
}
