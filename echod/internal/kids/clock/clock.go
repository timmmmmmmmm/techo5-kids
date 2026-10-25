//go:build linux

package clock

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/HuskerMinion/techo5/echod/internal/component"
	"github.com/HuskerMinion/techo5/echod/internal/feature/security"
	"github.com/HuskerMinion/techo5/echod/internal/hardware/buttons"
	"github.com/HuskerMinion/techo5/echod/internal/hardware/screen"
	"github.com/HuskerMinion/techo5/echod/internal/hardware/speaker"
	"github.com/HuskerMinion/techo5/echod/internal/hardware/touch"
	"github.com/HuskerMinion/techo5/echod/internal/kids/core"
	"github.com/HuskerMinion/techo5/echod/internal/kids/face"
	"github.com/HuskerMinion/techo5/echod/internal/service"
)

func init() {
	component.Register(component.Device, Get(), component.Order(60),
		component.Supervise(service.Restart(time.Second, 30*time.Second)))
}

const (
	// volumeShows is how long the volume stays on the rim after it last changed.
	volumeShows = 2 * time.Second
	// saveAfter is how long the volume has to stay put before it is written down: a ramp on the
	// buttons is one write, not thirty.
	saveAfter = 3 * time.Second
	// menuNightBrightness is the least the menu is lit at night, in percent.
	menuNightBrightness = 15

	// walkFrame is how often the hands are drawn while they walk back to the real time.
	walkFrame = 45 * time.Millisecond

	// tapLikeHold and tapLikeMove are the touch layer's tapHold and tapMove: in follow mode a wobbling
	// tap arrives as a hold and a drag instead, and the clock says the time for it all the same.
	tapLikeHold = 500 * time.Millisecond
	tapLikeMove = 40

	// tapAgain is how close two taps must be to be a pair: the second of one, on the children's
	// clock, sends it home.
	tapAgain = 350 * time.Millisecond
)

// Clock is the screen and everything that reaches it.
type Clock struct {
	mu       sync.Mutex
	set      core.Settings
	menu     bool      // the parent menu is up
	editing  int       // the setting open on its own page, -1 for the ring
	menuAt   time.Time // and was last touched
	volumeAt time.Time // the volume last changed
	chord    *time.Timer
	save     *time.Timer

	// the children's play, and what a finger that may still turn out to be a tap has done
	play      core.Play
	playPrev  core.Play   // the play before the finger took a hand, for a release that was a tap
	holdX, holdY  int     // where the finger went down
	holdAt        time.Time
	lastTap       time.Time // the tap before this one, for pairs
	soon          *time.Timer // a frame of the walk back, asked for

	dev   *screen.Device
	light int // the backlight as last set, -1 before the first frame
	poke  chan struct{}
	ears  sync.Once
}

var (
	once   sync.Once
	shared *Clock
)

// Get is the clock. There is one screen.
func Get() *Clock {
	once.Do(func() { shared = &Clock{poke: make(chan struct{}, 1), light: -1, editing: -1} })
	return shared
}

func (c *Clock) Name() string { return "kids clock" }

// Start reads the settings, opens the framebuffer and starts listening to the touch screen and the
// buttons.
func (c *Clock) Start(context.Context) error {
	s, err := core.Load(core.SettingsPath)
	if err != nil {
		slog.Warn("kids: reading the settings failed, using the defaults", "err", err)
	}
	c.mu.Lock()
	c.set = s
	c.mu.Unlock()
	speaker.Get().SetVolume(s.Volume)

	dev, err := screen.Open()
	if err != nil {
		return err
	}
	c.dev, c.light = dev, -1
	c.ears.Do(func() {
		// Every moving finger is followed, so a child can take a hand and turn it the moment the
		// finger moves; a tap is a tap still, and the wobbling ones the clock knows apart (gesture).
		touch.Get().SetFollow(true)
		touch.Get().Gestures.Listen(c.gesture)
		buttons.Get().Events.Listen(c.button)
	})
	slog.Info("kids: screen open", "fb", dev.String(), "style", s.Style, "bedtime", s.Bedtime, "waketime", s.Waketime)
	return nil
}

func (c *Clock) Close() error {
	if c.dev == nil {
		return nil
	}
	err := c.dev.Close()
	c.dev = nil
	return err
}

// Run draws a frame every second, and at once whenever something changes.
func (c *Clock) Run(ctx context.Context) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		c.frame(time.Now())
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		case <-c.poke:
		}
	}
}

// wake asks for a frame now rather than at the next second.
func (c *Clock) wake() {
	select {
	case c.poke <- struct{}{}:
	default:
	}
}

