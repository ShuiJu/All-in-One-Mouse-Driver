//go:build windows

package endgame

import (
	"fmt"
	"syscall"
	"unsafe"
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type HIDD_ATTRIBUTES struct {
	Size      uint32
	VendorID  uint16
	ProductID uint16
	Version   uint16
}

type SP_DEVICE_INTERFACE_DATA struct {
	CbSize             uint32
	InterfaceClassGuid GUID
	Flags              uint32
	Reserved           uintptr
}

type HIDP_CAPS struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

const ERROR_NO_MORE_ITEMS syscall.Errno = 259
const HIDP_STATUS_SUCCESS uint32 = 0x00110000

var (
	setupapiHID = syscall.NewLazyDLL("setupapi.dll")
	hidDLLHID   = syscall.NewLazyDLL("hid.dll")
	k32HID      = syscall.NewLazyDLL("kernel32.dll")

	procSetupDiGetClassDevsW_HID             = setupapiHID.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces_HID      = setupapiHID.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW_HID = setupapiHID.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList_HID     = setupapiHID.NewProc("SetupDiDestroyDeviceInfoList")

	procHidDGetHidGuid_HID            = hidDLLHID.NewProc("HidD_GetHidGuid")
	procHidDGetAttributes_HID         = hidDLLHID.NewProc("HidD_GetAttributes")
	procHidDGetManufacturerString_HID = hidDLLHID.NewProc("HidD_GetManufacturerString")
	procHidDGetProductString_HID      = hidDLLHID.NewProc("HidD_GetProductString")

	procHidDSetFeature_HID        = hidDLLHID.NewProc("HidD_SetFeature")
	procHidDGetFeature_HID        = hidDLLHID.NewProc("HidD_GetFeature")
	procHidDGetPreparsedData_HID  = hidDLLHID.NewProc("HidD_GetPreparsedData")
	procHidDFreePreparsedData_HID = hidDLLHID.NewProc("HidD_FreePreparsedData")
	procHidPGetCaps_HID           = hidDLLHID.NewProc("HidP_GetCaps")

	procCreateFileW_HID  = k32HID.NewProc("CreateFileW")
	procCloseHandle_HID  = k32HID.NewProc("CloseHandle")
	procGetLastError_HID = k32HID.NewProc("GetLastError")
)

const (
	DIGCF_PRESENT         = 0x00000002
	DIGCF_DEVICEINTERFACE = 0x00000010

	GENERIC_READ  = 0x80000000
	GENERIC_WRITE = 0x40000000

	FILE_SHARE_READ  = 0x00000001
	FILE_SHARE_WRITE = 0x00000002

	OPEN_EXISTING = 3
)

func detailCbSizeW() uint32 {
	if unsafe.Sizeof(uintptr(0)) == 8 {
		return 8
	}
	return 6
}

const detailDevicePathOffset = 4

type DeviceInfo struct {
	Path         string
	VID          uint16
	PID          uint16
	Manufacturer string
	Product      string
	FeatureLen   uint16
	Usage        uint16
	UsagePage    uint16
}

func lastErrno() syscall.Errno {
	r1, _, _ := procGetLastError_HID.Call()
	return syscall.Errno(r1)
}

func openHIDPath(path string) (syscall.Handle, error) {
	p16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	h, _, _ := procCreateFileW_HID.Call(
		uintptr(unsafe.Pointer(p16)),
		uintptr(GENERIC_READ|GENERIC_WRITE),
		uintptr(FILE_SHARE_READ|FILE_SHARE_WRITE),
		0,
		uintptr(OPEN_EXISTING),
		0,
		0,
	)
	if h != 0 && h != uintptr(syscall.InvalidHandle) {
		return syscall.Handle(h), nil
	}
	h2, _, _ := procCreateFileW_HID.Call(
		uintptr(unsafe.Pointer(p16)),
		uintptr(GENERIC_WRITE),
		uintptr(FILE_SHARE_READ|FILE_SHARE_WRITE),
		0,
		uintptr(OPEN_EXISTING),
		0,
		0,
	)
	if h2 != 0 && h2 != uintptr(syscall.InvalidHandle) {
		return syscall.Handle(h2), nil
	}
	return 0, fmt.Errorf("CreateFileW failed: %s (%v)", path, lastErrno())
}

func openHIDPathForQuery(path string) (syscall.Handle, error) {
	p16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	h, _, _ := procCreateFileW_HID.Call(
		uintptr(unsafe.Pointer(p16)),
		0,
		uintptr(FILE_SHARE_READ|FILE_SHARE_WRITE),
		0,
		uintptr(OPEN_EXISTING),
		0,
		0,
	)
	if h != 0 && h != uintptr(syscall.InvalidHandle) {
		return syscall.Handle(h), nil
	}
	return 0, fmt.Errorf("CreateFileW(query) failed: %s (%v)", path, lastErrno())
}

func closeHandle(h syscall.Handle) {
	procCloseHandle_HID.Call(uintptr(h))
}

func hidGuid() GUID {
	var g GUID
	procHidDGetHidGuid_HID.Call(uintptr(unsafe.Pointer(&g)))
	return g
}

func utf16FromPtr(p *uint16) string {
	if p == nil {
		return ""
	}
	var arr []uint16
	for i := 0; ; i++ {
		u := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(i*2)))
		if u == 0 {
			break
		}
		arr = append(arr, u)
	}
	return syscall.UTF16ToString(arr)
}

