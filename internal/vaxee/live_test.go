package vaxee

import (
	"os"
	"testing"
)

func TestLiveIdentity(t *testing.T) {
	if os.Getenv("MOUSEHUB_PROBE") != "1" {
		t.Skip("hardware probe is opt-in")
	}
	ds, err := EnumerateVaxeeDevices()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range ds {
		t.Logf("product=%s pid=%04x usage=%04x:%04x len=%d path=%s", d.Product, d.PID, d.UsagePage, d.Usage, d.FeatureLen, d.Path)
		if d.UsagePage == 0xff05 {
			identity, err := ReadIdentity(d)
			t.Logf("identity=%+v error=%v", identity, err)
		}
	}
}
