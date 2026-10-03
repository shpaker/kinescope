package kinescope

// Gorizont is a Minsk color set of the eighties, well kept: a slightly
// bulging tube with clear scanlines, a faint mask, glowing highlights and a
// short afterglow. It has no moods of its own; add sources, drives and
// schedules to the returned setup to give it some.
func Gorizont() Setup {
	return Setup{
		Effects: []Effect{
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
		},
	}
}

// Rubin is a Moscow color set of the eighties, well worn: a bulging tube,
// a soft beam through a slot mask, a dark hum rolling through, plenty of
// snow — and a glitch every minute or two.
func Rubin() Setup {
	return Setup{
		Effects: []Effect{
			NewPower(),
			&Curvature{X: 0.045, Y: 0.06},
			NewDegauss(),
			NewRoll(),
			&Tear{Amplitude: 2, Bands: 18},
			&Softness{Amount: 0.35},
			&Convergence{Offset: 0.73},
			&Glow{Threshold: 0, Strength: 0.25, Radius: 4},
			&Scanlines{Depth: 0.45, MinScale: 2},
			&Interlace{Strength: 0, MinScale: 2},
			&SlotMask{Strength: 0.35, MinScale: 3},
			&Grain{Strength: 0.05},
			NewSnow(),
			&Flicker{Strength: 0.0075, Frequency: 10},
			&Hum{Strength: -0.13, Period: 14, Width: 0.3},
			&Vignette{Strength: 0.7},
			&Corners{Radius: 15},
		},
		Schedules: map[string]Every{
			"glitches": {
				Mean:   90,
				Spread: 30,
				Episodes: []Episode{
					Jitter(), RollOver(), SnowBurst(), Degaussing(),
				},
			},
		},
	}
}

// Blend mixes two sets of values: t = 0 gives a, t = 1 gives b. A key found
// in only one of them keeps its value. Use it for a single knob a player
// turns from one preset to another:
//
//	tv.Apply(kinescope.Blend(knob, soft.Values(), worn.Values()))
func Blend(t float32, a, b map[ParamKey]float32) map[ParamKey]float32 {
	values := make(map[ParamKey]float32, max(len(a), len(b)))
	for key, va := range a {
		values[key] = va
		if vb, ok := b[key]; ok {
			values[key] = va + (vb-va)*t
		}
	}
	for key, vb := range b {
		if _, ok := a[key]; !ok {
			values[key] = vb
		}
	}
	return values
}
