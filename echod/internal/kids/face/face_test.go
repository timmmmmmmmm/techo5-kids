package face

import (
	"image"
	"image/png"
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
	rows := core.MenuRows(s, core.Status{SSH: true, Address: "192.0.2.10"})
	scenes := map[string]Scene{
		"day":         {Now: now, Style: core.StyleYellow},
		"day-volume":  {Now: now, Style: core.StyleYellow, ShowVolume: true, Volume: 12, MaxVolume: 20},
		"dark":        {Now: now, Style: core.StyleDark},
		"dark-volume": {Now: now, Style: core.StyleDark, ShowVolume: true, Volume: 12, MaxVolume: 20},
		"night":       {Now: time.Date(2026, 10, 25, 21, 40, 0, 0, time.Local), Night: true},
		"menu":        {Now: now, Menu: rows, Address: "192.0.2.10"},
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

func TestATapOnAMenuRowFindsIt(t *testing.T) {
	n := len(core.MenuRows(core.Defaults(), core.Status{}))
	for i := range n {
		y := menuTop + i*menuRow
		for _, dy := range []int{-menuRow/2 + 1, 0, menuRow/2 - 1} {
			if got := MenuRowAt(y+dy, n); got != i {
				t.Errorf("y %d: row %d, want %d", y+dy, got, i)
			}
		}
	}
	if MenuRowAt(menuTop-menuRow, n) != -1 || MenuRowAt(menuTop+n*menuRow, n) != -1 {
		t.Error("a tap above or below the list hits a row")
	}
}
