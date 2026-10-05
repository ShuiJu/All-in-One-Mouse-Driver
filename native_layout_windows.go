package main

import (
	"fmt"
	"mousehub/internal/hub"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func (u *nativeUI) profile() *hub.Profile {
	for i := range u.config.Profiles {
		if u.config.Profiles[i].ID == u.editingProfile {
			return &u.config.Profiles[i]
		}
	}
	u.editingProfile = u.config.Active
	for i := range u.config.Profiles {
		if u.config.Profiles[i].ID == u.editingProfile {
			return &u.config.Profiles[i]
		}
	}
	return &u.config.Profiles[0]
}
func (u *nativeUI) selectedDevice() *hub.Device {
	for i := range u.state.Devices {
		if u.state.Devices[i].ID == u.selected {
			return &u.state.Devices[i]
		}
	}
	return nil
}

const devicesPerPage = 4

func devicePageBounds(count, page int) (start, end, maxPage int) {
	if count <= 0 {
		return 0, 0, 0
	}
	maxPage = (count - 1) / devicesPerPage
	if page < 0 {
		page = 0
	}
	if page > maxPage {
		page = maxPage
	}
	start = page * devicesPerPage
	end = start + devicesPerPage
	if end > count {
		end = count
	}
	return start, end, maxPage
}
func (u *nativeUI) parameters() *hub.Parameters {
	d := u.selectedDevice()
	if d == nil {
		return u.profile().EditDevice(hub.Device{Brand: u.brand})
	}
	return u.profile().EditDevice(*d)
}
func (u *nativeUI) displayName(d hub.Device) string {
	if name := u.config.DeviceNames[d.ID]; name != "" {
		return name
	}
	count := 0
	for _, other := range u.state.Devices {
		if other.Name == d.Name {
			count++
		}
	}
	if count > 1 {
		id := d.ID
		if len(id) > 4 {
			id = id[len(id)-4:]
		}
		return d.Name + " · " + strings.ToUpper(id)
	}
	return d.Name
}
func (u *nativeUI) control(class, text string, style uintptr, x, y, w, h, id int) uintptr {
	r := u.box(x, y, w, h)
	// Create hidden: visible owner-draw controls flash the default light face before WM_DRAWITEM.
	style &^= 0x10000000
	hw, _, _ := winCreate.Call(0, uintptr(unsafe.Pointer(utf(class))), uintptr(unsafe.Pointer(utf(text))), style, uintptr(r.L), uintptr(r.T), uintptr(r.R-r.L), uintptr(r.B-r.T), u.hwnd, uintptr(id), u.instance, 0)
	if hw != 0 {
		u.children = append(u.children, hw)
		winSend.Call(hw, 0x30, u.font(14), 1)
	}
	return hw
}
func (u *nativeUI) button(label string, x, y, w, h int, selected, accent, enabled bool, fn func()) *nativeButton {
	u.nextID++
	id := u.nextID
	b := &nativeButton{label: label, selected: selected, accent: accent, enabled: enabled, x: x, y: y, w: w, h: h, area: u.box(x, y, w, h)}
	b.action = func() {
		if err := u.collect(); err != nil {
			u.note = err.Error()
			u.invalidate()
			return
		}
		fn()
	}
	u.buttons[id] = b
	b.hwnd = u.control("BUTTON", label, 0x5001000B, x, y, w, h, id)
	if b.hwnd != 0 {
		// Always subclass: non-interactive buttons still erase to COLOR_BTNFACE otherwise.
		old, _, _ := user32.NewProc("SetWindowLongPtrW").Call(b.hwnd, ^uintptr(3), u.buttonCallback)
		u.buttonOld[b.hwnd] = old
	}
	// Clip the actual child window, so corners reveal the parent's real surface.
	radius := int(u.y(24))
	region, _, _ := gdi.NewProc("CreateRoundRectRgn").Call(0, 0, uintptr(b.area.R-b.area.L+1), uintptr(b.area.B-b.area.T+1), uintptr(radius), uintptr(radius))
	if region != 0 {
		r, _, _ := user32.NewProc("SetWindowRgn").Call(b.hwnd, region, 1)
		if r == 0 {
			gdi.NewProc("DeleteObject").Call(region)
		}
	}
	if !enabled {
		user32.NewProc("EnableWindow").Call(b.hwnd, 0)
	}
	return b
}
func (u *nativeUI) interactive(b *nativeButton, kind string) {
	// Subclass is already installed in button(); re-setting it would make
	// buttonOld point at our own proc and recurse on WM_SHOWWINDOW.
	b.kind = kind
}
func (u *nativeUI) edit(key, value string, x, y, w, h int, multi bool) {
	style := uintptr(0x50810080)
	if multi {
		style |= 0x00201044
	}
	u.nextID++
	u.edits[key] = u.control("EDIT", value, style, x, y, w, h, u.nextID)
}
func (u *nativeUI) combo(items []string, current, x, y, w int, fn func(int)) {
	u.nextID++
	id := u.nextID
	hw := u.control("COMBOBOX", "", 0x50210203, x, y, w, 250, id)
	for _, s := range items {
		winSend.Call(hw, 0x143, 0, uintptr(unsafe.Pointer(utf(s))))
	}
	winSend.Call(hw, 0x14E, uintptr(current), 0)
	u.selects[id] = fn
}
func textOf(h uintptr) string {
	n, _, _ := user32.NewProc("GetWindowTextLengthW").Call(h)
	buf := make([]uint16, n+1)
	user32.NewProc("GetWindowTextW").Call(h, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}
func (u *nativeUI) collect() error {
	if h := u.edits["sleep"]; h != 0 {
		n, err := strconv.Atoi(strings.TrimSpace(textOf(h)))
		if err != nil || n < 0 || n > 65535 {
			return fmt.Errorf("休眠时间须为 0–65535 秒")
		}
		if u.parameters().Sleep != n {
			u.parameters().Sleep = n
			u.dirty = true
		}
	}
	if h := u.edits["name"]; h != 0 {
		n := strings.TrimSpace(textOf(h))
		if n == "" || len([]rune(n)) > 60 {
			return fmt.Errorf("配置名称须为 1–60 个字符")
		}
		if n != u.profile().Name {
			u.profile().Name = n
			u.dirty = true
		}
	}
	if h := u.edits["deviceName"]; h != 0 {
		n := strings.TrimSpace(textOf(h))
		if len([]rune(n)) > 60 {
			return fmt.Errorf("设备名称不能超过 60 个字符")
		}
		if u.config.DeviceNames == nil {
			u.config.DeviceNames = map[string]string{}
		}
		if u.config.DeviceNames[u.selected] != n {
			u.config.DeviceNames[u.selected] = n
			u.dirty = true
		}
	}
	return nil
}
func (u *nativeUI) navigate(view string) { u.cancelDrag(); u.view = view; u.note = ""; u.layout() }
func (u *nativeUI) layout() {
	if u.hwnd == 0 || u.rebuilding {
		return
	}
	u.rebuilding = true
	defer func() { u.rebuilding = false }()
	for _, h := range u.children {
		user32.NewProc("DestroyWindow").Call(h)
	}
	u.children = nil
	u.buttons = map[int]*nativeButton{}
	u.selects = map[int]func(int){}
	u.edits = map[string]uintptr{}
	u.nextID = 100
	if u.selectedDevice() == nil && len(u.state.Devices) > 0 && u.view == "home" {
		u.selected = u.state.Devices[0].ID
		u.brand = u.state.Devices[0].Brand
	}
	if d := u.selectedDevice(); d != nil {
		u.brand = d.Brand
	}
	u.button("⌂  主页", 26, 20, 114, 47, u.view == "home", false, true, func() { u.navigate("home") })
	u.button("◇  我的配置", 148, 20, 230, 47, u.view == "profiles", false, true, func() { u.navigate("profiles") })
	u.button("设置", 1158, 20, 96, 47, u.view == "settings", false, true, func() { u.navigate("settings") })
	devicePage := u.view == "performance" || u.view == "parameters" || u.view == "overview"
	if devicePage {
		names := []string{}
		index := 0
		for i, p := range u.config.Profiles {
			names = append(names, p.Name)
			if p.ID == u.profile().ID {
				index = i
			}
		}
		if u.view != "overview" {
			u.combo(names, index, 942, 125, 306, func(i int) {
				if i >= 0 && i < len(u.config.Profiles) {
					u.editingProfile = u.config.Profiles[i].ID
					u.layout()
				}
			})
		}
		for i, item := range []struct{ id, label string }{{"performance", "性能"}, {"parameters", "参数"}, {"overview", "设备信息"}} {
			v := item.id
			u.button(item.label, 420+i*150, 748, 140, 40, u.view == v, false, true, func() { u.navigate(v) })
		}
	}
	switch u.view {
	case "home":
		b := u.button("打开设备设置", 290, 122, 700, 505, false, false, u.selectedDevice() != nil, func() { u.navigate("performance") })
		u.interactive(b, "device")
		ds := u.state.Devices
		_, _, maxPage := devicePageBounds(len(ds), u.devicePage)
		if u.devicePage < 0 {
			u.devicePage = 0
		}
		if u.devicePage > maxPage {
			u.devicePage = maxPage
		}
		start, end, _ := devicePageBounds(len(ds), u.devicePage)
		selectedVisible := false
		for i := start; i < end; i++ {
			if ds[i].ID == u.selected {
				selectedVisible = true
				break
			}
		}
		if !selectedVisible && start < end {
			u.selected = ds[start].ID
			u.brand = ds[start].Brand
		}
		for i := start; i < end; i++ {
			d := ds[i]
			device := d
			label := u.displayName(d)
			if d.Battery != nil {
				label += fmt.Sprintf("  %d%%", *d.Battery)
			}
			slot := i - start
			b := u.button(label, 260+slot*245, 665, 235, 50, u.selected == d.ID, false, true, func() { u.selected = device.ID; u.brand = device.Brand; u.layout() })
			b.deviceID = d.ID
		}
		u.button("刷新设备", 31, 665, 160, 45, false, false, true, func() { u.engine.DevicesChanged(); u.note = "正在检测…"; u.invalidate() })
		if maxPage > 0 {
			u.button("‹", 200, 665, 42, 45, false, false, u.devicePage > 0, func() { u.devicePage--; u.layout() })
			u.button("›", 1234, 665, 42, 45, false, false, u.devicePage < maxPage, func() { u.devicePage++; u.layout() })
		}
	case "performance":
		modes := []struct{ id, name string }{{"office", "办公模式"}, {"speed", "高速模式"}, {"20000fps", "20,000 FPS"}}
		if u.brand == "vaxee" {
			modes = []struct{ id, name string }{{"standard", "标准模式"}, {"competitive", "竞技模式"}}
		}
		w := 1148 / len(modes)
		for i, m := range modes {
			mode := m.id
			u.button(m.name, 66+i*w, 274, w-12, 82, u.parameters().Mode == mode, false, true, func() {
				u.parameters().Mode = mode
				if u.brand == "vaxee" && mode == "standard" && u.parameters().Poll > 1000 {
					u.parameters().Poll = 1000
				}
				u.dirty = true
				u.layout()
			})
		}
		rates := []int{1000, 2000, 4000, 8000}
		if u.brand == "vaxee" {
			rates = []int{1000, 2000, 4000}
		}
		w = 1148 / len(rates)
		for i, rate := range rates {
			r := rate
			enabled := !(u.brand == "vaxee" && u.parameters().Mode == "standard" && r > 1000)
			u.button(fmt.Sprintf("%d Hz", r), 66+i*w, 478, w-12, 72, u.parameters().Poll == r, false, enabled, func() { u.parameters().Poll = r; u.dirty = true; u.layout() })
		}
		u.saveButtons()
	case "parameters":
		u.button("关闭", 60, 380, 245, 55, !u.parameters().MotionSync, false, true, func() { u.parameters().MotionSync = false; u.dirty = true; u.layout() })
		u.button("开启", 321, 380, 245, 55, u.parameters().MotionSync, false, true, func() { u.parameters().MotionSync = true; u.dirty = true; u.layout() })
		if u.brand == "lamzu" {
			for i, n := range []int{30, 60, 300, 600} {
				v := n
				u.button(fmt.Sprintf("%d 秒", n), 672+i*134, 365, 126, 48, u.parameters().Sleep == n, false, true, func() { u.parameters().Sleep = v; u.dirty = true; u.edits = map[string]uintptr{}; u.layout() })
			}
			u.edit("sleep", strconv.Itoa(u.parameters().Sleep), 675, 475, 300, 45, false)
		} else {
			u.button("顺滑灵敏", 672, 380, 255, 55, u.parameters().Trajectory == "smooth", false, true, func() { u.parameters().Trajectory = "smooth"; u.dirty = true; u.layout() })
			u.button("稳定易控", 943, 380, 255, 55, u.parameters().Trajectory == "stable", false, true, func() { u.parameters().Trajectory = "stable"; u.dirty = true; u.layout() })
		}
		u.saveButtons()
	case "profiles":
		u.layoutProfiles()
	case "settings":
		u.button("导入 LAMZU", 60, 275, 250, 50, false, false, true, func() { u.importLegacy("lamzu") })
		u.button("导入 VAXEE", 327, 275, 250, 50, false, false, true, func() { u.importLegacy("vaxee") })
		u.button("退出应用", 1060, 732, 190, 50, false, false, true, u.quit)
	case "overview":
		u.edit("deviceName", u.config.DeviceNames[u.selected], 245, 350, 700, 45, false)
		u.button("保存名称", 970, 350, 220, 45, false, true, true, u.save)
		u.button("刷新设备信息", 905, 593, 310, 50, false, false, true, func() { u.engine.DevicesChanged(); u.invalidate() })
	}
	u.finishLayout()
}

// Paint the dark parent first, then reveal already-customized children.
// Avoid RDW_ERASE on children — it paints the default light button face.
func (u *nativeUI) finishLayout() {
	user32.NewProc("RedrawWindow").Call(u.hwnd, 0, 0, 0x0081)
	for _, h := range u.children {
		winShow.Call(h, 5)
		if b := u.buttonAtWindow(h); b != nil && !b.enabled {
			user32.NewProc("EnableWindow").Call(h, 0)
		}
	}
	user32.NewProc("RedrawWindow").Call(u.hwnd, 0, 0, 0x0101)
}
func (u *nativeUI) layoutProfiles() {
	label := "自动切换：已开启"
	if !u.config.AutoEnabled {
		label = "自动切换：已暂停"
	}
	u.button(label, 1000, 125, 220, 44, u.config.AutoEnabled, false, true, func() { u.config.AutoEnabled = !u.config.AutoEnabled; u.save() })
	maxPage := (len(u.config.Profiles) - 1) / 5
	if u.profilePage > maxPage {
		u.profilePage = maxPage
	}
	u.button("‹", 800, 125, 45, 40, false, false, u.profilePage > 0, func() { u.profilePage--; u.layout() })
	u.button("›", 850, 125, 45, 40, false, false, u.profilePage < maxPage, func() { u.profilePage++; u.layout() })
	u.button("＋ 新建", 900, 125, 90, 40, false, false, len(u.config.Profiles) < 100, u.newProfile)
	start := u.profilePage * 5
	for i := start; i < len(u.config.Profiles) && i < start+5; i++ {
		p := u.config.Profiles[i]
		b := u.button(p.Name, 54+(i-start)*236, 194, 222, 150, u.profile().ID == p.ID, false, true, func() { u.editingProfile = p.ID; u.layout() })
		b.profileID = p.ID
		u.interactive(b, "profile")
	}
	u.edit("name", u.profile().Name, 54, 371, 365, 42, false)
	u.button("编辑设备参数", 431, 370, 186, 44, false, true, u.selectedDevice() != nil, func() { u.navigate("performance") })
	u.button("设为默认", 628, 370, 160, 44, false, false, u.profile().ID != u.config.Active, func() { u.config.Active = u.profile().ID; u.save() })
	u.button("复制", 800, 370, 100, 44, false, false, len(u.config.Profiles) < 100, u.newProfile)
	u.button("删除", 910, 370, 100, 44, false, false, u.profile().ID != u.config.Active, func() {
		id := u.profile().ID
		if err := u.config.RemoveProfile(id); err != nil {
			u.note = err.Error()
			u.invalidate()
			return
		}
		u.editingProfile = u.config.Active
		u.save()
	})
	u.button("保存", 1024, 370, 188, 44, false, true, true, u.save)
	u.button("＋ 添加程序", 999, 445, 213, 43, false, true, true, u.addProgram)
	u.button("5 秒后捕获并绑定", 666, 445, 211, 43, false, false, true, func() {
		c := hub.Clone(u.config)
		id := u.profile().ID
		u.work(func() error {
			if err := u.engine.Update(c); err != nil {
				return err
			}
			return u.engine.Capture(id)
		}, "请切换到目标程序")
	})
	count := len(u.config.Bindings)
	maxApp := 0
	if count > 0 {
		maxApp = (count - 1) / 8
	}
	if u.programPage > maxApp {
		u.programPage = maxApp
	}
	u.button("‹", 889, 445, 45, 40, false, false, u.programPage > 0, func() { u.programPage--; u.layout() })
	u.button("›", 939, 445, 45, 40, false, false, u.programPage < maxApp, func() { u.programPage++; u.layout() })
	for i := u.programPage * 8; i < count && i < u.programPage*8+8; i++ {
		binding := u.config.Bindings[i]
		j := i - u.programPage*8
		x, y := 54+(j%4)*294, 505+(j/4)*89
		b := u.button(binding.Process, x, y, 280, 77, false, false, true, func() { u.bindProgram(binding.Process, u.profile().ID) })
		b.process = binding.Process
		b.profileID = binding.ProfileID
		u.interactive(b, "program")
		u.button("×", x+247, y+8, 26, 24, false, false, true, func() {
			out := []hub.Binding{}
			for _, item := range u.config.Bindings {
				if item.Process != binding.Process {
					out = append(out, item)
				}
			}
			u.config.Bindings = out
			u.save()
		})
	}
	u.button("导入配置", 851, 739, 175, 45, false, false, true, u.importConfig)
	u.button("导出配置", 1040, 739, 175, 45, false, false, true, u.exportConfig)
	u.finishLayout()
}
func (u *nativeUI) newProfile() {
	p := *u.profile()
	c := hub.Clone(hub.Config{Profiles: []hub.Profile{p}})
	p = c.Profiles[0]
	p.ID = fmt.Sprintf("profile-%d", time.Now().UnixNano())
	p.Name = "新配置"
	u.config.Profiles = append(u.config.Profiles, p)
	u.editingProfile = p.ID
	u.profilePage = (len(u.config.Profiles) - 1) / 5
	u.save()
}
func (u *nativeUI) bindProgram(process, id string) {
	if err := u.config.Bind(process, id); err != nil {
		u.note = err.Error()
		u.invalidate()
		return
	}
	u.save()
}
func (u *nativeUI) addProgram() {
	if err := u.collect(); err != nil {
		u.note = err.Error()
		u.invalidate()
		return
	}
	c := hub.Clone(u.config)
	u.work(func() error {
		path, err := u.pickFile(false, "添加程序", "应用程序\x00*.exe\x00\x00", "exe")
		if err != nil {
			return err
		}
		name := hub.NormalizeProcess(path)
		for _, b := range c.Bindings {
			if b.Process == name {
				return fmt.Errorf("程序已添加")
			}
		}
		if err = c.Bind(name, ""); err != nil {
			return err
		}
		return u.engine.Update(c)
	}, "程序已添加")
}
func (u *nativeUI) save() {
	c := hub.Clone(u.config)
	deviceID, profileID := u.selected, u.editingProfile
	applyManual := !c.AutoEnabled && deviceID != ""
	note := "已保存"
	if applyManual {
		note = "已保存并应用到当前设备"
	}
	u.work(func() error {
		if err := u.engine.Update(c); err != nil {
			return err
		}
		if !applyManual {
			return nil
		}
		return u.engine.ApplyManual(deviceID, profileID)
	}, note)
}
func (u *nativeUI) saveButtons() {
	u.button("保存配置", 1035, 684, 214, 48, false, true, true, u.save)
}
func (u *nativeUI) requireClean() bool {
	if u.dirty {
		u.note = "请先保存更改"
		u.invalidate()
		return false
	}
	return true
}
func (u *nativeUI) importLegacy(brand string) {
	if !u.requireClean() {
		return
	}
	u.work(func() error {
		path, err := u.pickFile(false, "导入 "+strings.ToUpper(brand), "配置文件\x00*.conf\x00\x00", "conf")
		if err != nil {
			return err
		}
		return u.engine.ImportLegacy(brand, path)
	}, "配置已导入")
}
func (u *nativeUI) importConfig() {
	if !u.requireClean() {
		return
	}
	u.work(func() error {
		path, err := u.pickFile(false, "导入配置", "配置文件\x00*.json\x00\x00", "json")
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() > 1024*1024 {
			return fmt.Errorf("配置文件超过 1 MB")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		c, err := hub.DecodeConfig(data)
		if err != nil {
			return err
		}
		return u.engine.Update(c)
	}, "配置已导入")
}
func (u *nativeUI) exportConfig() {
	if !u.requireClean() {
		return
	}
	c := hub.Clone(u.config)
	u.work(func() error {
		path, err := u.pickFile(true, "导出配置", "配置文件\x00*.json\x00\x00", "json")
		if err != nil {
			return err
		}
		return hub.SaveConfig(path, c)
	}, "配置已导出")
}
