package platform

import "testing"

func TestLegacyProcessBrand(t *testing.T) {
	cases := map[string]string{
		"LAMZU-AutoSwitch.exe":                                 "lamzu",
		"vaxee-autoswitch.exe":                                 "vaxee",
		"Endgame_Gear_OP1w_4k_v2_Configuration_Tool_v1_02.exe": "endgame",
		"Endgame_Gear_OP1w_4k_v2_Configuration_Tool.exe":       "endgame",
		"Endgame Gear OP1w 4k v2 Configuration Tool.exe":       "endgame",
		"notepad.exe": "",
	}
	for process, want := range cases {
		if got := legacyProcessBrand(process); got != want {
			t.Errorf("legacyProcessBrand(%q) = %q, want %q", process, got, want)
		}
	}
}
