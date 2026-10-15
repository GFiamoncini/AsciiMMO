package main

import (
	"fmt"
	"image/color"
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Terreiro do Mestre Oren: lista das técnicas. Escolhida uma, o mestre mostra
// a sequência seta a seta e a esconde; o jogador repete (setas ou WASD) antes
// do tempo acabar. Errou uma seta, recomeça (Enter). Acertou tudo, o servidor
// confere nível, preço e sequência e ensina.

const (
	dojoList   = iota // escolhendo
	dojoShow          // o mestre mostrando
	dojoTry           // a vez do jogador
	dojoFail          // errou ou estourou o tempo
	dojoSent          // mandou ao servidor
	dojoLocked        // nível baixo ou já sabe: o mestre não mostra
)

const dojoStep = 0.55 // segundos por seta na demonstração

var arrowGlyph = map[byte]string{'U': "^", 'D': "v", 'L': "<", 'R': ">"}

func (g *Game) knowsTech(t int) bool { return g.cur != nil && g.cur.Tech&(1<<t) != 0 }

func (g *Game) openDojo() {
	g.openPanel, g.dojoSel, g.dojoMode = "dojo", 0, dojoList
}

func (g *Game) dojoStart(t int) {
	me := g.cur.pmap[g.info.ID]
	if me == nil || g.knowsTech(t) || me.Level < g.info.Techs[t].Lv {
		g.dojoSel, g.dojoMode, g.dojoAt = t, dojoLocked, g.now()/1000 // o mestre nem mostra
		return
	}
	g.dojoSel, g.dojoMode, g.dojoAt, g.dojoBuf = t, dojoShow, g.now()/1000, ""
}

// updateDojo trata as teclas do terreiro; true enquanto ele está aberto.
func (g *Game) updateDojo() bool {
	if g.openPanel != "dojo" {
		return false
	}
	ts := g.info.Techs
	if len(ts) == 0 {
		g.openPanel = ""
		return false
	}
	t := ts[clampi(g.dojoSel, 0, len(ts)-1)]
	el := g.now()/1000 - g.dojoAt
	if justKey(ebiten.KeyEscape) {
		if g.dojoMode == dojoList {
			g.openPanel = ""
		} else {
			g.dojoMode = dojoList
		}
		return true
	}
	dir := ""
	switch {
	case justKey(ebiten.KeyArrowUp), justKey(ebiten.KeyW):
		dir = "U"
	case justKey(ebiten.KeyArrowDown), justKey(ebiten.KeyS):
		dir = "D"
	case justKey(ebiten.KeyArrowLeft), justKey(ebiten.KeyA):
		dir = "L"
	case justKey(ebiten.KeyArrowRight), justKey(ebiten.KeyD):
		dir = "R"
	}
	enter := justKey(ebiten.KeyEnter) || justKey(ebiten.KeySpace)
	for d := 0; d < g.barN(); d++ { // técnica já aprendida: 1 a 9 põe na barra
		if justKey(ebiten.KeyDigit1+ebiten.Key(d)) && g.knowsTech(g.dojoSel) {
			g.send(proto.In{T: "bar", I: d, Slot: t.Skill})
		}
	}
	switch g.dojoMode {
	case dojoList:
		switch {
		case dir == "U":
			g.dojoSel = (g.dojoSel + len(ts) - 1) % len(ts)
		case dir == "D":
			g.dojoSel = (g.dojoSel + 1) % len(ts)
		case enter:
			g.dojoStart(g.dojoSel)
		}
	case dojoShow:
		if el >= float64(len(t.Seq))*dojoStep+0.5 {
			g.dojoMode, g.dojoAt = dojoTry, g.now()/1000
		}
	case dojoTry:
		if dir != "" {
			g.dojoBuf += dir
			if g.dojoBuf[len(g.dojoBuf)-1] != t.Seq[len(g.dojoBuf)-1] {
				g.dojoMode, g.dojoAt = dojoFail, g.now()/1000
			} else if len(g.dojoBuf) == len(t.Seq) {
				g.send(proto.In{T: "learn", I: g.dojoSel, Text: g.dojoBuf})
				g.dojoMode, g.dojoAt = dojoSent, g.now()/1000
			}
		} else if el > g.dojoLimit(t) {
			g.dojoMode, g.dojoAt = dojoFail, g.now()/1000
		}
	case dojoFail:
		if enter {
			g.dojoStart(g.dojoSel)
		}
	case dojoSent, dojoLocked:
		if el > 1.6 || enter || dir != "" {
			g.dojoMode = dojoList
		}
	}
	return true
}

// dojoLimit é o tempo para repetir: meio segundo por seta e mais um.
func (g *Game) dojoLimit(t proto.Tech) float64 { return 1 + 0.5*float64(len(t.Seq)) }

func (g *Game) drawDojo(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	ts := g.info.Techs
	w, h := math.Min(620, cw-rightW-40), 120+float64(len(ts))*44
	x, y := (cw-rightW-w)/2, math.Max(8, (chh-h)/2-40)
	g.panel(dst, x, y, w, h, cGold)
	g.txt(dst, "Terreiro das Sequências — Mestre Oren", x+12, y+10, cGold, 14, true, alignLeft)
	ly := y + 40
	for i, t := range ts {
		sk := g.info.Book[clampi(t.Skill, 0, len(g.info.Book)-1)]
		sel := i == g.dojoSel
		if sel {
			g.rect(dst, x+6, ly-4, w-12, 40, color.RGBA{60, 46, 20, 200})
		}
		known := g.knowsTech(i)
		g.drawIcon(dst, sk.Icon, sk.Elem, x+14, ly, 32, known)
		nc, st := color.Color(cText), ""
		switch {
		case known:
			nc, st = cDim, "aprendida"
		case me.Level < t.Lv:
			nc, st = cDimmer, fmt.Sprintf("nível %d", t.Lv)
		case me.Gold < t.Price:
			st = fmt.Sprintf("%d bosta (falta %d)", t.Price, t.Price-me.Gold)
		default:
			st = fmt.Sprintf("%d bosta", t.Price)
		}
		if sel && !known {
			nc = cGold
		}
		g.txt(dst, sk.Name, x+56, ly, nc, 13, true, alignLeft)
		g.txt(dst, st, x+w-14, ly, cDim, 12, false, alignRight)
		desc := g.fit(sk.Desc, 11, w-80)
		g.txt(dst, desc, x+56, ly+18, cDimmer, 11, false, alignLeft)
		i := i
		g.uiRects = append(g.uiRects, uiRect{x: x + 6, y: ly - 4, w: w - 12, h: 40, fn: func() {
			if g.dojoMode == dojoList || g.dojoMode == dojoFail {
				g.dojoStart(i)
			}
		}})
		ly += 44
	}
	// A área da sequência.
	t := ts[clampi(g.dojoSel, 0, len(ts)-1)]
	el := g.now()/1000 - g.dojoAt
	by := y + h - 64
	n := len(t.Seq)
	box := func(k int, s string, c color.Color, lit bool) {
		bx := x + w/2 - float64(n)*22 + float64(k)*44
		bg := color.RGBA{14, 12, 10, 255}
		if lit {
			bg = color.RGBA{70, 54, 20, 255}
		}
		g.rect(dst, bx, by, 36, 40, bg)
		g.stroke(dst, bx, by, 36, 40, rgbaHex(0x8a7040), 1)
		g.txt(dst, s, bx+18, by+8, c, 20, true, alignCenter)
	}
	msg, mc := "setas: escolher · Enter: o mestre mostra a sequência uma vez · Esc: sair", color.Color(cDim)
	switch g.dojoMode {
	case dojoList:
		for k := 0; k < n; k++ {
			box(k, "?", cDimmer, false) // quantas setas, sem dizer quais
		}
	case dojoShow:
		cur := int(el / dojoStep)
		for k := 0; k < n; k++ {
			if k == cur && math.Mod(el, dojoStep) < dojoStep*0.8 {
				box(k, arrowGlyph[t.Seq[k]], cGold, true)
			} else {
				box(k, "", cText, false)
			}
		}
		msg, mc = "Olhe bem…", cGold
	case dojoTry:
		for k := 0; k < n; k++ {
			s := ""
			if k < len(g.dojoBuf) {
				s = arrowGlyph[g.dojoBuf[k]]
			}
			box(k, s, rgbaHex(0xc8f0ff), false)
		}
		left := g.dojoLimit(t) - el
		msg, mc = fmt.Sprintf("Sua vez! %.1f s", math.Max(0, left)), cGreen
	case dojoFail:
		for k := 0; k < n; k++ {
			s := ""
			if k < len(g.dojoBuf) {
				s = arrowGlyph[g.dojoBuf[k]]
			}
			box(k, s, rgbaHex(0xff7060), false)
		}
		msg, mc = "Errou. O mestre suspira. Enter tenta de novo.", rgbaHex(0xff7060)
	case dojoSent:
		for k := 0; k < n; k++ {
			box(k, arrowGlyph[t.Seq[k]], cGreen, false)
		}
		msg, mc = "Sequência feita.", cGreen
	case dojoLocked:
		for k := 0; k < n; k++ {
			box(k, "?", cDimmer, false)
		}
		msg, mc = fmt.Sprintf("\"Volte quando tiver nível %d. Antes disso, só se machuca.\"", t.Lv), rgbaHex(0xff9a60)
		if g.knowsTech(g.dojoSel) {
			msg = "Você já sabe esta. 1 a 9 a põe na barra (ou pela teia, K)."
		}
	}
	g.txt(dst, msg, x+w/2, y+h-20, mc, 12, false, alignCenter)
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}
