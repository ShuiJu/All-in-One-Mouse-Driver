package main

import (
	"fmt"
	"mousehub/internal/hub"
	"mousehub/internal/platform"
)

var qaMode bool

func legacyProcesses() map[string]bool {
	if qaMode {
		return map[string]bool{}
	}
	return platform.LegacyProcesses()
}

type readOnlyHardware struct{ hub.WindowsHardware }

func (readOnlyHardware) Discover() ([]hub.Device, error) {
	battery := 82
	return []hub.Device{{
		ID:             "endgame-qa-op1w-4k-v2",
		Brand:          "endgame",
		Name:           "Endgame Gear OP1w 4K v2",
		MouseModel:     "OP1w 4K v2",
		Battery:        &battery,
		IdentityStatus: hub.IdentityCurrent,
		IdentityCode:   "OP1W-4K-V2-QA",
		IdentitySource: "隔离 UI 验证设备",
		Status:         "隔离验证：不会写入硬件",
	}}, nil
}
func (readOnlyHardware) Apply(hub.Device, hub.Parameters) error {
	return fmt.Errorf("UI 验证模式不写入设备")
}
func windowClassName() string {
	if qaMode {
		return "MouseHub.Native.QA.Window"
	}
	return "MouseHub.Native.Window"
}
func windowTitle() string {
	if qaMode {
		return "Mouse Hub · UI 验证"
	}
	return "Mouse Hub"
}
