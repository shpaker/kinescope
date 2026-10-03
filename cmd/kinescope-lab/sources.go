package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// source is a picture to show through the TV: a frame, maybe moving.
type source interface {
	// frame is the picture now.
	frame() *ebiten.Image
	// update moves the picture on by a tick.
	update()
}

// The frame size of the drawn sources: the tanks' screen
const frameW, frameH = 256, 224

// testCard is a still test card: a grid on grey, a circle, color bars,
// a grey ramp and fine stripes — everything a tube's flaws show on.
type testCard struct {
	img *ebiten.Image
}

func newTestCard() *testCard {
	img := ebiten.NewImage(frameW, frameH)
	img.Fill(color.RGBA{0x60, 0x60, 0x60, 0xff})
	white := color.RGBA{0xf0, 0xf0, 0xf0, 0xff}
	black := color.RGBA{0x08, 0x08, 0x08, 0xff}

	// Grid
	for x := float32(8); x < frameW; x += 16 {
		vector.StrokeLine(img, x, 0, x, frameH, 1, white, false)
	}
	for y := float32(8); y < frameH; y += 16 {
		vector.StrokeLine(img, 0, y, frameW, y, 1, white, false)
	}

	// Circle
	cx, cy := float32(frameW/2), float32(frameH/2)
	vector.FillCircle(img, cx, cy, 88, black, true)
	vector.StrokeCircle(img, cx, cy, 88, 2, white, true)

	// Color bars
	bars := []color.RGBA{
		{0xf0, 0xf0, 0xf0, 0xff}, {0xf0, 0xf0, 0x10, 0xff}, {0x10, 0xf0, 0xf0, 0xff},
		{0x10, 0xf0, 0x10, 0xff}, {0xf0, 0x10, 0xf0, 0xff}, {0xf0, 0x10, 0x10, 0xff},
		{0x10, 0x10, 0xf0, 0xff}, {0x10, 0x10, 0x10, 0xff},
	}
	const barW, barTop, barH = 20, 58, 40
	left := cx - barW*float32(len(bars))/2
	for i, c := range bars {
		vector.FillRect(img, left+float32(i*barW), barTop, barW, barH, c, false)
	}

	// Grey ramp
	for i := range 10 {
		v := uint8(i * 255 / 9)
		vector.FillRect(img, left+float32(i*16), barTop+barH, 16, 24, color.RGBA{v, v, v, 0xff}, false)
	}

	// Fine stripes: one, two and three pixels wide
	top := float32(barTop + barH + 24)
	for i := range 3 {
		w := float32(i + 1)
		x0 := left + float32(i)*54
		for x := x0; x < x0+48; x += 2 * w {
			vector.FillRect(img, x, top, w, 26, white, false)
		}
	}

	// Black and white boxes in the corners
	for _, p := range []image.Point{{8, 8}, {frameW - 40, 8}, {8, frameH - 40}, {frameW - 40, frameH - 40}} {
		vector.FillRect(img, float32(p.X), float32(p.Y), 32, 32, black, false)
		vector.FillRect(img, float32(p.X+8), float32(p.Y+8), 16, 16, white, false)
	}
	return &testCard{img: img}
}

func (c *testCard) frame() *ebiten.Image { return c.img }
func (c *testCard) update()              {}

// scene is a dark field with bright things moving over it: tracers, a
// bouncing block and a blast now and then — what afterglow and glow show on.
type scene struct {
	img   *ebiten.Image
	ticks int
}

func newScene() *scene {
	return &scene{img: ebiten.NewImage(frameW, frameH)}
}

func (s *scene) frame() *ebiten.Image { return s.img }

func (s *scene) update() {
	s.ticks++
	t := float64(s.ticks) / 60
	s.img.Fill(color.RGBA{0x0a, 0x0a, 0x10, 0xff})

	// Walls
	brick := color.RGBA{0x7a, 0x34, 0x1c, 0xff}
	for y := 24; y < frameH-24; y += 32 {
		for x := 16; x < frameW-16; x += 48 {
			vector.FillRect(s.img, float32(x), float32(y), 24, 12, brick, false)
		}
	}

	// Tracers
	for i := range 6 {
		y := float32(20 + i*34)
		x := float32(math.Mod(t*float64(90+i*25)+float64(i*40), frameW+40)) - 20
		vector.FillRect(s.img, x, y, 4, 2, color.RGBA{0xff, 0xf0, 0xa0, 0xff}, false)
	}

	// A bouncing block
	bx := frameW/2 + 90*math.Sin(t*1.3)
	by := frameH/2 + 70*math.Sin(t*1.9)
	vector.FillRect(s.img, float32(bx)-8, float32(by)-8, 16, 16, color.RGBA{0x40, 0xd0, 0xff, 0xff}, false)

	// A blast every three seconds
	if k := math.Mod(t, 3); k < 0.5 {
		r := float32(4 + k*40)
		a := uint8(255 * (1 - k/0.5))
		vector.FillCircle(s.img, 190, 60, r, color.RGBA{a, uint8(int(a) * 3 / 4), a / 4, a}, true)
	}
}

// picture is an image dropped onto the lab: a game's screenshot, say.
type picture struct {
	img *ebiten.Image
}

func (p *picture) frame() *ebiten.Image { return p.img }
func (p *picture) update()              {}
