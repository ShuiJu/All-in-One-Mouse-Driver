package main

import (
	"fmt"

	"syscall"
	"unicode/utf16"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")
var shell32 = syscall.NewLazyDLL("shell32.dll")
var kernel32 = syscall.NewLazyDLL("kernel32.dll")

type rawInputDevice struct {
	UsagePage uint16
	Usage     uint16
	Flags     uint32
	Target    uintptr
}

const (
	wmInput        = 0x00ff
	rawInputSink   = 0x00000100
	rawInputRemove = 0x00000001
	rawMousePage   = 0x01
	rawMouseUsage  = 0x02
)

// setRecoveryMouseInput subscribes to mouse activity only while the engine
// has a pending device recovery. It does not send any HID command.
func setRecoveryMouseInput(hwnd uintptr, enabled bool) error {
	device := rawInputDevice{UsagePage: rawMousePage, Usage: rawMouseUsage}
	if enabled {
		device.Flags = rawInputSink
		device.Target = hwnd
	} else {
		device.Flags = rawInputRemove
	}
	r, _, err := user32.NewProc("RegisterRawInputDevices").Call(
		uintptr(unsafe.Pointer(&device)),
		1,
		unsafe.Sizeof(device),
	)
	if r == 0 {
		return err
	}
	return nil
}

type rawInputHeader struct {
	Type   uint32
	Size   uint32
	Device uintptr
	WParam uintptr
}

const (
	ridInput       = 0x10000003
	ridiDeviceName = 0x20000007
)

func rawInputDevicePath(raw uintptr) string {
	if raw == 0 {
		return ""
	}
	getData := user32.NewProc("GetRawInputData")
	headerSize := uint32(unsafe.Sizeof(rawInputHeader{}))
	size := uint32(0)
	r, _, _ := getData.Call(raw, ridInput, 0, uintptr(unsafe.Pointer(&size)), uintptr(headerSize))
	if r == ^uintptr(0) || size < headerSize {
		return ""
	}
	data := make([]byte, size)
	r, _, _ = getData.Call(raw, ridInput, uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)), uintptr(headerSize))
	if r == ^uintptr(0) {
		return ""
	}
	header := (*rawInputHeader)(unsafe.Pointer(&data[0]))
	if header.Device == 0 {
		return ""
	}
	getInfo := user32.NewProc("GetRawInputDeviceInfoW")
	nameSize := uint32(0)
	r, _, _ = getInfo.Call(header.Device, ridiDeviceName, 0, uintptr(unsafe.Pointer(&nameSize)))
	if r == ^uintptr(0) || nameSize == 0 {
		return ""
	}
	name := make([]uint16, nameSize+1)
	r, _, _ = getInfo.Call(header.Device, ridiDeviceName, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameSize)))
	if r == ^uintptr(0) {
		return ""
	}
	return syscall.UTF16ToString(name)
}

func utf(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func message(s string) {
	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(utf(s))), uintptr(unsafe.Pointer(utf("Mouse Hub"))), 0x10)
}

var instanceHandle uintptr

func singleInstance() bool {
	mutex := `Local\MouseHub.SingleInstance`
	if qaMode {
		mutex += ".QA"
	}
	// CreateMutexW does not clear last-error on first creation; stale ERROR_ALREADY_EXISTS would falsely claim a peer.
	kernel32.NewProc("SetLastError").Call(0)
	h, _, err := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(utf(mutex))))
	if h == 0 {
		message("无法创建应用实例：" + err.Error())
		return false
	}
	instanceHandle = h
	if err == syscall.Errno(183) {
		hwnd, _, _ := user32.NewProc("FindWindowW").Call(uintptr(unsafe.Pointer(utf(windowClassName()))), 0)
		if hwnd != 0 {
			user32.NewProc("PostMessageW").Call(hwnd, 0x8004, 0, 0)
			syscall.CloseHandle(syscall.Handle(h))
			return false
		}
		// Mutex exists but no window: treat as abandoned peer and continue as the live instance.
	}
	return true
}

type notifyIcon struct {
	Size        uint32
	Window      uintptr
	ID          uint32
	Flags       uint32
	Callback    uint32
	Icon        uintptr
	Tip         [128]uint16
	State       uint32
	StateMask   uint32
	Info        [256]uint16
	Version     uint32
	InfoTitle   [64]uint16
	InfoFlags   uint32
	GUID        syscall.GUID
	BalloonIcon uintptr
}
type openFileName struct {
	Size            uint32
	Owner           uintptr
	Instance        uintptr
	Filter          *uint16
	CustomFilter    *uint16
	MaxCustomFilter uint32
	FilterIndex     uint32
	File            *uint16
	MaxFile         uint32
	FileTitle       *uint16
	MaxFileTitle    uint32
	InitialDir      *uint16
	Title           *uint16
	Flags           uint32
	FileOffset      uint16
	FileExtension   uint16
	DefaultExt      *uint16
	CustData        uintptr
	Hook            uintptr
	Template        *uint16
	Reserved        uintptr
	ReservedDWORD   uint32
	FlagsEx         uint32
}

func chooseFile(owner uintptr, save bool, title, filter, ext string) (string, error) {
	buf := make([]uint16, 32768)
	filter16 := utf16.Encode([]rune(filter))
	ofn := openFileName{Owner: owner, Filter: &filter16[0], File: &buf[0], MaxFile: uint32(len(buf)), Title: utf(title), DefaultExt: utf(ext), Flags: 0x80000 | 0x8 | 0x800 | 0x1000}
	ofn.Size = uint32(unsafe.Sizeof(ofn))
	name := "GetOpenFileNameW"
	if save {
		name = "GetSaveFileNameW"
		ofn.Flags = 0x80000 | 0x8 | 0x800 | 0x2
		copy(buf, utf16.Encode([]rune("MouseHub-config.json")))
	}
	dll := syscall.NewLazyDLL("comdlg32.dll")
	r, _, _ := dll.NewProc(name).Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		code, _, _ := dll.NewProc("CommDlgExtendedError").Call()
		if code != 0 {
			return "", fmt.Errorf("文件选择失败：0x%x", code)
		}
		return "", nil
	}
	return syscall.UTF16ToString(buf), nil
}
