package hub

import "testing"

func TestIdentityUnavailableRetainsOnlySessionHistory(t *testing.T) {
	for _, brand := range []string{"lamzu", "vaxee"} {
		t.Run(brand, func(t *testing.T) {
			e, hw, _, _ := setup(t)
			hw.devices = []Device{{ID: "receiver", Brand: brand, Name: "receiver", MousePID: 0x1010, MouseModel: "known mouse", IdentityCode: "1010", IdentityStatus: IdentityCurrent, IdentitySource: "mouse reply"}}
			e.Tick()
			hw.devices[0] = Device{ID: "receiver", Brand: brand, Name: "receiver", IdentityStatus: IdentityUnavailable}
			e.DevicesChanged()
			e.Tick()
			got := e.Snapshot().Devices[0]
			if got.MouseModel != "known mouse" || got.MousePID != 0x1010 || got.IdentityStatus != IdentityCached {
				t.Fatalf("missing visibly cached mouse identity: %+v", got)
			}
			hw.devices = nil
			e.DevicesChanged()
			e.Tick()
			hw.devices = []Device{{ID: "receiver", Brand: brand, Name: "receiver", IdentityStatus: IdentityUnavailable}}
			e.DevicesChanged()
			e.Tick()
			got = e.Snapshot().Devices[0]
			if got.MouseModel != "" || got.MousePID != 0 || got.IdentityStatus != IdentityUnavailable {
				t.Fatalf("removed receiver leaked previous mouse identity: %+v", got)
			}
		})
	}
}

func TestValidUnknownMouseReplacesPreviousModel(t *testing.T) {
	e, hw, _, _ := setup(t)
	hw.devices = []Device{{ID: "receiver", Brand: "vaxee", Name: "VAXEE E1", MousePID: 0x1010, MouseModel: "VAXEE E1", IdentityCode: "1010", IdentityStatus: IdentityCurrent}}
	e.Tick()
	hw.devices[0] = Device{ID: "receiver", Brand: "vaxee", Name: "VAXEE receiver", MousePID: 0x10fe, IdentityCode: "10fe", IdentityStatus: IdentityUnknown}
	e.DevicesChanged()
	e.Tick()
	got := e.Snapshot().Devices[0]
	if got.MousePID != 0x10fe || got.MouseModel != "" || got.Name == "VAXEE E1" || got.IdentityStatus != IdentityUnknown {
		t.Fatalf("valid new unknown identity replaced with previous E1: %+v", got)
	}
	hw.devices[0] = Device{ID: "receiver", Brand: "vaxee", Name: "VAXEE receiver", IdentityStatus: IdentityUnavailable}
	e.DevicesChanged()
	e.Tick()
	got = e.Snapshot().Devices[0]
	if got.MousePID != 0x10fe || got.MouseModel != "" || got.IdentityStatus != IdentityCached {
		t.Fatalf("unknown native code was lost or old E1 resurrected: %+v", got)
	}
}

func TestMouseChangeDoesNotReusePreviousAppliedSettings(t *testing.T) {
	e, hw, _, _ := setup(t)
	hw.devices = []Device{{ID: "receiver", Brand: "lamzu", Name: "LAMZU 8K Receiver", MouseModel: "LAMZU INCA", IdentityCode: "inca", IdentityStatus: IdentityCurrent}}
	e.Tick()
	if e.Snapshot().Devices[0].Applied == nil {
		t.Fatal("initial setting was not confirmed")
	}
	hw.devices[0].MouseModel = "LAMZU PARO"
	hw.devices[0].IdentityCode = "paro"
	hw.fail = true
	e.Tick()
	got := e.Snapshot().Devices[0]
	if got.MouseModel != "LAMZU PARO" || got.Applied != nil {
		t.Fatalf("new mouse inherited previous mouse's applied state: %+v", got)
	}
}

func TestExplicitNoMouseClearsModelAndAppliedState(t *testing.T) {
	e, hw, _, _ := setup(t)
	hw.devices = []Device{{ID: "receiver", Brand: "lamzu", Name: "LAMZU 8K Receiver", MouseModel: "LAMZU INCA", IdentityCode: "PID 0x0009", IdentityStatus: IdentityCurrent}}
	e.Tick()
	writes := len(hw.writes)
	hw.devices[0] = Device{ID: "receiver", Brand: "lamzu", Name: "LAMZU 8K Receiver", IdentityStatus: IdentityDisconnected}
	e.DevicesChanged()
	e.Tick()
	got := e.Snapshot().Devices[0]
	if got.MouseModel != "" || got.IdentityCode != "" || got.Applied != nil || got.IdentityStatus != IdentityDisconnected || len(hw.writes) != writes {
		t.Fatalf("explicit no-mouse reply retained model/settings or wrote to absent mouse: %+v", got)
	}
	hw.devices[0] = Device{ID: "receiver", Brand: "lamzu", Name: "LAMZU 8K Receiver", MouseModel: "LAMZU PARO", IdentityCode: "PID 0x0007", IdentityStatus: IdentityCurrent}
	e.DevicesChanged()
	e.Tick()
	got = e.Snapshot().Devices[0]
	if got.MouseModel != "LAMZU PARO" || got.IdentityStatus != IdentityCurrent || got.Applied == nil || len(hw.writes) != writes+1 {
		t.Fatalf("new connected mouse did not receive independently confirmed settings: %+v", got)
	}
}
