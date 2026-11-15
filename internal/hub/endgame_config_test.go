package hub

import (
	"encoding/json"
	"errors"
	"mousehub/internal/endgame"
	"testing"
)

func TestEndgameDefaultsAndV2Compatibility(t *testing.T) {
	legacy := `{"version":2,"active":"daily","autoEnabled":true,"intervalMS":1000,"profiles":[{"id":"daily","name":"daily","settings":{"lamzu":{"mode":"office","poll":1000,"motionSync":false,"sleep":30},"vaxee":{"mode":"standard","poll":1000,"motionSync":false,"sleep":0,"trajectory":"stable"}}},{"id":"custom-game","name":"game","settings":{"lamzu":{"mode":"20000fps","poll":2000,"motionSync":false,"sleep":600},"vaxee":{"mode":"competitive","poll":4000,"motionSync":false,"sleep":0,"trajectory":"smooth"}}}],"bindings":[]}`
	c, err := DecodeConfig([]byte(legacy))
	if err != nil {
		t.Fatal(err)
	}
	daily := c.Profiles[0].Settings["endgame"]
	game := c.Profiles[1].Settings["endgame"]
	if *daily != (Parameters{Mode: "standard", Poll: 1000, LOD: 10, PowerSaving: 1, DeepSleep: 2}) {
		t.Fatalf("daily defaults = %+v", *daily)
	}
	if *game != (Parameters{Mode: "competitive", Poll: 4000, LOD: 10, DeepSleep: 3}) {
		t.Fatalf("game defaults = %+v", *game)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var serialized struct {
		Profiles []struct {
			Settings map[string]map[string]any `json:"settings"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &serialized); err != nil {
		t.Fatal(err)
	}
	for _, profile := range serialized.Profiles {
		endgame := profile.Settings["endgame"]
		if profile.Settings["endgame"]["mode"] == "competitive" {
			if _, ok := endgame["powerSaving"]; ok {
				t.Fatalf("zero Endgame power saving was serialized: %s", raw)
			}
		}
	}
}

func TestEndgameValidationAndComparison(t *testing.T) {
	c := DefaultConfig()
	p := c.Profiles[0].Settings["endgame"]
	for _, poll := range []int{125, 1000, 2000, 4000} {
		p.Poll = poll
		p.PowerSaving = 0
		if err := c.Validate(); err != nil {
			t.Fatalf("poll %d rejected: %v", poll, err)
		}
	}
	p.Poll = 8000
	if err := c.Validate(); err == nil {
		t.Fatal("unsupported Endgame polling accepted")
	}
	p.Poll = 1000
	p.LOD = 18
	if err := c.Validate(); err == nil {
		t.Fatal("unsupported LOD accepted")
	}
	p.LOD = 20
	p.PowerSaving = 121
	if err := c.Validate(); err == nil {
		t.Fatal("unsupported power saving accepted")
	}
	p.PowerSaving = 1
	p.DeepSleep = 2
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Poll = 4000
	if err := c.Validate(); err == nil {
		t.Fatal("Endgame power saving was accepted above 1000 Hz")
	}
	p.Poll = 1000
	p.PowerSaving = 1
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Poll = 125
	if err := c.Validate(); err == nil {
		t.Fatal("Endgame power saving was accepted at 125 Hz")
	}
	p.Poll = 1000
	actual := *p
	actual.Sleep = 42
	actual.Trajectory = "stable"
	if !sameParameters("endgame", actual, *p) {
		t.Fatal("Endgame comparison included unsupported legacy fields")
	}
	c.Profiles[0].Settings["unknown"] = &Parameters{}
	if err := c.Validate(); err == nil {
		t.Fatal("unknown device brand accepted")
	}
	if err := validateParams("unknown", Parameters{}); err == nil {
		t.Fatal("unknown brand parameter validation accepted")
	}
}

func TestEndgameSettingsConversion(t *testing.T) {
	want := Parameters{Mode: "competitive", Poll: 4000, MotionSync: true, LOD: 10, PowerSaving: 0, DeepSleep: 3}
	if got := endgameSettings(want); got != (endgame.Settings{Poll: 4000, LOD: 10, DeepSleep: 3, ForceMaxFPS: true, MotionSync: true}) {
		t.Fatalf("converted settings = %+v", got)
	}
	if got := endgameParameters(endgame.Settings{Poll: 1000, LOD: 7, PowerSaving: 1, DeepSleep: 2}); got != (Parameters{Mode: "standard", Poll: 1000, LOD: 7, PowerSaving: 1, DeepSleep: 2}) {
		t.Fatalf("read settings = %+v", got)
	}
}

func TestEndgameProgramBindingAndCache(t *testing.T) {
	e, hw, foreground, _ := setup(t)
	hw.devices = append(hw.devices,
		Device{ID: "op1w-a", Brand: "endgame", Name: "OP1w 4K v2", path: `\\?\hid#vid_3367&pid_1970#op1w-a`, IdentityStatus: IdentityCurrent},
		Device{ID: "op1w-b", Brand: "endgame", Name: "OP1w 4K v2", path: `\\?\hid#vid_3367&pid_1970#op1w-b`, IdentityStatus: IdentityCurrent},
	)
	e.Tick()
	if len(hw.writes) != 5 {
		t.Fatalf("initial write count = %d, want 5", len(hw.writes))
	}
	for i := 3; i < 5; i++ {
		got := hw.values[i]
		if got.Mode != "standard" || got.Poll != 1000 || got.LOD != 10 || got.PowerSaving != 1 || got.DeepSleep != 2 || got.MotionSync {
			t.Fatalf("daily Endgame parameters = %+v", got)
		}
	}
	*foreground = "cs2.exe"
	e.Tick()
	if len(hw.writes) != 10 {
		t.Fatalf("program switch write count = %d, want 10", len(hw.writes))
	}
	for i := 8; i < 10; i++ {
		got := hw.values[i]
		if got.Mode != "competitive" || got.Poll != 4000 || got.LOD != 10 || got.PowerSaving != 0 || got.DeepSleep != 3 || got.MotionSync {
			t.Fatalf("game Endgame parameters = %+v", got)
		}
	}
	c := e.Snapshot().Config
	c.Profiles[1].Settings["endgame"].Sleep = 99
	c.Profiles[1].Settings["endgame"].Trajectory = "stable"
	if err := e.Update(c); err != nil {
		t.Fatal(err)
	}
	e.Tick()
	if len(hw.writes) != 10 {
		t.Fatalf("legacy Endgame fields caused redundant writes: %d", len(hw.writes))
	}
}

func TestEndgameUnavailableSkipsApply(t *testing.T) {
	e, hw, _, _ := setup(t)
	hw.devices = append(hw.devices, Device{ID: "op1w-unavailable", Brand: "endgame", Name: "OP1w 4K v2", IdentityStatus: IdentityUnavailable})
	e.Tick()
	if len(hw.writes) != 3 {
		t.Fatalf("unavailable Endgame device was applied: %v", hw.writes)
	}
	for _, d := range e.Snapshot().Devices {
		if d.ID == "op1w-unavailable" && d.Status != "离线" {
			t.Fatalf("unavailable Endgame status = %q", d.Status)
		}
	}
}

func TestEndgameMouseActivityRecoversCachedDevice(t *testing.T) {
	e, hw, _, _ := setup(t)
	path := `\\?\hid#vid_3367&pid_1970#op1w-sleeping`
	hw.devices = append(hw.devices, Device{ID: "op1w-sleeping", Brand: "endgame", Name: "OP1w 4K v2", MouseModel: "OP1w 4K v2", IdentityStatus: IdentityCurrent, path: path})
	e.Tick()
	if len(hw.writes) != 4 {
		t.Fatalf("initial writes = %d, want 4", len(hw.writes))
	}
	hw.devices[len(hw.devices)-1] = Device{ID: "op1w-sleeping", Brand: "endgame", Name: "OP1w 4K v2", IdentityStatus: IdentityUnavailable, path: path}
	e.Tick()
	if len(hw.writes) != 4 || !e.NeedsRecovery() {
		t.Fatalf("sleeping device wrote or was not queued for recovery: writes=%v recovery=%v", hw.writes, e.NeedsRecovery())
	}
	before := e.Counters()
	if !e.NotifyMouseActivity(path) {
		t.Fatal("matching Endgame activity was not accepted")
	}
	e.step(false)
	after := e.Counters()
	if after.DeviceScans != before.DeviceScans || len(hw.writes) != 5 || hw.writes[4] != "op1w-sleeping" {
		t.Fatalf("activity recovery scanned or wrote wrong device: before=%+v after=%+v writes=%v", before, after, hw.writes)
	}
	for _, d := range e.Snapshot().Devices {
		if d.ID == "op1w-sleeping" && (d.IdentityStatus != IdentityCurrent || d.Applied == nil || d.Status != "自动切换已暂停") {
			t.Fatalf("activity did not restore Endgame device: %+v", d)
		}
	}
	if e.NeedsRecovery() {
		t.Fatal("successful activity recovery remained pending")
	}
}

type endgameReadbackHardware struct {
	*fakeHardware
	readErr error
}

func (h *endgameReadbackHardware) ReadSettings(d Device) (Parameters, error) {
	if d.Brand == "endgame" && h.readErr != nil {
		return Parameters{}, h.readErr
	}
	return Parameters{}, nil
}

func TestEndgameStatusRefreshFailureQueuesActivityRecovery(t *testing.T) {
	e, base, _, _ := setup(t)
	path := `\\?\hid#vid_3367&pid_1970#op1w-refresh`
	base.devices = append(base.devices, Device{ID: "op1w-refresh", Brand: "endgame", Name: "OP1w 4K v2", MouseModel: "OP1w 4K v2", IdentityStatus: IdentityCurrent, path: path})
	e.Tick()
	hw := &endgameReadbackHardware{fakeHardware: base, readErr: errors.New("mouse sleeping")}
	e.hw = hw
	e.RefreshDevices()
	e.step(false)
	if !e.NeedsRecovery() {
		t.Fatal("Endgame status refresh failure did not queue recovery")
	}
	hw.readErr = nil
	before := e.Counters()
	if !e.NotifyMouseActivity(path) {
		t.Fatal("activity did not target Endgame refresh recovery")
	}
	e.step(false)
	after := e.Counters()
	if after.DeviceScans != before.DeviceScans || base.writes[len(base.writes)-1] != "op1w-refresh" {
		t.Fatalf("refresh recovery scanned or missed Endgame device: before=%+v after=%+v writes=%v", before, after, base.writes)
	}
	if e.NeedsRecovery() {
		t.Fatal("successful status-refresh recovery remained pending")
	}
}
