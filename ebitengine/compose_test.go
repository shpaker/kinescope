package ebitengine

import (
	"flag"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func compileEffects(t *testing.T, effects ...kinescope.Effect) {
	t.Helper()
	tv, err := kinescope.NewTV(kinescope.Setup{Effects: effects})
	if err != nil {
		t.Fatal(err)
	}
	source, err := compose(tv.Effects())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ebiten.NewShader(source); err != nil {
		t.Fatalf("%v\n%s", err, source)
	}
}

func TestEveryEffectCompilesAlone(t *testing.T) {
	for _, effect := range kinescope.Effects() {
		t.Run(effect.Name(), func(t *testing.T) {
			compileEffects(t, effect)
		})
	}
}

func TestAllEffectsCompileTogether(t *testing.T) {
	compileEffects(t, kinescope.Effects()...)
}

func TestNoEffectsCompile(t *testing.T) {
	compileEffects(t)
}

func TestPrepassShadersCompile(t *testing.T) {
	if _, err := NewRenderer(); err != nil {
		t.Fatal(err)
	}
}

func TestPrepare(t *testing.T) {
	tv, err := kinescope.NewTV(kinescope.Gorizont())
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Prepare(tv); err != nil {
		t.Fatal(err)
	}
}

func TestGorizontSource(t *testing.T) {
	tv, err := kinescope.NewTV(kinescope.Gorizont())
	if err != nil {
		t.Fatal(err)
	}
	source, err := compose(tv.Effects())
	if err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/gorizont.kage"
	if *update {
		if err := os.WriteFile(golden, source, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != string(want) {
		t.Errorf("composed source differs from %s; run go test -update if that is intended", golden)
	}
}

func TestNames(t *testing.T) {
	if got := uniformName(kinescope.ApertureMaskMinScale); got != "ApertureMaskMinScale" {
		t.Errorf("uniform %q", got)
	}
	if got := functionName("aperture_mask"); got != "apertureMask" {
		t.Errorf("function %q", got)
	}
}
