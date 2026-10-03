package kinescope

import (
	"math"
	"slices"
	"strings"
	"testing"
)

const tick = 1.0 / 60

func newTV(t *testing.T, setup Setup) *TV {
	t.Helper()
	tv, err := NewTV(setup)
	if err != nil {
		t.Fatal(err)
	}
	return tv
}

func run(tv *TV, seconds float64) {
	for range int(seconds / tick) {
		tv.Update(tick)
	}
}

func TestNewTVRejectsBrokenSetups(t *testing.T) {
	cases := map[string]Setup{
		"unknown source": {
			Drives: []Drive{{From: "shake", To: TearStrength, Weight: 1}},
		},
		"unknown schedule": {
			Sources: map[string]Source{"reception": Drift{Period: 10}},
			Drives:  []Drive{{From: "reception", To: Rate("glitches"), Weight: 1}},
		},
		"twice the same effect": {
			Effects: []Effect{NewTear(), NewTear()},
		},
		"nil effect": {
			Effects: []Effect{nil},
		},
		"drift without period": {
			Sources: map[string]Source{"reception": Drift{}},
		},
		"schedule without episodes": {
			Schedules: map[string]Every{"glitches": {Mean: 10}},
		},
		"spread past mean": {
			Schedules: map[string]Every{
				"glitches": {Mean: 10, Spread: 10, Episodes: []Episode{Jitter()}},
			},
		},
	}
	for name, setup := range cases {
		if _, err := NewTV(setup); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestSignalDrivesParam(t *testing.T) {
	tv := newTV(t, Setup{
		Effects: []Effect{NewTear()},
		Sources: map[string]Source{"shake": Signal{}},
		Drives:  []Drive{{From: "shake", To: TearStrength, Weight: 1}},
	})
	shake, err := tv.Signal("shake")
	if err != nil {
		t.Fatal(err)
	}

	shake.Set(0.5)
	tv.Update(tick)
	if got := tv.Value(TearStrength); got != 0.5 {
		t.Errorf("strength = %v, want 0.5", got)
	}

	shake.Set(3)
	tv.Update(tick)
	if got := tv.Value(TearStrength); got != 1 {
		t.Errorf("strength = %v, want 1: kept in range", got)
	}
	if got := tv.Values()[TearStrength]; got != 0 {
		t.Errorf("base = %v, want 0: modulation leaves the base alone", got)
	}
}

// A game may set a signal in Draw, after the TV's Update: the value follows
// at once.
func TestSignalSetAfterUpdate(t *testing.T) {
	tv := newTV(t, Setup{
		Effects: []Effect{NewTear()},
		Sources: map[string]Source{"shake": Signal{}},
		Drives:  []Drive{{From: "shake", To: TearStrength, Weight: 1}},
	})
	shake, _ := tv.Signal("shake")
	tv.Update(tick)
	shake.Set(0.25)
	if got := tv.Value(TearStrength); got != 0.25 {
		t.Errorf("strength = %v, want 0.25", got)
	}
}

func TestSignalLookup(t *testing.T) {
	tv := newTV(t, Setup{
		Sources: map[string]Source{"reception": Drift{Period: 10}},
	})
	if _, err := tv.Signal("shake"); err == nil {
		t.Error("missing signal: no error")
	}
	if _, err := tv.Signal("reception"); err == nil {
		t.Error("a drift is not a signal: no error")
	}
}

func TestDriveToMissingEffectDoesNothing(t *testing.T) {
	tv := newTV(t, Setup{
		Sources: map[string]Source{"shake": Signal{}},
		Drives:  []Drive{{From: "shake", To: TearStrength, Weight: 1}},
	})
	shake, _ := tv.Signal("shake")
	shake.Set(1)
	tv.Update(tick)
	if got := tv.Value(TearStrength); got != 0 {
		t.Errorf("strength = %v, want 0", got)
	}
}

func TestEpisodePlaysAndEnds(t *testing.T) {
	tv := newTV(t, Setup{Effects: []Effect{NewTear()}})
	tv.Play(Episode{
		Envelope: Envelope{Attack: 0.1, Hold: 0.1, Release: 0.1},
		Targets:  []Target{{Param: TearStrength, Weight: 0.8}},
	})

	run(tv, 0.15)
	if got := tv.Value(TearStrength); math.Abs(float64(got)-0.8) > 1e-6 {
		t.Errorf("at the peak strength = %v, want 0.8", got)
	}
	if got := tv.Values()[TearStrength]; got != 0 {
		t.Errorf("base during the episode = %v, want 0", got)
	}

	run(tv, 0.3)
	if got := tv.Value(TearStrength); got != 0 {
		t.Errorf("after the episode strength = %v, want 0", got)
	}
	if len(tv.playing) != 0 {
		t.Errorf("%d episodes still playing", len(tv.playing))
	}
}

func TestEnvelope(t *testing.T) {
	e := Envelope{Attack: 1, Hold: 1, Release: 2}
	cases := []struct {
		t    float64
		want float64
		done bool
	}{
		{0, 0, false},
		{0.5, 0.5, false},
		{1.5, 1, false},
		{3, 0.5, false},
		{4, 0, true},
	}
	for _, c := range cases {
		got, done := e.level(c.t)
		if math.Abs(got-c.want) > 1e-9 || done != c.done {
			t.Errorf("level(%v) = %v, %v; want %v, %v", c.t, got, done, c.want, c.done)
		}
	}
}

func TestDriftIsSmoothAndRepeatable(t *testing.T) {
	const period = 10.0
	previous := sourceValue(Drift{Period: period}, nil, 42, 0)
	for i := 1; i < 10000; i++ {
		at := float64(i) * 0.05
		v := sourceValue(Drift{Period: period}, nil, 42, at)
		if v < 0 || v > 1 {
			t.Fatalf("drift(%v) = %v, out of 0..1", at, v)
		}
		// Smoothstep between random levels: never steeper than 1.5 per period
		if math.Abs(v-previous) > 1.5*0.05/period+1e-9 {
			t.Fatalf("drift jumps at %v: %v -> %v", at, previous, v)
		}
		if again := sourceValue(Drift{Period: period}, nil, 42, at); again != v {
			t.Fatalf("drift(%v) not repeatable: %v, %v", at, v, again)
		}
		previous = v
	}
}

func TestWave(t *testing.T) {
	w := Wave{Period: 4}
	for _, c := range []struct{ t, want float64 }{{0, 0}, {2, 1}, {4, 0}, {1, 0.5}} {
		if got := sourceValue(w, nil, 0, c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("wave(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

// glitchTimes runs a TV for seconds and returns when its episodes started.
func glitchTimes(tv *TV, seconds float64) []int {
	var starts []int
	for i := range int(seconds / tick) {
		before := len(tv.playing)
		tv.Update(tick)
		if len(tv.playing) > before {
			starts = append(starts, i)
		}
	}
	return starts
}

func glitchySetup(seed uint64) Setup {
	return Setup{
		Seed:    seed,
		Effects: []Effect{NewTear()},
		Schedules: map[string]Every{
			"glitches": {Mean: 2, Spread: 1, Episodes: []Episode{Jitter()}},
		},
	}
}

func TestScheduleIsRepeatable(t *testing.T) {
	a := glitchTimes(newTV(t, glitchySetup(7)), 60)
	b := glitchTimes(newTV(t, glitchySetup(7)), 60)
	if len(a) < 15 || len(a) > 60 {
		t.Fatalf("%d glitches in a minute, want about 30", len(a))
	}
	if !slices.Equal(a, b) {
		t.Error("the same seed played different glitches")
	}
}

func TestHoldDelaysGlitches(t *testing.T) {
	tv := newTV(t, glitchySetup(7))
	tv.Hold(true)
	if n := len(glitchTimes(tv, 10)); n != 0 {
		t.Errorf("%d glitches while held", n)
	}
	tv.Hold(false)
	if n := len(glitchTimes(tv, tick)); n != 1 {
		t.Errorf("%d glitches on release, want the one waiting", n)
	}
}

func TestDrivenRateMakesGlitchesThicker(t *testing.T) {
	calm := newTV(t, glitchySetup(7))
	setup := glitchySetup(7)
	setup.Sources = map[string]Source{"storm": Signal{}}
	setup.Drives = []Drive{{From: "storm", To: Rate("glitches"), Weight: 2}}
	stormy := newTV(t, setup)
	storm, _ := stormy.Signal("storm")
	storm.Set(1)

	n, m := len(glitchTimes(calm, 120)), len(glitchTimes(stormy, 120))
	if m < 2*n {
		t.Errorf("%d glitches at rate 3, %d at rate 1: want about three times as many", m, n)
	}
}

func TestApply(t *testing.T) {
	tv := newTV(t, Setup{Effects: []Effect{NewGrain()}})
	tv.Apply(map[ParamKey]float32{GrainStrength: 7, "nope": 1})
	if got := tv.Value(GrainStrength); got != 0.5 {
		t.Errorf("grain = %v, want 0.5: kept in range", got)
	}
}

func TestSetEffectsKeepsRates(t *testing.T) {
	tv := newTV(t, glitchySetup(1))
	tv.Apply(map[ParamKey]float32{Rate("glitches"): 3})
	revision := tv.Revision()

	if err := tv.SetEffects(NewGrain(), NewCurvature()); err != nil {
		t.Fatal(err)
	}
	if tv.Revision() == revision {
		t.Error("revision unchanged")
	}
	if got := tv.Value(Rate("glitches")); got != 3 {
		t.Errorf("rate = %v, want 3", got)
	}
	if tv.Effects()[0].Name() != "curvature" {
		t.Errorf("first effect %q, want curvature: effects go in stage order", tv.Effects()[0].Name())
	}
	if err := tv.SetEffects(NewGrain(), NewGrain()); err == nil {
		t.Error("twice the same effect: no error")
	}
}

func TestTVOwnsItsEffects(t *testing.T) {
	setup := Setup{Effects: []Effect{NewGrain()}}
	tv := newTV(t, setup)
	setup.Effects[0].(*Grain).Strength = 0.3
	if got := tv.Value(GrainStrength); got != 0.05 {
		t.Errorf("grain = %v, want 0.05: the setup's effect is not the TV's", got)
	}
}

func TestBlend(t *testing.T) {
	got := Blend(0.25,
		map[ParamKey]float32{GrainStrength: 0, HumStrength: 1},
		map[ParamKey]float32{GrainStrength: 0.4, VignetteStrength: 0.5},
	)
	want := map[ParamKey]float32{GrainStrength: 0.1, HumStrength: 1, VignetteStrength: 0.5}
	for key, w := range want {
		if math.Abs(float64(got[key]-w)) > 1e-6 {
			t.Errorf("%s = %v, want %v", key, got[key], w)
		}
	}
}

func TestEffectsRegistry(t *testing.T) {
	seen := map[string]bool{}
	previous := StagePrepass
	for _, e := range Effects() {
		if seen[e.Name()] {
			t.Errorf("%s listed twice", e.Name())
		}
		seen[e.Name()] = true
		if e.Stage() < previous {
			t.Errorf("%s out of stage order", e.Name())
		}
		previous = e.Stage()
		for _, p := range e.Params() {
			if !strings.HasPrefix(string(p.Key), e.Name()+".") {
				t.Errorf("param %s of %s: key must start with the effect's name", p.Key, e.Name())
			}
			if *p.Value < p.Min || *p.Value > p.Max {
				t.Errorf("param %s: default %v out of %v..%v", p.Key, *p.Value, p.Min, p.Max)
			}
		}
	}
}

// A new TV is warm; PowerOn grows the picture from a dot with a flash,
// PowerOff folds it away and the tube goes dark.
func TestPower(t *testing.T) {
	tv := newTV(t, Setup{Effects: []Effect{NewPower()}})
	tv.Update(tick)
	if w, h, f := tv.Value(PowerWidth), tv.Value(PowerHeight), tv.Value(PowerFlash); w != 1 || h != 1 || f != 0 {
		t.Errorf("new TV: %v %v %v, want warm", w, h, f)
	}

	tv.PowerOn()
	tv.Update(tick)
	if w, h, f := tv.Value(PowerWidth), tv.Value(PowerHeight), tv.Value(PowerFlash); w >= 0.5 || h >= 0.1 || f <= 0 {
		t.Errorf("warming: %v %v %v, want a flashing dot", w, h, f)
	}
	run(tv, 0.6)
	if w, h, f := tv.Value(PowerWidth), tv.Value(PowerHeight), tv.Value(PowerFlash); w != 1 || h != 1 || f != 0 {
		t.Errorf("warm: %v %v %v", w, h, f)
	}

	tv.PowerOff()
	run(tv, 0.5)
	if tv.Dark() {
		t.Error("dark before the picture folded away")
	}
	run(tv, 0.25)
	if !tv.Dark() {
		t.Error("not dark after power off")
	}
	if w := tv.Value(PowerWidth); w != 0 {
		t.Errorf("dead tube width %v", w)
	}
}
