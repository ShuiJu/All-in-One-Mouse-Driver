package hub

import (
	"encoding/json"
	"mousehub/internal/vaxee"
	"os"
	"path/filepath"
	"testing"
)

func TestBindingsSelectIndependentProfilesAndFallback(t *testing.T) {
	c := DefaultConfig()
	d := Device{ID: "inca1", Brand: "lamzu"}
	if !c.AutoEnabled {
		t.Fatal("automatic switching must default on")
	}
	_ = c.Bind("C:\\games\\CS2.EXE", "game")
	p, id := c.Resolve(d, "cs2.exe")
	if id != "game" || p.Mode != "20000fps" {
		t.Fatal(p, id)
	}
	p, id = c.Resolve(d, "notepad.exe")
	if id != "daily" || p.Mode != "office" {
		t.Fatal(p, id)
	}
	_ = c.Bind("cs2.exe", "daily")
	if len(c.Bindings) != 1 {
		t.Fatal("rebind duplicated application")
	}
	_, id = c.Resolve(d, "cs2.exe")
	if id != "daily" {
		t.Fatal(id)
	}
	if c.Bind("broken.exe", "missing") == nil {
		t.Fatal("dangling binding accepted")
	}
}
func TestSameModelDeviceSettingsRemainIndependent(t *testing.T) {
	c := DefaultConfig()
	a := Device{ID: "receiver-a", Brand: "vaxee", MousePID: 0x1010}
	b := Device{ID: "receiver-b", Brand: "vaxee", MousePID: 0x1010}
	p := &c.Profiles[1]
	p.EditDevice(a).Poll = 4000
	p.EditDevice(b).Poll = 2000
	if p.ForDevice(a).Poll != 4000 || p.ForDevice(b).Poll != 2000 || p.Settings["vaxee"].Poll != 1000 {
		t.Fatal("same-model devices shared settings")
	}
	cloned := Clone(c)
	cloned.Profiles[1].EditDevice(a).Poll = 1000
	if c.Profiles[1].ForDevice(a).Poll != 4000 {
		t.Fatal("device settings alias clone")
	}
	_ = c.Bind("game.exe", "game")
	pa, _ := c.Resolve(a, "game.exe")
	pb, _ := c.Resolve(b, "game.exe")
	if pa.Poll != 4000 || pb.Poll != 2000 {
		t.Fatal("routing ignored device identity")
	}
}
func TestInstanceIDsDoNotDependOnEnumerationOrder(t *testing.T) {
	ds := []Device{{Brand: "vaxee", path: "port1", serial: "duplicated", v: vaxee.VaxeeDeviceInfo{ContainerID: "a"}}, {Brand: "vaxee", path: "port2", serial: "duplicated", v: vaxee.VaxeeDeviceInfo{ContainerID: "b"}}}
	assignDeviceIDs(ds)
	if ds[0].ID == ds[1].ID {
		t.Fatal("duplicated serial merged devices")
	}
	a, b := ds[0].ID, ds[1].ID
	ds[0], ds[1] = ds[1], ds[0]
	assignDeviceIDs(ds)
	if ds[0].ID != b || ds[1].ID != a {
		t.Fatal("enumeration order changed identity")
	}
	single := []Device{ds[0]}
	assignDeviceIDs(single)
	if single[0].ID != ds[0].ID {
		t.Fatal("unplugging a peer changed identity")
	}
	one := []Device{{Brand: "vaxee", path: "port1", serial: "real-serial", v: vaxee.VaxeeDeviceInfo{ContainerID: "stable-container"}}}
	assignDeviceIDs(one)
	id := one[0].ID
	one[0].path = "port2"
	assignDeviceIDs(one)
	if one[0].ID != id {
		t.Fatal("stable container failed across ports")
	}
}
func TestDeletingProfileUnbindsProgramButProtectsDefault(t *testing.T) {
	c := DefaultConfig()
	_ = c.Bind("game.exe", "game")
	if err := c.RemoveProfile("daily"); err == nil {
		t.Fatal("deleted default profile")
	}
	if err := c.RemoveProfile("game"); err != nil {
		t.Fatal(err)
	}
	if c.Bindings[0].ProfileID != "" {
		t.Fatal("dangling association")
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestV1MigrationPreservesValuesAndWritesBackup(t *testing.T) {
	old := `{"version":1,"active":"default","intervalMS":1000,"profiles":[{"id":"default","name":"日常与游戏","lamzu":{"enabled":false,"default":{"mode":"office","poll":1000,"sleep":30},"game":{"mode":"20000fps","poll":8000,"motionSync":true,"sleep":600},"processes":["CS2.EXE"]},"vaxee":{"enabled":false,"default":{"mode":"standard","poll":1000,"trajectory":"stable"},"game":{"mode":"competitive","poll":4000,"trajectory":"smooth"},"processes":[]}}]}`
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(old), 0600)
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != 2 || !c.AutoEnabled || len(c.Profiles) != 2 {
		t.Fatal("migration shape")
	}
	p, _ := c.Resolve(Device{Brand: "lamzu"}, "cs2.exe")
	if p.Poll != 8000 || !p.MotionSync {
		t.Fatal("game configuration lost")
	}
	p, _ = c.Resolve(Device{Brand: "vaxee"}, "cs2.exe")
	if p.Mode != "standard" {
		t.Fatal("brand-specific migration changed VAXEE association")
	}
	backup, _ := os.ReadFile(path + ".v1.bak")
	if string(backup) != old {
		t.Fatal("backup differs")
	}
	raw, _ := os.ReadFile(path)
	var header struct{ Version int }
	json.Unmarshal(raw, &header)
	if header.Version != 2 {
		t.Fatal("migration not persisted")
	}
	c.AutoEnabled = false
	if err := SaveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	again, err := LoadConfig(path)
	if err != nil || again.AutoEnabled {
		t.Fatal("explicit pause not retained")
	}
}
