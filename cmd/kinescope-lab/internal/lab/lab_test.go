package lab

import (
	"encoding/base64"
	"net/url"
	"reflect"
	"slices"
	"testing"

	"github.com/shpaker/kinescope"
)

func ptr[T any](v T) *T { return &v }

// worn is a lab with a bit of everything: a drifting reception driving the
// grain, a shake, glitches and a changed value.
func worn(t *testing.T) *Lab {
	t.Helper()
	l := New("v0.0.0", "")
	for _, c := range []Command{
		{Op: "addSource", Kind: kindDrift},
		{Op: "addSource", Kind: kindSignal},
		{Op: "addDrive"},
		{Op: "addSchedule"},
		{Op: "setSchedule", Name: "glitches", Rate: ptr[float32](2)},
		{Op: "setParam", Key: kinescope.ScanlinesDepth, Value: ptr[float32](0.5)},
		{Op: "seed", Seed: ptr[uint64](7)},
	} {
		l.Do(c)
		if l.notice != "" || l.err != nil {
			t.Fatalf("%s: %s %v", c.Op, l.notice, l.err)
		}
	}
	return l
}

func TestStateRoundTrip(t *testing.T) {
	l := worn(t)
	again := New("v0.0.0", l.State())
	if again.State() != l.State() {
		t.Error("the state changed on the way back")
	}
	if !reflect.DeepEqual(again.Values(), l.Values()) {
		t.Errorf("values %v, want %v", again.Values(), l.Values())
	}
	if got := again.TV().Value(kinescope.ScanlinesDepth); got != 0.5 {
		t.Errorf("scanlines depth = %v, want 0.5", got)
	}
}

func TestStateKeepsThePicture(t *testing.T) {
	l := New("v0.0.0", "")
	l.Do(Command{Op: "picture", Picture: &Picture{Source: PictureScene, Scale: 3, Bypass: true}})
	again := New("v0.0.0", l.State())
	if got, want := again.Picture(), (Picture{Source: PictureScene, Scale: 3, Bypass: true}); got != want {
		t.Errorf("picture %+v, want %+v", got, want)
	}
}

func TestUnknownPictureIsTheTestCard(t *testing.T) {
	l := New("v0.0.0", `j={"preset":"Rubin","picture":{"source":"tv","scale":7}}`)
	if l.Picture() != defaultPicture {
		t.Errorf("picture %+v, want the default", l.Picture())
	}
	if l.preset != "Rubin" {
		t.Errorf("preset %q, want Rubin", l.preset)
	}
}

func TestJSONStateIsTheSame(t *testing.T) {
	l := worn(t)
	data, err := base64.RawURLEncoding.DecodeString(l.State())
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"#j=" + string(data), "#j=" + url.PathEscape(string(data))} {
		if got := New("v0.0.0", text).State(); got != l.State() {
			t.Errorf("%s: not the same lab", text)
		}
	}
}

func TestJSONPresetIsTakenAsItIs(t *testing.T) {
	l := New("v0.0.0", `#j={"preset":"Rubin","seed":3}`)
	want := New("v0.0.0", "")
	want.Do(Command{Op: "preset", Name: "Rubin"})
	want.Do(Command{Op: "seed", Seed: ptr[uint64](3)})
	if l.State() != want.State() {
		t.Error("not Rubin on seed 3")
	}
}

func TestLinkTellsWhatItSkips(t *testing.T) {
	for _, c := range []struct{ link, notice string }{
		{`j={"preset":"Rubn"}`, `The link's skipped: no preset "Rubn"`},
		{`j={"effects":[{"name":"grian"},{"name":"grain","values":{"grain.amount":1,"grain.strength":5}}]}`,
			`The link's skipped: no effect "grian"; no param "grain.amount"; grain.strength out of 0…0.5, set to 0.5`},
		{`j={"sources":[{"name":"x","kind":"noise"}],"schedules":[{"name":"g","mean":5,"spread":1,"rate":1,"episodes":["boom"]}]}`,
			`The link's skipped: no kind of source "noise"; no episode "boom"`},
		{`j={"effect":[]}`, `The link was not read: unknown field "effect"`},
		{`j={"preset":`, `The link was not read: unexpected EOF`},
	} {
		l := New("v0.0.0", c.link)
		if got := l.View().Notice; got != c.notice {
			t.Errorf("%s:\n got %s\nwant %s", c.link, got, c.notice)
		}
	}
}

func TestIdleEpisodeWarns(t *testing.T) {
	l := New("v0.0.0", `j={"effects":[{"name":"grain"}],"schedules":[{"name":"g","mean":5,"spread":1,"rate":1,"episodes":["jitter","ripple"]}]}`)
	if got, want := l.warnings()["schedule:g"], "The TV has none of the effects jitter moves: it shows nothing."; got != want {
		t.Errorf("warning %q, want %q", got, want)
	}
}

