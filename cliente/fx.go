package main

import (
	"fmt"
	"math"
	"math/rand"

	"asciimmo/proto"
)

// ---------- grade do quadro ----------

func (g *Game) setCell(wx, wy int, code byte, r, gr, b float64) {
	sx, sy := wx-g.x0, wy-g.y0
	if sx < 0 || sy < 0 || sx >= g.cols || sy >= g.rows {
		return
	}
	o := (sy*g.cols + sx) * 4
	g.buf[o], g.buf[o+1], g.buf[o+2], g.buf[o+3] = cb(r), cb(gr), cb(b), code
}

func (g *Game) setC(wx, wy int, code byte, c col) { g.setCell(wx, wy, code, c[0], c[1], c[2]) }

func cb(v float64) byte {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return byte(v)
}

// ring desenha um trecho de círculo com o caractere acompanhando a curva.
func (g *Game) ring(cx, cy, r, a0, a1 float64, c col) {
	n := max(12, int(r*math.Abs(a1-a0)*1.5))
	for i := 0; i <= n; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		g.setC(round(cx+r*math.Cos(a)), round(cy+r*math.Sin(a)), slope(a+math.Pi/2), c)
	}
}

func (g *Game) streak(ax, ay, bx, by float64, c col) {
	n := max(1, int(math.Hypot(bx-ax, by-ay)))
	code := slope(math.Atan2(by-ay, bx-ax))
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		g.setC(round(ax+(bx-ax)*t), round(ay+(by-ay)*t), code, c)
	}
}

// ---------- partículas ----------

type pkind struct {
	cols  []col
	grav  float64
	chars string
}

var pkinds = map[string]*pkind{
	"fire":   {[]col{{255, 250, 200}, {255, 190, 70}, {255, 90, 24}, {120, 30, 16}}, -14, "@#*+:."},
	"arcane": {[]col{{240, 240, 255}, {150, 170, 255}, {110, 80, 230}, {50, 30, 110}}, 0, "@*+:."},
	"heal":   {[]col{{230, 255, 220}, {120, 240, 140}, {50, 160, 90}}, -18, "+++:."},
	"shadow": {[]col{{210, 255, 220}, {150, 90, 220}, {70, 30, 110}, {30, 14, 50}}, -4, "@*+:."},
	"snow":   {[]col{{255, 255, 255}, {225, 235, 255}, {170, 190, 225}}, 3, "*.."},
	"leaf":   {[]col{{255, 180, 70}, {225, 115, 45}, {130, 65, 30}}, 8, "%*,"},
	"petal":  {[]col{{255, 215, 235}, {255, 150, 200}, {190, 100, 150}}, 5, "*,."},
	"holy":   {[]col{{255, 255, 230}, {255, 226, 130}, {200, 150, 60}}, -10, "#*+:."},
	"bone":   {[]col{{240, 210, 170}, {200, 150, 110}, {110, 80, 60}}, 60, "%*o+."},
	"spark":  {[]col{{255, 255, 255}, {200, 210, 240}, {110, 120, 160}}, 20, "*+:."},
	"smoke":  {[]col{{110, 100, 110}, {70, 66, 80}, {40, 38, 50}}, -9, "o:.."},
	"foam":   {[]col{{230, 240, 255}, {150, 180, 220}, {60, 90, 140}}, 0, "~-."},
	"bubble": {[]col{{220, 255, 120}, {150, 210, 60}, {70, 110, 30}}, -6, "oo."},
	"blood":  {[]col{{220, 40, 40}, {150, 18, 18}, {70, 8, 8}}, 50, "*:.."},
}

type particle struct {
	k                     *pkind
	x, y, vx, vy, life, m float64
}

func (g *Game) spawn(kind string, x, y, vx, vy, life float64) {
	if k := pkinds[kind]; k != nil && len(g.parts) < 2500 {
		g.parts = append(g.parts, particle{k, x, y, vx, vy, life, life})
	}
}

func (g *Game) burst(kind string, x, y float64, n int, speed, life float64) {
	for i := 0; i < n; i++ {
		a, v := rand.Float64()*tau, speed*(0.25+0.75*rand.Float64())
		g.spawn(kind, x, y, math.Cos(a)*v, math.Sin(a)*v, life*(0.5+0.5*rand.Float64()))
	}
}

