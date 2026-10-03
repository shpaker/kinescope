package kinescope

// Keys of the picture effects' params: the phosphor, the guns, the light
// and the beam.
const (
	AfterglowDecay       ParamKey = "afterglow.decay"
	ConvergenceAmount    ParamKey = "convergence.amount"
	GlowThreshold        ParamKey = "glow.threshold"
	GlowStrength         ParamKey = "glow.strength"
	GlowRadius           ParamKey = "glow.radius"
	GlassLevel           ParamKey = "glass.level"
	ScanlinesDepth       ParamKey = "scanlines.depth"
	ScanlinesMinScale    ParamKey = "scanlines.min_scale"
	ApertureMaskStrength ParamKey = "aperture_mask.strength"
	ApertureMaskMinScale ParamKey = "aperture_mask.min_scale"
)

var (
	_ Effect = (*Afterglow)(nil)
	_ Effect = (*Convergence)(nil)
	_ Effect = (*Glow)(nil)
	_ Effect = (*Glass)(nil)
	_ Effect = (*Scanlines)(nil)
	_ Effect = (*ApertureMask)(nil)
)

// Afterglow keeps the phosphor glowing after the beam has passed: bright
// moving things leave a short trail, the blue one longest.
type Afterglow struct {
	Decay float32 // the share of the glow kept from one frame to the next
}

// NewAfterglow is a short trail.
func NewAfterglow() *Afterglow { return &Afterglow{Decay: 0.6} }

func (*Afterglow) Name() string { return "afterglow" }
func (*Afterglow) Stage() Stage { return StagePrepass }
func (e *Afterglow) Params() []Param {
	return []Param{{Key: AfterglowDecay, Min: 0, Max: 0.95, Value: &e.Decay}}
}
func (e *Afterglow) clone() Effect { c := *e; return &c }

// Convergence puts the red and blue guns out of register: their beams part
// towards the edges of the screen.
type Convergence struct {
	Amount float32 // the parting at the edges, in frame pixels
}

// NewConvergence is a set a little out of register.
func NewConvergence() *Convergence { return &Convergence{Amount: 0.45} }

func (*Convergence) Name() string { return "convergence" }
func (*Convergence) Stage() Stage { return StageSample }
func (e *Convergence) Params() []Param {
	return []Param{{Key: ConvergenceAmount, Min: 0, Max: 3, Value: &e.Amount}}
}
func (e *Convergence) clone() Effect { c := *e; return &c }

// Glow bleeds the light of bright areas around them.
type Glow struct {
	Threshold float32 // the brightness the glow starts at
	Strength  float32 // how much of the glow is added
	Radius    float32 // how far it spreads, in frame pixels
}

// NewGlow is a soft glow around the brightest areas.
func NewGlow() *Glow { return &Glow{Threshold: 0.65, Strength: 0.9, Radius: 6} }

func (*Glow) Name() string { return "glow" }
func (*Glow) Stage() Stage { return StageLight }
func (e *Glow) Params() []Param {
	return []Param{
		{Key: GlowThreshold, Min: 0, Max: 0.99, Value: &e.Threshold},
		{Key: GlowStrength, Min: 0, Max: 3, Value: &e.Strength},
		{Key: GlowRadius, Min: 1, Max: 12, Value: &e.Radius},
	}
}
func (e *Glow) clone() Effect { c := *e; return &c }

// Glass is the tube's glass catching the room's light: black is never quite
// black, and a soft reflection sits in the upper left corner.
type Glass struct {
	Level float32 // the glass's own light on black
}

// NewGlass is a faint glass, enough to tell the tube from the black around
// it in a dark scene.
func NewGlass() *Glass { return &Glass{Level: 0.045} }

func (*Glass) Name() string { return "glass" }
func (*Glass) Stage() Stage { return StageLight }
func (e *Glass) Params() []Param {
	return []Param{{Key: GlassLevel, Min: 0, Max: 0.3, Value: &e.Level}}
}
func (e *Glass) clone() Effect { c := *e; return &c }

// Scanlines are the beam's lines, one per frame row: dark gaps between
// them, narrower where the beam is bright and wide.
type Scanlines struct {
	Depth float32 // how dark the gaps get
	// MinScale is the screen pixels per frame pixel the lines need: below
	// it they fade out, as too few pixels turn them into moiré.
	MinScale float32
}

// NewScanlines are clear lines on a big enough screen.
func NewScanlines() *Scanlines { return &Scanlines{Depth: 0.3, MinScale: 3} }

func (*Scanlines) Name() string { return "scanlines" }
func (*Scanlines) Stage() Stage { return StageBeam }
func (e *Scanlines) Params() []Param {
	return []Param{
		{Key: ScanlinesDepth, Min: 0, Max: 1, Value: &e.Depth},
		{Key: ScanlinesMinScale, Min: 1, Max: 8, Value: &e.MinScale},
	}
}
func (e *Scanlines) clone() Effect { c := *e; return &c }

// ApertureMask is the phosphor's red, green and blue stripes, one triad per
// three screen pixels.
type ApertureMask struct {
	Strength float32 // how much the stripes show
	// MinScale is the screen pixels per frame pixel the mask needs: below
	// it the mask fades out.
	MinScale float32
}

// NewApertureMask is a faint mask on a big enough screen.
func NewApertureMask() *ApertureMask {
	return &ApertureMask{Strength: 0.3, MinScale: 3}
}

func (*ApertureMask) Name() string { return "aperture_mask" }
func (*ApertureMask) Stage() Stage { return StageMask }
func (e *ApertureMask) Params() []Param {
	return []Param{
		{Key: ApertureMaskStrength, Min: 0, Max: 1, Value: &e.Strength},
		{Key: ApertureMaskMinScale, Min: 1, Max: 8, Value: &e.MinScale},
	}
}
func (e *ApertureMask) clone() Effect { c := *e; return &c }
