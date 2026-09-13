package lamzu

import (
	"fmt"
	"time"
	"unsafe"
)

func QuerySetting(dev LamzuDeviceInfo, cmd byte) ([]byte, error) {
	length := int(dev.FeatureLen)
	if length < 10 {
		return nil, fmt.Errorf("invalid feature length")
	}
	h, err := openHIDPath(dev.Path)
	if err != nil {
		return nil, err
	}
	defer closeHandle(h)
	request := buildIncaCmdVal(cmd, 0, length)
	if cmd == 0x87 {
		request = buildIncaSleep(0, length)
		off := 0
		if length == 65 {
			off = 1
		}
		request[off+5] = 0x87
	}
	r, _, _ := procHidDSetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&request[0])), uintptr(length))
	if r == 0 {
		return nil, fmt.Errorf("query send failed")
	}
	var last []byte
	for i := 0; i < 25; i++ {
		time.Sleep(20 * time.Millisecond)
		response := make([]byte, length)
		r, _, _ = procHidDGetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&response[0])), uintptr(length))
		p := response
		if len(p) > 0 && p[0] == 0 {
			p = p[1:]
		}
		last = p
		if r != 0 && len(p) > 8 && p[0] == 0xa1 && p[5] == cmd {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no valid readback for command %02x: %x", cmd, last[:min(12, len(last))])
}
