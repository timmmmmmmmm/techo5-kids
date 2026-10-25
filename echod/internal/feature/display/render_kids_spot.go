//go:build spot

package display

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"sync"

	"golang.org/x/image/font"
)

// The Kids styles: the classroom learning clock, for a child learning to read one. The hours are big red
// numbers and the short hour hand is red too; the minutes are a ring of dots numbered every five, out to
// 60, and the long minute hand is blue. Both hands end in an arrow's head. Kids is on the classroom
// clock's yellow; Kids Dark is on black, like the Spot's bezel, with a white edge round the numbers and
// the hands so the red and blue stand out. Each keeps its colors and Comic Neue whatever the skin and the
// screen's font: a clock a child learns from looks the same every day. The rim is still the status ring
// (rim), in the look's own color while it has nothing to say.

// kidsLook is one of the two.
type kidsLook struct {
	ground, rim, hours, minutes, hourHand, minuteHand, peg color.RGBA
	edge                                                   color.RGBA // round the numbers and hands; none if A is 0
	shadow                                                 color.RGBA // under the hands; none if A is 0
}

var kidsLooks = map[string]kidsLook{
	styleKids: {
		ground: color.RGBA{255, 208, 36, 255}, rim: color.RGBA{255, 228, 120, 255},
		hours: color.RGBA{228, 36, 32, 255}, minutes: color.RGBA{26, 56, 72, 255},
		hourHand: color.RGBA{228, 36, 32, 255}, minuteHand: color.RGBA{34, 66, 196, 255}, peg: color.RGBA{86, 116, 230, 255},
		shadow: color.RGBA{150, 100, 0, 70},
	},
	styleKidsDark: {
		ground: color.RGBA{0, 0, 0, 255}, rim: color.RGBA{28, 30, 36, 255},
		hours: color.RGBA{238, 44, 38, 255}, minutes: color.RGBA{200, 208, 218, 255},
		hourHand: color.RGBA{238, 44, 38, 255}, minuteHand: color.RGBA{48, 104, 248, 255}, peg: color.RGBA{150, 180, 255, 255},
		edge: color.RGBA{255, 255, 255, 255},
	},
}

// isKidsStyle is whether style is one of the Kids styles, whose tap says the time (clock_say_spot.go).
func isKidsStyle(style string) bool {
	_, ok := kidsLooks[style]
	return ok
}

const (
	kidsMinuteRing = 202.0 // the minute dots and their numbers
	kidsHourRing   = 150.0 // the hour numbers' middles
)

var (
	kidsMu sync.Mutex
	// kidsFonts are the styles' faces by size, Comic Neue bold.
	kidsFonts = map[int]font.Face{}
	// kidsDials are each look's dial drawn once: everything but the hands. The face is drawn again every
	// frame, and the numbers - edged ones above all - are most of the work in it.
	kidsDials = map[string]*image.RGBA{}
	// kidsMask lets a dial onto the face everywhere but the rim, which is drawn before the face.
	kidsMask *image.Alpha
)

func kidsFont(px int) font.Face {
	if f, ok := kidsFonts[px]; ok {
		return f
	}
	f := fontFace("comicneue", true, float64(px))
	kidsFonts[px] = f
	return f
}

// kidsAt is the point dist from the middle toward turns of the dial, 0 at twelve and going clockwise.
func kidsAt(dist, turns float64) (float64, float64) {
	a := turns * 2 * math.Pi
	return center + dist*math.Sin(a), center - dist*math.Cos(a)
}

func (r *roundRenderer) kidsClockFace(s roundScene, style string) {
	look := kidsLooks[style]
	kidsMu.Lock()
	dial := kidsDial(style, look)
	mask := kidsRimMask()
	kidsMu.Unlock()
	draw.DrawMask(r.dst, r.dst.Rect, dial, image.Point{}, mask, image.Point{}, draw.Over)

	now := s.now
	minute := (float64(now.Minute()) + float64(now.Second())/60) / 60
	hour := (float64(now.Hour()%12) + float64(now.Minute())/60) / 12
	const (
		hourLen, minuteLen = 122, 186
		back               = 14
		hub                = 24
	)
	if look.shadow.A > 0 {
		const dx, dy = 2, 4
		r.arrowHand(hour, hourLen, back, 18, 34, 36, look.shadow, dx, dy)
		r.arrowHand(minute, minuteLen, back, 16, 30, 36, look.shadow, dx, dy)
		r.discAt(center+dx, center+dy, hub, look.shadow)
	}
	if look.edge.A > 0 {
		// The edge is the hand drawn a little larger underneath it.
		const e = 2.5
		r.arrowHand(hour, hourLen+e*1.5, back+e, 18+2*e, 34+2*e, 36+e*1.5, look.edge, 0, 0)
		r.arrowHand(hour, hourLen, back, 18, 34, 36, look.hourHand, 0, 0)
		r.arrowHand(minute, minuteLen+e*1.5, back+e, 16+2*e, 30+2*e, 36+e*1.5, look.edge, 0, 0)
		r.arrowHand(minute, minuteLen, back, 16, 30, 36, look.minuteHand, 0, 0)
		r.discAt(center, center, hub+e, look.edge)
	} else {
		r.arrowHand(hour, hourLen, back, 18, 34, 36, look.hourHand, 0, 0)
		r.arrowHand(minute, minuteLen, back, 16, 30, 36, look.minuteHand, 0, 0)
	}
	r.discAt(center, center, hub, look.minuteHand)
	r.discAt(center, center, 6, look.peg)
}

