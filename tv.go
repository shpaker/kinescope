package kinescope

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
)

// timeWrap is the period, in seconds, Time wraps around at: shaders keep
// their precision, at the cost of one jump an hour.
const timeWrap = 3600

// TV is a set at work: its effects, its clock and its moods. The game talks
// to it in facts — a signal's level, an episode to play — and the TV works
// out what each effect shows. A backend draws it.
//
// A TV is not safe for concurrent use.
type TV struct {
	seed   uint64
	random *rand.Rand

	// Effects in stage order, and every param: the effects' first, then
	// the schedules' rates
	effects []Effect
	params  []Param
	index   map[ParamKey]int
	// mods are what modulation adds to each param's base, by index; stale
	// when a signal has changed since they were worked out
	mods  []float32
	stale bool

	// Rates of the schedules, the bases their params point at
	rates map[string]*float32

	// Modulation
	sources   []source
	drives    []drive
	schedules []*schedule
	playing   []episode
	held      bool
	power     power

	// Clock
	time   float64
	frames int

	// revision counts changes of the set of effects, epoch the resets
	revision int
	epoch    int
}

// source is a named source at work.
type source struct {
	name   string
	source Source
	seed   uint64
	level  *Level
	value  float64
}

// drive is a Drive at work, its source found.
type drive struct {
	from   int // index of the source
	to     ParamKey
	weight float32
}

// episode is an episode playing, t seconds in.
type episode struct {
	episode Episode
	t       float64
}

