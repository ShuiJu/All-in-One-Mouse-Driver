package hub

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestEventLoopCachesDevicesAndSleeps(t *testing.T) {
	e, _, _, _ := setup(t)
	c := e.Snapshot().Config
	c.IntervalMS = 250
	if err := e.Update(c); err != nil {
		t.Fatal(err)
	}
	select {
	case <-e.wake:
	default:
	}
	events := make(chan struct{}, 10)
	e.SetNotify(func() { events <- struct{}{} })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { e.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	await := func() {
		t.Helper()
		select {
		case <-events:
		case <-time.After(2 * time.Second):
			t.Fatal("engine event timeout")
		}
	}
	await()
	first := e.Counters()
	if first.DeviceScans != 1 || first.Writes != 3 {
		t.Fatalf("startup counters %+v", first)
	}
	select {
	case <-events:
		t.Fatal("unexpected idle wake")
	case <-time.After(350 * time.Millisecond):
	}
	e.Wake()
	await()
	if e.Counters().DeviceScans != 1 || e.Counters().Writes != 3 {
		t.Fatal("foreground event rescanned or rewrote unchanged devices")
	}
	e.DevicesChanged()
	await()
	if e.Counters().DeviceScans != 2 || e.Counters().Writes != 6 {
		t.Fatal("same-path reconnect failed to reapply")
	}
}

type fakeHardware struct {
	devices     []Device
	writes      []string
	values      []Parameters
	fail        bool
	discoverErr error
	batteries   int
}

func (f *fakeHardware) Discover() ([]Device, error) {
	return append([]Device{}, f.devices...), f.discoverErr
}

func (f *fakeHardware) Apply(d Device, p Parameters) error {
	f.writes = append(f.writes, d.ID)
	f.values = append(f.values, p)
	if f.fail {
		return errors.New("device asleep")
	}
	return nil
}
func (f *fakeHardware) Battery(Device) (*int, bool) { f.batteries++; p := 75; return &p, false }
func setup(t *testing.T) (*Engine, *fakeHardware, *string, *bool) {
	t.Helper()
	fg := "notepad.exe"
	conflict := false
	hw := &fakeHardware{devices: []Device{
		{ID: "l1", Brand: "lamzu", Name: "INCA", path: `\\?\hid#vid_37b0&pid_0010&mi_02#l1`},
		{ID: "v1", Brand: "vaxee", Name: "E1", path: `\\?\hid#vid_25a7&pid_2001&mi_00#v1`},
		{ID: "v2", Brand: "vaxee", Name: "NP-01S", path: `\\?\hid#vid_25a7&pid_2001&mi_00#v2`},
	}}
	e, err := NewEngine(filepath.Join(t.TempDir(), "config.json"), hw, func() (string, error) { return fg, nil }, func() map[string]bool { return map[string]bool{"lamzu": conflict, "vaxee": conflict} })
	if err != nil {
		t.Fatal(err)
	}
	c := e.Snapshot().Config
	_ = c.Bind("cs2.exe", "game")
	if err = e.Update(c); err != nil {
		t.Fatal(err)
	}
	return e, hw, &fg, &conflict
}
func TestAutoSwitchBothBrandsAndNoDuplicateWrites(t *testing.T) {
	e, hw, fg, _ := setup(t)
	e.Tick()
	if len(hw.writes) != 3 {
		t.Fatalf("want 3 writes, got %v", hw.writes)
	}
	e.Tick()
	if len(hw.writes) != 3 {
		t.Fatal("rewrote unchanged state")
	}
	*fg = "CS2.EXE"
	e.Tick()
	if len(hw.writes) != 6 || hw.values[3].Mode != "20000fps" || hw.values[4].Mode != "competitive" {
		t.Fatalf("wrong game switch: %+v", hw.values)
	}
	*fg = "notepad.exe"
	e.Tick()
	if len(hw.writes) != 9 || hw.values[6].Mode != "office" {
		t.Fatal("did not restore daily profile")
	}
}
func TestReconnectRetriesAndPartialFailure(t *testing.T) {
	e, hw, _, _ := setup(t)
	e.Tick()
	original := hw.devices
	hw.devices = nil
	e.Tick()
	hw.devices = original
	e.Tick()
	if len(hw.writes) != 6 {
		t.Fatal("reconnect not applied")
	}
	c := e.Snapshot().Config
	c.Profiles[0].Settings["lamzu"].Poll = 4000
	if err := e.Update(c); err != nil {
		t.Fatal(err)
	}
	hw.fail = true
	e.Tick()
	n := len(hw.writes)
	hw.fail = false
	e.Tick()
	if len(hw.writes) != n+1 {
		t.Fatal("failed settings not retried")
	}
	if e.Snapshot().Devices[0].Applied == nil {
		t.Fatal("successful retry not visible")
	}
}
func TestDiscoveryErrorRetainsDevicesUntilActivity(t *testing.T) {
	e, hw, _, _ := setup(t)
	e.Tick()
	hw.discoverErr = errors.New("mouse sleeping")
	e.DevicesChanged()
	e.Tick()
	s := e.Snapshot()
	if len(s.Devices) != 3 || !e.NeedsRecovery() {
		t.Fatalf("sleeping discovery lost state: devices=%d recovery=%v", len(s.Devices), e.NeedsRecovery())
	}
	for _, d := range s.Devices {
		if d.Status != "设备暂时无响应，移动鼠标唤醒后将自动恢复" {
			t.Fatalf("unexpected sleeping status: %+v", d)
		}
	}
	writes := len(hw.writes)
	hw.discoverErr = nil
	if !e.NotifyMouseActivity(`\\?\hid#vid_37b0&pid_0010&mi_02#l1`) {
		t.Fatal("known recovery activity was not accepted")
	}
	e.step(false)
	if len(hw.writes) != writes+1 || !e.NeedsRecovery() || hw.writes[len(hw.writes)-1] != "l1" {
		t.Fatalf("activity touched the wrong devices: writes=%v recovery=%v", hw.writes, e.NeedsRecovery())
	}
	e.NotifyMouseActivity(`\\?\hid#vid_25a7&pid_2001&mi_00#v1`)
	e.step(false)
	e.NotifyMouseActivity(`\\?\hid#vid_25a7&pid_2001&mi_00#v2`)
	e.step(false)
	if len(hw.writes) != writes+3 || e.NeedsRecovery() {
		t.Fatalf("activity did not recover devices: writes=%d recovery=%v", len(hw.writes), e.NeedsRecovery())
	}
}
func TestRecoveryActivityCoalesces(t *testing.T) {
	e, _, _, _ := setup(t)
	e.Tick()
	e.recoveryTargets["l1"] = true
	e.recoveryPending.Store(true)
	if !e.NotifyMouseActivity(`\\?\hid#vid_37b0&pid_0010&mi_02#l1`) {
		t.Fatal("first recovery activity was rejected")
	}
	if e.NotifyMouseActivity(`\\?\hid#vid_37b0&pid_0010&mi_02#l1`) {
		t.Fatal("recovery activity was not coalesced")
	}
	if !e.NeedsRecovery() || e.scanRequested.Load() || len(e.recoveryRequested) != 1 {
		t.Fatal("recovery activity was not throttled")
	}
}
func TestRecoveryPathMatchesPhysicalDevice(t *testing.T) {
	base := `\\?\hid#vid_25a7&pid_2001#receiver-a`
	if !recoveryPathMatches(base+`&mi_02#collection`, base+`&mi_00#collection&col01\kbd`) {
		t.Fatal("interface paths for one physical device did not match")
	}
	if recoveryPathMatches(base, `\\?\hid#vid_25a7&pid_2001#receiver-b`) {
		t.Fatal("different physical devices matched")
	}
}
func TestUnknownRecoveryScansAfterActivity(t *testing.T) {
	e, hw, _, _ := setup(t)
	known := hw.devices
	hw.devices = nil
	hw.discoverErr = errors.New("mouse sleeping")
	e.Tick()
	if len(e.Snapshot().Devices) != 0 || !e.NeedsRecovery() {
		t.Fatal("startup discovery did not retain an unknown recovery candidate")
	}
	hw.devices = known
	hw.discoverErr = nil
	if !e.NotifyMouseActivity(`\\?\hid#vid_25a7&pid_2001&mi_00#unknown`) {
		t.Fatal("unknown recovery activity was not accepted")
	}
	e.step(false)
	if len(e.Snapshot().Devices) != 3 || len(hw.writes) != 3 || e.NeedsRecovery() {
		t.Fatalf("activity did not recover unknown devices: devices=%d writes=%d recovery=%v", len(e.Snapshot().Devices), len(hw.writes), e.NeedsRecovery())
	}
}
func TestLegacyConflictBlocksWritesAndBatteryRequests(t *testing.T) {
	e, hw, _, conflict := setup(t)
	*conflict = true
	e.Tick()
	if len(hw.writes) != 0 || hw.batteries != 0 {
		t.Fatal("communicated while legacy app active")
	}
	*conflict = false
	e.Tick()
	if len(hw.writes) != 3 {
		t.Fatal("did not resume after legacy exit")
	}
}
func TestManualOnlySelectedDeviceAndPause(t *testing.T) {
	e, hw, _, _ := setup(t)
	e.Tick()
	if e.ApplyManual("v2", "game") == nil {
		t.Fatal("manual allowed with automatic enabled")
	}
	c := e.Snapshot().Config
	c.AutoEnabled = false
	if err := e.Update(c); err != nil {
		t.Fatal(err)
	}
	before := len(hw.writes)
	if err := e.ApplyManual("v2", "game"); err != nil {
		t.Fatal(err)
	}
	if len(hw.writes) != before+1 || hw.writes[before] != "v2" {
		t.Fatal("wrong manual device")
	}
	if e.ApplyManual("missing", "game") == nil {
		t.Fatal("accepted disconnected device")
	}
	if e.ApplyManual("v2", "invalid") == nil {
		t.Fatal("accepted invalid scene")
	}
}
func TestBatteryCaching(t *testing.T) {
	e, hw, _, _ := setup(t)
	e.Tick()
	e.Tick()
	if hw.batteries != 3 {
		t.Fatalf("battery requested too often: %d", hw.batteries)
	}
}
func TestForegroundEventDoesNotPollBattery(t *testing.T) {
	e, hw, _, _ := setup(t)
	e.Tick()
	for id := range e.batteryAt {
		e.batteryAt[id] = time.Now().Add(-10 * time.Minute)
	}
	before := hw.batteries
	e.step(false)
	if hw.batteries != before {
		t.Fatalf("foreground step polled battery: before=%d after=%d", before, hw.batteries)
	}
}
func TestSnapshotAndUpdateIsolation(t *testing.T) {
	e, _, _, _ := setup(t)
	c := e.Snapshot().Config
	c.Bindings[0].Process = "other.exe"
	if e.Snapshot().Config.Bindings[0].Process != "cs2.exe" {
		t.Fatal("snapshot shared storage")
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				e.Snapshot()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		e.Tick()
	}
	wg.Wait()
}
func TestCaptureAndCancel(t *testing.T) {
	e, _, fg, _ := setup(t)
	*fg = "test.exe"
	if err := e.Capture("game"); err != nil {
		t.Fatal(err)
	}
	e.state.CaptureUntil = time.Now().Add(-time.Second).UnixMilli()
	e.Tick()
	p := e.Snapshot().Config.Bindings
	if len(p) != 2 || p[1].Process != "test.exe" {
		t.Fatal(p)
	}
	e.Capture("game")
	e.CancelCapture()
	e.Tick()
	if len(e.Snapshot().Config.Bindings) != 2 {
		t.Fatal("cancelled capture added process")
	}
}
func TestConfigValidationAndAtomicPreservation(t *testing.T) {
	c := DefaultConfig()
	_ = c.Bind(`C:\Games\CS2.EXE`, "game")
	_ = c.Bind("cs2.exe", "game")
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(c.Bindings) != 1 {
		t.Fatal("not deduplicated")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(path, c); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(path)
	c.Profiles[1].Settings["vaxee"].Poll = 8000
	if SaveConfig(path, c) == nil {
		t.Fatal("accepted unsupported polling")
	}
	after, _ := os.ReadFile(path)
	if string(original) != string(after) {
		t.Fatal("invalid config overwrote valid config")
	}
	c = DefaultConfig()
	c.Active = "absent"
	if c.Validate() == nil {
		t.Fatal("missing active profile accepted")
	}
	c = DefaultConfig()
	c.Profiles[1].Settings["lamzu"].Sleep = 65536
	if c.Validate() == nil {
		t.Fatal("overflow accepted")
	}
}
func TestCorruptConfigNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte("corrupt"), 0600)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("corrupt config accepted")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "corrupt" {
		t.Fatal("corrupt source overwritten")
	}
}
