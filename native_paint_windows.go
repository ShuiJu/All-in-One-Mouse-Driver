package main

import (
	"fmt"
	"math"
	"mousehub/internal/hub"
	"strings"
	"unsafe"
)

func rgb(v int) uintptr                     { return uintptr((v&255)<<16 | v&0xff00 | (v>>16)&255) }
func (u *nativeUI) x(v int) int32           { return int32(v * u.width / 1280) }
func (u *nativeUI) y(v int) int32           { return int32(v * u.height / 800) }
func (u *nativeUI) box(x, y, w, h int) rect { return rect{u.x(x), u.y(y), u.x(x + w), u.y(y + h)} }
func (u *nativeUI) titleStatus() string {
	name := "默认配置"
	if p, ok := u.config.Find(u.config.Active); ok {
		name = p.Name
	}
	if !u.config.AutoEnabled {
		return name + " · 自动切换已暂停"
	}
	if d := u.selectedDevice(); d != nil && d.Profile != "" {
		if p, ok := u.config.Find(d.Profile); ok {
			return p.Name + " · " + u.displayName(*d)
		}
	}
	return name
}
func (u *nativeUI) font(size int) uintptr {
	pixels := int(math.Round(float64(size) * math.Min(float64(u.width)/1280, float64(u.height)/800)))
	if pixels < 10 {
		pixels = 10
	}
	if f := u.fonts[pixels]; f != 0 {
		return f
	}
	h, _, _ := gdi.NewProc("CreateFontW").Call(uintptr(int32(-pixels)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf("Microsoft YaHei UI"))))
	u.fonts[pixels] = h
	return h
}
func fill(dc uintptr, r rect, color int) {
	b, _, _ := gdi.NewProc("CreateSolidBrush").Call(rgb(color))
	user32.NewProc("FillRect").Call(dc, uintptr(unsafe.Pointer(&r)), b)
	gdi.NewProc("DeleteObject").Call(b)
}
func rounded(dc uintptr, r rect, radius int, color, border int) {
	b, _, _ := gdi.NewProc("CreateSolidBrush").Call(rgb(color))
	p, _, _ := gdi.NewProc("CreatePen").Call(0, 1, rgb(border))
	oldB, _, _ := gdi.NewProc("SelectObject").Call(dc, b)
	oldP, _, _ := gdi.NewProc("SelectObject").Call(dc, p)
	gdi.NewProc("RoundRect").Call(dc, uintptr(r.L), uintptr(r.T), uintptr(r.R), uintptr(r.B), uintptr(radius), uintptr(radius))
	gdi.NewProc("SelectObject").Call(dc, oldB)
	gdi.NewProc("SelectObject").Call(dc, oldP)
	gdi.NewProc("DeleteObject").Call(b)
	gdi.NewProc("DeleteObject").Call(p)
}
func (u *nativeUI) text(dc uintptr, s string, x, y, w, h, size, color int, flags uintptr) {
	r := u.box(x, y, w, h)
	old, _, _ := gdi.NewProc("SelectObject").Call(dc, u.font(size))
	gdi.NewProc("SetBkMode").Call(dc, 1)
	gdi.NewProc("SetTextColor").Call(dc, rgb(color))
	user32.NewProc("DrawTextW").Call(dc, uintptr(unsafe.Pointer(utf(s))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags|0x800)
	gdi.NewProc("SelectObject").Call(dc, old)
}
func (u *nativeUI) drawButton(d *drawItem) {
	b := u.buttons[int(d.ID)]
	if b == nil {
		return
	}
	if b.kind != "" {
		u.drawCard(d, b)
		return
	}
	bg, fg, border := 0x20242b, 0xd5dce8, 0x20242b
	if b.selected {
		bg = 0x245dff
		fg = 0xffffff
		border = 0x4777ff
	}
	if b.accent {
		bg = 0x9bb9ff
		fg = 0x12203b
		border = bg
	}
	if !b.enabled {
		fg = 0x626974
	}
	if d.State&1 != 0 {
		bg = 0x3c5579
	}
	r := d.Area
	fill(d.DC, r, u.buttonSurface(b))

	if d.State&16 != 0 {
		border = 0x9bb9ff
	}
	rounded(d.DC, r, int(u.y(24)), bg, border)
	old, _, _ := gdi.NewProc("SelectObject").Call(d.DC, u.font(14))
	gdi.NewProc("SetBkMode").Call(d.DC, 1)
	gdi.NewProc("SetTextColor").Call(d.DC, rgb(fg))
	r.L += 6
	r.R -= 6
	user32.NewProc("DrawTextW").Call(d.DC, uintptr(unsafe.Pointer(utf(b.label))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), 0x25|0x800)
	gdi.NewProc("SelectObject").Call(d.DC, old)
}
func (u *nativeUI) buttonSurface(b *nativeButton) int {
	inPanel := false
	switch u.view {
	case "profiles":
		inPanel = b.y >= 118 && b.y+b.h <= 718
	case "performance":
		inPanel = b.y >= 180 && b.y+b.h <= 630
	case "parameters":
		inPanel = b.y >= 180 && b.y+b.h <= 620
	case "settings":
		inPanel = b.y >= 120 && b.y+b.h <= 645
	case "overview":
		inPanel = b.y >= 180 && b.y+b.h <= 670
	}
	if inPanel {
		return 0x171a1e
	}
	return 0x101214
}
func (u *nativeUI) mouse(dc uintptr, x, y, w, h int, purple bool) {
	body := 0xd4dce8
	if purple {
		body = 0xbbb7d3
	}
	rounded(dc, u.box(x, y, w, h), int(u.x(w*8/10)), body, 0xebedf5)
	// Simple device illustration, not a model-specific product photograph.
	line := u.box(x+w/2, y+3, 1, h*43/100)
	fill(dc, line, 0x687486)
	fill(dc, u.box(x+5, y+h*43/100, w-10, 1), 0x8592a4)
	rounded(dc, u.box(x+w*43/100, y+h*12/100, w*14/100, h*17/100), int(u.x(12)), 0x161b24, 0x465064)
	rounded(dc, u.box(x-2, y+h*46/100, 4, h*11/100), 4, 0x3d4655, 0x3d4655)
	u.text(dc, "M", x, y+h*72/100, w, 30, 20, 0xa8b4c7, 1)
}
func (u *nativeUI) paint(hwnd uintptr) {
	var ps paintStruct
	dc, _, _ := user32.NewProc("BeginPaint").Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer user32.NewProc("EndPaint").Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if u.width < 1 || u.height < 1 {
		return
	}
	mem, _, _ := gdi.NewProc("CreateCompatibleDC").Call(dc)
	bitmap, _, _ := gdi.NewProc("CreateCompatibleBitmap").Call(dc, uintptr(u.width), uintptr(u.height))
	old, _, _ := gdi.NewProc("SelectObject").Call(mem, bitmap)
	fill(mem, rect{0, 0, int32(u.width), int32(u.height)}, 0x101214)
	u.text(mem, u.titleStatus(), 320, 22, 640, 25, 16, 0xc9d3e5, 1)

	conflict := []string{}
	for _, b := range []string{"lamzu", "vaxee"} {
		if u.state.Conflicts[b] {
			conflict = append(conflict, strings.ToUpper(b))
		}
	}
	notice := u.note
	if len(conflict) > 0 && notice == "" {
		notice = strings.Join(conflict, " / ") + " 旧版正在运行，设备控制已暂停。请先退出旧版。"
	}
	if notice == "" {
		notice = u.state.Error
	}
	u.text(mem, notice, 32, 80, 1216, 30, 12, 0xe1ba81, 0)
	switch u.view {
	case "home":
		u.paintHome(mem)
	case "performance":
		u.paintPerformance(mem)
	case "parameters":
		u.paintParameters(mem)

	case "profiles":
		u.paintProfiles(mem)
	case "settings":
		u.paintSettings(mem)
	case "overview":
		u.paintOverview(mem)
	}
	if u.dirty {
		u.text(mem, "● 未保存", 36, 718, 180, 25, 12, 0xe1ba81, 0)
	}
	gdi.NewProc("BitBlt").Call(dc, 0, 0, uintptr(u.width), uintptr(u.height), mem, 0, 0, 0x00CC0020)
	gdi.NewProc("SelectObject").Call(mem, old)
	gdi.NewProc("DeleteObject").Call(bitmap)
	gdi.NewProc("DeleteDC").Call(mem)
}
func (u *nativeUI) panel(dc uintptr, x, y, w, h int) {
	rounded(dc, u.box(x, y, w, h), int(u.x(28)), 0x171a1e, 0x1a1e25)
}
func mouseIdentityText(d *hub.Device) string {
	if d == nil || d.IdentityStatus == hub.IdentityCached || d.IdentityStatus == hub.IdentityUnavailable || d.IdentityStatus == hub.IdentityDisconnected {
		return "离线"
	}
	model := d.MouseModel
	if model == "" {
		if d.IdentityCode != "" {
			model = "未识别（" + d.IdentityCode + "）"
		} else {
			model = "未取得型号"
		}
	}
	return model
}

func (u *nativeUI) paintHome(dc uintptr) {
	u.panel(dc, 290, 122, 700, 505)
	d := u.selectedDevice()
	name, sub := "连接鼠标", ""
	if d != nil {
		name = u.displayName(*d)
		sub = "电量未知"
		if d.Battery != nil {
			sub = fmt.Sprintf("电量 %d%%", *d.Battery)
		}
		if d.Charging {
			sub += " · 充电中"
		}
		sub = "鼠标：" + mouseIdentityText(d) + " · " + sub
	}
	u.text(dc, name, 325, 150, 620, 42, 28, 0xe9effa, 0)
	u.text(dc, sub, 325, 203, 600, 32, 13, 0x919eb2, 0)
	if !u.drawMouseImage(dc, mouseAssetKey(d), 500, 230, 280, 340) {
		u.mouse(dc, 555, 250, 165, 310, u.brand == "vaxee")
	}
	u.text(dc, "↗", 917, 568, 45, 34, 24, 0x9bb9ff, 1)
}
func (u *nativeUI) deviceHeading(dc uintptr) {
	name, model := "设备未连接", ""
	if d := u.selectedDevice(); d != nil {
		name = u.displayName(*d)
		model = "鼠标：" + mouseIdentityText(d)
	}
	u.text(dc, name, 34, 118, 790, 32, 22, 0xe9effa, 0)
	u.text(dc, model, 34, 153, 790, 22, 12, 0x919eb2, 0)
}
func (u *nativeUI) paintPerformance(dc uintptr) {
	u.deviceHeading(dc)
	u.panel(dc, 28, 180, 1224, 450)
	u.text(dc, "性能模式", 66, 220, 600, 35, 21, 0xe9effa, 0)
	u.text(dc, "回报率", 66, 420, 600, 35, 21, 0xe9effa, 0)
	if u.brand == "vaxee" && u.parameters().Mode == "standard" {
		u.text(dc, "标准模式最高 1000 Hz", 66, 574, 800, 25, 12, 0x8c9cb3, 0)
	}
}
func (u *nativeUI) paintParameters(dc uintptr) {
	u.deviceHeading(dc)
	u.panel(dc, 28, 180, 590, 440)
	u.panel(dc, 638, 180, 614, 440)
	motionTitleY := 255
	if u.brand == "lamzu" {
		motionTitleY = 220
	}
	u.text(dc, "Motion Sync", 60, motionTitleY, 510, 55, 30, 0xedf2fc, 0)
	title := "休眠时间"
	if u.brand == "vaxee" {
		title = "追踪轨迹"
	}
	u.text(dc, title, 672, 255, 520, 55, 30, 0xedf2fc, 0)
	if u.brand == "lamzu" {
		u.text(dc, "追踪轨迹", 60, 400, 510, 45, 24, 0xedf2fc, 0)
		description := "配置：未指定（保留设备当前设置）"
		if u.parameters().Trajectory == "smooth" {
			description = "配置：顺滑轨迹"
		} else if u.parameters().Trajectory == "stable" {
			description = "配置：稳定轨迹"
		}
		if !u.incaTrajectorySupported() {
			description = "需连接支持轨迹切换的新版固件设备"
		}
		u.text(dc, description, 60, 539, 510, 25, 13, 0x95a4b9, 0)
		applied := "已应用：尚未确认"
		if d := u.selectedDevice(); u.incaTrajectorySupported() && d.Applied != nil {
			if d.Applied.Trajectory == "smooth" {
				applied = "已应用：顺滑轨迹"
			} else if d.Applied.Trajectory == "stable" {
				applied = "已应用：稳定轨迹"
			}
		}
		if d := u.selectedDevice(); d != nil && (d.IdentityStatus == hub.IdentityCached || d.IdentityStatus == hub.IdentityUnavailable || d.IdentityStatus == hub.IdentityDisconnected) {
			applied = "离线"
		}
		u.text(dc, applied, 60, 574, 510, 25, 13, 0x95a4b9, 0)
		u.text(dc, "自定义（秒）", 674, 438, 330, 25, 13, 0x95a4b9, 0)
	}
}
func (u *nativeUI) paintProfiles(dc uintptr) {
	u.panel(dc, 28, 118, 1224, 600)
	u.text(dc, "我的配置", 54, 131, 550, 42, 26, 0xeef4ff, 0)
	u.text(dc, "游戏与应用程序", 54, 451, 250, 35, 21, 0xeef4ff, 0)
	u.text(dc, "将配置拖到程序上绑定", 300, 458, 335, 28, 12, 0x95a4b9, 0)
	if len(u.config.Bindings) == 0 {
		u.text(dc, "尚未添加程序", 54, 544, 1100, 45, 16, 0x7d8da4, 1)
	}
	if u.state.CaptureUntil > 0 {
		u.text(dc, "正在等待捕获前台程序", 54, 742, 700, 30, 13, 0x9bb9ff, 0)
	}
}
func (u *nativeUI) paintSettings(dc uintptr) {
	u.panel(dc, 28, 120, 590, 525)
	u.panel(dc, 638, 120, 614, 525)
	u.text(dc, "导入原有配置", 60, 170, 510, 50, 27, 0xedf2fc, 0)
	u.text(dc, "活动记录", 673, 170, 540, 50, 27, 0xedf2fc, 0)
	for i, e := range u.state.Events {
		if i >= 8 {
			break
		}
		u.text(dc, e.Time+"  "+e.Message, 672, 246+i*42, 545, 36, 12, 0xa4b5ce, 0)
	}
}
func (u *nativeUI) paintOverview(dc uintptr) {
	u.deviceHeading(dc)
	u.panel(dc, 28, 180, 1224, 490)
	d := u.selectedDevice()
	if d == nil {
		return
	}
	u.text(dc, "设备名称", 65, 354, 175, 35, 17, 0xaabbd3, 0)
	u.text(dc, "鼠标型号", 65, 234, 175, 35, 17, 0xaabbd3, 0)
	u.text(dc, mouseIdentityText(d), 245, 234, 950, 35, 22, 0xeef4ff, 0)
	battery := "未知"
	if d.Battery != nil {
		battery = fmt.Sprintf("%d%%", *d.Battery)
	}
	u.text(dc, "电池电量", 65, 290, 175, 35, 17, 0xaabbd3, 0)
	u.text(dc, battery, 245, 290, 950, 35, 18, 0xeef4ff, 0)
	u.text(dc, "设备标识", 65, 430, 175, 35, 17, 0xaabbd3, 0)
	u.text(dc, d.ID, 245, 430, 950, 35, 15, 0xaabbd3, 0)
	u.text(dc, d.Status, 65, 520, 1120, 45, 14, 0x95a4b9, 0)
	u.text(dc, "识别依据", 65, 480, 175, 30, 14, 0xaabbd3, 0)
	u.text(dc, d.IdentitySource, 245, 480, 950, 30, 14, 0x919eb2, 0)
}
