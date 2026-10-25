// Package kids is TECHO5 Kids, the fork's own part: a daemon for an Echo Spot in a child's room, built on
// TECHO5's hardware layer in place of TECHO5's daemon. Everything of it is under this directory and
// cmd/kidsd; the rest of the repository is TECHO5 as it is upstream, so it can be followed with a merge.
//
//	core   the clock's rules: settings, the night, the parent menu (no hardware; tests anywhere)
//	face   what the screen shows (draws into an image; previews anywhere)
//	clock  the screen, the touch screen and the buttons, put together (the Spot)
//	all    the parts the binary is made of
//	boot   start-up
//	cli    the command line
package kids
