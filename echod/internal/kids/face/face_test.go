package face

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// TestEveryScreenDraws draws each screen, and with KIDS_PREVIEW set writes them there as PNGs to look
// at: KIDS_PREVIEW=/tmp/kids go test ./internal/kids/face
func TestEveryScreenDraws(t *testing.T) {
	now := time.Date(2026, 10, 25, 14, 7, 38, 0, time.Local)
	s := core.Defaults()
	menu := core.MenuSettings(s, core.Status{SSH: true, Address: "192.0.2.10"})
	scenes := map[string]Scene{
		"day":             {Now: now, Style: core.StyleYellow},
		"day-play":        {Now: now, Style: core.StyleYellow, Playing: true},
		"day-volume":      {Now: now, Style: core.StyleYellow, ShowVolume: true, Volume: 12, MaxVolume: 20},
		"dark":            {Now: now, Style: core.StyleDark},
		"dark-play":       {Now: now, Style: core.StyleDark, Playing: true},
		"dark-volume":     {Now: now, Style: core.StyleDark, ShowVolume: true, Volume: 12, MaxVolume: 20},
		"night":           {Now: time.Date(2026, 10, 25, 21, 40, 0, 0, time.Local), Night: true},
		"night-play":      {Now: time.Date(2026, 10, 25, 21, 40, 0, 0, time.Local), Night: true, Playing: true},
		"menu":            {Now: now, Menu: menu, Editing: -1},
		"menu-volume":     {Now: now, Menu: menu, Editing: 0},
		"menu-bed":        {Now: now, Menu: menu, Editing: 4},
		"menu-style":      {Now: now, Menu: menu, Editing: 3},
		"menu-ssh":        {Now: now, Menu: menu, Editing: 6, Address: "192.0.2.10"},
		"menu-dark":       {Now: now, Style: core.StyleDark, Menu: menu, Editing: -1},
		"menu-dark-bed":   {Now: now, Style: core.StyleDark, Menu: menu, Editing: 4},
		"menu-dark-style": {Now: now, Style: core.StyleDark, Menu: core.MenuSettings(core.Settings{Style: core.StyleDark}.Normalized(), core.Status{}), Editing: 3},
	}
	dir := os.Getenv("KIDS_PREVIEW")
	for name, sc := range scenes {
		img := image.NewRGBA(image.Rect(0, 0, side, side))
		Draw(img, sc)
		if n := colors(img); n < 10 {
			t.Errorf("%s: drew next to nothing (%d colors)", name, n)
		}
		if dir == "" {
			continue
		}
		f, err := os.Create(filepath.Join(dir, "kids-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

// colors counts the different colors in img.
func colors(img *image.RGBA) int {
	seen := map[[4]uint8]bool{}
	for i := 0; i < len(img.Pix); i += 4 {
		seen[[4]uint8(img.Pix[i:i+4])] = true
	}
	return len(seen)
}

func TestEveryButtonIsWhereItIsDrawn(t *testing.T) {
	menu := core.MenuSettings(core.Defaults(), core.Status{})
	if got := MenuTarget(center, center, menu, -1); got.Kind != TargetDone {
		t.Errorf("the middle of the ring is %v, want Klaar", got)
	}
	for i := range menu {
		x, y := keyAt(i, len(menu))
		if got := MenuTarget(int(x), int(y), menu, -1); got != (Target{TargetOpen, i}) {
			t.Errorf("setting %d's button is %v", i, got)
		}
	}
	if got := MenuTarget(5, 5, menu, -1); got.Kind != TargetNone {
		t.Errorf("a corner is %v", got)
	}
	steps, choice := 0, 3 // max volume, the clock's style
	for _, c := range []struct {
		x, y    float64
		editing int
		want    Target
	}{
		{center - stepX, pageY, steps, Target{Kind: TargetMinus}},
		{center + stepX, pageY, steps, Target{Kind: TargetPlus}},
		{center, pillY, steps, Target{Kind: TargetDone}},
		{center, pageY, steps, Target{}},
		{center - pickX, pageY, choice, Target{TargetPick, 0}},
		{center + pickX, pageY, choice, Target{TargetPick, 1}},
		{center, pillY, choice, Target{Kind: TargetDone}},
	} {
		if got := MenuTarget(int(c.x), int(c.y), menu, c.editing); got != c.want {
			t.Errorf("page %d at %v,%v: %v, want %v", c.editing, c.x, c.y, got, c.want)
		}
	}
}

// TestNoTwoButtonsTouch keeps the ring's buttons apart by more than the slop a tap is allowed.
func TestNoTwoButtonsTouch(t *testing.T) {
	n := len(core.MenuSettings(core.Defaults(), core.Status{}))
	x0, y0 := keyAt(0, n)
	x1, y1 := keyAt(1, n)
	if gap := math.Hypot(x1-x0, y1-y0) - 2*keyR; gap < 2*tapSlop {
		t.Errorf("neighbours %0.f px apart, want at least %0.f", gap, 2*tapSlop)
	}
	if gap := ringR - keyR - doneR; gap < 2*tapSlop {
		t.Errorf("the ring is %0.f px from Klaar, want at least %0.f", gap, 2*tapSlop)
	}
}
