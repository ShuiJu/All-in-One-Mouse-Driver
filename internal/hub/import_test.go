package hub

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLegacyImportsPreserveSourcesAndBrands(t *testing.T) {
	e, _, _, _ := setup(t)
	dir := t.TempDir()
	l := filepath.Join(dir, "lamzu.conf")
	v := filepath.Join(dir, "vaxee.conf")
	lc := "auto_switch=true\ndefault_mode=office\ndefault_poll=1000\ndefault_sleep_seconds=30\nhit_mode=20000fps\nhit_poll=8000\nhit_motion_sync=on\nhit_sleep_seconds=600\nCS2.EXE\n"
	vc := "auto_switch=true\ndefault_mode=standard_ms_off\ndefault_poll=1000\ndefault_traj=stable_control\nhit_mode=competitive_ms_on\nhit_poll=4000\nhit_traj=smooth_sensitive\nvalorant.exe\n"
	os.WriteFile(l, []byte(lc), 0600)
	os.WriteFile(v, []byte(vc), 0600)
	if err := e.ImportLegacy("lamzu", l); err != nil {
		t.Fatal(err)
	}
	first, _ := e.Snapshot().Config.Resolve(Device{Brand: "lamzu"}, "cs2.exe")
	if first.Poll != 8000 || !first.MotionSync || first.Sleep != 600 {
		t.Fatal(first)
	}
	if err := e.ImportLegacy("vaxee", v); err != nil {
		t.Fatal(err)
	}
	c := e.Snapshot().Config
	p, _ := c.Resolve(Device{Brand: "lamzu"}, "cs2.exe")
	if p != first {
		t.Fatal("VAXEE import changed LAMZU binding")
	}
	p, _ = c.Resolve(Device{Brand: "vaxee"}, "valorant.exe")
	if p.Mode != "competitive" || !p.MotionSync || p.Trajectory != "smooth" {
		t.Fatal(p)
	}
	lb, _ := os.ReadFile(l)
	vb, _ := os.ReadFile(v)
	if string(lb) != lc || string(vb) != vc {
		t.Fatal("source changed")
	}
	out := filepath.Join(dir, "export.json")
	if err := SaveConfig(out, c); err != nil {
		t.Fatal(err)
	}
	round, err := LoadConfig(out)
	if err != nil || !reflect.DeepEqual(round, c) {
		t.Fatalf("round trip: %v", err)
	}
}
