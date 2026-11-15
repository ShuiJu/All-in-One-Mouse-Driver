package main

import (
	"mousehub/internal/hub"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestDeviceRemovalNotificationCarriesOnlyItsInterfacePath(t *testing.T) {
	path := `\\?\hid#vid_3367&pid_1970&mi_02#receiver-a`
	encoded, err := syscall.UTF16FromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]uint32, (28+len(encoded)*2+3)/4)
	buf[0], buf[1] = uint32(28+len(encoded)*2), 5
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(&buf[0]))+28)), len(encoded)), encoded)
	if got := deviceNotificationPath(uintptr(unsafe.Pointer(&buf[0]))); got != path {
		t.Fatalf("notification path=%q", got)
	}
	buf[1] = 2
	if got := deviceNotificationPath(uintptr(unsafe.Pointer(&buf[0]))); got != "" {
		t.Fatal("non-interface notification parsed as HID path")
	}
	if deviceNotificationPath(0) != "" {
		t.Fatal("empty notification parsed as path")
	}
}

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
func TestEndgamePresetsKeepLODAndSetAllControls(t *testing.T) {
	p := hub.Parameters{LOD: 20, MotionSync: true, Poll: 2000, PowerSaving: 120, DeepSleep: 120}
	applyEndgamePreset(&p, "standard")
	if p.Mode != "standard" || p.Poll != 1000 || p.MotionSync || p.PowerSaving != 1 || p.DeepSleep != 2 || p.LOD != 20 {
		t.Fatalf("standard preset = %+v", p)
	}
	applyEndgamePreset(&p, "competitive")
	if p.Mode != "competitive" || p.Poll != 4000 || p.MotionSync || p.PowerSaving != 0 || p.DeepSleep != 3 || p.LOD != 20 {
		t.Fatalf("competitive preset = %+v", p)
	}
}
func TestEndgameSummaryIncludesAllControls(t *testing.T) {
	s := endgameSummary(hub.Parameters{LOD: 10, Mode: "competitive", Poll: 4000, MotionSync: false, PowerSaving: 0, DeepSleep: 3})
	for _, want := range []string{"LOD 1.0 mm", "4000 Hz", "最高帧数开启", "Motion Sync 关闭", "省电关闭", "深度休眠3 分钟"} {
		if !strings.Contains(s, want) {
			t.Fatalf("summary %q missing %q", s, want)
		}
	}
}
func TestEndgameHighPollingDisablesPowerSaving(t *testing.T) {
	p := hub.Parameters{Poll: 1000, PowerSaving: 30}
	setEndgamePoll(&p, 4000)
	if p.Poll != 4000 || p.PowerSaving != 0 {
		t.Fatalf("high polling parameters = %+v", p)
	}
	p.PowerSaving = 30
	setEndgamePoll(&p, 125)
	if p.Poll != 125 || p.PowerSaving != 0 {
		t.Fatalf("125 Hz parameters = %+v", p)
	}
}
func TestEndgameLODOptionsFollowGlassMode(t *testing.T) {
	u := &nativeUI{selected: "op1", state: hub.State{Devices: []hub.Device{{ID: "op1", Brand: "endgame"}}}}
	if got := u.endgameLODOptions(); len(got) != 11 || got[0] != 7 || got[len(got)-1] != 17 {
		t.Fatalf("standard LOD options = %v", got)
	}
	u.state.Devices[0].GlassMode = true
	if got := u.endgameLODOptions(); len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Fatalf("glass LOD options = %v", got)
	}
}
func TestBrandDisplayName(t *testing.T) {
	if got := brandDisplayName("endgame"); got != "Endgame Gear" {
		t.Fatalf("endgame display name = %q", got)
	}
}
func TestQAEndgameFixtureIsReadOnlyOP1w4KV2(t *testing.T) {
	devices, err := (readOnlyHardware{}).Discover()
	if err != nil || len(devices) != 1 {
		t.Fatalf("qa devices = %#v, %v", devices, err)
	}
	d := devices[0]
	if d.Brand != "endgame" || d.MouseModel != "OP1w 4K v2" || d.Name != "Endgame Gear OP1w 4K v2" {
		t.Fatalf("qa device = %+v", d)
	}
	if err := (readOnlyHardware{}).Apply(d, hub.Parameters{}); err == nil {
		t.Fatal("qa hardware accepted a device write")
	}
}
func TestQAModeIgnoresExternalLegacyProcessChecks(t *testing.T) {
	old := qaMode
	defer func() { qaMode = old }()
	qaMode = true
	if got := legacyProcesses(); len(got) != 0 {
		t.Fatalf("qa legacy processes = %v", got)
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
