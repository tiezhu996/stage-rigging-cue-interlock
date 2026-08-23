package interlock

import (
	"sync"
	"testing"
)

func loadRule(id uint, code string, deviceID uint) RuleInput {
	threshold := 50.0
	return RuleInput{ID: id, RuleCode: code, RuleType: "load_limit", DeviceIDs: []uint{deviceID}, Threshold: RuleThreshold{MaxLoadKG: &threshold}, Severity: "warning", Enabled: true, RuleVersion: 1}
}

func baseCues() []CueInput {
	ordered := []GraphCue{
		{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 2000},
		{ID: 2, CueCode: "Q-2", SequenceNo: 2, DurationMS: 2000},
	}
	return []CueInput{
		{GraphCue: ordered[0], Actions: []ActionInput{{DeviceID: 1, DurationMS: 2000, FromPositionM: 10, ToPositionM: 9, LoadKG: 100}}},
		{GraphCue: ordered[1], Actions: []ActionInput{{DeviceID: 2, DurationMS: 2000, FromPositionM: 12, ToPositionM: 11, LoadKG: 120}}},
	}
}

func baseDevices() []DeviceInput {
	return []DeviceInput{
		{ID: 1, DeviceCode: "D-1", MaxLoadKG: 500, MaxSpeedMS: 1, TravelMinM: 0, TravelMaxM: 20, SafetyZone: "zone-a", DeviceStatus: "available"},
		{ID: 2, DeviceCode: "D-2", MaxLoadKG: 500, MaxSpeedMS: 1, TravelMinM: 0, TravelMaxM: 20, SafetyZone: "zone-b", DeviceStatus: "available"},
	}
}

type evalResult struct {
	evaluation Evaluation
	err        error
}

func concurrentEvaluate(cues []CueInput, devices []DeviceInput, ruleSetA, ruleSetB []RuleInput) (evalResult, evalResult) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	var resultA, resultB evalResult
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		resultA.evaluation, resultA.err = Evaluate(cues, devices, ruleSetA, 100)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		resultB.evaluation, resultB.err = Evaluate(cues, devices, ruleSetB, 100)
	}()
	close(start)
	wg.Wait()
	return resultA, resultB
}

func TestParallelRuleEvaluationDeviceScopeIsolation(t *testing.T) {
	resultA, resultB := concurrentEvaluate(baseCues(), baseDevices(),
		[]RuleInput{loadRule(1, "LOAD-A", 1), loadRule(2, "LOAD-A2", 1)},
		[]RuleInput{loadRule(3, "LOAD-B", 2), loadRule(4, "LOAD-B2", 2)},
	)
	if resultA.err != nil || resultB.err != nil {
		t.Fatalf("concurrent evaluations must not crash or error: %v / %v", resultA.err, resultB.err)
	}
	for _, ev := range append(append([]RuleEvidence{}, resultA.evaluation.RuleResults...), resultB.evaluation.RuleResults...) {
		switch ev.RuleCode {
		case "LOAD-A", "LOAD-A2":
			if len(ev.DeviceCodes) != 1 || ev.DeviceCodes[0] != "D-1" {
				t.Fatalf("rule %s must only cover device D-1, got %v", ev.RuleCode, ev.DeviceCodes)
			}
		case "LOAD-B", "LOAD-B2":
			if len(ev.DeviceCodes) != 1 || ev.DeviceCodes[0] != "D-2" {
				t.Fatalf("rule %s must only cover device D-2, got %v", ev.RuleCode, ev.DeviceCodes)
			}
		}
	}
}

func TestParallelRuleEvaluationPropagatesMissingDevice(t *testing.T) {
	resultA, _ := concurrentEvaluate(baseCues(), baseDevices(),
		[]RuleInput{loadRule(1, "LOAD-MISSING", 999)},
		[]RuleInput{loadRule(2, "LOAD-OK", 1)},
	)
	if resultA.err == nil {
		t.Fatal("a rule referencing a missing device must surface an error")
	}
}

