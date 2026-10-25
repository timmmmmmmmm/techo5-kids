package core

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestDutchTimeMatchesThePrototype pins the phrases against the prototype's fixture, the same file
// kids-ui tests against, so the Go port and the TypeScript original cannot drift apart.
func TestDutchTimeMatchesThePrototype(t *testing.T) {
	b, err := os.ReadFile("testdata/dutch-time.json")
	if err != nil {
		t.Fatal(err)
	}
	var pins []struct {
		Time    string `json:"time"`
		Exact   string `json:"exact"`
		Rounded string `json:"rounded"`
	}
	if err := json.Unmarshal(b, &pins); err != nil {
		t.Fatal(err)
	}
	if len(pins) < 20 {
		t.Fatalf("only %d pins, want the whole fixture", len(pins))
	}
	for _, pin := range pins {
		var h, m int
		if _, err := fmt.Sscanf(pin.Time, "%d:%d", &h, &m); err != nil {
			t.Fatalf("%s: %v", pin.Time, err)
		}
		if got := DutchTime(h, m); got != pin.Exact {
			t.Errorf("%s: exact %q, want %q", pin.Time, got, pin.Exact)
		}
		if got := RoundedTime(h, m); got != pin.Rounded {
			t.Errorf("%s: rounded %q, want %q", pin.Time, got, pin.Rounded)
		}
		if got := TimeSentence(h, m, false); got != "Het is "+pin.Exact+"." {
			t.Errorf("%s: sentence %q", pin.Time, got)
		}
	}
}

// TestEveryMinuteHasAPhrase walks the whole twelve-hour clock: every minute of it speaks, and every
// phrase is a sane one.
func TestEveryMinuteHasAPhrase(t *testing.T) {
	seen := map[string]bool{}
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m++ {
			s := DutchTime(h, m)
			if s == "" {
				t.Fatalf("%02d:%02d has no phrase", h, m)
			}
			seen[s] = true
			TimeSentence(h, m, true) // rounded too, and at night
			NightSentence(h, m, 45, false)
			NightSentence(h, m, 30, true)
		}
	}
	if len(seen) < 200 {
		t.Errorf("only %d phrases for 720 minutes", len(seen))
	}
}
