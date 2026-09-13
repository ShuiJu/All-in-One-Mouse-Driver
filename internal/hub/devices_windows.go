package hub

import (
	"crypto/sha256"
	"fmt"
	"mousehub/internal/lamzu"
	"mousehub/internal/platform"
	"mousehub/internal/vaxee"
	"sort"
	"strings"
)

type Device struct {
	ID             string      `json:"id"`
	Brand          string      `json:"brand"`
	Name           string      `json:"name"`
	Battery        *int        `json:"battery"`
	Charging       bool        `json:"charging"`
	Status         string      `json:"status"`
	Applied        *Parameters `json:"applied"`
	Profile        string      `json:"profile"`
	MousePID       uint16      `json:"mousePID"`
	IdentitySource string      `json:"identitySource"`
	path           string
	l              lamzu.LamzuDeviceInfo
	v              vaxee.VaxeeDeviceInfo
	serial         string
}
type Hardware interface {
	Discover() ([]Device, error)
	Apply(Device, Parameters) error
	Battery(Device) (*int, bool)
}
type WindowsHardware struct{}

func deviceID(brand, path string) string {
	return fmt.Sprintf("%s-%x", brand, sha256.Sum256([]byte(strings.ToLower(path))))[:22]
}
func (WindowsHardware) Discover() ([]Device, error) {
	out := []Device{}
	var errs []string
	ls, err := lamzu.EnumerateLamzuDevices()
	if err != nil {
		errs = append(errs, err.Error())
	}
	seen := map[string]bool{}
	for _, d := range ls {
		p := strings.ToLower(d.Path)
		if !strings.Contains(p, "mi_02") || strings.HasSuffix(p, `\kbd`) || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, Device{ID: deviceID("lamzu", p), Brand: "lamzu", Name: "LAMZU INCA", path: d.Path, l: d, serial: platform.HIDSerial(d.Path)})
	}
	vs, err := vaxee.EnumerateVaxeeDevices()
	if err != nil {
		errs = append(errs, err.Error())
	}
	if len(vs) > 0 {
		controls, e := vaxee.SelectAllVaxeeControlPathsFrom(vs)
		if e != nil {
			errs = append(errs, e.Error())
		}
		for _, d := range controls {
			identity, readErr := vaxee.ReadIdentity(d)
			name, source := "VAXEE · 型号未确认", "未取得鼠标型号响应"
			if readErr == nil {
				name = identity.Model
				source = "接收器查询鼠标 PID（0x0A）"
			} else {
				source = readErr.Error()
			}
			out = append(out, Device{ID: deviceID("vaxee", d.Path), Brand: "vaxee", Name: name, path: d.Path, v: d, MousePID: identity.MousePID, IdentitySource: source, serial: platform.HIDSerial(d.Path)})
		}
	}
	assignDeviceIDs(out)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(errs) > 0 {
		return out, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}

// Identity must not change merely because another same-model device is plugged
// in. A container identifies the receiver; the path disambiguates devices on
// systems that expose neither a stable container nor a unique serial number.
func assignDeviceIDs(ds []Device) {
	for i := range ds {
		d := &ds[i]
		key := "path:" + d.path
		if d.v.ContainerID != "" {
			key = "container:" + d.v.ContainerID
		} else if d.serial != "" {
			key = "serial:" + d.serial + "|path:" + d.path
		}
		d.ID = deviceID(d.Brand, key)
	}
}
func (WindowsHardware) Apply(d Device, p Parameters) error {
	if err := validateParams(d.Brand, p); err != nil {
		return err
	}
	if d.Brand == "lamzu" {
		m := lamzu.PerfOffice
		if p.Mode == "speed" {
			m = lamzu.PerfSpeed
		}
		if p.Mode == "20000fps" {
			m = lamzu.Perf20000FPS
		}
		if err := lamzu.ApplyLamzuSetting(d.path, m, lamzu.PollingRate(p.Poll), p.MotionSync, p.Sleep); err != nil {
			return err
		}
		return verifyApplied(d, p)
	}
	m := vaxee.PerfStandardMSOff
	if p.Mode == "competitive" {
		m = vaxee.PerfCompetitiveMSOff
	}
	if p.MotionSync {
		m += 2
	}
	t := vaxee.TrajStableControl
	if p.Trajectory == "smooth" {
		t = vaxee.TrajSmoothSensitive
	}
	_, err := vaxee.ApplyVaxeeProfileToAll([]vaxee.VaxeeDeviceInfo{d.v}, m, vaxee.PollingRate(p.Poll), t)
	if err != nil {
		return err
	}
	return verifyApplied(d, p)
}
func (WindowsHardware) Battery(d Device) (*int, bool) {
	var p int
	var ch, ok bool
	if d.Brand == "lamzu" {
		p, ch, ok = lamzu.ReadBatteryINCA(d.l)
	} else {
		p, ch, ok = vaxee.ReadBatteryVAXEE(d.v)
	}
	if !ok {
		return nil, false
	}
	return &p, ch
}
