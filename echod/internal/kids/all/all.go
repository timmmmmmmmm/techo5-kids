// Package all is what TECHO5 Kids is made of, in place of internal/component/all: the screen's touch,
// the buttons, the speaker, SSH for whoever looks after the device, and the clock.
//
// Components register themselves when they are imported, so what is not in this list - and not
// imported by anything in it - does not exist in the binary at all. The microphones, the camera, the
// voice assistant and Home Assistant's voice satellite are none of them here, and privacy_test.go
// keeps it that way.
package all
