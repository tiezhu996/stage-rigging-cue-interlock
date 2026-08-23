package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterConcurrentNoRace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewLocalRateLimiter(1000)
	router := gin.New()
	router.Use(limiter.Handler())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "203.0.113.7:1234"
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestRateLimiterRemainingNoRace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewLocalRateLimiter(1000)
	router := gin.New()
	router.Use(limiter.Handler())
	router.GET("/", func(c *gin.Context) {
		_ = limiter.Remaining(c.ClientIP())
		c.Status(http.StatusOK)
	})

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "198.51.100.9:4321"
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestAccessLogRecordsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	gin.SetMode(gin.TestMode)
	limiter := NewLocalRateLimiter(1000)
	router := gin.New()
	router.Use(RequestID())
	router.Use(AccessLog(logger, limiter))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if !strings.Contains(buf.String(), "request_id=") || strings.Contains(buf.String(), "request_id=unknown") {
		t.Fatalf("access log must record the real request id, got: %s", buf.String())
	}
}
