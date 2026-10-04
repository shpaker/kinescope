// Package lab is the kinescope lab's model: a setup being made, kept as the
// lists the page shows, the TV built from it, its Go code and its state for
// the page's address. It knows nothing of the screen or the browser.
package lab

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shpaker/kinescope"
)

// Lab is a setup being made and the TV built from it.
type Lab struct {
	// version is the library's version the lab is built with
	version string

	// The setup being made; preset names it while it is untouched
	preset    string
	seed      uint64
	effects   []kinescope.Effect // in stage order, as listed within a stage
	sources   []source
	drives    []kinescope.Drive
	schedules []schedule

	// The TV built from the setup and what the game has told it
	tv     *kinescope.TV
	levels map[string]float32
	held   bool
	off    bool

	// What went wrong: the setup's error, while it stands, and a
	// command's, shown once
	err    error
	notice string
}

// source is a named source of the setup.
type source struct {
	Name   string  `json:"name"`
	Kind   string  `json:"kind"`
	Period float32 `json:"period,omitempty"`
}

// schedule is a named schedule of the setup, its episodes by name, and its
// rate's base.
type schedule struct {
	Name     string   `json:"name"`
	Mean     float32  `json:"mean"`
	Spread   float32  `json:"spread"`
	Rate     float32  `json:"rate"`
	Episodes []string `json:"episodes"`
}

// New starts a lab on the first preset, then on what the state says. A
// state it cannot read leaves it on the preset.
func New(version, state string) *Lab {
	l := &Lab{version: version, levels: make(map[string]float32)}
	l.usePreset(presets[0])
	l.Load(state)
	return l
}

// TV is the TV built from the setup. A setup that does not hold together
// leaves the last TV that did.
func (l *Lab) TV() *kinescope.TV { return l.tv }

// Setup is the setup being made, as the library takes it.
func (l *Lab) Setup() kinescope.Setup {
	setup := kinescope.Setup{
		Seed:      l.seed,
		Effects:   slices.Clone(l.effects),
		Sources:   make(map[string]kinescope.Source),
		Drives:    slices.Clone(l.drives),
		Schedules: make(map[string]kinescope.Every),
	}
	for _, s := range l.sources {
		setup.Sources[s.Name] = s.source()
	}
	for _, s := range l.schedules {
		setup.Schedules[s.Name] = s.every()
	}
	return setup
}

// Values are the base values of the TV the lab makes: its effects' and its
// schedules' rates.
func (l *Lab) Values() map[kinescope.ParamKey]float32 {
	values := make(map[kinescope.ParamKey]float32)
	for _, e := range l.effects {
		for _, p := range e.Params() {
			values[p.Key] = *p.Value
		}
	}
	for _, s := range l.schedules {
		values[kinescope.Rate(s.Name)] = s.Rate
	}
	return values
}

func (s source) source() kinescope.Source {
	switch s.Kind {
	case kindDrift:
		return kinescope.Drift{Period: s.Period}
	case kindWave:
		return kinescope.Wave{Period: s.Period}
	}
	return kinescope.Signal{}
}

func (s schedule) every() kinescope.Every {
	every := kinescope.Every{Mean: s.Mean, Spread: s.Spread}
	for _, name := range s.Episodes {
		if e, ok := findEpisode(name); ok {
			every.Episodes = append(every.Episodes, e.make())
		}
	}
	return every
}

// Building the TV

// usePreset takes a preset's setup as a whole.
func (l *Lab) usePreset(p preset) {
	l.take(p.setup())
	l.preset = p.name
}

// take makes the lab's lists from a setup and builds its TV.
func (l *Lab) take(setup kinescope.Setup) {
	l.preset = ""
	l.seed = setup.Seed
	l.effects = slices.Clone(setup.Effects)
	slices.SortStableFunc(l.effects, func(a, b kinescope.Effect) int {
		return cmp.Compare(a.Stage(), b.Stage())
	})
	l.sources = l.sources[:0]
	for _, name := range slices.Sorted(maps.Keys(setup.Sources)) {
		s := source{Name: name, Kind: kindSignal}
		switch v := setup.Sources[name].(type) {
		case kinescope.Drift:
			s.Kind, s.Period = kindDrift, v.Period
		case kinescope.Wave:
			s.Kind, s.Period = kindWave, v.Period
		}
		l.sources = append(l.sources, s)
	}
	l.drives = slices.Clone(setup.Drives)
	l.schedules = l.schedules[:0]
	for _, name := range slices.Sorted(maps.Keys(setup.Schedules)) {
		every := setup.Schedules[name]
		s := schedule{Name: name, Mean: every.Mean, Spread: every.Spread, Rate: 1, Episodes: []string{}}
		for _, e := range every.Episodes {
			s.Episodes = append(s.Episodes, e.Name)
		}
		l.schedules = append(l.schedules, s)
	}
	l.rebuild()
}

