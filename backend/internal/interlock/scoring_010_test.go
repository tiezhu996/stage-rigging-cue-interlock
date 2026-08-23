package interlock

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/constants"
)

func TestCollisionWindowsExcludeTouchingActions(t *testing.T) {
	ordered := []GraphCue{
		{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 1000},
		{ID: 2, CueCode: "Q-2", SequenceNo: 2, StartOffsetMS: 1000, DurationMS: 1000},
	}
	cues := map[uint]CueInput{
		1: {GraphCue: ordered[0], Actions: []ActionInput{{DeviceID: 1, DurationMS: 1000, FromPositionM: 10, ToPositionM: 9, LoadKG: 100}}},
		2: {GraphCue: ordered[1], Actions: []ActionInput{{DeviceID: 2, DurationMS: 1000, FromPositionM: 12, ToPositionM: 11, LoadKG: 120}}},
	}
	devices := map[uint]DeviceInput{
		1: {ID: 1, DeviceCode: "D-1", SafetyZone: "zone-a", DeviceStatus: "available"},
		2: {ID: 2, DeviceCode: "D-2", SafetyZone: "zone-a", DeviceStatus: "available"},
	}
	events, err := ExpandTimeline(ordered, cues, devices)
	if err != nil {
		t.Fatalf("ExpandTimeline: %v", err)
	}
	windows := DetectCollisionWindows(events)
	if len(windows) != 0 {
		t.Fatalf("touching half-open intervals must not collide, got windows: %#v", windows)
	}
}

func TestCollisionWindowsIgnoreEmptyZone(t *testing.T) {
	events := []TimelineEvent{
		{CueCode: "Q-1", DeviceID: 1, DeviceCode: "D-1", SafetyZone: "", StartMS: 0, EndMS: 1000},
		{CueCode: "Q-2", DeviceID: 2, DeviceCode: "D-2", SafetyZone: "", StartMS: 200, EndMS: 900},
	}
	windows := DetectCollisionWindows(events)
	if len(windows) != 0 {
		t.Fatalf("empty safety zone must never collide, got windows: %#v", windows)
	}
}

func TestHighestSeverityOverridesResults(t *testing.T) {
	// mixed evidence where a blocker is followed by a warning must still be blocker
	got := constants.HighestSeverity(constants.ResultBlocker, constants.ResultWarning)
	if got != constants.ResultBlocker {
		t.Fatalf("HighestSeverity = %s, want blocker", got)
	}
}
