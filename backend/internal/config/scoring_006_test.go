package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigShutdownTimeoutParsed(t *testing.T) {
	os.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "25")
	defer os.Unsetenv("SHUTDOWN_TIMEOUT_SECONDS")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GracefulShutdownPeriod != 25*time.Second {
		t.Fatalf("shutdown period = %v, want 25s", cfg.GracefulShutdownPeriod)
	}
}

func TestConfigRejectsZeroShutdownTimeout(t *testing.T) {
	os.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "0")
	defer os.Unsetenv("SHUTDOWN_TIMEOUT_SECONDS")
	if _, err := Load(); err == nil {
		t.Fatal("zero shutdown timeout must be rejected")
	}
}
