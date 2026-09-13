package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type dialogExchange struct {
	Save                                  bool
	Title, Filter, Extension, Path, Error string
}

// The native shell picker is isolated so shell extensions and thumbnail caches
// are released when it exits. There is only one resident application process.
func runFileDialog(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var d dialogExchange
	if json.Unmarshal(b, &d) != nil {
		return
	}
	d.Path, err = chooseFile(0, d.Save, d.Title, d.Filter, d.Extension)
	if err != nil {
		d.Error = err.Error()
	}
	b, _ = json.Marshal(d)
	_ = os.WriteFile(path, b, 0600)
}
func (u *nativeUI) pickFile(save bool, title, filter, extension string) (string, error) {
	f, err := os.CreateTemp(u.data, ".dialog-*.json")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)
	request := dialogExchange{Save: save, Title: title, Filter: filter, Extension: extension}
	b, _ := json.Marshal(request)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(exe, "--file-dialog", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err = cmd.Run(); err != nil {
		return "", err
	}
	b, err = os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var response dialogExchange
	if err = json.Unmarshal(b, &response); err != nil {
		return "", err
	}
	if response.Error != "" {
		return "", fmt.Errorf("%s", response.Error)
	}
	if response.Path == "" {
		return "", fmt.Errorf("已取消文件选择")
	}
	return response.Path, nil
}
