package lab

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shpaker/kinescope"
)

// View is everything the page shows of the lab, ready for JSON.
type View struct {
	Version string   `json:"version"`
	Presets []string `json:"presets"`
	Preset  string   `json:"preset"`
	Seed    uint64   `json:"seed"`

	// What is wrong: the setup's error and the items it is about, items
	// that do nothing, and a command that could not be done
	Error      string            `json:"error"`
	ErrorItems []string          `json:"errorItems"`
	Warnings   map[string]string `json:"warnings"`
	Notice     string            `json:"notice"`

	// The catalog and the setup
	Catalog   CatalogView          `json:"catalog"`
	Effects   []EffectView         `json:"effects"`
	Sources   []SourceView         `json:"sources"`
	Drives    []DriveView          `json:"drives"`
	Targets   []kinescope.ParamKey `json:"targets"`
	Schedules []ScheduleView       `json:"schedules"`

	// What the game has told the TV
	Levels map[string]float32 `json:"levels"`
	Dark   bool               `json:"dark"`
	Held   bool               `json:"held"`

	Code CodeView `json:"code"`
}

// CatalogView is what a setup can be made of.
type CatalogView struct {
	Effects  []EntryView `json:"effects"`
	Single   []string    `json:"single"`
	Kinds    []EntryView `json:"kinds"`
	Episodes []EntryView `json:"episodes"`
}

// EntryView is something of the catalog: its name, its stage for an
// effect, and its doc.
type EntryView struct {
	Name  string `json:"name"`
	Stage string `json:"stage,omitempty"`
	Doc   string `json:"doc"`
}

// EffectView is an effect of the setup with its params.
type EffectView struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Stage  string      `json:"stage"`
	Doc    string      `json:"doc"`
	Params []ParamView `json:"params"`
}

// ParamView is a param of an effect: its range, value and default.
type ParamView struct {
	Key     kinescope.ParamKey `json:"key"`
	Field   string             `json:"field"`
	Min     float32            `json:"min"`
	Max     float32            `json:"max"`
	Value   float32            `json:"value"`
	Default float32            `json:"default"`
	Doc     string             `json:"doc"`
}

// SourceView is a source of the setup.
type SourceView struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Kind   string  `json:"kind"`
	Period float32 `json:"period"`
}

// DriveView is a drive of the setup.
type DriveView struct {
	ID     string             `json:"id"`
	From   string             `json:"from"`
	To     kinescope.ParamKey `json:"to"`
	Weight float32            `json:"weight"`
}

// ScheduleView is a schedule of the setup.
type ScheduleView struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Mean     float32  `json:"mean"`
	Spread   float32  `json:"spread"`
	Rate     float32  `json:"rate"`
	Episodes []string `json:"episodes"`
}

// CodeView is the setup in Go, alone and as a whole game.
type CodeView struct {
	Setup []Chunk `json:"setup"`
	Full  []Chunk `json:"full"`
}