func (g *Game) drawParts(dt float64) {
	n := 0
	for _, p := range g.parts {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vy += p.k.grav * dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		t := 1 - p.life/p.m
		f := t * float64(len(p.k.cols)-1)
		i := min(len(p.k.cols)-2, int(f))
		c := mix3(p.k.cols[i], p.k.cols[i+1], f-float64(i))
		g.setC(round(p.x), round(p.y), p.k.chars[min(len(p.k.chars)-1, int(t*float64(len(p.k.chars))))], c)
		g.parts[n] = p
		n++
	}
	g.parts = g.parts[:n]
}

// ---------- luzes e nomes ----------

type lightT struct {
	x, y, r, r2, k float64
	mul, add       col
	base           float64 // linha do chão da luz: quem está atrás de uma fachada não a ilumina
}

var (
	warm, warmAdd   = col{1.2, 0.8, 0.2}, col{90, 40, 0}
	cool, coolAdd   = col{0.7, 0.9, 1.1}, col{40, 60, 80}
	spore, sporeAdd = col{0.4, 1.3, 0.5}, col{20, 70, 20}
	torch, torchAdd = col{1.0, 1.2, 0.5}, col{60, 70, 10}
)

func (g *Game) light(x, y, r, k float64, mul, add col) {
	g.lights = append(g.lights, lightT{x, y, r, r * r, k, mul, add, y + 14})
}

type label struct {
	x, y    float64
	text    string
	color   col
	hp, mhp int
	hasBar  bool
	lv      string // nível do inimigo, ao lado do nome
	lvCol   col
	skull   bool // muito acima do jogador: caveira no lugar do nível
}

// ---------- efeitos ----------

// fxStart roda uma vez quando um efeito aparece.
func (g *Game) fxStart(e proto.Fx) {
	if g.fxStart2(e) {
		return
	}
	cy := e.Y - 14
	switch e.Kind {
	case proto.FxBlast:
		g.burst("fire", e.X, e.Y, 90, 60, 0.7)
	case proto.FxPuff:
		g.burst("bone", e.X, cy, 40, 34, 0.9)
	case proto.FxThrust:
		for d := 0.0; d < e.R; d += 1.5 {
			g.spawn("spark", e.X+math.Cos(e.Dir)*d, cy+math.Sin(e.Dir)*d, (rand.Float64()-0.5)*16, (rand.Float64()-0.5)*16, 0.3+rand.Float64()*0.3)
		}
	case proto.FxSpin:
		g.burst("spark", e.X, cy, 50, 70, 0.4)
	case proto.FxBoom:
		kinds := []string{"arcane", "fire", "spark", "shadow", "ice", "holy"}
		k := "fire"
		if d := int(e.Dir); d >= 0 && d < len(kinds) {
			k = kinds[d]
		}
		g.burst(k, e.X, cy, int(e.R*9), e.R*4.5, 0.75)
		if int(e.Dir) == 1 {
			g.burst("smoke", e.X, cy, int(e.R*3), e.R*2, 1.3)
		}
	case proto.FxHeal:
		for i := 0; i < 70; i++ {
			a, d := rand.Float64()*tau, math.Sqrt(rand.Float64())*e.R
			g.spawn("heal", e.X+math.Cos(a)*d, e.Y-rand.Float64()*20+math.Sin(a)*d*0.6, 0, -6-rand.Float64()*10, 0.5+rand.Float64()*0.7)
		}
	case proto.FxHoly:
		g.burst("holy", e.X, cy, 130, 75, 0.7)
	case proto.FxDark:
		g.burst("shadow", e.X, cy, 130, 75, 0.7)
	case proto.FxBolt: // raio: clarão na tela toda, tremor e faíscas no chão
		g.flashAt = g.now()
		g.shake = 1
		g.burst("spark", e.X, e.Y, 60, 50, 0.5)
		g.burst("smoke", e.X, e.Y, 20, 12, 1.2)
	}
}

