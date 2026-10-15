package vaxee

import (
	"errors"
	"testing"
)

func TestIdentityUsesMousePID(t *testing.T) {
	for _, tc := range []struct {
		low  byte
		name string
	}{{0x10, "VAXEE E1"}, {0x11, "VAXEE NP-01S V2"}} {
		r := []byte{0x0e, 0xa5, 0x0a, 1, 2, tc.low, 0x10}
		got, err := ParseIdentity(r)
		if err != nil || got.Model != tc.name || got.MousePID != 0x1000+uint16(tc.low) {
			t.Fatalf("got %+v %v", got, err)
		}
	}
}
func TestIdentityPreservesUnknownMousePID(t *testing.T) {
	got, err := ParseIdentity([]byte{0x0e, 0xa5, 0x0a, 1, 2, 0x34, 0x12})
	if err != nil || got.MousePID != 0x1234 || got.Model != "" {
		t.Fatalf("unknown identity lost: %+v %v", got, err)
	}
}

func TestIdentityRejectsUnavailableResponses(t *testing.T) {
	for _, r := range [][]byte{
		nil,
		{0x0e, 0xa5, 0x0a, 1, 2, 0x10},
		{0x0d, 0xa5, 0x0a, 1, 2, 0x10, 0x10},
		{0x0e, 0xa4, 0x0a, 1, 2, 0x10, 0x10},
		{0x0e, 0xa5, 0x0b, 1, 2, 0x10, 0x10},
		{0x0e, 0xa5, 0x0a, 2, 2, 0x10, 0x10},
		{0x0e, 0xa5, 0x0a, 3, 2, 0x10, 0x10},
		{0x0e, 0xa5, 0x0a, 1, 1, 0x10, 0x10},
		{0x0e, 0xa5, 0x0a, 1, 3, 0x10, 0x10},
		{0x0e, 0xa5, 0x0a, 1, 2, 0, 0},
		{0x0e, 0xa5, 0x0a, 1, 2, 0xff, 0xff},
	} {
		got, err := ParseIdentity(r)
		if !errors.Is(err, ErrIdentityResponse) || !errors.Is(err, ErrIdentityUnavailable) || got != (Identity{}) {
			t.Fatalf("invalid identity %x returned %+v %v", r, got, err)
		}
	}
}

func TestIdentityAcceptsPaddedFeatureReport(t *testing.T) {
	report := make([]byte, 64)
	copy(report, []byte{0x0e, 0xa5, 0x0a, 1, 2, 0x10, 0x10})
	got, err := ParseIdentity(report)
	if err != nil || got.MousePID != 0x1010 || got.Model != "VAXEE E1" {
		t.Fatalf("padded identity returned %+v %v", got, err)
	}
}
