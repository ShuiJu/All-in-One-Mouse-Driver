package vaxee

import "testing"

func TestIdentityUsesMousePID(t *testing.T) {
	for _, tc := range []struct {
		low  byte
		name string
	}{{0x10, "VAXEE E1"}, {0x11, "VAXEE NP-01S V2"}} {
		r := []byte{0x0e, 0xa5, 0x0a, 1, 2, tc.low, 0x10}
		got, err := ParseIdentity(r)
		if err != nil || got.Model != tc.name {
			t.Fatalf("got %+v %v", got, err)
		}
	}
}
func TestIdentityRejectsStaleAndUnknownReports(t *testing.T) {
	for _, r := range [][]byte{nil, {0x0e, 0xa5, 0x0b, 1, 2, 0x10, 0x10}, {0x0e, 0xa5, 0x0a, 2, 2, 0x10, 0x10}, {0x0e, 0xa5, 0x0a, 1, 2, 0xff, 0xff}} {
		if _, err := ParseIdentity(r); err == nil {
			t.Fatalf("accepted %x", r)
		}
	}
}
