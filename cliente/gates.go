package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Portões Rúnicos (o servidor explica em portoes.go): os acesos giram em
// luz violeta; E diante de um apagado abre a runa (setas); diante de um
// aceso, a lista dos outros acesos (atravessar gasta uma Pedra de Eco).

const gateReach = 12.0

func (g *Game) gateLit(i int) bool { return g.cur != nil && i < 32 && g.cur.Gates&(1<<i) != 0 }

func (g *Game) nearGate(me *proto.Player) int {
	for i, gt := range g.info.Gates {
		if math.Hypot(float64(gt.X-me.X), float64(gt.Y-me.Y)) <= gateReach {
			return i
		}
	}
	return -1
}

// gateFx: redemoinho nos acesos que estão na tela.
func (g *Game) gateFx() {
	if g.gateSel == -2 && g.cur != nil && g.cur.pmap[g.info.ID] != nil { // -painel gate
		if i := g.nearGate(g.cur.pmap[g.info.ID]); i >= 0 {
			g.gateSel = i
		}
	}
	for i, gt := range g.info.Gates {
		if gt.X < g.x0-10 || gt.X > g.x0+g.cols+10 || gt.Y < g.y0-10 || gt.Y > g.y0+g.rows+10 {
			continue
		}
		cx, cy := float64(gt.X), float64(gt.Y)-4
		if !g.gateLit(i) {
			g.light(cx, cy-4, 10, 0.3, cool, coolAdd) // a runa apagada ainda brilha um pouco
			continue
		}
		a := g.now()/300 + rand.Float64()*tau
		g.spawn("arcane", cx+math.Cos(a)*4, cy+math.Sin(a)*3, -math.Sin(a)*6, math.Cos(a)*4-2, 0.8)
		g.light(cx, cy, 30, 0.8+0.15*math.Sin(g.now()/200), col{0.9, 0.7, 1.4}, col{60, 30, 110})
	}
}

// openGate decide o que E faz diante do portão i.
func (g *Game) openGate(i int) {
	g.gateSel, g.gateBuf = i, ""
	g.openPanel = "gate"
}

// updateGate: no portão apagado, as setas tocam a runa; Esc sai.
func (g *Game) updateGate() bool {
	if g.openPanel != "gate" || g.cur == nil || g.info == nil {
		return false
	}
	if justKey(ebiten.KeyEscape) || justKey(ebiten.KeyE) {
		g.openPanel = ""
		return true
	}
	if me := g.cur.pmap[g.info.ID]; me != nil && g.gateSel >= 0 && g.nearGate(me) != g.gateSel {
		g.openPanel = "" // saiu de perto do portão
		return false
	}
	i := g.gateSel
	if i < 0 || i >= len(g.info.Gates) || g.gateLit(i) {
		return true
	}
	gt := g.info.Gates[i]
	for _, k := range []struct {
		a, b ebiten.Key
		s    string
	}{{ebiten.KeyArrowUp, ebiten.KeyW, "U"}, {ebiten.KeyArrowDown, ebiten.KeyS, "D"}, {ebiten.KeyArrowLeft, ebiten.KeyA, "L"}, {ebiten.KeyArrowRight, ebiten.KeyD, "R"}} {
		if justKey(k.a) || justKey(k.b) {
			g.gateBuf += k.s
		}
	}
	if gt.Seq != "" && len(g.gateBuf) >= len(gt.Seq) {
		g.send(proto.In{T: "gaterune", I: i, Text: g.gateBuf})
		g.openPanel = ""
	}
	return true
}

