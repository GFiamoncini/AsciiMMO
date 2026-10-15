package main

import (
	"image/color"
	"math"
	"math/rand"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Portas seladas: faíscas nas runas da porta e nas pedras rúnicas (a pista
// fica visível no mapa), a inscrição quando se chega perto e, nas de setas,
// o modo de tocar as runas: E abre, as setas entram, Esc desiste.

func (g *Game) nearSeal(me *proto.Player) int {
	for i, s := range g.info.Seals {
		if math.Hypot(float64(s.X-me.X), float64(s.Y-me.Y)) <= 18 {
			return i
		}
	}
	return -1
}

// sealOpen diz se a porta do selo está aberta agora (a célula do meio livre).
func (g *Game) sealOpen(s proto.Seal) bool {
	m := g.m
	return s.Y >= 0 && s.Y < m.h && m.flags[s.Y*m.w+s.X]&1 == 0
}

// sealSparks solta faíscas frias nas runas e nas pedras que estão na tela.
func (g *Game) sealSparks() {
	in := func(x, y int) bool {
		return x >= g.x0-4 && x < g.x0+g.cols+4 && y >= g.y0-4 && y < g.y0+g.rows+4
	}
	for _, s := range g.info.Seals {
		if in(s.X, s.Y) && !g.sealOpen(s) && rand.Float64() < 0.5 {
			g.spawn("arcane", float64(s.X)+(rand.Float64()-0.5)*10, float64(s.Y)-7, (rand.Float64()-0.5)*6, -10, 0.8)
			g.light(float64(s.X), float64(s.Y)-4, 22, 0.6+0.2*math.Sin(g.now()/300), cool, coolAdd)
		}
		for _, c := range s.Clues {
			if in(c[0], c[1]) && rand.Float64() < 0.15 {
				g.spawn("arcane", float64(c[0])+(rand.Float64()-0.5)*4, float64(c[1])-4, 0, -8, 0.7)
				g.light(float64(c[0]), float64(c[1]), 14, 0.5, cool, coolAdd)
			}
		}
	}
}

// updateSeal trata as teclas do modo de tocar as runas. true se está nele.
func (g *Game) updateSeal() bool {
	if g.sealMode < 0 {
		return false
	}
	just := inpututil.IsKeyJustPressed
	switch {
	case just(ebiten.KeyEscape):
		g.sealMode = -1
		return true
	case just(ebiten.KeyArrowUp), just(ebiten.KeyW):
		g.sealBuf += "U"
	case just(ebiten.KeyArrowDown), just(ebiten.KeyS):
		g.sealBuf += "D"
	case just(ebiten.KeyArrowLeft), just(ebiten.KeyA):
		g.sealBuf += "L"
	case just(ebiten.KeyArrowRight), just(ebiten.KeyD):
		g.sealBuf += "R"
	}
	if s := g.info.Seals[g.sealMode]; len(g.sealBuf) >= s.Len {
		g.send(proto.In{T: "seal", Text: g.sealBuf})
		g.sealMode, g.sealBuf = -1, ""
	}
	return true
}

func (g *Game) drawSealInput(dst *ebiten.Image, cw, chh float64) {
	if g.sealMode < 0 || g.sealMode >= len(g.info.Seals) {
		return
	}
	s := g.info.Seals[g.sealMode]
	w, h := 420.0, 110.0
	x, y := (cw-rightW-w)/2, chh*0.62
	g.panel(dst, x, y, w, h, rgbaHex(0x9ad8ff))
	g.txt(dst, s.Title+" — toque as runas na ordem (setas ou WASD · Esc sai)", x+12, y+10, rgbaHex(0x9ad8ff), 12, true, alignLeft)
	arrows := map[byte]string{'U': "^", 'D': "v", 'L': "<", 'R': ">"}
	for k := 0; k < s.Len; k++ {
		bx := x + w/2 - float64(s.Len)*22 + float64(k)*44
		g.rect(dst, bx, y+40, 36, 40, color.RGBA{10, 14, 24, 255})
		g.stroke(dst, bx, y+40, 36, 40, rgbaHex(0x4a6a90), 1)
		if k < len(g.sealBuf) {
			g.txt(dst, arrows[g.sealBuf[k]], bx+18, y+48, rgbaHex(0xc8f0ff), 20, true, alignCenter)
		}
	}
}

// sealHere é o selo de setas, ainda fechado, onde o jogador está; ou -1.
func (g *Game) sealHere() int {
	me := g.cur.pmap[g.info.ID]
	if me == nil {
		return -1
	}
	if i := g.nearSeal(me); i >= 0 && g.info.Seals[i].Kind == 1 && !g.sealOpen(g.info.Seals[i]) {
		return i
	}
	return -1
}
