package ebitengine

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// game runs the tests inside Ebitengine's loop, where images can be drawn
// and read back.
type game struct {
	m    *testing.M
	code int
}

func (g *game) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*game) Draw(*ebiten.Image) {}

func (*game) Layout(int, int) (int, int) { return 320, 240 }

func TestMain(m *testing.M) {
	g := &game{m: m, code: 1}
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{InitUnfocused: true}); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}
