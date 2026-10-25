package face

import (
	"image/color"
	"math"
	"strconv"
	"sync"
	"time"

	"golang.org/x/image/font"

	"github.com/HuskerMinion/techo5/echod/internal/lib/fonts"
)

// The learning clock, after the classroom one: the hours are big red numbers and the short hour hand
// is red too; the minutes are a ring of dots numbered every five, out to 60, and the long minute hand is
// blue. Both hands end in an arrow's head. The night draws the same clock smaller and quieter.

// look is one way of coloring the clock.
type look struct {
	ground, rim                     color.RGBA
	hours, minutes                  color.RGBA
	hourHand, minuteHand, peg       color.RGBA
	edge                            color.RGBA // round the numbers and hands; none if A is 0
	shadow                          color.RGBA // under the hands; none if A is 0
	minuteNumbers                   bool       // the 5 … 60 ring, or dots alone
	volume, volumeSpare, volumeOver color.RGBA // the rim while the volume is shown: set, allowed, not allowed
}

var (
	yellow = look{
		ground: rgb(255, 208, 36), rim: rgb(255, 228, 120),
		hours: rgb(228, 36, 32), minutes: rgb(26, 56, 72),
		hourHand: rgb(228, 36, 32), minuteHand: rgb(34, 66, 196), peg: rgb(86, 116, 230),
		shadow:        color.RGBA{150, 100, 0, 70},
		minuteNumbers: true,
		volume:        rgb(34, 66, 196), volumeSpare: rgb(255, 240, 180), volumeOver: rgb(214, 160, 20),
	}
	dark = look{
		ground: rgb(0, 0, 0), rim: rgb(28, 30, 36),
		hours: rgb(238, 44, 38), minutes: rgb(200, 208, 218),
		hourHand: rgb(238, 44, 38), minuteHand: rgb(48, 104, 248), peg: rgb(150, 180, 255),
		edge:          rgb(255, 255, 255),
		minuteNumbers: true,
		volume:        rgb(48, 104, 248), volumeSpare: rgb(70, 74, 86), volumeOver: rgb(28, 30, 36),
	}
	// night is the prototype's night clock: the colors of a sky after dark, low enough to sleep by.
	night = look{
		hours: rgb(93, 104, 148), minutes: rgb(60, 70, 112),
		hourHand: rgb(154, 166, 214), minuteHand: rgb(117, 132, 192), peg: rgb(40, 50, 90),
		volume: rgb(117, 132, 192), volumeSpare: rgb(44, 54, 96), volumeOver: rgb(24, 30, 60),
	}
)

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

// The clock's proportions at full size; k scales them.
const (
	minuteRing = 202.0 // the minute dots and their numbers
	hourRing   = 150.0 // the hour numbers' middles
	hourLen    = 122.0
	minuteLen  = 186.0
	handBack   = 14.0 // how far a hand reaches behind the middle
	hub        = 24.0
)

var (
	fontMu    sync.Mutex
	fontFaces = map[[2]int]font.Face{}
)

// comic is Comic Neue at px, bold or not: round, friendly figures a child knows from books.
func comic(bold bool, px int) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()
	b := 0
	if bold {
		b = 1
	}
	k := [2]int{b, px}
	if f, ok := fontFaces[k]; ok {
		return f
	}
	f, err := fonts.Face("comicneue", bold, float64(px))
	if err != nil {
		return nil
	}
	fontFaces[k] = f
	return f
}

// at is the point dist from (cx, cy) toward turns of the dial: 0 at twelve, going clockwise.
func at(cx, cy, dist, turns float64) (float64, float64) {
	a := turns * 2 * math.Pi
	return cx + dist*math.Sin(a), cy - dist*math.Cos(a)
}

