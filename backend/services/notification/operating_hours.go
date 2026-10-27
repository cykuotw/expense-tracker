package notification

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"time"
	_ "time/tzdata"
)

// OperatingHours is public policy, independent of database availability.
type OperatingHours struct {
	Enabled    bool   `json:"enabled"`
	Start      string `json:"start"`
	Stop       string `json:"stop"`
	SafetyStop string `json:"safety_stop"`
	Timezone   string `json:"timezone"`
}

func SenderAllowed(reader io.Reader, now time.Time) (bool, error) {
	var hours OperatingHours
	if err := json.UnmarshalRead(io.LimitReader(reader, 4097), &hours); err != nil {
		return false, errors.New("invalid operating-hours policy")
	}
	if !hours.Enabled {
		return true, nil
	}
	zone, err := time.LoadLocation(hours.Timezone)
	if err != nil {
		return false, errors.New("invalid operating-hours timezone")
	}
	start, err := time.Parse("15:04", hours.Start)
	if err != nil {
		return false, errors.New("invalid operating-hours start")
	}
	stop, err := time.Parse("15:04", hours.Stop)
	if err != nil {
		return false, errors.New("invalid operating-hours stop")
	}
	startMinute, stopMinute := start.Hour()*60+start.Minute(), stop.Hour()*60+stop.Minute()
	duration := (stopMinute - startMinute + 1440) % 1440
	if duration == 0 {
		return false, errors.New("empty operating-hours window")
	}
	local := now.In(zone)
	return (local.Hour()*60+local.Minute()-startMinute+1440)%1440 < duration, nil
}

// CheckSenderHours applies no database calls outside the expected window. A
// manual out-of-hours start serves HTTP normally but does not resume push sends.
func CheckSenderHours(ctx context.Context, fetch func(context.Context) (io.ReadCloser, error), now time.Time) (bool, error) {
	reader, err := fetch(ctx)
	if err != nil {
		return false, errors.New("load operating-hours policy failed")
	}
	defer reader.Close()
	return SenderAllowed(reader, now)
}
