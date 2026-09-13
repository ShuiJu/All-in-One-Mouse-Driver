package lamzu

import "testing"

func TestIncaReports(t *testing.T) {
	for _, length := range []int{64, 65} {
		offset := 0
		if length == 65 {
			offset = 1
		}
		r := buildIncaSleep(1800, length)
		if len(r) != length || r[offset+7] != 0x07 || r[offset+8] != 0x08 {
			t.Fatalf("bad sleep encoding %x", r)
		}
		r = buildIncaCmdVal(0x00, 0x80, length)
		if r[offset+5] != 0 || r[offset+7] != 0x80 {
			t.Fatalf("bad poll report %x", r)
		}
	}
	if _, err := pollingToYY(125); err == nil {
		t.Fatal("invalid polling accepted")
	}
}
