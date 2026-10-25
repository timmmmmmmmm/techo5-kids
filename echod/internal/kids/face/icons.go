package face

import (
	"image/color"
	"math"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// The parent menu's icons, one per setting, drawn with the same strokes as everything else so they are
// the same weight in any font: about 26 px across, about (x, y). bg is what they are drawn on, for the
// parts that are cut out of a shape (the moon's crescent, the sun below the horizon).

func icon(c canvas, id string, x, y float64, col, bg color.RGBA) {
	switch id {
	case core.RowMaxVolume:
		speakerIcon(c, x, y, col)
	case core.RowBrightness:
		sunIcon(c, x, y, col)
	case core.RowNightLight:
		moonIcon(c, x, y, col, bg)
	case core.RowStyle:
		clockIcon(c, x, y, col)
	case core.RowBedtime:
		bedIcon(c, x, y, col)
	case core.RowWaketime:
		sunriseIcon(c, x, y, col, bg)
	case core.RowSSH:
		f := comic(true, 17)
		c.centered(f, ">_", int(x), int(y)+capHeight(f)/2, col)
	}
}

// deg is degrees clockwise from twelve o'clock, in the radians ring takes.
func deg(d float64) float64 { return d * math.Pi / 180 }

func speakerIcon(c canvas, x, y float64, col color.RGBA) {
	x -= 4
	c.poly([][2]float64{{x - 9, y - 4}, {x - 3, y - 4}, {x - 3, y + 4}, {x - 9, y + 4}}, col)
	c.poly([][2]float64{{x - 3, y - 4}, {x + 4, y - 10}, {x + 4, y + 10}, {x - 3, y + 4}}, col)
	c.ringAt(x+4, y, 6, 8.4, deg(50), deg(130), col)
	c.ringAt(x+4, y, 11, 13.4, deg(45), deg(135), col)
}

func sunIcon(c canvas, x, y float64, col color.RGBA) {
	c.disc(x, y, 5.5, col)
	for i := range 8 {
		a := float64(i) * math.Pi / 4
		s, k := math.Sin(a), -math.Cos(a)
		c.bar(x+8.5*s, y+8.5*k, x+12*s, y+12*k, 2.4, col)
	}
}

func moonIcon(c canvas, x, y float64, col, bg color.RGBA) {
	c.disc(x, y, 10, col)
	c.disc(x+5.5, y-3.5, 8.5, bg)
}

func clockIcon(c canvas, x, y float64, col color.RGBA) {
	c.ringAt(x, y, 8, 10.5, 0, 2*math.Pi, col)
	c.bar(x, y, x, y-6, 2.4, col)
	c.bar(x, y, x+4.5, y+1, 2.4, col)
}

func bedIcon(c canvas, x, y float64, col color.RGBA) {
	c.bar(x-12, y-8, x-12, y+8, 3, col)                                                   // the headboard
	c.poly([][2]float64{{x - 12, y}, {x + 12, y}, {x + 12, y + 5}, {x - 12, y + 5}}, col) // the bed
	c.bar(x+12, y+2, x+12, y+8, 3, col)                                                   // the foot
	c.disc(x-6, y-3.5, 3.2, col)                                                          // the pillow
	c.poly([][2]float64{{x - 2, y - 5}, {x + 11, y - 5}, {x + 11, y}, {x - 2, y}}, col)   // the covers
}

func sunriseIcon(c canvas, x, y float64, col, bg color.RGBA) {
	h := y + 5 // the horizon
	c.disc(x, h, 7, col)
	c.poly([][2]float64{{x - 15, h}, {x + 15, h}, {x + 15, h + 10}, {x - 15, h + 10}}, bg)
	c.bar(x-13, h+0.5, x+13, h+0.5, 2.4, col)
	for _, d := range []float64{-60, -30, 0, 30, 60} {
		s, k := math.Sin(deg(d)), -math.Cos(deg(d))
		c.bar(x+10*s, h+10*k, x+13.5*s, h+13.5*k, 2.4, col)
	}
}