func (c *Clock) frame(now time.Time) {
	c.mu.Lock()
	if c.menu && now.Sub(c.menuAt) > core.MenuIdle {
		c.menu, c.editing = false, -1
		slog.Info("kids: parent menu closed, left alone")
	}
	walking := c.play.Tick(now)
	shown := c.play.Shown(now)
	s, menu := c.set, c.menu
	sc := face.Scene{
		Now: shown, Style: s.Style, Night: s.Night(now), Editing: c.editing,
		Volume: s.Volume, MaxVolume: s.MaxVolume, ShowVolume: now.Sub(c.volumeAt) < volumeShows,
		Playing: c.play.Playing(),
	}
	c.mu.Unlock()
	if menu {
		st := status()
		sc.Menu, sc.Address = core.MenuSettings(s, st), st.Address
	}
	face.Draw(c.dev.Canvas(), sc)
	if err := c.dev.Present(); err != nil {
		slog.Warn("kids: presenting a frame failed", "err", err)
	}

	// At night the menu is lit only enough to read, so a parent opening it in the dark does not light
	// the room.
	pct := s.Brightness
	if sc.Night {
		pct = s.NightBrightness
		if menu {
			pct = max(pct, menuNightBrightness)
		}
	}
	if level := max(1, pct*screen.BacklightMax/100); level != c.light {
		if err := screen.SetBacklight(level); err != nil {
			slog.Warn("kids: setting the backlight failed", "err", err)
		} else {
			c.light = level
		}
	}
	if walking {
		c.pokeSoon(walkFrame)
	}
}

// pokeSoon asks for a frame d later, while the hands walk back, replacing one asked for before.
func (c *Clock) pokeSoon(d time.Duration) {
	c.mu.Lock()
	if c.soon != nil {
		c.soon.Stop()
	}
	c.soon = time.AfterFunc(d, func() {
		c.mu.Lock()
		c.soon = nil
		c.mu.Unlock()
		c.wake()
	})
	c.mu.Unlock()
}

