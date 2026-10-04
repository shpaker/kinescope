package lab

import (
	"fmt"
	"go/format"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/shpaker/kinescope"
)

// Chunk is a piece of Go code, whole lines without the last line break,
// and the item of the page it writes out: "effect:grain", "drive:0",
// "game:play:jitter". ID is empty for the code around the items.
type Chunk struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Code is the setup written out in Go: a function returning it, like the
// library's presets, and what a game tells the TV at any moment. Full
// makes it a whole Ebitengine game to run.
func (l *Lab) Code(full bool) []Chunk {
	if full {
		return l.fullCode()
	}
	game := &code{}
	l.writeGame(game, "tv", true)
	chunks := append(l.setupCode(), Chunk{Text: ""})
	return append(chunks, game.formatted("package main\n\nfunc _() {\n", "}\n")...)
}

// setupCode is the function that returns the setup, with its import.
func (l *Lab) setupCode() []Chunk {
	c := &code{}
	c.add("", `import "github.com/shpaker/kinescope"`)
	c.add("", "")
	l.writeSetup(c)
	return c.formatted("package main\n\n", "")
}

// fullCode is a whole game that shows a picture through the setup's TV.
func (l *Lab) fullCode() []Chunk {
	c := &code{}
	c.add("", "package main")
	c.add("", "")
	c.add("", "import (")
	c.add("", `"log"`)
	c.add("", "")
	c.add("", `"github.com/hajimehoshi/ebiten/v2"`)
	c.add("", "")
	c.add("", `"github.com/shpaker/kinescope"`)
	c.add("", `"github.com/shpaker/kinescope/ebitengine"`)
	c.add("", ")")
	c.add("", "")
	l.writeSetup(c)
	c.add("", "")
	c.add("", "// game is a game seen through the TV.")
	c.add("", "type game struct {")
	c.add("", "tv *kinescope.TV")
	c.add("", "renderer *ebitengine.Renderer")
	for _, s := range l.signals() {
		c.add("game:signal:"+s.name, "%s *kinescope.Level", s.variable)
	}
	c.add("", "}")
	c.add("", "")
	c.add("", "func (g *game) Update() error {")
	c.add("", "g.tv.Update(1 / float64(ebiten.TPS()))")
	l.writeGame(c, "g.tv", false)
	c.add("", "return nil")
	c.add("", "}")
	c.add("", "")
	c.add("", "func (g *game) Draw(screen *ebiten.Image) {")
	c.add("", "// The game draws its picture here, at its own size")
	c.add("", "}")
	c.add("", "")
	c.add("", "// DrawFinalScreen draws the game's picture through the TV.")
	c.add("", "func (g *game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {")
	c.add("", "if err := g.renderer.Draw(screen, offscreen, g.tv, geoM); err != nil {")
	c.add("", "log.Print(err)")
	c.add("", "}")
	c.add("", "}")
	c.add("", "")
	c.add("", "func (g *game) Layout(int, int) (int, int) { return 320, 240 }")
	c.add("", "")
	c.add("", "func main() {")
	c.add("", "tv, err := kinescope.NewTV(MyTV())")
	c.add("", "if err != nil {")
	c.add("", "log.Fatal(err)")
	c.add("", "}")
	l.writeRates(c, "tv")
	c.add("", "renderer, err := ebitengine.NewRenderer()")
	c.add("", "if err != nil {")
	c.add("", "log.Fatal(err)")
	c.add("", "}")
	c.add("", "g := &game{tv: tv, renderer: renderer}")
	for _, s := range l.signals() {
		id := "game:signal:" + s.name
		c.add(id, "if g.%s, err = tv.Signal(%q); err != nil {", s.variable, s.name)
		c.add(id, "log.Fatal(err)")
		c.add(id, "}")
	}
	c.add("", "if err := ebiten.RunGame(g); err != nil {")
	c.add("", "log.Fatal(err)")
	c.add("", "}")
	c.add("", "}")
	return c.formatted("", "")
}

