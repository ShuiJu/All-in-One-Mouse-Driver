package endgame

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

// A profile with deliberately non-default CPI/filter bytes catches accidental
// resets of settings outside the six controls that Mouse Hub owns.
func fixtureProfile() []byte {
	p := make([]byte, 0x400)
	for i := range p {
		p[i] = byte(i*17 + 11)
	}
	p[3], p[4], p[5], p[6], p[9], p[12], p[111] = 2, 1, 0x80, 0x23, 3, 0, 0
	return p
}

func TestOfficialPresetReportsPreserveUnrelatedSettings(t *testing.T) {
	raw := fixtureProfile()
	baseline := append([]byte(nil), raw...)
	want := Settings{Poll: 4000, LOD: 17, ForceMaxFPS: true, DeepSleep: 3}
	sensor, perf, err := buildReports(raw, want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, baseline) {
		t.Fatal("mutated source profile")
	}
	if !bytes.Equal(sensor[:4], []byte{0xa1, 0x14, 0x0f, 0x1c}) || len(sensor) != 64 {
		t.Fatalf("sensor header %x", sensor)
	}
	if sensor[0x12] != 10 || sensor[0x10] != raw[7] || sensor[0x11] != raw[8] {
		t.Fatalf("LOD or adjacent sensor fields: %x", sensor[0x10:0x18])
	}
	if !bytes.Equal(sensor[0x18:0x2c], raw[0x23:0x37]) {
		t.Fatal("CPI stages were overwritten")
	}
	if !bytes.Equal(perf[:4], []byte{0xa1, 0x15, 0x0f, 0x0a}) {
		t.Fatalf("performance header %x", perf)
	}
	// 4000 Hz, max FPS bit without clearing other flags, PS disabled, DS 3 min.
	if !bytes.Equal(perf[0x10:0x14], []byte{0, 2, 0x63, 0x81}) || perf[0x19] != 3 || perf[0x1a] != raw[111] {
		t.Fatalf("performance preset %x", perf[0x10:0x1b])
	}
	for i, off := range []int{0x3d, 0x44, 0x4b, 0x52, 0x59} {
		if perf[0x14+i] != raw[off] {
			t.Fatalf("stage filter %x changed", off)
		}
	}
	raw[3], raw[4], raw[5], raw[6] = 3, 0x85, 2, 0x63
	sensor, perf, err = buildReports(raw, Settings{Poll: 1000, LOD: 10, PowerSaving: 1, DeepSleep: 2})
	if err != nil || sensor != nil || !bytes.Equal(perf[0x10:0x14], []byte{0, 0x80, 0x23, 1}) || perf[0x19] != 2 {
		t.Fatalf("daily preset: %x, %v", perf, err)
	}
}

