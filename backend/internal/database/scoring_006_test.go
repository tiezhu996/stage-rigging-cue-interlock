package database

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/config"
)

func TestDatabasePingContextHasDeadline(t *testing.T) {
	ctx, cancel := contextWithTimeout()
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("ping context must carry a deadline so health checks cannot hang forever")
	}
	if deadline.IsZero() {
		t.Fatal("ping context deadline must be set")
	}
}

func TestDatabaseOpenConfiguresPool(t *testing.T) {
	db, err := Open(config.Config{DBDriver: "sqlite", DBDSN: "file:pool006?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("sqlite pool must be limited to a single connection, got %d", sqlDB.Stats().MaxOpenConnections)
	}
}
