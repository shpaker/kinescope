package main

import (
	"image"
	"log"
	"strings"

	"github.com/ebitengine/debugui"

	"github.com/shpaker/kinescope"
	"github.com/shpaker/kinescope/ebitengine"
)

var sourceNames = []string{"Test card", "Moving scene", "Dropped picture"}

// panel is the lab's controls.
func (g *game) panel(ctx *debugui.Context) error {
	ctx.SetScale(g.uiScale())
	height := int(g.screenHeight) / g.uiScale()
	ctx.Window("kinescope lab", image.Rect(0, 0, panelWidth, height), func(debugui.ContainerLayout) {
		g.picturePanel(ctx)
		g.effectsPanel(ctx)
		g.moodsPanel(ctx)
		g.sharePanel(ctx)
		g.shaderPanel(ctx)
	})
	return nil
}

func (g *game) picturePanel(ctx *debugui.Context) {
	ctx.Header("Picture", true, func() {
		ctx.SetGridLayout([]int{-1, -2}, nil)
		ctx.Text("Source")
		ctx.Dropdown(&g.sourceIndex, sourceNames)
		ctx.Text("Scale (0 fits)")
		ctx.Slider(&g.scale, 0, 8, 1)
		ctx.SetGridLayout([]int{-1}, nil)
		ctx.Checkbox(&g.bypass, "Bypass the TV")
		if g.sources[2] == nil {
			ctx.Text("Drop a picture onto the window to see it here.")
		}
	})
}

func (g *game) effectsPanel(ctx *debugui.Context) {
	ctx.Header("Effects", true, func() {
		params := g.lab.tv.Params()
		for i := range g.lab.effects {
			t := &g.lab.effects[i]
			ctx.IDScope(t.name, func() {
				ctx.SetGridLayout([]int{-1}, nil)
				ctx.Checkbox(&t.on, t.name).On(func() {
					if err := g.lab.setEffects(); err != nil {
						log.Print(err)
					}
				})
				if !t.on {
					return
				}
				for _, p := range params {
					if strings.HasPrefix(string(p.Key), t.name+".") {
						g.slider(ctx, p)
					}
				}
			})
		}
	})
}

// slider is a param's slider. The param's float32 is mirrored in a float64
// the UI can hold.
func (g *game) slider(ctx *debugui.Context, p kinescope.Param) {
	ctx.IDScope(string(p.Key), func() {
		shadow, ok := g.shadows[string(p.Key)]
		if !ok {
			shadow = new(float64)
			g.shadows[string(p.Key)] = shadow
		}
		*shadow = float64(*p.Value)
		ctx.SetGridLayout([]int{-1, -2}, nil)
		_, field, _ := strings.Cut(string(p.Key), ".")
		ctx.Text("  " + field)
		step := float64(p.Max-p.Min) / 200
		ctx.SliderF(shadow, float64(p.Min), float64(p.Max), step, 3).On(func() {
			*p.Value = float32(*shadow)
		})
	})
}

func (g *game) moodsPanel(ctx *debugui.Context) {
	ctx.Header("Moods", true, func() {
		ctx.SetGridLayout([]int{-1}, nil)
		for i := range g.lab.moods {
			m := &g.lab.moods[i]
			ctx.IDScope(m.name, func() {
				ctx.Checkbox(&m.on, m.name).On(func() {
					g.lab.keep()
					if err := g.lab.rebuild(); err != nil {
						log.Print(err)
					}
					g.lab.shake.Set(float32(g.shake))
				})
			})
		}
		ctx.SetGridLayout([]int{-1, -2}, nil)
		ctx.Text("shake")
		ctx.SliderF(&g.shake, 0, 1, 0.01, 2).On(func() {
			g.lab.shake.Set(float32(g.shake))
		})
		ctx.SetGridLayout([]int{-1, -1}, nil)
		ctx.Button("Play jitter").On(func() { g.lab.tv.Play(kinescope.Jitter()) })
		ctx.Button("Play ripple").On(func() { g.lab.tv.Play(kinescope.Ripple()) })
	})
}

func (g *game) sharePanel(ctx *debugui.Context) {
	ctx.Header("Share", true, func() {
		ctx.SetGridLayout([]int{-1, -1}, nil)
		ctx.Button("Copy link").On(func() {
			g.platform.Export("link", g.platform.Link(g.lab.encode()))
		})
		ctx.Button("Copy as Go").On(func() {
			g.platform.Export("Go", g.lab.goCode())
		})
		ctx.SetGridLayout([]int{-1}, nil)
		ctx.Button("Back to Gorizont").On(func() {
			if err := g.lab.reset(); err != nil {
				log.Print(err)
			}
			g.shake = 0
		})
	})
}

func (g *game) shaderPanel(ctx *debugui.Context) {
	ctx.Header("Shader", false, func() {
		if g.lab.tv.Revision() != g.revision {
			g.revision = g.lab.tv.Revision()
			source, err := ebitengine.Source(g.lab.tv)
			if err != nil {
				g.shader = err.Error()
			} else {
				g.shader = strings.ReplaceAll(string(source), "\t", "  ")
			}
		}
		ctx.SetGridLayout([]int{-1}, nil)
		ctx.Text(g.shader)
	})
}
