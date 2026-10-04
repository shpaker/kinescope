// Package kinescope draws the picture of an old tube TV for Go games:
// scanlines, a phosphor mask, glow, a bulging glass — and the moods of a worn
// set: drifting reception, torn lines, a shiver when the channel changes.
//
// A Setup is a whole TV as data; NewTV builds a TV from it, and the game
// tells the TV its facts: time with TV.Update, signal levels, episodes. The
// ebitengine package draws a frame through the TV.
//
//	tv, err := kinescope.NewTV(kinescope.Gorizont())
//	renderer, err := ebitengine.NewRenderer()
//
//	// Every tick
//	tv.Update(1.0 / 60)
//
//	// In DrawFinalScreen
//	_ = renderer.Draw(screen, offscreen, tv, geoM)
//
// # The lab
//
// The lab, https://shpaker.github.io/kinescope/, shows a setup on a picture
// and writes it out as Go. A link opens it on a setup written as JSON, by
// hand or by an agent:
//
//	https://shpaker.github.io/kinescope/#j={"preset":"Rubin"}
//
// The format and everything a setup can be made of, effects and their params
// with ranges, are in https://shpaker.github.io/kinescope/llms.txt.
package kinescope
