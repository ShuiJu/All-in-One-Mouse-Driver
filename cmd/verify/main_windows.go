package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"mousehub/internal/hub"
	"mousehub/internal/platform"
	"os"
	"time"
)

type result struct {
	Name     string
	Baseline *hub.Parameters
	Steps    []string
	Error    string
	Restored bool
}

func main() {
	write := flag.Bool("write", false, "Apply test profiles and restore hardware settings")
	exhaustive := flag.Bool("exhaustive", false, "Test every offered mode/rate/sync/trajectory combination")
	output := flag.String("out", "research/hardware-verification.json", "Evidence output file")
	brand := flag.String("brand", "", "Limit verification to one brand (lamzu/vaxee/endgame)")
	flag.Parse()
	if c := platform.LegacyProcesses(); c["lamzu"] || c["vaxee"] || c["endgame"] {
		fmt.Println("Exit legacy apps before testing")
		os.Exit(1)
	}
	hw := hub.WindowsHardware{}
	devices, err := hw.Discover()
	if err != nil {
		fmt.Println(err)
	}
	if *brand != "" {
		selected := devices[:0]
		for _, d := range devices {
			if d.Brand == *brand {
				selected = append(selected, d)
			}
		}
		devices = selected
	}
	results := []result{}
	ok := len(devices) > 0
	save := func() { b, _ := json.MarshalIndent(results, "", "  "); _ = os.WriteFile(*output, b, 0600) }
	for _, d := range devices {
		r := result{Name: d.Name}
		base, err := hw.ReadSettings(d)
		if err != nil {
			r.Error = "baseline unavailable: " + err.Error()
			results = append(results, r)
			save()
			ok = false
			continue
		}
		r.Baseline = &base
		if !*write {
			r.Steps = append(r.Steps, "read-only baseline captured")
			results = append(results, r)
			save()
			continue
		}
		if base.Poll == 500 {
			r.Error = "500 Hz baseline cannot yet be restored by the supported writer"
			results = append(results, r)
			save()
			ok = false
			continue
		}
		// Persist baseline before the first write, then restore even after a failure.
		results = append(results, r)
		save()
		index := len(results) - 1
		func() {
			defer func() {
				restoreErr := hw.Apply(d, base)
				results[index].Restored = restoreErr == nil
				if restoreErr != nil {
					results[index].Error += " RESTORE FAILED: " + restoreErr.Error()
					ok = false
				}
				save()
			}()
			cases := []hub.Parameters{{Mode: "standard", Poll: 1000, Trajectory: "smooth"}, {Mode: "standard", Poll: 1000, MotionSync: true, Trajectory: "stable"}, {Mode: "competitive", Poll: 2000, Trajectory: "smooth"}, {Mode: "competitive", Poll: 4000, MotionSync: true, Trajectory: "stable"}}
			if d.Brand == "lamzu" {
				cases = []hub.Parameters{{Mode: "office", Poll: 1000, Sleep: 30}, {Mode: "speed", Poll: 2000, MotionSync: true, Sleep: 60}, {Mode: "20000fps", Poll: 4000, Sleep: 300}, {Mode: "20000fps", Poll: 8000, MotionSync: true, Sleep: 600}}
			}
			if d.Brand == "endgame" {
				cases = []hub.Parameters{
					{Mode: "standard", Poll: 1000, LOD: base.LOD, PowerSaving: 1, DeepSleep: 2},
					{Mode: "competitive", Poll: 4000, LOD: base.LOD, DeepSleep: 3},
					{Mode: "standard", Poll: 2000, LOD: base.LOD, MotionSync: true, DeepSleep: 120},
				}
			}
			if *exhaustive {
				cases = nil
				if d.Brand == "lamzu" {
					for _, mode := range []string{"office", "speed", "20000fps"} {
						for _, poll := range []int{1000, 2000, 4000, 8000} {
							for _, ms := range []bool{false, true} {
								cases = append(cases, hub.Parameters{Mode: mode, Poll: poll, MotionSync: ms, Sleep: 600})
							}
						}
					}
					cases = append(cases, hub.Parameters{Mode: "office", Poll: 1000, Sleep: 1800})
				} else if d.Brand == "endgame" {
					lods := []int{7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}
					if d.GlassMode {
						lods = []int{10, 20}
					}
					for _, lod := range lods {
						for _, poll := range []int{125, 1000, 2000, 4000} {
							for _, mode := range []string{"standard", "competitive"} {
								for _, ms := range []bool{false, true} {
									cases = append(cases, hub.Parameters{Mode: mode, Poll: poll, LOD: lod, MotionSync: ms, DeepSleep: 3})
								}
							}
						}
					}
					cases = append(cases, hub.Parameters{Mode: "standard", Poll: 1000, LOD: base.LOD, PowerSaving: 120, DeepSleep: 0})
				} else {
					for _, mode := range []string{"standard", "competitive"} {
						for _, poll := range []int{1000, 2000, 4000} {
							if mode == "standard" && poll > 1000 {
								continue
							}
							for _, ms := range []bool{false, true} {
								for _, traj := range []string{"smooth", "stable"} {
									cases = append(cases, hub.Parameters{Mode: mode, Poll: poll, MotionSync: ms, Trajectory: traj})
								}
							}
						}
					}
				}
			}
			if d.Brand == "lamzu" && d.TrajectorySupported {
				withTrajectory := make([]hub.Parameters, 0, len(cases)*2)
				for _, p := range cases {
					for _, trajectory := range []string{"smooth", "stable"} {
						p.Trajectory = trajectory
						withTrajectory = append(withTrajectory, p)
					}
				}
				cases = withTrajectory
			}
			for _, p := range cases {
				if err := hw.Apply(d, p); err != nil {
					results[index].Error = err.Error()
					ok = false
					return
				}
				results[index].Steps = append(results[index].Steps, fmt.Sprintf("PASS write + readback %+v", p))
				save()
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
	b, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(b))
	if !ok {
		os.Exit(1)
	}
}
