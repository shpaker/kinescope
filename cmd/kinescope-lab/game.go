package main

import (
	"image"
	"image/color"
	_ "image/jpeg" // dropped pictures
	_ "image/png"  // dropped pictures
	"io/fs"
	"log"
	"math"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope/ebitengine"
)

// The panel's width in the UI's units
const panelWidth = 320

// saveEvery is how many ticks pass between saves of the lab's state.
const saveEvery = 30

// game runs the lab: the panel on the left, the picture through the TV on
// the right.
type game struct {
	// Model and its view
	lab      *lab
	renderer *ebitengine.Renderer
	platform platform
	ui       debugui.DebugUI

	// Sources: the test card, the moving scene and a dropped picture
	sources     []source
	sourceIndex int

	// What the panel shows and sets
	scale     int // screen pixels per frame pixel; 0 fits the window
	bypass    bool
	shake     float64
	shadows   map[string]*float64
	shader    string
	revision  int
	failed    bool
	savedTick int
	saved     string

	// Screen
	displayScale float64
	screenWidth  float64
	screenHeight float64
}

func newGame(l *lab, r *ebitengine.Renderer, p platform) *game {
	return &game{
		lab:      l,
		renderer: r,
		platform: p,
		sources:  []source{newTestCard(), newScene(), nil},
		shadows:  make(map[string]*float64),
		revision: -1,
	}
}

func (g *game) source() source {
	if s := g.sources[g.sourceIndex]; s != nil {
		return s
	}
	return g.sources[0]
}

func (g *game) Update() error {
	g.takeDroppedPicture()
	if _, err := g.ui.Update(g.panel); err != nil {
		return err
	}
	g.lab.tv.Update(1 / float64(ebiten.TPS()))
	g.source().update()

	g.savedTick++
	if g.savedTick >= saveEvery {
		g.savedTick = 0
		if state := g.lab.encode(); state != g.saved {
			g.saved = state
			g.platform.SetState(state)
		}
	}
	return nil
}

// takeDroppedPicture shows a picture dropped onto the window.
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
	g.sources[2] = &picture{img: ebiten.NewImageFromImage(img)}
	g.sourceIndex = 2
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x18, 0x18, 0x1c, 0xff})
	frame := g.source().frame()
	x, y, scale := g.place(frame.Bounds().Size())

	if g.bypass || g.failed {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(scale), float64(scale))
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(frame, op)
	} else if err := g.renderer.Draw(screen, frame, g.lab.tv, x, y, scale); err != nil {
		log.Print(err)
		g.failed = true
	}
	g.ui.Draw(screen)
}

// place is where the frame goes: right of the panel, centered, scaled by a
// whole number.
func (g *game) place(frame image.Point) (x, y, scale int) {
	panel := int(float64(panelWidth*g.uiScale()) + 16*g.displayScale)
	w, h := int(g.screenWidth)-panel, int(g.screenHeight)
	scale = g.scale
	if scale == 0 {
		scale = max(1, min(w/frame.X, h/frame.Y))
	}
	return panel + (w-frame.X*scale)/2, (h - frame.Y*scale) / 2, scale
}

func (g *game) uiScale() int {
	return max(1, int(math.Round(g.displayScale)))
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
