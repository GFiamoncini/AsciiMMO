package main

import (
	"image/color"
	"math"
	"math/rand"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------- campo de visão ----------
// Sombra projetada (shadowcasting em 8 octantes): o jogador só vê o que está
// em linha reta até ele. O que já foi visto fica guardado e aparece em cinza
// escuro; o que nunca foi visto é preto.

type fov struct {
	cx, cy, r int
	vis       []bool
	mem       []bool // células do mapa já vistas nesta sessão
}

var octants = [8][4]int{{1, 0, 0, 1}, {0, 1, 1, 0}, {0, -1, 1, 0}, {-1, 0, 0, 1}, {-1, 0, 0, -1}, {0, -1, -1, 0}, {0, 1, -1, 0}, {1, 0, 0, -1}}

func (g *Game) opaque(x, y int) bool {
	m := g.m
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return true
	}
	f := m.flags[y*m.w+x]
	return f&1 != 0 && f&4 == 0
}

func (g *Game) computeFOV(cx, cy, r int) {
	f := &g.fov
	n := (2*r + 1) * (2*r + 1)
	if len(f.vis) != n {
		f.vis = make([]bool, n)
	} else {
		clear(f.vis)
	}
	if len(f.mem) != g.m.w*g.m.h {
		f.mem = make([]bool, g.m.w*g.m.h)
	}
	// O olho fica nos pés; se ali for obstáculo (colado numa parede), usa a
	// célula livre mais próxima, senão a tela inteira escureceria.
	if g.opaque(cx, cy) {
	search:
		for d := 1; d <= 4; d++ {
			for _, o := range [][2]int{{0, d}, {-d, 0}, {d, 0}, {0, -d}, {-d, d}, {d, d}, {-d, -d}, {d, -d}} {
				if !g.opaque(cx+o[0], cy+o[1]) {
					cx, cy = cx+o[0], cy+o[1]
					break search
				}
			}
		}
	}
	f.cx, f.cy, f.r = cx, cy, r
	g.markVis(cx, cy)
	for _, o := range octants {
		g.castLight(cx, cy, 1, 1, 0, r, o[0], o[1], o[2], o[3])
	}
}

func (g *Game) markVis(x, y int) {
	f := &g.fov
	dx, dy := x-f.cx+f.r, y-f.cy+f.r
	if dx < 0 || dy < 0 || dx > 2*f.r || dy > 2*f.r {
		return
	}
	f.vis[dy*(2*f.r+1)+dx] = true
	if x >= 0 && y >= 0 && x < g.m.w && y < g.m.h {
		f.mem[y*g.m.w+x] = true
	}
}

func (g *Game) castLight(cx, cy, row int, start, end float64, r, xx, xy, yx, yy int) {
	if start < end {
		return
	}
	newStart := 0.0
	for j := row; j <= r; j++ {
		dx, dy := -j-1, -j
		blocked := false
		for dx <= 0 {
			dx++
			x, y := cx+dx*xx+dy*xy, cy+dx*yx+dy*yy
			lSlope, rSlope := (float64(dx)-0.5)/(float64(dy)+0.5), (float64(dx)+0.5)/(float64(dy)-0.5)
			if start < rSlope {
				continue
			} else if end > lSlope {
				break
			}
			if dx*dx+dy*dy <= r*r {
				g.markVis(x, y)
			}
			if blocked {
				if g.opaque(x, y) {
					newStart = rSlope
					continue
				}
				blocked, start = false, newStart
			} else if g.opaque(x, y) && j < r {
				blocked = true
				g.castLight(cx, cy, j+1, start, lSlope, r, xx, xy, yx, yy)
				newStart = rSlope
			}
		}
		if blocked {
			break
		}
	}
}

// visible diz se a célula está à vista agora.
func (g *Game) visible(x, y int) bool {
	f := &g.fov
	dx, dy := x-f.cx+f.r, y-f.cy+f.r
	return dx >= 0 && dy >= 0 && dx <= 2*f.r && dy <= 2*f.r && f.vis[dy*(2*f.r+1)+dx]
}

// seen diz se um personagem em (x, y) aparece: os pés ou o corpo à vista.
func (g *Game) entityVisible(x, y float64) bool {
	ix, iy := int(x), int(y)
	return g.visible(ix, iy) || g.visible(ix, iy-8) || g.visible(ix, iy-16)
}

// fovRadius depende da luz: dia aberto vê longe; noite, neblina e subsolo, perto.
func (g *Game) fovRadius(under bool, sun float64) int {
	switch {
	case under:
		return 70
	case g.cur.Wx == 3:
		return 60
	}
	return int(100 + 140*sun)
}

// ---------- copas translúcidas ----------

