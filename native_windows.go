package main

import (
	"fmt"
	"mousehub/internal/hub"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var gdi = syscall.NewLazyDLL("gdi32.dll")
var winCreate = user32.NewProc("CreateWindowExW")
var winPost = user32.NewProc("PostMessageW")
var winSend = user32.NewProc("SendMessageW")
var winShow = user32.NewProc("ShowWindow")
var nativeCurrent *nativeUI

const appUpdate = 0x8002
const appLayout = 0x8003

type rect struct{ L, T, R, B int32 }
type point struct{ X, Y int32 }
type winMsg struct {
	H       uintptr
	Message uint32
	W, L    uintptr
	Time    uint32
	Pt      point
	Private uint32
}
type winClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	IconSmall                          uintptr
}
type paintStruct struct {
	DC                 uintptr
	Erase              int32
	Area               rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type drawItem struct {
	Type, ID, ItemID, Action, State uint32
	Window, DC                      uintptr
	Area                            rect
	Data                            uintptr
}
type nativeButton struct {
	kind, profileID, process, deviceID string
	area                               rect
	x, y, w, h                         int
	label                              string
	selected                           bool
	accent                             bool
	enabled                            bool
	action                             func()
	hwnd                               uintptr
}
type nativeUI struct {
	editingProfile                                                         string
	profilePage, programPage, devicePage                                   int
	buttonCallback                                                         uintptr
	buttonOld                                                              map[uintptr]uintptr
	dragProfile, dropProcess                                               string
	dragStart                                                              point
	dragging                                                               bool
	pendingBinding                                                         hub.Binding
	hwnd, instance, callback, hook, hookCallback, deviceNotify             uintptr
	engine                                                                 *hub.Engine
	state                                                                  hub.State
	config                                                                 hub.Config
	data, view, brand, selected, note                                      string
	dirty, busy, visible, rebuilding, recoveryInput, recoveryInputNotified bool
	width, height                                                          int
	fonts                                                                  map[int]uintptr
	buttons                                                                map[int]*nativeButton
	selects                                                                map[int]func(int)
	edits                                                                  map[string]uintptr
	children                                                               []uintptr
	nextID                                                                 int
	tray                                                                   notifyIcon
	taskbarMessage                                                         uint32
	results                                                                chan func()
	windowProfile                                                          string
	closeOnce                                                              sync.Once
}

func newNativeUI(e *hub.Engine, data string) (*nativeUI, error) {
	u := &nativeUI{engine: e, data: data, view: "home", brand: "lamzu", width: 1280, height: 800, fonts: map[int]uintptr{}, buttons: map[int]*nativeButton{}, edits: map[string]uintptr{}, results: make(chan func(), 8)}
	u.state = e.Snapshot()
	u.config = hub.Clone(u.state.Config)
	u.editingProfile = u.config.Active
	u.buttonOld = map[uintptr]uintptr{}
	u.buttonCallback = syscall.NewCallback(nativeButtonProc)
	nativeCurrent = u
	// DPI scaling is handled in our logical 1280x800 layout.
	user32.NewProc("SetProcessDPIAware").Call()
	loadEmbeddedAssets()
	u.instance, _, _ = kernel32.NewProc("GetModuleHandleW").Call(0)
	u.callback = syscall.NewCallback(nativeWndProc)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	bgBrush, _, _ := gdi.NewProc("CreateSolidBrush").Call(rgb(0x101214))
	wc := winClass{Proc: u.callback, Instance: u.instance, Cursor: cursor, Background: bgBrush, Name: utf(windowClassName())}
	wc.Size = uint32(unsafe.Sizeof(wc))
	wc.Icon = logoIcon
	if wc.Icon == 0 {
		wc.Icon, _, _ = user32.NewProc("LoadIconW").Call(0, 32512)
	}
	r, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return nil, fmt.Errorf("register window: %v", err)
	}
	screenW, _, _ := user32.NewProc("GetSystemMetrics").Call(0)
	screenH, _, _ := user32.NewProc("GetSystemMetrics").Call(1)
	width, height := 1280, 850
	if int(screenW) < width+80 {
		width = int(screenW) - 80
	}
	if int(screenH) < height+80 {
		height = int(screenH) - 80
	}
	// Fixed-size window: caption, system menu and minimize only; no sizing frame
	// or maximize box. WS_CLIPCHILDREN protects child controls during paint.
	u.hwnd, _, err = winCreate.Call(0, uintptr(unsafe.Pointer(wc.Name)), uintptr(unsafe.Pointer(utf(windowTitle()))), 0x02CA0000, uintptr((int(screenW)-width)/2), uintptr((int(screenH)-height)/2), uintptr(width), uintptr(height), 0, 0, u.instance, 0)
	if u.hwnd == 0 {
		return nil, fmt.Errorf("create window: %v", err)
	}
	u.updateWindowTitle(u.state)
	if logoIcon != 0 {
		winSend.Call(u.hwnd, 0x0080, 0, logoIcon) // WM_SETICON ICON_SMALL
		winSend.Call(u.hwnd, 0x0080, 1, logoIcon) // WM_SETICON ICON_BIG
	}
	var area rect
	user32.NewProc("GetClientRect").Call(u.hwnd, uintptr(unsafe.Pointer(&area)))
	u.width = int(area.R)
	u.height = int(area.B)
	dark := int32(1)
	syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute").Call(u.hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
	u.tray = notifyIcon{Window: u.hwnd, ID: 1, Flags: 7, Callback: 0x8001, Icon: wc.Icon}
	u.tray.Size = uint32(unsafe.Sizeof(u.tray))
	copy(u.tray.Tip[:], utf16.Encode([]rune("Mouse Hub · 双击打开 / 右键退出")))
	if r, _, _ := shell32.NewProc("Shell_NotifyIconW").Call(0, uintptr(unsafe.Pointer(&u.tray))); r == 0 {
		return nil, fmt.Errorf("无法创建系统托盘图标")
	}
	t, _, _ := user32.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(utf("TaskbarCreated"))))
	u.taskbarMessage = uint32(t)
	u.hookCallback = syscall.NewCallback(func(h uintptr, event uint32, window uintptr, obj, child int32, thread, ms uint32) uintptr {
		e.Wake()
		return 0
	})
	u.hook, _, _ = user32.NewProc("SetWinEventHook").Call(3, 3, 0, u.hookCallback, 0, 0, 0)
	if u.hook == 0 {
		return nil, fmt.Errorf("无法监听前台窗口事件")
	}
	// Register the HID interface class; Windows broadcasts arrivals/removals.
	type devFilter struct {
		Size, Type, Reserved uint32
		Class                syscall.GUID
		Name                 uint16
	}
	filter := devFilter{Type: 5, Class: syscall.GUID{Data1: 0x4d1e55b2, Data2: 0xf16f, Data3: 0x11cf, Data4: [8]byte{0x88, 0xcb, 0, 0x11, 0x11, 0, 0, 0x30}}}
	filter.Size = uint32(unsafe.Sizeof(filter))
	u.deviceNotify, _, _ = user32.NewProc("RegisterDeviceNotificationW").Call(u.hwnd, uintptr(unsafe.Pointer(&filter)), 0)
	u.layout()
	return u, nil
}
func nativeWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	u := nativeCurrent
	if u != nil {
		switch msg {
		case 0x001C: // WM_ACTIVATEAPP
			if wp != 0 {
				u.engine.RefreshDevices()
			}
		case wmInput:
			if u.recoveryInput && !u.recoveryInputNotified {
				u.recoveryInputNotified = u.engine.NotifyMouseActivity(rawInputDevicePath(lp))
			}
			return 0
		case 0x0014:
			return 1
		case 0x0133, 0x0135, 0x0136, 0x0138:
			// EDIT / BUTTON / STATIC / LISTBOX: dark face instead of COLOR_BTNFACE white flash.
			gdi.NewProc("SetTextColor").Call(wp, rgb(0xe7edf8))
			gdi.NewProc("SetBkColor").Call(wp, rgb(0x17191d))
			brush, _, _ := gdi.NewProc("GetStockObject").Call(18)
			gdi.NewProc("SetDCBrushColor").Call(wp, rgb(0x17191d))
			return brush
		case 0x000F:
			u.paint(hwnd)
			return 0
		case 0x0005:
			if wp == 1 {
				u.visible = false
				u.engine.SetVisible(false)
			} else {
				u.width = int(lp & 65535)
				u.height = int((lp >> 16) & 65535)
				if u.hwnd != 0 {
					u.visible = true
					u.engine.SetVisible(true)
					winPost.Call(hwnd, appLayout, 0, 0)
				}
			}
			return 0
		case 0x8004:
			u.show()
			return 0
		case appLayout:
			if u.width > 0 && u.height > 0 {
				if err := u.collect(); err != nil {
					u.note = err.Error()
				}
				u.layout()
			}
			return 0
		case 0x8005:
			binding := u.pendingBinding
			u.pendingBinding = hub.Binding{}
			if binding.Process != "" {
				u.bindProgram(binding.Process, binding.ProfileID)
			}
			return 0
		case 0x0024:
			p := (*[5]point)(unsafe.Pointer(lp))
			p[3] = point{1000, 720}
			return 0
		case 0x0111:
			id := int(wp & 65535)
			if b := u.buttons[id]; b != nil && wp>>16 == 0 && b.enabled && !u.busy {
				b.action()
				return 0
			}
			if f := u.selects[id]; f != nil && wp>>16 == 1 && !u.rebuilding {
				n, _, _ := winSend.Call(lp, 0x147, 0, 0)
				if err := u.collect(); err == nil {
					f(int(n))
				} else {
					u.note = err.Error()
					u.invalidate()
				}
				return 0
			}
		case 0x002B:
			u.drawButton((*drawItem)(unsafe.Pointer(lp)))
			return 1
		case 0x0010:
			u.hide()
			return 0
		case 0x0002:
			user32.NewProc("PostQuitMessage").Call(0)
			return 0
		case 0x0219:
			if wp == 0x8000 || wp == 0x8004 || wp == 7 {
				u.engine.DevicesChanged()
			}
			return 0
		case 0x8001:
			if lp == 0x203 {
				u.show()
			}
			if lp == 0x205 {
				u.trayMenu()
			}
			return 0
		case appUpdate:
			for {
				select {
				case fn := <-u.results:
					fn()
				default:
					goto drained
				}
			}
		drained:
			next := u.engine.Snapshot()
			u.syncRecoveryInput(u.engine.NeedsRecovery())
			u.recoveryInputNotified = false
			u.updateWindowTitle(next)
			if u.visible {
				if err := u.collect(); err != nil {
					u.dirty = true
				}
				// Engine finished a scan/step; drop the transient refresh status.
				if u.note == "正在检测…" {
					u.note = ""
				}
				structural := deviceTopologyChanged(u.state.Devices, next.Devices) ||
					len(u.config.Profiles) != len(next.Config.Profiles) ||
					len(u.config.Bindings) != len(next.Config.Bindings)
				trajectorySupported := u.incaTrajectorySupported()
				u.state = next
				if !u.dirty {
					u.config = hub.Clone(next.Config)
				}
				// Keep capability controls current when a selected device reconnects.
				if (structural && u.view == "home") || (u.view == "parameters" && trajectorySupported != u.incaTrajectorySupported()) {
					u.layout()
				} else {
					if u.view == "home" {
						u.refreshDeviceTabs()
					}
					u.invalidate()
				}
			}
			return 0

		}
		if uint32(msg) == u.taskbarMessage && u.taskbarMessage != 0 {
			shell32.NewProc("Shell_NotifyIconW").Call(0, uintptr(unsafe.Pointer(&u.tray)))
			return 0
		}
	}
	r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, msg, wp, lp)
	return r
}
func (u *nativeUI) run(hidden bool) {
	if hidden {
		u.hide()
	} else {
		u.show()
	}
	var m winMsg
	for {
		r, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if m.Message == 0x100 && m.W == 'A' {
			key, _, _ := user32.NewProc("GetKeyState").Call(0x11)
			if key&0x8000 != 0 {
				focused, _, _ := user32.NewProc("GetFocus").Call()
				handled := false
				for _, h := range u.edits {
					if h == focused {
						winSend.Call(h, 0xB1, 0, ^uintptr(0))
						handled = true
						break
					}
				}
				if handled {
					continue
				}
			}
		}
		handled, _, _ := user32.NewProc("IsDialogMessageW").Call(u.hwnd, uintptr(unsafe.Pointer(&m)))
		if handled == 0 {
			user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
			user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
		}
	}
}
func (u *nativeUI) notify() { winPost.Call(u.hwnd, appUpdate, 0, 0) }
func (u *nativeUI) syncRecoveryInput(enabled bool) {
	if u.recoveryInput == enabled {
		return
	}
	if err := setRecoveryMouseInput(u.hwnd, enabled); err != nil {
		if enabled {
			u.note = "无法监听鼠标恢复活动：" + err.Error()
		}
		return
	}
	u.recoveryInput = enabled
}
func deviceTopologyChanged(old, next []hub.Device) bool {
	if len(old) != len(next) {
		return true
	}
	for i := range old {
		if old[i].ID != next[i].ID || old[i].Name != next[i].Name || old[i].MouseModel != next[i].MouseModel || old[i].IdentityCode != next[i].IdentityCode || old[i].IdentityStatus != next[i].IdentityStatus {
			return true
		}
	}
	return false
}

