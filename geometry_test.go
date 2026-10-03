package kinescope

import (
	"math"
	"testing"
)

func TestMapWithoutGeometryIsIdentity(t *testing.T) {
	tv := newTV(t, Setup{Effects: []Effect{NewGrain()}})
	if x, y := tv.Map(10.5, 20.25, 256, 224); x != 10.5 || y != 20.25 {
		t.Errorf("Map = %v, %v", x, y)
	}
}

func TestCurvatureBowsOutward(t *testing.T) {
	tv := newTV(t, Setup{Effects: []Effect{NewCurvature()}})
	const w, h = 256, 224

	if x, y := tv.Map(w/2, h/2, w, h); x != w/2 || y != h/2 {
		t.Errorf("center maps to %v, %v", x, y)
	}
	// Near a corner the screen shows a point further out on the frame
	x, y := tv.Map(10, 10, w, h)
	if x >= 10 || y >= 10 {
		t.Errorf("corner maps to %v, %v: want further out", x, y)
	}
	// The middle of an edge stays on its line
	if _, y := tv.Map(w/2, 0, w, h); y != 0 {
		t.Errorf("top middle maps to y %v", y)
	}
}

func TestTearShiftsLinesOnly(t *testing.T) {
	tv := newTV(t, Setup{
		Effects: []Effect{&Tear{Strength: 1, Amplitude: 2}},
	})
	tv.Update(tick)
	shifted := false
	for y := range 20 {
		x, my := tv.Map(100, float64(y), 256, 224)
		if my != float64(y) {
			t.Fatalf("line %d moved to %v", y, my)
		}
		if math.Abs(x-100) > 2 {
			t.Fatalf("line %d shifted by %v, past the amplitude", y, x-100)
		}
		shifted = shifted || x != 100
	}
	if !shifted {
		t.Error("no line shifted")
	}
}