func (g *Game) drawFire(fx, fy int, size, T float64) {
	step := int(T * 12)
	h, w := round(13*size), 5*size
	for dy := 0; dy < h; dy++ {
		half := w * math.Pow(1-float64(dy)/float64(h), 0.7)
		for dx := -int(math.Ceil(w)); dx <= int(math.Ceil(w)); dx++ {
			n := hash2(dx+fx, dy, step)
			if math.Abs(float64(dx)) > half*(0.55+0.45*float64(n&255)/255) {
				continue
			}
			k := clamp(float64(dy)/float64(h)+math.Abs(float64(dx))/(w*1.8)+(float64(n>>8&255)/255-0.5)*0.3, 0, 1)
			g.setC(fx+dx, fy-dy, "&W)(^*"[n>>16%6], mix3(col{255, 236, 150}, col{255, 80, 24}, k))
		}
	}
	if size >= 1 {
		for dx := -5; dx <= 5; dx++ {
			g.setCell(fx+dx, fy+1, '=', 120, 80, 44)
		}
	} else {
		g.setCell(fx, fy+1, '|', 150, 110, 70)
		g.setCell(fx, fy+2, '|', 120, 86, 54)
	}
}

// drawChest desenha um baú: fechado brilha, aberto fica com a tampa erguida.
func (g *Game) drawChest(cx, cy int, opened bool, T float64) {
	for dy := -5; dy <= 0; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if opened && dy < -2 {
				if dy == -5 {
					g.setCell(cx+dx, cy+dy-1, '=', 120, 84, 44)
				}
				continue
			}
			band := dx == 0 || dy == -2
			k := 1.05 - 0.07*float64(dx+4) + 0.05*float64(dy)
			c := col{150, 98, 50}
			code := rampCode(0.6 + 0.3*k)
			if band {
				c, code = col{240, 200, 90}, '#'
			}
			g.setCell(cx+dx, cy+dy, code, c[0]*k, c[1]*k, c[2]*k)
		}
	}
	text, color := "Baú", col{255, 208, 112}
	if !opened {
		g.light(float64(cx), float64(cy), 24, 0.5+0.2*math.Sin(T*3), warm, col{80, 60, 10})
		if rand.Float64() < 0.1 {
			g.spawn("holy", float64(cx)+(rand.Float64()-0.5)*8, float64(cy)-5, 0, -8, 0.6)
		}
	} else {
		text, color = "Baú (aberto)", col{106, 106, 118}
	}
	g.labels = append(g.labels, label{x: float64(cx), y: float64(cy) - 8, text: text, color: color})
}