// NewTV builds a TV from a setup. The TV takes copies of the setup's
// effects, so the setup can be reused. It fails on a setup that does not
// hold together: a drive from a source that is not there, a schedule with
// no episodes and the like.
func NewTV(setup Setup) (*TV, error) {
	if err := setup.validate(); err != nil {
		return nil, fmt.Errorf("kinescope: %w", err)
	}
	tv := &TV{
		seed:   setup.Seed,
		random: rand.New(rand.NewPCG(setup.Seed, mix64(setup.Seed))),
		rates:  make(map[string]*float32),
		power:  newPower(),
	}

	// Map order is random: sources and schedules go by name, so the same
	// seed always plays the same
	for _, name := range sortedKeys(setup.Sources) {
		s := source{
			name:   name,
			source: setup.Sources[name],
			seed:   nameSeed(setup.Seed, name),
		}
		if _, ok := s.source.(Signal); ok {
			s.level = &Level{tv: tv}
		}
		tv.sources = append(tv.sources, s)
	}
	for _, d := range setup.Drives {
		from := slices.IndexFunc(tv.sources, func(s source) bool {
			return s.name == d.From
		})
		tv.drives = append(tv.drives, drive{from: from, to: d.To, weight: d.Weight})
	}
	for _, name := range sortedKeys(setup.Schedules) {
		rate := float32(1)
		tv.rates[name] = &rate
		s := &schedule{every: setup.Schedules[name], rate: Rate(name)}
		s.draw(tv.random)
		tv.schedules = append(tv.schedules, s)
	}

	tv.setEffects(setup.Effects)
	return tv, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Set of effects

// SetEffects replaces the TV's effects, keeping its sources, drives and
// schedules. A backend redraws the TV's shaders on the next frame, which
// may take a moment: change the set in a menu, not in play.
func (tv *TV) SetEffects(effects ...Effect) error {
	if errs := validateEffects(effects); len(errs) > 0 {
		return fmt.Errorf("kinescope: %w", errs[0])
	}
	tv.setEffects(effects)
	return nil
}

func (tv *TV) setEffects(effects []Effect) {
	tv.effects = tv.effects[:0]
	for _, effect := range effects {
		tv.effects = append(tv.effects, effect.clone())
	}
	slices.SortStableFunc(tv.effects, func(a, b Effect) int {
		return int(a.Stage()) - int(b.Stage())
	})

	tv.params = tv.params[:0]
	for _, effect := range tv.effects {
		tv.params = append(tv.params, effect.Params()...)
	}
	for _, name := range sortedKeys(tv.rates) {
		tv.params = append(tv.params, Param{
			Key:   Rate(name),
			Min:   0,
			Max:   maxRate,
			Value: tv.rates[name],
		})
	}
	tv.index = make(map[ParamKey]int, len(tv.params))
	for i, p := range tv.params {
		tv.index[p.Key] = i
	}
	tv.mods = make([]float32, len(tv.params))
	tv.revision++
}

// Effects are the TV's effects in stage order. They belong to the TV:
// change them through Params or Apply.
func (tv *TV) Effects() []Effect { return tv.effects }

// Revision changes whenever the set of effects does: a backend rebuilds its
// shaders when it sees a new one.
func (tv *TV) Revision() int { return tv.revision }

// Params

// Params are all the TV's tunable numbers, the effects' first, then the
// schedules' rates. Writing through a param's Value sets its base.
func (tv *TV) Params() []Param { return slices.Clone(tv.params) }

// Value is a param's value now: its base and the modulation on it, kept in
// range. A param the TV does not have is 0.
func (tv *TV) Value(key ParamKey) float32 {
	i, ok := tv.index[key]
	if !ok {
		return 0
	}
	if tv.stale {
		tv.modulate()
	}
	p := tv.params[i]
	return p.clamp(*p.Value + tv.mods[i])
}

// Values are the base values of the TV's params, without the modulation:
// what a game saves as the player's settings.
func (tv *TV) Values() map[ParamKey]float32 {
	values := make(map[ParamKey]float32, len(tv.params))
	for _, p := range tv.params {
		values[p.Key] = *p.Value
	}
	return values
}

// Apply sets the base values of the params it names, kept in range. Keys
// the TV does not have are left alone.
func (tv *TV) Apply(values map[ParamKey]float32) {
	for key, value := range values {
		if i, ok := tv.index[key]; ok {
			p := tv.params[i]
			*p.Value = p.clamp(value)
		}
	}
}

// Facts

// Signal is the level of the named signal, for the game to set. It fails if
// the setup has no signal by that name: better at start than a dead knob in
// play.
func (tv *TV) Signal(name string) (*Level, error) {
	for _, s := range tv.sources {
		if s.name != name {
			continue
		}
		if s.level == nil {
			return nil, fmt.Errorf("kinescope: source %q is not a signal", name)
		}
		return s.level, nil
	}
	return nil, fmt.Errorf("kinescope: no signal %q", name)
}

// Play plays an episode from its start, over whatever is playing.
func (tv *TV) Play(e Episode) {
	tv.playing = append(tv.playing, episode{episode: e})
}

// Hold keeps the schedules' episodes waiting while held is true — say,
// while the player drags something.
func (tv *TV) Hold(held bool) { tv.held = held }

// Reset starts the set afresh: no episode playing and, for the backend, no
// afterglow left on the phosphor. Call it when the effects come back on.
func (tv *TV) Reset() {
	tv.playing = tv.playing[:0]
	tv.epoch++
}

// Epoch changes on every Reset: a backend clears what it keeps between
// frames when it sees a new one.
func (tv *TV) Epoch() int { return tv.epoch }

// Clock

// Update moves the TV on by dt seconds: sources, episodes and schedules.
// Call it once per game tick; the TV has no clock of its own.
func (tv *TV) Update(dt float64) {
	tv.time += dt
	tv.frames++
	tv.power.since += dt

	// Episodes playing
	playing := tv.playing[:0]
	for _, e := range tv.playing {
		e.t += dt
		if _, done := e.episode.Envelope.level(e.t); !done {
			playing = append(playing, e)
		}
	}
	tv.playing = playing

	tv.modulate()

	for _, s := range tv.schedules {
		if s.advance(dt, tv.Value(s.rate), tv.held) {
			s.draw(tv.random)
			tv.Play(s.every.Episodes[tv.random.IntN(len(s.every.Episodes))])
		}
	}
}

// modulate works out what the sources and episodes add to each param.
func (tv *TV) modulate() {
	tv.stale = false
	clear(tv.mods)
	for i := range tv.sources {
		s := &tv.sources[i]
		s.value = sourceValue(s.source, s.level, s.seed, tv.time)
	}
	for _, d := range tv.drives {
		if i, ok := tv.index[d.to]; ok {
			tv.mods[i] += float32(tv.sources[d.from].value) * d.weight
		}
	}
	width, height, flash := tv.power.raster()
	tv.add(PowerWidth, float32(width-1))
	tv.add(PowerHeight, float32(height-1))
	tv.add(PowerFlash, float32(flash))
	for _, e := range tv.playing {
		level, _ := e.episode.Envelope.level(e.t)
		for _, target := range e.episode.Targets {
			if i, ok := tv.index[target.Param]; ok {
				tv.mods[i] += float32(level) * target.Weight
			}
		}
	}
}

// add adds to what modulation adds to a param, if the TV has it.
func (tv *TV) add(key ParamKey, value float32) {
	if i, ok := tv.index[key]; ok {
		tv.mods[i] += value
	}
}

// Time is the TV's clock in seconds, wrapped every hour to keep shaders
// precise.
func (tv *TV) Time() float64 { return math.Mod(tv.time, timeWrap) }

// Frame is the number of Updates so far.
func (tv *TV) Frame() int { return tv.frames }

// Geometry

// Map is the point of a w×h frame that the TV shows at the point x, y —
// where it would be without the bend of the glass and the tear of the
// lines. Use it for a pointer or a touch: map where the player pressed, hit
// what they see there. The result can fall outside the frame.
func (tv *TV) Map(x, y float64, w, h int) (float64, float64) {
	p := point{x, y}
	size := point{float64(w), float64(h)}
	for _, effect := range tv.effects {
		if w, ok := effect.(warper); ok {
			p = w.warp(p, size, tv.Value, tv.Time())
		}
	}
	return p.x, p.y
}
