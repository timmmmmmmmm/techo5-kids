package face

import (
	"math"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// Taking the hands. The children set the clock by hand, as with the real one on the classroom wall: a
// finger on a hand takes it, and the clock follows as it turns. The geometry of the dial lives here,
// beside the drawing of it, so the two cannot drift apart.

const (
	// grabSlop is how far from a hand's body a finger may be and still take it, in dial pixels at
	// full size: the hand is wide, a finger is blunt, and the hand is what a child aims at.
	grabSlop = 32
)

// dialAt is where the dial's middle is in a scene, and the clock's scale there: the night draws it
// smaller, below the moon.
func dialAt(s Scene) (cx, cy, k float64) {
	if s.Night {
		return nightX, nightY, nightK
	}
	return float64(center), float64(center), 1
}

// Turns is the point's angle about the dial's middle, in turns: 0 at twelve, going clockwise, the
// measure the hands themselves are drawn in.
func Turns(s Scene, x, y int) float64 {
	cx, cy, _ := dialAt(s)
	a := math.Atan2(float64(x)-cx, cy-float64(y)) / (2 * math.Pi)
	if a < 0 {
		a++
	}
	return a
}

// HandAt is the hand the point is on, if either: the one whose body it is nearer. About the hub,
// where both are, the minute hand is nearer, and that is the one a child means.
func HandAt(s Scene, x, y int) core.Hand {
	cx, cy, k := dialAt(s)
	px, py := float64(x)-cx, float64(y)-cy
	t := s.Now
	minute := (float64(t.Minute()) + float64(t.Second())/60) / 60
	hour := (float64(t.Hour()%12) + float64(t.Minute())/60) / 12
	near := func(turns, length float64) float64 {
		tx, ty := at(0, 0, length*k, turns)
		l := math.Hypot(tx, ty)
		along := (px*tx + py*ty) / l // how far toward the tip, in pixels
		along = min(max(along, 0), l)
		return math.Hypot(px-tx/l*along, py-ty/l*along)
	}
	hm, mm := near(hour, hourLen), near(minute, minuteLen)
	switch {
	case min(hm, mm) > grabSlop*k:
		return core.HandNone
	case hm < mm:
		return core.HandHour
	default:
		return core.HandMinute
	}
}
