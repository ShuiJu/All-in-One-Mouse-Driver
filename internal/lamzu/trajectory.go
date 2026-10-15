package lamzu

import (
	"errors"
	"fmt"
)

// TrajectoryMode is the shared runtime timing policy exposed by LAMZU TR/v1 firmware.
type TrajectoryMode byte

const (
	TrajSmooth TrajectoryMode = 0
	TrajStable TrajectoryMode = 1
)

var ErrTrajectoryUnsupported = errors.New("LAMZU trajectory requires firmware with TR protocol version 1")

// ParseTrajectoryReply accepts the ten-byte response, HID padding, or ReportID 0.
// Legacy A2/A3 replies and unmarked zero values do not establish capability.
func ParseTrajectoryReply(report []byte) (TrajectoryMode, error) {
	if len(report) == 65 && report[0] == 0 {
		report = report[1:]
	}
	if len(report) < 10 || len(report) > 64 || report[0] != 0xa1 || report[1] != 0 || report[2] != 2 || report[3] != 4 || report[4] != 1 || report[5] != 0x9e || report[6] != 1 || report[8] != 0x54 || report[9] != 0x52 || report[7] > 1 {
		return 0, ErrTrajectoryUnsupported
	}
	return TrajectoryMode(report[7]), nil
}

func validateTrajectory(mode TrajectoryMode) error {
	if mode != TrajSmooth && mode != TrajStable {
		return fmt.Errorf("invalid LAMZU trajectory mode: %d", mode)
	}
	return nil
}
