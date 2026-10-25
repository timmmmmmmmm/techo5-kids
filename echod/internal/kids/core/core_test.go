package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func at(hm string) time.Time {
	m, _ := minuteOf(hm)
	return time.Date(2026, 10, 25, m/60, m%60, 0, 0, time.Local)
}

func TestTheNightCrossesMidnight(t *testing.T) {
	s := Defaults() // 19:00 to 07:00
	for hm, night := range map[string]bool{
		"18:59": false, "19:00": true, "23:30": true, "00:00": true, "06:59": true, "07:00": false, "12:00": false,
	} {
		if got := s.Night(at(hm)); got != night {
			t.Errorf("%s: night %v, want %v", hm, got, night)
		}
	}
}

func TestANightWithinOneDay(t *testing.T) {
	s := Defaults()
	s.Bedtime, s.Waketime = "13:00", "15:00" // a nap
	if !s.Night(at("14:00")) || s.Night(at("15:00")) || s.Night(at("12:59")) {
		t.Error("the nap is not 13:00 to 15:00")
	}
	s.Waketime = s.Bedtime
	if s.Night(at("13:00")) {
		t.Error("a bedtime equal to the waketime is a night")
	}
}

func TestSettingsMissingOrNonsenseAreTheDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kids.json")
	s, err := Load(path)
	if err != nil || s != Defaults() {
		t.Fatalf("no file: %+v, %v", s, err)
	}
	if err := os.WriteFile(path, []byte(`{"style":"pink","volume":99,"max_volume":15,"bedtime":"25:00","language":"../x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	d := Defaults()
	if s.Style != d.Style || s.MaxVolume != 15 || s.Volume != 15 || s.Bedtime != d.Bedtime || s.Language != d.Language {
		t.Errorf("not normalized: %+v", s)
	}
}

func TestSettingsSurviveASaveAndALoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kids.json")
	s := Defaults()
	s.Style, s.Volume, s.Bedtime = StyleDark, 7, "19:30"
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != s {
		t.Errorf("got %+v, %v; want %+v", got, err, s)
	}
}

func TestEveryRowCyclesRoundToWhereItStarted(t *testing.T) {
	for _, id := range []string{RowMaxVolume, RowBrightness, RowNightLight, RowStyle, RowBedtime, RowWaketime} {
		start := Defaults()
		start.Volume = 5 // under every maximum, so lowering one does not move it
		s := Cycle(start, id)
		if s == start {
			t.Errorf("%s: a tap changed nothing", id)
		}
		for range 20 {
			if s = Cycle(s, id); s == start {
				break
			}
		}
		if s != start {
			t.Errorf("%s: never came round to the start again", id)
		}
	}
}

func TestLoweringTheMaximumTakesTheVolumeDownWithIt(t *testing.T) {
	s := Defaults()
	s.MaxVolume, s.Volume = 30, 28
	s = Cycle(s, RowMaxVolume) // 30 is the last choice: round to 10
	if s.MaxVolume != 10 || s.Volume != 10 {
		t.Errorf("max %d, volume %d", s.MaxVolume, s.Volume)
	}
}

func TestTheChordWantsBothButtonsForTheWholeHold(t *testing.T) {
	now := time.Now()
	long, short := now.Add(-ChordHold), now.Add(-ChordHold/2)
	if !ChordDue(true, long, true, long, now) {
		t.Error("both held long enough is not the chord")
	}
	if ChordDue(true, long, true, short, now) || ChordDue(true, long, false, long, now) {
		t.Error("one button, or one held too briefly, is the chord")
	}
}
