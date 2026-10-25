package core

import (
	"slices"
	"strconv"
	"time"
)

// The parent menu: a short list on the round screen where a tap on a row moves its setting to the next
// choice, round to the first again. Fewer choices than a slider, but every one is a single tap with a
// thumb, and nothing on it needs typing.

// ChordHold is how long both volume buttons have to be held together to open the menu: long enough
// that a child pressing everything at once lets go first.
const ChordHold = 3 * time.Second

// MenuIdle is how long the menu stays up untouched before the clock comes back by itself, so a menu
// left open is never what the children find.
const MenuIdle = 45 * time.Second

// Row ids. The menu's own actions (SSH, done) are handled by whoever owns what they change.
const (
	RowMaxVolume  = "max_volume"
	RowBrightness = "brightness"
	RowNightLight = "night_brightness"
	RowStyle      = "style"
	RowBedtime    = "bedtime"
	RowWaketime   = "waketime"
	RowSSH        = "ssh"
	RowDone       = "done"
)

// Row is one line of the menu.
type Row struct {
	ID, Label, Value string
}

// Status is what the menu shows that is not a setting of the clock's own.
type Status struct {
	SSH     bool
	Address string // the device's IPv4 address, empty while it has none
}

var (
	maxVolumes  = []int{10, 15, 20, 25, 30}
	brightness  = []int{20, 40, 60, 80, 100}
	nightLights = []int{1, 3, 5, 10, 20}
	bedtimes    = []string{"18:00", "18:30", "19:00", "19:30", "20:00", "20:30", "21:00"}
	waketimes   = []string{"06:00", "06:30", "07:00", "07:30", "08:00"}
)

// MenuRows is the menu as it stands with s.
func MenuRows(s Settings, st Status) []Row {
	style := "Geel"
	if s.Style == StyleDark {
		style = "Zwart"
	}
	ssh := "uit"
	if st.SSH {
		ssh = "aan"
	}
	return []Row{
		{RowMaxVolume, "Max. volume", strconv.Itoa(s.MaxVolume)},
		{RowBrightness, "Helderheid", strconv.Itoa(s.Brightness) + "%"},
		{RowNightLight, "Nachtlicht", strconv.Itoa(s.NightBrightness) + "%"},
		{RowStyle, "Klok", style},
		{RowBedtime, "Bedtijd", s.Bedtime},
		{RowWaketime, "Wektijd", s.Waketime},
		{RowSSH, "SSH", ssh},
		{RowDone, "Klaar", ""},
	}
}

// Cycle is s after a tap on the row with id: that setting moved on to its next choice. Rows that are
// not settings of the clock's own leave s as it is.
func Cycle(s Settings, id string) Settings {
	switch id {
	case RowMaxVolume:
		s.MaxVolume = next(maxVolumes, s.MaxVolume)
		s.Volume = min(s.Volume, s.MaxVolume)
	case RowBrightness:
		s.Brightness = next(brightness, s.Brightness)
	case RowNightLight:
		s.NightBrightness = next(nightLights, s.NightBrightness)
	case RowStyle:
		if s.Style == StyleDark {
			s.Style = StyleYellow
		} else {
			s.Style = StyleDark
		}
	case RowBedtime:
		s.Bedtime = next(bedtimes, s.Bedtime)
	case RowWaketime:
		s.Waketime = next(waketimes, s.Waketime)
	}
	return s
}

// next is the choice after cur, or the first when cur is the last or not one of them (a value set some
// other way, from Home Assistant say).
func next[T comparable](choices []T, cur T) T {
	i := slices.Index(choices, cur)
	return choices[(i+1)%len(choices)]
}

// ChordDue reports whether both buttons are down now and have been for ChordHold.
func ChordDue(downA bool, sinceA time.Time, downB bool, sinceB time.Time, now time.Time) bool {
	return downA && downB && now.Sub(sinceA) >= ChordHold && now.Sub(sinceB) >= ChordHold
}
