package service

import (
	"encoding/json"
	"testing"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"

	"gorm.io/datatypes"
)

func TestDeviceWithRulesPopulatesWithoutPanic(t *testing.T) {
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:dev005?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	aud := audit.NewRepository(db)
	devices := repository.NewRiggingDeviceRepository(db, aud)
	rules := repository.NewInterlockRuleRepository(db, aud)
	svc := NewRiggingDeviceService(devices, rules)
	items, _, err := devices.List(1, 10, "", "")
	if err != nil || len(items) == 0 {
		t.Fatalf("seed devices missing: %v", err)
	}
	resp, err := svc.Get(items[0].ID)
	if err != nil {
		t.Fatalf("device with rules must not panic: %v", err)
	}
	if len(resp.ApplicableRules) == 0 {
		t.Fatal("device participating in rules should report applicable rules")
	}
}

func TestDeviceApplicableRulesDeduplicated(t *testing.T) {
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:dev005b?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	aud := audit.NewRepository(db)
	devices := repository.NewRiggingDeviceRepository(db, aud)
	rules := repository.NewInterlockRuleRepository(db, aud)
	svc := NewRiggingDeviceService(devices, rules)
	device := &model.RiggingDevice{DeviceCode: "DEDUPE-05", Name: "Dedupe device", DeviceType: "motorized_batten", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 1, TravelMaxM: 20, SafetyZone: "zone-05", DeviceStatus: "available", Version: 1}
	if err := devices.Create(device, audit.Event{RequestID: "r", ActorID: 1, Action: "create", EntityType: "rigging_device", BeforeSummary: "{}", AfterSummary: "{}"}); err != nil {
		t.Fatalf("create device: %v", err)
	}
	idsJSON, _ := json.Marshal([]uint{device.ID, device.ID})
	thresholdJSON, _ := json.Marshal(map[string]any{"use_device_limits": true})
	if err := rules.Create(&model.InterlockRule{RuleCode: "DUP-05", RuleType: "load_limit", DeviceIDsJSON: datatypes.JSON(idsJSON), ThresholdJSON: datatypes.JSON(thresholdJSON), Severity: "blocker", Enabled: true, RuleVersion: 1, Explanation: "dedupe rule"}, audit.Event{RequestID: "r", ActorID: 1, Action: "create", EntityType: "interlock_rule", BeforeSummary: "{}", AfterSummary: "{}"}); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	resp, err := svc.Get(device.ID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	count := 0
	for _, ref := range resp.ApplicableRules {
		if ref.RuleCode == "DUP-05" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("rule listed twice in scope must appear once, got %d", count)
	}
}

func TestDeviceWithRulesPropagatesDecodeError(t *testing.T) {
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:dev005c?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	aud := audit.NewRepository(db)
	devices := repository.NewRiggingDeviceRepository(db, aud)
	rules := repository.NewInterlockRuleRepository(db, aud)
	svc := NewRiggingDeviceService(devices, rules)
	device := &model.RiggingDevice{DeviceCode: "DECODE-05", Name: "Decode device", DeviceType: "motorized_batten", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 1, TravelMaxM: 20, SafetyZone: "zone-05", DeviceStatus: "available", Version: 1}
	if err := devices.Create(device, audit.Event{RequestID: "r", ActorID: 1, Action: "create", EntityType: "rigging_device", BeforeSummary: "{}", AfterSummary: "{}"}); err != nil {
		t.Fatalf("create device: %v", err)
	}
	thr, _ := json.Marshal(map[string]any{"use_device_limits": true})
	if err := rules.Create(&model.InterlockRule{RuleCode: "BROKEN-05", RuleType: "load_limit", DeviceIDsJSON: datatypes.JSON("not-json"), ThresholdJSON: datatypes.JSON(thr), Severity: "blocker", Enabled: true, RuleVersion: 1, Explanation: "broken scope rule"}, audit.Event{RequestID: "r", ActorID: 1, Action: "create", EntityType: "interlock_rule", BeforeSummary: "{}", AfterSummary: "{}"}); err != nil {
		t.Fatalf("create broken rule: %v", err)
	}
	if _, err := svc.Get(device.ID); err == nil {
		t.Fatal("a malformed rule device scope must surface a decode error instead of being silently skipped")
	}
}
