package lamzu

import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

// ReadIdentity sends the official route-2 general firmware query once. It does
// not change pairing/settings or wake the mouse; sleeping devices may time out.
func ReadIdentity(dev LamzuDeviceInfo) (Identity, error) {
	if !IsControlDevice(dev) {
		return Identity{}, fmt.Errorf("unsupported LAMZU control interface")
	}
	length := int(dev.FeatureLen)
	if length != 64 && length != 65 {
		return Identity{}, fmt.Errorf("invalid LAMZU identity feature length: %d", length)
	}
	h, err := openHIDPath(dev.Path)
	if err != nil {
		return Identity{}, err
	}
	defer closeHandle(h)
	off := 0
	if length == 65 {
		off = 1
	}
	request := make([]byte, length)
	request[off+2] = 2
	request[off+3] = 0x10
	request[off+5] = 0x81
	r, _, _ := procHidDSetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&request[0])), uintptr(length))
	if r == 0 {
		return Identity{}, fmt.Errorf("LAMZU identity query send failed")
	}
	response := make([]byte, length)
	for range 25 {
		time.Sleep(20 * time.Millisecond)
		clear(response)
		r, _, _ = procHidDGetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&response[0])), uintptr(length))
		if r == 0 {
			continue
		}
		id, parseErr := ParseIdentity(response)
		if parseErr == nil || errors.Is(parseErr, ErrNoMouse) {
			return id, parseErr
		}
	}
	return Identity{}, fmt.Errorf("%w: reply %x", ErrIdentityUnavailable, response[off:off+12])
}
