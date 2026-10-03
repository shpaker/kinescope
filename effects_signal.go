package kinescope

// Keys of the signal and tube effects' params.
const (
	GrainStrength    ParamKey = "grain.strength"
	FlickerStrength  ParamKey = "flicker.strength"
	FlickerFrequency ParamKey = "flicker.frequency"
	HumStrength      ParamKey = "hum.strength"
	HumPeriod        ParamKey = "hum.period"
	HumWidth         ParamKey = "hum.width"
	SnowStrength     ParamKey = "snow.strength"
	VignetteStrength ParamKey = "vignette.strength"
	CornersRadius    ParamKey = "corners.radius"
)

var (
	_ Effect = (*Grain)(nil)
	_ Effect = (*Flicker)(nil)
	_ Effect = (*Hum)(nil)
	_ Effect = (*Vignette)(nil)
	_ Effect = (*Corners)(nil)
	_ Effect = (*Snow)(nil)
)

// Grain is the signal's snow: fine noise, new every frame, seen on black
// too.
type Grain struct {
	Strength float32
}

// NewGrain is a faint grain.
func NewGrain() *Grain { return &Grain{Strength: 0.05} }

func (*Grain) Name() string { return "grain" }
func (*Grain) Stage() Stage { return StagePost }
func (e *Grain) Params() []Param {
	return []Param{{Key: GrainStrength, Min: 0, Max: 0.5, Value: &e.Strength}}
}
func (e *Grain) clone() Effect { c := *e; return &c }

// Flicker trembles the whole picture's brightness.
type Flicker struct {
	Strength  float32 // the share of brightness the picture trembles by
	Frequency float32 // trembles per second
}

// NewFlicker is a barely seen tremble.
func NewFlicker() *Flicker { return &Flicker{Strength: 0.0075, Frequency: 22} }

func (*Flicker) Name() string { return "flicker" }
func (*Flicker) Stage() Stage { return StagePost }
func (e *Flicker) Params() []Param {
	return []Param{
		{Key: FlickerStrength, Min: 0, Max: 0.2, Value: &e.Strength},
		{Key: FlickerFrequency, Min: 0.1, Max: 60, Value: &e.Frequency},
	}
}
func (e *Flicker) clone() Effect { c := *e; return &c }

// Hum is the mains hum: a bar rolling slowly down the picture, light or
// dark.
type Hum struct {
	Strength float32 // how much brighter the bar is; below zero, darker
	Period   float32 // seconds the bar takes to cross the picture
	Width    float32 // the bar's width, in the picture's heights
}

// NewHum is a faint thin light bar crossing every eight seconds.
func NewHum() *Hum { return &Hum{Strength: 0.025, Period: 8, Width: 0.05} }

func (*Hum) Name() string { return "hum" }
func (*Hum) Stage() Stage { return StagePost }
func (e *Hum) Params() []Param {
	return []Param{
		{Key: HumStrength, Min: -0.5, Max: 0.5, Value: &e.Strength},
		{Key: HumPeriod, Min: 0.5, Max: 60, Value: &e.Period},
		{Key: HumWidth, Min: 0.01, Max: 1, Value: &e.Width},
	}
}
func (e *Hum) clone() Effect { c := *e; return &c }

// Vignette dims the picture towards its corners.
type Vignette struct {
	Strength float32
}

// NewVignette is a gentle falloff.
func NewVignette() *Vignette { return &Vignette{Strength: 0.4} }

func (*Vignette) Name() string { return "vignette" }
func (*Vignette) Stage() Stage { return StagePost }
func (e *Vignette) Params() []Param {
	return []Param{{Key: VignetteStrength, Min: 0, Max: 1, Value: &e.Strength}}
}
func (e *Vignette) clone() Effect { c := *e; return &c }

// Corners round the picture's corners to the tube's outline.
type Corners struct {
	Radius float32 // in frame pixels
}

// NewCorners are softly rounded corners.
func NewCorners() *Corners { return &Corners{Radius: 12} }

func (*Corners) Name() string { return "corners" }
func (*Corners) Stage() Stage { return StageFrame }
func (e *Corners) Params() []Param {
	return []Param{{Key: CornersRadius, Min: 0, Max: 48, Value: &e.Radius}}
}
func (e *Corners) clone() Effect { c := *e; return &c }

// Snow is the screen drowning in snow, as when the signal is lost. Its
// strength rests at zero and is driven by the Snow episode.
type Snow struct {
	Strength float32
}

// NewSnow is a clear signal.
func NewSnow() *Snow { return &Snow{} }

func (*Snow) Name() string { return "snow" }
func (*Snow) Stage() Stage { return StagePost }
func (e *Snow) Params() []Param {
	return []Param{{Key: SnowStrength, Min: 0, Max: 1, Value: &e.Strength}}
}
func (e *Snow) clone() Effect { c := *e; return &c }
