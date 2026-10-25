// Command kidsclips renders the time clips the clock plays: one sentence per minute of a
// twelve-hour clock ("Het is tien voor half acht."), as 16-bit mono WAV at 16 kHz, named HHMM.wav
// the way the clock looks for them (internal/kids/clock/say.go).
//
// It runs on the Mac, not on the Spot:
//
//	go run ./cmd/kidsclips say Xander                    macOS' own voice: free and instant, for trying
//	go run ./cmd/kidsclips voxtral voice_3145            Mistral's Voxtral: key from $MISTRAL_API_KEY
//
// Files already rendered are kept, so a run picks up where the last one stopped. Voxtral clips come
// back as WAV and are turned to the clock's own shape with ffmpeg, which must be on the path.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
)

const (
	// speechURL is Mistral's text-to-speech endpoint; the model is named by flag so a newer one
	// can be tried without a new build.
	speechURL = "https://api.mistral.ai/v1/audio/speech"

	// clips is how many clips a full set is: every minute of twelve hours, 12:00 and 00:00 being
	// the same sentence.
	clips = 12 * 60
)

func main() {
	out := flag.String("out", "", "the folder to render into, HHMM.wav each (default: clips/time/<engine>-<voice>)")
	model := flag.String("model", "voxtral-mini-tts-3-0", "the Voxtral model, for the voxtral engine")
	keyFlag := flag.String("key", "", "the Mistral API key, if not in $MISTRAL_API_KEY")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: kidsclips [flags] <say|voxtral> [voice]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	engine, voice := flag.Arg(0), flag.Arg(1)
	if engine != "say" && engine != "voxtral" {
		fmt.Fprintf(os.Stderr, "unknown engine %q: say or voxtral\n", engine)
		os.Exit(2)
	}
	if engine == "say" && voice == "" {
		voice = "Xander"
	}
	if engine == "voxtral" && voice == "" {
		fmt.Fprintln(os.Stderr, "the voxtral engine needs a voice id (kidsclips voxtral voice_3145)")
		os.Exit(2)
	}
	dir := *out
	if dir == "" {
		dir = filepath.Join("clips", "time", engine+"-"+voice)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	key := *keyFlag
	if engine == "voxtral" && key == "" {
		key = os.Getenv("MISTRAL_API_KEY")
		if key == "" {
			fmt.Fprintln(os.Stderr, "the voxtral engine needs $MISTRAL_API_KEY or -key")
			os.Exit(2)
		}
	}

	start, done := time.Now(), 0
	for h := range 12 {
		for m := range 60 {
			name := fmt.Sprintf("%02d%02d.wav", h, m)
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				continue // already rendered: a run picks up where the last stopped
			}
			sentence := core.TimeSentence(h, m, false)
			var err error
			if engine == "say" {
				err = say(voice, sentence, path)
			} else {
				err = voxtral(*model, voice, key, sentence, path)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s (%q): %v\n", name, sentence, err)
				os.Exit(1)
			}
			done++
			if done%60 == 0 {
				fmt.Printf("%d/%d rendered\n", done, clips)
			}
		}
	}
	fmt.Printf("%s: %d new clips, %d kept, %s\n", dir, done, clips-done, time.Since(start).Round(time.Second))
}

// say renders one sentence with macOS' say, which writes the clock's own shape directly.
func say(voice, sentence, path string) error {
	return exec.Command("say", "-v", voice, "-o", path,
		"--file-format=WAVE", "--data-format=LEI16@16000", sentence).Run()
}

// speechResponse is one clip as it comes back from the API.
type speechResponse struct {
	AudioData string `json:"audio_data"`
}

// voxtral renders one sentence with Mistral's Voxtral and turns it to the clock's own shape with
// ffmpeg: 16-bit mono at 16 kHz, the rate the speaker plays clips at.
func voxtral(model, voice, key, sentence, path string) error {
	body, err := json.Marshal(map[string]string{
		"model": model, "input": sentence, "voice": voice, "response_format": "wav",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, speechURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	var clip speechResponse
	if err := json.NewDecoder(res.Body).Decode(&clip); err != nil {
		return err
	}
	wav, err := base64.StdEncoding.DecodeString(clip.AudioData)
	if err != nil {
		return err
	}
	raw := path + ".raw.wav"
	if err := os.WriteFile(raw, wav, 0o644); err != nil {
		return err
	}
	defer os.Remove(raw)
	return exec.Command("ffmpeg", "-v", "error", "-y", "-i", raw,
		"-ar", "16000", "-ac", "1", "-sample_fmt", "s16", path).Run()
}