// drawLoot desenha um item caído: um saquinho na cor do item, balançando.
func (g *Game) drawLoot(l proto.Loot, T float64) {
	it := g.info.Items[l.Item]
	if it.Kind == proto.KindMoney { // dinheiro: um montinho que brilha de leve
		for dy := -2; dy <= 0; dy++ {
			for dx := -3 - dy; dx <= 3+dy; dx++ {
				n := hash2(dx, dy, l.ID)
				k := 0.7 + 0.4*float64(n&255)/255
				code := "$$o"[n>>8%3]
				if (n>>12)%9 == uint32(int(T*4)%9) {
					code, k = '*', 1.3
				}
				g.setCell(l.X+dx, l.Y+dy, code, 200*k, 150*k, 70*k)
			}
		}
		g.labels = append(g.labels, label{x: float64(l.X), y: float64(l.Y) - 4, text: fmt.Sprintf("%d %s", l.N, g.info.Currency), color: col{230, 190, 110}})
		return
	}
	c := hexc(it.Color)
	by := l.Y
	if math.Sin(T*4+float64(l.ID)) > 0 {
		by--
	}
	// O chão fala por símbolos: ! poção, ) arma, [ armadura, " amuleto,
	// * material, % erva — um bloquinho do símbolo, claro no meio.
	glyph := lootGlyph(it.Kind)
	for dy := -2; dy <= 0; dy++ {
		for dx := -1; dx <= 1; dx++ {
			k := 0.75
			if dx == 0 && dy == -1 {
				k = 1.25
			}
			g.setCell(l.X+dx, by+dy, glyph, c[0]*k, c[1]*k, c[2]*k)
		}
	}
	if it.Kind != proto.KindPotion {
		g.light(float64(l.X), float64(l.Y), 26, 0.7+0.2*math.Sin(T*5), cool, col{c[0] * 0.4, c[1] * 0.4, c[2] * 0.4})
		if rand.Float64() < 0.3 {
			g.spawn("spark", float64(l.X)+(rand.Float64()-0.5)*6, float64(by)-2, (rand.Float64()-0.5)*6, -8-rand.Float64()*8, 0.6)
		}
	}
	g.labels = append(g.labels, label{x: float64(l.X), y: float64(by) - 5, text: it.Name, color: mix3(c, white, 0.3)})
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// orb é uma bola de energia ou fogo, fervendo por dentro.
func (g *Game) orb(x, y, r float64, hot, cold col, T float64) {
	step := int(T * 20)
	R := int(math.Ceil(r))
	for dy := -R; dy <= R; dy++ {
		for dx := -R; dx <= R; dx++ {
			d, n := math.Hypot(float64(dx), float64(dy))/r, float64(hash2(dx, dy, step)&255)/255
			if d > 0.75+n*0.3 {
				continue
			}
			c := mix3(hot, cold, clamp(d*1.1+(n-0.5)*0.4, 0, 1))
			g.setC(round(x)+dx, round(y)+dy, "@#&%*"[clampi(int(d*5), 0, 4)], c)
		}
	}
}

// dragonClass é a mesma silhueta do servidor e do cliente web.
func dragonClass(px, py, t float64) int {
	ax := math.Abs(px)
	// Cinco cabeças em leque, nos pescoços longos que ondulam.
	hxs := [5]float64{-9, -4.6, 0, 4.6, 9}
	hys := [5]float64{-8, -11.5, -13, -11.5, -8}
	for k := 0; k < 5; k++ {
		sw := math.Sin(t*1.1+float64(k)*1.3) * 0.9
		hx, hy := hxs[k]+sw, hys[k]+math.Cos(t*0.9+float64(k))*0.5
		if math.Abs(math.Abs(px-hx)-0.8) < 0.45 && math.Abs(py-hy+0.3) < 0.45 {
			return 4 // olho
		}
		if math.Hypot((px-hx)/1.8, (py-hy)/1.4) < 1 {
			return 3
		}
		if segDist2(px, py, hxs[k]*0.18, -4.5, hx, hy+1) < 0.85 {
			return 1
		}
	}
	if ax < 0.6 && py > -6 && py < 6 {
		return 2
	}
	if math.Hypot(px/4.6, py/7.4) < 1 {
		return 1
	}
	// Duas caudas, serpenteando em oposição.
	if py >= 6 && py <= 20 {
		k := (py - 6) / 14
		for _, side := range []float64{-1, 1} {
			if math.Abs(px-side*(1.6+2.5*k)-3*math.Sin(k*4+t*2+side)*k) < 1.8*(1-k)+0.4 {
				return 1
			}
		}
	}
	flap := math.Sin(t * 3)
	span := 18 + 2.5*flap
	if ax >= 3 && ax <= 3+span {
		s := (ax - 3) / span
		top := -6 - 5*s + 2*flap*s
		bottom := 6 - 15*s - 1.2*math.Abs(math.Sin(s*math.Pi*3)) + 2*flap*s
		if py > top && py < bottom {
			if py < top+1 || math.Abs(math.Mod(s*3, 1)-0.5) > 0.43 {
				return 6
			}
			return 5
		}
	}
	return 0
}

var dragonColors = []col{{}, {176, 132, 64}, {226, 196, 130}, {200, 160, 90}, {255, 224, 96}, {120, 82, 40}, {214, 176, 112}}

func (g *Game) drawDragon(cx, cy int, t float64, shadow bool, flash float64) {
	for dy := -34; dy <= 44; dy++ {
		sy := cy + dy - g.y0
		if sy < 0 || sy >= g.rows {
			continue
		}
		for dx := -50; dx <= 50; dx++ {
			sx := cx + dx - g.x0
			if sx < 0 || sx >= g.cols {
				continue
			}
			px, py := float64(dx)/2, float64(dy)/2
			cls := dragonClass(px, py, t)
			if cls == 0 {
				continue
			}
			o := (sy*g.cols + sx) * 4
			if shadow {
				g.buf[o], g.buf[o+1], g.buf[o+2] = byte(float64(g.buf[o])*0.25), byte(float64(g.buf[o+1])*0.25), byte(float64(g.buf[o+2])*0.3)
				continue
			}
			n := hash2(dx, dy, 99)
			l := clamp(0.72-px*0.022-py*0.012+(float64(n&255)/255-0.5)*0.3, 0.28, 1)
			if cls == 6 {
				l = math.Min(1, l+0.2)
			}
			c := dragonColors[cls]
			g.buf[o] = cb(c[0]*l + (255-c[0]*l)*flash)
			g.buf[o+1] = cb(c[1]*l + (255-c[1]*l)*flash)
			g.buf[o+2] = cb(c[2]*l + (255-c[2]*l)*flash)
			switch cls {
			case 4:
				g.buf[o+3] = 'O'
			case 2, 6:
				g.buf[o+3] = '#'
			default:
				g.buf[o+3] = rampCode(l)
			}
		}
	}
}

// drawFx desenha os efeitos de área que o servidor mantém vivos.
func (g *Game) drawFx(e proto.Fx, T float64) {
	if g.drawFx2(e, T) {
		return
	}
	cy, p := e.Y-14, e.P
	fade := 1 - p
	switch e.Kind {
	case proto.FxRain: // chuva de flechas
		g.ring(e.X, e.Y, e.R, 0, tau, col{230, 210, 170})
		for i := 0; i < 12; i++ {
			n := hash2(i, int(T*30), 5)
			ax := e.X + float64(int(n%41)-20)*e.R/20
			ay := e.Y + float64(int(n>>8%41)-20)*e.R/24 - 60*fade
			g.streak(ax+6, ay-12, ax, ay, col{230, 210, 170})
		}
	case proto.FxBolt:
		g.drawBolt(e, T)
	case proto.FxDark:
		r := e.R * ease(p)
		g.ring(e.X, cy, r, 0, tau, mix3(col{200, 255, 210}, col{40, 10, 60}, p))
		g.ring(e.X, cy, r*0.7, 0, tau, mix3(col{150, 90, 220}, col{30, 10, 50}, p))
		g.light(e.X, e.Y, e.R*3, fade*1.3, col{0.6, 0.3, 1.2}, col{50, 10, 90})
	case proto.FxSlash:
		g.ring(e.X, cy, e.R+4, e.Dir-1.1, e.Dir+1.1, mix3(col{200, 210, 240}, col{40, 40, 60}, p))
	case proto.FxWarn, proto.FxMeteor:
		// Aviso legível do golpe em área: o chão atingido pisca em vermelho com "!".
		if int(T*8)%2 == 0 {
			R := int(e.R)
			for dy := -R; dy <= R; dy++ {
				for dx := -R; dx <= R; dx++ {
					if dx*dx+dy*dy > R*R || hash2(dx, dy, 77)%3 != 0 {
						continue
					}
					g.setCell(int(e.X)+dx, int(e.Y)+dy, '!', 200, 36, 28)
				}
			}
		}
		g.ring(e.X, e.Y, e.R, 0, tau, col{255, 90, 64})
		g.ring(e.X, e.Y, e.R*p, 0, tau, col{153, 32, 32})
		if e.Kind == proto.FxMeteor { // o meteoro vem caindo enquanto o aviso fecha
			mx, my := e.X+34*fade, e.Y-110*fade
			g.orb(mx, my, 5, col{255, 250, 200}, col{255, 90, 24}, T)
			for i := 0; i < 4; i++ {
				g.spawn("fire", mx+(rand.Float64()-0.5)*6, my-3, 20+rand.Float64()*20, -70-rand.Float64()*30, 0.5)
			}
			g.light(mx, my+20, 60, 0.9, warm, warmAdd)
		}
	case proto.FxBlast, proto.FxBoom:
		fire := e.Kind == proto.FxBlast || int(e.Dir) == 1
		dark := !fire && int(e.Dir) == 3
		r := e.R * (0.3 + 0.7*ease(math.Min(1, p*1.6)))
		hot, cold := col{240, 240, 255}, col{110, 80, 230}
		switch {
		case fire:
			hot, cold = col{255, 250, 200}, col{255, 90, 24}
		case dark:
			hot, cold = col{200, 255, 210}, col{90, 30, 130}
		case int(e.Dir) == 4: // gelo
			hot, cold = col{255, 255, 255}, col{80, 160, 230}
		case int(e.Dir) == 5: // luz
			hot, cold = col{255, 255, 230}, col{230, 180, 70}
		}
		yy := cy
		if e.Kind == proto.FxBlast {
			yy = e.Y
		}
		if p < 0.55 {
			g.orb(e.X, yy, r*(1-p*0.8), hot, cold, T)
		}
		g.ring(e.X, yy, r, 0, tau, mix3(hot, col{60, 20, 20}, p))
		g.ring(e.X, yy, math.Max(0, r-1.5), 0, tau, mix3(cold, col{30, 10, 10}, p))
		if fire {
			g.light(e.X, e.Y, e.R*4+20, fade*1.6, warm, col{120, 60, 0})
		} else {
			g.light(e.X, e.Y, e.R*4+20, fade*1.6, col{0.4, 0.5, 1.4}, col{30, 40, 120})
		}
	case proto.FxThrust:
		ex, ey := e.X+math.Cos(e.Dir)*e.R, cy+math.Sin(e.Dir)*e.R
		c := mix3(col{255, 255, 255}, col{50, 55, 90}, p)
		for _, o := range []float64{-2, 0, 2} {
			cc := c
			if o != 0 {
				cc = mix3(c, col{20, 20, 40}, 0.5)
			}
			g.streak(e.X-math.Sin(e.Dir)*o, cy+math.Cos(e.Dir)*o, ex-math.Sin(e.Dir)*o, ey+math.Cos(e.Dir)*o, cc)
		}
	case proto.FxSpin:
		for k := 0; k < 4; k++ {
			a := p*2*tau + float64(k)*math.Pi/2
			g.ring(e.X, cy, e.R*(0.55+0.45*math.Min(1, p*2))-float64(k%2*2), a, a+1.3, mix3(col{255, 255, 255}, col{90, 100, 150}, p))
		}
	case proto.FxHeal:
		g.ring(e.X, e.Y, e.R*ease(p), 0, tau, mix3(col{170, 255, 180}, col{20, 60, 30}, p))
		g.light(e.X, e.Y, e.R*1.6, fade*0.8, col{0.3, 1.1, 0.4}, col{10, 70, 20})
	case proto.FxHoly:
		r := e.R * ease(p)
		g.ring(e.X, cy, r, 0, tau, mix3(col{255, 250, 210}, col{90, 60, 20}, p))
		g.ring(e.X, cy, r*0.6, 0, tau, mix3(col{255, 226, 130}, col{60, 40, 10}, p))
		for k := 0; k < 8; k++ {
			a := float64(k)*math.Pi/4 + 0.3
			g.streak(e.X+math.Cos(a)*r*0.3, cy+math.Sin(a)*r*0.3, e.X+math.Cos(a)*r, cy+math.Sin(a)*r, mix3(col{255, 240, 170}, col{70, 50, 20}, p))
		}
		g.light(e.X, e.Y, e.R*3, fade*1.5, col{1.2, 1.0, 0.4}, col{110, 90, 20})
	}
}

// drawBolt desenha o raio descendo em zigue-zague do alto da tela até o
// ponto atingido, com galhos.
// Raio da magia (golden = true): dourado, como os milagres de luz; o da
// tempestade fica branco-azulado.
var (
	boltCore   = [2]col{{235, 240, 255}, {255, 236, 150}}
	boltBranch = [2]col{{150, 170, 255}, {255, 190, 70}}
	boltLight  = [2][2]col{{{1.2, 1.2, 1.6}, {140, 140, 190}}, {{1.5, 1.25, 0.6}, {170, 130, 40}}}
)

func (g *Game) drawBolt(e proto.Fx, T float64) {
	g.drawBoltC(e, T, false)
}

func (g *Game) drawBoltC(e proto.Fx, T float64, golden bool) {
	k := 0
	if golden {
		k = 1
	}
	if e.P > 0.4 {
		return
	}
	step, top, segs := int(T*25), float64(g.y0-2), 12
	px, py := e.X+float64(int(hash2(int(e.X), int(e.Y), 5)%30)-15), top
	for s := 1; s <= segs; s++ {
		t := float64(s) / float64(segs)
		nx, ny := e.X, top+(e.Y-top)*t
		if s != segs {
			nx = e.X + (px-e.X)*(1-t) + float64(int(hash2(s, step, int(e.X))%9)-4)
		}
		g.streak(px, py, nx, ny, boltCore[k])
		if hash2(s, int(e.Y), 9)%4 == 0 { // galho
			bl, dir := float64(4+hash2(s, 3, 9)%6), 1.0
			if hash2(s, 4, 9)&1 == 0 {
				dir = -1
			}
			g.streak(nx, ny, nx+dir*bl, ny+bl*0.8, boltBranch[k])
		}
		px, py = nx, ny
	}
	g.light(e.X, e.Y, 70, 2.2*(1-e.P), boltLight[k][0], boltLight[k][1])
}

// ---------- água, chuva e ervas ----------

type ripple struct{ x, y, t, max float64 }

// drawRipples desenha os anéis achatados que se abrem na água e somem.
func (g *Game) drawRipples(now float64) {
	n := 0
	m := g.m
	for _, rp := range g.ripples {
		age := (now - rp.t) / 1100
		if age >= 1 {
			continue
		}
		g.ripples[n] = rp
		n++
		r, k := 1+age*rp.max, 1-age
		steps := max(10, int(r*7))
		for s := 0; s < steps; s++ {
			a := float64(s) / float64(steps) * tau
			x, y := round(rp.x+math.Cos(a)*r), round(rp.y+math.Sin(a)*r*0.45)
			if x < 0 || y < 0 || x >= m.w || y >= m.h || m.flags[y*m.w+x]&4 == 0 {
				continue
			}
			g.setCell(x, y, slope(a+math.Pi/2), 120*k+40, 150*k+50, 200*k+60)
		}
	}
	g.ripples = g.ripples[:n]
}

func (g *Game) inRegion(x, y int, name string) bool {
	for _, r := range g.info.Regions {
		if r.Name == name && math.Hypot(float64(x-r.X), float64(y-r.Y)) < float64(r.R) {
			return true
		}
	}
	return false
}

// drawWeather desenha a chuva (ou a neve, nos picos) caindo na diagonal,
// os respingos no chão e os anéis nas poças.
func (g *Game) drawWeather(me *proto.Player, T, now float64) {
	if g.cur.Wx == 0 || g.cur.Wx == 3 || g.inRegion(me.X, me.Y, "Deserto de Ankar") {
		return
	}
	snow := g.inRegion(me.X, me.Y, "Picos de Gelo") || g.cur.Season == 3
	dens := 0.006
	if g.cur.Wx == 2 {
		dens = 0.011
	}
	n := int(float64(g.cols*g.rows) * dens)
	cols, rows := float64(g.cols), float64(g.rows)
	for i := 0; i < n; i++ {
		hx, hy := hash2(i, 1, 77), hash2(i, 2, 77)
		speed := 60 + float64(hx&31)
		if snow {
			speed = 7 + float64(hx&7)
		}
		sy := math.Mod(float64(hy%uint32(g.rows))+T*speed, rows)
		drift := -T * speed * 0.3
		if snow {
			drift = math.Sin(T+float64(i)) * 3
		}
		sx := math.Mod(math.Mod(float64(hx%uint32(g.cols))+drift, cols)+cols, cols)
		wx, wy := float64(g.x0)+sx, float64(g.y0)+sy
		if snow {
			code := byte('.')
			if hx&1 != 0 {
				code = '*'
			}
			g.setCell(round(wx), round(wy), code, 235, 240, 255)
			continue
		}
		c := col{120, 140, 180}
		if g.cur.Wx == 2 {
			c = col{150, 170, 210}
		}
		g.streak(wx+1.2, wy-3, wx, wy, c)
	}
	if snow {
		return
	}
	m := g.m
	for i := 0; i < n/5; i++ {
		x, y := g.x0+rand.Intn(g.cols), g.y0+rand.Intn(g.rows)
		if x < 0 || y < 0 || x >= m.w || y >= m.h {
			continue
		}
		f := m.flags[y*m.w+x]
		if f&4 != 0 && rand.Float64() < 0.15 {
			g.ripples = append(g.ripples, ripple{float64(x), float64(y), now, 3})
		} else if f&1 == 0 {
			code := byte('.')
			if rand.Float64() > 0.7 {
				code = 'o'
			}
			g.setCell(x, y, code, 170, 190, 225)
		}
	}
}

var herbCols = [][2]col{{{70, 170, 60}, {190, 110, 230}}, {{150, 60, 30}, {255, 150, 60}}, {{60, 160, 120}, {150, 230, 190}}}

// drawHerb desenha uma erva: antídoto, raiz-brasa (que brilha) ou babosa.
func (g *Game) drawHerb(h proto.Thing, T float64) {
	hc := herbCols[clampi(h.Kind, 0, 2)]
	sway := 0
	if math.Sin(T*2+float64(h.ID)) > 0.6 {
		sway = 1
	}
	x, y := int(h.X), int(h.Y)
	for dy := -4; dy <= 0; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if math.Abs(float64(dx)) > 1+float64(dy+4)*0.6 || hash2(dx, dy, h.Kind+7)%3 == 0 {
				continue
			}
			c, code := hc[0], "|/\\%"[(dx+3)%4]
			if dy <= -3 {
				c, code = hc[1], '*'
			}
			k := 0.8 + 0.3*float64(hash2(dx, dy, h.ID)&255)/255
			ox := 0
			if dy < -2 {
				ox = sway
			}
			g.setCell(x+dx+ox, y+dy, code, c[0]*k, c[1]*k, c[2]*k)
		}
	}
	if h.Kind == 1 {
		g.light(h.X, h.Y, 16, 0.6, warm, col{70, 30, 0})
	}
}