// writeSetup writes the function that returns the setup.
func (l *Lab) writeSetup(c *code) {
	c.add("", "// MyTV is the set made in the kinescope lab.")
	c.add("", "func MyTV() kinescope.Setup {")
	c.add("", "return kinescope.Setup{")
	if l.seed != 0 {
		c.add("", "Seed: %d,", l.seed)
	}
	c.add("", "Effects: []kinescope.Effect{")
	for _, e := range l.effects {
		c.add("effect:"+e.Name(), "%s,", effectCode(e))
	}
	c.add("", "},")
	if len(l.sources) > 0 {
		c.add("", "Sources: map[string]kinescope.Source{")
		for _, s := range l.sources {
			c.add("source:"+s.Name, "%q: %s,", s.Name, sourceCode(s))
		}
		c.add("", "},")
	}
	if len(l.drives) > 0 {
		c.add("", "Drives: []kinescope.Drive{")
		for i, d := range l.drives {
			c.add(fmt.Sprintf("drive:%d", i), "{From: %q, To: %s, Weight: %s},",
				d.From, keyCode(d.To), number(d.Weight))
		}
		c.add("", "},")
	}
	if len(l.schedules) > 0 {
		c.add("", "Schedules: map[string]kinescope.Every{")
		for _, s := range l.schedules {
			id := "schedule:" + s.Name
			c.add(id, "%q: {", s.Name)
			c.add(id, "Mean: %s,", number(s.Mean))
			c.add(id, "Spread: %s,", number(s.Spread))
			var calls []string
			for _, name := range s.Episodes {
				if e, ok := findEpisode(name); ok {
					calls = append(calls, "kinescope."+e.constructor+"()")
				}
			}
			c.add(id, "Episodes: []kinescope.Episode{%s},", strings.Join(calls, ", "))
			c.add(id, "},")
		}
		c.add("", "},")
	}
	c.add("", "}")
	c.add("", "}")
}

// writeGame writes what a game does with its TV: tv is how the code names
// it. Alone, the code builds the TV first; in a game's Update the facts
// that would fire every tick are left as comments.
func (l *Lab) writeGame(c *code, tv string, alone bool) {
	if alone {
		c.add("", "// In the game: build the TV once, then tell it facts at any moment")
		c.add("", "%s, err := kinescope.NewTV(MyTV())", tv)
		l.writeRates(c, tv)
	} else {
		c.add("", "")
		c.add("", "// The game's facts, at any moment")
	}
	for _, s := range l.signals() {
		id := "game:signal:" + s.name
		variable := s.variable
		if alone {
			c.add(id, "%s, err := %s.Signal(%q)", variable, tv, s.name)
		} else {
			variable = "g." + variable
		}
		c.add(id, "%s.Set(%s)", variable, number(l.levels[s.name]))
	}
	comment := ""
	if !alone {
		comment = "// "
	}
	for _, e := range episodes {
		c.add("game:play:"+e.make().Name, "%s%s.Play(kinescope.%s())", comment, tv, e.constructor)
	}
	if l.off {
		c.add("game:power", "%s%s.PowerOff() // the picture folds away; %s.Dark() reports it gone", comment, tv, tv)
	} else {
		c.add("game:power", "%s%s.PowerOn() // the picture grows from a dot", comment, tv)
	}
	c.add("game:hold", "%s%s.Hold(%t) // scheduled episodes wait while held", comment, tv, l.held)
	c.add("game:reset", "%s%s.Reset() // no episode playing, no afterglow left", comment, tv)
}

// writeRates writes the schedules' rates that differ from 1: a setup has no
// place for them, the TV takes them as values.
func (l *Lab) writeRates(c *code, tv string) {
	var rates []string
	for _, s := range l.schedules {
		if s.Rate != 1 {
			rates = append(rates, fmt.Sprintf("%s: %s", keyCode(kinescope.Rate(s.Name)), number(s.Rate)))
		}
	}
	if len(rates) > 0 {
		c.add("", "%s.Apply(map[kinescope.ParamKey]float32{%s})", tv, strings.Join(rates, ", "))
	}
}

