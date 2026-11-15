package endgame

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	VendorID         = 0x3367
	WiredPID         = 0x1984
	ReceiverPID      = 0x1970
	shortReportLen   = 64
	profileReportLen = 0x411
)

// The bootloader PIDs 1983/1969 and mouse/keyboard input collections must never
// receive configuration commands. These are the main EXE's FF01:0002 filters.
func IsControlDevice(d DeviceInfo) bool {
	return d.VID == VendorID && (d.PID == WiredPID || d.PID == ReceiverPID) &&
		d.UsagePage == 0xff01 && d.Usage == 2 && d.FeatureLen == profileReportLen
}

var deviceMu sync.Mutex

type featureTransport interface {
	Set([]byte) error
	Get(byte, int) ([]byte, error)
}

type hidTransport struct{ handle syscall.Handle }

func (h hidTransport) Set(b []byte) error {
	r, _, err := procHidDSetFeature_HID.Call(uintptr(h.handle), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if r == 0 {
		return fmt.Errorf("Endgame Gear 写入失败: %w", err)
	}
	return nil
}

func (h hidTransport) Get(id byte, length int) ([]byte, error) {
	b := make([]byte, length)
	b[0] = id
	r, _, err := procHidDGetFeature_HID.Call(uintptr(h.handle), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if r == 0 {
		return nil, fmt.Errorf("Endgame Gear 读取失败: %w", err)
	}
	return b, nil
}

func openDevice(d DeviceInfo) (hidTransport, error) {
	current, ok := queryDeviceInfo(d.Path)
	if !ok || !IsControlDevice(current) || !IsControlDevice(d) {
		return hidTransport{}, fmt.Errorf("OP1w 4K v2 控制接口不可用")
	}
	h, err := openHIDPath(d.Path)
	return hidTransport{h}, err
}

func ReadSettings(d DeviceInfo) (Settings, error) {
	deviceMu.Lock()
	defer deviceMu.Unlock()
	h, err := openDevice(d)
	if err != nil {
		return Settings{}, err
	}
	defer closeHandle(h.handle)
	if err = checkMouse(h, time.Sleep); err != nil {
		return Settings{}, err
	}
	p, err := readProfile(h, time.Sleep)
	if err != nil {
		return Settings{}, err
	}
	return decodeSettings(p)
}

func ApplySettings(d DeviceInfo, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	deviceMu.Lock()
	defer deviceMu.Unlock()
	h, err := openDevice(d)
	if err != nil {
		return err
	}
	defer closeHandle(h.handle)
	if err = checkMouse(h, time.Sleep); err != nil {
		return err
	}
	delay := 500 * time.Millisecond
	if d.PID == ReceiverPID {
		delay = 880 * time.Millisecond
	}
	return applySettings(h, s, time.Sleep, delay)
}

func ReadBattery(d DeviceInfo) (int, bool, bool) {
	deviceMu.Lock()
	defer deviceMu.Unlock()
	h, err := openDevice(d)
	if err != nil {
		return 0, false, false
	}
	defer closeHandle(h.handle)
	if err = checkMouse(h, time.Sleep); err != nil {
		return 0, false, false
	}
	b, err := exchange(h, request(0xb4), 0xa1, shortReportLen, 300*time.Millisecond, time.Sleep)
	if err != nil {
		return 0, false, false
	}
	p, _, ok := decodeBattery(b)
	return p, d.PID == WiredPID && p < 100, ok
}
