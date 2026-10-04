package lab

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// labURL is where the lab is published.
const labURL = "https://shpaker.github.io/kinescope/"

// llmsExamples are the links llms.txt shows: what each makes, and its JSON.
var llmsExamples = []struct{ what, json string }{
	{"A preset as it is", `{"preset":"Rubin"}`},
	{"A preset on the moving scene, three times as large",
		`{"preset":"Gorizont","picture":{"source":"scene","scale":3}}`},
	{"A worn set: reception drifting over 20 s thickens the grain, glitches every 6–14 s",
		`{"seed":7,"effects":[{"name":"curvature"},{"name":"scanlines","values":{"scanlines.depth":0.6}},` +
			`{"name":"grain","values":{"grain.strength":0.05}},{"name":"jitter"},{"name":"ripple"},{"name":"vignette"}],` +
			`"sources":[{"name":"reception","kind":"drift","period":20}],` +
			`"drives":[{"from":"reception","to":"grain.strength","weight":0.2}],` +
			`"schedules":[{"name":"glitches","mean":10,"spread":4,"rate":1,"episodes":["jitter","ripple"]}]}`},
}

// TestLLMsText checks web/llms.txt says what the lab is now; with -update
// (go generate) it writes it.
func TestLLMsText(t *testing.T) {
	path := filepath.Join(libraryDir, "web", "llms.txt")
	want := generateLLMsText()
	if *update {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("web/llms.txt is stale: run go generate ./cmd/kinescope-lab/...")
	}
}

// TestLLMsExamplesLoad checks every example of llms.txt makes a TV, and
// not the lab's first preset.
func TestLLMsExamplesLoad(t *testing.T) {
	first := New("v0.0.0", "").State()
	for _, e := range llmsExamples {
		l := New("v0.0.0", "")
		l.Load(llmsAddress(e.json))
		if l.err != nil || len(l.badItems()) > 0 || len(l.warnings()) > 0 {
			t.Errorf("%s: %v %v %v", e.what, l.err, l.badItems(), l.warnings())
		}
		if l.State() == first {
			t.Errorf("%s: the lab's first preset", e.what)
		}
	}
}

// llmsAddress is the address's part after the lab's URL for a setup in
// JSON.
func llmsAddress(json string) string {
	return "#j=" + url.PathEscape(json)
}

// generateLLMsText is web/llms.txt: how an agent writes a link to a setup
// in the lab, and everything a setup can be made of.
func generateLLMsText() []byte {
	var b bytes.Buffer
	w := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	number := func(v float32) string { return strconv.FormatFloat(float64(v), 'g', -1, 32) }

	w("# kinescope lab")
	w("")
	w("> kinescope draws the picture of an old tube TV for Go games (Ebitengine).")
	w("> The lab, %s, shows a setup — effects, sources, drives,", labURL)
	w("> schedules — on a picture and writes it out as Go. A link opens the lab on")
	w("> a setup: write one to show someone a TV.")
	w("")
	w("Library and docs: https://github.com/shpaker/kinescope")
	w("")
	w("## Links")
	w("")
	w("    %s#j=<JSON>", labURL)
	w("")
	w("The JSON is the setup below; percent-encode it (encodeURIComponent). The lab")
	w("rewrites the address to its own compact form, #s=<base64>, which its \"Copy")
	w("link\" button gives too. Every field may be left out; what the lab cannot")
	w("read it skips.")
	w("")
	w("    {")
	w("      \"preset\": \"Gorizont\",   // with no effects: the preset as it is")
	w("      \"seed\": 7,              // the randomness of grain, glitches and the like")
	w("      \"effects\": [            // in stage order; values left out stay at defaults")
	w("        {\"name\": \"scanlines\", \"values\": {\"scanlines.depth\": 0.6}}")
	w("      ],")
	w("      \"sources\": [            // values that change by themselves or by the game")
	w("        {\"name\": \"reception\", \"kind\": \"drift\", \"period\": 20}")
	w("      ],")
	w("      \"drives\": [             // weight × the source's value is added to the param")
	w("        {\"from\": \"reception\", \"to\": \"grain.strength\", \"weight\": 0.2}")
	w("      ],")
	w("      \"schedules\": [          // episodes at random intervals: mean ± spread seconds")
	w("        {\"name\": \"glitches\", \"mean\": 10, \"spread\": 4, \"rate\": 1, \"episodes\": [\"jitter\"]}")
	w("      ],")
	w("      \"picture\": {\"source\": \"test-card\", \"scale\": 0, \"bypass\": false}")
	w("    }")
	w("")
	w("Rules:")
	w("")
	w("- A TV has at most one effect in each of the stages %s.", strings.Join(catalog().Single, " and "))
	w("- A param's key is <effect>.<field>; values outside its range are clamped.")
	w("- A drive goes to a param of an effect the setup has, or to rate.<schedule>,")
	w("  which speeds that schedule up or slows it down.")
	w("- Drift and wave sources need a period > 0 in seconds; a signal is a level the")
	w("  game sets (0…1), set by hand under the picture in the lab.")
	w("- A schedule needs mean > 0, 0 ≤ spread < mean and at least one episode;")
	w("  its rate is 1 at the normal speed, and 0, if left out, stops it.")
	w("- An effect whose strength rests at 0 shows only when a drive or an episode")
	w("  moves it.")
	w("")
	w("## Presets")
	w("")
	for _, p := range presets {
		w("- %s", p.name)
	}
	w("")
	w("## Picture")
	w("")
	w("- source: %s (a dropped picture cannot travel in a link: the test card shows)", strings.Join(pictures, ", "))
	scaleNames := make([]string, len(scales))
	for i, s := range scales {
		scaleNames[i] = strconv.Itoa(s)
	}
	w("- scale: %s — screen pixels per picture pixel; 0 fits the screen", strings.Join(scaleNames, ", "))
	w("- bypass: true shows the picture without the TV")
	w("")
	w("## Effects")
	w("")
	w("By stage, in the order the picture goes through them. Each param: key, range,")
	w("default.")
	for _, e := range catalog().Effects {
		w("")
		w("### %s (%s)", e.Name, e.Stage)
		w("")
		w("%s", e.Doc)
		params := newEffect(e.Name).Params()
		if len(params) > 0 {
			w("")
		}
		for _, p := range params {
			line := fmt.Sprintf("- %s: %s…%s, default %s", p.Key, number(p.Min), number(p.Max), number(*p.Value))
			if doc := fieldDocs[p.Key]; doc != "" {
				line += " — " + doc
			}
			w("%s", line)
		}
	}
	w("")
	w("## Kinds of sources")
	w("")
	for _, k := range catalog().Kinds {
		w("- %s: %s", k.Name, k.Doc)
	}
	w("")
	w("## Episodes")
	w("")
	for _, e := range catalog().Episodes {
		w("- %s: %s", e.Name, e.Doc)
	}
	w("")
	w("## Examples")
	for _, e := range llmsExamples {
		w("")
		w("%s:", e.what)
		w("")
		w("    %s", e.json)
		w("    %s%s", labURL, llmsAddress(e.json))
	}
	return b.Bytes()
}
