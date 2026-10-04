package lab

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/shpaker/kinescope"
)

// stateFormat is the version of the state's layout; a state of another
// layout is not read.
const stateFormat = 1

// state is the setup as the page's address keeps it.
type state struct {
	Format    int           `json:"format"`
	Version   string        `json:"version,omitempty"` // the library's, for the record
	Preset    string        `json:"preset,omitempty"`
	Seed      uint64        `json:"seed,omitempty"`
	Effects   []stateEffect `json:"effects"`
	Sources   []source      `json:"sources,omitempty"`
	Drives    []stateDrive  `json:"drives,omitempty"`
	Schedules []schedule    `json:"schedules,omitempty"`
}

// stateEffect is an effect and its values.
type stateEffect struct {
	Name   string                         `json:"name"`
	Values map[kinescope.ParamKey]float32 `json:"values,omitempty"`
}

// stateDrive is a drive as the state keeps it.
type stateDrive struct {
	From   string             `json:"from"`
	To     kinescope.ParamKey `json:"to"`
	Weight float32            `json:"weight"`
}

// State is the setup as a string for the page's address: JSON in base64.
func (l *Lab) State() string {
	s := state{
		Format:    stateFormat,
		Version:   l.version,
		Preset:    l.preset,
		Seed:      l.seed,
		Effects:   []stateEffect{},
		Sources:   l.sources,
		Schedules: l.schedules,
	}
	for _, e := range l.effects {
		values := make(map[kinescope.ParamKey]float32)
		for _, p := range e.Params() {
			values[p.Key] = *p.Value
		}
		s.Effects = append(s.Effects, stateEffect{Name: e.Name(), Values: values})
	}
	for _, d := range l.drives {
		s.Drives = append(s.Drives, stateDrive{From: d.From, To: d.To, Weight: d.Weight})
	}
	data, _ := json.Marshal(s) // plain data: it always marshals
	return base64.RawURLEncoding.EncodeToString(data)
}

// Load takes a setup State made. What it cannot read it leaves alone; an
// effect it does not know it skips.
func (l *Lab) Load(text string) {
	text = strings.TrimPrefix(strings.TrimPrefix(text, "#"), "s=")
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil || len(data) == 0 {
		return
	}
	var s state
	if err := json.Unmarshal(data, &s); err != nil || s.Format != stateFormat {
		return
	}

	setup := kinescope.Setup{Seed: s.Seed}
	values := make(map[kinescope.ParamKey]float32)
	for _, se := range s.Effects {
		if e := newEffect(se.Name); e != nil {
			setup.Effects = append(setup.Effects, e)
			for key, value := range se.Values {
				values[key] = value
			}
		}
	}
	for _, d := range s.Drives {
		setup.Drives = append(setup.Drives, kinescope.Drive{From: d.From, To: d.To, Weight: d.Weight})
	}

	l.take(setup)
	l.sources = l.sources[:0]
	for _, src := range s.Sources {
		if _, ok := sourceNamesByKind[src.Kind]; ok {
			l.sources = append(l.sources, src)
		}
	}
	l.schedules = append(l.schedules[:0], s.Schedules...)
	for i := range l.schedules {
		if l.schedules[i].Episodes == nil {
			l.schedules[i].Episodes = []string{}
		}
	}
	for key, value := range values {
		if p, err := l.param(key); err == nil {
			*p.Value = min(max(value, p.Min), p.Max)
		}
	}
	l.preset = s.Preset
	l.rebuild()
}
