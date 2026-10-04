package lab

//go:generate go test -run TestDocs -update .

import (
	"github.com/shpaker/kinescope"
)

// The catalog: everything a setup in the lab can be made of.

// stageNames are the stages' names, by kinescope.Stage.
var stageNames = []string{"prepass", "geometry", "read", "sample", "light", "beam", "mask", "post", "frame"}

// singleStages are the stages a TV has at most one effect of.
var singleStages = []kinescope.Stage{kinescope.StageRead, kinescope.StageSample}

// Kinds of sources
const (
	kindSignal = "signal"
	kindDrift  = "drift"
	kindWave   = "wave"
)

// kinds are the kinds of sources, with their Go type names.
var kinds = []struct{ name, typ string }{
	{kindSignal, "Signal"},
	{kindDrift, "Drift"},
	{kindWave, "Wave"},
}

// episode is an episode the library has, and its constructor's name.
type episode struct {
	constructor string
	make        func() kinescope.Episode
}

// episodes are the library's episodes, in the order the lab lists them.
var episodes = []episode{
	{"Jitter", kinescope.Jitter},
	{"Ripple", kinescope.Ripple},
	{"RollOver", kinescope.RollOver},
	{"SnowBurst", kinescope.SnowBurst},
	{"Degaussing", kinescope.Degaussing},
}

// findEpisode is the library's episode by its name ("roll"), and whether
// there is one.
func findEpisode(name string) (episode, bool) {
	for _, e := range episodes {
		if e.make().Name == name {
			return e, true
		}
	}
	return episode{}, false
}

// preset is a TV the lab can start from.
type preset struct {
	name  string
	setup func() kinescope.Setup
}

// presets are the library's TVs; the lab starts on the first.
var presets = []preset{
	{"Gorizont", kinescope.Gorizont},
	{"Rubin", kinescope.Rubin},
}

// newEffect is a fresh effect by its name, at its defaults, or nil.
func newEffect(name string) kinescope.Effect {
	for _, e := range kinescope.Effects() {
		if e.Name() == name {
			return e
		}
	}
	return nil
}
