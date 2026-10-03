package kinescope

// Stage is where in the picture an effect works. Stages run in this order,
// which is the order the light of a real set goes through; within a stage,
// effects run in the order they are listed.
type Stage int

const (
	// StagePrepass effects work on whole frames before the picture is
	// drawn: the afterglow of the phosphor.
	StagePrepass Stage = iota
	// StageGeometry effects move the point of the frame a screen point shows.
	StageGeometry
	// StageSample effects change how the frame is read at that point.
	StageSample
	// StageLight effects add light before the beam draws it: glow, glass.
	StageLight
	// StageBeam effects shape the beam: scanlines.
	StageBeam
	// StageMask effects are the phosphor mask on the screen's own pixels.
	StageMask
	// StagePost effects are the signal's noise and the tube's falloff.
	StagePost
	// StageFrame effects cut the picture to the tube's outline.
	StageFrame
)

// Effect is one part of the look of a set: plain data, its parameters'
// base values. A TV owns its effects; what changes them over time —
// signals, drifts, glitches — is set up beside them, in Setup.
//
// The set of effects is closed: every effect is defined in this package,
// and every backend knows how to draw each of them.
type Effect interface {
	// Name is the effect's name, the first part of its params' keys.
	Name() string
	// Stage is where in the picture the effect works.
	Stage() Stage
	// Params lists the effect's tunable numbers.
	Params() []Param

	// clone copies the effect, so a TV never shares one with its caller.
	clone() Effect
}

// Effects returns a fresh copy of every effect there is, set to its
// defaults, in stage order. A lab lists them; a backend checks it can draw
// each one.
func Effects() []Effect {
	return []Effect{
		NewAfterglow(),
		NewCurvature(),
		NewTear(),
		NewConvergence(),
		NewGlow(),
		NewGlass(),
		NewScanlines(),
		NewApertureMask(),
		NewGrain(),
		NewFlicker(),
		NewHum(),
		NewVignette(),
		NewCorners(),
	}
}