func TestLODGlassModeAndUnsupportedCombinations(t *testing.T) {
	raw := fixtureProfile()
	want := Settings{Poll: 1000, LOD: 20, PowerSaving: 1, DeepSleep: 2}
	if _, _, err := buildReports(raw, want); err == nil {
		t.Fatal("ordinary sensor accepted ambiguous 2.0 mm LOD")
	}
	raw[111], raw[9] = 1, 1
	sensor, perf, err := buildReports(raw, want)
	if err != nil || sensor[0x12] != 2 || perf != nil {
		t.Fatalf("glass LOD write: %x %x %v", sensor, perf, err)
	}
	raw[9] = 2
	got, err := decodeSettings(raw)
	if err != nil || got.LOD != 20 || !got.GlassMode {
		t.Fatalf("glass decode: %+v %v", got, err)
	}
	want.LOD = 7
	if _, _, err := buildReports(raw, want); err == nil {
		t.Fatal("glass sensor accepted 0.7 mm LOD")
	}
	for _, p := range []Settings{
		{Poll: 500, LOD: 10}, {Poll: 8000, LOD: 10}, {Poll: 1000, LOD: 18},
		{Poll: 1000, LOD: 10, PowerSaving: 121}, {Poll: 1000, LOD: 10, DeepSleep: -1},
		{Poll: 4000, LOD: 10, PowerSaving: 1},
		{Poll: 125, LOD: 10, PowerSaving: 1},
	} {
		if p.Validate() == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
}

func TestAllOfferedSettingsRoundTripThroughOfficialFieldLayout(t *testing.T) {
	for _, glass := range []bool{false, true} {
		lods := []int{7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}
		if glass {
			lods = []int{10, 20}
		}
		for _, lod := range lods {
			for _, poll := range []int{125, 1000, 2000, 4000} {
				for _, on := range []bool{false, true} {
					raw := fixtureProfile()
					if glass {
						raw[111], raw[9] = 1, 1
					}
					want := Settings{Poll: poll, LOD: lod, MotionSync: on, ForceMaxFPS: on, DeepSleep: 120, GlassMode: glass}
					if poll == 1000 {
						want.PowerSaving = 120
					}
					sensor, perf, err := buildReports(raw, want)
					if err != nil {
						t.Fatal(err)
					}
					if sensor != nil {
						raw[9] = sensor[0x12]
					}
					if perf != nil {
						for i, off := range []int{12, 5, 6, 4, 61, 68, 75, 82, 89, 3, 111} {
							raw[off] = perf[0x10+i]
						}
					}
					got, err := decodeSettings(raw)
					if err != nil || got != want {
						t.Fatalf("want %+v, got %+v: %v", want, got, err)
					}
				}
			}
		}
	}
}

type scriptedTransport struct {
	replies [][]byte
	sent    [][]byte
	failSet bool
}

func (s *scriptedTransport) Set(b []byte) error {
	if s.failSet {
		return errors.New("disconnected")
	}
	s.sent = append(s.sent, append([]byte(nil), b...))
	return nil
}
func (s *scriptedTransport) Get(byte, int) ([]byte, error) {
	if len(s.replies) == 0 {
		return nil, errors.New("no reply")
	}
	b := s.replies[0]
	s.replies = s.replies[1:]
	return b, nil
}
func response(id byte, n int, status byte) []byte {
	b := make([]byte, n)
	b[0], b[1] = id, status
	return b
}
func profileResponse(p []byte) []byte { b := response(0xa0, 0x411, 1); copy(b[0x10:], p); return b }
func noPause(time.Duration)           {}

func TestApplyRequiresSuccessfulReadbackAndDoesNotRetryWrites(t *testing.T) {
	raw := fixtureProfile()
	want := Settings{Poll: 4000, LOD: 10, ForceMaxFPS: true, DeepSleep: 3}
	s := &scriptedTransport{replies: [][]byte{profileResponse(raw), response(0xa1, 64, 1), profileResponse(raw)}}
	if err := applySettings(s, want, noPause, 500*time.Millisecond); err == nil {
		t.Fatal("ack without changed readback accepted")
	}
	if len(s.sent) != 3 || s.sent[1][1] != 0x15 {
		t.Fatalf("unexpected commands: %x", s.sent)
	}
	s = &scriptedTransport{replies: [][]byte{response(0xa0, 0x411, 3), response(0xa0, 0x411, 3), response(0xa0, 0x411, 3)}}
	if err := applySettings(s, want, noPause, 500*time.Millisecond); err == nil || len(s.sent) != 1 {
		t.Fatal("busy read caused write or success")
	}
	for _, bad := range [][]byte{response(0xa0, 12, 1), response(0xa2, 0x411, 1), response(0xa0, 0x411, 0)} {
		s = &scriptedTransport{replies: [][]byte{bad}}
		if err := applySettings(s, want, noPause, 500*time.Millisecond); err == nil || len(s.sent) != 1 {
			t.Fatal("invalid profile accepted")
		}
	}
	s = &scriptedTransport{failSet: true}
	if _, err := readProfile(s, noPause); err == nil {
		t.Fatal("failed send accepted")
	}
}

func TestObservedA1HeaderForLargeProfileAndDisabledTimers(t *testing.T) {
	raw := fixtureProfile()
	raw[3], raw[4], raw[5] = 0x83, 0x81, 2
	b := profileResponse(raw)
	b[0] = 0xa1 // Observed on the user's receiver for an A0 / 0x411 request.
	s := &scriptedTransport{replies: [][]byte{b}}
	got, err := readProfile(s, noPause)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("large A1 response: %v", err)
	}
	settings, err := decodeSettings(got)
	if err != nil || settings.PowerSaving != 0 || settings.DeepSleep != 0 {
		t.Fatalf("disabled timers: %+v %v", settings, err)
	}
	settings.DeepSleep = 120
	_, perf, err := buildReports(raw, settings)
	if err != nil || perf[0x19] != 120 || perf[0x13] != 0x81 {
		t.Fatalf("timer preservation: %x %v", perf, err)
	}
}

func TestControlInterfaceIdentityAndBattery(t *testing.T) {
	d := DeviceInfo{VID: VendorID, PID: ReceiverPID, UsagePage: 0xff01, Usage: 2, FeatureLen: 0x411}
	if !IsControlDevice(d) {
		t.Fatal("receiver rejected")
	}
	for _, pid := range []uint16{0x1983, 0x1969, 0x1503} {
		b := d
		b.PID = pid
		if IsControlDevice(b) {
			t.Fatalf("unexpected PID %04x", pid)
		}
	}
	d.Usage = 1
	if IsControlDevice(d) {
		t.Fatal("input collection accepted")
	}
	for _, pid := range []uint16{0, 0x1970, 0x1984} {
		b := response(0xa1, 64, 1)
		b[0x12], b[0x13] = byte(pid), byte(pid>>8)
		s := &scriptedTransport{replies: [][]byte{b}}
		if (checkMouse(s, noPause) == nil) != (pid == 0x1984) {
			t.Fatalf("identity PID %04x", pid)
		}
	}
	b := response(0xa1, 64, 1)
	b[0x10] = 82
	if p, _, ok := decodeBattery(b); !ok || p != 82 {
		t.Fatal("battery percent decode")
	}
	b[0x10] = 255
	if _, _, ok := decodeBattery(b); ok {
		t.Fatal("invalid battery accepted")
	}
}
