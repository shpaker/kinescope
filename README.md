# kinescope

The picture of an old tube TV for Go games: scanlines, a phosphor mask, glow,
afterglow, a bulging glass — and the moods of a worn set: drifting reception,
torn lines, a shiver when the channel changes. Drawn with
[Ebitengine](https://ebitengine.org).

**[Try the lab →](https://shpaker.github.io/kinescope/)** — switch effects on
and off, move their sliders, give the set moods, drop your own screenshot on
it, then share the setup as a link or copy it as Go.

## Used in

- [tnk9x](https://github.com/shpaker/tnk9x) — tanks from the 90s the way we
  remember them, with real-time 2D lighting; the `Gorizont` set.
- [modern-robinson](https://github.com/shpaker/modern-robinson) — a remake of
  the 1999 quest «Новый Робинзон»; the `Rubin` set in its beige case.

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

// In DrawFinalScreen: the frame where geoM puts it, the TV over the whole screen
func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	_ = g.renderer.Draw(screen, offscreen, g.tv, geoM)
}
```

Any scale works: a whole number keeps pixel art crisp, a fraction is
smoothed by `Softness`.

## How it works

A **TV** is a set at work. It is built from a **Setup** — plain data:

- **Effects** — the look of the set. Each effect is a struct of numbers:
  `&kinescope.Scanlines{Depth: 0.3, MinScale: 3}`. Use one or any mix of them.
- **Sources** — values that change in time, from 0 to 1: a `Signal` the game
  sets (a shake), a `Drift` that wanders at random (the reception), a `Wave`.
- **Drives** — a source moving a param: `{From: "shake", To: kinescope.TearStrength, Weight: 1}`.
- **Schedules** — episodes played now and then: `Every{Mean: 90, Spread: 30, Episodes: ...}`.
  An **episode** is a short fault with an envelope: `Jitter()`, `Ripple()`,
  `RollOver()`, `SnowBurst()`, `Degaussing()`.

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
tv.Hold(dragging)                // scheduled glitches wait

tv.PowerOn()                     // the picture grows from a dot
tv.PowerOff()                    // and folds away on quit…
if tv.Dark() { /* …then quit */ }
```

A signal can drive a look, too: the Rubin's monitor case shows only in full
screen, where there is room for it.

```go
setup.Sources["fullscreen"] = kinescope.Signal{}
setup.Drives = append(setup.Drives, kinescope.Drive{
	From: "fullscreen", To: kinescope.CabinetMargin, Weight: 0.15,
})
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
| `Cabinet` | geometry | the monitor's beige case and the dim room around the picture |
| `Power` | geometry | the picture grows from a dot and folds away (`PowerOn`, `PowerOff`) |
| `Curvature` | geometry | the glass bulges the picture |
| `Degauss` | geometry | the picture wobbles as the coil shakes the mask |
| `Roll` | geometry | the picture slips a whole height, the blanking bar with it |
| `Tear` | geometry | lines shift sideways: a running wave and torn bands |
| `Softness` | read | the beam spreads between neighboring pixels |
| `Convergence` | sample | red and blue guns out of register |
| `Glow` | light | light bleeds around bright areas |
| `Glass` | light | black is never quite black; a reflection in the corner |
| `Scanlines` | beam | dark gaps between the beam's lines |
| `Interlace` | beam | the two fields of lines take turns, a shimmer |
| `ApertureMask` | mask | the phosphor's red, green and blue stripes |
| `SlotMask` | mask | the stripes of a shadow-mask set, cut by slots |
| `Grain` | post | fine noise over the picture |
| `Snow` | post | the picture drowns in snow |
| `Flicker` | post | the picture's brightness trembles |
| `Hum` | post | a bar, light or dark, rolls down the picture |
| `Vignette` | post | the corners dim |
| `Corners` | frame | the picture's corners round off |

Presets are named after Soviet sets:

- `Gorizont()` — a well-kept Minsk color set of the eighties: crisp scanlines,
  a faint aperture mask, glowing highlights, a short afterglow.
- `Rubin()` — a well-worn Moscow set in a beige case: a soft beam through a
  slot mask, a dark hum, plenty of grain and a glitch every minute or two.

## Under the hood

```
kinescope            the core: effects as data, TV, modulation — no engine
kinescope/ebitengine the Ebitengine backend: Renderer, Kage shaders
cmd/kinescope-lab    the lab: desktop and web
```

Dependencies point inwards only, checked by depguard: the core knows no
engine, the backend knows no lab. The backend composes **one** picture
shader per set of effects from a Kage fragment per effect, in the order
light goes through a real set (geometry → read → sample → light → beam →
mask → post → frame), plus a prepass for glow and afterglow. Effects a TV
does not have cost nothing.

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
