package lamzu

import "testing"

func TestInvalidPollingRejected(t *testing.T) {
	if _, err := pollingToYY(125); err == nil {
		t.Fatal("invalid polling accepted")
	}
}
func TestIncaPerfCommandOrder(t *testing.T) {
	tests := []struct {
		name string
		mode PerfMode
		want []incaPerfCommand
	}{
		{
			name: "office",
			mode: PerfOffice,
			want: []incaPerfCommand{{cmd: 0x0B, val: 0x00}, {cmd: 0x13, val: 0x00}},
		},
		{
			name: "speed",
			mode: PerfSpeed,
			want: []incaPerfCommand{{cmd: 0x0B, val: 0x00}, {cmd: 0x13, val: 0x00}, {cmd: 0x0B, val: 0x01}},
		},
		{
			name: "high performance",
			mode: Perf20000FPS,
			want: []incaPerfCommand{{cmd: 0x0B, val: 0x00}, {cmd: 0x13, val: 0x00}, {cmd: 0x13, val: 0x01}, {cmd: 0x0B, val: 0x01}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, count, err := incaPerfCommands(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(tt.want) {
				t.Fatalf("count=%d, want %d", count, len(tt.want))
			}
			for i, want := range tt.want {
				if commands[i] != want {
					t.Fatalf("command %d=%+v, want %+v", i, commands[i], want)
				}
			}
		})
	}
}

func TestLamzuControlDeviceWhitelist(t *testing.T) {
	dev := LamzuDeviceInfo{VID: 0x37b0, PID: 0x0010, Path: `\\?\HID#VID_37B0&PID_0010&MI_02#control`}
	if !IsControlDevice(dev) {
		t.Fatal("receiver control interface was rejected")
	}
	dev.Path = `\\?\hid#vid_37b0&pid_0010&mi_02&col01#control`
	if !IsControlDevice(dev) {
		t.Fatal("mi_02 HID collection was rejected")
	}
	for _, test := range []struct {
		name string
		vid  uint16
		pid  uint16
		path string
	}{
		{"different vendor", 0x3057, 0x0010, `\\?\hid#vid_3057&pid_0010&mi_02#control`},
		{"PARO wired mouse", 0x37b0, 0x0007, `\\?\hid#vid_37b0&pid_0007&mi_02#control`},
		{"INCA wired mouse", 0x37b0, 0x0009, `\\?\hid#vid_37b0&pid_0009&mi_02#control`},
		{"PARO bootloader", 0x37b0, 0x0008, `\\?\hid#vid_37b0&pid_0008&mi_02#control`},
		{"INCA bootloader", 0x37b0, 0x000a, `\\?\hid#vid_37b0&pid_000a&mi_02#control`},
		{"unknown mouse", 0x37b0, 0x0011, `\\?\hid#vid_37b0&pid_0011&mi_02#control`},
		{"input interface", 0x37b0, 0x0010, `\\?\hid#vid_37b0&pid_0010&mi_00#input`},
		{"keyboard collection", 0x37b0, 0x0010, `\\?\hid#vid_37b0&pid_0010&mi_02#control\kbd`},
		{"interface prefix", 0x37b0, 0x0010, `\\?\hid#vid_37b0&pid_0010&mi_020#input`},
		{"interface text in instance", 0x37b0, 0x0010, `\\?\hid#vid_37b0&pid_0010&mi_00#mi_02`},
		{"fake interface field in instance", 0x37b0, 0x0010, `\\?\hid#vid_37b0&pid_0010&mi_00#instance&mi_02#control`},
		{"empty path", 0x37b0, 0x0010, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			dev := LamzuDeviceInfo{VID: test.vid, PID: test.pid, Path: test.path}
			if IsControlDevice(dev) {
				t.Fatal("unsupported device or interface accepted for control")
			}
		})
	}
}
