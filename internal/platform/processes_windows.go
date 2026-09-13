package platform

import (
	"strings"
	"syscall"
	"unsafe"
)

func LegacyProcesses() map[string]bool {
	result := map[string]bool{}
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return map[string]bool{"lamzu": true, "vaxee": true}
	}
	defer syscall.CloseHandle(snap)
	var p syscall.ProcessEntry32
	p.Size = uint32(unsafe.Sizeof(p))
	err = syscall.Process32First(snap, &p)
	for err == nil {
		name := strings.ToLower(syscall.UTF16ToString(p.ExeFile[:]))
		for _, brand := range []string{"lamzu", "vaxee"} {
			if strings.HasPrefix(name, brand+"-autoswitch") && strings.HasSuffix(name, ".exe") {
				result[brand] = true
			}
		}
		err = syscall.Process32Next(snap, &p)
	}
	return result
}
