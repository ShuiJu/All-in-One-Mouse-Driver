package main

import (
	"encoding/json"
	"fmt"
	"mousehub/internal/lamzu"
	"mousehub/internal/vaxee"
)

func main() {
	out := map[string]any{}
	ld, _ := lamzu.FindOneLamzuDevice()
	ls := []lamzu.LamzuDeviceInfo{ld}
	for _, d := range ls {
		if d.FeatureLen < 10 {
			continue
		}
		pages := map[string]string{}
		for _, cmd := range []byte{0x80, 0x89, 0x8b, 0x93, 0x87} {
			b, err := lamzu.QuerySetting(d, cmd)
			if err != nil {
				pages[fmt.Sprintf("%02x", cmd)] = err.Error()
			} else {
				pages[fmt.Sprintf("%02x", cmd)] = fmt.Sprintf("%x", b[:min(12, len(b))])
			}
		}
		out["LAMZU"] = pages
		break
	}
	vs, _ := vaxee.SelectAllVaxeeControlPaths()
	for _, d := range vs {
		identity, _ := vaxee.ReadIdentity(d)
		pages := map[string]string{}
		for _, cmd := range []byte{0x07, 0x08, 0x13} {
			b, err := vaxee.QuerySetting(d, cmd)
			if err != nil {
				pages[fmt.Sprintf("%02x", cmd)] = err.Error()
			} else {
				pages[fmt.Sprintf("%02x", cmd)] = fmt.Sprintf("%x", b[:min(12, len(b))])
			}
		}
		out[identity.Model] = pages
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}
