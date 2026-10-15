package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pesca, modo de combate e destino no mapa, do lado do cliente.
//   - G lança a linha (precisa da Vara de Pesca); a boia boia na água e,
//     quando o peixe morde, aparece "!" e a água espirra: G, E ou Espaço
//     puxam (servidor/pesca.go).
//   - B troca o modo de combate (pacifista, defensivo, agressivo); o ícone
//     fica ao lado do card, com dica, e clicar nele também troca.
//   - No mapa grande (M), clicar num lugar manda o personagem caminhar até
//     lá; o destino aparece no mapa, no mundo e na bússola. Andar cancela.

func init() {
	iconArt["fish"] = [12]string{
		"............",
		"............",
		"....oooo....",
		"..oo3344oo.o",
		".o3344a34oo2",
		"o334443332o2",
		"o223333322o2",
		".o2222222oo2",
		"..oo2222oo.o",
		"....oooo....",
		"............",
		"............",
	}
	iconArt["rod"] = [12]string{
		"..........o.",
		".........ob.",
		"........ob.s",
		".......ob..s",
		"......ob...s",
		".....ob....s",
		"....ob.....s",
		"...ob......o",
		"..ob......o3",
		".ob........o",
		"ob..........",
		"o...........",
	}
	iconArt["dove"] = [12]string{
		"............",
		"..........oo",
		".oo......o4o",
		"o44o....o43o",
		".o443oo4433o",
		"..o34444443o",
		"...o3333aoo.",
		"....o2222o..",
		".....oooo...",
		"......o.o...",
		"............",
		"............",
	}
	iconArt["book"] = [12]string{
		"............",
		".oooooooooo.",
		".o2333o333o.",
		".o2344o443o.",
		".o2333o333o.",
		".o2344o443o.",
		".o2333o333o.",
		".o2344o443o.",
		".o2333o333o.",
		".oooooooooo.",
		"............",
		"............",
	}
}

// modeIcons: pacifista (pomba), defensivo (escudo), agressivo (espada).
var modeIcons = []struct {
	key  string
	c    col
	name string
	hint string
}{
	{"dove", col{220, 230, 240}, "Pacifista", "só golpeia quando você manda"},
	{"shield", col{150, 180, 220}, "Defensivo", "revida sozinho em quem te feriu"},
	{"sword", col{230, 90, 70}, "Agressivo", "golpeia sozinho o que achar no caminho"},
}

func (g *Game) cycleMode() {
	g.send(proto.In{T: "mode", I: (g.cur.Mode + 1) % len(modeIcons)})
}

// drawModeIcon desenha o ícone do modo ao lado do card.
func (g *Game) drawModeIcon(dst *ebiten.Image, x, y float64) {
	m := modeIcons[clampi(g.cur.Mode, 0, len(modeIcons)-1)]
	const s = 30.0
	hov := g.hover(x, y, s, s)
	g.rect(dst, x, y, s, s, color.RGBA{0, 0, 0, 150})
	g.stroke(dst, x, y, s, s, pickRGBA(hov, rgbaHex(0xffd34a), color.RGBA{214, 176, 80, 120}), 1)
	g.drawTinted(dst, m.key, m.c, x+3, y+3, s-6, false)
	g.txt(dst, "B", x+s-6, y+s-12, cDimmer, 9, false, alignCenter)
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: s, h: s, fn: g.cycleMode})
	if hov {
		g.tooltipAt(dst, "Modo "+m.name+": "+m.hint+" (B troca)", x+s/2+140, y+s+6)
	}
}

// drawFishLine desenha a boia e a linha (no mundo, antes dos personagens).
func (g *Game) drawFishLine(me *proto.Player, T float64) {
	f := g.cur.Fish
	if f == nil {
		g.fishBit = false
		return
	}
	bob := int(math.Round(math.Sin(T*3) * 0.6))
	// linha: pontinhos do personagem até a boia
	sx, sy := float64(me.X), float64(me.Y)-10
	n := int(math.Hypot(float64(f.X)-sx, float64(f.Y)-sy))
	for i := 1; i < n; i += 2 {
		t := float64(i) / float64(n)
		g.setC(round(sx+(float64(f.X)-sx)*t), round(sy+(float64(f.Y)-sy)*t-math.Sin(t*math.Pi)*4), '.', col{200, 200, 190})
	}
	g.setC(f.X, f.Y+bob, 'o', col{240, 70, 60})
	g.setC(f.X, f.Y+bob-1, '|', col{230, 230, 230})
	if f.Bite {
		g.setC(f.X, f.Y-4, '!', col{255, 230, 80})
		for k := 0; k < 3; k++ {
			if rand.Float64() < 0.5 {
				g.spawn("foam", float64(f.X)+(rand.Float64()-0.5)*6, float64(f.Y), (rand.Float64()-0.5)*8, -6-rand.Float64()*6, 0.4)
			}
		}
		if !g.fishBit {
			g.sfx("splash", 0.9)
			g.sfx("msg", 0.6)
		}
	}
	g.fishBit = f.Bite
}

// drawGoal marca o destino no mundo (uma bandeirinha que pulsa).
func (g *Game) drawGoal(T float64) {
	gl := g.cur.Goal
	if gl == nil {
		return
	}
	k := 0.6 + 0.4*math.Sin(T*4)
	c := col{255 * k, 211 * k, 74 * k}
	g.setC(gl[0], gl[1], 'X', c)
	for dy := 1; dy <= 5; dy++ {
		g.setC(gl[0], gl[1]-dy, '|', col{200, 180, 140})
	}
	g.setC(gl[0]+1, gl[1]-5, '>', c)
	g.setC(gl[0]+1, gl[1]-4, '>', c)
}

// bigMapClick registra o clique no mapa grande: caminhar até o ponto.
func (g *Game) bigMapClick(mx, my, mw, mh float64, src zoneRect) {
	g.uiRects = append(g.uiRects, uiRect{x: mx, y: my, w: mw, h: mh, fn: func() {
		wx := src.x + int((g.mouseX-mx)/mw*float64(src.w))
		wy := src.y + int((g.mouseY-my)/mh*float64(src.h))
		g.send(proto.In{T: "goto", I: wx, Slot: wy})
		g.bigMap = false
		g.sfx("click", 0.6)
	}})
}

// goalText é a linha do destino para o mapa grande.
func (g *Game) goalText(me *proto.Player) string {
	if gl := g.cur.Goal; gl != nil {
		return fmt.Sprintf("Destino a %dm · andar cancela", int(math.Hypot(float64(gl[0]-me.X), float64(gl[1]-me.Y))/2))
	}
	return "Clique num lugar para caminhar até lá"
}
