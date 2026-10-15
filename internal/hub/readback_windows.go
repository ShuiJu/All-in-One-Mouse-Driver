package hub

import (
	"fmt"
	"mousehub/internal/lamzu"
	"mousehub/internal/vaxee"
)

func (WindowsHardware) ReadSettings(d Device) (Parameters, error) {
	p := Parameters{}
	if d.Brand == "vaxee" {
		values := map[byte][]byte{}
		for _, cmd := range []byte{7, 8, 0x13} {
			b, err := vaxee.QuerySetting(d.v, cmd)
			if err != nil {
				return p, err
			}
			if len(b) < 6 || b[3] != 1 || b[4] != 1 {
				return p, fmt.Errorf("unsupported VAXEE response for %02x", cmd)
			}
			values[cmd] = b
		}
		polls := map[byte]int{1: 500, 2: 1000, 3: 2000, 4: 4000}
		p.Poll = polls[values[7][5]]
		mode := values[8][5]
		if mode < 1 || mode > 4 || p.Poll == 0 {
			return p, fmt.Errorf("unknown VAXEE performance response")
		}
		p.Mode = "standard"
		if mode == 1 || mode == 3 {
			p.Mode = "competitive"
		}
		p.MotionSync = mode == 3 || mode == 4
		switch values[0x13][5] {
		case 0:
			p.Trajectory = "smooth"
		case 1:
			p.Trajectory = "stable"
		default:
			return p, fmt.Errorf("unknown VAXEE trajectory response")
		}
		return p, nil
	}
	values := map[byte][]byte{}
	for _, cmd := range []byte{0x80, 0x89, 0x8b, 0x93, 0x87} {
		b, err := lamzu.QuerySetting(d.l, cmd)
		if err != nil {
			return p, err
		}
		if len(b) < 9 || b[6] != 1 {
			return p, fmt.Errorf("unsupported LAMZU profile response")
		}
		values[cmd] = b
	}
	polls := map[byte]int{1: 1000, 0x10: 1000, 0x20: 2000, 0x40: 4000, 0x80: 8000}
	p.Poll = polls[values[0x80][7]]
	if p.Poll == 0 {
		return p, fmt.Errorf("unknown LAMZU polling response")
	}
	high, tracking, ms := values[0x8b][7], values[0x93][7], values[0x89][7]
	if high > 1 || tracking > 1 || ms > 1 {
		return p, fmt.Errorf("unknown LAMZU mode response")
	}
	p.Mode = "office"
	if high == 1 {
		p.Mode = "speed"
	}
	if tracking == 1 {
		if high != 1 {
			return p, fmt.Errorf("unsupported LAMZU mode combination")
		}
		p.Mode = "20000fps"
	}
	p.MotionSync = ms == 1
	p.Sleep = int(values[0x87][7])<<8 | int(values[0x87][8])
	if d.TrajectorySupported {
		trajectory, err := lamzu.ReadTrajectory(d.l)
		if err != nil {
			return p, err
		}
		if trajectory == lamzu.TrajSmooth {
			p.Trajectory = "smooth"
		} else {
			p.Trajectory = "stable"
		}
	}
	return p, nil
}
func verifyApplied(d Device, want Parameters) error {
	actual, err := (WindowsHardware{}).ReadSettings(d)
	if err != nil {
		return fmt.Errorf("设置已发送，但读回未确认：%w", err)
	}
	if d.Brand == "lamzu" && want.Trajectory == "" {
		// Unspecified profiles leave firmware policy alone; other values still
		// require exact readback confirmation.
		actual.Trajectory = ""
	}
	if actual != want {
		return fmt.Errorf("设置读回不一致：期望 %+v，实际 %+v", want, actual)
	}
	return nil
}
