package repository

import (
	"encoding/json"
	"errors"
	"testing"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/datatypes"
)

func newRepo007(t *testing.T) (*InterlockRuleRepository, *RiggingDeviceRepository) {
	t.Helper()
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:repo007?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	aud := audit.NewRepository(db)
	return NewInterlockRuleRepository(db, aud), NewRiggingDeviceRepository(db, aud)
}

func testEvent() audit.Event {
	return audit.Event{RequestID: "r7", ActorID: 1, ActorUsername: "u", Action: "x", EntityType: "x", BeforeSummary: "{}", AfterSummary: "{}"}
}

func ruleItem(code string) model.InterlockRule {
	ids, _ := json.Marshal([]uint{1})
	thr, _ := json.Marshal(map[string]any{"use_device_limits": true})
	return model.InterlockRule{RuleCode: code, RuleType: "load_limit", DeviceIDsJSON: datatypes.JSON(ids), ThresholdJSON: datatypes.JSON(thr), Severity: "blocker", Enabled: true, RuleVersion: 1, Explanation: "transaction guard rule"}
}

func TestRuleCreateDuplicateConflictPreserved(t *testing.T) {
	ruleRepo, _ := newRepo007(t)
	item := ruleItem("DUP-007")
	if err := ruleRepo.Create(&item, testEvent()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	dup := ruleItem("DUP-007")
	err := ruleRepo.Create(&dup, testEvent())
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("duplicate rule code must return a conflict error, got %v", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("duplicate status = %d, want 409", appErr.Status)
	}
}

func TestRuleUpdateVersionConflictPreserved(t *testing.T) {
	ruleRepo, _ := newRepo007(t)
	item := ruleItem("VER-007")
	if err := ruleRepo.Create(&item, testEvent()); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	item.RuleVersion = 99
	err := ruleRepo.Update(&item, 99, testEvent())
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("stale rule version must return a conflict error, got %v", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("stale version status = %d, want 409", appErr.Status)
	}
}

func TestDeviceCreateDuplicateConflictPreserved(t *testing.T) {
	_, deviceRepo := newRepo007(t)
	item := model.RiggingDevice{DeviceCode: "DUP-007-D", Name: "dup device", DeviceType: "motorized_batten", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 1, TravelMaxM: 20, SafetyZone: "zone", DeviceStatus: "available", Version: 1}
	if err := deviceRepo.Create(&item, testEvent()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	dup := model.RiggingDevice{DeviceCode: "DUP-007-D", Name: "dup device", DeviceType: "motorized_batten", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 1, TravelMaxM: 20, SafetyZone: "zone", DeviceStatus: "available", Version: 1}
	err := deviceRepo.Create(&dup, testEvent())
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("duplicate device code must return a conflict error, got %v", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("duplicate status = %d, want 409", appErr.Status)
	}
}

func TestDeviceUpdateVersionConflictPreserved(t *testing.T) {
	_, deviceRepo := newRepo007(t)
	item := model.RiggingDevice{DeviceCode: "VER-007-D", Name: "version device", DeviceType: "motorized_batten", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 1, TravelMaxM: 20, SafetyZone: "zone", DeviceStatus: "available", Version: 1}
	if err := deviceRepo.Create(&item, testEvent()); err != nil {
		t.Fatalf("create device: %v", err)
	}
	item.MaxLoadKG = 600
	err := deviceRepo.Update(&item, 99, testEvent())
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("stale device version must return a conflict error, got %v", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("stale version status = %d, want 409", appErr.Status)
	}
}
