package kinescope

// Keys of the picture effects' params: the phosphor, the guns, the light
// and the beam.
const (
	AfterglowDecay       ParamKey = "afterglow.decay"
	ConvergenceAmount    ParamKey = "convergence.amount"
	ConvergenceOffset    ParamKey = "convergence.offset"
	GlowThreshold        ParamKey = "glow.threshold"
	GlowStrength         ParamKey = "glow.strength"
	GlowRadius           ParamKey = "glow.radius"
	GlassLevel           ParamKey = "glass.level"
	ScanlinesDepth       ParamKey = "scanlines.depth"
	ScanlinesMinScale    ParamKey = "scanlines.min_scale"
	ApertureMaskStrength ParamKey = "aperture_mask.strength"
	ApertureMaskMinScale ParamKey = "aperture_mask.min_scale"
	SoftnessAmount       ParamKey = "softness.amount"
	InterlaceStrength    ParamKey = "interlace.strength"
	InterlaceMinScale    ParamKey = "interlace.min_scale"
	SlotMaskStrength     ParamKey = "slot_mask.strength"
	SlotMaskMinScale     ParamKey = "slot_mask.min_scale"
)

var (
	_ Effect = (*Afterglow)(nil)
	_ Effect = (*Convergence)(nil)
	_ Effect = (*Glow)(nil)
	_ Effect = (*Glass)(nil)
	_ Effect = (*Scanlines)(nil)
	_ Effect = (*ApertureMask)(nil)
	_ Effect = (*Softness)(nil)
	_ Effect = (*Interlace)(nil)
	_ Effect = (*SlotMask)(nil)
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
// towards the edges of the screen, and on a worn set drift apart sideways
// all over it.
type Convergence struct {
	Amount float32 // the parting at the edges, outwards, in frame pixels
	// Offset is the sideways drift, red to the right and blue to the left:
	// half of it in the middle, all of it at the edges, in frame pixels.
	Offset float32
}

// NewConvergence is a set a little out of register.
func NewConvergence() *Convergence { return &Convergence{Amount: 0.45} }

func (*Convergence) Name() string { return "convergence" }
func (*Convergence) Stage() Stage { return StageSample }
func (e *Convergence) Params() []Param {
	return []Param{
		{Key: ConvergenceAmount, Min: 0, Max: 3, Value: &e.Amount},
		{Key: ConvergenceOffset, Min: 0, Max: 3, Value: &e.Offset},
	}
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

// Softness spreads the beam between neighboring pixels of a line: 0 keeps
// them square, 1 blends them smoothly. It shows on a screen scaled by a
// fraction, where square pixels come out uneven.
type Softness struct {
	Amount float32
}

// NewSoftness is a beam a little soft.
func NewSoftness() *Softness { return &Softness{Amount: 0.35} }

func (*Softness) Name() string { return "softness" }
func (*Softness) Stage() Stage { return StageRead }
func (e *Softness) Params() []Param {
	return []Param{{Key: SoftnessAmount, Min: 0, Max: 1, Value: &e.Amount}}
}
func (e *Softness) clone() Effect { c := *e; return &c }

// Interlace draws the lines in two fields taking turns, frame by frame:
// the lines of the field not drawn fade, and the picture shimmers.
type Interlace struct {
	Strength float32
	// MinScale is the screen pixels per frame pixel the fields need: below
	// it they fade out.
	MinScale float32
}

// NewInterlace is a faint shimmer.
func NewInterlace() *Interlace { return &Interlace{Strength: 0.3, MinScale: 2} }

func (*Interlace) Name() string { return "interlace" }
func (*Interlace) Stage() Stage { return StageBeam }
func (e *Interlace) Params() []Param {
	return []Param{
		{Key: InterlaceStrength, Min: 0, Max: 1, Value: &e.Strength},
		{Key: InterlaceMinScale, Min: 1, Max: 8, Value: &e.MinScale},
	}
}
func (e *Interlace) clone() Effect { c := *e; return &c }

// SlotMask is the phosphor of a shadow-mask set: red, green and blue
// stripes cut by slots, each column of triads half a slot below the last.
type SlotMask struct {
	Strength float32
	// MinScale is the screen pixels per frame pixel the mask needs: below
	// it the mask fades out.
	MinScale float32
}

// NewSlotMask is a mask that shows on a big enough screen.
func NewSlotMask() *SlotMask { return &SlotMask{Strength: 0.35, MinScale: 3} }

func (*SlotMask) Name() string { return "slot_mask" }
func (*SlotMask) Stage() Stage { return StageMask }
func (e *SlotMask) Params() []Param {
	return []Param{
		{Key: SlotMaskStrength, Min: 0, Max: 1, Value: &e.Strength},
		{Key: SlotMaskMinScale, Min: 1, Max: 8, Value: &e.MinScale},
	}
}
func (e *SlotMask) clone() Effect { c := *e; return &c }
