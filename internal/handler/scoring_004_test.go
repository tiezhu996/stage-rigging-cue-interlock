package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
	"stage-rigging-cue-interlock/backend/internal/handler"
	"stage-rigging-cue-interlock/backend/internal/middleware"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/router"
	"stage-rigging-cue-interlock/backend/internal/service"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func newEngine(t *testing.T) (*gin.Engine, *auth.Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		DBDriver: "sqlite", DBDSN: "file:http004?mode=memory&cache=shared", DBAutoMigrate: true,
		JWTSecret: "http-test-004-secret-at-least-32-bytes", JWTTTL: 60 * 60 * 1000 * 1000 * 1000,
		TimelineStepMS: 100, MaxCuesPerRun: 40, RateLimitPerMinute: 10000,
	}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	aud := audit.NewRepository(db)
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, cfg.JWTSecret, cfg.JWTTTL)
	deviceRepo := repository.NewRiggingDeviceRepository(db, aud)
	cueRepo := repository.NewCueDefinitionRepository(db, aud)
	ruleRepo := repository.NewInterlockRuleRepository(db, aud)
	runRepo := repository.NewRehearsalRunRepository(db, aud)
	_ = service.NewRiggingDeviceService(deviceRepo, ruleRepo)
	cueService := service.NewCueDefinitionService(cueRepo, deviceRepo)
	_ = service.NewInterlockRuleService(ruleRepo, deviceRepo)
	_ = service.NewRehearsalRunService(runRepo, cueRepo, deviceRepo, ruleRepo, cfg.TimelineStepMS, cfg.MaxCuesPerRun)

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Recovery(nil))
	api := engine.Group("/api/v1")
	api.POST("/auth/login", auth.NewHandler(authService).Login)
	protected := api.Group("")
	protected.Use(middleware.Auth(authService))
	write := middleware.RBAC(auth.RoleProgrammer, auth.RoleAdmin)
	review := middleware.RBAC(auth.RoleSafetyReviewer, auth.RoleAdmin)
	router.RegisterCueDefinitionRoutes(protected, handler.NewCueDefinitionHandler(cueService), write, review)
	return engine, authService
}

func loginToken(t *testing.T, engine *gin.Engine, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d", username, rec.Code)
	}
	var resp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Data.AccessToken == "" {
		t.Fatalf("bad login response: %s", rec.Body.String())
	}
	return resp.Data.AccessToken
}

func TestCueGetNotFoundReturns404(t *testing.T) {
	engine, _ := newEngine(t)
	token := loginToken(t, engine, "programmer", "programmer123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cues/999999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET missing cue = %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestCueCreateDuplicateReturns409(t *testing.T) {
	engine, _ := newEngine(t)
	token := loginToken(t, engine, "programmer", "programmer123")
	payload := `{"cue_code":"HTTP-409","name":"duplicate cue","sequence_no":5,"duration_ms":1000,"actions":[{"device_id":1,"duration_ms":1000,"from_position_m":10,"to_position_m":9,"load_kg":100}]}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cues", bytes.NewReader([]byte(payload)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("duplicate cue create = %d, want 409 (body %s)", rec.Code, rec.Body.String())
		}
	}
}

func TestCueUpdateNotFoundReturns404(t *testing.T) {
	engine, _ := newEngine(t)
	token := loginToken(t, engine, "programmer", "programmer123")
	payload := fmt.Sprintf(`{"name":"update missing","sequence_no":7,"duration_ms":1000,"actions":[{"device_id":1,"duration_ms":1000,"from_position_m":10,"to_position_m":9,"load_kg":100}],"version":1}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/cues/888888", bytes.NewReader([]byte(payload)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PUT missing cue = %d, want 404 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestFailMapsWrappedAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		util.Fail(c, fmt.Errorf("outer wrap: %w", util.NotFound("CUE_NOT_FOUND", "cue was not found")))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("wrapped AppError must map to 404, got %d", rec.Code)
	}
}