func hidGetString(h syscall.Handle, proc *syscall.LazyProc) string {
	buf := make([]uint16, 256)
	r1, _, _ := proc.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2))
	if r1 == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func queryCaps(h syscall.Handle) (HIDP_CAPS, error) {
	var pp uintptr
	r1, _, _ := procHidDGetPreparsedData_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&pp)))
	if r1 == 0 || pp == 0 {
		return HIDP_CAPS{}, fmt.Errorf("HidD_GetPreparsedData failed: %v", lastErrno())
	}
	defer procHidDFreePreparsedData_HID.Call(pp)

	var caps HIDP_CAPS
	st, _, _ := procHidPGetCaps_HID.Call(pp, uintptr(unsafe.Pointer(&caps)))
	if uint32(st) != HIDP_STATUS_SUCCESS {
		return HIDP_CAPS{}, fmt.Errorf("HidP_GetCaps failed: 0x%08x", uint32(st))
	}
	return caps, nil
}

func queryDeviceInfo(path string) (DeviceInfo, bool) {
	h, err := openHIDPathForQuery(path)
	if err != nil {
		return DeviceInfo{}, false
	}
	defer closeHandle(h)

	var attr HIDD_ATTRIBUTES
	attr.Size = uint32(unsafe.Sizeof(attr))
	r1, _, _ := procHidDGetAttributes_HID.Call(uintptr(h), uintptr(unsafe.Pointer(&attr)))
	if r1 == 0 {
		return DeviceInfo{}, false
	}

	manu := hidGetString(h, procHidDGetManufacturerString_HID)
	prod := hidGetString(h, procHidDGetProductString_HID)

	caps, _ := queryCaps(h)

	return DeviceInfo{
		Path: path, VID: attr.VendorID, PID: attr.ProductID,
		Manufacturer: manu, Product: prod,
		FeatureLen: caps.FeatureReportByteLength, Usage: caps.Usage, UsagePage: caps.UsagePage,
	}, true
}

func EnumerateDevices() ([]DeviceInfo, error) {
	g := hidGuid()

	hDevInfo, _, _ := procSetupDiGetClassDevsW_HID.Call(
		uintptr(unsafe.Pointer(&g)), 0, 0,
		uintptr(DIGCF_PRESENT|DIGCF_DEVICEINTERFACE),
	)
	if hDevInfo == 0 || hDevInfo == uintptr(syscall.InvalidHandle) {
		return nil, fmt.Errorf("SetupDiGetClassDevsW failed: %v", lastErrno())
	}
	defer procSetupDiDestroyDeviceInfoList_HID.Call(hDevInfo)

	var out []DeviceInfo
	for idx := 0; ; idx++ {
		var ifData SP_DEVICE_INTERFACE_DATA
		ifData.CbSize = uint32(unsafe.Sizeof(ifData))

		r1, _, eEnum := procSetupDiEnumDeviceInterfaces_HID.Call(
			hDevInfo, 0,
			uintptr(unsafe.Pointer(&g)),
			uintptr(idx),
			uintptr(unsafe.Pointer(&ifData)),
		)
		if r1 == 0 {
			if errno, ok := eEnum.(syscall.Errno); ok && errno == ERROR_NO_MORE_ITEMS {
				break
			}
			break
		}

		var required uint32
		procSetupDiGetDeviceInterfaceDetailW_HID.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&ifData)),
			0, 0,
			uintptr(unsafe.Pointer(&required)),
			0,
		)
		if required == 0 {
			continue
		}

		buf := make([]byte, required)
		*(*uint32)(unsafe.Pointer(&buf[0])) = detailCbSizeW()

		r2, _, _ := procSetupDiGetDeviceInterfaceDetailW_HID.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&ifData)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(required),
			uintptr(unsafe.Pointer(&required)),
			0,
		)
		if r2 == 0 {
			continue
		}

		pathPtr := (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(&buf[0])) + detailDevicePathOffset))
		path := utf16FromPtr(pathPtr)
		if path == "" {
			continue
		}

		info, ok := queryDeviceInfo(path)
		if !ok {
			continue
		}

		// Accept only the OP1w normal-mode configuration collection.
		if IsControlDevice(info) {
			out = append(out, info)
		}
	}
	return out, nil
}
