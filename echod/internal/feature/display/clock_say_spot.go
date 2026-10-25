//go:build spot

package display

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/hardware/speaker"
	"github.com/HuskerMinion/techo5/echod/internal/layout"
	"github.com/HuskerMinion/techo5/echod/internal/lib/safe"
)

// A tap on a Kids clock says the time instead of starting a turn. The device has no text-to-speech
// (docs/voice-clips.md), so it plays clips rendered beforehand: one per minute of a twelve-hour clock,
// named for it, in a folder per screen language. <state>/time/nl/0720.wav says "Het is tien voor half
// acht". Each is 16-bit mono at VoiceRate, the rate a reply from Home Assistant arrives at.

// largestTimeClip keeps a file far past a spoken sentence from being read at all.
const largestTimeClip = 1 << 20

// sayingTime is a clip playing: taps while it plays are not queued behind it.
var sayingTime atomic.Bool

// timeClip is the clip for the minute now is in.
func timeClip(now time.Time) string {
	lang := screenLang()
	if lang == "" {
		lang = "en"
	}
	return filepath.Join(layout.StateDir, "time", lang, fmt.Sprintf("%02d%02d.wav", now.Hour()%12, now.Minute()))
}

// sayTime plays the clip for now, unless one is already playing.
func sayTime(now time.Time) {
	if !sayingTime.CompareAndSwap(false, true) {
		return
	}
	safe.Go("say the time", func() {
		defer sayingTime.Store(false)
		path := timeClip(now)
		samples, err := readTimeClip(path)
		if err != nil {
			slog.Warn("no clip to say the time", "file", path, "err", err)
			return
		}
		claim := speaker.Sound().ClaimSpeech("time", func(ctx context.Context, p *speaker.Player) error {
			p.PlayVoice(samples)
			return nil
		})
		<-claim.Done()
		if err := claim.Err(); err != nil {
			slog.Warn("saying the time failed", "err", err)
			return
		}
		slog.Info("said the time", "clip", filepath.Base(path))
	})
}

func readTimeClip(path string) ([]int16, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > largestTimeClip {
		return nil, fmt.Errorf("not a clip: %v, %d bytes", info.Mode(), info.Size())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	samples, f, err := speaker.MonoWAV(b)
	if err != nil {
		return nil, err
	}
	if f.Rate != speaker.VoiceRate {
		return nil, fmt.Errorf("recorded at %d Hz; the clips are %d", f.Rate, speaker.VoiceRate)
	}
	return samples, nil
}
