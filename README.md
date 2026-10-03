# kinescope

The picture of an old tube TV for Go games: scanlines, a phosphor mask, glow,
afterglow, a bulging glass — and the moods of a worn set: drifting reception,
torn lines, a shiver when the channel changes. Drawn with
[Ebitengine](https://ebitengine.org).

**[Try the lab →](https://shpaker.github.io/kinescope/)** — switch effects on
and off, move their sliders, give the set moods, drop your own screenshot on
it, then share the setup as a link or copy it as Go.

## Quick start

```go
import (
	"github.com/shpaker/kinescope"
	"github.com/shpaker/kinescope/ebitengine"
)

tv, err := kinescope.NewTV(kinescope.Gorizont())
renderer, err := ebitengine.NewRenderer()

// Every tick
tv.Update(1.0 / 60)

// In DrawFinalScreen: the game's low-resolution frame, scaled by a whole number
err = renderer.Draw(screen, offscreen, tv, x, y, scale)
```

## How it works

A **TV** is a set at work. It is built from a **Setup** — plain data:

- **Effects** — the look of the set. Each effect is a struct of numbers:
  `&kinescope.Scanlines{Depth: 0.3, MinScale: 3}`. Use one or any mix of them.
- **Sources** — values that change in time, from 0 to 1: a `Signal` the game
  sets (a shake), a `Drift` that wanders at random (the reception), a `Wave`.
- **Drives** — a source moving a param: `{From: "shake", To: kinescope.TearStrength, Weight: 1}`.
- **Schedules** — episodes played now and then: `Every{Mean: 90, Spread: 30, Episodes: ...}`.
  An **episode** is a short fault with an envelope: `kinescope.Jitter()`, `kinescope.Ripple()`.

The game talks to the TV in facts — a signal's level, an episode to play —
and never holds the effects themselves:

```go
setup := kinescope.Gorizont()
setup.Sources = map[string]kinescope.Source{
	"shake":     kinescope.Signal{},
	"reception": kinescope.Drift{Period: 20},
}
setup.Drives = []kinescope.Drive{
	{From: "shake", To: kinescope.TearStrength, Weight: 1},
	{From: "reception", To: kinescope.GrainStrength, Weight: 0.08},
	{From: "reception", To: kinescope.Rate("glitches"), Weight: 2}, // more glitches on poor reception
}
setup.Schedules = map[string]kinescope.Every{
	"glitches": {Mean: 90, Spread: 30, Episodes: []kinescope.Episode{kinescope.Jitter()}},
}
tv, err := kinescope.NewTV(setup)

shake, err := tv.Signal("shake") // once; a typo is an error here, not a dead knob in play
shake.Set(level)                 // whenever the game likes
tv.Play(kinescope.Ripple())      // on a change of scene
```

Modulation adds to a param's base and never changes it, so `tv.Values()` is
always what the player set, even in the middle of a glitch.

### Player settings

Changing a value is free: values are shader uniforms, sent every frame
anyway. Changing the *set* of effects (`tv.SetEffects`) composes and compiles
a new shader, which can stall a frame — do it in a menu, not in play. To let
a player switch an effect off, keep it in the set at strength 0.

`tv.Values()` and `tv.Apply()` save and restore the player's values;
`kinescope.Blend(t, a, b)` turns one knob from one preset to another.

### Pointer and touch

The glass bends the picture. `tv.Map(x, y, w, h)` is the point of the frame
the TV shows at a point — hit what the player sees.

## Effects

| Effect | Stage | What it does |
|---|---|---|
| `Afterglow` | prepass | bright moving things leave a short, cold trail |
| `Curvature` | geometry | the glass bulges the picture |
| `Tear` | geometry | lines shift sideways in a running wave |
| `Convergence` | sample | red and blue guns out of register towards the edges |
| `Glow` | light | light bleeds around bright areas |
| `Glass` | light | black is never quite black; a reflection in the corner |
| `Scanlines` | beam | dark gaps between the beam's lines |
| `ApertureMask` | mask | the phosphor's red, green and blue stripes |
| `Grain` | post | the signal's snow |
| `Flicker` | post | the picture's brightness trembles |
| `Hum` | post | a light bar rolls down the picture |
| `Vignette` | post | the corners dim |
| `Corners` | frame | the picture's corners round off |

Presets are named after Soviet sets: `Gorizont()` is a well-kept Minsk colour
set of the eighties.

## Under the hood

```
kinescope            the core: effects as data, TV, modulation — no engine
kinescope/ebitengine the Ebitengine backend: Renderer, Kage shaders
cmd/kinescope-lab    the lab: desktop and web
```

Dependencies point inwards only, checked by depguard: the core knows no
engine, the backend knows no lab. The backend composes **one** picture
shader per set of effects from a Kage fragment per effect, in the order
light goes through a real set (geometry → sample → light → beam → mask →
post → frame), plus a prepass for glow and afterglow. Effects a TV does not
have cost nothing.

Tests check that every effect compiles alone and with all the others, and
that the shader's geometry agrees with `TV.Map` pixel for pixel.

## Lab

```bash
go run github.com/shpaker/kinescope/cmd/kinescope-lab@latest
```

On the desktop, *Copy link* and *Copy as Go* print to the terminal, and
`-state` takes a printed state back.

## License

[MIT](LICENSE)