// dial draws everything of the clock but the hands, about (cx, cy) at scale k.
func dial(c canvas, l look, cx, cy, k float64) {
	small := comic(true, int(18*k))
	smallLift := 0
	if small != nil {
		smallLift = capHeight(small) / 2
	}
	for i := range 60 {
		x, y := at(cx, cy, minuteRing*k, float64(i)/60)
		if i%5 != 0 || !l.minuteNumbers {
			r := 2.6 * k
			if i%5 == 0 {
				r = 4 * k // the five-minute marks stand out when they carry no number
			}
			c.disc(x, y, r, l.minutes)
			continue
		}
		n := strconv.Itoa(i)
		if i == 0 {
			n = "60"
		}
		c.centered(small, n, int(x), int(y)+smallLift, l.minutes)
	}

	big := comic(true, int(60*k))
	if big == nil {
		return
	}
	bigLift := capHeight(big) / 2
	for h := 1; h <= 12; h++ {
		x, y := at(cx, cy, hourRing*k, float64(h)/12)
		n := strconv.Itoa(h)
		if l.edge.A > 0 {
			c.edged(big, n, int(x), int(y)+bigLift, l.edge, 3*k)
		}
		c.centered(big, n, int(x), int(y)+bigLift, l.hours)
	}
}

// hands draws the hands for t about (cx, cy) at scale k, with their shadow or edge.
func hands(c canvas, l look, t time.Time, cx, cy, k float64) {
	minute := (float64(t.Minute()) + float64(t.Second())/60) / 60
	hour := (float64(t.Hour()%12) + float64(t.Minute())/60) / 12
	hourHand := func(col color.RGBA, dx, dy, grow float64) {
		arrowHand(c, hour, cx+dx, cy+dy, (hourLen+grow*1.5)*k, (handBack+grow)*k, (18+2*grow)*k, (34+2*grow)*k, (36+grow*1.5)*k, col)
	}
	minuteHand := func(col color.RGBA, dx, dy, grow float64) {
		arrowHand(c, minute, cx+dx, cy+dy, (minuteLen+grow*1.5)*k, (handBack+grow)*k, (16+2*grow)*k, (30+2*grow)*k, (36+grow*1.5)*k, col)
	}
	if l.shadow.A > 0 {
		hourHand(l.shadow, 2*k, 4*k, 0)
		minuteHand(l.shadow, 2*k, 4*k, 0)
		c.disc(cx+2*k, cy+4*k, hub*k, l.shadow)
	}
	const e = 2.5 // how wide an edge is: the hand drawn that much larger underneath it
	if l.edge.A > 0 {
		hourHand(l.edge, 0, 0, e)
	}
	hourHand(l.hourHand, 0, 0, 0)
	if l.edge.A > 0 {
		minuteHand(l.edge, 0, 0, e)
	}
	minuteHand(l.minuteHand, 0, 0, 0)
	if l.edge.A > 0 {
		c.disc(cx, cy, (hub+e)*k, l.edge)
	}
	c.disc(cx, cy, hub*k, l.minuteHand)
	c.disc(cx, cy, 6*k, l.peg)
}

// arrowHand draws a hand as the classroom clock has them: a shaft w wide from back behind the middle,
// and a head like a little house, hw wide and hl long, the roof's point at length.
func arrowHand(c canvas, turns, cx, cy, length, back, w, hw, hl float64, col color.RGBA) {
	a := turns * 2 * math.Pi
	ux, uy := math.Sin(a), -math.Cos(a) // along the hand
	vx, vy := -uy, ux                   // across it
	p := func(along, across float64) [2]float64 {
		return [2]float64{cx + ux*along + vx*across, cy + uy*along + vy*across}
	}
	base := length - hl
	c.poly([][2]float64{p(-back, -w/2), p(base+2, -w/2), p(base+2, w/2), p(-back, w/2)}, col)
	c.poly([][2]float64{p(base, -hw/2), p(base+hl*0.4, -hw/2), p(length, 0), p(base+hl*0.4, hw/2), p(base, hw/2)}, col)
}
