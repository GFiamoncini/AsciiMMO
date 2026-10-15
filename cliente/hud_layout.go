package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Arrumação da tela (pedido de 14/10): o minimapa fica à esquerda, embaixo do
// card do personagem; os outros jogadores viram fichas pequenas (ícone,
// nome, nível e um fio de vida); estação, clima e hora viram ícones com dica,
// embaixo da bússola, junto da hora e do nome do lugar. A coluna da direita
// fica só com a missão e o menu.

func init() {
	iconArt["sun"] = [12]string{
		".....a......",
		"..a..4..a...",
		"...a.4.a....",
		"....o44o....",
		".a.o3444o.a.",
		"aaao34444aaa",
		".a.o23344o..",
		"....o233o...",
		"...a.oo.a...",
		"..a..a...a..",
		".....a......",
		"............",
	}
	iconArt["moon"] = [12]string{
		"....oooo....",
		"..oo3344o...",
		".o334o.oo...",
		".o34o.......",
		"o334o.......",
		"o334o.......",
		"o3344o......",
		".o3344oo.oo.",
		".oo334444o..",
		"..oo2333oo..",
		"....oooo....",
		"............",
	}
	iconArt["cloud"] = [12]string{
		"............",
		"....oooo....",
		"...o3344o...",
		".ooo33344oo.",
		"o2333333344o",
		"o2223333334o",
		".oooooooooo.",
		"............",
		"............",
		"............",
		"............",
		"............",
	}
	iconArt["rain"] = [12]string{
		"....oooo....",
		"...o3344o...",
		".ooo33344oo.",
		"o2333333344o",
		"o2223333334o",
		".oooooooooo.",
		"..a...a...a.",
		".a...a...a..",
		"....a...a...",
		"...a...a....",
		"............",
		"............",
	}
	iconArt["storm"] = [12]string{
		"....oooo....",
		"...o2233o...",
		".ooo22233oo.",
		"o1222222233o",
		"o1112222223o",
		".ooooo4oooo.",
		".....44.....",
		"....444.....",
		"......44....",
		".....44.....",
		"....4.......",
		"............",
	}
	iconArt["fogicon"] = [12]string{
		"............",
		"..33333333..",
		"............",
		".2222222222.",
		"............",
		"..33333333..",
		"............",
		".2222222222.",
		"............",
		"...333333...",
		"............",
		"............",
	}
	iconArt["flower"] = [12]string{
		"....o3o.....",
		"...o343o....",
		".oo3o4o3oo..",
		"o3433a3343o.",
		".oo3o4o3oo..",
		"...o343o....",
		"....o3o.....",
		".....b......",
		"...b.b.b....",
		"....bbb.....",
		".....b......",
		"............",
	}
	iconArt["leaf"] = [12]string{
		"........oo..",
		"......oo4o..",
		"....oo344o..",
		"...o3344o...",
		"..o3343o....",
		"..o334o.....",
		".o334o......",
		".o23o.......",
		".oo.........",
		"o...........",
		"............",
		"............",
	}
	iconArt["snow"] = [12]string{
		".....4......",
		"...4.4.4....",
		"....444.....",
		".4..434..4..",
		"..4.434.4...",
		"444434344444",
		"..4.434.4...",
		".4..434..4..",
		"....444.....",
		"...4.4.4....",
		".....4......",
		"............",
	}
}

// drawMiniLeft desenha o minimapa (cabeçalho que recolhe e o mapa) a partir de
// (x, y) com largura w; devolve onde termina.
func (g *Game) drawMiniLeft(dst *ebiten.Image, me *proto.Player, x, y, w float64) float64 {
	src := g.mapSrc(me)
	lh := 17.0
	hh := 1.3 * lh
	mw := w - 2
	mh := math.Min(math.Round(mw*float64(src.h)/float64(src.w)), 240)
	if g.fog == nil {
		g.initFog()
	}
	g.reveal(me, g.now())
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w * btoF(!g.bigMap), h: hh, fn: func() { // com o mapa grande aberto, o clique é dele
		g.mapShut = !g.mapShut
		v := "0"
		if g.mapShut {
			v = "1"
		}
		os.WriteFile(filepath.Join(configDir(), "mapa_recolhido"), []byte(v), 0o644)
	}})
	g.rect(dst, x, y, w, hh, color.RGBA{0, 0, 0, 170})
	g.stroke(dst, x, y, w, hh, color.RGBA{214, 176, 80, 90}, 1)
	name := "Mapa"
	if src.name != "" {
		name = "Mapa · " + src.name
	}
	g.txt(dst, name, x+8, y+3, cDim, 12, false, alignLeft)
	g.txt(dst, "[M] ampliar", x+w-26, y+3, cDimmer, 12, false, alignRight)
	arrow := "v"
	if g.mapShut {
		arrow = ">"
	}
	g.txt(dst, arrow, x+w-12, y+3, cGold, 12, false, alignCenter)
	y += hh
	if !g.mapShut {
		g.panel(dst, x, y, w, mh+2, color.RGBA{214, 176, 80, 90})
		g.drawMap(dst, me, x+1, y+1, mw, mh, src, false)
		y += mh + 2
	}
	return y
}

