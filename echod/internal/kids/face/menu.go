package face

import (
	"image/color"
	"math"

	"golang.org/x/image/font"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// The parent menu, in the shapes TECHO5's round screen already uses: its round keys (the PIN pad) and
// its value page (the volume's jog wheel), with nothing that needs turning or a precise touch. The ring
// is the settings as large round buttons, each showing its value, with Klaar in the middle; a setting's
// own page has its value large in the middle with an arc for how far along it is, − and + either side,
// or two buttons for a choice of two, and Klaar below.
//
// It wears the clock's own colors, in the look the clock is set to: the red of the hours for values,
// the blue of the minute hand for what a tap does, on the yellow of the classroom clock or on black.

// theme is the menu's colors in one of the clock's looks.
type theme struct {
	ground             color.RGBA
	key, keyText       color.RGBA // a button, and a name on one
	icon, value, label color.RGBA // on a setting's button: its icon, its value, its name
	done, doneText     color.RGBA // Klaar
	title, hint        color.RGBA // a page's heading and the line under it
	track, arc         color.RGBA // the arc: all of it, and as far as the value goes
	sign, signEnd      color.RGBA // − and +, and either when it can go no further
	chosen, chosenText color.RGBA // the choice that is made
}

var (
	yellowMenu = theme{
		ground: rgb(255, 208, 36),
		key:    rgb(255, 248, 230), keyText: rgb(26, 56, 72),
		icon: rgb(34, 66, 196), value: rgb(228, 36, 32), label: rgb(26, 56, 72),
		done: rgb(34, 66, 196), doneText: rgb(255, 255, 255),
		title: rgb(26, 56, 72), hint: rgb(120, 88, 0),
		track: rgb(255, 230, 130), arc: rgb(34, 66, 196),
		sign: rgb(34, 66, 196), signEnd: rgb(190, 196, 220),
		chosen: rgb(34, 66, 196), chosenText: rgb(255, 255, 255),
	}
	darkMenu = theme{
		ground: rgb(0, 0, 0),
		key:    rgb(28, 32, 42), keyText: rgb(236, 240, 244),
		icon: rgb(72, 134, 255), value: rgb(255, 92, 76), label: rgb(150, 158, 172),
		done: rgb(255, 208, 36), doneText: rgb(20, 24, 30),
		title: rgb(236, 240, 244), hint: rgb(140, 148, 160),
		track: rgb(40, 44, 56), arc: rgb(255, 208, 36),
		sign: rgb(236, 240, 244), signEnd: rgb(84, 90, 104),
		chosen: rgb(72, 134, 255), chosenText: rgb(255, 255, 255),
	}
)

func themeFor(style string) theme {
	if style == core.StyleDark {
		return darkMenu
	}
	return yellowMenu
}

// The ring: where its buttons are and how big. Every button is at least as wide as a fingertip
// (112 px is about 15 mm on the Spot's panel), with room between them.
const (
	ringR   = 156.0
	keyR    = 56.0
	doneR   = 60.0
	pageY   = 240.0 // the middle line of a setting's page
	stepX   = 136.0 // how far − and + are from the middle
	stepR   = 50.0
	pickX   = 92.0 // how far a choice's button is from the middle
	pickR   = 72.0
	pillY   = 392.0 // Klaar on a setting's page
	pillW   = 150.0
	pillH   = 64.0
	tapSlop = 10.0 // a tap this far outside a button still counts
)

// TargetKind is what a tap on the menu landed on.
type TargetKind int

const (
	TargetNone  TargetKind = iota
	TargetDone             // Klaar: close the menu from the ring, back to the ring from a page
	TargetOpen             // a setting on the ring (Index)
	TargetMinus            // − on a setting's page
	TargetPlus             // + on a setting's page
	TargetPick             // one of a choice's two buttons (Index)
)

func (k TargetKind) String() string {
	return [...]string{"nothing", "done", "open", "minus", "plus", "pick"}[k]
}

// Target is a tap's meaning.
type Target struct {
	Kind  TargetKind
	Index int
}

// keyAt is the middle of setting i of n on the ring, the first at the top and the rest clockwise.
func keyAt(i, n int) (float64, float64) {
	return at(center, center, ringR, float64(i)/float64(n))
}

// MenuTarget is what a tap at x, y means on the menu as drawn: the ring when editing is -1, setting
// editing's page otherwise.
func MenuTarget(x, y int, menu []core.Setting, editing int) Target {
	fx, fy := float64(x), float64(y)
	near := func(cx, cy, r float64) bool { return math.Hypot(fx-cx, fy-cy) <= r+tapSlop }
	if editing < 0 || editing >= len(menu) {
		if near(center, center, doneR) {
			return Target{Kind: TargetDone}
		}
		for i := range menu {
			if x, y := keyAt(i, len(menu)); near(x, y, keyR) {
				return Target{Kind: TargetOpen, Index: i}
			}
		}
		return Target{}
	}
	if math.Abs(fy-pillY) <= pillH/2+tapSlop && math.Abs(fx-center) <= pillW/2+pillH/2+tapSlop {
		return Target{Kind: TargetDone}
	}
	switch menu[editing].Kind {
	case core.Steps:
		if near(center-stepX, pageY, stepR) {
			return Target{Kind: TargetMinus}
		}
		if near(center+stepX, pageY, stepR) {
			return Target{Kind: TargetPlus}
		}
	case core.Choice:
		for i, dx := range []float64{-pickX, pickX} {
			if near(center+dx, pageY, pickR) {
				return Target{Kind: TargetPick, Index: i}
			}
		}
	}
	return Target{}
}

func drawMenu(c canvas, s Scene) {
	t := themeFor(s.Style)
	c.fill(t.ground)
	if s.Editing >= 0 && s.Editing < len(s.Menu) {
		drawPage(c, t, s.Menu[s.Editing], s.Address)
		return
	}
	label, done := comic(false, 14), comic(true, 24)
	if label == nil || done == nil {
		return
	}
	// Each button: what it is as an icon, its value large, and its name under that.
	for i, m := range s.Menu {
		x, y := keyAt(i, len(s.Menu))
		c.disc(x, y, keyR, t.key)
		icon(c, m.ID, x, y-26, t.icon, t.key)
		f := fitted(m.Value, 24, int(keyR*2-20))
		c.centered(f, m.Value, int(x), int(y)+4+capHeight(f)/2, t.value)
		c.centered(label, m.Short, int(x), int(y)+33, t.label)
	}
	c.disc(center, center, doneR, t.done)
	c.centered(done, "Klaar", center, center+capHeight(done)/2, t.doneText)
}

// drawPage is one setting on its own.
func drawPage(c canvas, t theme, m core.Setting, address string) {
	title, hint := comic(true, 26), comic(false, 16)
	if title == nil || hint == nil {
		return
	}
	c.centered(fitted(m.Label, 26, 330), m.Label, center, 98, t.title)
	c.centered(hint, m.Hint, center, 128, t.hint)

	switch m.Kind {
	case core.Steps:
		// How far along, on an arc open at the bottom like TECHO5's jog wheel.
		const from, sweep = 225.0, 270.0
		arc(c, 200, 212, from, sweep, t.track)
		if m.Index >= 0 && m.Steps > 1 {
			arc(c, 200, 212, from, sweep*float64(m.Index)/float64(m.Steps-1), t.arc)
		}
		f := fitted(m.Value, 64, int(2*(stepX-stepR)-24))
		c.centered(f, m.Value, center, int(pageY)+capHeight(f)/2, t.value)
		first, last := m.Index == 0, m.Index == m.Steps-1
		stepKey(c, t, center-stepX, false, first)
		stepKey(c, t, center+stepX, true, last)
	case core.Choice:
		if m.ID == core.RowStyle {
			styleChoices(c, t, m)
			break
		}
		for i, dx := range []float64{-pickX, pickX} {
			face, text := t.key, t.keyText
			if i == m.Index {
				face, text = t.chosen, t.chosenText
			}
			c.disc(center+dx, pageY, pickR, face)
			f := fitted(m.Names[i], 28, int(pickR*2-20))
			c.centered(f, m.Names[i], int(center+dx), int(pageY)+capHeight(f)/2, text)
		}
		if m.ID == core.RowSSH && address != "" {
			c.centered(hint, address, center, int(pageY+pickR)+30, t.hint)
		}
	}

	pill(c, center-pillW/2, center+pillW/2, pillY, pillH, t.done)
	done := comic(true, 24)
	c.centered(done, "Klaar", center, int(pillY)+capHeight(done)/2, t.doneText)
}

// styleChoices shows the clock's two looks as small clocks in their own colors, the one chosen ringed,
// with its name under each: what is chosen is seen rather than read.
func styleChoices(c canvas, t theme, m core.Setting) {
	const r = pickR - 10
	name := comic(true, 20)
	for i, l := range []look{yellow, dark} {
		x := center + []float64{-pickX, pickX}[i]
		if i == m.Index {
			c.ringAt(x, pageY, r+4, r+12, 0, 2*math.Pi, t.chosen)
		} else {
			c.ringAt(x, pageY, r+4, r+6, 0, 2*math.Pi, t.track)
		}
		c.disc(x, pageY, r, l.ground)
		c.bar(x, pageY, x-26, pageY-15, 9, l.hourHand)   // ten past ten, the way clocks are shown
		c.bar(x, pageY, x+30, pageY-30, 7, l.minuteHand) //
		c.disc(x, pageY, 8, l.minuteHand)
		c.disc(x, pageY, 3, l.peg)
		col := t.title
		if i == m.Index {
			col = t.chosen
		}
		c.centered(name, m.Names[i], int(x), int(pageY+r)+40, col)
	}
}

// stepKey is − or + as a round key, its sign drawn as bars rather than a glyph so it is the same
// weight in any font; faded at the end it cannot go past.
func stepKey(c canvas, t theme, x float64, plus, end bool) {
	sign := t.sign
	if end {
		sign = t.signEnd
	}
	c.disc(x, pageY, stepR, t.key)
	const half, w = 17.0, 7.0
	c.poly([][2]float64{{x - half, pageY - w/2}, {x + half, pageY - w/2}, {x + half, pageY + w/2}, {x - half, pageY + w/2}}, sign)
	if plus {
		c.poly([][2]float64{{x - w/2, pageY - half}, {x + w/2, pageY - half}, {x + w/2, pageY + half}, {x - w/2, pageY + half}}, sign)
	}
}

// arc is a band between radii r0 and r1 from angle from, in degrees clockwise from twelve, round
// sweep degrees, passing twelve o'clock if it has to.
func arc(c canvas, r0, r1, from, sweep float64, col color.RGBA) {
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	start := math.Mod(from, 360)
	if end := start + sweep; end > 360 {
		c.ring(r0, r1, rad(start), 2*math.Pi, col)
		c.ring(r0, r1, 0, rad(end-360), col)
		return
	}
	if sweep > 0 {
		c.ring(r0, r1, rad(start), rad(start+sweep), col)
	}
}

// fitted is Comic Neue bold at px, smaller if that is what it takes for s to fit w.
func fitted(s string, px, w int) font.Face {
	f := comic(true, px)
	for f != nil && px > 12 && textWidth(f, s) > w {
		px -= 2
		f = comic(true, px)
	}
	return f
}

func textWidth(f font.Face, s string) int { return font.MeasureString(f, s).Round() }

// pill is a bar with round ends from x0 to x1, h tall, its middle at y.
func pill(c canvas, x0, x1, y, h float64, col color.RGBA) {
	c.poly([][2]float64{{x0, y - h/2}, {x1, y - h/2}, {x1, y + h/2}, {x0, y + h/2}}, col)
	c.disc(x0, y, h/2, col)
	c.disc(x1, y, h/2, col)
}