func TestParallelRuleEvaluationConcurrentNoRace(t *testing.T) {
	resultA, resultB := concurrentEvaluate(baseCues(), baseDevices(),
		[]RuleInput{loadRule(1, "LOAD-C1", 1), loadRule(2, "LOAD-C2", 2)},
		[]RuleInput{loadRule(3, "LOAD-C3", 1), loadRule(4, "LOAD-C4", 2)},
	)
	if resultA.err != nil || resultB.err != nil {
		t.Fatalf("concurrent evaluation failed: %v / %v", resultA.err, resultB.err)
	}
	if len(resultA.evaluation.RuleResults) != 2 || len(resultB.evaluation.RuleResults) != 2 {
		t.Fatalf("both waves must collect all rule evidences: A=%d B=%d", len(resultA.evaluation.RuleResults), len(resultB.evaluation.RuleResults))
	}
}

func TestParallelRuleEvaluationStableOrder(t *testing.T) {
	rules := []RuleInput{
		{ID: 3, RuleCode: "ALOAD", RuleType: "load_limit", DeviceIDs: []uint{1}, Threshold: RuleThreshold{MaxLoadKG: floatPtr(50)}, Severity: "warning", Enabled: true, RuleVersion: 1},
		{ID: 2, RuleCode: "MLOAD", RuleType: "load_limit", DeviceIDs: []uint{2}, Threshold: RuleThreshold{MaxLoadKG: floatPtr(50)}, Severity: "warning", Enabled: true, RuleVersion: 1},
		{ID: 1, RuleCode: "ZLOAD", RuleType: "load_limit", DeviceIDs: []uint{1}, Threshold: RuleThreshold{MaxLoadKG: floatPtr(50)}, Severity: "warning", Enabled: true, RuleVersion: 1},
	}
	resultA, resultB := concurrentEvaluate(baseCues(), baseDevices(), rules, rules)
	if resultA.err != nil || resultB.err != nil {
		t.Fatalf("evaluation failed: %v / %v", resultA.err, resultB.err)
	}
	want := []string{"ALOAD", "MLOAD", "ZLOAD"}
	for _, result := range []evalResult{resultA, resultB} {
		if len(result.evaluation.RuleResults) != 3 {
			t.Fatalf("want 3 results, got %d", len(result.evaluation.RuleResults))
		}
		for i, code := range want {
			if result.evaluation.RuleResults[i].RuleCode != code {
				t.Fatalf("evidence order = %s..., want stable %v", ruleCodes(result.evaluation.RuleResults), want)
			}
		}
	}
}

func TestTopologicalOrderStableForUnsortedInput(t *testing.T) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	var orderA, orderB []GraphCue
	cues := []GraphCue{
		{ID: 2, CueCode: "Q-020", SequenceNo: 20, StartOffsetMS: 500, DurationMS: 1000},
		{ID: 1, CueCode: "Q-010", SequenceNo: 10, DurationMS: 1000},
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		orderA, _ = ValidateAndSort(cues)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		orderB, _ = ValidateAndSort(cues)
	}()
	close(start)
	wg.Wait()
	for _, ordered := range [][]GraphCue{orderA, orderB} {
		if len(ordered) != 2 || ordered[0].ID != 1 || ordered[1].ID != 2 {
			t.Fatalf("unsorted input must still produce sequence-ordered cues, got %v", idsOf(ordered))
		}
	}
}

func floatPtr(value float64) *float64 { return &value }

func ruleCodes(results []RuleEvidence) []string {
	codes := make([]string, 0, len(results))
	for _, result := range results {
		codes = append(codes, result.RuleCode)
	}
	return codes
}

func idsOf(cues []GraphCue) []uint {
	ids := make([]uint, 0, len(cues))
	for _, cue := range cues {
		ids = append(ids, cue.ID)
	}
	return ids
}
