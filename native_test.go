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
