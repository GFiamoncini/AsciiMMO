package main

import (
	"image/color"
	"math"
	"math/rand"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Véu de Cinzas e regiões corrompidas (servidor/veu.go). Enquanto o guardião
// de uma região não cai, chove cinza nela e a borda da tela pega um tom de
// brasa; derrubado, ela fica limpa. O Pico de Tiamath tem um muro de cinza
// tremulando na borda até os cinco caírem.

var veilRegions = []string{"Cemitério", "Selva de Marapá", "Talássia", "Deserto de Ankar", "Picos de Gelo"}

const veilPeak = "Pico de Tiamath"

func init() {
	pkinds["ash"] = &pkind{[]col{{150, 130, 120}, {110, 90, 85}, {70, 55, 50}}, 5, ".,'`"}
}

// corruptedHere diz se o jogador está numa região ainda não purificada.
func (g *Game) corruptedHere(me *proto.Player) bool {
	name := g.placeName(me)
	for i, n := range veilRegions {
		if n == name && g.cur.Pure&(1<<i) == 0 {
			return true
		}
	}
	return false
}

// drawVeil: cinza caindo na região corrompida e o muro do Véu no Pico.
func (g *Game) drawVeil(me *proto.Player, T float64) {
	if me.Y >= g.info.OverH {
		return
	}
	if g.corruptedHere(me) {
		for k := 0; k < 3; k++ {
			x := float64(g.x0) + rand.Float64()*float64(g.cols)
			y := float64(g.y0) + rand.Float64()*float64(g.rows)*0.6
			g.spawn("ash", x, y, -2+rand.Float64()*2, 4+rand.Float64()*4, 2.5)
		}
	}
	if g.cur.Pure == 1<<len(veilRegions)-1 {
		return
	}
	for _, r := range g.info.Regions {
		if r.Name != veilPeak {
			continue
		}
		if math.Hypot(float64(me.X-r.X), float64(me.Y-r.Y)) > float64(r.R)+260 {
			return
		}
		// Muro de cinza na borda: colunas que sobem e tremulam.
		R := float64(r.R)
		for a := 0.0; a < 2*math.Pi; a += 0.6 / R {
			bx, by := float64(r.X)+math.Cos(a)*R, float64(r.Y)+math.Sin(a)*R*0.98
			if bx < float64(g.x0-2) || bx > float64(g.x0+g.cols+2) || by < float64(g.y0-2) || by > float64(g.y0+g.rows+20) {
				continue
			}
			h := 6 + int(4*math.Sin(a*9+T*1.7)+3*math.Sin(a*23-T*2.3))
			for k := 0; k < h; k++ {
				fl := 0.5 + 0.5*math.Sin(T*3+a*40+float64(k))
				c := mix3(col{60, 40, 40}, col{200, 120, 90}, fl*(1-float64(k)/float64(h)))
				g.setC(round(bx), round(by)-k, ":|'.!"[(int(a*97)+k)%5], c)
			}
		}
		g.labels = append(g.labels, label{x: float64(r.X), y: float64(r.Y) + float64(r.R) + 4, text: "Véu de Cinzas — derrote os cinco guardiões", color: col{220, 160, 130}})
	}
}

// veilVignette: a borda da tela pega um tom de brasa na região corrompida.
func (g *Game) veilVignette(dst *ebiten.Image, me *proto.Player) {
	if me.Y < g.info.OverH && g.corruptedHere(me) {
		g.vignette(dst, color.RGBA{120, 50, 30, 45})
	}
}
