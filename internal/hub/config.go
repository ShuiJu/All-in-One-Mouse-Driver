package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Parameters struct {
	Mode       string `json:"mode"`
	Poll       int    `json:"poll"`
	MotionSync bool   `json:"motionSync"`
	Sleep      int    `json:"sleep"`
	Trajectory string `json:"trajectory"`
}
type Profile struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Settings map[string]*Parameters `json:"settings"`
}
type Binding struct {
	Process   string `json:"process"`
	ProfileID string `json:"profileID"`
	Brand     string `json:"brand,omitempty"` // Only needed to retain brand-specific v1 associations.
}
type Config struct {
	Version     int               `json:"version"`
	Active      string            `json:"active"` // Fallback profile, never the profile currently being edited.
	AutoEnabled bool              `json:"autoEnabled"`
	IntervalMS  int               `json:"intervalMS"` // Retained for v1 compatibility; the engine is event driven.
	Profiles    []Profile         `json:"profiles"`
	Bindings    []Binding         `json:"bindings"`
	DeviceNames map[string]string `json:"deviceNames,omitempty"`
}

func DefaultConfig() Config {
	return Config{Version: 2, Active: "daily", AutoEnabled: true, IntervalMS: 1000, Bindings: []Binding{}, Profiles: []Profile{
		{ID: "daily", Name: "日常配置", Settings: map[string]*Parameters{"lamzu": {Mode: "office", Poll: 1000, Sleep: 30}, "vaxee": {Mode: "standard", Poll: 1000, Trajectory: "stable"}}},
		{ID: "game", Name: "游戏配置", Settings: map[string]*Parameters{"lamzu": {Mode: "20000fps", Poll: 2000, Sleep: 600}, "vaxee": {Mode: "competitive", Poll: 1000, Trajectory: "smooth"}}},
	}}
}
func NormalizeProcess(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), `\`, "/")
	return strings.ToLower(s[strings.LastIndex(s, "/")+1:])
}
func DeviceScope(d Device) string {
	if d.ID != "" {
		return d.Brand + ":" + d.ID
	}
	return d.Brand
}
func (p Profile) ForDevice(d Device) Parameters {
	if v := p.Settings[DeviceScope(d)]; v != nil {
		return *v
	}
	if v := p.Settings[d.Brand]; v != nil {
		return *v
	}
	return Parameters{}
}
func (p *Profile) EditDevice(d Device) *Parameters {
	key := DeviceScope(d)
	if p.Settings[key] == nil {
		v := p.ForDevice(d)
		p.Settings[key] = &v
	}
	return p.Settings[key]
}
func (c Config) Find(id string) (Profile, bool) {
	for _, p := range c.Profiles {
		if p.ID == id {
			return p, true
		}
	}
	return Profile{}, false
}
func (c Config) Current() Profile {
	p, ok := c.Find(c.Active)
	if !ok {
		panic("validated config has no fallback")
	}
	return p
}
func (c Config) Resolve(d Device, process string) (Parameters, string) {
	id := c.Active
	process = NormalizeProcess(process)
	for _, b := range c.Bindings {
		if b.Process == process && b.ProfileID != "" && b.Brand == "" {
			id = b.ProfileID
			break
		}
	}
	for _, b := range c.Bindings {
		if b.Process == process && b.ProfileID != "" && b.Brand == d.Brand {
			id = b.ProfileID
			break
		}
	}
	p, ok := c.Find(id)
	if !ok {
		p = c.Current()
	}
	return p.ForDevice(d), p.ID
}

// Bind replaces every association for the program. A blank profile means added,
// but not bound yet. The same method is used by drag/drop and keyboard binding.
func (c *Config) Bind(process, id string) error {
	process = NormalizeProcess(process)
	if !validProcess(process) {
		return fmt.Errorf("请选择 .exe 程序")
	}
	if id != "" {
		if _, ok := c.Find(id); !ok {
			return fmt.Errorf("配置不存在")
		}
	}
	next := make([]Binding, 0, len(c.Bindings)+1)
	for _, b := range c.Bindings {
		if b.Process != process {
			next = append(next, b)
		}
	}
	c.Bindings = append(next, Binding{Process: process, ProfileID: id})
	return nil
}
func (c *Config) RemoveProfile(id string) error {
	if len(c.Profiles) == 1 {
		return fmt.Errorf("至少保留一份配置")
	}
	if id == c.Active {
		return fmt.Errorf("请先将另一份配置设为默认")
	}
	if _, ok := c.Find(id); !ok {
		return fmt.Errorf("配置不存在")
	}
	ps := []Profile{}
	for _, p := range c.Profiles {
		if p.ID != id {
			ps = append(ps, p)
		}
	}
	c.Profiles = ps
	for i := range c.Bindings {
		if c.Bindings[i].ProfileID == id {
			c.Bindings[i].ProfileID = ""
		}
	}
	return nil
}
func validProcess(s string) bool {
	return s != "" && strings.HasSuffix(s, ".exe") && !strings.ContainsAny(s, "\r\n\x00")
}
func validateParams(brand string, p Parameters) error {
	if p.Poll != 1000 && p.Poll != 2000 && p.Poll != 4000 && !(brand == "lamzu" && p.Poll == 8000) {
		return fmt.Errorf("%s 不支持 %d Hz", brand, p.Poll)
	}
	if brand == "lamzu" {
		if p.Mode != "office" && p.Mode != "speed" && p.Mode != "20000fps" {
			return fmt.Errorf("LAMZU 性能模式无效")
		}
		if p.Sleep < 0 || p.Sleep > 65535 {
			return fmt.Errorf("休眠时间须为 0–65535 秒")
		}
	} else {
		if p.Mode != "standard" && p.Mode != "competitive" {
			return fmt.Errorf("VAXEE 性能模式无效")
		}
		if p.Mode == "standard" && p.Poll > 1000 {
			return fmt.Errorf("VAXEE 标准模式最高支持 1000 Hz")
		}
		if p.Trajectory != "smooth" && p.Trajectory != "stable" {
			return fmt.Errorf("追踪轨迹无效")
		}
	}
	return nil
}
func (c *Config) Validate() error {
	if c.DeviceNames == nil {
		c.DeviceNames = map[string]string{}
	}
	if c.Version != 2 {
		return fmt.Errorf("不支持的配置版本")
	}
	if c.IntervalMS < 250 || c.IntervalMS > 10000 {
		return fmt.Errorf("无效检测间隔")
	}
	if len(c.Profiles) < 1 || len(c.Profiles) > 100 {
		return fmt.Errorf("配置数量须为 1–100")
	}
	seen := map[string]bool{}
	for i := range c.Profiles {
		p := &c.Profiles[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.ID == "" || len(p.ID) > 100 || seen[p.ID] || p.Name == "" || len([]rune(p.Name)) > 60 {
			return fmt.Errorf("配置名称或编号无效/重复")
		}
		seen[p.ID] = true
		if p.Settings["lamzu"] == nil || p.Settings["vaxee"] == nil {
			return fmt.Errorf("配置缺少设备参数")
		}
		for key, v := range p.Settings {
			brand := strings.SplitN(key, ":", 2)[0]
			if (brand != "lamzu" && brand != "vaxee") || v == nil {
				return fmt.Errorf("未知设备参数")
			}
			if err := validateParams(brand, *v); err != nil {
				return err
			}
		}
	}
	if !seen[c.Active] {
		return fmt.Errorf("默认配置不存在")
	}
	if len(c.Bindings) > 500 {
		return fmt.Errorf("关联程序不能超过 500 个")
	}
	keys := map[string]bool{}
	for i := range c.Bindings {
		b := &c.Bindings[i]
		b.Process = NormalizeProcess(b.Process)
		key := b.Process + "|" + b.Brand
		if !validProcess(b.Process) || keys[key] {
			return fmt.Errorf("关联程序无效或重复")
		}
		keys[key] = true
		if b.Brand != "" && b.Brand != "lamzu" && b.Brand != "vaxee" {
			return fmt.Errorf("关联设备无效")
		}
		if b.ProfileID != "" && !seen[b.ProfileID] {
			return fmt.Errorf("程序关联的配置不存在")
		}
	}
	return nil
}
func Clone(c Config) Config {
	out := c
	out.DeviceNames = map[string]string{}
	for k, v := range c.DeviceNames {
		out.DeviceNames[k] = v
	}
	out.Profiles = append([]Profile(nil), c.Profiles...)
	out.Bindings = append([]Binding{}, c.Bindings...)
	for i, p := range c.Profiles {
		out.Profiles[i].Settings = map[string]*Parameters{}
		for k, v := range p.Settings {
			if v != nil {
				x := *v
				out.Profiles[i].Settings[k] = &x
			}
		}
	}
	return out
}
func SaveConfig(path string, c Config) error {
	c = Clone(c)
	if err := c.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".mousehub-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		c := DefaultConfig()
		return c, SaveConfig(path, c)
	}
	if err != nil {
		return Config{}, err
	}
	c, err := DecodeConfig(b)
	if err != nil {
		return c, err
	}
	var header struct{ Version int }
	_ = json.Unmarshal(b, &header)
	if header.Version == 1 {
		backup, err := os.OpenFile(path+".v1.bak", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, err = backup.Write(b)
			closeErr := backup.Close()
			if err != nil {
				return c, err
			}
			if closeErr != nil {
				return c, closeErr
			}
		} else if !os.IsExist(err) {
			return c, err
		}
		if err = SaveConfig(path, c); err != nil {
			return c, err
		}
	}
	return c, nil
}
