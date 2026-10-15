package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Estações: nome, o que cai do céu (tipo de partícula) e quantas por quadro.
var seasons = []struct {
	name, fall string
	rate       float64
}{{"Primavera", "petal", 0.4}, {"Verão", "", 0}, {"Outono", "leaf", 0.8}, {"Inverno", "snow", 3}}

// Tons das folhas no outono, sorteados por célula.
var autumn = [][3]float64{{1.35, 0.72, 0.4}, {1.25, 0.55, 0.35}, {1.3, 0.95, 0.4}, {1.05, 0.5, 0.3}}

// daylight vai de 0 (noite fechada) a 1 (dia claro) conforme a hora.
func daylight(tod float64) float64 { return clamp(math.Sin((tod-0.25)*tau)*1.4+0.35, 0, 1) }

// ambient é o multiplicador de cor do céu: azulado à noite, claro de dia e
// alaranjado no nascer e no pôr do sol.
func ambient(sun float64) col {
	// Noites mais fundas e um dia encoberto, puxado para o frio; o crepúsculo
	// fica cor de sangue.
	a := mix3(col{0.55, 0.6, 0.86}, col{1.42, 1.4, 1.3}, sun)
	dusk := math.Exp(-math.Pow((sun-0.3)/0.22, 2))
	return col{a[0] + 0.55*dusk, a[1] - 0.05*dusk, a[2] - 0.3*dusk}
}

// fogAt é a densidade da névoa (0..0.3) numa célula: manchas de ruído que
// andam com o vento.
func fogAt(x, y int, T float64) float64 {
	fx, fy := float64(x)/46+T*0.05, float64(y)/30
	ix, iy := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(ix), fy-float64(iy)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	v := func(a, b int) float64 { return float64(hash2(a, b, 909)&1023) / 1023 }
	top := v(ix, iy) + (v(ix+1, iy)-v(ix, iy))*tx
	bot := v(ix, iy+1) + (v(ix+1, iy+1)-v(ix, iy+1))*tx
	n := top + (bot-top)*ty
	return clamp((n-0.4)*0.4, 0, 0.22)
}

// pos é a posição interpolada de um personagem entre dois estados.
type pos struct{ x, y, vx, vy float64 }

type ent struct {
	key    string
	pos    pos
	player *proto.Player
	mob    *proto.Mob
	steed  *proto.Thing
}