// rebuild builds the TV afresh from the setup. A setup that does not hold
// together keeps its error and the TV before it.
func (l *Lab) rebuild() {
	tv, err := kinescope.NewTV(l.Setup())
	l.err = err
	if err != nil {
		if l.tv == nil {
			l.tv, _ = kinescope.NewTV(kinescope.Setup{})
		}
		return
	}
	l.tv = tv
	l.tv.Apply(l.Values())
	for name, level := range l.levels {
		if signal, err := tv.Signal(name); err == nil {
			signal.Set(level)
		}
	}
	tv.Hold(l.held)
	if l.off {
		tv.PowerOff()
	}
}

// Commands

// Command is a change the page asks for: Op names it, the other fields are
// what it takes.
type Command struct {
	Op string `json:"op"`

	// What the command is about, and what it changes it to
	Name    string             `json:"name"`
	Index   int                `json:"index"`
	Key     kinescope.ParamKey `json:"key"`
	Episode string             `json:"episode"`
	To      string             `json:"to"`
	Before  string             `json:"before"`
	From    string             `json:"from"`
	Kind    string             `json:"kind"`

	// Numbers; nil leaves a field as it is
	Value  *float32 `json:"value"`
	Weight *float32 `json:"weight"`
	Period *float32 `json:"period"`
	Mean   *float32 `json:"mean"`
	Spread *float32 `json:"spread"`
	Rate   *float32 `json:"rate"`
	Level  *float32 `json:"level"`
	Seed   *uint64  `json:"seed"`

	Episodes []string `json:"episodes"`
}

// Do runs a command. A command that cannot be done is reported once, as
// the view's notice.
func (l *Lab) Do(c Command) {
	if err := l.do(c); err != nil {
		l.notice = err.Error()
	}
}

func (l *Lab) do(c Command) error {
	switch c.Op {
	// The game's facts: they leave the setup alone
	case "signal":
		return l.setLevel(c.Name, c.Level)
	case "play":
		e, ok := findEpisode(c.Episode)
		if !ok {
			return fmt.Errorf("no episode %q", c.Episode)
		}
		l.tv.Play(e.make())
		return nil
	case "power":
		l.off = !l.off
		if l.off {
			l.tv.PowerOff()
		} else {
			l.tv.PowerOn()
		}
		return nil
	case "hold":
		l.held = !l.held
		l.tv.Hold(l.held)
		return nil
	case "reset":
		l.tv.Reset()
		return nil

	// The setup as a whole
	case "preset":
		for _, p := range presets {
			if p.name == c.Name {
				l.usePreset(p)
				return nil
			}
		}
		return fmt.Errorf("no preset %q", c.Name)
	case "seed":
		if c.Seed == nil {
			return fmt.Errorf("no seed")
		}
		l.seed = *c.Seed
	case "setParam":
		if err := l.setParam(c.Key, c.Value); err != nil {
			return err
		}
		l.preset = ""
		return nil // values need no new TV
	case "resetParam":
		if err := l.resetParam(c.Key); err != nil {
			return err
		}
		l.preset = ""
		return nil
	default:
		if err := l.edit(c); err != nil {
			return err
		}
	}
	l.preset = ""
	l.rebuild()
	return nil
}

// edit changes the setup's lists.
func (l *Lab) edit(c Command) error {
	switch c.Op {
	case "addEffect":
		return l.addEffect(c.Name)
	case "removeEffect":
		i, err := l.effect(c.Name)
		if err != nil {
			return err
		}
		l.effects = slices.Delete(l.effects, i, i+1)
	case "moveEffect":
		return l.moveEffect(c.Name, c.Before)
	case "addSource":
		return l.addSource(c.Kind)
	case "removeSource":
		i, err := l.source(c.Name)
		if err != nil {
			return err
		}
		l.sources = slices.Delete(l.sources, i, i+1)
		delete(l.levels, c.Name)
	case "renameSource":
		return l.renameSource(c.Name, c.To)
	case "setSource":
		return l.setSource(c)
	case "addDrive":
		return l.addDrive()
	case "removeDrive":
		if c.Index < 0 || c.Index >= len(l.drives) {
			return fmt.Errorf("no drive %d", c.Index)
		}
		l.drives = slices.Delete(l.drives, c.Index, c.Index+1)
	case "setDrive":
		return l.setDrive(c)
	case "addSchedule":
		l.schedules = append(l.schedules, schedule{
			Name:     freeName("glitches", l.scheduleNames()),
			Mean:     12,
			Spread:   8,
			Rate:     1,
			Episodes: []string{"jitter", "ripple"},
		})
	case "removeSchedule":
		i, err := l.schedule(c.Name)
		if err != nil {
			return err
		}
		l.schedules = slices.Delete(l.schedules, i, i+1)
	case "renameSchedule":
		return l.renameSchedule(c.Name, c.To)
	case "setSchedule":
		return l.setSchedule(c)
	default:
		return fmt.Errorf("no command %q", c.Op)
	}
	return nil
}

