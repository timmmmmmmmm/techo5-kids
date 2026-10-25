package core

import (
	"testing"
	"time"
)

// TestTheChildrenTurnTheClock is the play from the children's side: a hand taken, turned, left, and
// the clock walking home.
func TestTheChildrenTurnTheClock(t *testing.T) {
	var p Play
	now := at("14:20")
	if got := p.Shown(now); !got.Equal(now) {
		t.Errorf("no play, but shown %v", got)
	}

	// The minute hand turned a whole hour, in quarters, the hour riding along.
	p.Grab(HandMinute, now, 0)
	p.Drag(0.25)
	p.Drag(0.5)
	p.Drag(0.75)
	p.Drag(1.0)
	if got := p.Shown(now); !got.Equal(at("15:20")) {
		t.Errorf("a turn of the minute hand gave %v, want 15:20", got)
	}
	// And back again: it turns as easily the other way.
	p.Drag(0.75)
	p.Drag(0.5)
	if got := p.Shown(now); !got.Equal(at("14:50")) {
		t.Errorf("half a turn back gave %v, want 14:50", got)
	}
	// Round and round in little steps, six minutes at a time.
	p.Drag(0.6)
	p.Drag(0.7)
	if got := p.Shown(now); !got.Equal(at("15:02")) {
		t.Errorf("turns that add up gave %v, want 15:02", got)
	}

	// Let go, and it stands: ten seconds is not long enough to send it home.
	p.Release(now)
	later := now.Add(9 * time.Second)
	if p.Tick(later) {
		t.Error("the clock walks back before it is left alone long enough")
	}
	if got := p.Shown(later); !got.Equal(at("15:02")) {
		t.Errorf("left alone, it shows %v, want 15:02", got)
	}

	// Ten seconds, and it walks back, to be itself again a moment later.
	if !p.Tick(now.Add(10 * time.Second)) {
		t.Fatal("the clock did not walk back at ten seconds")
	}
	if got := p.Shown(now.Add(10*time.Second + PlayBack/2)); got.After(at("15:02")) {
		t.Errorf("midway home it shows %v, want on the way back from 15:02", got)
	}
	if p.Tick(now.Add(10*time.Second + PlayBack)) {
		t.Error("the walk back never ended")
	}
	end := now.Add(10*time.Second + PlayBack)
	if got := p.Shown(end); !got.Equal(end) {
		t.Errorf("after the walk it shows %v, want the real time", got)
	}
}

// TestTheWalkBackBounces checks the bounce: past the real time, then settling on it.
func TestTheWalkBackBounces(t *testing.T) {
	var p Play
	now := at("10:00")
	p.Grab(HandMinute, now, 0)
	p.Drag(-0.25) // 09:45, a quarter to
	p.Release(now)
	p.Tick(now.Add(PlayIdle))
	// Somewhere on the way the hands stand beyond the real time: the bounce.
	beyond := false
	for d := time.Duration(0); d <= PlayBack; d += PlayBack / 20 {
		if p.Shown(now.Add(PlayIdle + d)).After(now.Add(PlayIdle + d)) {
			beyond = true
		}
	}
	if !beyond {
		t.Error("the walk back never passes the real time: no bounce")
	}
}

// TestTheWalkTakesTheShortWayHome: the dial is twelve hours round, so the walk goes the short way,
// forward or back. Played 03:00 with the real time at 20:00, home is five hours forward, not seven
// back.
func TestTheWalkTakesTheShortWayHome(t *testing.T) {
	var p Play
	now := at("20:00")
	p.Grab(HandMinute, now, 0)
	for i := 1; i <= 28; i++ { // seven turns, a quarter at a time
		p.Drag(float64(i) * 0.25)
	}
	played := p.Shown(now)
	if got := played.Sub(at("20:00")); got != 7*time.Hour {
		t.Fatalf("seven turns gave %v, want 03:00", played)
	}
	p.Release(now)
	later := now.Add(PlayIdle)
	p.Tick(later)
	got := p.Shown(later.Add(PlayBack / 2))
	if !got.After(played) || got.Sub(played) > 5*time.Hour+time.Minute {
		t.Errorf("the long way home: midway it shows %v, want most of five hours past 03:00", got)
	}
}

// TestTheHourHandTurnsTheClockByHours: a turn of the hour hand is twelve hours, in steps as small as
// one.
func TestTheHourHandTurnsTheClockByHours(t *testing.T) {
	var p Play
	now := at("14:20")
	p.Grab(HandHour, now, 0)
	p.Drag(0.25)
	if got := p.Shown(now); !got.Equal(at("17:20")) {
		t.Errorf("a quarter of the hour hand gave %v, want 17:20", got)
	}
	p.Drag(0.25 - 1.0/12)
	if got := p.Shown(now); !got.Equal(at("16:20")) {
		t.Errorf("a twelfth back gave %v, want 16:20", got)
	}
}

// TestATapKeepsThePlayedClock: hearing the time counts as playing, so it stays a while longer.
func TestATapKeepsThePlayedClock(t *testing.T) {
	var p Play
	now := at("14:20")
	p.Grab(HandMinute, now, 0)
	p.Drag(0.25)
	p.Release(now)
	p.Touch(now.Add(9 * time.Second))
	if p.Tick(now.Add(15 * time.Second)) {
		t.Error("a tap did not keep the children's time")
	}
	if got := p.Shown(now.Add(15 * time.Second)); !got.Equal(at("14:35")) {
		t.Errorf("after a tap it shows %v, want 14:35", got)
	}
}

// TestNothingTakenHoldsNothing: a hand of none takes nothing, and a drag with nothing held does
// nothing.
func TestNothingTakenHoldsNothing(t *testing.T) {
	var p Play
	now := at("14:20")
	p.Grab(HandNone, now, 0)
	p.Drag(0.5)
	p.Release(now)
	if p.Held() != HandNone || p.Tick(now.Add(time.Hour)) {
		t.Errorf("nothing happened, but the play is %+v", p)
	}
	if got := p.Shown(now.Add(time.Hour)); !got.Equal(now.Add(time.Hour)) {
		t.Errorf("after nothing it shows %v, want the real time", got)
	}
}

// TestCancelEndsThePlayAtOnce: the menu takes the screen, and the play is gone.
func TestCancelEndsThePlayAtOnce(t *testing.T) {
	var p Play
	now := at("14:20")
	p.Grab(HandMinute, now, 0)
	p.Drag(0.5)
	p.Cancel()
	if p.Tick(now.Add(time.Hour)) || !p.Shown(now).Equal(now) {
		t.Errorf("cancelled, but the play is %+v", p)
	}
}

// TestASecondGrabStartsWhereTheFirstLeft is taking a hand again, mid-stand or mid-walk: it turns from
// the time shown, not from the real time.
func TestASecondGrabStartsWhereTheFirstLeft(t *testing.T) {
	var p Play
	now := at("14:20")
	p.Grab(HandMinute, now, 0)
	p.Drag(0.25) // 14:35
	p.Release(now)
	p.Grab(HandHour, p.Shown(now), 0)
	p.Drag(0.25) // three hours on
	if got := p.Shown(now); !got.Equal(at("17:35")) {
		t.Errorf("the second grab gave %v, want 17:35", got)
	}
}
