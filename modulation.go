package kinescope

import (
	"hash/fnv"
	"math"
)

// Source is a value that changes in time, from 0 to 1, and drives params
// through Drives. Sources are named in Setup.Sources.
type Source interface {
	source()
}

// Signal is a level the game sets, frame by frame: a fact of the game, such
// as the set being shaken. Get its Level with TV.Signal.
type Signal struct{}

// Drift wanders smoothly and at random between 0 and 1, like the reception
// of a far station: now clearer, now worse.
type Drift struct {
	Period float32 // seconds between one random level and the next
}

// Wave rises from 0 to 1 and falls back, again and again.
type Wave struct {
	Period float32 // seconds of a whole rise and fall
}

var (
	_ Source = Signal{}
	_ Source = Drift{}
	_ Source = Wave{}
)

func (Signal) source() {}
func (Drift) source()  {}
func (Wave) source()   {}

// Drive lets a source move a param: Weight times the source's value is
// added to the param's base. A drive to a param of an effect the TV does
// not have does nothing.
type Drive struct {
	From   string   // a source's name in Setup.Sources
	To     ParamKey // the param it moves
	Weight float32  // how far the param moves at the source's full value
}

// Level is the live level of a signal. The game sets it whenever it likes —
// in Update or in Draw — and the TV's values follow at once.
type Level struct {
	value float64
	tv    *TV
}

// Set sets the signal's level, kept within 0..1.
func (l *Level) Set(level float32) {
	l.value = math.Min(math.Max(float64(level), 0), 1)
	l.tv.stale = true
}

// Envelope is the shape of an episode in time, in seconds: the rise to its
// peak, the stay there and the fall back.
type Envelope struct {
	Attack  float32
	Hold    float32
	Release float32
}

// level is the envelope's level t seconds in, and whether it is over.
func (e Envelope) level(t float64) (float64, bool) {
	attack, hold, release := float64(e.Attack), float64(e.Hold), float64(e.Release)
	switch {
	case t < attack:
		return t / attack, false
	case t < attack+hold:
		return 1, false
	case t < attack+hold+release:
		return 1 - (t-attack-hold)/release, false
	}
	return 0, true
}

// Target is a param an episode moves, and how far at its peak.
type Target struct {
	Param  ParamKey
	Weight float32
}

// Episode is a fault of the set played over a moment: its targets move by
// their weights, shaped by the envelope. Play one with TV.Play or let a
// schedule (Every) play it.
type Episode struct {
	Name     string
	Envelope Envelope
	Targets  []Target
}

// Jitter tears the lines for a moment, as when the horizontal sync slips.
func Jitter() Episode {
	return Episode{
		Name:     "jitter",
		Envelope: Envelope{Release: 0.4},
		Targets:  []Target{{Param: TearStrength, Weight: 1}},
	}
}

// Ripple is the signal shivering as a channel is switched: the lines tear
// a little and the grain thickens, both dying out.
func Ripple() Episode {
	return Episode{
		Name:     "ripple",
		Envelope: Envelope{Release: 0.45},
		Targets: []Target{
			{Param: TearStrength, Weight: 0.8},
			{Param: GrainStrength, Weight: 0.15},
		},
	}
}

// sourceValue is a source's value at time t; seed makes a drift the TV's
// own. level is the signal's level, for a signal.
func sourceValue(s Source, level *Level, seed uint64, t float64) float64 {
	switch s := s.(type) {
	case Signal:
		return level.value
	case Drift:
		return drift(seed, t/float64(s.Period))
	case Wave:
		return 0.5 - 0.5*math.Cos(2*math.Pi*t/float64(s.Period))
	}
	return 0
}

// drift is smooth value noise at x: random levels at whole x, eased between.
func drift(seed uint64, x float64) float64 {
	i := math.Floor(x)
	f := x - i
	a := unit(seed, int64(i))
	b := unit(seed, int64(i)+1)
	return a + (b-a)*f*f*(3-2*f)
}

// unit is a random number in 0..1 for the seed and i, the same every time.
func unit(seed uint64, i int64) float64 {
	return float64(mix64(seed^uint64(i)*0x9e3779b97f4a7c15)>>11) / (1 << 53)
}

// mix64 is the SplitMix64 finalizer: it scatters the bits of x.
func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// nameSeed is the seed of a named source: the TV's seed mixed with the name,
// so adding a source leaves the others as they were.
func nameSeed(seed uint64, name string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return mix64(seed ^ h.Sum64())
}