// ---------- o que fica na frente de quem (2.5D) ----------
// Copas, telhados e fachadas são células "por cima" (flag 2) que sobem a
// partir de uma linha de base: o pé do tronco, o chão da muralha. Quem tem os
// pés acima dessa linha (mais ao norte) está atrás e some sob elas; quem está
// abaixo fica na frente e é desenhado por cima. A base de cada célula alta sai
// de uma varredura de baixo para cima em cada coluna: segue as células por
// cima e, logo abaixo delas, as que bloqueiam (o pé da parede).

type bases struct {
	x0, y0, w, h int
	y            []int32
}

const baseBelow = 80 // linhas abaixo da tela onde ainda se procura a base

func (g *Game) computeBases(x0, y0, cols, rows int) {
	m, b := g.m, &g.bases
	b.x0, b.y0, b.w, b.h = x0, y0, cols, rows+baseBelow
	if len(b.y) != b.w*b.h {
		b.y = make([]int32, b.w*b.h)
	}
	for sx := 0; sx < b.w; sx++ {
		wx := x0 + sx
		var prev int32 = -1 // base da célula de baixo, se ela for por cima ou parede
		prevOver := false
		for sy := b.h - 1; sy >= 0; sy-- {
			wy := y0 + sy
			i := sy*b.w + sx
			if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h {
				b.y[i], prev, prevOver = -1, -1, false
				continue
			}
			f := m.flags[wy*m.w+wx]
			switch {
			case f&2 != 0:
				if prev < 0 {
					prev = int32(wy)
				}
				b.y[i], prevOver = prev, true
			case f&1 != 0 && f&4 == 0:
				if prevOver || prev < 0 {
					prev = int32(wy) // o pé da parede é a última linha que bloqueia
				}
				b.y[i], prevOver = prev, false
			default:
				b.y[i], prev, prevOver = -1, -1, false
			}
		}
	}
}

// baseAt é a linha de base da estrutura alta na célula, ou -1.
func (g *Game) baseAt(wx, wy int) int {
	b := &g.bases
	sx, sy := wx-b.x0, wy-b.y0
	if sx < 0 || sy < 0 || sx >= b.w || sy >= b.h {
		return -1
	}
	return int(b.y[sy*b.w+sx])
}

// over diz se a célula é "por cima" (copa, telhado, fachada).
func (g *Game) over(wx, wy int) bool {
	m := g.m
	return wx >= 0 && wy >= 0 && wx < m.w && wy < m.h && m.flags[wy*m.w+wx]&2 != 0
}

// cutoutAt diz como a célula alta fica em volta do jogador quando ele está
// atrás dela: 2 = vazada (buraco), 1 = pontilhada (borda), 0 = inteira.
func (g *Game) cutoutAt(wx, wy int) int {
	if !g.over(wx, wy) || float64(g.baseAt(wx, wy)) <= g.meFeet {
		return 0
	}
	dx, dy := float64(wx)-g.seeX, (float64(wy)-g.seeY)*1.2
	switch d2 := dx*dx + dy*dy; {
	case d2 < 12*12:
		return 2
	case d2 < 16*16 && (wx+wy)%2 == 0:
		return 1
	}
	return 0
}

// hides diz se a célula alta tapa um corpo com os pés na linha feet.
func (g *Game) hides(wx, wy, feet int) bool {
	if !g.over(wx, wy) {
		return false
	}
	if base := g.baseAt(wx, wy); base >= 0 && feet >= base {
		return false // na frente da estrutura
	}
	return g.cutoutAt(wx, wy) == 0
}

// ---------- marcas no chão: pegadas e sangue ----------

type decal struct {
	x, y int
	t0   float64
	c    col
	code byte
	life float64
}

func (g *Game) addDecal(x, y int, code byte, c col, life float64) {
	if len(g.decals) > 900 {
		g.decals = g.decals[1:]
	}
	g.decals = append(g.decals, decal{x, y, g.now(), c, code, life})
}

// drawDecals pinta as marcas por cima do chão, sumindo aos poucos.
func (g *Game) drawDecals(now float64) {
	n := 0
	for _, d := range g.decals {
		age := (now - d.t0) / (d.life * 1000)
		if age >= 1 {
			continue
		}
		g.decals[n] = d
		n++
		sx, sy := d.x-g.x0, d.y-g.y0
		if sx < 0 || sy < 0 || sx >= g.cols || sy >= g.rows {
			continue
		}
		o := (sy*g.cols + sx) * 4
		k := 1 - age
		g.buf[o] = cb(float64(g.buf[o])*(1-k) + d.c[0]*k)
		g.buf[o+1] = cb(float64(g.buf[o+1])*(1-k) + d.c[1]*k)
		g.buf[o+2] = cb(float64(g.buf[o+2])*(1-k) + d.c[2]*k)
		g.buf[o+3] = d.code
	}
	g.decals = g.decals[:n]
}

// soft diz se o chão guarda pegada: neve (clara e sem cor) ou lama.
func (g *Game) soft(x, y int) bool {
	m := g.m
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return false
	}
	i := y*m.w + x
	if m.flags[i]&8 != 0 {
		return true
	}
	r, gg, b := int(m.color[i*3]), int(m.color[i*3+1]), int(m.color[i*3+2])
	return r > 175 && gg > 180 && b > 190 && max(r, gg, b)-min(r, gg, b) < 50 || g.cur.Season == 3 && m.flags[i] == 0
}

