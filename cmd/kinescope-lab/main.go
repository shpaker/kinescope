//go:build js

// Command kinescope-lab is a lab for kinescope, built for the browser: make
// a setup — effects, sources, drives, schedules — watch a test card, a
// moving scene or a dropped picture through it, tell the TV the game's
// facts, and take the setup away as Go. The page around it is in web/.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope/cmd/kinescope-lab/internal/lab"
	"github.com/shpaker/kinescope/ebitengine"
)

// version is the library's version, set by the build.
var version = "dev"

func main() {
	renderer, err := ebitengine.NewRenderer()
	if err != nil {
		log.Fatal(err)
	}
	g := newGame(lab.New(version, ""), renderer)
	newBridge(g).publish()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
