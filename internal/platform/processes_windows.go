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
		return map[string]bool{"lamzu": true, "vaxee": true, "endgame": true}
	}
	defer syscall.CloseHandle(snap)
	var p syscall.ProcessEntry32
	p.Size = uint32(unsafe.Sizeof(p))
	err = syscall.Process32First(snap, &p)
	for err == nil {
		name := strings.ToLower(syscall.UTF16ToString(p.ExeFile[:]))
		if brand := legacyProcessBrand(name); brand != "" {
			result[brand] = true
		}
		err = syscall.Process32Next(snap, &p)
	}
	return result
}

func legacyProcessBrand(name string) string {
	name = strings.ToLower(name)
	for _, brand := range []string{"lamzu", "vaxee"} {
		if strings.HasPrefix(name, brand+"-autoswitch") && strings.HasSuffix(name, ".exe") {
			return brand
		}
	}
	if name == "endgame_gear_op1w_4k_v2_configuration_tool_v1_02.exe" ||
		name == "endgame_gear_op1w_4k_v2_configuration_tool.exe" ||
		name == "endgame gear op1w 4k v2 configuration tool.exe" {
		return "endgame"
	}
	return ""
}
