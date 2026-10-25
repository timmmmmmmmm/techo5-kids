// Package clock is TECHO5 Kids' one screen. It owns the panel, the touch screen and the buttons, and
// turns them into the learning clock (kids/face) by the clock's rules (kids/core):
//
//   - a tap on the clock, or a press of the top button, says the time;
//   - the volume buttons and a swipe up or down set the volume, up to the parent's maximum;
//   - both volume buttons held together for three seconds open the parent menu, which closes again by
//     itself if it is left alone;
//   - between bedtime and waketime the night sky is up instead, with the backlight down.
//
// Nothing else reaches the screen: there is no ring menu and no assistant to wake. It needs Linux (the
// speaker is ALSA), so elsewhere the package is empty and only kids/core and kids/face are built.
package clock
