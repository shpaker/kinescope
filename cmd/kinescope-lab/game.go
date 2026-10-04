//go:build js

package main

import (
	"image"
	"image/color"
	_ "image/jpeg" // dropped pictures
	_ "image/png"  // dropped pictures
	"io/fs"
	"log"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope/cmd/kinescope-lab/internal/lab"
	"github.com/shpaker/kinescope/ebitengine"
)

// game shows a picture through the lab's TV over the whole screen. The
// page changes the lab between ticks: every tick and every call of the
// page holds mu.
type game struct {
	mu sync.Mutex

	// Model and its view
	lab      *lab.Lab
	renderer *ebitengine.Renderer

	// Sources: the test card, the moving scene and a dropped picture
	sources []source
	picture picture

	// failed stops the TV after its shader would not compile
	failed bool

	// Screen
	displayScale float64
	screenWidth  float64
	screenHeight float64
}

var _ ebiten.Game = (*game)(nil)

// picture is how the page wants the picture shown.
type picture struct {
	Source  int  `json:"source"`  // 0 the test card, 1 the scene, 2 the dropped picture
	Scale   int  `json:"scale"`   // screen pixels per frame pixel; 0 fits the screen
	Bypass  bool `json:"bypass"`  // the picture without the TV
	Dropped bool `json:"dropped"` // a picture has been dropped
}

func newGame(l *lab.Lab, r *ebitengine.Renderer) *game {
	return &game{
		lab:      l,
		renderer: r,
		sources:  []source{newTestCard(), newScene(), nil},
	}
}

func (g *game) source() source {
	if s := g.sources[g.picture.Source]; s != nil {
		return s
	}
	return g.sources[0]
}

func (g *game) Update() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.takeDroppedPicture()
	g.lab.TV().Update(1 / float64(ebiten.TPS()))
	g.source().update()
	return nil
}

// takeDroppedPicture shows a picture dropped onto the screen.
func (g *game) takeDroppedPicture() {
	files := ebiten.DroppedFiles()
	if files == nil {
		return
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil || len(entries) == 0 {
		return
	}
	file, err := files.Open(entries[0].Name())
	if err != nil {
		log.Print(err)
		return
	}
	defer func() { _ = file.Close() }()
	img, _, err := image.Decode(file)
	if err != nil {
		log.Print(err)
		return
	}
	g.sources[2] = &still{img: ebiten.NewImageFromImage(img)}
	g.picture.Source, g.picture.Dropped = 2, true
}

func (g *game) Draw(screen *ebiten.Image) {
	g.mu.Lock()
	defer g.mu.Unlock()
	screen.Fill(color.Black)
	frame := g.source().frame()
	geoM := g.place(frame.Bounds().Size())

	if g.picture.Bypass || g.failed {
		screen.DrawImage(frame, &ebiten.DrawImageOptions{GeoM: geoM})
	} else if err := g.renderer.Draw(screen, frame, g.lab.TV(), geoM); err != nil {
		log.Print(err)
		g.failed = true
	}
}

// place puts the frame in the middle of the screen, scaled by a whole
// number.
func (g *game) place(frame image.Point) ebiten.GeoM {
	w, h := int(g.screenWidth), int(g.screenHeight)
	scale := g.picture.Scale * max(1, int(math.Round(g.displayScale)))
	if g.picture.Scale == 0 {
		scale = max(1, min(w/frame.X, h/frame.Y))
	}
	var geoM ebiten.GeoM
	geoM.Scale(float64(scale), float64(scale))
	geoM.Translate(float64((w-frame.X*scale)/2), float64((h-frame.Y*scale)/2))
	return geoM
}

func (g *game) Layout(int, int) (int, int) {
	panic("LayoutF is used instead")
}

func (g *game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	g.displayScale = ebiten.Monitor().DeviceScaleFactor()
	g.screenWidth = math.Max(1, outsideWidth*g.displayScale)
	g.screenHeight = math.Max(1, outsideHeight*g.displayScale)
	return g.screenWidth, g.screenHeight
}
