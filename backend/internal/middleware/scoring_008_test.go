package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func requestIDRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestRequestIDRejectsShortClientValue(t *testing.T) {
	router := requestIDRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "ab")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	got := rec.Header().Get("X-Request-ID")
	if len(got) < 8 {
		t.Fatalf("short client request id must be replaced by a generated one, got %q", got)
	}
}

func TestRequestIDGeneratedWhenAbsent(t *testing.T) {
	router := requestIDRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	got := rec.Header().Get("X-Request-ID")
	if got == "" {
		t.Fatal("missing client request id must be replaced by a generated one")
	}
}

func TestRecoveryResponseIncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.Use(Recovery(slog.New(slog.NewTextHandler(io.Discard, nil))))
	router.GET("/boom", func(c *gin.Context) { panic("kaboom") })
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic must map to 500, got %d", rec.Code)
	}
	if !containsRequestID(rec.Body.String()) {
		t.Fatalf("panic response must carry request_id, body=%s", rec.Body.String())
	}
}

func containsRequestID(body string) bool {
	return len(body) > 0 && strings.Contains(body, "request_id")
}

func TestRequestIDRejectsMalformedValue(t *testing.T) {
	router := requestIDRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "bad id with spaces!!")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	got := rec.Header().Get("X-Request-ID")
	if len(got) < 8 || strings.ContainsAny(got, " !") {
		t.Fatalf("malformed client request id must be replaced, got %q", got)
	}
}