// Effects

func (l *Lab) effect(name string) (int, error) {
	i := slices.IndexFunc(l.effects, func(e kinescope.Effect) bool { return e.Name() == name })
	if i < 0 {
		return 0, fmt.Errorf("no effect %q", name)
	}
	return i, nil
}

// addEffect adds an effect at its defaults, last in its stage.
func (l *Lab) addEffect(name string) error {
	e := newEffect(name)
	if e == nil {
		return fmt.Errorf("no effect %q", name)
	}
	if _, err := l.effect(name); err == nil {
		return fmt.Errorf("the TV has %s already", name)
	}
	if slices.Contains(singleStages, e.Stage()) {
		for _, other := range l.effects {
			if other.Stage() == e.Stage() {
				return fmt.Errorf("a TV has one %s effect: remove %s first",
					stageNames[e.Stage()], other.Name())
			}
		}
	}
	i := len(l.effects)
	for i > 0 && l.effects[i-1].Stage() > e.Stage() {
		i--
	}
	l.effects = slices.Insert(l.effects, i, e)
	return nil
}

// moveEffect puts an effect before another of its stage.
func (l *Lab) moveEffect(name, before string) error {
	i, err := l.effect(name)
	if err != nil {
		return err
	}
	j, err := l.effect(before)
	if err != nil {
		return err
	}
	e := l.effects[i]
	if e.Stage() != l.effects[j].Stage() {
		return fmt.Errorf("%s and %s work in different stages", name, before)
	}
	l.effects = slices.Delete(l.effects, i, i+1)
	j, _ = l.effect(before)
	l.effects = slices.Insert(l.effects, j, e)
	return nil
}

// param is the param of the setup's effects by its key.
func (l *Lab) param(key kinescope.ParamKey) (kinescope.Param, error) {
	for _, e := range l.effects {
		for _, p := range e.Params() {
			if p.Key == key {
				return p, nil
			}
		}
	}
	return kinescope.Param{}, fmt.Errorf("no param %q", key)
}

func (l *Lab) setParam(key kinescope.ParamKey, value *float32) error {
	p, err := l.param(key)
	if err != nil {
		return err
	}
	if value == nil {
		return fmt.Errorf("no value for %s", key)
	}
	*p.Value = min(max(*value, p.Min), p.Max)
	l.tv.Apply(map[kinescope.ParamKey]float32{key: *p.Value})
	return nil
}

func (l *Lab) resetParam(key kinescope.ParamKey) error {
	value, ok := defaultValue(key)
	if !ok {
		return fmt.Errorf("no param %q", key)
	}
	return l.setParam(key, &value)
}

// defaultValue is a param's value in a fresh effect.
func defaultValue(key kinescope.ParamKey) (float32, bool) {
	name, _, _ := strings.Cut(string(key), ".")
	if e := newEffect(name); e != nil {
		for _, p := range e.Params() {
			if p.Key == key {
				return *p.Value, true
			}
		}
	}
	return 0, false
}

// Sources

func (l *Lab) source(name string) (int, error) {
	i := slices.IndexFunc(l.sources, func(s source) bool { return s.Name == name })
	if i < 0 {
		return 0, fmt.Errorf("no source %q", name)
	}
	return i, nil
}

func (l *Lab) sourceNames() []string {
	var names []string
	for _, s := range l.sources {
		names = append(names, s.Name)
	}
	return names
}

// sourceNamesByKind are the names the lab gives new sources of each kind.
var sourceNamesByKind = map[string]string{
	kindSignal: "shake",
	kindDrift:  "reception",
	kindWave:   "swell",
}

func (l *Lab) addSource(kind string) error {
	name, ok := sourceNamesByKind[kind]
	if !ok {
		return fmt.Errorf("no kind of source %q", kind)
	}
	s := source{Name: freeName(name, l.sourceNames()), Kind: kind}
	if kind != kindSignal {
		s.Period = 20
	}
	l.sources = append(l.sources, s)
	return nil
}

