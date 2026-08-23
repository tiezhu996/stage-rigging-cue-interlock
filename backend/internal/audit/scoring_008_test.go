package audit_test

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
)

func TestAuditEventPreservesRequestID(t *testing.T) {
	event := audit.NewEvent(audit.ActorContext{ID: 1, Username: "u", RequestID: "req-abc-12345"}, "cue.create", "cue", 0, "{}", "{}")
	if event.RequestID != "req-abc-12345" {
		t.Fatalf("audit event must keep the request id, got %q", event.RequestID)
	}
}

func TestAuditSearchByRequestID(t *testing.T) {
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:aud008?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := audit.NewRepository(db)
	if err := repo.Record(audit.Event{RequestID: "req-11111", ActorID: 1, ActorUsername: "alice", Action: "cue.create", EntityType: "cue", BeforeSummary: "{}", AfterSummary: "{}"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := repo.Record(audit.Event{RequestID: "req-22222", ActorID: 2, ActorUsername: "bob", Action: "cue.update", EntityType: "cue", BeforeSummary: "{}", AfterSummary: "{}"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	_, total, err := repo.List(1, 50, "", "", "req-11111")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("searching by request id must find exactly one event, got %d", total)
	}
}
