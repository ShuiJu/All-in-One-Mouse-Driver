package vaxee

import (
	"encoding/binary"
	"fmt"
	"time"
	"unsafe"
)

// Device identity uses the mouse PID returned by the receiver, not its own USB
// PID or Windows ContainerID. Source: official api.js mousePidRead, v3.6.
type Identity struct {
	MousePID uint16
	Model    string
}

func ParseIdentity(report []byte) (Identity, error) {
	if len(report) < 7 || report[0] != 0x0e || report[1] != 0xa5 || report[2] != 0x0a || report[3] != 1 || report[4] != 2 {
		return Identity{}, fmt.Errorf("unexpected mouse identity response")
	}
	pid := binary.LittleEndian.Uint16(report[5:7])
	names := map[uint16]string{0x1003: "VAXEE XE", 0x1004: "VAXEE NP-01S", 0x1005: "VAXEE AX", 0x1006: "VAXEE NP-01", 0x1007: "VAXEE XE-S", 0x1008: "VAXEE XE-S-L", 0x1009: "VAXEE x NINJUTSO Sora", 0x1010: "VAXEE E1", 0x1011: "VAXEE NP-01S V2", 0x1012: "VAXEE XE V2", 0x1013: "VAXEE NP-01S Ergo", 0x1014: "VAXEE NP-01S V3", 0x1015: "VAXEE NP-01 Ergo"}
	name, ok := names[pid]
	if !ok {
		return Identity{MousePID: pid}, fmt.Errorf("unknown mouse PID 0x%04X", pid)
	}
	return Identity{MousePID: pid, Model: name}, nil
}
func ReadIdentity(dev VaxeeDeviceInfo) (Identity, error) {
	length := int(dev.FeatureLen)
	if length < 7 {
		return Identity{}, fmt.Errorf("invalid identity report length")
	}
	h, err := openHIDPath(dev.Path)
	if err != nil {
		return Identity{}, err
	}
	defer closeHandle(h)
	request := make([]byte, length)
	copy(request, []byte{0x0e, 0xa5, 0x0a, 0x01, 0x02})
	r, _, _ := procHidDSetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&request[0])), uintptr(length))
	if r == 0 {
		return Identity{}, fmt.Errorf("mouse identity query failed: %v", lastErrno())
	}
	var last []byte
	for i := 0; i < 4; i++ {
		time.Sleep(100 * time.Millisecond)
		response := make([]byte, length)
		response[0] = 0x0e
		r, _, _ := procHidDGetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&response[0])), uintptr(length))
		if r != 0 {
			last = append([]byte(nil), response[:min(12, len(response))]...)
			identity, err := ParseIdentity(response)
			if err == nil {
				return identity, nil
			}
		}
	}
	return Identity{}, fmt.Errorf("mouse identity unavailable; response=%x", last)
}
