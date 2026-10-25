package face

import (
	"math"
	"testing"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// TestAHandIsTakenWhereItIsDrawn is the children's grab: each hand where it is drawn, the hub the
// minute hand's, and the empty dial nobody's.
func TestAHandIsTakenWhereItIsDrawn(t *testing.T) {
	now := time.Date(2026, 10, 25, 14, 7, 38, 0, time.Local)
	s := Scene{Now: now, Style: core.StyleYellow}
	minute := (float64(now.Minute()) + float64(now.Second())/60) / 60
	hour := (float64(now.Hour()%12) + float64(now.Minute())/60) / 12
	mx, my := at(center, center, minuteLen, minute)
	hx, hy := at(center, center, hourLen, hour)
	if got := HandAt(s, int(mx), int(my)); got != core.HandMinute {
		t.Errorf("the minute hand's tip is %v", got)
	}
	if got := HandAt(s, int(hx), int(hy)); got != core.HandHour {
		t.Errorf("the hour hand's tip is %v", got)
	}
	// Along the shaft, not only the tip: a child presses the hand, not the point of it.
	sx, sy := at(center, center, hourLen*2/3, hour)
	if got := HandAt(s, int(sx), int(sy)); got != core.HandHour {
		t.Errorf("the hour hand's shaft is %v", got)
	}
	// The hub, where both hands are, is the minute hand's: the one a child means.
	if got := HandAt(s, center, center); got != core.HandMinute {
		t.Errorf("the hub is %v", got)
	}
	if got := HandAt(s, 5, 5); got != core.HandNone {
		t.Errorf("a corner is %v", got)
	}
	// The night's small clock below the moon, taken the same.
	sn := Scene{Now: now, Night: true}
	nx, ny := at(nightX, nightY, minuteLen*nightK, minute)
	if got := HandAt(sn, int(nx), int(ny)); got != core.HandMinute {
		t.Errorf("the night's minute hand is %v", got)
	}
	if got := HandAt(sn, center, center); got != core.HandNone {
		t.Errorf("the empty night sky is %v", got)
	}
}

// TestTheFingersAngleIsTheClocksOwn: three o'clock is a quarter turn, twelve is none, and the night's
// small clock measures from its own middle.
func TestTheFingersAngleIsTheClocksOwn(t *testing.T) {
	s := Scene{Now: time.Date(2026, 10, 25, 10, 0, 0, 0, time.Local)}
	if got := Turns(s, center+100, center); math.Abs(got-0.25) > 1e-9 {
		t.Errorf("three o'clock is %v turns", got)
	}
	if got := Turns(s, center, center-100); got != 0 {
		t.Errorf("twelve is %v turns", got)
	}
	if got := Turns(s, center, center+100); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("six is %v turns", got)
	}
	if got := Turns(Scene{Now: s.Now, Night: true}, int(nightX)+50, int(nightY)); math.Abs(got-0.25) > 1e-9 {
		t.Errorf("the night's three o'clock is %v turns", got)
	}
}
