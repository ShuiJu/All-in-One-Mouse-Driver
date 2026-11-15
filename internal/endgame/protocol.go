package endgame

import (
	"fmt"
	"time"
)

// Settings contains only the controls exposed by Mouse Hub. LOD is measured in
// tenths of a millimetre; sleep timers are minutes, with zero meaning disabled.
type Settings struct {
	Poll, LOD, PowerSaving, DeepSleep int
	ForceMaxFPS, MotionSync           bool
	GlassMode                         bool // Read-only; applying settings preserves the device's glass mode.
}

func (s Settings) Validate() error {
	switch s.Poll {
	case 125, 1000, 2000, 4000:
	default:
		return fmt.Errorf("OP1w 4K v2 不支持 %d Hz", s.Poll)
	}
	if (s.LOD < 7 || s.LOD > 17) && s.LOD != 20 {
		return fmt.Errorf("OP1w 4K v2 LOD 无效")
	}
	if s.PowerSaving < 0 || s.PowerSaving > 120 || s.DeepSleep < 0 || s.DeepSleep > 120 {
		return fmt.Errorf("省电和深度休眠须为 0–120 分钟")
	}
	if s.Poll != 1000 && s.PowerSaving != 0 {
		return fmt.Errorf("OP1w 4K v2 省电模式仅支持 1000 Hz")
	}
	return nil
}

func request(cmd byte) []byte {
	b := make([]byte, shortReportLen)
	b[0], b[1] = 0xa1, cmd
	return b
}

