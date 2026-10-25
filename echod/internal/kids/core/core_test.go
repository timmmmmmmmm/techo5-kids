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

func TestStepsStopAtEitherEnd(t *testing.T) {
	for _, id := range []string{RowMaxVolume, RowBrightness, RowNightLight, RowBedtime, RowWaketime} {
		s := Defaults()
		s.Volume = 0
		for range 20 {
			s = Step(s, id, +1)
		}
		top := s
		if Step(top, id, +1) != top {
			t.Errorf("%s: + past the last step moved", id)
		}
		for range 20 {
			s = Step(s, id, -1)
		}
		if Step(s, id, -1) != s || s == top {
			t.Errorf("%s: - did not stop at the first step", id)
		}
	}
}

func TestAValueSetElsewhereStepsToItsNeighbor(t *testing.T) {
	s := Defaults()
	s.Brightness, s.Bedtime = 50, "19:10"
	if got := Step(s, RowBrightness, +1).Brightness; got != 60 {
		t.Errorf("50%% + is %d, want 60", got)
	}
	if got := Step(s, RowBrightness, -1).Brightness; got != 40 {
		t.Errorf("50%% - is %d, want 40", got)
	}
	if got := Step(s, RowBedtime, +1).Bedtime; got != "19:30" {
		t.Errorf("19:10 + is %s, want 19:30", got)
	}
	if got := Step(s, RowBedtime, -1).Bedtime; got != "19:00" {
		t.Errorf("19:10 - is %s, want 19:00", got)
	}
}

func TestLoweringTheMaximumTakesTheVolumeDownWithIt(t *testing.T) {
	s := Defaults()
	s.MaxVolume, s.Volume = 15, 14
	s = Step(s, RowMaxVolume, -1)
	if s.MaxVolume != 10 || s.Volume != 10 {
		t.Errorf("max %d, volume %d", s.MaxVolume, s.Volume)
	}
}

func TestTheMenuShowsWhatIsSet(t *testing.T) {
	s := Defaults()
	s = Choose(s, RowStyle, 1)
	if s.Style != StyleDark {
		t.Fatal("choosing Zwart did not make the clock dark")
	}
	for _, m := range MenuSettings(s, Status{SSH: true}) {
		switch m.ID {
		case RowStyle:
			if m.Index != 1 || m.Value != "Zwart" {
				t.Errorf("style shows %d %q", m.Index, m.Value)
			}
		case RowSSH:
			if m.Index != 1 || m.Value != "Aan" {
				t.Errorf("ssh shows %d %q", m.Index, m.Value)
			}
		case RowMaxVolume:
			if m.Value != "20" || m.Index < 0 {
				t.Errorf("max volume shows %d %q", m.Index, m.Value)
			}
		}
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
