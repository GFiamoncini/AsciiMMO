package main

import (
	"image/color"
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Painel da fogueira: E descansa (o servidor cura, grava e acende) e abre
// este painel para deixar o tempo passar até uma hora do dia ou até o clima
// virar (servidor/espera.go). Vale para o mundo todo. Afastar-se fecha.

func (g *Game) nearBonfire(me *proto.Player) bool {
	for _, b := range g.info.Bonfires {
		if math.Hypot(float64(b.X-me.X), float64(b.Y-me.Y)) <= 12 {
			return true
		}
	}
	return false
}

func (g *Game) drawBonfirePanel(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	if !g.nearBonfire(me) {
		g.openPanel = ""
		return
	}
	w, h := 420.0, 250.0
	x, y := (cw-rightW-w)/2+rightW/2-150, chh-h-130
	x = math.Max(8, (cw-w)/2)
	g.panel(dst, x, y, w, h, rgbaHex(0xe08040))
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, rgbaHex(0xffa050), 14, true, alignCenter)
	}
	g.txt(dst, "-=< Fogueira >=-", x+w/2, y+10, rgbaHex(0xffc080), 15, true, alignCenter)
	g.txt(dst, "Você descansou. Deixar o tempo passar? (vale para todos)", x+w/2, y+32, cDim, 11, false, alignCenter)
	opts := []struct {
		key, label string
		fn         func()
	}{
		{"1", "Esperar o amanhecer", func() { g.send(proto.In{T: "wait", I: 0}) }},
		{"2", "Esperar o meio-dia", func() { g.send(proto.In{T: "wait", I: 1}) }},
		{"3", "Esperar o anoitecer", func() { g.send(proto.In{T: "wait", I: 2}) }},
		{"4", "Esperar a meia-noite", func() { g.send(proto.In{T: "wait", I: 3}) }},
		{"5", "Esperar o tempo virar", func() { g.send(proto.In{T: "waitwx"}) }},
	}
	for i, o := range opts {
		oy := y + 54 + float64(i)*36
		hov := g.hover(x+16, oy, w-32, 30)
		g.rect(dst, x+16, oy, w-32, 30, color.RGBA{18, 12, 8, 255})
		g.stroke(dst, x+16, oy, w-32, 30, pickRGBA(hov, rgbaHex(0xffa050), rgbaHex(0x5a3a20)), 1)
		g.keyTile(dst, o.key, x+26, oy+6)
		g.txt(dst, o.label, x+60, oy+7, pickC(hov, cGold, cText), 13, false, alignLeft)
		fn := o.fn
		g.uiRects = append(g.uiRects, uiRect{x: x + 16, y: oy, w: w - 32, h: 30, fn: func() { fn(); g.sfx("bonfire", 0.5) }})
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

// updateBonfireKeys: 1 a 5 escolhem com o painel aberto.
func (g *Game) updateBonfireKeys() bool {
	if g.openPanel != "fogo" {
		return false
	}
	for i := 0; i < 5; i++ {
		if justKey(ebiten.KeyDigit1 + ebiten.Key(i)) {
			if i < 4 {
				g.send(proto.In{T: "wait", I: i})
			} else {
				g.send(proto.In{T: "waitwx"})
			}
			g.sfx("bonfire", 0.5)
			return true
		}
	}
	if justKey(ebiten.KeyEscape) || justKey(ebiten.KeyE) {
		g.openPanel = ""
		return true
	}
	return false
}
