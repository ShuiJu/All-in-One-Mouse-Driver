package hub

import (
	"encoding/json"
	"fmt"
)

type legacyBrand struct {
	Enabled   bool       `json:"enabled"`
	Default   Parameters `json:"default"`
	Game      Parameters `json:"game"`
	Processes []string   `json:"processes"`
}

func DecodeConfig(data []byte) (Config, error) {
	var header struct{ Version int }
	if err := json.Unmarshal(data, &header); err != nil {
		return Config{}, err
	}
	if header.Version == 2 {
		var c Config
		if err := json.Unmarshal(data, &c); err != nil {
			return c, err
		}
		return c, c.Validate()
	}
	if header.Version != 1 {
		return Config{}, fmt.Errorf("不支持的配置版本")
	}
	var old struct {
		Active     string
		IntervalMS int
		Profiles   []struct {
			ID, Name     string
			Lamzu, Vaxee legacyBrand
		}
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return Config{}, err
	}
	c := Config{Version: 2, Active: old.Active + "-daily", AutoEnabled: true, IntervalMS: old.IntervalMS, Bindings: []Binding{}}
	for _, p := range old.Profiles {
		daily, game := p.Name+" · 日常", p.Name+" · 游戏"
		if len(old.Profiles) == 1 && (p.Name == "日常与游戏" || p.ID == "default") {
			daily = "日常配置"
			game = "游戏配置"
		}
		lD, vD, lG, vG := p.Lamzu.Default, p.Vaxee.Default, p.Lamzu.Game, p.Vaxee.Game
		c.Profiles = append(c.Profiles, Profile{ID: p.ID + "-daily", Name: daily, Settings: map[string]*Parameters{"lamzu": &lD, "vaxee": &vD}}, Profile{ID: p.ID + "-game", Name: game, Settings: map[string]*Parameters{"lamzu": &lG, "vaxee": &vG}})
		if p.ID == old.Active {
			seen := map[string]bool{}
			for brand, b := range map[string]legacyBrand{"lamzu": p.Lamzu, "vaxee": p.Vaxee} {
				for _, proc := range b.Processes {
					proc = NormalizeProcess(proc)
					key := brand + "|" + proc
					if proc != "" && !seen[key] {
						c.Bindings = append(c.Bindings, Binding{Process: proc, ProfileID: p.ID + "-game", Brand: brand})
						seen[key] = true
					}
				}
			}
		}
	}
	return c, c.Validate()
}
