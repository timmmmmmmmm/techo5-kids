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
)

// Clock is the screen and everything that reaches it.
type Clock struct {
	mu       sync.Mutex
	set      core.Settings
	menu     bool      // the parent menu is up
	menuAt   time.Time // and was last touched
	volumeAt time.Time // the volume last changed
	chord    *time.Timer
	save     *time.Timer

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
	once.Do(func() { shared = &Clock{poke: make(chan struct{}, 1), light: -1} })
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
		c.menu = false
		slog.Info("kids: parent menu closed, left alone")
	}
	s, menu := c.set, c.menu
	sc := face.Scene{
		Now: now, Style: s.Style, Night: s.Night(now),
		Volume: s.Volume, MaxVolume: s.MaxVolume, ShowVolume: now.Sub(c.volumeAt) < volumeShows,
	}
	c.mu.Unlock()
	if menu {
		sc.Menu, sc.Address = core.MenuRows(s, status()), address()
	}

	face.Draw(c.dev.Canvas(), sc)
	if err := c.dev.Present(); err != nil {
		slog.Warn("kids: presenting a frame failed", "err", err)
	}

	pct := s.Brightness
	if sc.Night && !menu {
		pct = s.NightBrightness
	}
	if level := max(1, pct*screen.BacklightMax/100); level != c.light {
		if err := screen.SetBacklight(level); err != nil {
			slog.Warn("kids: setting the backlight failed", "err", err)
		} else {
			c.light = level
		}
	}
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
			c.menuTap(g.Y)
		}
		c.wake()
		return
	}
	switch g.Kind {
	case touch.Tap:
		sayTime(time.Now(), c.settings().Language)
	case touch.SwipeUp:
		c.stepVolume(+1)
	case touch.SwipeDown:
		c.stepVolume(-1)
	}
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
		c.menu, c.menuAt = !c.menu, time.Now()
		slog.Info("kids: parent menu", "open", c.menu)
	}
	c.mu.Unlock()
	if due {
		tick()
		c.wake()
	}
}

// menuTap is a tap in the parent menu at height y.
func (c *Clock) menuTap(y int) {
	c.mu.Lock()
	rows := core.MenuRows(c.set, status())
	i := face.MenuRowAt(y, len(rows))
	if i < 0 {
		c.mu.Unlock()
		return
	}
	id := rows[i].ID
	var saved core.Settings
	switch id {
	case core.RowDone:
		c.menu = false
	case core.RowSSH:
		on := !security.Get().State().SSH
		go security.Get().SetSSH(on)
	default:
		c.set = core.Cycle(c.set, id)
		saved = c.set
	}
	c.mu.Unlock()
	slog.Info("kids: parent menu", "tapped", id)
	tick()
	if saved != (core.Settings{}) {
		speaker.Get().SetVolume(saved.Volume)
		if err := saved.Save(core.SettingsPath); err != nil {
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
