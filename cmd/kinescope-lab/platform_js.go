//go:build js

package main

import "syscall/js"

// browser keeps the state in the page's address and exports to the
// clipboard.
type browser struct {
	location js.Value
	history  js.Value
}

var _ platform = (*browser)(nil)

func newPlatform() platform {
	window := js.Global()
	return &browser{location: window.Get("location"), history: window.Get("history")}
}

func (b *browser) State() string {
	return b.location.Get("hash").String()
}

// SetState replaces the address's hash without adding to the history: the
// back button leaves the lab, not the last slider move.
func (b *browser) SetState(state string) {
	b.history.Call("replaceState", js.Null(), "", "#"+state)
}

func (b *browser) Export(_ string, text string) {
	clipboard := js.Global().Get("navigator").Get("clipboard")
	if clipboard.Truthy() {
		clipboard.Call("writeText", text)
	}
}

func (b *browser) Link(state string) string {
	return b.location.Get("origin").String() + b.location.Get("pathname").String() + "#" + state
}
