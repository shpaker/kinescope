package lab

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/shpaker/kinescope"
)

// The Go code names types and fields after effects and keys: they must
// exist.
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

// The Go code names a param's key by the library's constant: every key
// must have one, named after its effect and field.
func TestGoCodeKeyConstantsExist(t *testing.T) {
	fset := token.NewFileSet()
	files, err := libraryFiles(fset, 0)
	if err != nil {
		t.Fatal(err)
	}
	constants := make(map[string]string)
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok || len(spec.Values) != len(spec.Names) {
				return true
			}
			for i, name := range spec.Names {
				if lit, ok := spec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					constants[name.Name], _ = strconv.Unquote(lit.Value)
				}
			}
			return true
		})
	}
	for _, e := range kinescope.Effects() {
		for _, p := range e.Params() {
			name := strings.TrimPrefix(keyCode(p.Key), "kinescope.")
			if constants[name] != string(p.Key) {
				t.Errorf("no constant %s = %q", name, p.Key)
			}
		}
	}
}

func TestGoCodeSpeaksTheLibrary(t *testing.T) {
	l := worn(t)
	l.Do(Command{Op: "setDrive", Index: 0, To: string(kinescope.Rate("glitches"))})
	text := joined(l.Code(false))
	for _, want := range []string{
		"func MyTV() kinescope.Setup {",
		"kinescope.NewCurvature(),",
		"&kinescope.Scanlines{Depth: 0.5, MinScale: 3},",
		`"reception": kinescope.Drift{Period: 20},`,
		`{From: "reception", To: kinescope.Rate("glitches"), Weight: 0.1},`,
		"Episodes: []kinescope.Episode{kinescope.Jitter(), kinescope.Ripple()},",
		`tv.Apply(map[kinescope.ParamKey]float32{kinescope.Rate("glitches"): 2})`,
		`shake, err := tv.Signal("shake")`,
		"tv.Play(kinescope.RollOver())",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("Go code lacks %q:\n%s", want, text)
		}
	}
}

func TestSignalVariables(t *testing.T) {
	l := New("v0.0.0", "")
	for _, name := range []string{"bad reception", "tv", "9lives", "type"} {
		l.sources = append(l.sources, source{Name: name, Kind: kindSignal})
	}
	var got []string
	for _, s := range l.signals() {
		got = append(got, s.variable)
	}
	want := []string{"badReception", "tvSignal", "signal9lives", "typeSignal"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("variables %v, want %v", got, want)
	}
}

// TestGoCodeBuilds builds the lab's Go code against the library: the setup
// must make the same TV as the lab's, the whole game must compile.
func TestGoCodeBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds Go code")
	}
	l := worn(t)
	l.Do(Command{Op: "setParam", Key: kinescope.GlowRadius, Value: ptr[float32](4.123)})

	rates := &code{}
	l.writeRates(rates, "tv")
	dir := module(t)
	write(t, dir, "setup.go", "package main\n\n"+joined(l.setupCode())+"\n")
	write(t, dir, "main.go", `package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shpaker/kinescope"
)

func main() {
	setup := MyTV()
	tv, err := kinescope.NewTV(setup)
	if err != nil {
		panic(err)
	}
	`+joined(rates.chunks)+`
	json.NewEncoder(os.Stdout).Encode(struct {
		Values map[kinescope.ParamKey]float32
		Rest   string
	}{tv.Values(), fmt.Sprintf("%d %#v %#v %#v", setup.Seed, setup.Sources, setup.Drives, setup.Schedules)})
}
`)
	out := goCommand(t, dir, "run", ".")
	var got struct {
		Values map[kinescope.ParamKey]float32
		Rest   string
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	setup := l.Setup()
	if !reflect.DeepEqual(got.Values, l.Values()) {
		t.Errorf("values %v, want %v", got.Values, l.Values())
	}
	if want := fmt.Sprintf("%d %#v %#v %#v", setup.Seed, setup.Sources, setup.Drives, setup.Schedules); got.Rest != want {
		t.Errorf("setup %s, want %s", got.Rest, want)
	}

	game := module(t)
	write(t, game, "main.go", joined(l.Code(true))+"\n")
	goCommand(t, game, "vet", ".")
}

// module is a new Go module that uses the library from this repository.
func module(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(libraryDir)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	text := regexp.MustCompile(`(?m)^module .*$`).ReplaceAllString(string(mod), "module labcheck")
	text += fmt.Sprintf("\nrequire github.com/shpaker/kinescope v0.0.0\n\nreplace github.com/shpaker/kinescope => %s\n", root)
	write(t, dir, "go.mod", text)
	write(t, dir, "go.sum", string(sum))
	return dir
}

func goCommand(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off", "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = string(exit.Stderr)
		}
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return out
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func joined(chunks []Chunk) string {
	var texts []string
	for _, c := range chunks {
		texts = append(texts, c.Text)
	}
	return strings.Join(texts, "\n")
}
