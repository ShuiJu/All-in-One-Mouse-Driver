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
}

func (h *stateRefreshHardware) ReadSettings(d Device) (Parameters, error) {
	return h.settings[d.ID], h.readErr
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
