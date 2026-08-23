package dto

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/model"

	"gorm.io/datatypes"
)

func TestRunFromModelEmptyCollectionsNotNull(t *testing.T) {
	item := model.RehearsalRun{
		CueSetVersion:        "v1",
		RunStatus:            "evaluated",
		TimelineSnapshotJSON: datatypes.JSON("{}"),
		RuleResultsJSON:      datatypes.JSON("null"),
		CollisionWindowsJSON: datatypes.JSON("null"),
		HighestSeverity:      "pass",
		StartedBy:            1,
	}
	response, err := RunFromModel(item)
	if err != nil {
		t.Fatalf("RunFromModel: %v", err)
	}
	if response.RuleResults == nil {
		t.Fatal("rule_results must be an empty array, not null")
	}
	if response.CollisionWindows == nil {
		t.Fatal("collision_windows must be an empty array, not null")
	}
}
