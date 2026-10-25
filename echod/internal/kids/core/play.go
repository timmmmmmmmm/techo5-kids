package core

import (
	"math"
	"time"
)

// The children playing clockmaker. A finger takes a hand and turns it, and the clock stays where they
// leave it for a while, then walks back to the real time with a little bounce that says it is itself
// again. These are the rules, in times and angles alone; the fingers are the clock service's
// (clock.go), which keeps all of this behind its mutex.

// Hand is one of the dial's hands.
type Hand int

const (
	HandNone Hand = iota
	HandHour
	HandMinute
)

// minutesPerTurn is what one full turn of the hand is worth: the minute hand goes round once an hour,
// the hour hand once in twelve, so turning either one is the same as turning the clock itself.
func (h Hand) minutesPerTurn() time.Duration {
	if h == HandHour {
		return 12 * time.Hour
	}
	return time.Hour
}

const (
	// PlayIdle is how long the clock stays where the children left it before walking back.
	PlayIdle = 10 * time.Second

	// PlayBack is how long that walk back takes: long enough to read as a turn, at the frame rate
	// the Spot draws.
	PlayBack = 1500 * time.Millisecond
)

// playState is where the play is.
type playState int

const (
	playOff   playState = iota // the real clock
	playHeld                   // a finger has a hand
	playSet                    // the children's time stands, left alone for now
	playBack                   // walking back to the real time
)

// Play is the clock the children set by hand. Zero is the real clock, showing nothing of its own.
type Play struct {
	state  playState
	held   Hand
	anchor time.Time    // where the turning started: the time shown when the hand was taken, on the minute
	turned time.Duration // how far the finger has turned it from the anchor
	finger float64       // the finger's angle at the last drag, in turns: 0 at twelve, going clockwise
	idle   time.Time     // playSet: when the hands were let go
	backAt time.Time     // playBack: when the walk back began
}

// Grab takes hand at the finger's angle, from the time shown now. A hand of none takes nothing.
func (p *Play) Grab(hand Hand, shown time.Time, finger float64) {
	if hand == HandNone {
		return
	}
	p.state, p.held = playHeld, hand
	p.anchor, p.turned, p.finger = shown.Round(time.Minute), 0, finger
	p.idle, p.backAt = time.Time{}, time.Time{}
}

// Drag turns the held hand to the finger's angle, taking up the turn since the last one: a hand may
// go round and round and the hours ride along, forward or back, as on the clock itself.
func (p *Play) Drag(finger float64) {
	if p.state != playHeld {
		return
	}
	d := finger - p.finger
	d -= math.Round(d) // one drag is the short way round; round and round adds up all the same
	p.finger = finger
	p.turned += time.Duration(d * float64(p.held.minutesPerTurn()))
}

// Release lets the hand go: the children's time stands, for now.
func (p *Play) Release(now time.Time) {
	if p.state != playHeld {
		return
	}
	p.state, p.idle = playSet, now
}

// Touch is a tap while the children's time stands: hearing it counts as playing, so it stays a while
// longer.
func (p *Play) Touch(now time.Time) {
	if p.state == playSet {
		p.idle = now
	}
}

// Cancel ends the play at once, wherever it is: the real clock again. The parent menu wants the
// screen to itself.
func (p *Play) Cancel() {
	*p = Play{}
}

// Held is the hand a finger has, none while it has none.
func (p *Play) Held() Hand {
	if p.state != playHeld {
		return HandNone
	}
	return p.held
}

// Playing says whether the clock shows the children's time, or is walking back to the real one.
func (p *Play) Playing() bool {
	return p.state != playOff
}

// Turned is how far the held hand has been turned. A grab that never turned anything was no grab.
func (p *Play) Turned() bool {
	return p.turned != 0
}

// Tick advances the play to the real time now: a time left alone walks back, and the walk ends. It
// says whether the walk is still going, for whoever draws the frames of it.
func (p *Play) Tick(now time.Time) bool {
	switch {
	case p.state == playSet && now.Sub(p.idle) >= PlayIdle:
		p.state, p.backAt = playBack, now
	case p.state == playBack && now.Sub(p.backAt) >= PlayBack:
		p.state = playOff
	}
	return p.state == playBack
}

// playBounce eases the walk back: quickly away, a little past the real time, settling. u runs 0 to 1.
func playBounce(u float64) float64 {
	const c1 = 0.9 // wider is a bigger bounce; 1.7, the usual back, lands a tenth of the way past
	u--
	return 1 + (c1+1)*u*u*u + c1*u*u
}

// Shown is the time the clock shows at the real time now: the children's while they play, the real
// one when they do not. Tick first, or the walk back never ends.
func (p *Play) Shown(now time.Time) time.Time {
	switch p.state {
	case playHeld, playSet:
		return p.anchor.Add(p.turned).Round(time.Minute)
	case playBack:
		set := p.anchor.Add(p.turned)
		way := now.Sub(set)
		for way > 6*time.Hour { // the dial is twelve hours round: the short way home
			way -= 12 * time.Hour
		}
		for way <= -6*time.Hour {
			way += 12 * time.Hour
		}
		u := float64(now.Sub(p.backAt)) / float64(PlayBack)
		return set.Add(time.Duration(float64(way) * playBounce(u)))
	}
	return now
}
