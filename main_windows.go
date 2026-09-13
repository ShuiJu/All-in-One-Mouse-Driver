package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mousehub/internal/hub"
	"mousehub/internal/platform"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	runtime.LockOSThread()
	probe := flag.Bool("probe", false, "Read device identities without applying settings")
	dataArg := flag.String("data", "", "Portable data directory")
	hidden := flag.Bool("hidden", false, "Start in the system tray")
	dialogFile := flag.String("file-dialog", "", "Internal native file-picker request")
	qa := flag.Bool("qa", false, "Isolated UI verification without device writes")
	flag.Parse()
	qaMode = *qa
	if *dialogFile != "" {
		runFileDialog(*dialogFile)
		return
	}
	exe, _ := os.Executable()
	data := *dataArg
	if data == "" {
		data = filepath.Join(filepath.Dir(exe), "data")
	}
	if *probe {
		devices, err := (hub.WindowsHardware{}).Discover()
		payload := map[string]any{"devices": devices, "conflicts": platform.LegacyProcesses()}
		if err != nil {
			payload["error"] = err.Error()
		}
		b, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(b))
		return
	}
	if !singleInstance() {
		return
	}
	if err := os.MkdirAll(data, 0700); err != nil {
		message(err.Error())
		return
	}
	logger, err := os.OpenFile(filepath.Join(data, "mousehub.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		defer logger.Close()
		log.SetOutput(logger)
	}
	var hardware hub.Hardware = hub.WindowsHardware{}
	if qaMode {
		hardware = readOnlyHardware{}
	}
	engine, err := hub.NewEngine(filepath.Join(data, "config.json"), hardware, platform.ForegroundProcessName, platform.LegacyProcesses)
	if err != nil {
		message("配置读取失败，原文件已保留：\n" + err.Error())
		return
	}
	ui, err := newNativeUI(engine, data)
	if err != nil {
		message(err.Error())
		return
	}
	defer ui.cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.SetNotify(ui.notify)
	engine.SetVisible(!*hidden)
	go engine.Run(ctx)
	ui.run(*hidden)
}
