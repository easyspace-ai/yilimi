package datacollect

import "testing"

func TestRollbackAshareWeekendISO(t *testing.T) {
	// 2026-03-29 周日 → 2026-03-27 五
	if got := rollbackAshareWeekendISO("2026-03-29"); got != "2026-03-27" {
		t.Fatalf("Sunday: got %q", got)
	}
	// 2026-03-28 周六 → 2026-03-27
	if got := rollbackAshareWeekendISO("2026-03-28"); got != "2026-03-27" {
		t.Fatalf("Saturday: got %q", got)
	}
	// 周五不变
	if got := rollbackAshareWeekendISO("2026-03-27"); got != "2026-03-27" {
		t.Fatalf("Friday: got %q", got)
	}
}
