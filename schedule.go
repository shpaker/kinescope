package kinescope

import "math/rand/v2"

// Every plays one of its episodes now and then. At rate 1 the waits between
// them are random, from Mean-Spread to Mean+Spread seconds; the schedule's
// rate (Rate(name)) speeds them up or slows them down, and a drive can move
// it like any other param.
type Every struct {
	Mean     float32
	Spread   float32
	Episodes []Episode
}

// schedule is an Every at work in a TV.
type schedule struct {
	every Every
	rate  ParamKey // the key of its rate
	wait  float64  // seconds at rate 1 until the next episode
	done  float64  // seconds at rate 1 since the last one
}

// draw picks the wait until the next episode.
func (s *schedule) draw(random *rand.Rand) {
	spread := float64(s.every.Spread)
	s.wait = float64(s.every.Mean) + spread*(2*random.Float64()-1)
	s.done = 0
}

// advance moves the schedule on by dt seconds at the given rate and reports
// whether an episode is due. A held schedule keeps a due episode waiting.
func (s *schedule) advance(dt float64, rate float32, held bool) bool {
	s.done = min(s.done+dt*float64(rate), s.wait)
	return !held && s.done >= s.wait
}
