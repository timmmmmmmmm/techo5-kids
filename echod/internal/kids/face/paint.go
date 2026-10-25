// Package face draws TECHO5 Kids' screens on the Spot's round 480×480 panel: the learning clock by day,
// the night sky by night, and the parent menu. It only draws into an image, so it builds and previews
// anywhere (KIDS_PREVIEW, face_test.go); the clock package puts the image on the panel.
//
// The primitives are the ones TECHO5's round screen draws with (feature/display/render_spot.go):
// anti-aliased discs, rings, lines and convex shapes, and text in the fonts the image carries.
package face

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	side   = 480
	center = side / 2
	rimOut = 236 // the status ring's outer edge, as on TECHO5's Spot
	rimIn  = 222 // and its inner edge
)

// canvas is an image being drawn on.
type canvas struct{ dst *image.RGBA }

func (c canvas) blend(x, y int, col color.RGBA, cover float64) {
	if cover <= 0 || !(image.Point{x, y}.In(c.dst.Rect)) {
		return
	}
	i := c.dst.PixOffset(x, y)
	a := math.Min(cover, 1) * float64(col.A) / 255
	p := c.dst.Pix[i : i+4 : i+4]
	p[0] = uint8(float64(p[0])*(1-a) + float64(col.R)*a)
	p[1] = uint8(float64(p[1])*(1-a) + float64(col.G)*a)
	p[2] = uint8(float64(p[2])*(1-a) + float64(col.B)*a)
	p[3] = 255
}

func (c canvas) fill(col color.RGBA) {
	draw.Draw(c.dst, c.dst.Rect, image.NewUniform(col), image.Point{}, draw.Src)
}

// gradient fills top to bottom, from one color to the other.
func (c canvas) gradient(top, bottom color.RGBA) {
	b := c.dst.Rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		col := mix(top, bottom, float64(y-b.Min.Y)/float64(b.Dy()-1))
		draw.Draw(c.dst, image.Rect(b.Min.X, y, b.Max.X, y+1), image.NewUniform(col), image.Point{}, draw.Src)
	}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

func (c canvas) disc(cx, cy, r float64, col color.RGBA) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			c.blend(x, y, col, r+0.5-d)
		}
	}
}

// ring draws the ring between radii r0 and r1 about the middle, from angle a0 to a1 in radians
// clockwise from twelve o'clock.
func (c canvas) ring(r0, r1, a0, a1 float64, col color.RGBA) {
	c.ringAt(center, center, r0, r1, a0, a1, col)
}

// ringAt is ring about any point.
func (c canvas) ringAt(cx, cy, r0, r1, a0, a1 float64, col color.RGBA) {
	full := a1-a0 >= 2*math.Pi-1e-9
	for y := int(cy - r1 - 1); y <= int(cy+r1+1); y++ {
		for x := int(cx - r1 - 1); x <= int(cx+r1+1); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := math.Hypot(dx, dy)
			if d < r0-1 || d > r1+1 {
				continue
			}
			if !full {
				a := math.Atan2(dx, -dy)
				if a < 0 {
					a += 2 * math.Pi
				}
				if a < a0 || a > a1 {
					continue
				}
			}
			c.blend(x, y, col, math.Min(d-(r0-0.5), (r1+0.5)-d))
		}
	}
}

// poly fills a convex polygon with a soft edge: a pixel is covered as far as it is inside every edge,
// so a shape drawn whole has no seams where triangles would meet.
func (c canvas) poly(pts [][2]float64, col color.RGBA) {
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
	for y := int(y0) - 1; y <= int(y1)+1; y++ {
		for x := int(x0) - 1; x <= int(x1)+1; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Inf(1)
			for i := range n {
				d = math.Min(d, sign*edge(px, py, i))
			}
			c.blend(x, y, col, d+0.5)
		}
	}
}

// bar is a straight stroke w wide from one point to the other, with round ends.
func (c canvas) bar(x0, y0, x1, y1, w float64, col color.RGBA) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l > 0 {
		nx, ny := -dy/l*w/2, dx/l*w/2
		c.poly([][2]float64{{x0 + nx, y0 + ny}, {x1 + nx, y1 + ny}, {x1 - nx, y1 - ny}, {x0 - nx, y0 - ny}}, col)
	}
	c.disc(x0, y0, w/2, col)
	c.disc(x1, y1, w/2, col)
}

// centered draws s centered about x, its baseline at y.
func (c canvas) centered(face font.Face, s string, x, y int, col color.Color) {
	if face == nil {
		return
	}
	w := font.MeasureString(face, s).Round()
	d := &font.Drawer{Dst: c.dst, Src: image.NewUniform(col), Face: face, Dot: fixed.P(x-w/2, y)}
	d.DrawString(s)
}

// edged draws s centered about x in edge, spread width pixels every way: what is drawn on top of it in
// the text's own color then has an edge that wide.
func (c canvas) edged(face font.Face, s string, x, y int, edge color.RGBA, width float64) {
	for ring := 1.0; ring <= width; ring++ {
		steps := int(8 * ring)
		for i := range steps {
			a := float64(i) / float64(steps) * 2 * math.Pi
			c.centered(face, s, x+int(math.Round(ring*math.Cos(a))), y+int(math.Round(ring*math.Sin(a))), edge)
		}
	}
}

// capHeight is how far a digit reaches above the baseline, for setting a number's middle on a point.
func capHeight(face font.Face) int {
	if b, _, ok := face.GlyphBounds('8'); ok {
		return (-b.Min.Y).Ceil()
	}
	return face.Metrics().Ascent.Ceil() * 7 / 10
}
