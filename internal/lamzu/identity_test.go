package lamzu

import (
	"errors"
	"testing"
)

func TestIdentityModelsAndUnknownCode(t *testing.T) {
	for _, tc := range []struct {
		pid   uint16
		model string
	}{
		{0x0009, "LAMZU INCA"},
		{0x0007, "LAMZU PARO"},
		{0x1234, ""},
		{0, ""},
	} {
		reply := []byte{0xa1, 0, 2, 6, 0, 0x81, 0, 0, 0, 19, byte(tc.pid >> 8), byte(tc.pid)}
		for _, report := range [][]byte{reply, append(append([]byte{}, reply...), make([]byte, 52)...), append([]byte{0}, append(append([]byte{}, reply...), make([]byte, 52)...)...)} {
			id, err := ParseIdentity(report)
			if err != nil || id.MousePID != tc.pid || id.Model != tc.model {
				t.Fatalf("PID %04x: identity=%+v, error=%v", tc.pid, id, err)
			}
		}
	}
}

func TestIdentityRejectsOtherRoutesAndTruncatedPID(t *testing.T) {
	reply := []byte{0xa1, 0, 2, 6, 0, 0x81, 0, 0, 0, 19, 0, 9}
	for _, tc := range []struct {
		name  string
		index int
		value byte
	}{
		{"pending", 0, 0xa0},
		{"rejected", 0, 0xa3},
		{"wrong channel", 1, 1},
		{"receiver route", 2, 0},
		{"USB route", 2, 1},
		{"wrong length", 3, 4},
		{"profile group", 4, 1},
		{"other command", 5, 0x8b},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := append([]byte{}, reply...)
			p[tc.index] = tc.value
			if _, err := ParseIdentity(p); !errors.Is(err, ErrIdentityUnavailable) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
	for size := range len(reply) {
		if _, err := ParseIdentity(reply[:size]); !errors.Is(err, ErrIdentityUnavailable) {
			t.Fatalf("truncated size %d: error=%v", size, err)
		}
	}
	if _, err := ParseIdentity(append([]byte{1}, append(reply, make([]byte, 52)...)...)); !errors.Is(err, ErrIdentityUnavailable) {
		t.Fatalf("nonzero report ID: error=%v", err)
	}
}

func TestIdentityNoPeerIsDistinctFromPendingAndZeroCode(t *testing.T) {
	p := []byte{0xa4, 0, 2, 0x10, 0, 0x81}
	if _, err := ParseIdentity(p); !errors.Is(err, ErrNoMouse) {
		t.Fatalf("explicit no-peer response: %v", err)
	}
	p[0] = 0xa0
	if _, err := ParseIdentity(p); !errors.Is(err, ErrIdentityUnavailable) || errors.Is(err, ErrNoMouse) {
		t.Fatalf("pending reply conflated with no peer: %v", err)
	}
	p[0], p[2] = 0xa4, 0
	if _, err := ParseIdentity(p); !errors.Is(err, ErrIdentityUnavailable) {
		t.Fatalf("receiver route no-peer cannot identify radio state: %v", err)
	}
	id, err := ParseIdentity([]byte{0xa1, 0, 2, 6, 0, 0x81, 0, 0, 0, 19, 0, 0})
	if err != nil || id.MousePID != 0 || id.Model != "" {
		t.Fatalf("valid unknown zero identity: %+v, %v", id, err)
	}
}
