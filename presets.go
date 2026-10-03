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
