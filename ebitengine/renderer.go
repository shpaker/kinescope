// Package ebitengine draws kinescope TVs with Ebitengine: each TV's effects
// are composed into one picture shader, with a prepass or two at the
// frame's size for the effects that need whole frames (glow, afterglow).
package ebitengine

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/shpaker/kinescope"
)

// grainCycle is the period Frame wraps around at: the grain's two cycles
// (97 and 89 frames) both fit it, so the wrap goes unseen.
const grainCycle = 97 * 89

// Renderer draws a TV. It keeps the shaders it has composed, by set of
// effects, and the afterglow's history between frames: draw each TV with a
// renderer of its own. A Renderer is not safe for concurrent use.
type Renderer struct {
	// Shaders
	blur      *ebiten.Shader
	afterglow *ebiten.Shader
	pictures  map[string]*ebiten.Shader // composed, by the effects' names
	picture   *ebiten.Shader            // the current TV's

	// The TV drawn last and what the renderer knows of it
	tv           *kinescope.TV
	revision     int
	epoch        int
	hasGlow      bool
	hasAfterglow bool

	// Buffers at the frame's size; recreated when it changes
	glowAcross *ebiten.Image
	glow       *ebiten.Image
	history    *ebiten.Image
	current    *ebiten.Image
	// historyStale means history holds nothing worth fading in
	historyStale bool

	// Uniforms, reused between frames: each value is a one-element slice
	// the renderer writes into, so a frame allocates nothing
	params     []param
	uniforms   map[string]any
	time       []float32
	frame      []float32
	scale      []float32
	glowPasses [2]map[string]any
	glowRadius []float32
	glowStart  []float32
	decay      map[string]any
	decayValue []float32

	// Draw options, reused between frames
	pictureOp   ebiten.DrawRectShaderOptions
	afterglowOp ebiten.DrawRectShaderOptions
	acrossOp    ebiten.DrawRectShaderOptions
	downOp      ebiten.DrawRectShaderOptions
}

// param is a param of the TV and its uniform's value.
type param struct {
	key   kinescope.ParamKey
	value []float32
}

// NewRenderer compiles the prepass shaders; the picture shaders are
// composed when a TV is first drawn.
func NewRenderer() (*Renderer, error) {
	blur, err := compile("shaders/glow_blur.kage")
	if err != nil {
		return nil, err
	}
	afterglow, err := compile("shaders/afterglow.kage")
	if err != nil {
		return nil, err
	}
	r := &Renderer{
		blur:       blur,
		afterglow:  afterglow,
		pictures:   make(map[string]*ebiten.Shader),
		time:       []float32{0},
		frame:      []float32{0},
		scale:      []float32{0},
		glowRadius: []float32{0},
		glowStart:  []float32{0},
		decayValue: []float32{0},
	}
	r.glowPasses = [2]map[string]any{
		{"Direction": []float32{1, 0}, "Threshold": r.glowStart, "GlowRadius": r.glowRadius},
		{"Direction": []float32{0, 1}, "Threshold": []float32{0}, "GlowRadius": r.glowRadius},
	}
	r.decay = map[string]any{"AfterglowDecay": r.decayValue}
	return r, nil
}

func compile(path string) (*ebiten.Shader, error) {
	source, err := shaders.ReadFile(path)
	if err != nil {
		return nil, err
	}
	shader, err := ebiten.NewShader(source)
	if err != nil {
		return nil, fmt.Errorf("kinescope: compiling %s: %w", path, err)
	}
	return shader, nil
}

// Prepare composes and compiles the tv's shader now, so a set of effects
// that cannot be drawn fails at start rather than on the first frame. Draw
// prepares by itself when it needs to.
func (r *Renderer) Prepare(tv *kinescope.TV) error {
	return r.prepare(tv)
}

// Draw draws frame through the tv onto dst, scaled by a whole scale and
// placed at x, y. It fails only if the TV's set of effects cannot be
// composed into a shader.
func (r *Renderer) Draw(
	dst ebiten.FinalScreen,
	frame *ebiten.Image,
	tv *kinescope.TV,
	x int,
	y int,
	scale int,
) error {
	if tv != r.tv || tv.Revision() != r.revision {
		if err := r.prepare(tv); err != nil {
			return err
		}
	}
	if tv.Epoch() != r.epoch {
		r.epoch = tv.Epoch()
		r.historyStale = true
	}

	for _, p := range r.params {
		p.value[0] = tv.Value(p.key)
	}
	r.time[0] = float32(tv.Time())
	r.frame[0] = float32(tv.Frame() % grainCycle)
	r.scale[0] = float32(scale)

	size := frame.Bounds().Size()
	source := r.drawAfterglow(frame, size, tv)

	op := &r.pictureOp
	*op = ebiten.DrawRectShaderOptions{}
	op.Images[0] = source
	if r.hasGlow {
		r.drawGlow(source, size, tv)
		op.Images[1] = r.glow
	}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(x), float64(y))
	op.Uniforms = r.uniforms
	dst.DrawRectShader(size.X, size.Y, r.picture, op)
	return nil
}

