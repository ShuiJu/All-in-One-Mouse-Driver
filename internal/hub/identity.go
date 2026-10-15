package hub

const (
	IdentityCurrent      = "current"
	IdentityUnknown      = "unknown"
	IdentityUnavailable  = "unavailable"
	IdentityCached       = "cached"
	IdentityDisconnected = "disconnected"
)

// A failed read can retain a prior response only within this receiver's session.
// The state must remain visibly cached; a valid unknown code replaces old data.
func mergeMouseIdentity(current *Device, previous Device) bool {
	if current.IdentityStatus == IdentityDisconnected {
		return previous.IdentityStatus != IdentityDisconnected
	}
	if current.IdentityStatus == IdentityUnavailable {
		if previous.MouseModel != "" || previous.IdentityCode != "" || previous.MousePID != 0 {
			current.MouseModel = previous.MouseModel
			current.MousePID = previous.MousePID
			current.IdentityCode = previous.IdentityCode
			current.IdentitySource = previous.IdentitySource
			current.IdentityStatus = IdentityCached
			if current.Brand == "vaxee" {
				current.Name = previous.Name
			}
		}
		return false
	}
	if current.IdentityStatus != IdentityCurrent && current.IdentityStatus != IdentityUnknown {
		return false
	}
	return previous.IdentityCode != "" && current.IdentityCode != previous.IdentityCode
}
