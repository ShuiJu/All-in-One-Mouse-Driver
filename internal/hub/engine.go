package hub

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Event struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}
type State struct {
	Config       Config          `json:"config"`
	Devices      []Device        `json:"devices"`
	Foreground   string          `json:"foreground"`
	Conflicts    map[string]bool `json:"conflicts"`
	Events       []Event         `json:"events"`
	Error        string          `json:"error"`
	CaptureUntil int64           `json:"captureUntil"`
}
type Engine struct {
	mu                     sync.Mutex
	io                     sync.Mutex
	path                   string
	state                  State
	hw                     Hardware
	foreground             func() (string, error)
	conflicts              func() map[string]bool
	last                   map[string]Parameters
	batteryAt              map[string]time.Time
	captureBrand           string
	captureProfile         string
	wake                   chan struct{}
	visible                atomic.Bool
	scanRequested          atomic.Bool
	removedPaths           map[string]bool // Specific removal notifications; guarded by mu.
	statusRefreshRequested atomic.Bool
	recoveryPending        atomic.Bool
	recoveryTargets        map[string]bool
	recoveryRequested      map[string]bool
	recoveryLastNano       map[string]int64
	lastScan               time.Time
	cachedDevices          []Device
	cachedError            error
	changed                func()
	steps                  atomic.Uint64
	scans                  atomic.Uint64
	writes                 atomic.Uint64
	queries                atomic.Uint64
}

type Counters struct{ Steps, DeviceScans, Writes, BatteryQueries uint64 }

func (e *Engine) Counters() Counters {
	return Counters{e.steps.Load(), e.scans.Load(), e.writes.Load(), e.queries.Load()}
}

// NeedsRecovery is true while one or more devices need user/device activity
// before another device operation is worthwhile.
func (e *Engine) NeedsRecovery() bool { return e.recoveryPending.Load() }

const recoveryActivityCooldown = 2 * time.Second

func recoveryPathKey(path string) string {
	path = strings.ToLower(strings.TrimSpace(path))
	for _, marker := range []string{"&mi_", "#mi_", "&col", "#col"} {
		for {
			i := strings.Index(path, marker)
			if i < 0 || len(path) < i+len(marker)+2 {
				break
			}
			path = path[:i] + path[i+len(marker)+2:]
		}
	}
	return strings.TrimSuffix(path, `\kbd`)
}

func recoveryPathMatches(devicePath, inputPath string) bool {
	if devicePath == "" || inputPath == "" {
		return false
	}
	return recoveryPathKey(devicePath) == recoveryPathKey(inputPath)
}

