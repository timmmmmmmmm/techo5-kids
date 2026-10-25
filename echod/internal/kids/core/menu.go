package core

import (
	"slices"
	"strconv"
	"time"
)

// The parent menu. It is used with a finger on a screen the size of a coin, so everything on it is a
// large round button and nothing needs a precise touch or any typing: the settings sit round the ring,
// and a tap on one opens it on its own, where − and + move it a step (or two buttons choose between two
// things) and Klaar goes back.

// ChordHold is how long both volume buttons have to be held together to open the menu: long enough
// that a child pressing everything at once lets go first.
const ChordHold = 3 * time.Second

// MenuIdle is how long the menu stays up untouched before the clock comes back by itself, so a menu
// left open is never what the children find.
const MenuIdle = 45 * time.Second

// Setting ids. SSH is not the clock's own: whoever owns it acts on it.
const (
	RowMaxVolume  = "max_volume"
	RowBrightness = "brightness"
	RowNightLight = "night_brightness"
	RowStyle      = "style"
	RowBedtime    = "bedtime"
	RowWaketime   = "waketime"
	RowSSH        = "ssh"
)

// Kind is how a setting is changed.
type Kind int

const (
	// Steps is a setting moved a step at a time with − and +, stopping at either end.
	Steps Kind = iota
	// Choice is one of two, each its own button.
	Choice
)

// Setting is one button on the menu as it stands.
type Setting struct {
	ID    string
	Label string   // its name on its own page: "Max. volume"
	Short string   // under its value on the ring: "volume"
	Value string   // the value now, as shown
	Kind  Kind     //
	Steps int      // how many steps there are (Steps), for drawing how far along it is
	Index int      // which step or choice it is on, -1 for a value set elsewhere that is not one of them
	Names []string // the two choices' names (Choice)
}

// Status is what the menu shows that is not a setting of the clock's own.
type Status struct {
	SSH     bool
	Address string // the device's IPv4 address, empty while it has none
}

var (
	maxVolumes  = []int{5, 10, 15, 20, 25, 30}
	brightness  = []int{20, 40, 60, 80, 100}
	nightLights = []int{1, 3, 5, 10, 20}
	bedtimes    = []string{"18:00", "18:30", "19:00", "19:30", "20:00", "20:30", "21:00", "21:30"}
	waketimes   = []string{"06:00", "06:30", "07:00", "07:30", "08:00", "08:30"}
)

// MenuSettings is the menu's buttons, in their order round the ring.
func MenuSettings(s Settings, st Status) []Setting {
	style := Setting{ID: RowStyle, Label: "Klok", Short: "klok", Kind: Choice, Names: []string{"Geel", "Zwart"}}
	style.Index = 0
	if s.Style == StyleDark {
		style.Index = 1
	}
	style.Value = style.Names[style.Index]
	ssh := Setting{ID: RowSSH, Label: "SSH", Short: "ssh", Kind: Choice, Names: []string{"Uit", "Aan"}}
	if st.SSH {
		ssh.Index = 1
	}
	ssh.Value = ssh.Names[ssh.Index]
	return []Setting{
		ints(RowMaxVolume, "Max. volume", "volume", maxVolumes, s.MaxVolume, ""),
		ints(RowBrightness, "Helderheid", "licht", brightness, s.Brightness, "%"),
		ints(RowNightLight, "Nachtlicht", "nacht", nightLights, s.NightBrightness, "%"),
		style,
		times(RowBedtime, "Bedtijd", "bed", bedtimes, s.Bedtime),
		times(RowWaketime, "Wektijd", "op", waketimes, s.Waketime),
		ssh,
	}
}

func ints(id, label, short string, steps []int, v int, unit string) Setting {
	return Setting{ID: id, Label: label, Short: short, Value: strconv.Itoa(v) + unit, Kind: Steps,
		Steps: len(steps), Index: slices.Index(steps, v)}
}

func times(id, label, short string, steps []string, v string) Setting {
	return Setting{ID: id, Label: label, Short: short, Value: v, Kind: Steps,
		Steps: len(steps), Index: slices.Index(steps, v)}
}

// Step is s with a Steps setting moved by delta (−1 or +1), stopping at either end. A value that is not
// one of the steps moves to the nearest step in that direction.
func Step(s Settings, id string, delta int) Settings {
	switch id {
	case RowMaxVolume:
		s.MaxVolume = stepInt(maxVolumes, s.MaxVolume, delta)
		s.Volume = min(s.Volume, s.MaxVolume)
	case RowBrightness:
		s.Brightness = stepInt(brightness, s.Brightness, delta)
	case RowNightLight:
		s.NightBrightness = stepInt(nightLights, s.NightBrightness, delta)
	case RowBedtime:
		s.Bedtime = stepTime(bedtimes, s.Bedtime, delta)
	case RowWaketime:
		s.Waketime = stepTime(waketimes, s.Waketime, delta)
	}
	return s
}

// Choose is s with a Choice setting of the clock's own set to choice i.
func Choose(s Settings, id string, i int) Settings {
	if id == RowStyle {
		s.Style = StyleYellow
		if i == 1 {
			s.Style = StyleDark
		}
	}
	return s
}

func stepInt(steps []int, v, delta int) int {
	if i := slices.Index(steps, v); i >= 0 {
		return steps[max(0, min(i+delta, len(steps)-1))]
	}
	if delta > 0 {
		for _, x := range steps {
			if x > v {
				return x
			}
		}
		return steps[len(steps)-1]
	}
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i] < v {
			return steps[i]
		}
	}
	return steps[0]
}

func stepTime(steps []string, v string, delta int) string {
	m, ok := minuteOf(v)
	if !ok {
		return steps[0]
	}
	minutes := make([]int, len(steps))
	for i, s := range steps {
		minutes[i], _ = minuteOf(s)
	}
	return steps[slices.Index(minutes, stepInt(minutes, m, delta))]
}

// ChordDue reports whether both buttons are down now and have been for ChordHold.
func ChordDue(downA bool, sinceA time.Time, downB bool, sinceB time.Time, now time.Time) bool {
	return downA && downB && now.Sub(sinceA) >= ChordHold && now.Sub(sinceB) >= ChordHold
}
