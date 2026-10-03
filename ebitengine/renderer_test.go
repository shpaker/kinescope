package ebitengine

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope"
)

const frameW, frameH = 200, 150

// coordinateFrame is a frame whose every pixel tells where it is: red is
// its x, green its y, blue full.
func coordinateFrame() *ebiten.Image {
	frame := ebiten.NewImage(frameW, frameH)
	pixels := make([]byte, 4*frameW*frameH)
	for y := range frameH {
		for x := range frameW {
			i := 4 * (y*frameW + x)
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = byte(x), byte(y), 255, 255
		}
	}
	frame.WritePixels(pixels)
	return frame
}

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func draw(t *testing.T, r *Renderer, tv *kinescope.TV, frame *ebiten.Image, scale int) []byte {
	t.Helper()
	b := frame.Bounds()
	dst := ebiten.NewImage(b.Dx()*scale, b.Dy()*scale)
	var geoM ebiten.GeoM
	geoM.Scale(float64(scale), float64(scale))
	if err := r.Draw(dst, frame, tv, geoM); err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, 4*dst.Bounds().Dx()*dst.Bounds().Dy())
	dst.ReadPixels(pixels)
	return pixels
}

// The shader's geometry and TV.Map must agree: a pointer mapped by Map
// lands on the pixel the screen shows under it.
func TestShaderGeometryMatchesMap(t *testing.T) {
	geometryMatchesMap(t, func(blue byte) bool { return blue == 255 },
		&kinescope.Power{Width: 0.8, Height: 0.9},
		&kinescope.Curvature{X: 0.1, Y: 0.15},
		&kinescope.Degauss{Strength: 0.6},
		&kinescope.Tear{Strength: 1, Amplitude: 3},
	)
}

func TestRollMatchesMap(t *testing.T) {
	// The blanking bar darkens the frame: any blue is the frame
	geometryMatchesMap(t, func(blue byte) bool { return blue > 0 },
		&kinescope.Roll{Strength: 0.3})
}

// geometryMatchesMap draws the coordinate frame through effects and checks
// every screen pixel against Map; onFrame tells a frame pixel by its blue.
func geometryMatchesMap(
	t *testing.T,
	onFrame func(blue byte) bool,
	effects ...kinescope.Effect,
) {
	t.Helper()
	tv, err := kinescope.NewTV(kinescope.Setup{Effects: effects})
	if err != nil {
		t.Fatal(err)
	}
	for range 7 {
		tv.Update(1.0 / 60)
	}
	pixels := draw(t, newRenderer(t), tv, coordinateFrame(), 1)

	misses := 0
	for y := range frameH {
		for x := range frameW {
			mx, my := tv.Map(float64(x)+0.5, float64(y)+0.5, frameW, frameH)
			// Right at the picture's edge, rounding and the roll's soft
			// blanking bar decide
			if nearEdge(mx, my) {
				continue
			}
			i := 4 * (y*frameW + x)
			inside := onFrame(pixels[i+2])
			wantInside := mx >= 0 && my >= 0 && mx < frameW && my < frameH
			if inside != wantInside {
				t.Fatalf("(%d, %d): inside %v, Map says %v (%.2f, %.2f)", x, y, inside, wantInside, mx, my)
			}
			if !inside {
				continue
			}
			dx := math.Abs(float64(pixels[i]) - math.Floor(mx))
			dy := math.Abs(float64(pixels[i+1]) - math.Floor(my))
			if dx > 1 || dy > 1 {
				t.Fatalf("(%d, %d) shows (%d, %d), Map says (%.2f, %.2f)", x, y, pixels[i], pixels[i+1], mx, my)
			}
			if dx+dy > 0 {
				misses++
			}
		}
	}
	// A pixel off happens only where float precision rounds across a
	// pixel's edge
	if misses > frameW*frameH/100 {
		t.Errorf("%d pixels a pixel off, more than 1%%", misses)
	}
}

func nearEdge(x, y float64) bool {
	const e = 1.5
	// The roll's blanking bar darkens the top and bottom twentieth
	const bar = 0.05 * frameH
	return math.Abs(x) < e || math.Abs(x-frameW) < e ||
		math.Abs(y) < bar || math.Abs(y-frameH) < bar
}

func TestGorizontDrawsAPicture(t *testing.T) {
	tv, err := kinescope.NewTV(kinescope.Gorizont())
	if err != nil {
		t.Fatal(err)
	}
	frame := ebiten.NewImage(frameW, frameH)
	frame.Fill(color.Gray{Y: 128})
	const scale = 3
	pixels := draw(t, newRenderer(t), tv, frame, scale)

	w := frameW * scale
	at := func(x, y int) (byte, byte, byte) {
		i := 4 * (y*w + x)
		return pixels[i], pixels[i+1], pixels[i+2]
	}
	if r, g, b := at(0, 0); r+g+b != 0 {
		t.Errorf("corner (%d, %d, %d), want black: rounded off", r, g, b)
	}
	r, g, b := at(w/2, frameH*scale/2)
	if r == 0 && g == 0 && b == 0 {
		t.Error("center is black")
	}
}

func TestRendererFollowsSetEffects(t *testing.T) {
	tv, err := kinescope.NewTV(kinescope.Setup{
		Effects: []kinescope.Effect{&kinescope.Corners{Radius: 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	frame := ebiten.NewImage(frameW, frameH)
	frame.Fill(color.White)
	r := newRenderer(t)
	if pixels := draw(t, r, tv, frame, 1); pixels[0] != 0 {
		t.Fatal("corner not rounded off")
	}
	if err := tv.SetEffects(); err != nil {
		t.Fatal(err)
	}
	if pixels := draw(t, r, tv, frame, 1); pixels[0] != 255 {
		t.Error("corner still rounded off after the effects were removed")
	}
}