func (g *Game) frame(screen *ebiten.Image) {
	now := g.now()
	dt := clamp((now-g.lastFrame)/1000, 0, 0.1)
	g.lastFrame = now
	g.frames++
	if now-g.fpsAt >= 1000 {
		g.fps, g.frames, g.fpsAt = int(math.Round(float64(g.frames)*1000/(now-g.fpsAt))), 0, now
		if g.shotPath != "" {
			fmt.Fprintf(os.Stderr, "fps %d cels %dx%d cs %d alvo %d tela %dx%d\n", g.fps, g.cols, g.rows, g.cs, g.csTarget, g.W, g.H)
		}
	}
	if g.cs != g.csTarget && now-g.csAt > 40 {
		g.cs += sign(float64(g.csTarget - g.cs))
		g.csAt = now
	}
	cs := float64(g.cs)
	g.g.buildAtlas(g.cs)

	// Os personagens andam do estado anterior ao atual; um tick de atraso
	// compra movimento liso entre os 20 estados por segundo.
	a := clamp((now-g.tCur)/55, 0, 1)
	cur, prev, m := g.cur, g.prev, g.m
	T := cur.T + (now-g.tCur)/1000
	at := func(x, y float64, ox, oy float64, ok bool) pos {
		if !ok || math.Abs(ox-x)+math.Abs(oy-y) > 70 {
			return pos{x, y, 0, 0}
		}
		return pos{ox + (x-ox)*a, oy + (y-oy)*a, x - ox, y - oy}
	}
	me := cur.pmap[g.info.ID]
	if me == nil {
		screen.Fill(rgbaHex(0x09090f))
		return
	}
	var mePos pos
	if prev != nil {
		if o := prev.pmap[me.ID]; o != nil {
			mePos = at(float64(me.X), float64(me.Y), float64(o.X), float64(o.Y), true)
		} else {
			mePos = pos{float64(me.X), float64(me.Y), 0, 0}
		}
	} else {
		mePos = pos{float64(me.X), float64(me.Y), 0, 0}
	}

	W, H := float64(g.W), float64(g.H)
	vw, vh := W/cs, H/cs
	mw, mh := float64(m.w), float64(m.h)
	camL := clamp(mePos.x-vw/2, 0, mw-vw)
	if vw >= mw {
		camL = (mw - vw) / 2
	}
	camT := clamp(mePos.y-14-vh/2, 0, mh-vh)
	if vh >= mh {
		camT = (mh - vh) / 2
	}
	if mePos.y >= float64(g.info.OverH) {
		// No subsolo a câmera segue o personagem sem parar na borda do mapa:
		// a masmorra fica no centro da tela, com escuro em volta.
		camL, camT = mePos.x-vw/2, mePos.y-14-vh/2
	}
	g.x0, g.y0 = int(math.Floor(camL)), int(math.Floor(camT))
	g.cols, g.rows = int(math.Ceil(vw))+2, int(math.Ceil(vh))+2
	if len(g.buf) != g.cols*g.rows*4 {
		g.buf = make([]byte, g.cols*g.rows*4)
	}
	g.meScreen = &[2]float64{(mePos.x - camL) * cs, (mePos.y - 14 - camT) * cs}
	cols, rows, x0, y0, buf := g.cols, g.rows, g.x0, g.y0, g.buf

	// Luzes deste quadro: as fixas que alcançam a tela e as dos efeitos (que
	// entram um quadro depois de criadas, o que não se nota).
	flick := 0.85 + 0.12*math.Sin(T*9) + 0.05*math.Sin(T*23)
	lit := g.lights
	g.lights = nil
	under := mePos.y >= float64(g.info.OverH)
	sun := 0.0
	if !under {
		sun = daylight(cur.Tod)
	}
	amb := col{1, 1, 1}
	if !under {
		amb = ambient(sun)
	}
	glowK := 1 - 0.7*sun
	season := cur.Season
	if under {
		season = 1
	}
	if f := seasons[season]; f.fall != "" {
		for n := f.rate; n > 0; n-- {
			if rand.Float64() < n {
				g.spawn(f.fall, float64(x0)+rand.Float64()*float64(cols), float64(y0)+rand.Float64()*float64(rows)*0.9, -3-rand.Float64()*5, 6+rand.Float64()*6, 1.5+rand.Float64())
			}
		}
	}
	g.computeBases(x0-70, y0-70, cols+140, rows+70) // com margem: o que está fora da tela também faz sombra
	g.labels = g.labels[:0]
	for i, l := range g.info.Lights {
		lx, ly, r, kind := float64(l[0]), float64(l[1]), float64(l[2]), l[3]
		if i == 0 && cur.Event == 1 {
			continue // invasão: a fogueira da vila se apagou
		}
		if lx+r < float64(x0) || ly+r < float64(y0) || lx-r > float64(x0+cols) || ly-r > float64(y0+rows) {
			continue
		}
		fi := float64(i)
		switch kind {
		case 3:
			lit = append(lit, lightT{lx, ly, r, r * r, 0.9 + 0.1*math.Sin(T*1.5+fi), spore, sporeAdd, g.lightBase(lx, ly)})
		case 2:
			lit = append(lit, lightT{lx, ly, r, r * r, glowK * (0.85 + 0.15*math.Sin(T*2+fi)), cool, coolAdd, g.lightBase(lx, ly)})
		default:
			lit = append(lit, lightT{lx, ly, r, r * r, glowK * flick * (0.9 + 0.1*math.Sin(T*7+fi*2)), warm, warmAdd, g.lightBase(lx, ly)})
		}
	}
	g.seeX, g.seeY, g.meFeet = mePos.x, mePos.y-12, mePos.y
	g.computeFOV(int(mePos.x), int(mePos.y), g.fovRadius(under, sun))
	// No subsolo, sem tocha nem lanterna, o jogador só enxerga um pouco em
	// volta (olhos acostumados ao escuro: luz fria, fraca e curta). Antes era
	// uma luz verde-amarelada de raio 38, mais forte que a fogueira, que
	// pintava a caverna toda de verde. A luz de verdade vem da tocha/lanterna
	// (Player.Light), das fogueiras, tochas na parede e cogumelos.
	if under {
		lit = append(lit, lightT{mePos.x, mePos.y - 8, 20, 20 * 20, 0.55, col{0.75, 0.82, 1.0}, col{14, 16, 26}, mePos.y})
	}

	// Quem dobra o capim ao passar.
	var benders []float64
	inView := func(x, y float64) bool {
		return x >= float64(x0-6) && x < float64(x0+cols+6) && y >= float64(y0-4) && y < float64(y0+rows+4)
	}
	for _, p := range cur.Players {
		if inView(float64(p.X), float64(p.Y)) {
			benders = append(benders, float64(p.X), float64(p.Y))
		}
	}
	for _, e := range cur.Mobs {
		if inView(e.X, e.Y) {
			benders = append(benders, e.X, e.Y)
		}
	}
	for _, e := range cur.Steeds {
		if inView(e.X, e.Y) {
			benders = append(benders, e.X, e.Y)
		}
	}
	wind := T * 1.6
	tw := int(T * 3)

	// Terreno, com a luz, a água, o capim e os vaga-lumes.
	parallelRows(rows, func(sy int) {
		wy := y0 + sy
		rl := rowLit(lit, float64(wy)) // só as luzes que alcançam esta linha
		for sx := 0; sx < cols; sx++ {
			wx, o := x0+sx, (sy*cols+sx)*4
			if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h {
				buf[o+3] = 0
				continue
			}
			i := wy*m.w + wx
			ch := m.glyph[i]
			fl := m.flags[i]
			r, gg, b := float64(m.color[i*3])*amb[0], float64(m.color[i*3+1])*amb[1], float64(m.color[i*3+2])*amb[2]
			if fl&64 != 0 { // janela acesa, cristal, lava: brilham sozinhas, sem o escuro da noite
				r, gg, b = float64(m.color[i*3]), float64(m.color[i*3+1]), float64(m.color[i*3+2])
			}
			if season != 1 && fl&4 == 0 {
				veg := gg > r*1.08 && gg > b*1.05
				switch {
				case season == 3:
					mm, k := (r+gg+b)/3, 0.22
					if veg {
						k = 0.6
					}
					r += (mm*0.9 + 70 - r) * k
					gg += (mm*0.95 + 78 - gg) * k
					b += (mm + 95 - b) * k
				case veg && season == 2:
					t, v := autumn[(wx*7+wy*13)>>2&3], gg
					r, gg, b = v*t[0], v*t[1], v*t[2]
				case veg && (wx*31+wy*17)%11 == 0: // primavera: flores entre as folhas
					r, b, gg = gg*1.2+40, gg*0.9+40, gg*0.75
				}
			}
			if fl&2 != 0 {
				// Recorte: com o jogador atrás, a copa ou a muralha em volta dele
				// fica vazada no meio e pontilhada na borda.
				switch g.cutoutAt(wx, wy) {
				case 2:
					r, gg, b = r*0.24, gg*0.24, b*0.3
				case 1:
					r, gg, b = r*0.5, gg*0.5, b*0.56
				}
			}
			switch {
			case fl&128 != 0:
				// Lava: corre devagar, com crosta escura e veios que pulsam.
				v := math.Sin(float64(wx)*0.23+float64(wy)*0.41+T*0.9+float64(hash2(wx>>3, wy>>2, 5)&7)) +
					0.5*math.Sin(float64(wx)*0.11-T*0.5)
				switch {
				case v > 0.9:
					ch, r, gg, b = '~', 255, 210, 90
				case v > 0.1:
					ch, r, gg, b = '~', 240, 110+40*v, 30
				case v > -0.6:
					ch, r, gg, b = '=', 190, 60, 20
				default:
					ch, r, gg, b = '#', 90, 30, 18 // crosta
				}
				if hash2(wx, wy, int(T*3))&255 == 0 {
					ch, r, gg, b = '*', 255, 240, 160 // bolha estourando
				}
			case fl&32 != 0:
				// Capim em chamas: o caractere e a cor tremem a cada quadro.
				n := hash2(wx, wy, int(T*12))
				ch = "^*'\""[n%4]
				k := float64(n>>8&255) / 255
				r, gg, b = 255, 120+130*k, 30+90*k*k
			case fl&8 != 0:
				n := hash2(wx, wy, 11)
				v := math.Sin(float64(wx)*0.4 + float64(wy)*0.7 + T*0.8 + float64(n&7))
				ch = '.'
				if v > 0.5 {
					ch = '~'
				} else if v > -0.2 {
					ch = '-'
				}
				if fl&16 != 0 && int(n>>8%61) == int(T*2)%61 {
					ch, r, gg, b = 'o', 200, 255, 90
				}
			case fl&4 != 0:
				// Água escura com cristas finas e um brilho aqui e ali.
				v := math.Sin(float64(wx)*0.31 + float64(wy)*0.57 + T*1.6 + float64(hash2(wx>>2, wy>>1, 3)&7))
				ch = ' '
				if v > 0.7 {
					ch = '~'
				} else if v > 0.2 {
					ch = '-'
				}
				r *= 0.32 + 0.2*v
				gg *= 0.4 + 0.2*v
				b *= 0.55 + 0.2*v
				if sun > 0.4 && hash2(wx, wy, tw)&511 == 0 {
					ch, r, gg, b = '*', 220, 230, 255
				}
			case !under && fl&3 == 0 && ch != ' ' && gg > r*1.08 && gg > b*1.05:
				// Capim: lâminas que o vento inclina e que dobram com quem passa.
				// Cada lâmina tem a sua inclinação, altura e tom (base escura,
				// ponta clara); aqui e ali um pontinho de flor.
				if n := hash2(wx, wy, 21); n%5 < 4 {
					s := math.Sin(float64(wx)*0.07 + float64(wy)*0.025 - wind + float64(n>>4&3)*0.5)
					lean := s*0.75 + (float64(n>>6&7)/7-0.5)*0.5
					for k := 0; k < len(benders); k += 2 {
						dx, dy := float64(wx)-benders[k], float64(wy)-benders[k+1]
						if dx > -6 && dx < 6 && dy > -3 && dy < 2 {
							lean = 1
							if dx < 0 {
								lean = -1
							}
							gg *= 0.85
							break
						}
					}
					ch = bladeGlyph(lean, n&128 != 0)
					k := 0.78 + 0.5*float64(n>>9&15)/15
					r, gg, b = r*k, gg*k, b*k
					if n%41 == 0 {
						ch = byte(glyphFlower + int(n>>3%4))
						fc := []col{{240, 240, 230}, {255, 230, 120}, {255, 170, 210}, {200, 210, 255}}[n>>13%4]
						r, gg, b = fc[0]*amb[0], fc[1]*amb[1], fc[2]*amb[2]
					}
				}
			}
			if ch != ' ' {
				// Fachada, copa e telhado: luz que está atrás da base não
				// alcança a face da frente, e a da frente pega mais de leve
				// (pedra clara estourava em branco).
				tallBase, tallK := -1, 1.0
				if fl&2 != 0 {
					tallBase, tallK = g.baseAt(wx, wy), 0.5
				}
				for _, L := range rl {
					dx, dy := float64(wx)-L.x, float64(wy)-L.y
					d2 := dx*dx + dy*dy
					if d2 >= L.r2 || tallBase >= 0 && float64(tallBase) > L.base+16 {
						continue
					}
					k := 1 - math.Sqrt(d2)/L.r
					k *= k * L.k * tallK
					r = r*(1+L.mul[0]*k) + L.add[0]*k
					gg = gg*(1+L.mul[1]*k) + L.add[1]*k
					b = b*(1+L.mul[2]*k) + L.add[2]*k
				}
			} else if fl == 0 && sun < 0.3 {
				if n := hash2(wx, wy, 7); n%97 == 0 && (tw+int(n>>8))%8 < 5 {
					ch, r, gg, b = '*', 255, 244, 192
					if n>>16&3 != 0 {
						ch = '.'
					}
				}
			}
			buf[o], buf[o+1], buf[o+2], buf[o+3] = cb(r), cb(gg), cb(b), ch
		}
	})

	g.drawDecals(now)
	g.sealSparks()
	g.gateFx()

	// Sombras projetadas: copas, paredes, rochas e falésias escurecem o chão
	// do lado oposto ao sol. É o que dá altura (2.5D) às coisas. As duas
	// sombras (a curta, que procura o que é alto na direção do sol, e a longa,
	// do tamanho da altura, em castShadows) vão para uma só máscara, g.shadeF,
	// que fica com a mais escura de cada célula e é aplicada uma vez: antes
	// elas se somavam e escureciam o chão duas vezes.
	sdx, sdy := clamp((cur.Tod-0.5)*3, -1.4, 1.4), 0.85
	shadeK := 0.0
	if !under {
		shadeK = 0.6 * daylight(cur.Tod) * []float64{1, 0.6, 0.4}[clampi(cur.Wx, 0, 2)]
	}
	if len(g.shadeF) != cols*rows {
		g.shadeF = make([]float32, cols*rows)
	}
	for i := range g.shadeF {
		g.shadeF[i] = 1
	}
	if shadeK > 0.05 {
		tall := func(x, y int) bool {
			if x < 0 || y < 0 || x >= m.w || y >= m.h {
				return false
			}
			j := y*m.w + x
			f := m.flags[j]
			return f&2 != 0 || f&1 != 0 && f&4 == 0 && m.glyph[j] != ' '
		}
		parallelRows(rows, func(sy int) {
			wy := y0 + sy
			for sx := 0; sx < cols; sx++ {
				wx := x0 + sx
				if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h || m.flags[wy*m.w+wx]&7 != 0 {
					continue
				}
				for k := 1; k <= 6; k++ {
					if !tall(round(float64(wx)-sdx*float64(k)*1.5), round(float64(wy)-sdy*float64(k)*1.5)) {
						continue
					}
					i, f := sy*cols+sx, float32(1-shadeK*(1-float64(k)/9))
					g.shadeF[i] = min(g.shadeF[i], f)
					break
				}
			}
		})
	}
	// Volume do cenário: a luz vem do lado oposto ao da sombra; no subsolo,
	// de cima e da esquerda, mais fraca.
	if !g.opts.NoDepth {
		lx, ly, k, clouds := -sdx, -sdy, 0.55+0.45*sun, 0.0
		if under {
			lx, ly, k = -0.6, -0.8, 0.7
		} else if cur.Wx == 0 && sun > 0.3 {
			clouds = 0.14 * sun
		}
		haze := 0.45 + 0.5*(1-sun)
		if cur.Wx == 3 {
			haze = 1
		}
		g.depthPass(buf, cols, rows, x0, y0, lx, ly, k, clouds, haze, T)
	}
	if !under && !g.opts.NoDepth {
		g.castShadows(cols, rows, x0, y0, sdx*0.7, 0.35, shadeK*1.4)
	}
	g.applyShade(buf, cols, rows, x0, y0)
	tbuf := append([]byte(nil), buf...) // o terreno como ficou, para o reflexo saber o que é só água

	for i, l := range g.info.Lights {
		if i == 0 && cur.Event == 1 {
			continue
		}
		if l[3] < 2 {
			size := 1.0
			if l[3] == 1 {
				size = 0.4
			}
			g.drawFire(l[0], l[1], size, T)
		}
	}
	for i, c := range g.info.Chests {
		g.drawChest(c[0], c[1], i < len(cur.Chests) && cur.Chests[i], T)
	}
	if mo := cur.Moor; mo != nil { // o barco que você deixou atracado
		g.drawBoat(mo[0], mo[1])
		g.labels = append(g.labels, label{x: float64(mo[0]), y: float64(mo[1]) - 6, text: "Seu barco", color: col{150, 200, 255}})
	}
	for _, l := range cur.Loot {
		g.drawLoot(l, T)
	}
	g.drawShops(me, T)
	g.drawNPCs(me, T)
	g.bonfireLabels(me)
	for _, h := range cur.Herbs {
		g.drawHerb(h, T)
		if math.Hypot(h.X-mePos.x, h.Y-mePos.y) < 40 {
			g.labels = append(g.labels, label{x: h.X, y: h.Y - 6, text: g.info.Items[g.info.HerbK[clampi(h.Kind, 0, len(g.info.HerbK)-1)]].Name, color: col{184, 240, 160}})
		}
	}
	// Bolhas subindo do charco venenoso por perto.
	for k := 0; k < 3; k++ {
		bx, by := x0+rand.Intn(cols), y0+rand.Intn(rows)
		if bx >= 0 && by >= 0 && bx < m.w && by < m.h && m.flags[by*m.w+bx]&16 != 0 {
			g.spawn("bubble", float64(bx), float64(by), 0, -3, 1.2)
		}
	}

	boss := cur.Boss
	var bossPos *pos
	if boss.Alive {
		bp := pos{boss.X, boss.Y, 0, 0}
		if prev != nil && prev.Boss.Alive {
			bp = pos{prev.Boss.X + (boss.X-prev.Boss.X)*a, prev.Boss.Y + (boss.Y-prev.Boss.Y)*a, 0, 0}
		}
		bossPos = &bp
		g.drawDragon(round(bp.x)+18, round(bp.y)+18, T, true, 0)
	}
	for _, e := range cur.Fx {
		if e.Kind == proto.FxWarn || e.Kind == proto.FxHeal || e.Kind == proto.FxMeteor {
			g.drawFx(e, T)
		}
	}

	// Personagens de trás para a frente.
	for _, p := range g.info.Portals {
		g.labels = append(g.labels, label{x: float64(p.X), y: float64(p.Y) - 10, text: p.Label, color: col{185, 168, 255}})
		if p.Up {
			g.drawExit(p, cur.Tod, T)
		}
	}
	g.drawFishLine(me, T) // boia de pesca (pesca.go)
	g.drawGoal(T)         // destino escolhido no mapa
	var ents []ent
	for i := range cur.Mobs {
		e := &cur.Mobs[i]
		var p pos
		if prev != nil {
			o, ok := prev.mmap[e.ID]
			if ok {
				p = at(e.X, e.Y, o.X, o.Y, true)
			} else {
				p = pos{e.X, e.Y, 0, 0}
			}
		} else {
			p = pos{e.X, e.Y, 0, 0}
		}
		ents = append(ents, ent{key: fmt.Sprint("m", e.ID), pos: p, mob: e})
	}
	for i := range cur.Steeds {
		e := &cur.Steeds[i]
		p := pos{e.X, e.Y, 0, 0}
		if prev != nil {
			if o, ok := prev.smap[e.ID]; ok {
				p = at(e.X, e.Y, o.X, o.Y, true)
			}
		}
		ents = append(ents, ent{key: fmt.Sprint("s", e.ID), pos: p, steed: e})
	}
	for i := range cur.Players {
		e := &cur.Players[i]
		p := pos{float64(e.X), float64(e.Y), 0, 0}
		if e.ID == me.ID {
			p = mePos
		} else if prev != nil {
			if o, ok := prev.pmap[e.ID]; ok {
				p = at(float64(e.X), float64(e.Y), float64(o.X), float64(o.Y), true)
			}
		}
		ents = append(ents, ent{key: fmt.Sprint("p", e.ID), pos: p, player: e})
	}
	sort.SliceStable(ents, func(i, j int) bool { return ents[i].pos.y < ents[j].pos.y })
	// O mapa aparece inteiro; a linha de visão só esconde quem está atrás das
	// paredes quando está escuro (noite ou subsolo), ou no modo clássico.
	dark := under || sun < 0.35 || g.opts.FOVClassic
	seen := ents[:0]
	for _, e := range ents {
		if !dark || e.player != nil && e.player.ID == me.ID || g.entityVisible(e.pos.x, e.pos.y) {
			seen = append(seen, e)
		}
	}
	ents = seen

	// Anéis no chão: dourado no seu personagem, vermelho no inimigo sob o
	// mouse, como em Diablo e Age of Empires. Os corpos são desenhados depois,
	// por cima da metade de trás.
	if !g.showcase {
		rx := 6.5
		if me.Ride > 0 {
			rx = 10
		}
		g.groundRing(mePos.x, mePos.y+0.5, rx, rx*0.36, col{255, 214, 110}, 0.85)
		mwx, mwy := camL+g.mouseX*g.ui/cs, camT+g.mouseY*g.ui/cs
		best, bp := 1e9, pos{}
		for _, e := range ents {
			if e.mob == nil || e.mob.Age < 0.8 || e.mob.Ally {
				continue
			}
			if dx, dy := mwx-e.pos.x, mwy-(e.pos.y-12); math.Abs(dx) < 8 && math.Abs(dy) < 16 && math.Abs(dx)+math.Abs(dy) < best {
				best, bp = math.Abs(dx)+math.Abs(dy), e.pos
			}
		}
		if best < 1e9 {
			g.groundRing(bp.x, bp.y+0.5, 8, 2.9, col{240, 60, 40}, 0.8)
		}
	}
	talkers := map[string][2]float64{} // onde ancorar os balões de fala

	// Sombra de cada personagem e bicho no chão, puxada para o lado oposto ao sol.
	// Só o chão recebe: antes ela escurecia também parede, telhado e copa por
	// onde a elipse passava, e parecia uma mancha aplicada por cima de tudo.
	for _, e := range ents {
		big, k := 5, 0.45+0.25*shadeK
		if e.steed != nil || e.mob != nil && mobStyles[clampi(e.mob.Kind, 0, len(mobStyles)-1)].beast > 0 {
			big = 7
		}
		if under {
			k = 0.5
		}
		cx, cy := e.pos.x+sdx*3, e.pos.y+0.5
		for dy := -2; dy <= 2; dy++ {
			for dx := -big - 2; dx <= big+2; dx++ {
				d := math.Pow(float64(dx)/float64(big+2), 2) + math.Pow(float64(dy)/2.2, 2)
				if d > 1 {
					continue
				}
				sx, sy := round(cx)+dx-x0, round(cy)+dy-y0
				if sx < 0 || sy < 0 || sx >= cols || sy >= rows {
					continue
				}
				if wx, wy := sx+x0, sy+y0; wx < 0 || wy < 0 || wx >= m.w || wy >= m.h || m.flags[wy*m.w+wx]&(1|2|64) != 0 {
					continue
				}
				o, f := (sy*cols+sx)*4, 1-k*(1-d*0.6)
				buf[o], buf[o+1], buf[o+2] = cb(float64(buf[o])*f), cb(float64(buf[o+1])*f), cb(float64(buf[o+2])*f)
			}
		}
	}

	for _, e := range ents {
		moving := math.Abs(e.pos.vx)+math.Abs(e.pos.vy) > 0.05
		speed := math.Hypot(e.pos.vx, e.pos.vy)
		ph := g.walkPhase[e.key]
		if moving {
			ph += dt * (4 + math.Min(10, speed*5))
			g.walkPhase[e.key] = ph
		}
		var phase *float64
		if moving {
			phase = &ph
		}
		face := g.faces[e.key]
		if face == 0 {
			face = 1
		}
		hf := 0.35
		if math.Abs(e.pos.vx) >= math.Abs(e.pos.vy) {
			hf = 1
		}
		b := bodyDraw{x: round(e.pos.x), y: round(e.pos.y), T: T, phase: phase, hf: hf, face: face, tint: white}
		switch {
		case e.steed != nil:
			s := e.steed
			sk := g.info.SteedK[clampi(s.Kind, 0, len(g.info.SteedK)-1)]
			st := &steedStyles[clampi(s.Kind, 0, len(steedStyles)-1)]
			g.drawSteedAny(s.Kind, b, e.key, e.pos, dt)
			c := col{160, 138, 138}
			if me.Level >= sk.Level {
				c = col{216, 240, 192}
			}
			g.labels = append(g.labels, label{x: e.pos.x, y: e.pos.y - saddleH(st) - (st.br+4+st.neck)*st.sc(), text: fmt.Sprintf("%s · nível %d", sk.Name, sk.Level), color: c})
		case e.mob != nil:
			mb := e.mob
			st := &mobStyles[clampi(mb.Kind, 0, len(mobStyles)-1)]
			b.style = st
			tall := 30 * st.sc()
			if st.beast > 0 {
				tall = (st.bh + 2*st.br + 6) * st.sc()
			}
			if mb.Age < 0.8 {
				b.sink = int(math.Ceil(tall * (1 - mb.Age/0.8)))
			}
			if mb.Hit < 0.15 {
				b.flash, b.tint = 0.8, white
			} else if k := float64(mb.HP) / float64(max(1, mb.MaxHP)); k < 0.25 {
				b.flash, b.tint = 0.35, col{230, 40, 30}
			} else if k < 0.5 {
				b.flash, b.tint = 0.2, col{240, 210, 60}
			}
			if mb.Ally && mb.Hit >= 0.15 { // servo: brilho verde de alma penada
				b.flash, b.tint = 0.3+0.08*math.Sin(T*3+float64(mb.ID)), col{90, 230, 140}
			}
			if f, c := mobTint(mb.St, T); f > 0 && mb.Hit >= 0.15 {
				b.flash, b.tint = f, c // congelado, lento, queimando ou envenenado
				if mb.St&2 != 0 {
					b.phase = nil // congelado não mexe as pernas
				}
			}
			if st.embers { // bichos de fogo: brasas subindo e luz quente em volta
				r := 10 * st.sc()
				if rand.Float64() < 0.35*st.sc() {
					g.spawn("fire", e.pos.x+(rand.Float64()-0.5)*r, e.pos.y-tall*rand.Float64(), (rand.Float64()-0.5)*6, -12-rand.Float64()*10, 0.9)
				}
				g.light(e.pos.x, e.pos.y-tall/2, 16+10*st.sc(), 0.7+0.15*math.Sin(T*7+float64(mb.ID)), warm, col{90, 40, 10})
			}
			g.trackSteps(e.key, e.pos.x, e.pos.y, mb.Hit, st.bone)
			if mb.Atk >= 8.9 && !moving && mb.Age >= 0.8 {
				near := false
				for _, p := range cur.Players {
					near = near || math.Hypot(float64(p.X)-mb.X, float64(p.Y)-mb.Y) < 80
				}
				if !near { // dormindo: um z sobe e some sobre a cabeça
					zt := math.Mod(T*0.8+float64(mb.ID)*0.37, 1)
					z := "z"
					if zt > 0.5 {
						z = "Z"
					}
					g.labels = append(g.labels, label{x: e.pos.x + 3 + zt*3, y: e.pos.y - 34*st.sc() - zt*8, text: z, color: col{150, 180, 255}})
				}
			}
			if mb.Atk < 0.3 {
				// Golpeia na direção do jogador mais próximo.
				best, dir := 1e9, 0.0
				for _, p := range cur.Players {
					if d := math.Hypot(float64(p.X)-mb.X, float64(p.Y)-mb.Y); d < best {
						best, dir = d, math.Atan2(float64(p.Y)-mb.Y, float64(p.X)-mb.X)
					}
				}
				b.act = &action{"swing", mb.Atk / 0.3, dir}
				b.face = 1
				if math.Cos(dir) < 0 {
					b.face = -1
				}
			}
			if mb.Age < 0.8 && rand.Float64() < 0.5 {
				g.spawn("smoke", float64(b.x)+(rand.Float64()-0.5)*8, float64(b.y), (rand.Float64()-0.5)*10, -8, 0.5)
			}
			switch {
			case g.mode >= modeMix && st.beast > 0:
				g.drawBeast3D(b, g.yawFor(e.key, e.pos.vx, e.pos.vy, b.act, b.face, dt))
			case g.use3D:
				g.drawBody3D(b, g.yawFor(e.key, e.pos.vx, e.pos.vy, b.act, b.face, dt))
			case st.beast > 0:
				g.op(func() { g.drawBeast(b) })
			default:
				b.lit = g.mode == modeMix
				g.op(func() { g.drawBody(b) })
			}
			if mb.Age >= 0.8 {
				c := white
				if mb.Boss {
					c = col{224, 168, 255}
				}
				// Nível ao lado do nome (como em The Witcher 3): cinza bem abaixo,
				// branco perto, laranja acima, caveira vermelha quando pode matar
				// de um golpe.
				l := label{x: e.pos.x, y: e.pos.y - tall - 1, text: mb.Name, hp: mb.HP, mhp: mb.MaxHP, hasBar: true, color: c, lv: fmt.Sprint(mb.Lv), lvCol: col{230, 230, 230}}
				switch d := mb.Lv - me.Level; {
				case d >= 6:
					l.skull, l.color = true, col{255, 90, 70}
				case d >= 3:
					l.lvCol = col{255, 150, 60}
				case d <= -5:
					l.lvCol = col{130, 130, 140}
				}
				if mb.Ally { // servo de necromante: verde, sem nível
					l.color, l.lv, l.skull = col{140, 230, 150}, "", false
				}
				g.labels = append(g.labels, l)
			}
		default:
			p := e.player
			anim := anims[clampi(p.Class, 0, len(anims)-1)][clampi(p.Act, 0, 2)]
			dur := animDur[anim]
			b.style = g.playerStyle(p.Class, p.Color, p.Skin)
			if b.style.aura && rand.Float64() < 0.3 {
				g.spawn("holy", float64(b.x)+(rand.Float64()-0.5)*10, float64(b.y)-rand.Float64()*26, 0, -6, 0.6)
			}
			b.guard = p.Guard
			if p.Equip > 0 && p.Equip < len(g.info.Items) {
				c := hexc(g.info.Items[p.Equip].Color)
				b.wcol = &c
			}
			if p.ActAt < dur {
				b.act = &action{anim, p.ActAt / dur, p.Dir}
				if math.Abs(math.Cos(p.Dir)) > 0.25 {
					b.face = 1
					if math.Cos(p.Dir) < 0 {
						b.face = -1
					}
				}
			}
			if p.Hit < 0.2 {
				b.flash, b.tint = 0.6, red
			} else if p.ID == me.ID && cur.Sts[0] > 0 {
				b.flash, b.tint = 0.3+0.2*math.Sin(T*6), col{90, 230, 60} // envenenado: pulsa em verde
			}
			g.trackSteps(e.key, e.pos.x, e.pos.y, p.Hit, false)
			bx, by := float64(b.x), float64(b.y)
			reach := 1.0
			if g.regionName(p.X, p.Y) == "Cavernas de Cristal" {
				reach = 1.6 // os cristais devolvem a luz
			}
			switch p.Light { // lanterna (luz amarela firme) ou tocha (mais forte, tremendo)
			case 1:
				g.light(bx, by-8, 46*reach, 1.1*(0.85+0.15*math.Sin(T*9+float64(p.ID))), warm, warmAdd)
				if rand.Float64() < 0.4 {
					g.spawn("fire", bx+3, by-14, 0, -12, 0.4)
				}
			case 2:
				g.light(bx, by-8, 40*reach, 0.95, col{1.1, 0.95, 0.5}, col{70, 55, 15})
			}
			switch p.Class {
			case 1: // mago: luz azulada em volta
				g.light(bx, by-9, 28, 0.75, col{0.5, 0.6, 1.5}, col{30, 40, 120})
			case 6: // necromante: luz verde doentia
				g.light(bx, by-9, 24, 0.65, col{0.5, 1.2, 0.6}, col{20, 70, 40})
			case 3: // paladino: aura dourada
				g.light(bx, by-12, 26, 0.6, col{1.2, 1.0, 0.4}, col{90, 70, 10})
				if rand.Float64() < 0.15 {
					g.spawn("holy", bx+(rand.Float64()-0.5)*12, by-rand.Float64()*24, 0, -6, 0.7)
				}
			case 5: // ladino: no escuro some nas sombras
				if (sun < 0.3 || under) && b.flash == 0 {
					b.flash, b.tint = 0.55, col{8, 8, 14}
				}
			}
			// Montado, o bicho vai embaixo e o cavaleiro sobe até a sela; de
			// barco, as pernas somem dentro do casco.
			lift := 0
			if p.Ride > 0 {
				st := &steedStyles[clampi(p.Ride-1, 0, len(steedStyles)-1)]
				g.drawSteedAny(p.Ride-1, b, e.key+"m", e.pos, dt)
				// Sentado: o quadril na sela (não os pés), as pernas descem
				// pelos flancos do bicho.
				lift = round(saddleH(st) - 12*b.style.sc())
				b.y -= lift
				b.phase, b.seated = nil, true
			} else if p.Boat {
				b.sink = 4
			}
			if g.use3D {
				g.drawBody3D(b, g.yawFor(e.key, e.pos.vx, e.pos.vy, b.act, b.face, dt))
			} else {
				b.lit = g.mode == modeMix
				g.op(func() { g.drawBody(b) })
			}
			if p.Boat {
				g.op(func() { g.drawBoat(b.x, b.y) })
			}
			if p.Guard {
				g.op(func() { g.ring(float64(b.x), float64(b.y)-9, 6, T*3, T*3+4.5, col{170, 200, 255}) })
			}
			talkers[p.Name] = [2]float64{e.pos.x, e.pos.y - 34*charScale - float64(lift)}
			name := p.Name
			if p.Pz && p.ID != g.info.ID {
				name += " · pausado"
			}
			l := label{x: e.pos.x, y: e.pos.y - 31*charScale - float64(lift), text: name, color: white}
			if p.Boat {
				l.y += 7
			}
			if p.ID == me.ID {
				l.color = col{255, 224, 102}
			} else {
				l.hp, l.mhp, l.hasBar = p.HP, p.MaxHP, true
			}
			g.labels = append(g.labels, l)
		}
	}

	if g.showcase {
		g.drawShowcase(T, dt)
	}
	if bossPos != nil {
		flash := 0.0
		if boss.Hit < 0.12 {
			flash = 0.6
		}
		if g.mode >= modeMix {
			vx, vy := 0.0, 0.0
			if prev != nil && prev.Boss.Alive {
				vx, vy = boss.X-prev.Boss.X, boss.Y-prev.Boss.Y
			}
			g.drawDragon3D(round(bossPos.x), round(bossPos.y), g.yawFor("boss", vx, vy, nil, 1, dt), T, flash)
		} else {
			g.op(func() { g.drawDragon(round(bossPos.x), round(bossPos.y), T, false, flash) })
		}
		g.labels = append(g.labels, label{x: bossPos.x, y: bossPos.y - 36, text: "Tiamath", hp: boss.HP, mhp: boss.MaxHP, hasBar: true, color: white})
	}

	g.flush3d() // desenha em paralelo os modelos 3D e roda a fila na ordem

	// Reflexo: cada célula de água que ainda é só água copia, espelhada na
	// linha da margem, o que está logo acima dela, escurecido e azulado.
	if !under {
		for sx := 0; sx < cols; sx++ {
			wx := x0 + sx
			shore := -1
			for sy := 0; sy < rows; sy++ {
				wy := y0 + sy
				if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h {
					continue
				}
				f := m.flags[wy*m.w+wx]
				if f&4 == 0 || f&8 != 0 {
					shore = sy
					continue
				}
				o := (sy*cols + sx) * 4
				if shore < 0 || sy-shore > 46 || buf[o+3] != tbuf[o+3] || buf[o] != tbuf[o] {
					continue
				}
				ry, rx := 2*shore-sy+1, sx+round(math.Sin(float64(wy)*0.7+T*2.4)*0.8)
				if ry < 0 || rx < 0 || rx >= cols {
					continue
				}
				q := (ry*cols + rx) * 4
				if buf[q+3] == ' ' || buf[q+3] == 0 {
					continue
				}
				fade := 0.55 * (1 - float64(sy-shore)/50)
				buf[o] = cb(float64(buf[o])*(1-fade) + float64(buf[q])*0.45*fade)
				buf[o+1] = cb(float64(buf[o+1])*(1-fade) + float64(buf[q+1])*0.55*fade)
				buf[o+2] = cb(float64(buf[o+2])*(1-fade) + (float64(buf[q+2])*0.7+30)*fade)
				buf[o+3] = buf[q+3]
			}
		}
	}
	// Ondulação em volta dos barcos.
	for _, p := range cur.Players {
		if !p.Boat || prev == nil {
			continue
		}
		key := fmt.Sprint("r", p.ID)
		if o, ok := prev.pmap[p.ID]; ok && now-g.rippleAt[key] > 280 && (o.X != p.X || o.Y != p.Y) {
			g.ripples = append(g.ripples, ripple{float64(p.X), float64(p.Y) + 1, now, 9})
			g.rippleAt[key] = now
		}
	}
	g.drawRipples(now)

	// Projéteis: a bola de fogo deixa rastro de brasa e fumaça e ilumina o chão.
	for _, j := range cur.Projs {
		x, y, vx, vy := j.X, j.Y-9, 0.0, 0.0
		if prev != nil {
			if o, ok := prev.jmap[j.ID]; ok {
				x, y, vx, vy = o.X+(j.X-o.X)*a, o.Y+(j.Y-o.Y)*a-9, j.X-o.X, j.Y-o.Y
			}
		}
		switch j.Kind {
		case proto.ProjFireball:
			g.orb(x, y, 3.4, col{255, 250, 200}, col{255, 90, 24}, T)
			for i := 0; i < 4; i++ {
				g.spawn("fire", x+(rand.Float64()-0.5)*4, y+(rand.Float64()-0.5)*4, -vx*4+(rand.Float64()-0.5)*14, -vy*4+(rand.Float64()-0.5)*14, 0.25+rand.Float64()*0.35)
			}
			if rand.Float64() < 0.5 {
				g.spawn("smoke", x, y, -vx*2, -vy*2-4, 0.8)
			}
			g.light(x, y+14, 46, 1.1, warm, warmAdd)
		case proto.ProjArrow:
			ang := math.Atan2(vy, vx)
			g.streak(x-math.Cos(ang)*7, y-math.Sin(ang)*7, x, y, col{230, 210, 170})
			g.setCell(round(x), round(y), '*', 255, 255, 255)
		case proto.ProjIce: // lança de gelo: ponta clara e rastro de neve
			ang := math.Atan2(vy, vx)
			g.streak(x-math.Cos(ang)*6, y-math.Sin(ang)*6, x, y, col{190, 235, 255})
			g.orb(x, y, 1.8, col{255, 255, 255}, col{100, 170, 230}, T)
			if rand.Float64() < 0.7 {
				g.spawn("ice", x, y, -vx*3+(rand.Float64()-0.5)*8, -vy*3+(rand.Float64()-0.5)*8, 0.4)
			}
			g.light(x, y+14, 26, 0.8, cool, coolAdd)
		case proto.ProjHoly: // esfera de luz sagrada
			g.orb(x, y, 2.6, col{255, 255, 235}, col{230, 185, 70}, T)
			if rand.Float64() < 0.6 {
				g.spawn("holy", x, y, -vx*3, -vy*3, 0.4)
			}
			g.light(x, y+14, 30, 0.9, col{1.2, 1.0, 0.4}, col{90, 70, 20})
		case proto.ProjShadow:
			g.orb(x, y, 2.4, col{200, 255, 210}, col{90, 30, 130}, T)
			for i := 0; i < 2; i++ {
				g.spawn("shadow", x+(rand.Float64()-0.5)*2, y+(rand.Float64()-0.5)*2, -vx*3+(rand.Float64()-0.5)*8, -vy*3+(rand.Float64()-0.5)*8, 0.3+rand.Float64()*0.3)
			}
			g.light(x, y+14, 28, 0.8, col{0.6, 0.3, 1.2}, col{50, 10, 90})
		default:
			g.orb(x, y, 2, col{240, 240, 255}, col{110, 80, 230}, T)
			for i := 0; i < 2; i++ {
				g.spawn("arcane", x+(rand.Float64()-0.5)*2, y+(rand.Float64()-0.5)*2, -vx*3+(rand.Float64()-0.5)*8, -vy*3+(rand.Float64()-0.5)*8, 0.2+rand.Float64()*0.25)
			}
			g.light(x, y+14, 30, 0.9, col{0.4, 0.5, 1.4}, col{30, 40, 120})
		}
	}

	for _, e := range cur.Fx {
		if e.Kind != proto.FxWarn && e.Kind != proto.FxHeal && e.Kind != proto.FxMeteor {
			g.drawFx(e, T)
		}
	}
	g.drawParts(dt)
	if !under {
		g.drawWeather(me, T, now)
	}

	// Passe final: vinheta, céu fechado na chuva, clarão do raio e, no
	// subsolo, o escuro em que só se vê o que alguma luz alcança.
	flash := math.Max(0, 1-(now-g.flashAt)/160)
	if g.flashAt == 0 {
		flash = 0
	}
	wdim := 1.0
	if !under {
		wdim = []float64{1, 0.86, 0.7}[clampi(cur.Wx, 0, 2)]
	}
	vxs, vys := make([]float64, cols), make([]float64, rows)
	for sx := range vxs {
		vxs[sx] = math.Pow((float64(sx)/float64(cols)-0.5)*2, 2)
	}
	for sy := range vys {
		vys[sy] = math.Pow((float64(sy)/float64(rows)-0.45)*2, 2)
	}
	fg := newFogGrid(x0, y0, cols, rows, T)
	// Sombra atrás das paredes: some de dia e pesa à noite e no subsolo.
	shadeOut := 1 - 0.45*(1-clamp(sun/0.6, 0, 1))
	if under {
		shadeOut = 0.5
	}
	parallelRows(rows, func(sy int) {
		wy := float64(y0 + sy)
		for sx := 0; sx < cols; sx++ {
			o := (sy*cols + sx) * 4
			f := math.Max(0.5, 1-0.3*(vxs[sx]+vys[sy])) * wdim
			if under {
				wx := float64(x0 + sx)
				vis := 0.035
				for _, L := range lit {
					dx, dy := wx-L.x, wy-L.y
					d2 := dx*dx + dy*dy
					if d2 >= L.r2 {
						continue
					}
					if k := 1 - math.Sqrt(d2)/L.r; k > vis {
						vis = k
					}
				}
				if ix, iy := int(wx), int(wy); ix < 0 || iy < 0 || ix >= m.w || iy >= m.h || m.flags[iy*m.w+ix]&64 == 0 {
					f *= math.Min(1, vis*1.6)
				}
			}
			bb := 1.0
			if flash > 0 {
				f, bb = f*(1+1.6*flash)+0.5*flash, 1.1
			}
			// Campo de visão: no modo clássico o que não está à vista vira
			// memória (cinza escuro) ou escuro total, se nunca foi visto. No
			// normal o mapa fica inteiro e só a sombra atrás das paredes
			// escurece, mais forte quanto menos luz houver.
			if wx, wyI := x0+sx, y0+sy; !g.visible(wx, wyI) {
				if !g.opts.FOVClassic {
					if wx < 0 || wyI < 0 || wx >= m.w || wyI >= m.h || m.flags[wyI*m.w+wx]&64 == 0 {
						f *= shadeOut
					}
				} else {
					if wx >= 0 && wyI >= 0 && wx < m.w && wyI < m.h && g.fov.mem[wyI*m.w+wx] {
						l := (0.3*float64(buf[o]) + 0.59*float64(buf[o+1]) + 0.11*float64(buf[o+2])) * 0.28
						buf[o], buf[o+1], buf[o+2] = cb(l*0.9), cb(l*0.95), cb(l*1.15)
					} else {
						buf[o], buf[o+1], buf[o+2], buf[o+3] = 0, 0, 0, ' '
					}
					continue
				}
			}
			// Clima sombrio: cores lavadas (puxadas para o cinza) e uma névoa
			// fria que corre devagar pelo chão, mais densa longe do centro.
			r, gg, b := float64(buf[o]), float64(buf[o+1]), float64(buf[o+2])
			lum := 0.3*r + 0.59*gg + 0.11*b
			r, gg, b = r+(lum-r)*0.18, gg+(lum-gg)*0.18, b+(lum-b)*0.14
			if !under {
				fog := fg.at(sx, sy) * (0.35 + 0.65*(vxs[sx]+vys[sy])*0.5)
				if cur.Wx == 3 {
					fog = math.Min(0.75, fog*2.5+0.25) // neblina fechada
				}
				r, gg, b = r+(38-r)*fog, gg+(42-gg)*fog, b+(52-b)*fog
			}
			buf[o], buf[o+1], buf[o+2] = cb(r*f), cb(gg*f), cb(b*f*bb)
		}
	})
	if under {
		n := 0
		for _, l := range g.labels {
			if math.Hypot(l.x-mePos.x, l.y-mePos.y) < 60 {
				g.labels[n] = l
				n++
			}
		}
		g.labels = g.labels[:n]
	}

	n := 0
	for _, l := range g.labels {
		lx, ly := int(l.x), int(l.y)+14
		if !dark || g.visible(lx, ly) || g.visible(lx, ly+16) || g.opts.FOVClassic && lx >= 0 && ly >= 0 && lx < m.w && ly < m.h && g.fov.mem[ly*m.w+lx] && !l.hasBar {
			g.labels[n] = l
			n++
		}
	}
	g.labels = g.labels[:n]
	g.talkers = talkers

	// O trovão sacode a câmera por um instante.
	g.shake = math.Max(0, g.shake-dt*3)
	offX := math.Round((camL-float64(x0))*cs + (rand.Float64()-0.5)*g.shake*cs*1.5)
	offY := math.Round((camT-float64(y0))*cs + (rand.Float64()-0.5)*g.shake*cs*1.5)
	bloom := 0.0
	if g.opts.Bloom {
		bloom = 0.55
	}
	smooth := 0.0
	if !g.opts.NoSmoothLight {
		smooth = 0.85
	}
	g.g.drawS(screen, buf, cols, rows, offX, offY, g.cs, bloom, smooth)

	g.drawLabels(screen, camL, camT)
	g.drawBubbles(screen, camL, camT, g.talkers)
	g.drawFloats(screen, camL, camT)
	g.drawHUD(screen, me)
	g.drawOverlays(screen, W/g.ui, H/g.ui)
	g.drawOpening(screen, W/g.ui, H/g.ui)
	g.drawDeepWait(screen, W/g.ui, H/g.ui)
	g.drawPause(screen, W/g.ui, H/g.ui) // menu de pausa e aviso de "salvo", por cima de tudo
}