func (g *Game) drawGate(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	i := g.gateSel
	if i < 0 || i >= len(g.info.Gates) {
		return
	}
	gt := g.info.Gates[i]
	violet := rgbaHex(0xb8a0ff)
	if !g.gateLit(i) {
		n := max(1, len(gt.Seq))
		w, h := 460.0, 140.0
		x, y := (cw-rightW-w)/2, chh*0.55
		g.panel(dst, x, y, w, h, violet)
		g.txt(dst, gt.Name+" — toque a runa (setas ou WASD · Esc sai)", x+12, y+10, violet, 12, true, alignLeft)
		for k := 0; k < n; k++ {
			bx := x + w/2 - float64(n)*22 + float64(k)*44
			g.rect(dst, bx, y+40, 36, 40, color.RGBA{14, 10, 24, 255})
			g.stroke(dst, bx, y+40, 36, 40, rgbaHex(0x6a5a9a), 1)
			if k < len(g.gateBuf) {
				g.txt(dst, arrowGlyph[g.gateBuf[k]], bx+18, y+48, rgbaHex(0xe0d0ff), 20, true, alignCenter)
			}
		}
		hint := "A runa deste portão está gravada no portão anterior da corrente."
		if i > 0 && g.gateLit(i-1) {
			hint = "Gravada no " + g.info.Gates[i-1].Name + ": " + runeText(gt.Seq)
		}
		g.txt(dst, g.fit(hint, 12, w-24), x+12, y+h-26, cDim, 12, false, alignLeft)
		return
	}
	// Aceso: lista dos outros acesos.
	stones := 0
	for _, s := range g.cur.Inv {
		if s[0] >= 0 && s[0] < len(g.info.Items) && g.info.Items[s[0]].Name == "Pedra de Eco" {
			stones += s[1]
		}
	}
	w := 440.0
	h := 90 + 22*float64(len(g.info.Gates))
	x, y := (cw-rightW-w)/2, math.Max(8, (chh-h)/2-30)
	g.panel(dst, x, y, w, h, violet)
	g.txt(dst, gt.Name+" — atravessar (E ou Esc fecha)", x+12, y+10, violet, 13, true, alignLeft)
	g.txt(dst, fmt.Sprintf("Pedras de Eco: %d (cada travessia gasta uma)", stones), x+12, y+30, pickC(stones > 0, cText, rgbaHex(0xff7060)), 12, false, alignLeft)
	ly := y + 56
	for j, o := range g.info.Gates {
		switch {
		case j == i:
			g.txt(dst, "• "+o.Name+" (você está aqui)", x+16, ly, cDimmer, 12, false, alignLeft)
		case g.gateLit(j):
			hov := g.hover(x+8, ly-2, w-16, 20)
			c := color.Color(cText)
			if hov {
				c = violet
				g.rect(dst, x+8, ly-2, w-16, 20, color.RGBA{50, 36, 80, 200})
			}
			g.txt(dst, "→ "+o.Name, x+16, ly, c, 12, false, alignLeft)
			act := &proto.In{T: "gate", I: j}
			g.uiRects = append(g.uiRects, uiRect{x: x + 8, y: ly - 2, w: w - 16, h: 20, act: act, fn: func() { g.send(*act); g.openPanel = "" }})
		default:
			g.txt(dst, "  "+o.Name+" — apagado", x+16, ly, cDimmer, 12, false, alignLeft)
		}
		ly += 22
	}
	if n := i + 1; n < len(g.info.Gates) && !g.gateLit(n) {
		g.txt(dst, g.fit("Runa do "+g.info.Gates[n].Name+": "+runeText(g.info.Gates[n].Seq), 12, w-24), x+12, y+h-24, violet, 12, false, alignLeft)
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

func runeText(seq string) string {
	r := strings.NewReplacer("U", "↑ ", "D", "↓ ", "L", "← ", "R", "→ ")
	return strings.TrimSpace(r.Replace(seq))
}

// regionName é o nome da região em (x, y), ou "".
func (g *Game) regionName(x, y int) string {
	for _, r := range g.info.Regions {
		if math.Hypot(float64(x-r.X), float64(y-r.Y)) <= float64(r.R) {
			return r.Name
		}
	}
	return ""
}
