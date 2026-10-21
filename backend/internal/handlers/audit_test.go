package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func TestAdminAuditMiddleware_RecordsMutationsOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("username", "admin")
		c.Set("telegram_id", int64(42))
		c.Next()
	})
	r.Use(AdminAuditMiddleware(db))
	r.POST("/mutate", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/read", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, method := range []string{"POST", "GET"} {
		req := httptest.NewRequest(method, "/"+map[string]string{"POST": "mutate", "GET": "read"}[method], nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
	var entries []models.AdminAuditLog
	db.Find(&entries)
	if len(entries) != 1 {
		t.Fatalf("expected only the mutation to be logged, got %d", len(entries))
	}
	if entries[0].AdminUsername != "admin" || entries[0].Method != "POST" || entries[0].Path != "/mutate" {
		t.Fatalf("unexpected audit entry: %+v", entries[0])
	}
}
