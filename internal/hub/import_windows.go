package hub

import (
	"fmt"
	"mousehub/internal/lamzu"
	"mousehub/internal/vaxee"
	"time"
)

func (e *Engine) ImportLegacy(brand, path string) error {
	c := e.Snapshot().Config
	var imported legacyBrand
	if brand == "lamzu" {
		old, err := lamzu.LoadLegacy(path)
		if err != nil {
			return err
		}
		convert := func(m lamzu.PerfMode, poll lamzu.PollingRate, ms bool, sleep int) Parameters {
			mode := "office"
			if m == lamzu.PerfSpeed {
				mode = "speed"
			}
			if m == lamzu.Perf20000FPS {
				mode = "20000fps"
			}
			return Parameters{Mode: mode, Poll: int(poll), MotionSync: ms, Sleep: sleep}
		}
		imported = legacyBrand{Default: convert(old.DefaultMode, old.DefaultPoll, old.DefaultMotionSync, old.DefaultSleepSec), Game: convert(old.HitMode, old.HitPoll, old.HitMotionSync, old.HitSleepSec), Processes: old.Whitelist}
	} else if brand == "vaxee" {
		old, err := vaxee.LoadLegacy(path)
		if err != nil {
			return err
		}
		convert := func(m vaxee.PerfMode, poll vaxee.PollingRate, t vaxee.TrajectoryMode) Parameters {
			mode := "standard"
			if m == vaxee.PerfCompetitiveMSOff || m == vaxee.PerfCompetitiveMSOn {
				mode = "competitive"
			}
			traj := "stable"
			if t == vaxee.TrajSmoothSensitive {
				traj = "smooth"
			}
			return Parameters{Mode: mode, Poll: int(poll), MotionSync: m == vaxee.PerfStandardMSOn || m == vaxee.PerfCompetitiveMSOn, Trajectory: traj}
		}
		imported = legacyBrand{Default: convert(old.DefaultMode, old.DefaultPoll, old.DefaultTraj), Game: convert(old.HitMode, old.HitPoll, old.HitTraj), Processes: old.Whitelist}
	} else {
		return fmt.Errorf("未知品牌")
	}
	base := c.Current()
	tmp := Clone(Config{Profiles: []Profile{base, base}})
	id := fmt.Sprintf("import-%d", time.Now().UnixNano())
	daily, game := tmp.Profiles[0], tmp.Profiles[1]
	daily.ID = id + "-daily"
	daily.Name = brand + " 日常"
	daily.Settings[brand] = &imported.Default
	game.ID = id + "-game"
	game.Name = brand + " 游戏"
	game.Settings[brand] = &imported.Game
	c.Profiles = append(c.Profiles, daily, game)
	c.Active = daily.ID
	for _, proc := range imported.Processes {
		proc = NormalizeProcess(proc)
		kept := []Binding{}
		for _, b := range c.Bindings {
			if b.Process != proc || b.Brand != brand {
				kept = append(kept, b)
			}
		}
		c.Bindings = append(kept, Binding{Process: proc, ProfileID: game.ID, Brand: brand})
	}
	return e.Update(c)
}
