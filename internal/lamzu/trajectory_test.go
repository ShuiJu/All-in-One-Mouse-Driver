package lamzu

import "testing"

func trajectoryReply(mode TrajectoryMode) []byte {
	reply := make([]byte, 64)
	copy(reply, []byte{0xa1, 0, 2, 4, 1, 0x9e, 1, byte(mode), 0x54, 0x52})
	return reply
}

func TestTrajectoryCapabilityReplies(t *testing.T) {
	for _, mode := range []TrajectoryMode{TrajSmooth, TrajStable} {
		for _, reportID := range []bool{false, true} {
			reply := trajectoryReply(mode)
			if reportID {
				reply = append([]byte{0}, reply...)
			}
			actual, err := ParseTrajectoryReply(reply)
			if err != nil || actual != mode {
				t.Fatalf("mode %d reportID %v: mode=%d err=%v", mode, reportID, actual, err)
			}
		}
	}
	if actual, err := ParseTrajectoryReply(trajectoryReply(TrajStable)[:10]); err != nil || actual != TrajStable {
		t.Fatalf("ten-byte wireless reply rejected: mode=%d err=%v", actual, err)
	}
	invalid := map[string][]byte{
		"no reply":             nil,
		"short header":         trajectoryReply(TrajSmooth)[:9],
		"oversized payload":    append(trajectoryReply(TrajSmooth), 0, 0),
		"unmarked legacy zero": make([]byte, 64),
	}
	for _, change := range []struct {
		name  string
		index int
		value byte
	}{
		{"legacy A2", 0, 0xa2}, {"legacy A3", 0, 0xa3},
		{"header reserved", 1, 1}, {"header family", 2, 3}, {"header length", 3, 5},
		{"header channel", 4, 0}, {"stale different query", 5, 0x80}, {"value count", 6, 0},
		{"invalid mode", 7, 2}, {"missing T marker", 8, 0}, {"missing R marker", 9, 0},
	} {
		reply := trajectoryReply(TrajSmooth)
		reply[change.index] = change.value
		invalid[change.name] = reply
	}
	badID := append([]byte{1}, trajectoryReply(TrajStable)...)
	invalid["nonzero report ID"] = badID
	for name, reply := range invalid {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseTrajectoryReply(reply); err == nil {
				t.Fatal("invalid reply established trajectory capability")
			}
		})
	}
}
