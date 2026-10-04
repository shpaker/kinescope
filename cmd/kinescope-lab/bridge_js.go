//go:build js

package main

import (
	"encoding/json"
	"log"
	"syscall/js"

	"github.com/shpaker/kinescope"
	"github.com/shpaker/kinescope/cmd/kinescope-lab/internal/lab"
	"github.com/shpaker/kinescope/ebitengine"
)

// bridge is the lab as the page sees it: window.kinescope, whose functions
// take and give JSON. It is the only place that knows the browser.
//
//	view()          the whole view
//	do(command)     runs a command and returns the view
//	live()          the values now, the modulation on
//	state()         the setup as a string for the address
//	load(state)     takes such a string, or the address's j=<JSON>, and returns the view
type bridge struct {
	game *game

	// The shader's source, kept until the TV or its effects change
	kage     string
	tv       *kinescope.TV
	revision int
}

// page is the view the page gets: the lab's, the shader and whether a
// picture has been dropped.
type page struct {
	lab.View
	Kage    string `json:"kage"`
	Dropped bool   `json:"dropped"`
}

func newBridge(g *game) *bridge {
	return &bridge{game: g}
}

// publish puts the bridge on the page's window. The functions live as long
// as the page.
func (b *bridge) publish() {
	api := js.Global().Get("Object").New()
	api.Set("view", b.func0(b.view))
	api.Set("live", b.func0(b.live))
	api.Set("state", b.func0(func() string { return b.game.lab.State() }))
	api.Set("do", b.func1(b.do))
	api.Set("load", b.func1(func(state string) string {
		b.game.lab.Load(state)
		return b.view()
	}))
	js.Global().Set("kinescope", api)
}

// func0 and func1 wrap a function for the page: it holds the game's lock
// while it runs.
func (b *bridge) func0(f func() string) js.Func {
	return js.FuncOf(func(js.Value, []js.Value) any {
		b.game.mu.Lock()
		defer b.game.mu.Unlock()
		return f()
	})
}

func (b *bridge) func1(f func(string) string) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		b.game.mu.Lock()
		defer b.game.mu.Unlock()
		if len(args) == 0 {
			return f("")
		}
		return f(args[0].String())
	})
}

func (b *bridge) view() string {
	p := page{View: b.game.lab.View(), Kage: b.shader(), Dropped: b.game.dropped()}
	data, err := json.Marshal(p)
	if err != nil {
		log.Print(err)
		return "{}"
	}
	return string(data)
}

// do runs a command of the lab's.
func (b *bridge) do(command string) string {
	var c lab.Command
	if err := json.Unmarshal([]byte(command), &c); err != nil {
		log.Print(err)
		return b.view()
	}
	b.game.lab.Do(c)
	b.game.failed = false
	return b.view()
}

func (b *bridge) live() string {
	data, err := json.Marshal(b.game.lab.Live())
	if err != nil {
		log.Print(err)
		return "{}"
	}
	return string(data)
}

// shader is the Kage source of the TV's picture shader.
func (b *bridge) shader() string {
	tv := b.game.lab.TV()
	if tv == b.tv && tv.Revision() == b.revision {
		return b.kage
	}
	b.tv, b.revision = tv, tv.Revision()
	source, err := ebitengine.Source(tv)
	if err != nil {
		b.kage = "// " + err.Error()
	} else {
		b.kage = string(source)
	}
	return b.kage
}
