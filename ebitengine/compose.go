package ebitengine

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/shpaker/kinescope"
)

// shaders holds the prepass shaders, the picture template and a fragment
// per picture effect. A fragment is one Kage function named after its
// effect (aperture_mask → apertureMask), with the signature of its stage:
//
//	geometry:  func name(pos vec2, size vec2) vec2
//	sample:    func name(pos vec2, size vec2) vec3
//	the rest:  func name(c vec3, pos vec2, size vec2, screen vec2) vec3
//
// pos is in frame pixels, screen in screen pixels. A fragment reads its
// params as uniforms named after their keys (tear.strength → TearStrength)
// and may use the helpers of picture.kage.tmpl.
//
//go:embed shaders
var shaders embed.FS

var pictureTemplate = template.Must(
	template.ParseFS(shaders, "shaders/picture.kage.tmpl"),
)

// picture is what the picture template is filled with.
type picture struct {
	Uniforms  []string
	Fragments []string
	Geometry  []string
	Sample    string
	Color     []string
}

// compose builds the picture shader's source for effects in stage order.
// Prepass effects have no fragment: the renderer runs them before.
func compose(effects []kinescope.Effect) ([]byte, error) {
	p := picture{Sample: "sample"}
	for _, effect := range effects {
		for _, param := range effect.Params() {
			p.Uniforms = append(p.Uniforms, uniformName(param.Key))
		}
		if effect.Stage() == kinescope.StagePrepass {
			continue
		}
		fragment, err := fragment(effect)
		if err != nil {
			return nil, err
		}
		p.Fragments = append(p.Fragments, fragment)

		name := functionName(effect.Name())
		switch effect.Stage() {
		case kinescope.StageGeometry:
			p.Geometry = append(p.Geometry, name)
		case kinescope.StageSample:
			if p.Sample != "sample" {
				return nil, errors.New("kinescope: more than one effect reads the frame")
			}
			p.Sample = name
		default:
			p.Color = append(p.Color, name)
		}
	}

	var source bytes.Buffer
	if err := pictureTemplate.Execute(&source, p); err != nil {
		return nil, err
	}
	return source.Bytes(), nil
}

// fragment is the Kage source of an effect's fragment.
func fragment(effect kinescope.Effect) (string, error) {
	source, err := shaders.ReadFile("shaders/fragments/" + effect.Name() + ".kage")
	if err != nil {
		return "", fmt.Errorf("kinescope: no fragment for %q", effect.Name())
	}
	return string(source), nil
}

// uniformName is the uniform of a param: tear.strength → TearStrength.
func uniformName(key kinescope.ParamKey) string {
	return camel(string(key), true)
}

// functionName is the fragment function of an effect: aperture_mask →
// apertureMask.
func functionName(effect string) string {
	return camel(effect, false)
}

// camel joins the words of s, split at dots and underscores, in camel case.
func camel(s string, upper bool) string {
	var b strings.Builder
	for i, word := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == '_'
	}) {
		if i > 0 || upper {
			word = strings.ToUpper(word[:1]) + word[1:]
		}
		b.WriteString(word)
	}
	return b.String()
}

// Source is the Kage source of the picture shader composed for the TV's
// effects, to read when an effect looks wrong.
func Source(tv *kinescope.TV) ([]byte, error) {
	return compose(tv.Effects())
}
