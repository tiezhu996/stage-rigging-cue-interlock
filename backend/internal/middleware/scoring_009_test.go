package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"stage-rigging-cue-interlock/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestRBACAllowsDeclaredRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/review",
		func(c *gin.Context) { c.Set(auth.ContextRole, auth.RoleAdmin); c.Next() },
		RBAC(auth.RoleProgrammer, auth.RoleAdmin),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	req := httptest.NewRequest(http.MethodGet, "/review", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin should pass RBAC(programmer, admin), got %d", rec.Code)
	}
}