// kidsDial is the look's dial, drawn the first time it is asked for. Called with kidsMu held.
func kidsDial(style string, look kidsLook) *image.RGBA {
	if d, ok := kidsDials[style]; ok {
		return d
	}
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	d := &roundRenderer{paint: paint{dst: img, w: side, h: side, round: true}}
	draw.Draw(img, img.Rect, image.NewUniform(look.ground), image.Point{}, draw.Src)

	small := kidsFont(18)
	smallLift := capHeightOf(small) / 2
	for i := range 60 {
		x, y := kidsAt(kidsMinuteRing, float64(i)/60)
		if i%5 != 0 {
			d.discAt(x, y, 2.6, look.minutes)
			continue
		}
		n := strconv.Itoa(i)
		if i == 0 {
			n = "60"
		}
		d.centered2(small, n, int(x), int(y)+smallLift, look.minutes)
	}

	big := kidsFont(60)
	bigLift := capHeightOf(big) / 2
	for h := 1; h <= 12; h++ {
		x, y := kidsAt(kidsHourRing, float64(h)/12)
		n := strconv.Itoa(h)
		if look.edge.A > 0 {
			d.edgedText(big, n, int(x), int(y)+bigLift, look.edge, 3)
		}
		d.centered2(big, n, int(x), int(y)+bigLift, look.hours)
	}
	kidsDials[style] = img
	return img
}

// edgedText draws s centered about x in edge, spread width pixels every way: what goes on top of it in
// the text's own color then has an edge that wide.
func (r *roundRenderer) edgedText(face font.Face, s string, x, baseline int, edge color.RGBA, width float64) {
	for ring := 1.0; ring <= width; ring++ {
		steps := int(8 * ring)
		for i := range steps {
			a := float64(i) / float64(steps) * 2 * math.Pi
			r.centered2(face, s, x+int(math.Round(ring*math.Cos(a))), baseline+int(math.Round(ring*math.Sin(a))), edge)
		}
	}
}

// kidsRimMask is opaque inside the rim and outside it, and clear over it. Called with kidsMu held.
func kidsRimMask() *image.Alpha {
	if kidsMask != nil {
		return kidsMask
	}
	m := image.NewAlpha(image.Rect(0, 0, side, side))
	for y := range side {
		for x := range side {
			d := math.Hypot(float64(x)+0.5-center, float64(y)+0.5-center)
			cover := math.Max(math.Min(rimIn-d, 1), math.Min(d-rimOut, 1))
			m.Pix[m.PixOffset(x, y)] = uint8(255 * math.Max(cover, 0))
		}
	}
	kidsMask = m
	return m
}

// arrowHand draws a hand as the classroom clock has them: a shaft w wide from back behind the middle,
// and a head like a little house, hw wide and hl long, its roof's point at length; moved by dx, dy for
// its shadow.
func (r *roundRenderer) arrowHand(turns, length, back, w, hw, hl float64, c color.RGBA, dx, dy float64) {
	a := turns * 2 * math.Pi
	ux, uy := math.Sin(a), -math.Cos(a) // along the hand
	vx, vy := -uy, ux                   // across it
	p := func(along, across float64) [2]float64 {
		return [2]float64{center + dx + ux*along + vx*across, center + dy + uy*along + vy*across}
	}
	base := length - hl
	r.convexFill([][2]float64{p(-back, -w/2), p(base+2, -w/2), p(base+2, w/2), p(-back, w/2)}, c)
	r.convexFill([][2]float64{p(base, -hw/2), p(base+hl*0.4, -hw/2), p(length, 0), p(base+hl*0.4, hw/2), p(base, hw/2)}, c)
}

// convexFill fills a convex polygon with a soft edge, as triangle does for three corners: a pixel is
// covered as far as it is inside every edge, so a shape drawn whole has no seams where triangles meet.
func (r *roundRenderer) convexFill(pts [][2]float64, c color.RGBA) {
	n := len(pts)
	var mx, my float64
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, q := range pts {
		mx, my = mx+q[0]/float64(n), my+q[1]/float64(n)
		x0, y0, x1, y1 = math.Min(x0, q[0]), math.Min(y0, q[1]), math.Max(x1, q[0]), math.Max(y1, q[1])
	}
	edge := func(px, py float64, i int) float64 {
		a, b := pts[i], pts[(i+1)%n]
		ex, ey := b[0]-a[0], b[1]-a[1]
		return ((px-a[0])*ey - (py-a[1])*ex) / math.Hypot(ex, ey)
	}
	sign := 1.0
	if edge(mx, my, 0) < 0 {
		sign = -1
	}
	b := r.dst.Rect
	for y := max(int(y0)-1, b.Min.Y); y <= min(int(y1)+1, b.Max.Y-1); y++ {
		for x := max(int(x0)-1, b.Min.X); x <= min(int(x1)+1, b.Max.X-1); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Inf(1)
			for i := range n {
				d = math.Min(d, sign*edge(px, py, i))
			}
			r.blend(x, y, c, math.Min(d+0.5, 1))
		}
	}
}