// effectCode is an effect in Go: its constructor at its defaults, a
// literal with every field otherwise — a field left out would be zero.
func effectCode(e kinescope.Effect) string {
	typ := camel(e.Name())
	fresh := newEffect(e.Name()).Params()
	params := e.Params()
	if slices.EqualFunc(params, fresh, func(a, b kinescope.Param) bool { return *a.Value == *b.Value }) {
		return "kinescope.New" + typ + "()"
	}
	var fields []string
	for _, p := range params {
		_, field, _ := strings.Cut(string(p.Key), ".")
		fields = append(fields, camel(field)+": "+number(*p.Value))
	}
	return "&kinescope." + typ + "{" + strings.Join(fields, ", ") + "}"
}

func sourceCode(s source) string {
	for _, k := range kinds {
		if k.name == s.Kind && s.Kind != kindSignal {
			return "kinescope." + k.typ + "{Period: " + number(s.Period) + "}"
		}
	}
	return "kinescope.Signal{}"
}

// keyCode is a param's key in Go: the library's constant, or Rate for a
// schedule's rate.
func keyCode(key kinescope.ParamKey) string {
	if name, ok := strings.CutPrefix(string(key), "rate."); ok {
		return fmt.Sprintf("kinescope.Rate(%q)", name)
	}
	effect, field, _ := strings.Cut(string(key), ".")
	return "kinescope." + camel(effect) + camel(field)
}

// number is a float32 in Go: the shortest form that reads back the same.
func number(v float32) string {
	return strconv.FormatFloat(float64(v), 'g', -1, 32)
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

// Signals in Go

// signal is a signal source and the Go variable that holds its level.
type signal struct {
	name     string
	variable string
}

// taken are names the code uses itself, or Go keeps for itself.
var taken = []string{"tv", "err", "g", "game", "renderer", "main", "log", "ebiten", "ebitengine", "kinescope", "screen", "offscreen", "geoM"}

// signals are the setup's signals with a variable each.
func (l *Lab) signals() []signal {
	var signals []signal
	used := slices.Clone(taken)
	for _, s := range l.sources {
		if s.Kind != kindSignal {
			continue
		}
		variable := identifier(s.Name)
		if slices.Contains(used, variable) || token.IsKeyword(variable) {
			variable += "Signal"
		}
		variable = freeName(variable, used)
		used = append(used, variable)
		signals = append(signals, signal{name: s.Name, variable: variable})
	}
	return signals
}

// identifier is a name in Go's lowerCamelCase: "bad reception" →
// badReception.
func identifier(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var b strings.Builder
	for i, word := range words {
		if i == 0 {
			b.WriteString(strings.ToLower(word[:1]) + word[1:])
		} else {
			b.WriteString(strings.ToUpper(word[:1]) + word[1:])
		}
	}
	id := b.String()
	if id == "" || !unicode.IsLetter([]rune(id)[0]) {
		id = "signal" + id
	}
	return id
}

// Writing code

// code is Go code being written, chunk by chunk.
type code struct {
	chunks []Chunk
}

// add adds a line to the chunk of id, or starts one.
func (c *code) add(id, format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	if n := len(c.chunks); n > 0 && c.chunks[n-1].ID == id {
		c.chunks[n-1].Text += "\n" + line
		return
	}
	c.chunks = append(c.chunks, Chunk{ID: id, Text: line})
}

// formatted is the code laid out by gofmt, within before and after where it
// is not a file by itself. Gofmt keeps the lines, so the chunks keep
// theirs; if it fails, the code stays as written.
func (c *code) formatted(before, after string) []Chunk {
	var lines []string
	for _, chunk := range c.chunks {
		lines = append(lines, strings.Split(chunk.Text, "\n")...)
	}
	source, err := format.Source([]byte(before + strings.Join(lines, "\n") + "\n" + after))
	if err != nil {
		return c.chunks
	}
	text := strings.TrimSuffix(strings.TrimPrefix(string(source), before), after)
	out := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(out) != len(lines) {
		return c.chunks
	}
	if after != "" { // the code was inside a function: take its indent off
		for i, line := range out {
			out[i] = strings.TrimPrefix(line, "\t")
		}
	}
	var chunks []Chunk
	for _, chunk := range c.chunks {
		n := strings.Count(chunk.Text, "\n") + 1
		chunks = append(chunks, Chunk{ID: chunk.ID, Text: strings.Join(out[:n], "\n")})
		out = out[n:]
	}
	return chunks
}
