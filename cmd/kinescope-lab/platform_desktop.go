//go:build !js

package main

import (
	"flag"
	"fmt"
)

// desktop keeps the state in memory, takes it from the -state flag and
// prints what it exports.
type desktop struct {
	state string
}

var _ platform = (*desktop)(nil)

func newPlatform() platform {
	state := flag.String("state", "", "the lab's state, as printed by Copy link")
	flag.Parse()
	return &desktop{state: *state}
}

func (d *desktop) State() string         { return d.state }
func (d *desktop) SetState(state string) { d.state = state }

func (d *desktop) Export(label string, text string) {
	fmt.Printf("── %s ──\n%s\n", label, text)
}

func (d *desktop) Link(state string) string {
	return fmt.Sprintf("kinescope-lab -state '%s'", state)
}
