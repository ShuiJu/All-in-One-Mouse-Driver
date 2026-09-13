package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"mousehub/internal/hub"
	"os"
)

func main() {
	all := flag.Bool("all", false, "Restore all saved baselines")
	flag.Parse()
	if flag.NArg() != 1 {
		panic("provide saved verification file")
	}
	b, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	var results []struct {
		Name     string
		Baseline *hub.Parameters
		Restored bool
	}
	if err = json.Unmarshal(b, &results); err != nil {
		panic(err)
	}
	hw := hub.WindowsHardware{}
	ds, err := hw.Discover()
	if err != nil {
		panic(err)
	}
	failed := false
	for _, r := range results {
		if (r.Restored && !*all) || r.Baseline == nil {
			continue
		}
		matches := []hub.Device{}
		for _, d := range ds {
			if d.Name == r.Name {
				matches = append(matches, d)
			}
		}
		if len(matches) != 1 {
			fmt.Println("Cannot uniquely identify", r.Name)
			failed = true
			continue
		}
		if err = hw.Apply(matches[0], *r.Baseline); err != nil {
			fmt.Println("RESTORE FAILED", r.Name, err)
			failed = true
		} else {
			fmt.Println("RESTORED + READBACK VERIFIED", r.Name, *r.Baseline)
		}
	}
	if failed {
		os.Exit(1)
	}
}
