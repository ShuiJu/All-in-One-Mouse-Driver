package lamzu

import (
	"fmt"
	"time"
	"unsafe"
)

func QuerySetting(dev LamzuDeviceInfo, cmd byte) ([]byte, error) {
	if !IsControlDevice(dev) {
		return nil, fmt.Errorf("unsupported LAMZU control interface")
	}
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
		if len(p) == 65 && p[0] == 0 {
			p = p[1:]
		}
		last = p
		if r != 0 && len(p) > 8 && p[0] == 0xa1 && p[5] == cmd {
			if cmd == 0x9e {
				if _, err := ParseTrajectoryReply(p); err != nil {
					continue
				}
			}
			return p, nil
		}
	}
	return nil, fmt.Errorf("no valid readback for command %02x: %x", cmd, last[:min(12, len(last))])
}

func ReadTrajectory(dev LamzuDeviceInfo) (TrajectoryMode, error) {
	reply, err := QuerySetting(dev, 0x9e)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrTrajectoryUnsupported, err)
	}
	return ParseTrajectoryReply(reply)
}

func ApplyTrajectory(dev LamzuDeviceInfo, mode TrajectoryMode) error {
	if err := validateTrajectory(mode); err != nil {
		return err
	}
	actual, err := selectedControlDevice(dev.Path)
	if err != nil {
		return err
	}
	dev = actual
	if _, err := ReadTrajectory(dev); err != nil {
		return err
	}
	return writeTrajectory(dev, mode)
}

func writeTrajectory(dev LamzuDeviceInfo, mode TrajectoryMode) error {
	if err := sendFeatureReport(dev.Path, buildIncaCmdVal(0x1e, byte(mode), int(dev.FeatureLen))); err != nil {
		return fmt.Errorf("LAMZU trajectory write failed: %w", err)
	}
	time.Sleep(lamzuCommandInterval)
	actual, err := ReadTrajectory(dev)
	if err != nil {
		return fmt.Errorf("LAMZU trajectory write was not confirmed: %w", err)
	}
	if actual != mode {
		return fmt.Errorf("LAMZU trajectory readback mismatch: requested %d, received %d", mode, actual)
	}
	return nil
}
