package kinescope

// ParamKey names a tunable number of a TV: a parameter of one of its effects
// ("tear.strength") or the rate of one of its schedules ("rate.glitches").
// Values, Apply, drives and episode targets all speak these keys.
type ParamKey string

// Param is one tunable number: its key, its range and where its base value
// lives. Value points into the effect (or the TV, for a schedule's rate), so
// writing through it changes the base the modulation is added to.
type Param struct {
	Key   ParamKey
	Min   float32
	Max   float32
	Value *float32
}

// ratePrefix starts the key of every schedule's rate.
const ratePrefix = "rate."

// Rate is the key of a schedule's rate: 1 keeps its pace, 2 doubles it, 0
// stops it. Drive it to make glitches come thicker or thinner.
func Rate(schedule string) ParamKey { return ParamKey(ratePrefix + schedule) }

// maxRate caps a schedule's rate.
const maxRate = 10

// clamp keeps v within the param's range.
func (p Param) clamp(v float32) float32 {
	return min(max(v, p.Min), p.Max)
}
