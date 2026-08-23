package auth_test

import (
	"errors"
	"testing"
	"time"

	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "scoring-009-test-secret-at-least-32-bytes"

func newAuthService(t *testing.T) *auth.Service {
	t.Helper()
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:auth009?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return auth.NewService(auth.NewRepository(db), testSecret, time.Hour)
}

func TestParseRejectsNonHS256Method(t *testing.T) {
	svc := newAuthService(t)
	claims := auth.Claims{UserID: 1, Username: "programmer", Role: "programmer", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign hs512 token: %v", err)
	}
	if _, err := svc.Parse(token); err == nil {
		t.Fatal("a token signed with a non-HS256 method must be rejected")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	svc := newAuthService(t)
	claims := auth.Claims{UserID: 1, Username: "programmer", Role: "programmer", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	if _, err := svc.Parse(token); err == nil {
		t.Fatal("an expired token must be rejected")
	}
}

func TestLoginWrongPasswordUnauthorized(t *testing.T) {
	svc := newAuthService(t)
	_, _, err := svc.Login("programmer", "definitely-wrong")
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("wrong password must map to an AppError, got %v", err)
	}
	if appErr.Status != 401 {
		t.Fatalf("wrong password status = %d, want 401", appErr.Status)
	}
}
