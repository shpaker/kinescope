package main

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/shpaker/kinescope"
)

// lab is the lab's model: which effects are on, every param's value, the
// moods, and the TV built from them. It knows nothing of the screen.
type lab struct {
	// effects are every effect there is, in stage order, and whether it is on
	effects []toggle
	// values are the base values of every param, kept while an effect is off
	values map[kinescope.ParamKey]float32
	// moods are the setup's misbehavior, each on or off
	moods []mood

	tv    *kinescope.TV
	shake *kinescope.Level
}

// toggle is an effect switched on or off.
type toggle struct {
	name string
	on   bool
}

// mood is a part of a setup beyond its effects: sources, drives and
// schedules. code is the same, written out in Go for "Copy as Go".
type mood struct {
	name  string
	on    bool
	apply func(*kinescope.Setup)
	code  string
}

// The lab's own signal: the shake slider
const shakeSignal = "shake"

func newMoods() []mood {
	return []mood{
		{
			name: "reception",
			apply: func(s *kinescope.Setup) {
				s.Sources["reception"] = kinescope.Drift{Period: 20}
				s.Drives = append(s.Drives,
					kinescope.Drive{From: "reception", To: kinescope.GrainStrength, Weight: 0.08},
					kinescope.Drive{From: "reception", To: kinescope.FlickerStrength, Weight: 0.02},
					kinescope.Drive{From: "reception", To: kinescope.HumStrength, Weight: 0.05},
				)
			},
			code: `// Reception drifts: the noise swells and fades
setup.Sources["reception"] = kinescope.Drift{Period: 20}
setup.Drives = append(setup.Drives,
	kinescope.Drive{From: "reception", To: kinescope.GrainStrength, Weight: 0.08},
	kinescope.Drive{From: "reception", To: kinescope.FlickerStrength, Weight: 0.02},
	kinescope.Drive{From: "reception", To: kinescope.HumStrength, Weight: 0.05},
)
`,
		},
		{
			name: "glitches",
			apply: func(s *kinescope.Setup) {
				s.Schedules["glitches"] = kinescope.Every{
					Mean:     12,
					Spread:   8,
					Episodes: []kinescope.Episode{kinescope.Jitter(), kinescope.Ripple()},
				}
				if _, ok := s.Sources["reception"]; ok {
					s.Drives = append(s.Drives, kinescope.Drive{
						From: "reception", To: kinescope.Rate("glitches"), Weight: 2,
					})
				}
			},
			code: `// Glitches now and then, thicker when the reception is poor
setup.Schedules["glitches"] = kinescope.Every{
	Mean:     12,
	Spread:   8,
	Episodes: []kinescope.Episode{kinescope.Jitter(), kinescope.Ripple()},
}
if _, ok := setup.Sources["reception"]; ok {
	setup.Drives = append(setup.Drives, kinescope.Drive{
		From: "reception", To: kinescope.Rate("glitches"), Weight: 2,
	})
}
`,
		},
	}
}

// newLab starts the lab on Gorizont, then on what the state says.
func newLab(state string) (*lab, error) {
	l := &lab{values: make(map[kinescope.ParamKey]float32), moods: newMoods()}
	gorizont := map[string]bool{}
	for _, e := range kinescope.Gorizont().Effects {
		gorizont[e.Name()] = true
	}
	for _, e := range kinescope.Effects() {
		l.effects = append(l.effects, toggle{name: e.Name(), on: gorizont[e.Name()]})
		for _, p := range e.Params() {
			l.values[p.Key] = *p.Value
		}
	}
	l.decode(state)
	return l, l.rebuild()
}

// rebuild builds the TV afresh from the lab's model.
func (l *lab) rebuild() error {
	setup := kinescope.Setup{
		Effects:   l.enabledEffects(),
		Sources:   map[string]kinescope.Source{shakeSignal: kinescope.Signal{}},
		Drives:    []kinescope.Drive{{From: shakeSignal, To: kinescope.TearStrength, Weight: 1}},
		Schedules: map[string]kinescope.Every{},
	}
	for _, m := range l.moods {
		if m.on {
			m.apply(&setup)
		}
	}
	tv, err := kinescope.NewTV(setup)
	if err != nil {
		return err
	}
	tv.Apply(l.values)
	shake, err := tv.Signal(shakeSignal)
	if err != nil {
		return err
	}
	l.tv, l.shake = tv, shake
	return nil
}

