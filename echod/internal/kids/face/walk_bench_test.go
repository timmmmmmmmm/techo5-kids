package face

import (
	"image"
	"image/draw"
	"testing"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

// The benchmarks are for the walk back to the real time: on the Spot a whole frame costs a tenth of
// a second and more, and these say where it goes.

func benchScene() (Scene, *image.RGBA) {
	s := Scene{Now: time.Date(2026, 10, 25, 12, 13, 0, 0, time.Local), Style: core.StyleYellow, Playing: true}
	return s, image.NewRGBA(image.Rect(0, 0, side, side))
}

func BenchmarkTheBlit(b *testing.B) {
	_, img := benchScene()
	bg := background(core.StyleYellow, func(c canvas) {
		c.fill(yellow.ground)
		dial(c, yellow, center, center, 1)
	})
	b.ResetTimer()
	for range b.N {
		draw.Draw(img, img.Rect, bg, image.Point{}, draw.Src)
	}
}

func BenchmarkTheHands(b *testing.B) {
	s, img := benchScene()
	c := canvas{img}
	b.ResetTimer()
	for range b.N {
		hands(c, yellow, s.Now, center, center, 1)
	}
}

func BenchmarkThePlayRim(b *testing.B) {
	_, img := benchScene()
	c := canvas{img}
	b.ResetTimer()
	for range b.N {
		playRim(c, yellow)
	}
}

func BenchmarkTheWholeWalkFrame(b *testing.B) {
	s, img := benchScene()
	b.ResetTimer()
	for range b.N {
		s.Now = s.Now.Add(4 * time.Second)
		Draw(img, s)
	}
}