func TestBadStateIsIgnored(t *testing.T) {
	for _, state := range []string{"", "#%%%", "on=grain&grain.strength=1", "s=e30"} {
		l := New("v0.0.0", state)
		if l.preset != "Gorizont" || l.TV().Value(kinescope.GrainStrength) != 0.05 {
			t.Errorf("state %q: not Gorizont as it is", state)
		}
	}
}

func TestPresetsAreTakenAsTheyAre(t *testing.T) {
	l := New("v0.0.0", "")
	for _, p := range presets {
		l.Do(Command{Op: "preset", Name: p.name})
		want, got := p.setup(), l.Setup()
		if !reflect.DeepEqual(got.Values(), want.Values()) {
			t.Errorf("%s: values differ", p.name)
		}
		schedulesDiffer := len(got.Schedules)+len(want.Schedules) > 0 && !reflect.DeepEqual(got.Schedules, want.Schedules)
		if schedulesDiffer || len(got.Sources) != len(want.Sources) {
			t.Errorf("%s: schedules or sources differ", p.name)
		}
		if l.preset != p.name {
			t.Errorf("preset %q, want %q", l.preset, p.name)
		}
	}
}

// A setup that does not hold together keeps the TV before it and says
// which item is wrong.
func TestBrokenSetupKeepsTheTV(t *testing.T) {
	l := worn(t)
	tv := l.TV()
	l.Do(Command{Op: "removeSource", Name: "reception"})
	v := l.View()
	if v.Error == "" || !slices.Contains(v.ErrorItems, "drive:0") {
		t.Errorf("error %q on %v, want one on drive:0", v.Error, v.ErrorItems)
	}
	if l.TV() != tv {
		t.Error("the TV was replaced by a broken one")
	}
	l.Do(Command{Op: "removeDrive", Index: 0})
	if l.err != nil || l.TV() == tv {
		t.Errorf("fixed setup: error %v, TV not rebuilt", l.err)
	}
}

func TestEffectsKeepTheirStages(t *testing.T) {
	l := New("v0.0.0", "")
	l.Do(Command{Op: "addEffect", Name: "snow"})
	l.Do(Command{Op: "moveEffect", Name: "hum", Before: "grain"})
	var post []string
	for _, e := range l.effects {
		if e.Stage() == kinescope.StagePost {
			post = append(post, e.Name())
		}
	}
	if want := []string{"hum", "grain", "flicker", "vignette", "snow"}; !slices.Equal(post, want) {
		t.Errorf("post stage %v, want %v", post, want)
	}

	l.Do(Command{Op: "moveEffect", Name: "hum", Before: "curvature"})
	if l.View().Notice == "" {
		t.Error("moved across stages: no notice")
	}
	l.Do(Command{Op: "addEffect", Name: "grain"})
	if l.View().Notice == "" {
		t.Error("added twice: no notice")
	}
}

func TestRenamesFollowDrives(t *testing.T) {
	l := worn(t)
	l.Do(Command{Op: "setDrive", Index: 0, To: string(kinescope.Rate("glitches"))})
	l.Do(Command{Op: "renameSource", Name: "reception", To: "weather"})
	l.Do(Command{Op: "renameSchedule", Name: "glitches", To: "faults"})
	want := kinescope.Drive{From: "weather", To: kinescope.Rate("faults"), Weight: 0.1}
	if l.drives[0] != want {
		t.Errorf("drive %+v, want %+v", l.drives[0], want)
	}
	if l.err != nil {
		t.Error(l.err)
	}
	l.Do(Command{Op: "renameSource", Name: "weather", To: "shake"})
	if l.View().Notice == "" {
		t.Error("renamed to a taken name: no notice")
	}
}

func TestDriveToMissingEffectWarns(t *testing.T) {
	l := worn(t)
	l.Do(Command{Op: "removeEffect", Name: "grain"})
	if l.View().Warnings["drive:0"] == "" {
		t.Error("no warning on a drive to a missing effect")
	}
}

func TestGameFacts(t *testing.T) {
	l := worn(t)
	l.Do(Command{Op: "signal", Name: "shake", Level: ptr[float32](0.4)})
	l.Do(Command{Op: "addEffect", Name: "snow"}) // a new TV keeps the levels
	l.TV().Update(1.0 / 60)
	if got := l.Live().Sources["shake"]; got != 0.4 {
		t.Errorf("shake = %v, want 0.4", got)
	}
	l.Do(Command{Op: "power"})
	for range 60 {
		l.TV().Update(1.0 / 60)
	}
	if !l.TV().Dark() || !l.View().Dark {
		t.Error("power off: the tube is not dark")
	}
}
