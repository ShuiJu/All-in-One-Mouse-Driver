package hub

import (
	"errors"
	"testing"
)

type stateRefreshHardware struct {
	*fakeHardware
	settings map[string]Parameters
	battery  int
	readErr  error
	reads    []string
}

func (h *stateRefreshHardware) ReadSettings(d Device) (Parameters, error) {
	h.reads = append(h.reads, d.ID)
	return h.settings[d.ID], h.readErr
}

func TestRepeatedNotificationsKeepDailyAndCompetitiveApplyCache(t *testing.T) {
	e, hw, foreground, _ := setup(t)
	hw.devices = append(hw.devices, Device{ID: "op1", Brand: "endgame", Name: "OP1w 4K v2", IdentityStatus: IdentityCurrent})
	for _, process := range []string{"notepad.exe", "cs2.exe"} {
		*foreground = process
		e.step(false)
		before := len(hw.writes)
		events := len(e.Snapshot().Events)
		for range 10 {
			e.DevicesChanged()
			e.DevicesChanged()
			e.step(false)
			e.RefreshDevices()
			e.step(false)
			e.Wake()
			e.step(false)
		}
		if len(hw.writes) != before || len(e.Snapshot().Events) != events {
			t.Fatalf("%s repeatedly reapplied unchanged settings: before=%d after=%d", process, before, len(hw.writes))
		}
	}
	if len(hw.writes) != 8 {
		t.Fatalf("daily/game transitions should apply once per device: %v", hw.writes)
	}
	*foreground = "another-unbound-app.exe"
	e.step(false)
	before := len(hw.writes)
	*foreground = "notepad.exe"
	e.step(false)
	if len(hw.writes) != before {
		t.Fatal("same fallback profile reapplied for another unbound app")
	}
}

func TestNotificationForAnotherDeviceDoesNotReapplyExistingMice(t *testing.T) {
	e, hw, _, _ := setup(t)
	e.step(false)
	hw.devices = append(hw.devices, Device{ID: "op1", Brand: "endgame", IdentityStatus: IdentityCurrent})
	e.DevicesChanged()
	e.step(false)
	if len(hw.writes) != 4 || hw.writes[3] != "op1" {
		t.Fatalf("adding mouse rewrote other devices: %v", hw.writes)
	}
	hw.devices = hw.devices[:3]
	e.DeviceRemoved(`\\?\hid#unrelated-device`)
	e.step(false)
	if len(hw.writes) != 4 {
		t.Fatalf("removing unrelated device rewrote mice: %v", hw.writes)
	}
	// A separately observed disconnection still invalidates that device's cache.
	saved := hw.devices[1]
	hw.devices = append(hw.devices[:1:1], hw.devices[2:]...)
	e.DevicesChanged()
	e.step(false)
	hw.devices = append(hw.devices, saved)
	e.DevicesChanged()
	e.step(false)
	if len(hw.writes) != 5 || hw.writes[4] != "v1" {
		t.Fatalf("reconnect did not apply only returning mouse: %v", hw.writes)
	}
}

func TestCoalescedReconnectChecksOnlyRemovedReceiver(t *testing.T) {
	e, base, _, _ := setup(t)
	e.step(false)
	hw := &stateRefreshHardware{fakeHardware: base, settings: map[string]Parameters{}}
	for _, d := range e.Snapshot().Devices {
		hw.settings[d.ID] = *d.Applied
	}
	e.hw = hw
	e.DeviceRemoved(base.devices[1].path)
	e.DevicesChanged()
	e.step(false)
	if len(base.writes) != 3 || len(hw.reads) != 1 || hw.reads[0] != "v1" {
		t.Fatalf("unchanged reconnect rewrote mice: writes=%v reads=%v", base.writes, hw.reads)
	}
	reset := hw.settings["v1"]
	reset.MotionSync = !reset.MotionSync
	hw.settings["v1"] = reset
	e.DeviceRemoved(base.devices[1].path)
	e.step(false)
	if len(base.writes) != 4 || base.writes[3] != "v1" {
		t.Fatalf("reset receiver did not exclusively reapply: %v", base.writes)
	}
}

func (h *stateRefreshHardware) Battery(Device) (*int, bool) {
	h.batteries++
	value := h.battery
	return &value, false
}

func TestStatusRefreshUpdatesObservedStateWithoutReapplying(t *testing.T) {
	e, base, _, _ := setup(t)
	e.step(false)
	hw := &stateRefreshHardware{fakeHardware: base, settings: map[string]Parameters{}, battery: 42}
	for _, d := range e.Snapshot().Devices {
		hw.settings[d.ID] = *d.Applied
	}
	e.hw = hw
	observed := hw.settings["l1"]
	observed.Poll = 4000
	hw.settings["l1"] = observed
	before := e.Counters()
	for range 3 {
		e.RefreshDevices()
	}
	e.step(false)
	after := e.Counters()
	if after.DeviceScans != before.DeviceScans+1 || after.Writes != before.Writes {
		t.Fatalf("coalesced status refresh rescanned or rewrote: before=%+v after=%+v", before, after)
	}
	if after.BatteryQueries != before.BatteryQueries+3 {
		t.Fatalf("foreground refresh used battery cache: %+v", after)
	}
	for _, d := range e.Snapshot().Devices {
		if d.Battery == nil || *d.Battery != 42 || d.Applied == nil || *d.Applied != hw.settings[d.ID] {
			t.Fatalf("fresh hardware state missing: %+v", d)
		}
	}
	e.step(false)
	if e.Counters().DeviceScans != after.DeviceScans || e.Counters().BatteryQueries != after.BatteryQueries {
		t.Fatal("ordinary foreground event repeated HID queries")
	}
	if e.Snapshot().Devices[0].Applied.Poll != 4000 {
		t.Fatal("cached desired configuration replaced observed hardware state")
	}
	hw.battery = 57
	e.RefreshDevices()
	e.step(false)
	if *e.Snapshot().Devices[0].Battery != 57 || e.Counters().Writes != before.Writes {
		t.Fatal("second foreground refresh was throttled or forced configuration writes")
	}
}

func TestStatusRefreshWhilePausedClearsUnconfirmedAppliedState(t *testing.T) {
	e, base, _, _ := setup(t)
	base.devices[0].MouseModel = "LAMZU INCA"
	base.devices[0].MousePID = 9
	base.devices[0].IdentityStatus = IdentityCurrent
	e.step(false)
	hw := &stateRefreshHardware{fakeHardware: base, settings: map[string]Parameters{}, battery: 42}
	for _, d := range e.Snapshot().Devices {
		hw.settings[d.ID] = *d.Applied
	}
	e.hw = hw
	config := e.Snapshot().Config
	config.AutoEnabled = false
	if err := e.Update(config); err != nil {
		t.Fatal(err)
	}
	writes := len(base.writes)
	hw.readErr = errors.New("read timeout")
	e.RefreshDevices()
	e.step(false)
	device := e.Snapshot().Devices[0]
	if len(base.writes) != writes || device.Applied != nil || device.IdentityStatus != IdentityCached {
		t.Fatalf("failed read retained a current confirmation or wrote settings: %+v", device)
	}
	hw.readErr = nil
	e.RefreshDevices()
	e.step(false)
	device = e.Snapshot().Devices[0]
	if device.IdentityStatus != IdentityCurrent || device.Applied == nil || *device.Applied != hw.settings[device.ID] || len(base.writes) != writes {
		t.Fatalf("paused foreground refresh did not recover actual settings: %+v", device)
	}
}