// rowLit separa as luzes que alcançam a linha wy: cada célula testa só elas.
func rowLit(lit []lightT, wy float64) []lightT {
	var out []lightT
	for _, L := range lit {
		if math.Abs(wy-L.y) < L.r {
			out = append(out, L)
		}
	}
	return out
}

// fogGrid calcula a névoa numa grade grossa (a cada 3 células); o ruído é
// suave, então a interpolação não se distingue do cálculo célula a célula.
type fogGrid struct {
	w    int
	vals []float64
}

func newFogGrid(x0, y0, cols, rows int, T float64) *fogGrid {
	w, h := cols/3+2, rows/3+2
	f := &fogGrid{w: w, vals: make([]float64, w*h)}
	parallelRows(h, func(gy int) {
		for gx := 0; gx < w; gx++ {
			f.vals[gy*w+gx] = fogAt(x0+gx*3, y0+gy*3, T)
		}
	})
	return f
}

func (f *fogGrid) at(sx, sy int) float64 {
	gx, gy := sx/3, sy/3
	tx, ty := float64(sx%3)/3, float64(sy%3)/3
	a, b := f.vals[gy*f.w+gx], f.vals[gy*f.w+gx+1]
	c, d := f.vals[(gy+1)*f.w+gx], f.vals[(gy+1)*f.w+gx+1]
	return (a+(b-a)*tx)*(1-ty) + (c+(d-c)*tx)*ty
}

// newGridOf calcula fn numa grade grossa (a cada 3 células), como a névoa.
func newGridOf(x0, y0, cols, rows int, fn func(wx, wy int) float64) *fogGrid {
	w, h := cols/3+2, rows/3+2
	f := &fogGrid{w: w, vals: make([]float64, w*h)}
	parallelRows(h, func(gy int) {
		for gx := 0; gx < w; gx++ {
			f.vals[gy*w+gx] = fn(x0+gx*3, y0+gy*3)
		}
	})
	return f
}