// The official program reads status 3 as busy and retries GetFeature without
// repeating the command. Never accept a failed/busy response as current data.
func exchange(t featureTransport, command []byte, id byte, length int, delay time.Duration, pause func(time.Duration)) ([]byte, error) {
	if err := t.Set(command); err != nil {
		return nil, err
	}
	pause(delay)
	for attempt := 0; attempt < 3; attempt++ {
		b, err := t.Get(id, length)
		if err != nil {
			return nil, err
		}
		// The receiver returns an A1 header even for the large A0 GetFeature
		// request. The official application checks status rather than requiring
		// the returned ID to equal the requested ID; allow that observed form.
		validID := len(b) > 0 && (b[0] == id || (id == 0xa0 && b[0] == 0xa1))
		if len(b) != length || !validID {
			return nil, fmt.Errorf("OP1w 4K v2 回复长度或报告编号无效: 命令 %02x，报告 %02x，长度 %d，回复 %x", command[1], id, len(b), b[:min(len(b), 16)])
		}
		if b[1] == 1 {
			return b, nil
		}
		if b[1] != 3 {
			return nil, fmt.Errorf("OP1w 4K v2 命令 %02x 失败，状态 %02x", command[1], b[1])
		}
		if attempt < 2 {
			pause(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("OP1w 4K v2 忙碌或离线")
}

func readProfile(t featureTransport, pause func(time.Duration)) ([]byte, error) {
	b, err := exchange(t, request(0x12), 0xa0, profileReportLen, 350*time.Millisecond, pause)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b[0x10:0x410]...), nil
}

func checkMouse(t featureTransport, pause func(time.Duration)) error {
	b, err := exchange(t, request(0x0e), 0xa1, shortReportLen, 300*time.Millisecond, pause)
	if err != nil {
		return err
	}
	pid := uint16(b[0x12]) | uint16(b[0x13])<<8
	if pid != WiredPID {
		return fmt.Errorf("未收到 OP1w 4K v2 鼠标身份回复: PID %04x", pid)
	}
	return nil
}

func timerValue(b byte) (int, error) {
	if b&0x80 != 0 {
		return 0, nil
	}
	if b == 0 || b > 120 {
		return 0, fmt.Errorf("OP1w 4K v2 休眠时间回复无效: %02x", b)
	}
	return int(b), nil
}

func decodeSettings(raw []byte) (Settings, error) {
	s := Settings{}
	if len(raw) != 0x400 {
		return s, fmt.Errorf("OP1w 4K v2 配置回复长度无效")
	}
	s.GlassMode = raw[0x6f] == 1
	if raw[0x6f] > 1 {
		return s, fmt.Errorf("未知的 OP1w 4K v2 Glass Mode")
	}
	s.LOD = int(raw[9]) + 7
	if s.GlassMode {
		if raw[9] != 1 && raw[9] != 2 {
			return s, fmt.Errorf("无效 Glass Mode LOD 回复")
		}
		s.LOD = int(raw[9]) * 10
	} else if raw[9] > 10 {
		return s, fmt.Errorf("未知的 OP1w 4K v2 LOD")
	}
	switch raw[5] {
	case 0, 0x80:
		s.Poll = 1000
	case 4:
		s.Poll = 2000
	case 2:
		s.Poll = 4000
	case 0x40:
		s.Poll = 125
	default:
		return s, fmt.Errorf("未知的 OP1w 4K v2 回报率: %02x", raw[5])
	}
	if raw[0x0c] > 1 {
		return s, fmt.Errorf("未知的 OP1w 4K v2 Motion Sync")
	}
	s.MotionSync = raw[0x0c] == 1
	s.ForceMaxFPS = raw[6]&0x40 != 0
	var err error
	s.PowerSaving, err = timerValue(raw[4])
	if err != nil {
		return s, err
	}
	s.DeepSleep, err = timerValue(raw[3])
	return s, err
}

// Preserve inactive timer durations when disabling them. The high bit alone
// controls enablement, so unrelated stored values need not be reset on switch.
func encodeTimer(minutes int, old byte) byte {
	if minutes != 0 {
		return byte(minutes)
	}
	n := old & 0x7f
	if n < 1 || n > 120 {
		n = 1
	}
	return n | 0x80
}

func buildReports(raw []byte, want Settings) (sensor, performance []byte, err error) {
	if err = want.Validate(); err != nil {
		return nil, nil, err
	}
	if _, err = decodeSettings(raw); err != nil {
		return nil, nil, err
	}
	next := append([]byte(nil), raw...)
	if raw[0x6f] == 1 {
		if want.LOD != 10 && want.LOD != 20 {
			return nil, nil, fmt.Errorf("当前 Glass Mode 的 LOD 仅支持 1.0 / 2.0 mm")
		}
		next[9] = byte(want.LOD / 10)
	} else {
		if want.LOD == 20 {
			return nil, nil, fmt.Errorf("2.0 mm LOD 需要设备当前已开启 Glass Mode")
		}
		next[9] = byte(want.LOD - 7)
	}
	next[3], next[4] = encodeTimer(want.DeepSleep, raw[3]), encodeTimer(want.PowerSaving, raw[4])
	next[5] = 0
	if want.PowerSaving != 0 {
		next[5] = 0x80
	}
	if want.Poll == 2000 {
		next[5] = 4
	}
	if want.Poll == 4000 {
		next[5] = 2
	}
	if want.Poll == 125 {
		next[5] = 0x40
	}
	next[6] &^= 0x40
	if want.ForceMaxFPS {
		next[6] |= 0x40
	}
	next[0x0c] = 0
	if want.MotionSync {
		next[0x0c] = 1
	}
	if next[9] != raw[9] {
		sensor = request(0x14)
		sensor[2], sensor[3] = 0x0f, 0x1c
		// EXE 404ce0 serializes the CPI/LOD group. Preserve every CPI stage,
		// angle and filter byte instead of constructing a default sensor profile.
		order := []int{7, 8, 9, 10, 11, 1, 14, 13, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54}
		for i, off := range order {
			sensor[0x10+i] = next[off]
		}
	}
	performance = request(0x15)
	performance[2], performance[3] = 0x0f, 0x0a
	// EXE 404b80 includes per-CPI-stage button/filter bytes and Glass Mode.
	order := []int{12, 5, 6, 4, 61, 68, 75, 82, 89, 3, 111}
	changed := false
	for i, off := range order {
		performance[0x10+i] = next[off]
		changed = changed || next[off] != raw[off]
	}
	if !changed {
		performance = nil
	}
	return sensor, performance, nil
}

func applySettings(t featureTransport, want Settings, pause func(time.Duration), writeDelay time.Duration) error {
	raw, err := readProfile(t, pause)
	if err != nil {
		return err
	}
	sensor, performance, err := buildReports(raw, want)
	if err != nil {
		return err
	}
	for _, report := range [][]byte{sensor, performance} {
		if report == nil {
			continue
		}
		if _, err = exchange(t, report, 0xa1, shortReportLen, writeDelay, pause); err != nil {
			return err
		}
	}
	if sensor == nil && performance == nil {
		return nil
	}
	after, err := readProfile(t, pause)
	if err != nil {
		return fmt.Errorf("设置已发送，但读回未确认: %w", err)
	}
	got, err := decodeSettings(after)
	if err != nil {
		return err
	}
	// Glass Mode is read-only and excluded from the requested controls.
	got.GlassMode = want.GlassMode
	if got != want {
		return fmt.Errorf("OP1w 4K v2 设置读回不一致: 期望 %+v，实际 %+v", want, got)
	}
	return nil
}

func decodeBattery(b []byte) (int, bool, bool) {
	if len(b) != shortReportLen || b[0] != 0xa1 || b[1] != 1 || b[0x10] > 100 {
		return 0, false, false
	}
	return int(b[0x10]), false, true
}