// gesture is a finger on the screen. Listeners run on the touch reader: nothing here may block.
func (c *Clock) gesture(g touch.Gesture) {
	c.mu.Lock()
	menu := c.menu
	if menu {
		c.menuAt = time.Now()
	}
	c.mu.Unlock()
	if menu {
		if g.Kind == touch.Tap {
			c.menuTap(g.X, g.Y)
		}
		c.wake()
		return
	}
	now := time.Now()
	c.mu.Lock()
	// The scene says where the hands are, which is what a child takes one of.
	sc := face.Scene{Now: c.play.Shown(now), Night: c.set.Night(now)}
	say, hand := time.Time{}, core.HandNone
	switch g.Kind {
	case touch.Tap:
		if c.play.Set() && now.Sub(c.lastTap) < tapAgain {
			// The second tap of a pair on the children's clock: done playing, home at once.
			c.play.SendHome(now)
			slog.Info("kids: a second tap sends the clock home")
		} else {
			c.play.Touch(now)
			say = c.play.Shown(now)
		}
		c.lastTap = now
	case touch.Hold:
		c.holdX, c.holdY, c.holdAt, c.playPrev = g.X, g.Y, now, c.play
		hand = face.HandAt(sc, g.X, g.Y)
		if hand != core.HandNone {
			c.play.Grab(hand, sc.Now, face.Turns(sc, g.X, g.Y))
		}
	case touch.Drag:
		if c.play.Held() != core.HandNone {
			c.play.Drag(face.Turns(sc, g.X, g.Y))
		}
	case touch.Release:
		// A wobble of a tap arrives as a hold and a drag in follow mode, and a grab that never
		// turned the hand was no grab either: both are taps, or nothing, and the play goes back to
		// what it was.
		if now.Sub(c.holdAt) < tapLikeHold && abs(g.X-c.holdX) <= tapLikeMove && abs(g.Y-c.holdY) <= tapLikeMove {
			c.play, say = c.playPrev, c.playPrev.Shown(now)
			c.play.Touch(now)
			c.lastTap = now
		} else if c.play.Held() != core.HandNone && !c.play.Turned() {
			c.play = c.playPrev
		} else {
			c.play.Release(now)
		}
	}
	c.mu.Unlock()
	if hand != core.HandNone {
		slog.Info("kids: a hand taken", "hand", hand == core.HandMinute)
	}
	if !say.IsZero() {
		sayTime(say, c.settings().Language)
	}
	c.wake()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// button is a button. Volume steps on the press and as it repeats; the top button, which on TECHO5
// mutes the microphones this build does not have, says the time.
func (c *Clock) button(e buttons.Event) {
	switch e.Name {
	case buttons.VolumeUp, buttons.VolumeDown:
		if e.Kind != buttons.Tap && e.Kind != buttons.Repeat {
			return
		}
		if c.chording() {
			return
		}
		if e.Name == buttons.VolumeUp {
			c.stepVolume(+1)
		} else {
			c.stepVolume(-1)
		}
	case buttons.Mute:
		if e.Kind == buttons.Tap {
			sayTime(time.Now(), c.settings().Language)
		}
	}
}

// chording reports whether both volume buttons are down, the start of the parent's chord, and times
// the chord the first time it sees them so.
func (c *Clock) chording() bool {
	up, _ := buttons.Get().Pressed(buttons.VolumeUp)
	down, _ := buttons.Get().Pressed(buttons.VolumeDown)
	if !up || !down {
		return false
	}
	c.mu.Lock()
	if c.chord == nil {
		c.chord = time.AfterFunc(core.ChordHold+100*time.Millisecond, c.chordDue)
	}
	c.mu.Unlock()
	return true
}

// chordDue opens the menu, or closes it, if both buttons are still down and have been all along.
func (c *Clock) chordDue() {
	up, upAt := buttons.Get().Pressed(buttons.VolumeUp)
	down, downAt := buttons.Get().Pressed(buttons.VolumeDown)
	due := core.ChordDue(up, upAt, down, downAt, time.Now())
	c.mu.Lock()
	c.chord = nil
	if due {
		c.menu, c.editing, c.menuAt = !c.menu, -1, time.Now()
		if c.menu {
			c.play.Cancel() // the menu wants the screen to itself; the real clock waits behind it
		}
		slog.Info("kids: parent menu", "open", c.menu)
	}
	c.mu.Unlock()
	if due {
		tick()
		c.wake()
	}
}

// menuTap is a tap in the parent menu at x, y.
func (c *Clock) menuTap(x, y int) {
	c.mu.Lock()
	menu := core.MenuSettings(c.set, status())
	t := face.MenuTarget(x, y, menu, c.editing)
	before := c.set
	switch t.Kind {
	case face.TargetNone:
		c.mu.Unlock()
		return
	case face.TargetDone:
		if c.editing < 0 {
			c.menu = false
		}
		c.editing = -1
	case face.TargetOpen:
		c.editing = t.Index
	case face.TargetMinus, face.TargetPlus:
		delta := 1
		if t.Kind == face.TargetMinus {
			delta = -1
		}
		c.set = core.Step(c.set, menu[c.editing].ID, delta)
	case face.TargetPick:
		if id := menu[c.editing].ID; id == core.RowSSH {
			go security.Get().SetSSH(t.Index == 1)
		} else {
			c.set = core.Choose(c.set, id, t.Index)
		}
	}
	after, open, editing := c.set, c.menu, c.editing
	c.mu.Unlock()
	slog.Info("kids: parent menu", "tap", t.Kind, "open", open, "editing", editing)
	tick()
	if after != before {
		speaker.Get().SetVolume(after.Volume)
		if err := after.Save(core.SettingsPath); err != nil {
			slog.Error("kids: saving the settings failed", "err", err)
		}
	}
}

// stepVolume moves the volume one step, never past the parent's maximum, and lets it be heard.
func (c *Clock) stepVolume(step int) {
	c.mu.Lock()
	v := max(0, min(c.set.Volume+step, c.set.MaxVolume))
	c.set.Volume, c.volumeAt = v, time.Now()
	if c.save != nil {
		c.save.Stop()
	}
	c.save = time.AfterFunc(saveAfter, func() {
		if err := c.settings().Save(core.SettingsPath); err != nil {
			slog.Error("kids: saving the volume failed", "err", err)
		}
	})
	c.mu.Unlock()
	speaker.Get().SetVolume(v)
	tick()
	c.wake()
}

func (c *Clock) settings() core.Settings {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.set
}

// tick is the short click a step makes, at the volume it is now, so a turn of the volume is heard.
func tick() {
	speaker.Sound().Chime([]speaker.Note{{Freq: 1320, Ms: 28}})
}

func status() core.Status {
	return core.Status{SSH: security.Get().State().SSH, Address: address()}
}

// address is the Wi-Fi's IPv4 address, for a parent who wants to reach the device.
func address() string {
	iface, err := net.InterfaceByName("wlan0")
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
			return n.IP.String()
		}
	}
	return ""
}
