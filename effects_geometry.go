package kinescope

import "math"

// Keys of the geometry effects' params.
const (
	CurvatureX    ParamKey = "curvature.x"
	CurvatureY    ParamKey = "curvature.y"
	TearStrength  ParamKey = "tear.strength"
	TearAmplitude ParamKey = "tear.amplitude"
)

var (
	_ Effect = (*Curvature)(nil)
	_ Effect = (*Tear)(nil)
	_ warper = (*Curvature)(nil)
	_ warper = (*Tear)(nil)
)

// point is a position on the frame, in frame pixels.
type point struct{ x, y float64 }

// warper is an effect that moves the point of the frame a screen point
// shows. warp is the CPU mirror of the effect's shader: the backend's
// fragment must give the same point, which the backend's tests check.
type warper interface {
	warp(p point, size point, value func(ParamKey) float32, time float64) point
}

// Curvature bulges the glass: the picture bows out towards the tube's
// corners.
type Curvature struct {
	X float32 // bulge across, the stronger the nearer the top and bottom
	Y float32 // bulge down, the stronger the nearer the sides
}

// NewCurvature is a slightly bulging tube.
func NewCurvature() *Curvature { return &Curvature{X: 0.035, Y: 0.045} }

func (*Curvature) Name() string { return "curvature" }
func (*Curvature) Stage() Stage { return StageGeometry }
func (e *Curvature) Params() []Param {
	return []Param{
		{Key: CurvatureX, Min: 0, Max: 0.25, Value: &e.X},
		{Key: CurvatureY, Min: 0, Max: 0.25, Value: &e.Y},
	}
}
func (e *Curvature) clone() Effect { c := *e; return &c }

func (*Curvature) warp(
	p point,
	size point,
	value func(ParamKey) float32,
	_ float64,
) point {
	ux := p.x/size.x*2 - 1
	uy := p.y/size.y*2 - 1
	wx := ux * (1 + uy*uy*float64(value(CurvatureX)))
	wy := uy * (1 + ux*ux*float64(value(CurvatureY)))
	return point{(wx + 1) / 2 * size.x, (wy + 1) / 2 * size.y}
}

// Tear shifts the lines sideways in a running wave, as when the set is
// knocked. Its strength usually rests at zero and is driven: by a signal
// such as a shake, or by a glitch.
type Tear struct {
	Strength  float32 // 0 calm, 1 fully torn
	Amplitude float32 // a line's shift at full strength, in frame pixels
}

// NewTear is a calm tear that shifts lines by a pixel and a half at full
// strength.
func NewTear() *Tear { return &Tear{Strength: 0, Amplitude: 1.5} }

func (*Tear) Name() string { return "tear" }
func (*Tear) Stage() Stage { return StageGeometry }
func (e *Tear) Params() []Param {
	return []Param{
		{Key: TearStrength, Min: 0, Max: 1, Value: &e.Strength},
		{Key: TearAmplitude, Min: 0, Max: 8, Value: &e.Amplitude},
	}
}
func (e *Tear) clone() Effect { c := *e; return &c }

// The tear's two waves: how fast they change along the lines and in time.
const (
	tearSlowAlong = 0.35
	tearSlowSpeed = 54.0
	tearFastAlong = 1.7
	tearFastSpeed = 126.0
)

func (*Tear) warp(
	p point,
	_ point,
	value func(ParamKey) float32,
	time float64,
) point {
	shift := float64(value(TearStrength) * value(TearAmplitude))
	wave := 0.7*math.Sin(p.y*tearSlowAlong+time*tearSlowSpeed) +
		0.3*math.Sin(p.y*tearFastAlong-time*tearFastSpeed)
	return point{p.x + shift*wave, p.y}
}
