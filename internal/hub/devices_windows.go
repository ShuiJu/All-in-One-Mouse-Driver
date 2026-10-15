package hub

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"mousehub/internal/lamzu"
	"mousehub/internal/platform"
	"mousehub/internal/vaxee"
	"sort"
	"strings"
)

type Device struct {
	ID                  string      `json:"id"`
	Brand               string      `json:"brand"`
	Name                string      `json:"name"`
	Battery             *int        `json:"battery"`
	Charging            bool        `json:"charging"`
	Status              string      `json:"status"`
	Applied             *Parameters `json:"applied"`
	Profile             string      `json:"profile"`
	MousePID            uint16      `json:"mousePID"`
	MouseModel          string      `json:"mouseModel,omitempty"`
	IdentityStatus      string      `json:"identityStatus,omitempty"`
	IdentityCode        string      `json:"identityCode,omitempty"`
	IdentitySource      string      `json:"identitySource"`
	TrajectorySupported bool        `json:"trajectorySupported"`
	path                string
	l                   lamzu.LamzuDeviceInfo
	v                   vaxee.VaxeeDeviceInfo
	serial              string
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
		if !lamzu.IsControlDevice(d) || seen[p] {
			continue
		}
		seen[p] = true
		identity, identityErr := lamzu.ReadIdentity(d)
		device := Device{ID: deviceID("lamzu", p), Brand: "lamzu", Name: "LAMZU 8K Receiver", path: d.Path, l: d, serial: platform.HIDSerial(d.Path), IdentityStatus: IdentityUnavailable, IdentitySource: "未收到有效鼠标身份回复"}
		if errors.Is(identityErr, lamzu.ErrNoMouse) {
			device.IdentityStatus = IdentityDisconnected
			device.IdentitySource = "接收器明确回复无鼠标连接"
		} else {
			_, trajectoryErr := lamzu.ReadTrajectory(d)
			device.TrajectorySupported = trajectoryErr == nil
			if identityErr == nil {
				device.MousePID = identity.MousePID
				device.MouseModel = identity.Model
				device.IdentityCode = fmt.Sprintf("PID 0x%04X", identity.MousePID)
				device.IdentitySource = "鼠标固件版本回复中的 PID"
				device.IdentityStatus = IdentityCurrent
				if identity.Model == "" {
					device.IdentityStatus = IdentityUnknown
				}
			}
		}
		out = append(out, device)
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
			device := Device{ID: deviceID("vaxee", d.Path), Brand: "vaxee", Name: "VAXEE 接收器", path: d.Path, v: d, serial: platform.HIDSerial(d.Path), TrajectorySupported: true, IdentityStatus: IdentityUnavailable, IdentitySource: "未收到有效鼠标身份回复"}
			if readErr == nil {
				device.MousePID = identity.MousePID
				device.MouseModel = identity.Model
				device.IdentityCode = fmt.Sprintf("PID 0x%04X", identity.MousePID)
				device.IdentitySource = "接收器鼠标 PID 查询"
				device.IdentityStatus = IdentityCurrent
				if identity.Model != "" {
					device.Name = identity.Model
				} else {
					device.IdentityStatus = IdentityUnknown
					device.Name = "VAXEE 鼠标"
				}
			}
			out = append(out, device)
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
		var trajectory *lamzu.TrajectoryMode
		if p.Trajectory != "" {
			if !d.TrajectorySupported {
				return lamzu.ErrTrajectoryUnsupported
			}
			mode := lamzu.TrajStable
			if p.Trajectory == "smooth" {
				mode = lamzu.TrajSmooth
			}
			trajectory = &mode
		}
		if err := lamzu.ApplyLamzuSetting(d.path, m, lamzu.PollingRate(p.Poll), p.MotionSync, p.Sleep, trajectory); err != nil {
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
		p, ch, ok = lamzu.ReadBattery(d.l)
	} else {
		p, ch, ok = vaxee.ReadBatteryVAXEE(d.v)
	}
	if !ok {
		return nil, false
	}
	return &p, ch
}
