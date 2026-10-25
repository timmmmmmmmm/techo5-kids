package kids

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// listensOrLooks are the packages that open a microphone or the camera, or would hand what they hear or
// see to anybody: none of them may be in the kids binary, directly or through anything it imports.
var listensOrLooks = []string{
	"hardware/mic",     // the microphones
	"hardware/camera",  // the camera
	"feature/voice",    // conversations with an assistant
	"feature/wakeword", // listening for a wake word
	"feature/microphone",
	"feature/recording",
	"feature/detect", // words without a wake word
	"feature/clap",
	"feature/noise",
	"feature/room",
	"feature/presence", // the camera watching for somebody
	"feature/camera",
	"feature/camstream", // the camera's pages
	"feature/talkback",  // talking through a camera
	"feature/phone",     // calls
	"feature/announce",  // announcements, which record
	"feature/assistant",
	"feature/api", // Home Assistant's voice satellite
}

// TestNothingThatListensOrLooksIsInTheBinary asks the Go toolchain what the Spot's kids binary is built
// from, and fails if a single package that could listen or look is among it.
func TestNothingThatListensOrLooksIsInTheBinary(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no Go toolchain to ask")
	}
	const module = "github.com/HuskerMinion/techo5/echod/"
	cmd := exec.Command("go", "list", "-deps", "-tags", "spot", "-f", "{{.ImportPath}}", module+"cmd/kidsd")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm", "GOARM=7", "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	deps := map[string]bool{}
	for _, p := range strings.Fields(string(out)) {
		deps[strings.TrimPrefix(p, module+"internal/")] = true
	}
	if !deps["kids/clock"] {
		t.Fatal("the list is not the kids binary's: no kids/clock in it")
	}
	for _, p := range listensOrLooks {
		if deps[p] {
			t.Errorf("%s is in the kids binary", p)
		}
	}
}
