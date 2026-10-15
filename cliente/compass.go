package main

import (
	"fmt"
	"image/color"
	"math"
	"sort"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Bússola no alto da tela, como uma faixa: gira com o rumo em que o personagem
// anda (o mapa não gira; quem gira é o rumo) e mostra, na direção de cada um,
// os pontos de viagem já descobertos no mesmo andar e, com a opção da seta
// ligada, a missão. No subsolo a agulha treme; perto do Abismo de Dorrim e do
// Coração de Magma ela enlouquece.

const (
	compassW   = 380.0 // largura da faixa
	compassFOV = 200.0 // graus mostrados de ponta a ponta
)

var compassNames = [8]string{"N", "NE", "L", "SE", "S", "SO", "O", "NO"}

// trackHeading acompanha o rumo pelo andar do personagem (graus, 0 = norte,
// 90 = leste), suavizado para a faixa não pular a cada passo.
func (g *Game) trackHeading(me *proto.Player) {
	if g.headLast == nil {
		g.headLast = &[2]int{me.X, me.Y}
		return
	}
	dx, dy := me.X-g.headLast[0], me.Y-g.headLast[1]
	if dx == 0 && dy == 0 {
		return
	}
	if math.Abs(float64(dx)) > 30 || math.Abs(float64(dy)) > 30 { // viagem, portal: só reposiciona
		g.headLast = &[2]int{me.X, me.Y}
		return
	}
	*g.headLast = [2]int{me.X, me.Y}
	// Média dos últimos passos: andar na diagonal alternando teclas não faz
	// a faixa saltar de um lado para o outro.
	g.hvx, g.hvy = g.hvx*0.8+float64(dx)*0.2, g.hvy*0.8+float64(dy)*0.2
	if math.Hypot(g.hvx, g.hvy) > 0.15 {
		g.headWant = math.Atan2(g.hvx, -g.hvy) * 180 / math.Pi
	}
}

// stepHeading aproxima o rumo mostrado do desejado (chamado por quadro).
func (g *Game) stepHeading(dt, jitter float64) {
	d := math.Mod(g.headWant-g.heading+540, 360) - 180
	g.heading += d * math.Min(1, dt*2.2)
	if jitter > 0 {
		t := g.now() / 1000
		g.heading += jitter * (math.Sin(t*7.3)*0.6 + math.Sin(t*17.1)*0.4)
	}
	g.heading = math.Mod(g.heading+360, 360)
}

// compassJitter diz quanto a agulha treme onde o jogador está.
func (g *Game) compassJitter(me *proto.Player) float64 {
	if me.Y < g.info.OverH {
		return 0
	}
	switch g.regionName(me.X, me.Y) {
	case "Abismo de Dorrim", "Ventre de Fogo":
		return 28
	}
	return 0 // no resto do subsolo a agulha fica quieta (antes tremia sempre)
}

func (g *Game) drawCompass(dst *ebiten.Image, me *proto.Player, cw float64) {
	if g.opts.NoCompass {
		g.drawWorldInfo(dst, me, cw, 8) // sem a bússola, os ícones ficam no alto
		return
	}
	g.stepHeading(1.0/60, g.compassJitter(me))
	x0 := cw/2 - compassW/2
	y0 := 8.0
	const h = 26.0
	g.rect(dst, x0, y0, compassW, h, color.RGBA{0, 0, 0, 140})
	g.stroke(dst, x0, y0, compassW, h, color.RGBA{214, 176, 80, 90}, 1)
	for _, cx := range []float64{x0, x0 + compassW} {
		g.txt(dst, "+", cx, y0-8, color.RGBA{236, 200, 110, 200}, 12, true, alignCenter)
		g.txt(dst, "+", cx, y0+h-8, color.RGBA{236, 200, 110, 200}, 12, true, alignCenter)
	}
	// posX converte um rumo em graus na posição da faixa (ok = está à vista).
	posX := func(deg float64) (float64, bool) {
		d := math.Mod(deg-g.heading+540, 360) - 180
		if math.Abs(d) > compassFOV/2-4 {
			return 0, false
		}
		return x0 + compassW/2 + d/compassFOV*compassW, true
	}
	fade := func(x float64) uint8 { // some nas pontas
		e := 1 - math.Abs(x-(x0+compassW/2))/(compassW/2)
		return uint8(255 * clamp(e*2.2, 0, 1))
	}
	for deg := 0; deg < 360; deg += 15 {
		x, ok := posX(float64(deg))
		if !ok {
			continue
		}
		a := fade(x)
		if deg%45 == 0 {
			name := compassNames[deg/45]
			c := color.RGBA{200, 190, 170, a}
			if deg == 0 {
				c = color.RGBA{235, 90, 70, a} // o norte em vermelho, como numa agulha
			}
			g.txt(dst, name, x, y0+5, c, 13, true, alignCenter)
		} else {
			g.txt(dst, "'", x, y0+4, color.RGBA{140, 130, 115, a / 2}, 12, false, alignCenter)
		}
	}
	// Marcas: pontos descobertos mais perto (até 6), e a missão se a seta estiver ligada.
	type mark struct {
		deg, dist float64
		icon      string
		c         col
		name      string
	}
	var ms []mark
	under := me.Y >= g.info.OverH
	for i, t := range g.info.Travel {
		if !g.found(i) || (t.Y >= g.info.OverH) != under {
			continue
		}
		dx, dy := float64(t.X-me.X), float64(t.Y-me.Y)
		d := math.Hypot(dx, dy)
		if d < 6 || d > 700 {
			continue
		}
		icon, c := "ember", col{255, 160, 60}
		if t.Kind == 1 {
			icon, c = "map", col{210, 180, 120}
		}
		ms = append(ms, mark{math.Atan2(dx, -dy) * 180 / math.Pi, d, icon, c, t.Name})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].dist < ms[j].dist })
	ms = ms[:min(len(ms), 6)]
	if g.opts.QuestArrow {
		for _, mk := range g.cur.Marks {
			if mk.Side {
				continue
			}
			if wx, wy, ok, _ := g.markPos(mk, me); ok {
				dx, dy := wx-float64(me.X), wy-float64(me.Y)
				ms = append(ms, mark{math.Atan2(dx, -dy) * 180 / math.Pi, math.Hypot(dx, dy), "!", col{255, 211, 74}, mk.Title})
			}
			break
		}
	}
	best, bestD := -1, 1e9
	for i, m := range ms {
		x, ok := posX(m.deg)
		if !ok {
			continue
		}
		if d := math.Abs(x - (x0 + compassW/2)); d < bestD {
			best, bestD = i, d
		}
		if m.icon == "!" {
			g.txt(dst, "!", x, y0+h-12, cGold, 13, true, alignCenter)
			continue
		}
		g.drawTinted(dst, m.icon, m.c, x-6, y0+h-11, 12, false)
	}
	// Estação, clima, hora e lugar logo embaixo (hud_layout.go).
	iy := g.drawWorldInfo(dst, me, cw, y0+h+3)
	// O que está bem à frente ganha o nome e a distância embaixo; o nome fica
	// um pouco depois de sair da frente, para não piscar.
	now := g.now()
	if best >= 0 && bestD < 26 {
		m := ms[best]
		g.compassLbl, g.compassLblAt = fmt.Sprintf("%s · %dm", m.name, int(m.dist/2)), now
	}
	if t := now - g.compassLblAt; g.compassLbl != "" && t < 1200 {
		a := uint8(255 * clamp((1200-t)/400, 0, 1))
		tw := g.textW(g.compassLbl, 11)
		g.rect(dst, x0+compassW/2-tw/2-5, iy+2, tw+10, 15, color.RGBA{0, 0, 0, a / 2})
		g.txt(dst, g.compassLbl, x0+compassW/2, iy+3, color.RGBA{200, 184, 144, a}, 11, false, alignCenter)
	}
	// Agulha: um traço no meio da faixa.
	g.rect(dst, x0+compassW/2-0.5, y0+1, 1, h-2, color.RGBA{255, 230, 150, 160})
	g.txt(dst, "v", x0+compassW/2, y0-9, color.RGBA{255, 220, 130, 230}, 11, true, alignCenter)
}