// drawPartyChips: cada outro jogador numa ficha de uma linha.
func (g *Game) drawPartyChips(dst *ebiten.Image, me *proto.Player, x, y, w float64) float64 {
	shown := 0
	for _, p := range g.cur.Players {
		if p.ID == me.ID || shown == 6 {
			continue
		}
		const h = 22.0
		cy := y + 4 + float64(shown)*(h+3)
		g.rect(dst, x, cy, w, h, color.RGBA{0, 0, 0, 150})
		g.stroke(dst, x, cy, w, h, color.RGBA{120, 110, 90, 120}, 1)
		g.drawPortrait(dst, p.Class, x+2, cy+2, h-4)
		name := p.Name
		if p.Pz {
			name += " · pausado"
		}
		g.txt(dst, g.fit(name, 12, w*0.5), x+h+4, cy+4, playerColor(p.Color).rgba(), 12, false, alignLeft)
		g.txt(dst, fmt.Sprintf("nv %d", p.Level), x+w*0.62, cy+4, cDim, 11, false, alignLeft)
		k := clamp(float64(p.HP)/float64(max(1, p.MaxHP)), 0, 1)
		bx, bw := x+w*0.75, w*0.25-6
		g.rect(dst, bx, cy+8, bw, 6, color.RGBA{60, 20, 20, 255})
		g.rect(dst, bx, cy+8, bw*k, 6, color.RGBA{220, 70, 60, 255})
		if g.hover(x, cy, w, h) {
			g.tooltip(dst, fmt.Sprintf("%s · nível %d · vida %d/%d · mana %d/%d", p.Name, p.Level, p.HP, p.MaxHP, p.MP, p.MaxMP))
		}
		shown++
	}
	if shown == 0 {
		return y
	}
	return y + 4 + float64(shown)*25
}

// placeName é o nome do lugar onde o jogador está.
func (g *Game) placeName(me *proto.Player) string {
	place := "Ermos"
	if me.Y >= g.info.OverH {
		place = g.mapSrc(me).name
	}
	for _, r := range g.info.Regions {
		if math.Hypot(float64(me.X-r.X), float64(me.Y-r.Y)) <= float64(r.R) {
			return r.Name
		}
	}
	return place
}

// drawWorldInfo: embaixo da bússola, os ícones de estação, clima e hora (com
// dica ao passar o mouse), a hora e o lugar. Devolve onde a linha termina.
func (g *Game) drawWorldInfo(dst *ebiten.Image, me *proto.Player, cw, y float64) float64 {
	cur := g.cur
	season := clampi(cur.Season, 0, 3)
	sIcon := []struct {
		key string
		c   col
	}{{"flower", col{240, 150, 200}}, {"sun", col{255, 200, 80}}, {"leaf", col{230, 130, 50}}, {"snow", col{220, 235, 255}}}[season]
	wx := clampi(cur.Wx, 0, 3)
	wIcon := []struct {
		key, name string
		c         col
	}{{"cloud", "tempo bom", col{200, 210, 230}}, {"rain", "chuva", col{120, 160, 230}}, {"storm", "tempestade", col{255, 220, 90}}, {"fogicon", "neblina", col{170, 175, 190}}}[wx]
	mins := int(cur.Tod * 1440)
	day := daylight(cur.Tod) > 0.3
	tKey, tc, tName := "sun", col{255, 210, 90}, "dia"
	if !day {
		tKey, tc, tName = "moon", col{170, 190, 255}, "noite"
	}
	under := me.Y >= g.info.OverH
	clock := fmt.Sprintf("%02d:%02d", mins/60, mins%60)
	place := g.placeName(me)
	const isz = 18.0
	tw := g.textW(clock, 12) + 10 + g.textW(place, 12)
	w := 3*(isz+4) + 8 + tw + 12
	x := cw/2 - w/2
	g.rect(dst, x, y, w, isz+6, color.RGBA{0, 0, 0, 140})
	icons := []struct {
		key  string
		c    col
		hint string
	}{
		{sIcon.key, sIcon.c, "Estação: " + seasons[season].name},
		{wIcon.key, wIcon.c, "Clima: " + wIcon.name},
		{tKey, tc, fmt.Sprintf("Hora: %s (%s)", clock, tName)},
	}
	if under {
		icons[1].hint = "Clima: no subsolo não se sente o tempo"
	}
	ix := x + 6
	for _, ic := range icons {
		g.drawTinted(dst, ic.key, ic.c, ix, y+3, isz, under && ic.key != sIcon.key)
		if g.hover(ix, y, isz+4, isz+6) {
			g.tooltipAt(dst, ic.hint, ix+isz/2, y+isz+10)
		}
		ix += isz + 4
	}
	g.txt(dst, clock, ix+4, y+5, pickC(day, cGold, rgbaHex(0x8aa0ff)), 12, true, alignLeft)
	g.txt(dst, place, ix+14+g.textW(clock, 12), y+5, cText, 12, false, alignLeft)
	return y + isz + 6
}

// tooltip escreve uma dica junto ao mouse.
func (g *Game) tooltip(dst *ebiten.Image, s string) {
	g.tooltipAt(dst, s, g.mouseX+14, g.mouseY+18)
}

func (g *Game) tooltipAt(dst *ebiten.Image, s string, x, y float64) {
	tw := g.textW(s, 12)
	x = clamp(x-tw/2, 4, float64(g.W)/g.ui-tw-12)
	g.rect(dst, x-4, y-2, tw+10, 18, color.RGBA{10, 8, 14, 235})
	g.stroke(dst, x-4, y-2, tw+10, 18, rgbaHex(0x8a7040), 1)
	g.txt(dst, s, x+1, y, cText, 12, false, alignLeft)
}

func btoF(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
