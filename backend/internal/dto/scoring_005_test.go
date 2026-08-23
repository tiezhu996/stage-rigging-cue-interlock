package dto

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/model"
)

func TestDeviceResponseApplicableRulesNotNull(t *testing.T) {
	resp := RiggingDeviceFromModel(model.RiggingDevice{DeviceCode: "D-1", Name: "One"})
	if resp.ApplicableRules == nil {
		t.Fatal("applicable_rules must serialize as an empty array, not null")
	}
	if len(resp.ApplicableRules) != 0 {
		t.Fatalf("expected no applicable rules, got %d", len(resp.ApplicableRules))
	}
}
