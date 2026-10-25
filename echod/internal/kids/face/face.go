package face

import (
	"image"
	"image/draw"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// Scene is everything one frame shows.
type Scene struct {
	Now   time.Time
	Style string // core.StyleYellow or core.StyleDark, for the day
	Night bool

	// Menu is the parent menu's settings while it is up, nil while it is not. Editing is the one open
	// on its own page, -1 for the ring of them all. Address is shown with SSH.
	Menu    []core.Setting
	Editing int
	Address string

	// Volume is shown on the rim while ShowVolume: the level, and the most the children may set.
	Volume, MaxVolume int
	ShowVolume        bool
}

// Draw paints s onto dst, a 480×480 image.
func Draw(dst *image.RGBA, s Scene) {
	c := canvas{dst}
	switch {
	case s.Menu != nil:
		drawMenu(c, s)
		return
	case s.Night:
		draw.Draw(dst, dst.Rect, background("night", nightSky), image.Point{}, draw.Src)
		volumeRim(c, night, s)
		hands(c, night, s.Now, nightX, nightY, nightK)
	default:
		l := yellow
		if s.Style == core.StyleDark {
			l = dark
		}
		draw.Draw(dst, dst.Rect, background(s.Style, func(c canvas) {
			c.fill(l.ground)
			dial(c, l, center, center, 1)
		}), image.Point{}, draw.Src)
		c.ring(rimIn, rimOut, 0, 2*math.Pi, l.rim)
		volumeRim(c, l, s)
		hands(c, l, s.Now, center, center, 1)
	}
}

var (
	bgMu sync.Mutex
	bgs  = map[string]*image.RGBA{}
)

// background is everything of a scene that does not move, drawn once and kept: the numbers, edged
// ones above all, are most of the work in a frame, and the clock is drawn every second.
func background(key string, paint func(canvas)) *image.RGBA {
	bgMu.Lock()
	defer bgMu.Unlock()
	if img, ok := bgs[key]; ok {
		return img
	}
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	paint(canvas{img})
	bgs[key] = img
	return img
}

// volumeRim shows the volume on the rim for a moment after it changes: a band as far round as the
// volume, then as far as the children may turn it, then the rest.
func volumeRim(c canvas, l look, s Scene) {
	if !s.ShowVolume {
		return
	}
	turn := func(v int) float64 { return 2 * math.Pi * float64(v) / float64(core.VolumeSteps) }
	c.ring(rimIn, rimOut, 0, 2*math.Pi, l.volumeOver)
	c.ring(rimIn, rimOut, 0, turn(s.MaxVolume), l.volumeSpare)
	if s.Volume > 0 {
		c.ring(rimIn, rimOut, 0, turn(s.Volume), l.volume)
	}
}

// The night: the prototype's sky, deep blue going lighter toward the bottom, with stars and a moon, and
// the clock small and dim below it.
const (
	nightK = 0.62
	nightX = center
	nightY = 268
)

var (
	nightTop    = rgb(11, 16, 38)
	nightBottom = rgb(27, 35, 80)
	moonColor   = rgb(244, 233, 193)
)

func nightSky(c canvas) {
	c.gradient(nightTop, nightBottom)
	// Stars where the clock and the moon are not, the same every night.
	r := rand.New(rand.NewPCG(7, 11))
	for n := 0; n < 34; {
		x, y := 20+r.Float64()*440, 20+r.Float64()*440
		inSky := math.Hypot(x-center, y-center) < rimIn-12
		clearOfClock := math.Hypot(x-nightX, y-nightY) > minuteRing*nightK+16
		clearOfMoon := math.Hypot(x-center, y-moonY) > 40
		if !inSky || !clearOfClock || !clearOfMoon {
			continue
		}
		star := rgb(200, 210, 255)
		star.A = uint8(90 + r.IntN(140))
		c.disc(x, y, 0.8+r.Float64()*1.4, star)
		n++
	}
	// A crescent: the moon, with the sky drawn over most of it.
	c.disc(center, moonY, 22, moonColor)
	c.disc(center+10, moonY-6, 19, mix(nightTop, nightBottom, (moonY-6)/float64(side-1)))
	dial(c, night, nightX, nightY, nightK)
}

const moonY = 74.0

// Label is for tests and logs: what a scene is, in a word.
func (s Scene) Label() string {
	switch {
	case s.Menu != nil:
		return "menu"
	case s.Night:
		return "night"
	}
	return strings.TrimPrefix(s.Style, "kids")
}
