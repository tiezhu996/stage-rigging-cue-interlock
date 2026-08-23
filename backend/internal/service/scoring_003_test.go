package service

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/database"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"
)

func newRunService003(t *testing.T) (*RehearsalRunService, []uint) {
	t.Helper()
	db, err := database.Open(config.Config{DBDriver: "sqlite", DBDSN: "file:run003?mode=memory&cache=shared", DBAutoMigrate: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	aud := audit.NewRepository(db)
	cues := repository.NewCueDefinitionRepository(db, aud)
	devices := repository.NewRiggingDeviceRepository(db, aud)
	rules := repository.NewInterlockRuleRepository(db, aud)
	runs := repository.NewRehearsalRunRepository(db, aud)
	svc := NewRehearsalRunService(runs, cues, devices, rules, 100, 40)
	locked, _, err := cues.List(1, 10, "locked", "")
	if err != nil || len(locked) < 2 {
		t.Fatalf("seeded locked cues missing: %v", err)
	}
	return svc, []uint{locked[0].ID, locked[1].ID}
}

func TestRehearsalRunPreservesRequestCueIDOrder(t *testing.T) {
	svc, ids := newRunService003(t)
	reversed := []uint{ids[1], ids[0]}
	request := dto.RunRehearsalRequest{CueIDs: append([]uint(nil), reversed...)}
	if _, err := svc.Run(request, audit.ActorContext{ID: 1, Username: "u"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	for i := range reversed {
		if request.CueIDs[i] != reversed[i] {
			t.Fatalf("run must not mutate the caller's cue_ids slice: got %v want %v", request.CueIDs, reversed)
		}
	}
}

func TestRehearsalRejectsDuplicateCueSelection(t *testing.T) {
	svc, ids := newRunService003(t)
	request := dto.RunRehearsalRequest{CueIDs: []uint{ids[0], ids[0]}}
	if _, err := svc.Run(request, audit.ActorContext{ID: 1, Username: "u"}); err == nil {
		t.Fatal("selecting the same cue twice must be rejected")
	}
}

func TestCueSetVersionTracksDeviceLimits(t *testing.T) {
	devices := []model.RiggingDevice{{ID: 7, Version: 1}}
	first := cueSetVersion(nil, nil, devices)
	devices[0].Version = 2
	second := cueSetVersion(nil, nil, devices)
	if first == second {
		t.Fatal("changing device limits must change the rehearsal input version")
	}
}

func TestCompareIdenticalSnapshotsSame(t *testing.T) {
	svc, ids := newRunService003(t)
	left, err := svc.Run(dto.RunRehearsalRequest{CueIDs: []uint{ids[0], ids[1]}}, audit.ActorContext{ID: 1, Username: "u"})
	if err != nil {
		t.Fatalf("left run: %v", err)
	}
	right, err := svc.Run(dto.RunRehearsalRequest{CueIDs: []uint{ids[1], ids[0]}}, audit.ActorContext{ID: 1, Username: "u"})
	if err != nil {
		t.Fatalf("right run: %v", err)
	}
	compared, err := svc.Compare(left.ID, right.ID)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	same, _ := compared["delta"].(map[string]any)["same_snapshot"].(bool)
	if !same {
		t.Fatal("two runs of the same cue set must be considered the same snapshot")
	}
}
