package kinescope

import (
	"errors"
	"fmt"
	"strings"
)

// Setup is a whole TV as data: what it looks like and how it misbehaves.
// A preset is a function returning a Setup; NewTV builds a TV from one.
type Setup struct {
	// Seed makes the TV's randomness its own: two TVs with one seed, fed
	// the same Update steps, show the same.
	Seed uint64
	// Effects are the look of the set, at most one of each kind.
	Effects []Effect
	// Sources are named values in time: signals the game sets, drifts and
	// waves.
	Sources map[string]Source
	// Drives connect sources to params.
	Drives []Drive
	// Schedules play episodes now and then, by name; Rate(name) is the key
	// of each one's rate.
	Schedules map[string]Every
}

// Values are the base values of the setup's params: its effects' and its
// schedules' rates. They are what Blend mixes and TV.Apply takes.
func (s Setup) Values() map[ParamKey]float32 {
	values := make(map[ParamKey]float32)
	for _, effect := range s.Effects {
		for _, p := range effect.Params() {
			values[p.Key] = *p.Value
		}
	}
	for name := range s.Schedules {
		values[Rate(name)] = 1
	}
	return values
}

// validate reports what is wrong with the setup, all of it at once.
func (s Setup) validate() error {
	var errs []error
	errs = append(errs, validateEffects(s.Effects)...)
	for name, source := range s.Sources {
		if err := validateSource(source); err != nil {
			errs = append(errs, fmt.Errorf("source %q: %w", name, err))
		}
	}
	for i, d := range s.Drives {
		if _, ok := s.Sources[d.From]; !ok {
			errs = append(errs, fmt.Errorf("drive %d: no source %q", i, d.From))
		}
		if name, ok := strings.CutPrefix(string(d.To), ratePrefix); ok {
			if _, ok := s.Schedules[name]; !ok {
				errs = append(errs, fmt.Errorf("drive %d: no schedule %q", i, name))
			}
		}
	}
	for name, every := range s.Schedules {
		if err := validateEvery(every); err != nil {
			errs = append(errs, fmt.Errorf("schedule %q: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// single are the stages a TV has at most one effect of: each reads the
// frame in its own way.
var single = map[Stage]string{
	StageRead:   "read the frame",
	StageSample: "gather the picture",
}

func validateEffects(effects []Effect) []error {
	var errs []error
	seen := make(map[string]bool)
	stages := make(map[Stage]string)
	for i, effect := range effects {
		if effect == nil {
			errs = append(errs, fmt.Errorf("effect %d is nil", i))
			continue
		}
		if seen[effect.Name()] {
			errs = append(errs, fmt.Errorf("effect %q is listed twice", effect.Name()))
		}
		seen[effect.Name()] = true
		if what, ok := single[effect.Stage()]; ok {
			if other, ok := stages[effect.Stage()]; ok {
				errs = append(errs, fmt.Errorf(
					"effects %q and %q both %s", other, effect.Name(), what))
			}
			stages[effect.Stage()] = effect.Name()
		}
	}
	return errs
}

func validateSource(source Source) error {
	switch s := source.(type) {
	case nil:
		return errors.New("is nil")
	case Drift:
		if s.Period <= 0 {
			return errors.New("drift period must be positive")
		}
	case Wave:
		if s.Period <= 0 {
			return errors.New("wave period must be positive")
		}
	}
	return nil
}

func validateEvery(every Every) error {
	switch {
	case every.Mean <= 0:
		return errors.New("mean must be positive")
	case every.Spread < 0 || every.Spread >= every.Mean:
		return errors.New("spread must be at least 0 and less than mean")
	case len(every.Episodes) == 0:
		return errors.New("no episodes")
	}
	return nil
}
