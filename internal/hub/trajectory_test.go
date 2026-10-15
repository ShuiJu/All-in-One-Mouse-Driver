package hub

import (
	"path/filepath"
	"testing"
)

func TestLamzuTrajectoryPersistenceAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	for _, trajectory := range []string{"", "smooth", "stable"} {
		c := DefaultConfig()
		c.Profiles[0].Settings["lamzu"].Trajectory = trajectory
		if err := SaveConfig(path, c); err != nil {
			t.Fatal(err)
		}
		loaded, err := LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		p, _ := loaded.Resolve(Device{Brand: "lamzu"}, "notepad.exe")
		if p.Trajectory != trajectory || p.Mode != "office" || p.Sleep != 30 {
			t.Fatalf("profile round trip changed parameters: %+v", p)
		}
	}
	c := DefaultConfig()
	c.Profiles[0].Settings["lamzu"].Trajectory = "unknown"
	if err := c.Validate(); err == nil {
		t.Fatal("unknown INCA trajectory accepted")
	}
}

func TestLamzuTrajectorySwitchAndReconnect(t *testing.T) {
	e, hw, fg, _ := setup(t)
	c := e.Snapshot().Config
	c.Profiles[0].Settings["lamzu"].Trajectory = "stable"
	// Hold other INCA settings constant to isolate a trajectory-only change.
	game := *c.Profiles[0].Settings["lamzu"]
	game.Trajectory = "smooth"
	c.Profiles[1].Settings["lamzu"] = &game
	if err := e.Update(c); err != nil {
		t.Fatal(err)
	}
	e.Tick()
	if e.Snapshot().Devices[0].Applied.Trajectory != "stable" {
		t.Fatal("daily trajectory was not applied")
	}
	*fg = "cs2.exe"
	e.Tick()
	if e.Snapshot().Devices[0].Applied.Trajectory != "smooth" {
		t.Fatal("trajectory-only profile switch was deduplicated")
	}
	writes := len(hw.writes)
	e.Tick()
	if len(hw.writes) != writes {
		t.Fatal("unchanged trajectory triggered another apply")
	}
	e.DevicesChanged()
	e.Tick()
	if len(hw.writes) != writes+3 || e.Snapshot().Devices[0].Applied.Trajectory != "smooth" {
		t.Fatal("same-path reconnect did not restore selected trajectory")
	}
	*fg = "notepad.exe"
	e.Tick()
	if e.Snapshot().Devices[0].Applied.Trajectory != "stable" {
		t.Fatal("return to daily profile did not restore stable")
	}
}
