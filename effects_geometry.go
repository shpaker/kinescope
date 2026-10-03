package kinescope

import "math"

// Keys of the geometry effects' params.
const (
	CabinetMargin   ParamKey = "cabinet.margin"
	CabinetRadius   ParamKey = "cabinet.radius"
	PowerWidth      ParamKey = "power.width"
	PowerHeight     ParamKey = "power.height"
	PowerFlash      ParamKey = "power.flash"
	CurvatureX      ParamKey = "curvature.x"
	CurvatureY      ParamKey = "curvature.y"
	DegaussStrength ParamKey = "degauss.strength"
	RollStrength    ParamKey = "roll.strength"
	TearStrength    ParamKey = "tear.strength"
	TearAmplitude   ParamKey = "tear.amplitude"
	TearBands       ParamKey = "tear.bands"
)

var (
	_ Effect = (*Cabinet)(nil)
	_ Effect = (*Power)(nil)
	_ Effect = (*Curvature)(nil)
	_ Effect = (*Degauss)(nil)
	_ Effect = (*Roll)(nil)
	_ Effect = (*Tear)(nil)
	_ warper = (*Cabinet)(nil)
	_ warper = (*Power)(nil)
	_ warper = (*Curvature)(nil)
	_ warper = (*Degauss)(nil)
	_ warper = (*Roll)(nil)
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

// inPicture is p in the picture's own coordinates: -1 to 1 edge to edge.
func inPicture(p point, size point) point {
	return point{p.x/size.x*2 - 1, p.y/size.y*2 - 1}
}

// onFrame is the inverse of inPicture.
func onFrame(u point, size point) point {
	return point{(u.x + 1) / 2 * size.x, (u.y + 1) / 2 * size.y}
}

// Cabinet puts the picture into a monitor of the day: the picture shrinks
// within the frame's place, and the tube's dark glass, the beige plastic
// case and the dim room behind show around it. Its margin usually rests at
// zero and is driven, say by a signal of full screen, where there is room
// for the case.
type Cabinet struct {
	Margin float32 // the case's width on each side, in the picture's half-heights
	Radius float32 // the rounding of the glass's corners, in frame pixels
}

// NewCabinet is a case as wide as a seventh of the picture's height.
func NewCabinet() *Cabinet { return &Cabinet{Margin: 0.15, Radius: 16} }

func (*Cabinet) Name() string { return "cabinet" }
func (*Cabinet) Stage() Stage { return StageGeometry }
func (e *Cabinet) Params() []Param {
	return []Param{
		{Key: CabinetMargin, Min: 0, Max: 0.5, Value: &e.Margin},
		{Key: CabinetRadius, Min: 0, Max: 64, Value: &e.Radius},
	}
}
func (e *Cabinet) clone() Effect { c := *e; return &c }

func (*Cabinet) warp(
	p point,
	size point,
	value func(ParamKey) float32,
	_ float64,
) point {
	grow := 1 + float64(value(CabinetMargin))
	return point{
		size.x/2 + (p.x-size.x/2)*grow,
		size.y/2 + (p.y-size.y/2)*grow,
	}
}

// Power squeezes the raster as the tube warms up or dies — a dot, a line,
// the picture — and flashes the beam. The TV drives it on PowerOn and
// PowerOff; at rest it is the whole picture.
type Power struct {
	Width  float32 // the raster's width, 1 whole
	Height float32 // the raster's height, 1 whole
	Flash  float32 // the beam's flash added to the picture
}

// NewPower is a tube at rest: the whole picture, no flash.
func NewPower() *Power { return &Power{Width: 1, Height: 1} }

func (*Power) Name() string { return "power" }
func (*Power) Stage() Stage { return StageGeometry }
func (e *Power) Params() []Param {
	return []Param{
		{Key: PowerWidth, Min: 0, Max: 1, Value: &e.Width},
		{Key: PowerHeight, Min: 0, Max: 1, Value: &e.Height},
		{Key: PowerFlash, Min: 0, Max: 2, Value: &e.Flash},
	}
}
func (e *Power) clone() Effect { c := *e; return &c }

// powerFloor keeps the squeezed raster from dividing by zero.
const powerFloor = 0.001

func (*Power) warp(
	p point,
	size point,
	value func(ParamKey) float32,
	_ float64,
) point {
	w := math.Max(float64(value(PowerWidth)), powerFloor)
	h := math.Max(float64(value(PowerHeight)), powerFloor)
	return point{
		size.x/2 + (p.x-size.x/2)/w,
		size.y/2 + (p.y-size.y/2)/h,
	}
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
	u := inPicture(p, size)
	return onFrame(point{
		u.x * (1 + u.y*u.y*float64(value(CurvatureX))),
		u.y * (1 + u.x*u.x*float64(value(CurvatureY))),
	}, size)
}

// Degauss is the degaussing coil shaking the mask: the picture wobbles and
// settles. Its strength rests at zero and is driven by the Degauss episode.
type Degauss struct {
	Strength float32
}

// NewDegauss is a coil at rest.
func NewDegauss() *Degauss { return &Degauss{} }

func (*Degauss) Name() string { return "degauss" }
func (*Degauss) Stage() Stage { return StageGeometry }
func (e *Degauss) Params() []Param {
	return []Param{{Key: DegaussStrength, Min: 0, Max: 1, Value: &e.Strength}}
}
func (e *Degauss) clone() Effect { c := *e; return &c }

// The wobble of the coil: across and down, along and in time. They match
// the fragment in the Ebitengine backend.
const (
	degaussAcross      = 0.02
	degaussAcrossAlong = 18.0
	degaussAcrossSpeed = 30.0
	degaussDown        = 0.01
	degaussDownAlong   = 14.0
	degaussDownSpeed   = 23.0
)

func (*Degauss) warp(
	p point,
	size point,
	value func(ParamKey) float32,
	time float64,
) point {
	s := float64(value(DegaussStrength))
	u := inPicture(p, size)
	u.x += s * degaussAcross *
		math.Sin(u.y*degaussAcrossAlong+time*degaussAcrossSpeed)
	u.y += s * degaussDown *
		math.Sin(u.x*degaussDownAlong+time*degaussDownSpeed)
	return onFrame(u, size)
}

// Roll slips the picture down as the vertical sync is lost: Strength is the
// share of the height it has slipped, the blanking bar crossing with it.
// It rests at zero and is driven by the Roll episode.
type Roll struct {
	Strength float32
}

// NewRoll is a picture held still.
func NewRoll() *Roll { return &Roll{} }

func (*Roll) Name() string { return "roll" }
func (*Roll) Stage() Stage { return StageGeometry }
func (e *Roll) Params() []Param {
	return []Param{{Key: RollStrength, Min: 0, Max: 1, Value: &e.Strength}}
}
func (e *Roll) clone() Effect { c := *e; return &c }

func (*Roll) warp(
	p point,
	size point,
	value func(ParamKey) float32,
	_ float64,
) point {
	s := float64(value(RollStrength))
	if s <= 0 {
		return p
	}
	y := math.Mod(p.y+s*size.y, size.y)
	if y < 0 {
		y += size.y
	}
	return point{p.x, y}
}

// Tear shifts the lines sideways, as when the set is knocked or the
// horizontal sync slips: in a running wave, and in bands of lines torn at
// random. Its strength usually rests at zero and is driven: by a signal
// such as a shake, or by a glitch.
type Tear struct {
	Strength  float32 // 0 calm, 1 fully torn
	Amplitude float32 // the wave's shift of a line at full strength, in frame pixels
	// Bands is the shift of the bands torn at random, in frame pixels. They
	// last a moment, so Map leaves them out.
	Bands float32
}

// NewTear is a calm tear that shifts lines by a pixel and a half at full
// strength, without bands.
func NewTear() *Tear { return &Tear{Amplitude: 1.5} }

func (*Tear) Name() string { return "tear" }
func (*Tear) Stage() Stage { return StageGeometry }
func (e *Tear) Params() []Param {
	return []Param{
		{Key: TearStrength, Min: 0, Max: 1, Value: &e.Strength},
		{Key: TearAmplitude, Min: 0, Max: 8, Value: &e.Amplitude},
		{Key: TearBands, Min: 0, Max: 32, Value: &e.Bands},
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
