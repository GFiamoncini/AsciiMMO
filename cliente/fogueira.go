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

func init() {
	iconArt["sunrise"] = [12]string{
		"............",
		"............",
		"..a..a..a...",
		"...a.a.a....",
		"....oooo....",
		"..oo3444oo..",
		".o33444444o.",
		"aaaaaaaaaaaa",
		"............",
		".2222222222.",
		"...222222...",
		"............",
	}
}

// waitOpts são as esperas da fogueira: ícone, cor, dica e o que mandar.
var waitOpts = []struct {
	key  string
	c    col
	hint string
	tod  float64
}{
	{"sunrise", col{255, 180, 120}, "Esperar o amanhecer", 0.27},
	{"sun", col{255, 220, 90}, "Esperar o meio-dia", 0.5},
	{"sunrise", col{230, 110, 80}, "Esperar o anoitecer", 0.76},
	{"moon", col{170, 190, 255}, "Esperar a meia-noite", 0.0},
	{"rain", col{150, 180, 230}, "Esperar o tempo virar", -1},
}

func (g *Game) sendWait(i int) {
	if waitOpts[i].tod < 0 {
		g.send(proto.In{T: "waitwx"})
	} else {
		g.send(proto.In{T: "wait", I: i})
	}
	g.sfx("bonfire", 0.5)
}

func (g *Game) drawBonfirePanel(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	if !g.nearBonfire(me) {
		g.openPanel = ""
		return
	}
	w, h := 460.0, 250.0
	x, y := math.Max(8, (cw-w)/2), chh-h-130
	g.panel(dst, x, y, w, h, rgbaHex(0xe08040))
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, rgbaHex(0xffa050), 14, true, alignCenter)
	}
	name := "Fogueira"
	for _, b := range g.info.Bonfires {
		if math.Hypot(float64(b.X-me.X), float64(b.Y-me.Y)) <= 12 {
			name = b.Name
		}
	}
	g.txt(dst, "-=< "+name+" >=-", x+w/2, y+10, rgbaHex(0xffc080), 15, true, alignCenter)
	// Ações grandes.
	acts := []struct {
		key, label, hint string
		fn               func()
	}{
		{"E", "Descansar aqui", "cura, grava e você volta aqui ao entrar", func() { g.send(proto.In{T: "act"}); g.sfx("bonfire", 0.6) }},
		{"V", "Viagem rápida", "ir a outra fogueira ou placa já descoberta", func() { g.openPanel = "travel" }},
		{"", "Descansar e sair do jogo", "grava aqui e fecha", func() {
			g.send(proto.In{T: "act"})
			g.send(proto.In{T: "save"})
			g.quitAt = g.now() + 700
		}},
	}
	for i, a := range acts {
		ay := y + 38 + float64(i)*38
		hov := g.hover(x+16, ay, w-32, 32)
		g.rect(dst, x+16, ay, w-32, 32, color.RGBA{18, 12, 8, 255})
		g.stroke(dst, x+16, ay, w-32, 32, pickRGBA(hov, rgbaHex(0xffa050), rgbaHex(0x5a3a20)), 1)
		if a.key != "" {
			g.keyTile(dst, a.key, x+26, ay+7)
		}
		g.txt(dst, a.label, x+60, ay+4, pickC(hov, cGold, cText), 13, true, alignLeft)
		g.txt(dst, a.hint, x+60, ay+19, cDim, 10, false, alignLeft)
		fn := a.fn
		g.uiRects = append(g.uiRects, uiRect{x: x + 16, y: ay, w: w - 32, h: 32, fn: fn})
	}
	// Esperar: ícones com dica.
	wy := y + 38 + 3*38 + 8
	g.txt(dst, "Esperar (1–5) — vale para todos:", x+16, wy, cDim, 11, false, alignLeft)
	const isz = 40.0
	ix := x + 16
	tod := g.cur.Tod
	for i, o := range waitOpts {
		bx := ix + float64(i)*(isz+10)
		hov := g.hover(bx, wy+16, isz, isz)
		now := o.tod >= 0 && math.Abs(math.Mod(tod-o.tod+1.5, 1)-0.5) < 0.06 // já é essa hora
		g.rect(dst, bx, wy+16, isz, isz, pickRGBA(hov, color.RGBA{50, 34, 18, 255}, color.RGBA{18, 12, 8, 255}))
		g.stroke(dst, bx, wy+16, isz, isz, pickRGBA(hov || now, rgbaHex(0xffa050), rgbaHex(0x5a3a20)), 1)
		g.drawTinted(dst, o.key, o.c, bx+4, wy+20, isz-8, false)
		g.txt(dst, itoaC(i+1), bx+isz-6, wy+isz+3, cDimmer, 9, false, alignCenter)
		ii := i
		g.uiRects = append(g.uiRects, uiRect{x: bx, y: wy + 16, w: isz, h: isz, fn: func() { g.sendWait(ii) }})
		if hov {
			hint := o.hint
			if now {
				hint += " (já é agora)"
			}
			g.tooltipAt(dst, hint, bx+isz/2, wy+isz+22)
		}
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

// updateBonfireKeys: 1 a 5 escolhem com o painel aberto.
func (g *Game) updateBonfireKeys() bool {
	if g.openPanel != "fogo" {
		return false
	}
	for i := range waitOpts {
		if justKey(ebiten.KeyDigit1 + ebiten.Key(i)) {
			g.sendWait(i)
			return true
		}
	}
	if justKey(ebiten.KeyV) {
		g.openPanel = "travel"
		return true
	}
	if justKey(ebiten.KeyE) { // E descansa de novo (cura e grava aqui)
		g.send(proto.In{T: "act"})
		g.sfx("bonfire", 0.6)
		return true
	}
	if justKey(ebiten.KeyEscape) {
		g.openPanel = ""
		return true
	}
	return false
}
