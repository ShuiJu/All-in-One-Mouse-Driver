package vaxee

import "testing"

func TestBatteryRejectsInvalidPayload(t *testing.T) {
	if got := parsePercentFromPage([]byte{14, 165, 11, 1, 1, 255}); got != -1 {
		t.Fatalf("invalid battery became %d%%", got)
	}
	if got := parsePercentFromPage([]byte{14, 165, 11, 1, 1, 15}); got != 75 {
		t.Fatal(got)
	}
}

func TestVaxeeReports(t *testing.T) {
	r := buildReportSized(64, 0x08, byte(PerfCompetitiveMSOn))
	if len(r) != 64 || r[0] != 0x0e || r[1] != 0xa5 || r[2] != 0x08 || r[3] != 2 || r[4] != 1 || r[5] != 3 {
		t.Fatalf("bad perf report %x", r)
	}
	r = buildTrajectoryReportSized(64, TrajSmoothSensitive)
	if r[2] != 0x13 || r[3] != 2 || r[5] != 0 {
		t.Fatal("wrong trajectory")
	}
	r = buildTrajectoryReportSized(64, TrajStableControl)
	if r[3] != 2 || r[5] != 1 {
		t.Fatal("stable trajectory must use WRITE opcode and value 1")
	}
	if _, err := pollingToYY(8000); err == nil {
		t.Fatal("8k unexpectedly accepted")
	}
}
