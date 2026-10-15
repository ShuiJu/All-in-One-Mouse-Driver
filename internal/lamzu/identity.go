package lamzu

import "errors"

// Identity comes from the connected mouse's general firmware-version reply.
// MousePID is the native big-endian identity field, not the receiver USB PID.
type Identity struct {
	MousePID uint16
	Model    string
}

var (
	ErrIdentityUnavailable = errors.New("LAMZU connected mouse identity unavailable")
	ErrNoMouse             = errors.New("LAMZU receiver reports no connected mouse")
)

// ParseIdentity accepts a normalized reply, HID padding, or ReportID 0.
// Route 2 reaches the radio mouse; route 0/1 and profile-group 1 replies cannot
// establish this identity. An unknown or zero identity remains a valid reply.
func ParseIdentity(report []byte) (Identity, error) {
	if len(report) == 65 && report[0] == 0 {
		report = report[1:]
	}
	if len(report) < 6 || len(report) > 64 || report[1] != 0 || report[2] != 2 || report[4] != 0 || report[5] != 0x81 {
		return Identity{}, ErrIdentityUnavailable
	}
	// The original receiver 820 explicitly returns A4 when its connection
	// predicate fails. A0 is only a pending reply, never evidence of no mouse.
	if report[0] == 0xa4 && report[3] == 0x10 {
		return Identity{}, ErrNoMouse
	}
	if len(report) < 12 || report[0] != 0xa1 || report[3] != 6 {
		return Identity{}, ErrIdentityUnavailable
	}
	id := Identity{MousePID: uint16(report[10])<<8 | uint16(report[11])}
	switch id.MousePID {
	case 0x0009:
		id.Model = "LAMZU INCA"
	case 0x0007:
		id.Model = "LAMZU PARO"
	}
	return id, nil
}
