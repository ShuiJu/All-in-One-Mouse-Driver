package vaxee

import (
	"encoding/binary"
	"errors"
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

// ErrIdentityUnavailable means that no usable mouse identity was obtained. It
// does not establish that the receiver has no paired or awake mouse.
var ErrIdentityUnavailable = errors.New("mouse identity unavailable")

// ErrIdentityResponse also matches ErrIdentityUnavailable. A response with the
// wrong command, opcode, length, or an empty PID is not a usable identity.
var ErrIdentityResponse = fmt.Errorf("%w: invalid response", ErrIdentityUnavailable)

func ParseIdentity(report []byte) (Identity, error) {
	if len(report) < 7 || report[0] != 0x0e || report[1] != 0xa5 || report[2] != 0x0a || report[3] != 1 || report[4] != 2 {
		return Identity{}, ErrIdentityResponse
	}
	pid := binary.LittleEndian.Uint16(report[5:7])
	if pid == 0 || pid == 0xffff {
		return Identity{}, fmt.Errorf("%w: empty mouse PID 0x%04X", ErrIdentityResponse, pid)
	}
	return Identity{MousePID: pid, Model: mouseModel(pid)}, nil
}

func mouseModel(pid uint16) string {
	switch pid {
	case 0x1003:
		return "VAXEE XE"
	case 0x1004:
		return "VAXEE NP-01S"
	case 0x1005:
		return "VAXEE AX"
	case 0x1006:
		return "VAXEE NP-01"
	case 0x1007:
		return "VAXEE XE-S"
	case 0x1008:
		return "VAXEE XE-S-L"
	case 0x1009:
		return "VAXEE x NINJUTSO Sora"
	case 0x1010:
		return "VAXEE E1"
	case 0x1011:
		return "VAXEE NP-01S V2"
	case 0x1012:
		return "VAXEE XE V2"
	case 0x1013:
		return "VAXEE NP-01S Ergo"
	case 0x1014:
		return "VAXEE NP-01S V3"
	case 0x1015:
		return "VAXEE NP-01 Ergo"
	default:
		return ""
	}
}
func ReadIdentity(dev VaxeeDeviceInfo) (Identity, error) {
	length := int(dev.FeatureLen)
	if length < 7 {
		return Identity{}, fmt.Errorf("%w: invalid report length %d", ErrIdentityUnavailable, length)
	}
	h, err := openHIDPath(dev.Path)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %w", ErrIdentityUnavailable, err)
	}
	defer closeHandle(h)
	request := make([]byte, length)
	copy(request, []byte{0x0e, 0xa5, 0x0a, 0x01, 0x02})
	r, _, _ := procHidDSetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&request[0])), uintptr(length))
	if r == 0 {
		return Identity{}, fmt.Errorf("%w: query failed: %v", ErrIdentityUnavailable, lastErrno())
	}
	response := make([]byte, length)
	var last [12]byte
	lastLen := min(len(last), length)
	for i := 0; i < 4; i++ {
		time.Sleep(100 * time.Millisecond)
		clear(response)
		response[0] = 0x0e
		r, _, _ := procHidDGetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&response[0])), uintptr(length))
		if r != 0 {
			copy(last[:], response[:lastLen])
			identity, err := ParseIdentity(response)
			if err == nil {
				return identity, nil
			}
		}
	}
	return Identity{}, fmt.Errorf("%w: response=%x", ErrIdentityUnavailable, last[:lastLen])
}
