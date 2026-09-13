package platform

import (
	"strings"
	"syscall"
	"unsafe"
)

func HIDSerial(path string) string {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	h, err := syscall.CreateFile(p, 0, 3, nil, 3, 0, 0)
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(h)
	buf := make([]uint16, 256)
	r, _, _ := syscall.NewLazyDLL("hid.dll").NewProc("HidD_GetSerialNumberString").Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2))
	if r == 0 {
		return ""
	}
	s := strings.TrimSpace(syscall.UTF16ToString(buf))
	if strings.Trim(s, "0 -") == "" {
		return ""
	}
	return s
}
