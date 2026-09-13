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
	mu              sync.Mutex
	io              sync.Mutex
	path            string
	state           State
	hw              Hardware
	foreground      func() (string, error)
	conflicts       func() map[string]bool
	last            map[string]Parameters
	batteryAt       map[string]time.Time
	captureBrand    string
	captureProfile  string
	wake            chan struct{}
	visible         atomic.Bool
	scanRequested   atomic.Bool
	topologyChanged atomic.Bool
	lastScan        time.Time
	cachedDevices   []Device
	cachedError     error
	changed         func()
	steps           atomic.Uint64
	scans           atomic.Uint64
	writes          atomic.Uint64
	queries         atomic.Uint64
}

type Counters struct{ Steps, DeviceScans, Writes, BatteryQueries uint64 }

func (e *Engine) Counters() Counters {
	return Counters{e.steps.Load(), e.scans.Load(), e.writes.Load(), e.queries.Load()}
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
	e.topologyChanged.Store(true)
	e.scanRequested.Store(true)
	e.Wake()
}

func NewEngine(path string, hw Hardware, fg func() (string, error), conflicts func() map[string]bool) (*Engine, error) {
	c, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return &Engine{path: path, hw: hw, foreground: fg, conflicts: conflicts, last: map[string]Parameters{}, batteryAt: map[string]time.Time{}, wake: make(chan struct{}, 1), state: State{Config: c, Devices: []Device{}, Events: []Event{}, Conflicts: map[string]bool{}}}, nil
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
	maintenance := time.NewTicker(5 * time.Minute)
	defer maintenance.Stop()
	retryDelay := 5 * time.Second
	var retry <-chan time.Time
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
		failed := false
		for _, d := range e.state.Devices {
			if strings.HasPrefix(d.Status, "应用失败") {
				failed = true
			}
		}
		e.mu.Unlock()
		if failed {
			if delay == 0 || delay > retryDelay {
				delay = retryDelay
			}
			retryDelay *= 2
			if retryDelay > time.Minute {
				retryDelay = time.Minute
			}
		} else {
			retryDelay = 5 * time.Second
		}
		if timer != nil {
			timer.Stop()
		}
		retry = nil
		if delay > 0 {
			timer = time.NewTimer(delay)
			retry = timer.C
		}
		select {
		case <-ctx.Done():
			return
		case <-e.wake:
		case <-maintenance.C:
			e.scanRequested.Store(true)
		case <-retry:
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
	if e.topologyChanged.Swap(false) {
		e.last = map[string]Parameters{}
	}
	proc, fgErr := e.foreground()
	conflict := e.conflicts()
	requested := e.scanRequested.Swap(false)
	if force || e.lastScan.IsZero() || requested {
		e.cachedDevices, e.cachedError = e.hw.Discover()
		e.lastScan = time.Now()
		e.scans.Add(1)
	}
	devices, discoverErr := append([]Device(nil), e.cachedDevices...), e.cachedError
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Foreground = proc
	e.state.Conflicts = conflict
	e.state.Error = ""
	if discoverErr != nil {
		e.state.Error = discoverErr.Error()
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

	present := map[string]bool{}
	previous := map[string]Device{}
	for _, d := range e.state.Devices {
		previous[d.ID] = d
	}
	for i := range devices {
		d := &devices[i]
		present[d.ID] = true
		p, kind := e.state.Config.Resolve(*d, proc)
		d.Profile = kind
		d.Status = "自动切换已暂停"
		if conflict[d.Brand] {
			d.Status = "旧版程序正在运行，已暂停设备通信"
			delete(e.last, d.ID)
			delete(e.batteryAt, d.ID)
		} else {
			old := previous[d.ID]
			d.Battery = old.Battery
			d.Charging = old.Charging
			batteryInterval := 5 * time.Minute
			if e.visible.Load() {
				batteryInterval = time.Minute
			}
			if time.Since(e.batteryAt[d.ID]) >= batteryInterval {
				d.Battery, d.Charging = e.hw.Battery(*d)
				e.queries.Add(1)
				e.batteryAt[d.ID] = time.Now()
			}
			if e.state.Config.AutoEnabled && fgErr == nil {
				last, ok := e.last[d.ID]
				if !ok || last != p {
					e.writes.Add(1)
					if err := e.hw.Apply(*d, p); err != nil {
						d.Status = "应用失败：" + err.Error()
						delete(e.last, d.ID)
						if d.Status != old.Status {
							e.event(d.Name + " · " + d.Status)
						}
					} else {
						e.last[d.ID] = p
						d.Status = "已应用"
						e.event(fmt.Sprintf("%s → %s · %d Hz", d.Name, kind, p.Poll))
					}
				} else {
					d.Status = "已应用"
				}
			} else if e.state.Config.AutoEnabled {
				d.Status = "等待可识别的前台程序"
			}
			if last, ok := e.last[d.ID]; ok {
				v := last
				d.Applied = &v
			}
		}
	}
	for id := range e.last {
		if !present[id] {
			delete(e.last, id)
			delete(e.batteryAt, id)
		}
	}
	e.state.Devices = devices
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
		e.event(d.Name + " · 手动应用成功")
		return nil
	}
	return fmt.Errorf("设备已断开，请刷新后重试")
}
