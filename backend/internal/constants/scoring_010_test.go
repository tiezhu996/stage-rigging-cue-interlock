package constants

import "testing"

func TestSeverityRankBlockerAboveWarning(t *testing.T) {
	if SeverityRank(ResultBlocker) <= SeverityRank(ResultWarning) {
		t.Fatalf("blocker rank %d must exceed warning rank %d", SeverityRank(ResultBlocker), SeverityRank(ResultWarning))
	}
}

func TestHighestSeverityReturnsBlocker(t *testing.T) {
	if got := HighestSeverity(ResultBlocker, ResultWarning); got != ResultBlocker {
		t.Fatalf("HighestSeverity([blocker, warning]) = %s, want blocker", got)
	}
}
