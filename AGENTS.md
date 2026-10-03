# Rules for AI agents

kinescope draws the picture of an old tube TV for Go games. Module
`github.com/shpaker/kinescope`, Ebitengine backend.

## Architecture

Dependencies point inwards only: outer layers depend on inner ones, the core depends on
nothing. depguard (`.golangci.yml`) checks the layers.

```
kinescope            core: effects as data, TV, sources/drives/episodes/schedules, presets
kinescope/ebitengine backend: Renderer, Kage fragments, composition, prepasses
cmd/kinescope-lab    the lab: model (lab.go), UI (ui.go), platform (platform_*.go)
web/                 the lab's page for the web build
```

Architecture and the right abstractions come first.

- The core never imports Ebitengine or the backend. The backend never imports the lab.
- An effect is plain data: a struct of float32 params, `Name()`, `Stage()`, `Params()`.
  GPU resources and per-frame state live only in the backend's `Renderer`.
- The set of effects is closed (unexported `clone`): a new effect is added to the core,
  to `kinescope.Effects()` and as a fragment in `ebitengine/shaders/fragments/<name>.kage`.
- The fragment contract (stage signatures, hooks into later stages such as `<name>Post`,
  the helpers) is documented in `ebitengine/compose.go`. A TV has at most one effect in
  the read and sample stages.
- Param keys are `<effect name>.<field in snake_case>`; uniforms and the lab's Go export
  are derived from them. Keep the convention.
- Geometry effects implement the core's `warper` (CPU mirror); the shader must agree
  with it — `TestShaderGeometryMatchesMap` checks.
- Time comes from the game (`TV.Update(dt)`); randomness from the setup's seed. No wall
  clock, no global rand, no global state.
- The game talks to a TV in facts (signal levels, episodes) and never holds its effects.

## Platforms

The lab builds for the desktop and the web (js/wasm, GitHub Pages); the library must work
on every platform Ebitengine supports. A change for one platform must not break the others
or change their behavior.

- Platform specifics only behind the lab's `platform` interface, with implementations by
  build tags (`!js` — desktop, `js` — web). `syscall/js` only in `platform_js.go`.
- Branching by platform only with build tags; the core and the backend have none.
- A change for one platform is checked by building and linting every target.
- No secrets or tokens in the repository, CI or metadata.

## Principles

- Every dependency comes through a `New*` constructor; no setters and no nil checks of
  dependencies, no configuring an object after it is built.
- Every implementation of an interface has a compile-time check: `var _ X = (*Y)(nil)`.
- One type — one responsibility.
- No deprecated APIs (e.g. `rand.Seed`, `imageSrcNAt` in Kage — use `imageSrcNAtFromSrc0Pos`).

## Effects and shaders

- Read and follow the official Ebitengine skills before engine work:
  [writing-kage-shaders](https://github.com/hajimehoshi/ebiten/tree/main/skills/writing-kage-shaders),
  [efficient-ebitengine-rendering](https://github.com/hajimehoshi/ebiten/tree/main/skills/efficient-ebitengine-rendering),
  [avoiding-blurry-ebitengine-rendering](https://github.com/hajimehoshi/ebiten/tree/main/skills/avoiding-blurry-ebitengine-rendering),
  [run-ebitengine-app-headless](https://github.com/hajimehoshi/ebiten/tree/main/skills/run-ebitengine-app-headless).
- Effects are written from scratch. Third-party shaders (GLSL, libretro, Shadertoy) are not
  translated into Kage. If someone's work inspired an effect, check its license and credit
  it in README under Inspiration (author, link, license).
- A frame allocates nothing: uniforms are one-element slices reused between frames.

## Naming

- Constructors `New*`; getters without `Get` only where Go idiom asks (`Value`, `Params`),
  setters `Set*`.
- No abbreviations: `renderer`, `schedule`, not `rnd`-style names for fields that live long.
- Short clear names: `Image`, not `ImageGetter`.
- Fields as private as possible, grouped by kind with a comment per group.
- Methods grouped by area, in a logical order within a group.

## Style

- Standard Go conventions; short variable declarations where they fit.
- Comments and docs in English, plain words.
- Document every exported identifier and say which interfaces a type implements.
- Tests as simple and readable as possible.

## Required after changes

- `just fmt`, `just lint`, `just lint-wasm`, `just test`, `just build-web`; the linter stays
  clean for both targets (desktop and js/wasm).
- Find and update every use of a changed identifier; update the tests.
- Update README when an effect, a source or the API changes: what a game developer sees —
  short and to the point.

## Git

- Any writing git operation (commit, push, merge, rebase, reset, tag and the like) only with
  the user's explicit confirmation.
- Commits are made as the user only.
- No mention of AI or neural networks (`Co-Authored-By`, `Generated with ...`, `claude/`,
  `ai/` and the like) in commits, their metadata, branch names and PRs.
- PR and release descriptions as short as possible; the only formatting is a bulleted list,
  no headings or emphasis.
- Conventional Commits: `feat`, `fix`, `perf`, `refactor` raise the minor version,
  `type!:` or a `BREAKING CHANGE:` footer the major (while at v0 — the minor); `docs`,
  `chore`, `ci`, `test`, `style`, `build` make no release.
- Release tags (`vMAJOR.MINOR.0`, as Go modules require) are set by CI
  (`.github/workflows/version.yml`), run by hand on main with a green CI: changes pile up,
  a release goes out when ready; it tags, publishes the release with its changelog and
  deploys the lab to Pages. Never tag by hand.

## CI/CD

- OS dependencies (Linux packages) are the same in every job and pipeline; when they
  change, update every job that uses them.
