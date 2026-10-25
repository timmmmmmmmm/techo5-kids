//go:build linux

package clock

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

// Saying the time. The device has no text-to-speech (TECHO5's docs/voice-clips.md), so it plays clips
// rendered beforehand: one sentence per minute of a twelve-hour clock, named for it, in a folder per
// language. <state>/time/nl/0720.wav says "Het is tien voor half acht." Each is 16-bit mono at
// VoiceRate, the rate a reply from Home Assistant arrives at, which the speaker already knows how to
// play. The fork's README says how to make a set.

// largestClip keeps a file far past a spoken sentence from being read at all.
const largestClip = 1 << 20

// saying is a clip playing: taps while it plays are not queued behind it.
var saying atomic.Bool

// clipFor is the clip for the minute t is in.
func clipFor(t time.Time, lang string) string {
	return filepath.Join(layout.StateDir, "time", lang, fmt.Sprintf("%02d%02d.wav", t.Hour()%12, t.Minute()))
}

// sayTime plays the clip for t, unless one is already playing.
func sayTime(t time.Time, lang string) {
	if !saying.CompareAndSwap(false, true) {
		return
	}
	safe.Go("say the time", func() {
		defer saying.Store(false)
		path := clipFor(t, lang)
		samples, err := readClip(path)
		if err != nil {
			slog.Warn("kids: no clip to say the time", "file", path, "err", err)
			return
		}
		claim := speaker.Sound().ClaimSpeech("time", func(ctx context.Context, p *speaker.Player) error {
			p.PlayVoice(samples)
			return nil
		})
		<-claim.Done()
		if err := claim.Err(); err != nil {
			slog.Warn("kids: saying the time failed", "err", err)
			return
		}
		slog.Info("kids: said the time", "clip", filepath.Base(path))
	})
}

func readClip(path string) ([]int16, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > largestClip {
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