func (l *lab) enabledEffects() []kinescope.Effect {
	var effects []kinescope.Effect
	for _, e := range kinescope.Effects() {
		if l.isOn(e.Name()) {
			effects = append(effects, e)
		}
	}
	return effects
}

func (l *lab) isOn(name string) bool {
	i := slices.IndexFunc(l.effects, func(t toggle) bool { return t.name == name })
	return i >= 0 && l.effects[i].on
}

// keep takes the TV's current base values into the lab's model.
func (l *lab) keep() {
	if l.tv == nil {
		return
	}
	for key, value := range l.tv.Values() {
		l.values[key] = value
	}
}

// setEffects switches the TV's effects to the lab's toggles, keeping the
// values.
func (l *lab) setEffects() error {
	l.keep()
	if err := l.tv.SetEffects(l.enabledEffects()...); err != nil {
		return err
	}
	l.tv.Apply(l.values)
	return nil
}

// reset puts the lab back to Gorizont without moods.
func (l *lab) reset() error {
	fresh, err := newLab("")
	if err != nil {
		return err
	}
	*l = *fresh
	return nil
}

// State

// encode is the lab's state as a URL query: what is on, every value, the
// moods.
func (l *lab) encode() string {
	l.keep()
	q := url.Values{}
	q.Set("on", strings.Join(l.names(func(t toggle) bool { return t.on }), ","))
	var moods []string
	for _, m := range l.moods {
		if m.on {
			moods = append(moods, m.name)
		}
	}
	q.Set("moods", strings.Join(moods, ","))
	for key, value := range l.values {
		q.Set(string(key), strconv.FormatFloat(float64(value), 'g', 4, 32))
	}
	return q.Encode()
}

// decode takes a state encode made; what it cannot read it leaves alone.
func (l *lab) decode(state string) {
	q, err := url.ParseQuery(strings.TrimPrefix(state, "#"))
	if err != nil || len(q) == 0 {
		return
	}
	if q.Has("on") {
		on := strings.Split(q.Get("on"), ",")
		for i := range l.effects {
			l.effects[i].on = slices.Contains(on, l.effects[i].name)
		}
	}
	if q.Has("moods") {
		moods := strings.Split(q.Get("moods"), ",")
		for i := range l.moods {
			l.moods[i].on = slices.Contains(moods, l.moods[i].name)
		}
	}
	for key := range l.values {
		if v, err := strconv.ParseFloat(q.Get(string(key)), 32); err == nil {
			l.values[key] = float32(v)
		}
	}
}

func (l *lab) names(keep func(toggle) bool) []string {
	var names []string
	for _, t := range l.effects {
		if keep(t) {
			names = append(names, t.name)
		}
	}
	return names
}

// goCode is the lab's TV written out in Go, to paste into a game.
func (l *lab) goCode() string {
	var b strings.Builder
	b.WriteString("setup := kinescope.Setup{\n\tEffects: []kinescope.Effect{\n")
	for _, e := range l.tv.Effects() {
		var fields []string
		for _, p := range e.Params() {
			_, field, _ := strings.Cut(string(p.Key), ".")
			fields = append(fields, fmt.Sprintf("%s: %s",
				camel(field), strconv.FormatFloat(float64(*p.Value), 'g', 4, 32)))
		}
		fmt.Fprintf(&b, "\t\t&kinescope.%s{%s},\n", camel(e.Name()), strings.Join(fields, ", "))
	}
	b.WriteString("\t},\n\tSources:   map[string]kinescope.Source{},\n")
	b.WriteString("\tSchedules: map[string]kinescope.Every{},\n}\n")
	for _, m := range l.moods {
		if m.on {
			b.WriteString(m.code)
		}
	}
	return b.String()
}

// camel is a snake_case name in CamelCase: aperture_mask → ApertureMask.
func camel(s string) string {
	var b strings.Builder
	for _, word := range strings.Split(s, "_") {
		if word != "" {
			b.WriteString(strings.ToUpper(word[:1]) + word[1:])
		}
	}
	return b.String()
}
