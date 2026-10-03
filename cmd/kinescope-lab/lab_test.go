package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shpaker/kinescope"
)

func TestStateRoundTrip(t *testing.T) {
	l, err := newLab("")
	if err != nil {
		t.Fatal(err)
	}
	l.effects[0].on = false
	l.moods[0].on = true
	if err := l.rebuild(); err != nil {
		t.Fatal(err)
	}
	l.tv.Apply(map[kinescope.ParamKey]float32{kinescope.GrainStrength: 0.123})

	again, err := newLab(l.encode())
	if err != nil {
		t.Fatal(err)
	}
	if again.effects[0].on || !again.moods[0].on {
		t.Error("toggles lost")
	}
	if got := again.tv.Value(kinescope.GrainStrength); got != 0.123 {
		t.Errorf("grain = %v, want 0.123", got)
	}
}

func TestBadStateIsIgnored(t *testing.T) {
	l, err := newLab("#%%%&grain.strength=abc")
	if err != nil {
		t.Fatal(err)
	}
	if got := l.tv.Value(kinescope.GrainStrength); got != 0.05 {
		t.Errorf("grain = %v, want the default", got)
	}
}

// Copy as Go names types and fields after effects and keys: they must exist.
func TestGoCodeNamesExist(t *testing.T) {
	for _, e := range kinescope.Effects() {
		typ := reflect.TypeOf(e).Elem()
		if typ.Name() != camel(e.Name()) {
			t.Errorf("effect %q is type %s, Go code says %s", e.Name(), typ.Name(), camel(e.Name()))
		}
		for _, p := range e.Params() {
			_, field, _ := strings.Cut(string(p.Key), ".")
			if _, ok := typ.FieldByName(camel(field)); !ok {
				t.Errorf("%s has no field %s for %s", typ.Name(), camel(field), p.Key)
			}
		}
	}
}

func TestGoCodeHasMoods(t *testing.T) {
	l, err := newLab("")
	if err != nil {
		t.Fatal(err)
	}
	for i := range l.moods {
		l.moods[i].on = true
	}
	if err := l.rebuild(); err != nil {
		t.Fatal(err)
	}
	code := l.goCode()
	for _, want := range []string{"&kinescope.Curvature{X: 0.035, Y: 0.045}", "kinescope.Drift{", "kinescope.Every{"} {
		if !strings.Contains(code, want) {
			t.Errorf("Go code lacks %q:\n%s", want, code)
		}
	}
}
