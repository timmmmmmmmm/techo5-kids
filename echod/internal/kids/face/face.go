package face

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/font"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// Scene is everything one frame shows.
type Scene struct {
	Now   time.Time
	Style string // core.StyleYellow or core.StyleDark, for the day
	Night bool

	// Menu is the parent menu's rows while it is up, nil while it is not; Address goes under it.
	Menu    []core.Row
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

// The menu: a dark page with a list, the label on the left of the middle and its value on the right.
var (
	menuGround = rgb(16, 20, 26)
	menuText   = rgb(220, 226, 234)
	menuDim    = rgb(120, 130, 142)
	menuAccent = rgb(255, 208, 36)
)

const (
	menuTitleY = 70
	menuTop    = 112 // the first row's middle
	menuRow    = 36
)

// MenuRowAt is the row of an n-row menu under y, or -1 for none.
func MenuRowAt(y, n int) int {
	i := int(math.Floor(float64(y-menuTop+menuRow/2) / menuRow))
	if i < 0 || i >= n {
		return -1
	}
	return i
}

func drawMenu(c canvas, s Scene) {
	c.fill(menuGround)
	title, body := comic(true, 26), comic(true, 22)
	if title == nil || body == nil {
		return
	}
	c.centered(title, "Ouders", center, menuTitleY, menuAccent)
	lift := capHeight(body) / 2
	for i, row := range s.Menu {
		y := menuTop + i*menuRow
		if row.Value == "" {
			// An action: a button across the middle.
			w := float64(textWidth(body, row.Label)) + 36
			pill(c, center-w/2, center+w/2, float64(y), 30, menuAccent)
			c.centered(body, row.Label, center, y+lift, menuGround)
			continue
		}
		right(c, body, row.Label, center-10, y+lift, menuText)
		c.centered(body, row.Value, center+10+textWidth(body, row.Value)/2, y+lift, menuAccent)
	}
	if s.Address != "" {
		small := comic(false, 16)
		c.centered(small, s.Address, center, menuTop+len(s.Menu)*menuRow+6, menuDim)
	}
}

func textWidth(f font.Face, s string) int { return font.MeasureString(f, s).Round() }

// right draws s ending at x.
func right(c canvas, f font.Face, s string, x, y int, col color.Color) {
	c.centered(f, s, x-textWidth(f, s)/2, y, col)
}

// pill is a bar with round ends from x0 to x1, h tall, its middle at y.
func pill(c canvas, x0, x1, y, h float64, col color.RGBA) {
	c.poly([][2]float64{{x0, y - h/2}, {x1, y - h/2}, {x1, y + h/2}, {x0, y + h/2}}, col)
	c.disc(x0, y, h/2, col)
	c.disc(x1, y, h/2, col)
}

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
