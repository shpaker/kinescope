package lab

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
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
	Picture   *Picture      `json:"picture,omitempty"`
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
		Picture:   &l.picture,
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

// Load takes a setup State made, "s=<base64>", or the same JSON as it is,
// "j=<JSON>", percent-encoded or not: the form for people and agents who
// write links by hand. In JSON a missing format is the current one, and a
// preset with no effects is that preset as it is, on the seed given if any. What Load cannot read it
// leaves alone; an effect it does not know it skips.
func (l *Lab) Load(text string) {
	s, ok := readState(strings.TrimPrefix(text, "#"))
	if !ok {
		return
	}

	l.picture = defaultPicture
	if s.Picture != nil {
		_ = l.setPicture(s.Picture) // a picture it does not know leaves the default
	}
	if len(s.Effects) == 0 {
		if p, ok := findPreset(s.Preset); ok {
			l.usePreset(p)
			if s.Seed != 0 && s.Seed != l.seed {
				l.seed, l.preset = s.Seed, "" // the preset no more, as with the seed's field
				l.rebuild()
			}
			return
		}
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

// readState reads a state in either form, and whether it could.
func readState(text string) (state, bool) {
	var s state
	var data []byte
	if j, ok := strings.CutPrefix(text, "j="); ok {
		if unescaped, err := url.PathUnescape(j); err == nil {
			j = unescaped
		}
		s.Format, data = stateFormat, []byte(j)
	} else {
		var err error
		data, err = base64.RawURLEncoding.DecodeString(strings.TrimPrefix(text, "s="))
		if err != nil {
			return state{}, false
		}
	}
	if len(data) == 0 || json.Unmarshal(data, &s) != nil || s.Format != stateFormat {
		return state{}, false
	}
	return s, true
}