// prepare composes the tv's picture shader, or takes it from the ones
// composed before, and lays out its uniforms.
func (r *Renderer) prepare(tv *kinescope.TV) error {
	effects := tv.Effects()
	key := ""
	r.hasGlow, r.hasAfterglow = false, false
	for _, effect := range effects {
		key += effect.Name() + " "
		switch effect.(type) {
		case *kinescope.Glow:
			r.hasGlow = true
		case *kinescope.Afterglow:
			r.hasAfterglow = true
		}
	}
	picture, ok := r.pictures[key]
	if !ok {
		source, err := compose(effects)
		if err != nil {
			return err
		}
		picture, err = ebiten.NewShader(source)
		if err != nil {
			return fmt.Errorf("kinescope: compiling the picture of %q: %w", key, err)
		}
		r.pictures[key] = picture
	}
	r.picture = picture

	if tv != r.tv {
		r.historyStale = true
	}
	r.tv, r.revision, r.epoch = tv, tv.Revision(), tv.Epoch()

	r.params = r.params[:0]
	r.uniforms = map[string]any{"Time": r.time, "Frame": r.frame, "Scale": r.scale}
	for _, p := range tv.Params() {
		value := []float32{0}
		r.params = append(r.params, param{key: p.Key, value: value})
		r.uniforms[uniformName(p.Key)] = value
	}
	return nil
}

// drawAfterglow lays the frame over the fading history of the frames before
// and returns the result; without an afterglow it returns the frame.
func (r *Renderer) drawAfterglow(
	frame *ebiten.Image,
	size image.Point,
	tv *kinescope.TV,
) *ebiten.Image {
	decay := tv.Value(kinescope.AfterglowDecay)
	if !r.hasAfterglow || decay <= 0 {
		r.historyStale = true
		return frame
	}
	r.history = ensureImage(r.history, size)
	r.current = ensureImage(r.current, size)
	r.history, r.current = r.current, r.history
	if r.historyStale {
		r.history.Clear()
		r.historyStale = false
	}

	r.decayValue[0] = decay
	op := &r.afterglowOp
	*op = ebiten.DrawRectShaderOptions{Blend: ebiten.BlendCopy}
	op.Images[0] = frame
	op.Images[1] = r.history
	op.Uniforms = r.decay
	r.current.DrawRectShader(size.X, size.Y, r.afterglow, op)
	return r.current
}

// drawGlow picks the bright areas of source and blurs them into r.glow,
// across and then down. With no strength it leaves the glow as it is: the
// picture adds none of it.
func (r *Renderer) drawGlow(
	source *ebiten.Image,
	size image.Point,
	tv *kinescope.TV,
) {
	r.glowAcross = ensureImage(r.glowAcross, size)
	r.glow = ensureImage(r.glow, size)
	if tv.Value(kinescope.GlowStrength) <= 0 {
		return
	}
	r.glowStart[0] = tv.Value(kinescope.GlowThreshold)
	r.glowRadius[0] = tv.Value(kinescope.GlowRadius)

	across := &r.acrossOp
	*across = ebiten.DrawRectShaderOptions{Blend: ebiten.BlendCopy}
	across.Images[0] = source
	across.Uniforms = r.glowPasses[0]
	r.glowAcross.DrawRectShader(size.X, size.Y, r.blur, across)

	down := &r.downOp
	*down = ebiten.DrawRectShaderOptions{Blend: ebiten.BlendCopy}
	down.Images[0] = r.glowAcross
	down.Uniforms = r.glowPasses[1]
	r.glow.DrawRectShader(size.X, size.Y, r.blur, down)
}

// ensureImage returns img if it is size, or a new image of size in its
// place.
func ensureImage(img *ebiten.Image, size image.Point) *ebiten.Image {
	if img != nil && img.Bounds().Size() == size {
		return img
	}
	if img != nil {
		img.Deallocate()
	}
	return ebiten.NewImage(size.X, size.Y)
}
