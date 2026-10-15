package main

import (
	"math"
	"strings"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Leituras: placas, postes de direção, pedras rúnicas e inscrições (o
// servidor manda a lista em info.Reads). No mapa as letras são células
// minúsculas; E perto de uma abre a janela onde se lê de verdade.

const readReach = 10.0

func (g *Game) nearRead(me *proto.Player) int {
	best, bd := -1, readReach
	for i, r := range g.info.Reads {
		if d := math.Hypot(float64(r.X-me.X), float64(r.Y-me.Y)); d <= bd {
			best, bd = i, d
		}
	}
	return best
}

func (g *Game) openRead(i int) {
	g.readSel, g.openPanel = i, "read"
}

// updateRead fecha a leitura com E, Esc, Espaço ou ao se afastar.
func (g *Game) updateRead() bool {
	if g.openPanel != "read" || g.cur == nil || g.info == nil {
		return false
	}
	me := g.cur.pmap[g.info.ID]
	if me != nil && g.readSel == -2 { // -painel read: a mais perto
		g.readSel = g.nearRead(me)
	}
	if me == nil || g.nearRead(me) != g.readSel {
		g.openPanel = ""
		return false
	}
	if justKey(ebiten.KeyE) || justKey(ebiten.KeyEscape) || justKey(ebiten.KeySpace) || justKey(ebiten.KeyEnter) {
		g.openPanel = ""
		return true
	}
	return false // a leitura não prende o andar
}

func (g *Game) drawRead(dst *ebiten.Image, cw, chh float64) {
	if g.readSel < 0 || g.readSel >= len(g.info.Reads) {
		return
	}
	r := g.info.Reads[g.readSel]
	w := math.Min(560, cw-rightW-40)
	var lines []string
	for _, para := range strings.Split(r.Text, "\n") {
		if para == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, g.wrapPx(para, 15, w-32)...)
	}
	h := 64 + float64(len(lines))*22
	x, y := (cw-rightW-w)/2, chh-130-h
	g.panel(dst, x, y, w, h, rgbaHex(0xc8b48c))
	g.txt(dst, r.Title, x+16, y+12, cGold, 14, true, alignLeft)
	for i, l := range lines {
		g.txt(dst, l, x+16, y+38+float64(i)*22, cText, 15, false, alignLeft)
	}
	g.txt(dst, "E / Esc: fechar", x+w-12, y+h-20, cDimmer, 11, false, alignRight)
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}