// drawBoat desenha o casco com o fundo em (x, y), por cima das pernas.
func (g *Game) drawBoat(x, y int) {
	for dy := -3; dy <= 1; dy++ {
		half := 8 - max(0, dy+1)
		if dy == 1 {
			half = 5
		}
		for dx := -half; dx <= half; dx++ {
			rim := dy == -3 || abs(dx) == half
			k := 1.05 - 0.04*float64(dx+half) + 0.06*float64(dy)
			c, code := col{116, 78, 46}, byte('#')
			if rim {
				c, code = col{168, 120, 72}, '='
			}
			g.setCell(x+dx, y+dy, code, c[0]*k, c[1]*k, c[2]*k)
		}
	}
	if rand.Float64() < 0.3 {
		g.spawn("foam", float64(x)+(rand.Float64()-0.5)*18, float64(y)+1, (rand.Float64()-0.5)*6, 1, 0.8)
	}
}

// segDist2 é a distância do ponto ao segmento de reta.
func segDist2(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	t := clamp(((px-ax)*vx+(py-ay)*vy)/(vx*vx+vy*vy), 0, 1)
	return math.Hypot(px-ax-t*vx, py-ay-t*vy)
}

// drawExit desenha a saída para a superfície: um vão escuro em arco, a luz de
// fora caindo em raios e uma poça clara no chão, com poeira flutuando. De dia
// a luz é quente e branca; de noite, fria e fraca, de luar.
func (g *Game) drawExit(p proto.Portal, tod, T float64) {
	x, y := float64(p.X), float64(p.Y)
	if x < float64(g.x0-40) || x > float64(g.x0+g.cols+40) || y < float64(g.y0-40) || y > float64(g.y0+g.rows+40) {
		return
	}
	sun := daylight(tod)
	beam := mix3(col{120, 150, 230}, col{255, 240, 190}, sun)
	k := 0.45 + 0.75*sun
	// Arco escuro em volta do vão e os degraus.
	for dx := -6; dx <= 6; dx++ {
		g.setC(int(x)+dx, int(y)-13, "_-~-_"[(dx+6)%5], mix3(col{60, 54, 50}, beam, 0.15))
	}
	for dy := -12; dy <= -1; dy++ {
		g.setC(int(x)-7, int(y)+dy, '(', col{70, 62, 58})
		g.setC(int(x)+7, int(y)+dy, ')', col{70, 62, 58})
	}
	for i, dy := range []int{-3, -6, -9} {
		for dx := -4 + i; dx <= 4-i; dx++ {
			g.setC(int(x)+dx, int(y)+dy, '=', mix3(col{90, 84, 76}, beam, 0.35+0.15*float64(i)))
		}
	}
	// Raios de luz descendo do vão, tremendo de leve.
	for r := 0; r < 9; r++ {
		a := -0.5 + float64(r)/8
		for d := 2.0; d < 22; d += 1.5 {
			if hash2(r, int(d), int(T*3))%3 == 0 {
				continue
			}
			px, py := x+a*d*0.9, y-12+d
			fade := (1 - d/22) * (0.6 + 0.4*math.Sin(T*1.3+float64(r)))
			g.setC(round(px), round(py), "|/\\'"[r%4], mix3(col{40, 38, 44}, beam, fade*k*0.9))
		}
	}
	g.light(x, y-4, 30, k, col{1 + 0.4*sun, 1 + 0.35*sun, 0.9 + 0.2*sun}, col{50 * sun, 46 * sun, 30*sun + 20})
	if rand.Float64() < 0.25 {
		g.spawn("spark", x+(rand.Float64()-0.5)*12, y-6+rand.Float64()*8, (rand.Float64()-0.5)*2, -1-rand.Float64()*2, 1.6)
	}
}
