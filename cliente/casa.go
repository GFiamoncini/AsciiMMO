package main

import (
	"fmt"
	"image/color"
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Casa (servidor/casa.go): E na porta de uma casa à venda abre a compra; na
// sua casa, abre o baú. Clique num item da mochila guarda 1 (botão direito
// guarda todos); clique no baú tira 1 (direito tira todos). O dinheiro tem
// botões de 100 e de tudo.

func (g *Game) nearLot(me *proto.Player) int {
	for i, h := range g.info.Houses {
		if math.Hypot(float64(h.X-me.X), float64(h.Y-me.Y)) <= 10 {
			return i
		}
	}
	return -1
}

func (g *Game) drawHouse(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	i := g.nearLot(me)
	if i < 0 {
		g.openPanel = ""
		return
	}
	h := g.info.Houses[i]
	owned := g.cur.House == i+1
	w, ph := math.Min(820, cw-rightW-40), 440.0
	if !owned {
		ph = 200
	}
	x, y := math.Max(8, (cw-rightW-w)/2), math.Max(8, (chh-ph)/2)
	g.panel(dst, x, y, w, ph, rgbaHex(0xc08040))
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + ph}, {x + w, y + ph}} {
		g.txt(dst, "+", p[0], p[1]-9, rgbaHex(0xe0a050), 14, true, alignCenter)
	}
	g.txt(dst, h.Name, x+16, y+12, rgbaHex(0xffd090), 16, true, alignLeft)
	g.txt(dst, fmt.Sprintf("$ %d %s", me.Gold, g.info.Currency), x+w-16, y+14, rgbaHex(0xc8a860), 13, false, alignRight)
	if !owned {
		g.txt(dst, fmt.Sprintf("À venda por %d %s. Vem com um baú para guardar itens e dinheiro,", h.Price, g.info.Currency), x+16, y+50, cText, 13, false, alignLeft)
		g.txt(dst, "que ficam a salvo quando você morre. A casa é sua: outro jogador tem a dele.", x+16, y+70, cText, 13, false, alignLeft)
		if g.cur.House > 0 && g.cur.House-1 != i {
			g.txt(dst, "Você já tem a "+g.info.Houses[g.cur.House-1].Name+"; comprar esta troca de casa (o baú vem junto).", x+16, y+94, cDim, 12, false, alignLeft)
		}
		ok := me.Gold >= h.Price
		bw := 180.0
		bx, by := x+16, y+ph-56
		hov := ok && g.hover(bx, by, bw, 34)
		g.rect(dst, bx, by, bw, 34, pickRGBA(ok, color.RGBA{50, 34, 18, 255}, color.RGBA{24, 20, 20, 255}))
		g.stroke(dst, bx, by, bw, 34, pickRGBA(hov, rgbaHex(0xffd34a), rgbaHex(0x6a4a2a)), 1)
		g.txt(dst, fmt.Sprintf("Comprar (%d)", h.Price), bx+bw/2, by+9, pickC(ok, cText, cDimmer), 14, true, alignCenter)
		if ok {
			g.uiRects = append(g.uiRects, uiRect{x: bx, y: by, w: bw, h: 34, fn: func() { g.send(proto.In{T: "house", I: i}); g.sfx("coin", 0.7) }})
		} else {
			g.txt(dst, fmt.Sprintf("faltam %d", h.Price-me.Gold), bx+bw+12, by+10, rgbaHex(0xd06060), 12, false, alignLeft)
		}
		g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: ph, fn: func() {}})
		return
	}
	g.txt(dst, "Baú da casa — clique guarda/tira 1 · botão direito: todos · Esc ou E fecha", x+16, y+36, cDim, 11, false, alignLeft)
	colW := (w - 48) / 2
	list := func(lx float64, title string, rows [][2]int, op int) {
		g.txt(dst, title, lx, y+58, cGold, 13, true, alignLeft)
		const rowH = 26.0
		for k, r := range rows {
			if k >= 11 {
				g.txt(dst, fmt.Sprintf("… e mais %d", len(rows)-k), lx, y+80+float64(k)*rowH, cDimmer, 11, false, alignLeft)
				break
			}
			if r[0] < 0 || r[0] >= len(g.info.Items) {
				continue
			}
			it := g.info.Items[r[0]]
			ry := y + 80 + float64(k)*rowH
			hov := g.hover(lx, ry, colW, rowH-2)
			g.rect(dst, lx, ry, colW, rowH-2, pickRGBA(hov, color.RGBA{48, 34, 22, 255}, color.RGBA{14, 12, 18, 255}))
			g.drawItemIcon(dst, it, lx+2, ry+1, rowH-4)
			g.txt(dst, g.fit(it.Name, 12, colW-80), lx+30, ry+5, rarCols[clampi(it.Rarity, 0, 3)], 12, false, alignLeft)
			g.txt(dst, fmt.Sprintf("x%d", r[1]), lx+colW-8, ry+5, cDim, 12, false, alignRight)
			one := &proto.In{T: "stash", I: op, Item: r[0], Slot: 1}
			all := &proto.In{T: "stash", I: op, Item: r[0], Slot: r[1]}
			g.uiRects = append(g.uiRects, uiRect{x: lx, y: ry, w: colW, h: rowH - 2, act: one, alt: all})
		}
		if len(rows) == 0 {
			g.txt(dst, "vazio", lx, y+82, cDimmer, 12, false, alignLeft)
		}
	}
	list(x+16, "Mochila", g.cur.Inv, 0)
	list(x+32+colW, fmt.Sprintf("Baú (%d tipos)", len(g.cur.Stash)), g.cur.Stash, 1)
	// Dinheiro.
	gy := y + ph - 48
	g.txt(dst, fmt.Sprintf("No baú: %d %s", g.cur.StashGold, g.info.Currency), x+16, gy-20, rgbaHex(0xc8a860), 13, false, alignLeft)
	bx := x + 16
	for _, b := range []struct {
		label string
		op, n int
	}{{"Guardar 100", 2, 100}, {"Guardar tudo", 2, me.Gold}, {"Tirar 100", 3, 100}, {"Tirar tudo", 3, g.cur.StashGold}} {
		bw := g.textW(b.label, 12) + 20
		hov := g.hover(bx, gy, bw, 28)
		g.rect(dst, bx, gy, bw, 28, color.RGBA{30, 24, 14, 255})
		g.stroke(dst, bx, gy, bw, 28, pickRGBA(hov, rgbaHex(0xffd34a), rgbaHex(0x6a4a2a)), 1)
		g.txt(dst, b.label, bx+bw/2, gy+7, pickC(hov, cGold, cText), 12, false, alignCenter)
		if b.n > 0 {
			g.uiRects = append(g.uiRects, uiRect{x: bx, y: gy, w: bw, h: 28, act: &proto.In{T: "stash", I: b.op, Slot: b.n}})
		}
		bx += bw + 8
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: ph, fn: func() {}})
}
