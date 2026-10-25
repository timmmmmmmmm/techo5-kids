package core

import (
	"fmt"
	"time"
)

// minuteOf is the minute of the day an HH:MM time names.
func minuteOf(hm string) (int, bool) {
	var h, m int
	if n, err := fmt.Sscanf(hm, "%d:%d", &h, &m); n != 2 || err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// Night reports whether t is between bedtime and waketime. The night crosses midnight when bedtime
// is the later of the two, as it usually is; a bedtime equal to waketime is no night at all.
func (s Settings) Night(t time.Time) bool {
	bed, ok1 := minuteOf(s.Bedtime)
	wake, ok2 := minuteOf(s.Waketime)
	if !ok1 || !ok2 || bed == wake {
		return false
	}
	now := t.Hour()*60 + t.Minute()
	if bed > wake {
		return now >= bed || now < wake
	}
	return now >= bed && now < wake
}
