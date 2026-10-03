// Command kinescope-lab is a lab for kinescope's effects: turn them on and
// off, move their sliders, give the set moods, and watch a test card, a
// moving scene or a dropped picture through it. Share the setup as a link
// or copy it as Go.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope/ebitengine"
)

func main() {
	p := newPlatform()
	l, err := newLab(p.State())
	if err != nil {
		log.Fatal(err)
	}
	r, err := ebitengine.NewRenderer()
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("kinescope lab")
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(newGame(l, r, p)); err != nil {
		log.Fatal(err)
	}
}
