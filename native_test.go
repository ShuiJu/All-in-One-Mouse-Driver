package main

import (
	"mousehub/internal/hub"
	"testing"
	"unsafe"
)

func TestNativeWin64ABI(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("Win64 layout")
	}
	var d drawItem
	if unsafe.Offsetof(d.Window) != 24 || unsafe.Offsetof(d.DC) != 32 || unsafe.Sizeof(d) != 64 {
		t.Fatalf("DRAWITEMSTRUCT ABI mismatch: %d", unsafe.Sizeof(d))
	}
	var o openFileName
	if unsafe.Sizeof(o) != 152 {
		t.Fatalf("OPENFILENAMEW size %d", unsafe.Sizeof(o))
	}
	var ri rawInputDevice
	if unsafe.Sizeof(ri) != 16 {
		t.Fatalf("RAWINPUTDEVICE size %d", unsafe.Sizeof(ri))
	}
	var rh rawInputHeader
	if unsafe.Sizeof(rh) != 24 {
		t.Fatalf("RAWINPUTHEADER size %d", unsafe.Sizeof(rh))
	}
}
func TestNativeBrandSceneIsolation(t *testing.T) {
	u := &nativeUI{config: hub.DefaultConfig(), brand: "lamzu", editingProfile: "game"}
	u.parameters().Poll = 8000
	u.brand = "vaxee"
	if u.parameters().Poll == 8000 {
		t.Fatal("brand parameters leaked")
	}
	u.editingProfile = "daily"
	u.parameters().Trajectory = "smooth"
	u.brand = "lamzu"
	if u.parameters().Poll != 1000 {
		t.Fatal("game parameters leaked into daily")
	}
}
func TestDevicePageBounds(t *testing.T) {
	tests := []struct {
		count, page, start, end, max int
	}{
		{0, 0, 0, 0, 0},
		{1, 0, 0, 1, 0},
		{4, 0, 0, 4, 0},
		{5, 0, 0, 4, 1},
		{5, 1, 4, 5, 1},
		{5, 9, 4, 5, 1},
	}
	for _, tt := range tests {
		start, end, max := devicePageBounds(tt.count, tt.page)
		if start != tt.start || end != tt.end || max != tt.max {
			t.Errorf("devicePageBounds(%d, %d) = %d, %d, %d; want %d, %d, %d", tt.count, tt.page, start, end, max, tt.start, tt.end, tt.max)
		}
	}
}
