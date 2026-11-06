package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func TestValidatePromo_TierRestriction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	if err := db.Create(&models.PromoCode{
		Code:            "DEADLINE",
		DiscountPercent: 14,
		IsActive:        true,
		ApplicableTiers: "accelerator,vip",
	}).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewPromoHandler(db)
	router := gin.New()
	router.POST("/validate", handler.Validate)

	validate := func(tier string) ValidatePromoResponse {
		body, _ := json.Marshal(map[string]interface{}{"code": "DEADLINE", "tier": tier})
		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("validate %s: status %d", tier, w.Code)
		}
		var res ValidatePromoResponse
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatal(err)
		}
		return res
	}

	if res := validate("basic"); res.Valid {
		t.Fatalf("basic must be rejected for a promo limited to accelerator/vip")
	}
	if res := validate("accelerator"); !res.Valid || res.DiscountPercent != 14 {
		t.Fatalf("accelerator must be valid with 14%%, got %+v", res)
	}
	if res := validate("vip"); !res.Valid || res.DiscountPercent != 14 {
		t.Fatalf("vip must be valid with 14%%, got %+v", res)
	}
}

func TestValidatePromo_UnrestrictedPromoWorksEverywhere(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	db.Create(&models.PromoCode{Code: "SPARK5", DiscountPercent: 5, IsActive: true})

	handler := NewPromoHandler(db)
	router := gin.New()
	router.POST("/validate", handler.Validate)

	for _, tier := range []string{"basic", "accelerator", "vip"} {
		body, _ := json.Marshal(map[string]interface{}{"code": "SPARK5", "tier": tier})
		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var res ValidatePromoResponse
		json.Unmarshal(w.Body.Bytes(), &res)
		if !res.Valid {
			t.Fatalf("promo without restrictions must work for %s", tier)
		}
	}
}
