package main

import (
	"syscall"
	"unsafe"
)

func (u *nativeUI) buttonAtWindow(hwnd uintptr) *nativeButton {
	for _, b := range u.buttons {
		if b.hwnd == hwnd {
			return b
		}
	}
	return nil
}
func contains(r rect, p point) bool { return p.X >= r.L && p.X < r.R && p.Y >= r.T && p.Y < r.B }
func (u *nativeUI) programAt(p point) string {
	for _, b := range u.buttons {
		if b.kind == "program" && contains(b.area, p) {
			return b.process
		}
	}
	return ""
}
func (u *nativeUI) cancelDrag() {
	was := u.dragProfile != ""
	u.dragProfile = ""
	u.dropProcess = ""
	u.dragging = false
	if was {
		user32.NewProc("ReleaseCapture").Call()
	}
}
func nativeButtonProc(hwnd, msg, wp, lp uintptr) uintptr {
	u := nativeCurrent
	if u == nil {
		return 0
	}
	old := u.buttonOld[hwnd]
	b := u.buttonAtWindow(hwnd)
	if msg == 0x82 {
		delete(u.buttonOld, hwnd)
	}
	// Suppress the default light erase that flashes before owner-draw paints.
	if msg == 0x0014 {
		return 1
	}
	if b != nil && !u.busy {
		p := point{b.area.L + int32(int16(lp&0xffff)), b.area.T + int32(int16((lp>>16)&0xffff))}
		switch msg {
		case 0x20:
			cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32649)
			user32.NewProc("SetCursor").Call(cursor)
			return 1
		case 0x201:
			if b.kind == "profile" {
				u.dragProfile = b.profileID
				u.dragStart = p
				u.dragging = false
				user32.NewProc("SetFocus").Call(hwnd)
				user32.NewProc("SetCapture").Call(hwnd)
				return 0
			}
		case 0x200:
			if u.dragProfile != "" && wp&1 != 0 {
				dx, dy := p.X-u.dragStart.X, p.Y-u.dragStart.Y
				if dx*dx+dy*dy > 36 {
					u.dragging = true
				}
				if u.dragging {
					user32.NewProc("SetCapture").Call(hwnd)
					target := u.programAt(p)
					if target != u.dropProcess {
						u.dropProcess = target
						for _, item := range u.buttons {
							if item.kind == "program" {
								user32.NewProc("InvalidateRect").Call(item.hwnd, 0, 0)
							}
						}
					}
					return 0
				}
			}
		case 0x202:
			if u.dragging {
				source, target := u.dragProfile, u.programAt(p)
				u.cancelDrag()
				winSend.Call(hwnd, 0xF3, 0, 0)
				if target != "" {
					u.pendingBinding.Process = target
					u.pendingBinding.ProfileID = source
					winPost.Call(u.hwnd, 0x8005, 0, 0)
				} else {
					u.invalidate()
				}
				return 0
			}
			if b.kind == "profile" {
				u.cancelDrag()
				id, _, _ := user32.NewProc("GetDlgCtrlID").Call(hwnd)
				winPost.Call(u.hwnd, 0x111, id, hwnd)
				return 0
			}
		case 0x215:
			if u.dragging {
				u.dragging = false
				u.dragProfile = ""
				u.dropProcess = ""
				u.invalidate()
			}
		}
	}
	if old != 0 && old != u.buttonCallback {
		r, _, _ := user32.NewProc("CallWindowProcW").Call(old, hwnd, msg, wp, lp)
		return r
	}
	r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, msg, wp, lp)
	return r
}
func (u *nativeUI) localText(dc uintptr, s string, r rect, size, color int, flags uintptr) {
	old, _, _ := gdi.NewProc("SelectObject").Call(dc, u.font(size))
	gdi.NewProc("SetBkMode").Call(dc, 1)
	gdi.NewProc("SetTextColor").Call(dc, rgb(color))
	user32.NewProc("DrawTextW").Call(dc, uintptr(unsafe.Pointer(utf(s))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags|0x800)
	gdi.NewProc("SelectObject").Call(dc, old)
}
func (u *nativeUI) drawCard(d *drawItem, b *nativeButton) {
	r := d.Area
	if b.kind == "device" {
		fill(d.DC, r, 0x101214)
		saved, _, _ := gdi.NewProc("SaveDC").Call(d.DC)
		gdi.NewProc("SetViewportOrgEx").Call(d.DC, uintptr(-b.area.L), uintptr(-b.area.T), 0)
		u.paintHome(d.DC)
		gdi.NewProc("RestoreDC").Call(d.DC, saved)
		return
	}
	bg, border := 0x252d39, 0x252d39
	fill(d.DC, r, u.buttonSurface(b))
	if b.selected {
		bg = 0x293d60
		border = 0x4777ff
	}
	if b.kind == "program" {
		bg = 0x20262e
		border = bg
		if u.dropProcess == b.process {
			border = 0x9bb9ff
			bg = 0x314d6e
		}
	}
	if d.State&16 != 0 {
		border = 0x9bb9ff
	}
	rounded(d.DC, r, int(u.y(24)), bg, border)
	text := r
	text.L += u.x(18)
	text.R -= u.x(18)
	if b.kind == "profile" {
		text.T += u.y(36)
		text.B = text.T + u.y(55)
		u.localText(d.DC, b.label, text, 19, 0xeef3ff, 0x11)
		text.T = r.B - u.y(35)
		text.B = r.B - u.y(12)
		badge := ""
		if b.profileID == u.config.Active {
			badge = "默认配置"
		} else if b.selected {
			badge = "已选中"
		}
		u.localText(d.DC, badge, text, 12, 0xa8c4ff, 1)
	} else {
		text.R -= u.x(22)
		text.T += u.y(10)
		text.B = text.T + u.y(25)
		u.localText(d.DC, b.process, text, 14, 0xe8eef9, 0x20|0x8000)
		text.T += u.y(30)
		text.B = text.T + u.y(22)
		name := "未绑定"
		if p, ok := u.config.Find(b.profileID); ok {
			name = p.Name
		}
		u.localText(d.DC, name, text, 12, 0xa8c4ff, 0x20|0x8000)
	}
}

var _ = syscall.Errno(0)
