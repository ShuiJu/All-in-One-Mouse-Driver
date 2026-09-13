package main

import (
	"fmt"
	"mousehub/internal/hub"
)

var qaMode bool

type readOnlyHardware struct{ hub.WindowsHardware }

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