// trackSteps deixa pegadas de quem anda na neve ou na lama e marca sangue
// (ou pó de osso) e faíscas quando alguém é atingido.
func (g *Game) trackSteps(key string, x, y float64, hit float64, bone bool) {
	ix, iy := int(x), int(y)
	last, ok := g.lastStep[key]
	if !ok || abs(last[0]-ix)+abs(last[1]-iy) >= 3 {
		g.lastStep[key] = [2]int{ix, iy}
		if ok && g.soft(ix, iy) {
			side := 1
			if (ix+iy)/3%2 == 0 {
				side = -1
			}
			g.addDecal(ix+side, iy, '.', col{70, 74, 90}, 30)
		}
	}
	if hit < 0.06 && !g.hitMark[key] {
		g.hitMark[key] = true
		g.burst("spark", x, y-14, 10, 30, 0.3)
		c, kind := col{150, 18, 18}, "blood"
		if bone {
			c, kind = col{190, 180, 160}, "bone"
		}
		for k := 0; k < 4; k++ {
			g.addDecal(ix+rand.Intn(7)-3, iy+rand.Intn(3)-1, '.', c, 12)
		}
		g.burst(kind, x, y-14, 14, 26, 0.6)
	} else if hit > 0.3 {
		g.hitMark[key] = false
	}
}

// ---------- balões de fala ----------

type bubble struct {
	text string
	t0   float64
}

// takeChat acha as falas novas e prende um balão na cabeça de quem falou.
func (g *Game) takeChat(s *proto.State) {
	old := map[string]bool{}
	if g.cur != nil {
		for _, l := range g.cur.Chat {
			old[l.From+"\x00"+l.Text] = true
		}
	}
	for _, l := range s.Chat {
		if old[l.From+"\x00"+l.Text] {
			continue
		}
		// Tudo vai para o registro (tecla L); as falas também viram balão.
		g.chatLog = append(g.chatLog, logLine{l.From, l.Color, l.Text, g.now()})
		if g.logScroll > 0 { // lendo mais atrás: a janela fica parada no que se lia
			g.logScroll++
		}
		if len(g.chatLog) > 300 {
			g.chatLog = g.chatLog[len(g.chatLog)-300:]
			g.logSeen = max(0, g.logSeen-1)
		}
		if l.From != "" && g.cur != nil {
			g.bubbles[l.From] = bubble{l.Text, g.now()}
		}
	}
}

// logLine é uma linha do registro: fala de jogador (from) ou aviso do sistema.
type logLine struct {
	from  string
	color int
	text  string
	at    float64
}

// drawBubbles desenha os balões sobre quem falou nos últimos 5 segundos.
func (g *Game) drawBubbles(dst *ebiten.Image, camL, camT float64, pos map[string][2]float64) {
	now, cs := g.now(), float64(g.cs)
	for name, b := range g.bubbles {
		if now-b.t0 > 5000 {
			delete(g.bubbles, name)
			continue
		}
		p, ok := pos[name]
		if !ok {
			continue
		}
		lines := wrap(b.text, 26)
		if len(lines) > 3 {
			lines = lines[:3]
		}
		w := 0.0
		for _, l := range lines {
			w = math.Max(w, g.textW(l, 12))
		}
		x := (p[0] - camL) * cs / g.ui
		y := (p[1]-camT)*cs/g.ui - 18 - float64(len(lines))*15
		g.rect(dst, x-w/2-6, y-4, w+12, float64(len(lines))*15+8, color.RGBA{230, 226, 210, 235})
		g.txt(dst, "v", x, y+float64(len(lines))*15+1, color.RGBA{230, 226, 210, 235}, 12, true, alignCenter)
		for i, l := range lines {
			g.txt(dst, l, x, y+float64(i)*15, color.RGBA{20, 16, 24, 255}, 12, false, alignCenter)
		}
	}
}

// ---------- itens no chão como texto ----------

// lootGlyph é o "idioma" do chão: ! poção, $ dinheiro, ) arma, [ armadura,
// " amuleto, * material, % erva.
func lootGlyph(kind int) byte {
	switch kind {
	case 0:
		return '!'
	case 1:
		return ')'
	case 2, 3, 4:
		return '['
	case 5:
		return '"'
	case 6:
		return '*'
	case 7:
		return '%'
	case 8:
		return '$'
	}
	return '?'
}

// lightBase é a linha do chão de uma luz fixa: se ela está numa estrutura
// alta (braseiro no alto da torre), é a base da estrutura.
func (g *Game) lightBase(x, y float64) float64 {
	if b := g.baseAt(int(x), int(y)); b >= 0 {
		return float64(b)
	}
	return y
}
