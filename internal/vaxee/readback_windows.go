package vaxee

import (
	"fmt"
	"time"
	"unsafe"
)

// QuerySetting reads a setting using the official READ opcode. It never writes
// a configuration value. Calls are serialized by the application's engine.
func QuerySetting(dev VaxeeDeviceInfo, cmd byte) ([]byte, error) {
	length := int(dev.FeatureLen)
	if length < 7 {
		return nil, fmt.Errorf("invalid feature length")
	}
	h, err := openHIDPath(dev.Path)
	if err != nil {
		return nil, err
	}
	defer closeHandle(h)
	for i := 0; i < 5; i++ {
		request := make([]byte, length)
		copy(request, []byte{0x0e, 0xa5, cmd, 1, 1})
		r, _, _ := procHidDSetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&request[0])), uintptr(length))
		if r == 0 {
			return nil, fmt.Errorf("query send failed")
		}
		time.Sleep(100 * time.Millisecond)
		response := make([]byte, length)
		response[0] = 0x0e
		r, _, _ = procHidDGetFeature_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&response[0])), uintptr(length))
		if r != 0 && response[1] == 0xa5 && response[2] == cmd && response[3] == 1 {
			return response, nil
		}
	}
	return nil, fmt.Errorf("no valid readback for command %02x", cmd)
}