// View is the lab as the page shows it. A command's notice is shown once:
// View clears it.
func (l *Lab) View() View {
	v := View{
		Version:    l.version,
		Preset:     l.preset,
		Seed:       l.seed,
		ErrorItems: l.badItems(),
		Warnings:   l.warnings(),
		Notice:     l.notice,
		Catalog:    catalog(),
		Effects:    []EffectView{},
		Sources:    []SourceView{},
		Drives:     []DriveView{},
		Targets:    l.targets(),
		Schedules:  []ScheduleView{},
		Levels:     l.levels,
		Dark:       l.off,
		Held:       l.held,
		Code:       CodeView{Setup: l.Code(false), Full: l.Code(true)},
	}
	l.notice = ""
	if l.err != nil {
		v.Error = strings.TrimPrefix(l.err.Error(), "kinescope: ")
	}
	for _, p := range presets {
		v.Presets = append(v.Presets, p.name)
	}
	for _, e := range l.effects {
		ev := EffectView{
			ID:    "effect:" + e.Name(),
			Name:  e.Name(),
			Stage: stageNames[e.Stage()],
			Doc:   typeDocs[camel(e.Name())],
		}
		for _, p := range e.Params() {
			_, field, _ := strings.Cut(string(p.Key), ".")
			def, _ := defaultValue(p.Key)
			ev.Params = append(ev.Params, ParamView{
				Key: p.Key, Field: field, Min: p.Min, Max: p.Max,
				Value: *p.Value, Default: def, Doc: fieldDocs[p.Key],
			})
		}
		v.Effects = append(v.Effects, ev)
	}
	for _, s := range l.sources {
		v.Sources = append(v.Sources, SourceView{ID: "source:" + s.Name, Name: s.Name, Kind: s.Kind, Period: s.Period})
	}
	for i, d := range l.drives {
		v.Drives = append(v.Drives, DriveView{ID: fmt.Sprintf("drive:%d", i), From: d.From, To: d.To, Weight: d.Weight})
	}
	for _, s := range l.schedules {
		v.Schedules = append(v.Schedules, ScheduleView{
			ID: "schedule:" + s.Name, Name: s.Name, Mean: s.Mean, Spread: s.Spread,
			Rate: s.Rate, Episodes: s.Episodes,
		})
	}
	return v
}

func catalog() CatalogView {
	c := CatalogView{}
	for _, e := range kinescope.Effects() {
		c.Effects = append(c.Effects, EntryView{
			Name:  e.Name(),
			Stage: stageNames[e.Stage()],
			Doc:   typeDocs[camel(e.Name())],
		})
	}
	for _, s := range singleStages {
		c.Single = append(c.Single, stageNames[s])
	}
	for _, k := range kinds {
		c.Kinds = append(c.Kinds, EntryView{Name: k.name, Doc: typeDocs[k.typ]})
	}
	for _, e := range episodes {
		c.Episodes = append(c.Episodes, EntryView{Name: e.make().Name, Doc: typeDocs[e.constructor]})
	}
	return c
}

// badItems are the items of the setup that keep it from holding together:
// the library's checks, item by item.
func (l *Lab) badItems() []string {
	items := []string{}
	for _, s := range l.sources {
		if s.Kind != kindSignal && s.Period <= 0 {
			items = append(items, "source:"+s.Name)
		}
	}
	for i, d := range l.drives {
		_, err := l.source(d.From)
		rate, isRate := strings.CutPrefix(string(d.To), "rate.")
		if err != nil || isRate && !slices.Contains(l.scheduleNames(), rate) {
			items = append(items, fmt.Sprintf("drive:%d", i))
		}
	}
	for _, s := range l.schedules {
		if s.Mean <= 0 || s.Spread < 0 || s.Spread >= s.Mean || len(s.Episodes) == 0 {
			items = append(items, "schedule:"+s.Name)
		}
	}
	return items
}

// warnings are the items that hold together but do nothing.
func (l *Lab) warnings() map[string]string {
	warnings := make(map[string]string)
	for i, d := range l.drives {
		if strings.HasPrefix(string(d.To), "rate.") {
			continue
		}
		effect, _, _ := strings.Cut(string(d.To), ".")
		if _, err := l.effect(effect); err != nil {
			warnings[fmt.Sprintf("drive:%d", i)] = fmt.Sprintf("The TV has no %s effect: this drive does nothing.", effect)
		}
	}
	return warnings
}

// Live are the values now, the modulation on.
type Live struct {
	Params  map[kinescope.ParamKey]float32 `json:"params"`
	Sources map[string]float32             `json:"sources"`
}

// Live is the TV's values now: every param's, modulation on, and every
// source's.
func (l *Lab) Live() Live {
	live := Live{Params: make(map[kinescope.ParamKey]float32), Sources: make(map[string]float32)}
	for _, p := range l.tv.Params() {
		live.Params[p.Key] = l.tv.Value(p.Key)
	}
	for _, s := range l.sources {
		live.Sources[s.Name] = l.tv.SourceValue(s.Name)
	}
	return live
}