func (l *Lab) renameSource(name, to string) error {
	i, err := l.source(name)
	if err != nil {
		return err
	}
	if err := checkName(to, l.sourceNames()); err != nil {
		return err
	}
	l.sources[i].Name = to
	for j := range l.drives {
		if l.drives[j].From == name {
			l.drives[j].From = to
		}
	}
	if level, ok := l.levels[name]; ok {
		delete(l.levels, name)
		l.levels[to] = level
	}
	return nil
}

func (l *Lab) setSource(c Command) error {
	i, err := l.source(c.Name)
	if err != nil {
		return err
	}
	s := &l.sources[i]
	if c.Kind != "" {
		if _, ok := sourceNamesByKind[c.Kind]; !ok {
			return fmt.Errorf("no kind of source %q", c.Kind)
		}
		s.Kind = c.Kind
		switch {
		case s.Kind == kindSignal:
			s.Period = 0
		case s.Period == 0:
			s.Period = 20
		}
	}
	if c.Period != nil {
		s.Period = *c.Period
	}
	return nil
}

func (l *Lab) setLevel(name string, level *float32) error {
	if level == nil {
		return fmt.Errorf("no level for %s", name)
	}
	signal, err := l.tv.Signal(name)
	if err != nil {
		return err
	}
	l.levels[name] = min(max(*level, 0), 1)
	signal.Set(l.levels[name])
	return nil
}

// Drives

// targets are the keys a drive can move: the effects' params and the
// schedules' rates.
func (l *Lab) targets() []kinescope.ParamKey {
	var keys []kinescope.ParamKey
	for _, e := range l.effects {
		for _, p := range e.Params() {
			keys = append(keys, p.Key)
		}
	}
	for _, s := range l.schedules {
		keys = append(keys, kinescope.Rate(s.Name))
	}
	return keys
}

func (l *Lab) addDrive() error {
	if len(l.sources) == 0 {
		return fmt.Errorf("add a source first: a drive goes from a source to a param")
	}
	targets := l.targets()
	if len(targets) == 0 {
		return fmt.Errorf("add an effect or a schedule first: a drive moves one of their params")
	}
	to := targets[0]
	if slices.Contains(targets, kinescope.GrainStrength) {
		to = kinescope.GrainStrength
	}
	l.drives = append(l.drives, kinescope.Drive{From: l.sources[0].Name, To: to, Weight: 0.1})
	return nil
}

func (l *Lab) setDrive(c Command) error {
	if c.Index < 0 || c.Index >= len(l.drives) {
		return fmt.Errorf("no drive %d", c.Index)
	}
	d := &l.drives[c.Index]
	if c.From != "" {
		d.From = c.From
	}
	if c.To != "" {
		d.To = kinescope.ParamKey(c.To)
	}
	if c.Weight != nil {
		d.Weight = *c.Weight
	}
	return nil
}

// Schedules

func (l *Lab) schedule(name string) (int, error) {
	i := slices.IndexFunc(l.schedules, func(s schedule) bool { return s.Name == name })
	if i < 0 {
		return 0, fmt.Errorf("no schedule %q", name)
	}
	return i, nil
}

func (l *Lab) scheduleNames() []string {
	var names []string
	for _, s := range l.schedules {
		names = append(names, s.Name)
	}
	return names
}

func (l *Lab) renameSchedule(name, to string) error {
	i, err := l.schedule(name)
	if err != nil {
		return err
	}
	if err := checkName(to, l.scheduleNames()); err != nil {
		return err
	}
	l.schedules[i].Name = to
	for j := range l.drives {
		if l.drives[j].To == kinescope.Rate(name) {
			l.drives[j].To = kinescope.Rate(to)
		}
	}
	return nil
}

func (l *Lab) setSchedule(c Command) error {
	i, err := l.schedule(c.Name)
	if err != nil {
		return err
	}
	s := &l.schedules[i]
	if c.Mean != nil {
		s.Mean = *c.Mean
	}
	if c.Spread != nil {
		s.Spread = *c.Spread
	}
	if c.Rate != nil {
		s.Rate = min(max(*c.Rate, 0), 10)
	}
	if c.Episodes != nil {
		s.Episodes = s.Episodes[:0]
		for _, e := range episodes {
			if name := e.make().Name; slices.Contains(c.Episodes, name) {
				s.Episodes = append(s.Episodes, name)
			}
		}
	}
	return nil
}

// Names

// freeName is base, or base with a number, not yet taken.
func freeName(base string, taken []string) string {
	name := base
	for i := 2; slices.Contains(taken, name); i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	return name
}

// checkName reports what is wrong with a new name.
func checkName(name string, taken []string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("a name can't be empty")
	case slices.Contains(taken, name):
		return fmt.Errorf("the name %q is taken", name)
	}
	return nil
}
