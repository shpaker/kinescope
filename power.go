package kinescope

import "math"

// The power's timing, in seconds. Warming up shows a dot, then a line,
// then the picture; dying folds the picture to a line, the line to a dot,
// and fades the dot.
const (
	warmDot  = 0.12
	warmUp   = 0.55
	dieLine  = 0.18
	dieDot   = 0.32
	coolDown = 0.7
	// The thinnest the raster gets, still a visible line or dot
	thinWarm = 0.006
	thinDie  = 0.004
)

// power is the TV's power: warming up since PowerOn or dying since
// PowerOff. A new TV is warm already.
type power struct {
	dying bool
	since float64 // seconds since the last PowerOn or PowerOff
}

func newPower() power { return power{since: warmUp} }

// raster is the raster's squeeze — its width and height, 1 whole — and the
// beam's flash.
func (p power) raster() (width, height, flash float64) {
	k := p.since
	if p.dying {
		switch {
		case k < dieLine:
			u := k / dieLine
			return 1, math.Max(thinDie, 1-u*u), 0.8 * u
		case k < dieDot:
			u := (k - dieLine) / (dieDot - dieLine)
			return math.Max(thinDie, 1-u), thinDie, 0.9
		case k < coolDown:
			u := (k - dieDot) / (coolDown - dieDot)
			return thinDie, thinDie, 0.9 * (1 - u)
		}
		return 0, 0, 0
	}
	switch {
	case k < warmDot:
		return math.Max(thinDie, k/warmDot), thinWarm, 1.2
	case k < warmUp:
		u := (k - warmDot) / (warmUp - warmDot)
		return 1, math.Max(thinWarm, 1-(1-u)*(1-u)*(1-u)), 1.2 * (1 - u)
	}
	return 1, 1, 0
}

// dark reports the tube dead after a power off.
func (p power) dark() bool { return p.dying && p.since >= coolDown }

// PowerOn switches the set on: the picture grows from a dot over half a
// second. The Power effect shows it.
func (tv *TV) PowerOn() {
	tv.power = power{}
	tv.stale = true
}

// PowerOff switches the set off: the picture folds to a line and a dot and
// fades; Dark reports it gone. Calling it again while dying changes nothing.
func (tv *TV) PowerOff() {
	if !tv.power.dying {
		tv.power = power{dying: true}
		tv.stale = true
	}
}

// Dark reports the tube gone dark after PowerOff — the moment a game that
// lets its picture fold away can quit.
func (tv *TV) Dark() bool { return tv.power.dark() }