func (u *nativeUI) updateWindowTitle(state hub.State) {
	id := state.Config.Active
	if state.Config.AutoEnabled {
		device := hub.Device{Brand: u.brand}
		for i := range state.Devices {
			if state.Devices[i].ID == u.selected {
				device = state.Devices[i]
				break
			}
		}
		if device.Profile != "" {
			id = device.Profile
		} else {
			_, id = state.Config.Resolve(device, state.Foreground)
		}
	}
	profile, ok := state.Config.Find(id)
	if !ok {
		profile, ok = state.Config.Find(state.Config.Active)
	}
	if !ok || u.hwnd == 0 || u.windowProfile == profile.Name {
		return
	}
	winSend.Call(u.hwnd, 0x000C, 0, uintptr(unsafe.Pointer(utf(windowTitle()+": "+profile.Name))))
	u.windowProfile = profile.Name
}
func (u *nativeUI) refreshDeviceTabs() {
	for _, b := range u.buttons {
		if b.deviceID == "" {
			continue
		}
		for _, d := range u.state.Devices {
			if d.ID != b.deviceID {
				continue
			}
			label := u.displayName(d)
			if d.Battery != nil {
				label += fmt.Sprintf("  %d%%", *d.Battery)
			}
			if b.label != label {
				b.label = label
				if b.hwnd != 0 {
					winSend.Call(b.hwnd, 0x000C, 0, uintptr(unsafe.Pointer(utf(label))))
				}
			}
			b.selected = u.selected == d.ID
			break
		}
	}
}
func (u *nativeUI) invalidate() {
	if u.visible {
		user32.NewProc("InvalidateRect").Call(u.hwnd, 0, 0)
	}
}
func (u *nativeUI) show() {
	u.visible = true
	u.state = u.engine.Snapshot()
	if !u.dirty {
		u.config = hub.Clone(u.state.Config)
	}
	u.engine.SetVisible(true)
	winShow.Call(u.hwnd, 9)
	user32.NewProc("SetForegroundWindow").Call(u.hwnd)
	u.layout()
}
func (u *nativeUI) hide() {
	if err := u.collect(); err != nil {
		u.note = err.Error()
		u.dirty = true
	}
	u.visible = false
	u.engine.SetVisible(false)
	winShow.Call(u.hwnd, 0)
	// Release actual UI resources, without artificially trimming the working set.
	for _, h := range u.children {
		user32.NewProc("DestroyWindow").Call(h)
	}
	u.children = nil
	u.buttons = map[int]*nativeButton{}
	u.edits = map[string]uintptr{}
	u.selects = map[int]func(int){}
	for _, f := range u.fonts {
		gdi.NewProc("DeleteObject").Call(f)
	}
	u.fonts = map[int]uintptr{}
}
func (u *nativeUI) cleanup() {
	u.closeOnce.Do(func() {
		u.syncRecoveryInput(false)
		if u.hook != 0 {
			user32.NewProc("UnhookWinEvent").Call(u.hook)
		}
		if u.deviceNotify != 0 {
			user32.NewProc("UnregisterDeviceNotification").Call(u.deviceNotify)
		}
		shell32.NewProc("Shell_NotifyIconW").Call(2, uintptr(unsafe.Pointer(&u.tray)))
		for _, f := range u.fonts {
			gdi.NewProc("DeleteObject").Call(f)
		}
		releaseEmbeddedAssets()
		if instanceHandle != 0 {
			syscall.CloseHandle(syscall.Handle(instanceHandle))
		}
	})
}
func (u *nativeUI) trayMenu() {
	m, _, _ := user32.NewProc("CreatePopupMenu").Call()
	defer user32.NewProc("DestroyMenu").Call(m)
	user32.NewProc("AppendMenuW").Call(m, 0, 1, uintptr(unsafe.Pointer(utf("打开 Mouse Hub"))))
	user32.NewProc("AppendMenuW").Call(m, 0, 2, uintptr(unsafe.Pointer(utf("退出"))))
	var p point
	user32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&p)))
	user32.NewProc("SetForegroundWindow").Call(u.hwnd)
	c, _, _ := user32.NewProc("TrackPopupMenu").Call(m, 0x102, uintptr(p.X), uintptr(p.Y), 0, u.hwnd, 0)
	if c == 1 {
		u.show()
	}
	if c == 2 {
		u.quit()
	}
}
func (u *nativeUI) quit() {
	if u.dirty {
		r, _, _ := user32.NewProc("MessageBoxW").Call(u.hwnd, uintptr(unsafe.Pointer(utf("尚有未保存的更改。放弃更改并退出？"))), uintptr(unsafe.Pointer(utf("Mouse Hub"))), 0x24)
		if r != 6 {
			return
		}
	}
	user32.NewProc("DestroyWindow").Call(u.hwnd)
}
func (u *nativeUI) work(fn func() error, success string) {
	if u.busy {
		return
	}
	u.busy = true
	for _, h := range u.children {
		user32.NewProc("EnableWindow").Call(h, 0)
	}
	u.note = "正在处理…"
	u.invalidate()
	go func() {
		err := fn()
		u.results <- func() {
			u.busy = false
			if err != nil {
				u.note = err.Error()
			} else {
				u.note = success
				u.dirty = false
				u.config = u.engine.Snapshot().Config
			}
			u.state = u.engine.Snapshot()
			u.layout()
		}
		u.notify()
	}()
}