// NotifyMouseActivity is called only while recovery input is registered. It
// does no HID work; it queues one retry for the matching physical device.
func (e *Engine) NotifyMouseActivity(path string) bool {
	if path == "" || !e.recoveryPending.Load() {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var id string
	for targetID := range e.recoveryTargets {
		for _, d := range e.state.Devices {
			if d.ID == targetID && recoveryPathMatches(d.path, path) {
				id = targetID
				break
			}
		}
		if id != "" {
			break
		}
	}
	if id == "" {
		if len(e.recoveryTargets) > 0 {
			return false
		}
		now := time.Now().UnixNano()
		if last := e.recoveryLastNano[""]; last != 0 && now-last < recoveryActivityCooldown.Nanoseconds() {
			return false
		}
		e.recoveryLastNano[""] = now
		e.scanRequested.Store(true)
		e.Wake()
		return true
	}
	now := time.Now().UnixNano()
	if last := e.recoveryLastNano[id]; last != 0 && now-last < recoveryActivityCooldown.Nanoseconds() {
		return false
	}
	if e.recoveryRequested == nil {
		e.recoveryRequested = map[string]bool{}
	}
	if e.recoveryRequested[id] {
		return false
	}
	e.recoveryLastNano[id] = now
	e.recoveryRequested[id] = true
	delete(e.last, id)
	delete(e.batteryAt, id)
	e.cachedDevices = append([]Device(nil), e.state.Devices...)
	e.cachedError = nil
	e.Wake()
	return true
}

// SetNotify is set before Run starts. It must only post a message, not read state.
func (e *Engine) SetNotify(fn func()) { e.changed = fn }
func (e *Engine) SetVisible(v bool) {
	e.visible.Store(v)
	if v {
		e.Wake()
	}
}
func (e *Engine) DevicesChanged() {
	e.scanRequested.Store(true)
	e.Wake()
}

// DeviceRemoved retains the removed interface so a coalesced remove/arrival
// can verify just that receiver instead of invalidating every mouse's cache.
func (e *Engine) DeviceRemoved(path string) {
	if path != "" {
		e.mu.Lock()
		if e.removedPaths == nil {
			e.removedPaths = map[string]bool{}
		}
		e.removedPaths[path] = true
		e.mu.Unlock()
	}
	e.DevicesChanged()
}

// RefreshDevices updates observed device state without treating it as a reconnect.
func (e *Engine) RefreshDevices() {
	e.statusRefreshRequested.Store(true)
	e.Wake()
}

func NewEngine(path string, hw Hardware, fg func() (string, error), conflicts func() map[string]bool) (*Engine, error) {
	c, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return &Engine{
		path: path, hw: hw, foreground: fg, conflicts: conflicts,
		last: map[string]Parameters{}, batteryAt: map[string]time.Time{},
		recoveryTargets: map[string]bool{}, recoveryRequested: map[string]bool{}, recoveryLastNano: map[string]int64{},
		wake:  make(chan struct{}, 1),
		state: State{Config: c, Devices: []Device{}, Events: []Event{}, Conflicts: map[string]bool{}},
	}, nil
}
func (e *Engine) Snapshot() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.state
	s.Config = Clone(s.Config)
	s.Devices = append([]Device{}, s.Devices...)
	s.Events = append([]Event{}, s.Events...)
	s.Conflicts = map[string]bool{}
	for k, v := range e.state.Conflicts {
		s.Conflicts[k] = v
	}
	return s
}
func (e *Engine) event(s string) {
	e.state.Events = append([]Event{{Time: time.Now().Format("15:04:05"), Message: s}}, e.state.Events...)
	if len(e.state.Events) > 80 {
		e.state.Events = e.state.Events[:80]
	}
}
func (e *Engine) Wake() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}
func (e *Engine) Update(c Config) error {
	c = Clone(c)
	if err := c.Validate(); err != nil {
		return err
	}
	e.io.Lock()
	defer e.io.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := SaveConfig(e.path, c); err != nil {
		return err
	}
	e.state.Config = c
	e.event("配置已保存")
	e.Wake()
	return nil
}
func (e *Engine) Capture(profileID string) error {
	if _, ok := e.Snapshot().Config.Find(profileID); !ok {
		return fmt.Errorf("配置不存在")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.captureBrand = "program"
	e.captureProfile = profileID
	e.state.CaptureUntil = time.Now().Add(5 * time.Second).UnixMilli()
	e.Wake()
	return nil
}
func (e *Engine) CancelCapture() {
	e.mu.Lock()
	e.captureBrand = ""
	e.state.CaptureUntil = 0
	e.mu.Unlock()
}
func (e *Engine) Run(ctx context.Context) {
	var capture <-chan time.Time
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		e.step(false)
		if e.changed != nil {
			e.changed()
		}
		e.mu.Lock()
		delay := time.Duration(0)
		if e.state.CaptureUntil > 0 {
			delay = time.Until(time.UnixMilli(e.state.CaptureUntil))
			if delay < time.Millisecond {
				delay = time.Millisecond
			}
		}
		e.mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		capture = nil
		if delay > 0 {
			timer = time.NewTimer(delay)
			capture = timer.C
		}
		select {
		case <-ctx.Done():
			return
		case <-e.wake:
		case <-capture:
		}
	}
}
func (e *Engine) Tick() {
	e.step(true)
}
func (e *Engine) step(force bool) {
	e.io.Lock()
	defer e.io.Unlock()
	e.steps.Add(1)
	proc, fgErr := e.foreground()
	conflict := e.conflicts()
	requested := e.scanRequested.Swap(false)
	statusRefresh := e.statusRefreshRequested.Swap(false)
	scanNow := force || e.lastScan.IsZero() || requested || statusRefresh
	refreshBattery := scanNow
	if scanNow {
		e.cachedDevices, e.cachedError = e.hw.Discover()
		e.lastScan = time.Now()
		e.scans.Add(1)
	}
	devices, discoverErr := append([]Device(nil), e.cachedDevices...), e.cachedError
	e.mu.Lock()
	defer e.mu.Unlock()
	removedPaths := e.removedPaths
	e.removedPaths = nil
	recoveryActivity := map[string]bool{}
	for id := range e.recoveryRequested {
		recoveryActivity[id] = true
	}
	e.recoveryRequested = map[string]bool{}
	refreshBattery = refreshBattery || len(recoveryActivity) > 0
	activityOnly := len(recoveryActivity) > 0 && !scanNow
	e.state.Foreground = proc
	e.state.Conflicts = conflict
	e.state.Error = ""
	if discoverErr != nil {
		e.state.Error = "离线"
	}
	if e.captureBrand != "" && time.Now().UnixMilli() >= e.state.CaptureUntil {
		if fgErr != nil {
			e.event("捕获失败：无法读取前台程序")
		} else {
			next := Clone(e.state.Config)
			_ = next.Bind(proc, e.captureProfile)
			if err := next.Validate(); err != nil {
				e.event(err.Error())
			} else if err := SaveConfig(e.path, next); err != nil {
				e.event("保存失败：" + err.Error())
			} else {
				e.state.Config = next
				e.event("已添加关联程序：" + proc)
			}
		}
		e.captureBrand = ""
		e.state.CaptureUntil = 0
	}

	previous := map[string]Device{}
	previousDevices := append([]Device(nil), e.state.Devices...)
	for _, d := range previousDevices {
		previous[d.ID] = d
	}
	if discoverErr != nil {
		seen := map[string]bool{}
		for _, d := range devices {
			seen[d.ID] = true
		}
		for _, old := range previousDevices {
			if !seen[old.ID] {
				if old.IdentityStatus == IdentityCurrent || old.IdentityStatus == IdentityUnknown || old.IdentityStatus == IdentityCached {
					old.IdentityStatus = IdentityCached
				}
				devices = append(devices, old)
			}
		}
	}
	nextTargets := map[string]bool{}
	for id := range e.recoveryTargets {
		nextTargets[id] = true
	}
	if discoverErr != nil {
		for _, d := range devices {
			nextTargets[d.ID] = true
		}
	}

	present := map[string]bool{}
	for i := range devices {
		d := &devices[i]
		present[d.ID] = true
		old, hadOld := previous[d.ID]
		if mergeMouseIdentity(d, old) {
			// A different mouse on the same receiver needs independent readback.
			delete(e.last, d.ID)
			delete(e.batteryAt, d.ID)
			old.Battery = nil
			old.Charging = false
			old.Applied = nil
			d.Applied = nil
		}
		p, kind := e.state.Config.Resolve(*d, proc)
		d.Profile = kind
		d.Status = "自动切换已暂停"
		if !e.state.Config.AutoEnabled {
			delete(nextTargets, d.ID)
		}
		blocked := activityOnly && !recoveryActivity[d.ID]
		blocked = blocked || (e.recoveryTargets[d.ID] && !recoveryActivity[d.ID] && !scanNow)
		if _, applied := e.last[d.ID]; applied && !blocked && !conflict[d.Brand] && discoverErr == nil {
			for path := range removedPaths {
				if !recoveryPathMatches(d.path, path) {
					continue
				}
				// A real same-path reconnect may reset firmware. When it kept
				// the requested values, even this notification needs no Apply.
				if reader, ok := e.hw.(interface {
					ReadSettings(Device) (Parameters, error)
				}); ok {
					actual, err := reader.ReadSettings(*d)
					if err != nil {
						delete(e.last, d.ID)
						nextTargets[d.ID] = true
						blocked = true
						d.Applied = nil
						old.Applied = nil
					} else if sameParameters(d.Brand, actual, p) {
						e.last[d.ID] = p
					} else {
						delete(e.last, d.ID)
					}
				} else {
					delete(e.last, d.ID)
				}
				break
			}
		}
		if conflict[d.Brand] {
			d.Status = "其他配置工具正在运行，已暂停设备通信"
			delete(e.last, d.ID)
			delete(e.batteryAt, d.ID)
			delete(nextTargets, d.ID)
		} else if discoverErr != nil {
			d.Status = "离线"
			if hadOld {
				d.Battery = old.Battery
				d.Charging = old.Charging
				d.Applied = old.Applied
			}
		} else if d.IdentityStatus == IdentityDisconnected {
			d.Status = "离线"
			if e.state.Config.AutoEnabled {
				nextTargets[d.ID] = true
			}
		} else if d.Brand == "endgame" && d.IdentityStatus != IdentityCurrent && !recoveryActivity[d.ID] {
			// OP1w is considered present only after its settings can be read.
			// Do not send an apply command to an unavailable control interface.
			d.Status = "离线"
			if e.state.Config.AutoEnabled {
				nextTargets[d.ID] = true
			}
		} else if blocked {
			d.Battery = old.Battery
			d.Charging = old.Charging
			d.Applied = old.Applied
			if old.Applied != nil {
				d.Status = "已应用"
			} else {
				d.Status = "离线"
			}
		} else {
			d.Battery = old.Battery
			d.Charging = old.Charging
			d.Applied = old.Applied
			batteryInterval := 5 * time.Minute
			if e.visible.Load() {
				batteryInterval = time.Minute
			}
			if refreshBattery && (statusRefresh || time.Since(e.batteryAt[d.ID]) >= batteryInterval) {
				d.Battery, d.Charging = e.hw.Battery(*d)
				e.queries.Add(1)
				e.batteryAt[d.ID] = time.Now()
			}
			if e.state.Config.AutoEnabled && fgErr == nil {
				last, ok := e.last[d.ID]
				if recoveryActivity[d.ID] || !ok || !sameParameters(d.Brand, last, p) {
					e.writes.Add(1)
					if err := e.hw.Apply(*d, p); err != nil {
						nextTargets[d.ID] = true
						d.Status = "应用失败：" + err.Error()
						delete(e.last, d.ID)
						if d.Status != old.Status {
							e.event(d.Name + " · " + d.Status)
						}
					} else {
						e.last[d.ID] = p
						applied := p
						d.Applied = &applied
						if d.Brand == "endgame" {
							d.IdentityStatus = IdentityCurrent
						}
						delete(nextTargets, d.ID)
						delete(e.recoveryLastNano, d.ID)
						e.event(fmt.Sprintf("%s → %s · %d Hz", d.Name, kind, p.Poll))
					}
				} else {
					d.Status = "已应用"
				}
			} else if e.state.Config.AutoEnabled {
				d.Status = "等待可识别的前台程序"
			}
			if statusRefresh {
				if reader, ok := e.hw.(interface {
					ReadSettings(Device) (Parameters, error)
				}); ok {
					actual, err := reader.ReadSettings(*d)
					if err != nil {
						d.Status = "离线"
						d.Applied = nil
						if d.Brand == "endgame" {
							nextTargets[d.ID] = true
							delete(e.last, d.ID)
						}
						if d.MouseModel != "" {
							d.IdentityStatus = IdentityCached
						} else {
							d.IdentityStatus = IdentityUnavailable
						}
					} else {
						d.Applied = &actual
					}
				}
			}
		}
		if d.IdentityStatus == IdentityUnavailable || d.IdentityStatus == IdentityCached {
			d.Status = "离线"
		}
	}
	for id := range e.last {
		if !present[id] {
			delete(e.last, id)
			delete(e.batteryAt, id)
			delete(e.recoveryLastNano, id)
		}
	}
	for id := range nextTargets {
		if !present[id] {
			delete(nextTargets, id)
		}
	}
	e.state.Devices = devices
	e.recoveryTargets = nextTargets
	pending := len(nextTargets) > 0
	if !pending && discoverErr != nil && len(devices) == 0 && e.state.Config.AutoEnabled {
		pending = true
	}
	e.recoveryPending.Store(pending)
}
func (e *Engine) ApplyManual(id, kind string) error {
	e.io.Lock()
	defer e.io.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.state.Devices {
		d := &e.state.Devices[i]
		if d.ID != id {
			continue
		}
		if e.conflicts()[d.Brand] {
			if d.Brand == "endgame" {
				return fmt.Errorf("请先退出 OP1w 4K v2 官方配置工具")
			}
			return fmt.Errorf("请先退出旧版 %s AutoSwitch", d.Brand)
		}
		p, ok := e.state.Config.Find(kind)
		if !ok {
			return fmt.Errorf("配置不存在")
		}
		if e.state.Config.AutoEnabled {
			return fmt.Errorf("请先在我的配置中暂停自动切换")
		}
		params := p.ForDevice(*d)
		if err := e.hw.Apply(*d, params); err != nil {
			delete(e.recoveryTargets, id)
			delete(e.recoveryLastNano, id)
			e.recoveryPending.Store(len(e.recoveryTargets) > 0)
			delete(e.last, id)
			d.Applied = nil
			d.Status = "应用失败：" + err.Error()
			return err
		}
		e.writes.Add(1)
		e.last[id] = params
		d.Applied = &params
		d.Status = "手动应用成功"
		d.Profile = kind
		delete(e.recoveryTargets, id)
		delete(e.recoveryRequested, id)
		delete(e.recoveryLastNano, id)
		e.recoveryPending.Store(len(e.recoveryTargets) > 0)
		e.event(d.Name + " · 手动应用成功")
		return nil
	}
	return fmt.Errorf("设备已断开，请刷新后重试")
}
