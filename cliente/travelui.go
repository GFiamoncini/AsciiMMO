package main

import (
	"fmt"
	"image/color"
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Viagem rápida (tecla V): perto de uma fogueira ou placa já descoberta,
// abre o mapa com os pontos descobertos; clicar num deles viaja.

func (g *Game) found(i int) bool { return i < 64 && g.cur.Found&(1<<i) != 0 }

// nearTravel devolve o ponto de viagem descoberto onde o jogador está, ou -1.
func (g *Game) nearTravel(me *proto.Player) int {
	for i, t := range g.info.Travel {
		if g.found(i) && math.Hypot(float64(t.X-me.X), float64(t.Y-me.Y)) <= 24 {
			return i
		}
	}
	return -1
}

func (g *Game) drawTravel(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	here := g.nearTravel(me)
	src := zoneRect{"", 0, 0, g.m.w, g.info.OverH}
	listW := 360.0
	k := math.Min((cw-rightW-listW-60)/float64(src.w), (chh-120)/float64(src.h))
	mw, mh := float64(src.w)*k, float64(src.h)*k
	w := mw + listW + 30
	x0, y0 := math.Max(8, (cw-rightW-w)/2), math.Max(8, (chh-mh-50)/2)
	g.panel(dst, x0, y0, w, mh+50, cGold)
	title := "Viagem rápida — vá até uma fogueira ou placa já descoberta para viajar (V fecha)"
	if here >= 0 {
		title = "Viagem rápida — clique num destino (V fecha) · você está em: " + g.info.Travel[here].Name
	}
	g.txt(dst, title, x0+12, y0+10, cGold, 13, true, alignLeft)
	mx, my := x0+10, y0+36
	g.drawMap(dst, me, mx, my, mw, mh, src, false)
	lx, ly := mx+mw+12, my
	for i, t := range g.info.Travel {
		if !g.found(i) {
			continue
		}
		px, py := mx+float64(t.X)/float64(src.w)*mw, my+float64(t.Y)/float64(src.h)*mh
		key, c := "ember", col{255, 160, 60}
		if t.Kind == 1 {
			key, c = "map", col{210, 180, 120}
		}
		if t.Y < g.info.OverH { // as do subsolo ficam só na lista
			g.drawTinted(dst, key, c, px-8, py-8, 16, false)
		}
		hov := g.hover(lx, ly, listW, 20) || g.hover(px-8, py-8, 16, 16)
		tc := color.Color(cText)
		switch {
		case i == here:
			tc = cDim
		case hov:
			tc = cGold
			g.rect(dst, lx-4, ly-1, listW, 19, color.RGBA{60, 46, 20, 200})
		}
		g.drawTinted(dst, key, c, lx, ly, 16, i == here)
		g.txt(dst, g.fit(t.Name, 12, listW-30), lx+22, ly+1, tc, 12, false, alignLeft)
		if i != here && here >= 0 {
			act := &proto.In{T: "travel", I: i}
			g.uiRects = append(g.uiRects, uiRect{x: lx - 4, y: ly - 1, w: listW, h: 19, act: act, fn: func() { g.send(*act); g.openPanel = "" }})
			g.uiRects = append(g.uiRects, uiRect{x: px - 8, y: py - 8, w: 16, h: 16, fn: func() { g.send(*act); g.openPanel = "" }})
		}
		ly += 21
	}
	if ly == my {
		g.txt(dst, "Nenhum outro ponto descoberto ainda.", lx, ly, cDim, 12, false, alignLeft)
	}
	n := 0
	for i := range g.info.Travel {
		if g.found(i) {
			n++
		}
	}
	g.txt(dst, fmt.Sprintf("%d de %d pontos descobertos", n, len(g.info.Travel)), lx, my+mh-14, cDimmer, 12, false, alignLeft)
	g.uiRects = append(g.uiRects, uiRect{x: x0, y: y0, w: w, h: mh + 50, fn: func() {}})
}
