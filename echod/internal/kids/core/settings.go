// Package core is TECHO5 Kids without hardware: the settings a parent makes, when the night is, the
// parent menu and the button chord that opens it. It builds and tests anywhere, a Mac included, so the
// rules of the clock can be worked on without a Spot at hand.
package core

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/HuskerMinion/techo5/echod/internal/config"
	"github.com/HuskerMinion/techo5/echod/internal/layout"
)

// The clock's two looks: the classroom clock's yellow, and black like the Spot's bezel.
const (
	StyleYellow = "kids"
	StyleDark   = "kidsdark"
)

// VolumeSteps is the top of the volume scale, the speaker's own.
const VolumeSteps = config.VolumeSteps

// SettingsPath is where the settings are kept: beside TECHO5's own, on userdata, so an update of the
// slot leaves them where they are.
var SettingsPath = filepath.Join(layout.StateDir, "kids.json")

// Settings are what a parent sets, and the one thing the children set: the volume.
type Settings struct {
	Style           string `json:"style"`
	Volume          int    `json:"volume"`           // 0..VolumeSteps, as the children last left it
	MaxVolume       int    `json:"max_volume"`       // the most they can turn it up to
	Brightness      int    `json:"brightness"`       // percent, by day
	NightBrightness int    `json:"night_brightness"` // percent, at night
	Bedtime         string `json:"bedtime"`          // HH:MM: the night face comes
	Waketime        string `json:"waketime"`         // HH:MM: it goes again
	Language        string `json:"language"`         // the folder the spoken time is in
}

// Defaults are a calm clock for a young child: not loud, not bright, asleep from seven to seven.
func Defaults() Settings {
	return Settings{
		Style:           StyleYellow,
		Volume:          12,
		MaxVolume:       20,
		Brightness:      60,
		NightBrightness: 5,
		Bedtime:         "19:00",
		Waketime:        "07:00",
		Language:        "nl",
	}
}

// Load reads the settings at path. Whatever is missing or makes no sense is the default, so a file
// from an older build, or none at all, is still a working clock.
func Load(path string) (Settings, error) {
	s := Defaults()
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return Defaults(), err
	}
	return s.Normalized(), nil
}

// Save writes the settings to path whole, through a file beside it, so a power cut never leaves half
// a file behind.
func (s Settings) Save(path string) error {
	b, err := json.MarshalIndent(s.Normalized(), "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Normalized is s with every value inside its range and every unknown one back to its default.
func (s Settings) Normalized() Settings {
	d := Defaults()
	if s.Style != StyleYellow && s.Style != StyleDark {
		s.Style = d.Style
	}
	s.MaxVolume = clamp(s.MaxVolume, 1, VolumeSteps)
	s.Volume = clamp(s.Volume, 0, s.MaxVolume)
	s.Brightness = clamp(s.Brightness, 5, 100)
	s.NightBrightness = clamp(s.NightBrightness, 1, 100)
	if _, ok := minuteOf(s.Bedtime); !ok {
		s.Bedtime = d.Bedtime
	}
	if _, ok := minuteOf(s.Waketime); !ok {
		s.Waketime = d.Waketime
	}
	if s.Language == "" || s.Language != filepath.Base(s.Language) {
		s.Language = d.Language
	}
	return s
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
