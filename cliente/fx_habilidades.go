package main

import (
	"math"
	"math/rand"

	"asciimmo/proto"
)

// Efeitos das habilidades da teia (os tipos ficam em proto/kinds.go).
const (
	fxChain      = proto.FxChain
	fxFrost      = proto.FxFrost
	fxFlame      = proto.FxFlame
	fxZap        = proto.FxZap
	fxBlink      = proto.FxBlink
	fxBuff       = proto.FxBuff
	fxSmoke      = proto.FxSmoke
	fxBone       = proto.FxBone
	fxPlague     = proto.FxPlague
	fxBlizzard   = proto.FxBlizzard
	fxCrack      = proto.FxCrack
	fxWhip       = proto.FxWhip
	fxConsecrate = proto.FxConsecrate
	fxHammer     = proto.FxHammer
	fxGrave      = proto.FxGrave
	fxStorm      = proto.FxStorm
	fxSeal       = proto.FxSeal
	fxOblit      = proto.FxOblit
	fxGrass      = proto.FxGrass
)

func init() {
	pkinds["ice"] = &pkind{[]col{{255, 255, 255}, {190, 235, 255}, {100, 170, 230}, {40, 80, 150}}, 4, "*+:."}
	pkinds["volt"] = &pkind{[]col{{255, 250, 210}, {255, 225, 110}, {240, 170, 60}, {130, 80, 30}}, 0, "*+'."} // faíscas douradas
	pkinds["poison"] = &pkind{[]col{{220, 255, 160}, {140, 220, 70}, {70, 130, 40}, {30, 60, 20}}, -5, "o*:."}
	pkinds["cut"] = &pkind{[]col{{180, 230, 120}, {110, 180, 70}, {60, 110, 40}}, 40, "',`."}
	pkinds["blood2"] = &pkind{[]col{{255, 120, 120}, {200, 40, 40}, {110, 14, 14}}, -8, "*+:."}
}

// Cor de cada tipo de reforço (fxBuff, Dir): fúria, proteção, veneno, arcano, mira, sangue.
var buffKinds = []struct {
	part string
	c    col
}{{"fire", col{255, 120, 60}}, {"holy", col{255, 226, 130}}, {"poison", col{150, 230, 80}},
	{"arcane", col{170, 140, 255}}, {"spark", col{230, 230, 255}}, {"blood2", col{230, 60, 60}}}

// fxStart2 solta as partículas do começo dos efeitos novos. true se tratou.
func (g *Game) fxStart2(e proto.Fx) bool {
	cy := e.Y - 14
	switch e.Kind {
	case fxFrost:
		g.burst("ice", e.X, cy, int(e.R*6), e.R*3, 0.8)
	case fxGrass: // capim cortado voando
		for i := 0; i < min(60, int(e.R)*2); i++ {
			a := rand.Float64() * tau
			g.spawn("cut", e.X+math.Cos(a)*rand.Float64()*14, e.Y-2+math.Sin(a)*rand.Float64()*6, math.Cos(a)*20, -14-rand.Float64()*16, 0.6+rand.Float64()*0.5)
		}
	case fxOblit:
		g.flashAt = g.now()
		g.burst("arcane", e.X, cy, 120, 90, 0.9)
	case fxZap:
		g.burst("volt", e.X, e.Y-4, 30, 40, 0.4)
		g.shake = math.Max(g.shake, 0.3)
	case fxBlink:
		g.burst("arcane", e.X, cy, 40, 30, 0.5)
	case fxBuff:
		k := buffKinds[clampi(int(e.Dir), 0, len(buffKinds)-1)]
		for i := 0; i < 50; i++ {
			a, d := rand.Float64()*tau, math.Sqrt(rand.Float64())*math.Max(8, e.R*0.4)
			g.spawn(k.part, e.X+math.Cos(a)*d, e.Y-rand.Float64()*18+math.Sin(a)*d*0.5, 0, -10-rand.Float64()*14, 0.6+rand.Float64()*0.6)
		}
	case fxSmoke:
		for i := 0; i < int(e.R*5); i++ {
			a, d := rand.Float64()*tau, math.Sqrt(rand.Float64())*e.R
			g.spawn("smoke", e.X+math.Cos(a)*d, e.Y-6+math.Sin(a)*d*0.6, (rand.Float64()-0.5)*6, -2-rand.Float64()*4, 1.2+rand.Float64())
		}
	case fxPlague:
		for i := 0; i < int(e.R*4); i++ {
			a, d := rand.Float64()*tau, math.Sqrt(rand.Float64())*e.R
			g.spawn("poison", e.X+math.Cos(a)*d, e.Y-4+math.Sin(a)*d*0.6, 0, -4-rand.Float64()*6, 0.8+rand.Float64()*0.8)
		}
	case fxBone:
		ex, ey := e.X+math.Cos(e.Dir)*e.R, cy+math.Sin(e.Dir)*e.R
		g.burst("bone", ex, ey, 24, 30, 0.6)
	case fxGrave:
		g.burst("shadow", e.X, e.Y, 40, 26, 0.9)
	case fxCrack:
		g.burst("smoke", e.X+math.Cos(e.Dir)*e.R*0.5, e.Y+math.Sin(e.Dir)*e.R*0.5, 20, 16, 0.9)
	default:
		return false
	}
	return true
}

// drawFx2 desenha os efeitos novos. true se tratou.
func (g *Game) drawFx2(e proto.Fx, T float64) bool {
	if g.drawObliterate(e, T) {
		return true
	}
	cy, p := e.Y-14, e.P
	fade := 1 - p
	switch e.Kind {
	case fxChain:
		// Raio em zigue-zague entre dois pontos, tremendo a cada quadro.
		ex, ey := e.X+math.Cos(e.Dir)*e.R, e.Y+math.Sin(e.Dir)*e.R
		px, py := e.X, e.Y
		segs := max(3, int(e.R/6))
		nx, ny := -math.Sin(e.Dir), math.Cos(e.Dir)
		for s := 1; s <= segs; s++ {
			t := float64(s) / float64(segs)
			j := 0.0
			if s != segs {
				j = float64(int(hash2(s, int(T*30), int(e.X))%7) - 3)
			}
			qx, qy := e.X+(ex-e.X)*t+nx*j, e.Y+(ey-e.Y)*t+ny*j
			g.streak(px, py, qx, qy, mix3(col{255, 245, 190}, col{230, 150, 40}, p))
			px, py = qx, qy
		}
		g.light(ex, ey+14, 30, fade*1.4, col{1.5, 1.25, 0.6}, col{150, 110, 30})
	case fxFrost:
		r := e.R * ease(p)
		g.ring(e.X, cy, r, 0, tau, mix3(col{230, 250, 255}, col{40, 80, 150}, p))
		g.ring(e.X, cy, r*0.75, 0, tau, mix3(col{150, 210, 255}, col{30, 60, 120}, p))
		for k := 0; k < 10; k++ {
			a := float64(k)*tau/10 + p
			g.setC(round(e.X+math.Cos(a)*r*0.9), round(cy+math.Sin(a)*r*0.9), '*', col{220, 245, 255})
		}
		g.light(e.X, e.Y, e.R*3, fade*1.2, cool, coolAdd)
	case fxFlame:
		// Muralha: chamas em fila, atravessada na mira.
		dx, dy := math.Cos(e.Dir), math.Sin(e.Dir)
		for t := -e.R; t <= e.R; t++ {
			bx, by := e.X+dx*t, e.Y+dy*t
			n := hash2(int(t), int(T*12), 5)
			h := 2 + int(n%4)
			for k := 0; k < h; k++ {
				c := mix3(col{255, 245, 170}, col{230, 60, 20}, float64(k)/float64(h))
				g.setC(round(bx), round(by)-k, "^*'\")"[(n>>uint(k))%5], c)
			}
		}
		g.light(e.X, e.Y, e.R*2+16, 1.3*math.Min(1, fade*4), warm, warmAdd)
		if rand.Float64() < 0.6 {
			t := (rand.Float64()*2 - 1) * e.R
			g.spawn("fire", e.X+dx*t, e.Y+dy*t-3, 0, -20, 0.5)
		}
	case fxZap:
		g.drawBoltC(e, T, true) // raio do mago: dourado
	case fxBlink:
		g.ring(e.X, cy, e.R*(0.4+0.6*p), 0, tau, mix3(col{230, 210, 255}, col{60, 30, 120}, p))
	case fxBuff:
		k := buffKinds[clampi(int(e.Dir), 0, len(buffKinds)-1)]
		g.ring(e.X, e.Y, math.Max(6, e.R*ease(p)*0.5), 0, tau, mix3(k.c, col{30, 20, 30}, p))
	case fxSmoke:
		g.ring(e.X, e.Y-4, e.R*(0.6+0.4*p), 0, tau, mix3(col{120, 116, 130}, col{30, 28, 36}, p))
	case fxBone:
		ex, ey := e.X+math.Cos(e.Dir)*e.R*math.Min(1, p*2.5), cy+math.Sin(e.Dir)*e.R*math.Min(1, p*2.5)
		g.streak(e.X, cy, ex, ey, mix3(col{250, 235, 210}, col{90, 70, 60}, p))
		g.setC(round(ex), round(ey), '>', col{255, 245, 225})
	case fxPlague:
		r := e.R * ease(p)
		g.ring(e.X, e.Y, r, 0, tau, mix3(col{180, 255, 120}, col{20, 50, 10}, p))
		g.light(e.X, e.Y, e.R*2, fade, col{0.4, 1.2, 0.4}, col{20, 70, 10})
	case fxBlizzard:
		// Neve caindo dentro do círculo e o chão esbranquiçado na borda.
		g.ring(e.X, e.Y, e.R, 0, tau, col{150, 200, 240})
		for i := 0; i < 5; i++ {
			a, d := rand.Float64()*tau, math.Sqrt(rand.Float64())*e.R
			g.spawn("snow", e.X+math.Cos(a)*d+6, e.Y+math.Sin(a)*d*0.7-30, -14, 40, 0.7)
		}
		g.light(e.X, e.Y, e.R*1.5, 0.6, cool, coolAdd)
	case fxWhip:
		// Aviso do chicote de Vulkhar: uma linha em brasa que pulsa mais
		// rápido quanto mais perto de arder.
		ex, ey := e.X+math.Cos(e.Dir)*e.R, e.Y+math.Sin(e.Dir)*e.R
		k := 0.5 + 0.5*math.Sin(T*(8+30*p))
		c := mix3(col{120, 30, 10}, col{255, 150, 40}, k)
		for i := 0; i <= int(e.R); i++ {
			t := float64(i) / math.Max(1, e.R)
			for _, w := range []float64{-1, 0, 1} {
				x, y := e.X+(ex-e.X)*t-math.Sin(e.Dir)*w*2, e.Y+(ey-e.Y)*t+math.Cos(e.Dir)*w*2
				ch := byte('-')
				if w != 0 {
					ch = '.'
				}
				g.setC(round(x), round(y), ch, c)
			}
		}
		g.light(e.X+(ex-e.X)/2, e.Y+(ey-e.Y)/2, e.R*0.6, 0.4+0.4*k, warm, col{60, 20, 0})
	case fxCrack:
		ex, ey := e.X+math.Cos(e.Dir)*e.R, e.Y+math.Sin(e.Dir)*e.R
		n := max(1, int(e.R))
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(n)
			if t > math.Min(1, p*3) {
				break
			}
			j := float64(int(hash2(i, int(e.X), 3)%3) - 1)
			g.setC(round(e.X+(ex-e.X)*t-math.Sin(e.Dir)*j), round(e.Y+(ey-e.Y)*t+math.Cos(e.Dir)*j), "/\\v^"[hash2(i, 1, int(e.Y))%4],
				mix3(col{60, 40, 30}, col{20, 14, 10}, p))
		}
	case fxConsecrate:
		r := e.R * math.Min(1, p*8)
		g.ring(e.X, e.Y, r, 0, tau, col{255, 226, 130})
		g.ring(e.X, e.Y, r*0.6, T, T+tau, col{230, 190, 90})
		for k := 0; k < 6; k++ {
			a := float64(k)*tau/6 + T*0.5
			g.streak(e.X, e.Y, e.X+math.Cos(a)*r*0.6, e.Y+math.Sin(a)*r*0.6, col{200, 160, 70})
		}
		if rand.Float64() < 0.5 {
			a, d := rand.Float64()*tau, rand.Float64()*e.R
			g.spawn("holy", e.X+math.Cos(a)*d, e.Y+math.Sin(a)*d*0.6, 0, -10, 0.8)
		}
		g.light(e.X, e.Y, e.R*1.8, 0.9, col{1.2, 1.0, 0.4}, col{90, 70, 20})
	case fxHammer:
		g.ring(e.X, e.Y, e.R, 0, tau, col{255, 226, 130})
		g.ring(e.X, e.Y, e.R*p, 0, tau, col{200, 160, 60})
		mx, my := e.X, e.Y-100*fade
		g.orb(mx, my, 4, col{255, 255, 230}, col{230, 180, 70}, T)
		g.light(mx, my+20, 50, 1, col{1.2, 1.0, 0.4}, col{110, 90, 20})
	case fxGrave:
		// Dedos de osso brotando do chão em círculo.
		h := math.Min(1, p*3)
		for k := 0; k < 9; k++ {
			a := float64(k) * tau / 9
			bx, by := e.X+math.Cos(a)*e.R*0.7, e.Y+math.Sin(a)*e.R*0.5
			for d := 0; d < int(5*h); d++ {
				g.setC(round(bx), round(by)-d, "|!'"[d%3], mix3(col{235, 220, 200}, col{80, 60, 70}, float64(d)/5))
			}
		}
		g.ring(e.X, e.Y, e.R, 0, tau, mix3(col{150, 90, 220}, col{30, 10, 50}, p))
	case fxStorm:
		// Nuvem escura: o chão embaixo escurece e a chuva cai.
		R := int(e.R)
		for dy := -R; dy <= R; dy += 2 {
			for dx := -R; dx <= R; dx += 2 {
				if dx*dx+dy*dy*2 > R*R {
					continue
				}
				sx, sy := int(e.X)+dx-g.x0, int(e.Y)+dy-g.y0
				if sx < 0 || sy < 0 || sx >= g.cols || sy >= g.rows {
					continue
				}
				o := (sy*g.cols + sx) * 4
				g.buf[o], g.buf[o+1], g.buf[o+2] = g.buf[o]*3/5, g.buf[o+1]*3/5, g.buf[o+2]*2/3
			}
		}
		for i := 0; i < 6; i++ {
			a, d := rand.Float64()*tau, math.Sqrt(rand.Float64())*e.R
			x, y := e.X+math.Cos(a)*d, e.Y+math.Sin(a)*d*0.7
			g.streak(x+3, y-8, x, y, col{150, 170, 220})
		}
	default:
		return false
	}
	return true
}

// mobTint pinta o inimigo pelo efeito que ele carrega: congelado fica
// branco-azulado, lento azulado, queimando alaranjado, envenenado verde.
func mobTint(st int, T float64) (float64, col) {
	switch {
	case st&2 != 0:
		return 0.55, col{170, 220, 255}
	case st&4 != 0:
		return 0.25 + 0.15*math.Sin(T*14), col{255, 130, 40}
	case st&8 != 0:
		return 0.25 + 0.1*math.Sin(T*6), col{120, 230, 60}
	case st&1 != 0:
		return 0.3, col{120, 170, 255}
	}
	return 0, col{}
}

// drawObliterate desenha o Obliterar: primeiro os cinco selos girando e se
// fechando num pentagrama em volta do mago; depois o feixe colossal.
func (g *Game) drawObliterate(e proto.Fx, T float64) bool {
	cy, p := e.Y-14, e.P
	switch e.Kind {
	case fxSeal:
		r := e.R * (1 - 0.55*ease(p))
		var pts [5][2]float64
		for k := 0; k < 5; k++ {
			a := -math.Pi/2 + float64(k)*tau/5 + p*2.5
			pts[k] = [2]float64{e.X + math.Cos(a)*r, cy + math.Sin(a)*r*0.75}
			g.orb(pts[k][0], pts[k][1], 2.2+p*1.5, col{255, 245, 255}, col{170, 90, 255}, T)
		}
		c := mix3(col{120, 60, 220}, col{255, 230, 255}, p)
		for k := 0; k < 5; k++ { // pentagrama: cada selo liga ao segundo seguinte
			a, b := pts[k], pts[(k+2)%5]
			g.streak(a[0], a[1], b[0], b[1], c)
		}
		g.ring(e.X, cy, r+3, 0, tau, c)
		if rand.Float64() < 0.8 {
			a := rand.Float64() * tau
			g.spawn("arcane", e.X+math.Cos(a)*e.R*1.4, cy+math.Sin(a)*e.R, -math.Cos(a)*30, -math.Sin(a)*24, 0.6)
		}
		g.light(e.X, e.Y, 50+40*p, 0.8+1.2*p, col{0.7, 0.4, 1.6}, col{80, 30, 140})
		g.shake = math.Max(g.shake, 0.25*p)
	case fxOblit:
		dx, dy := math.Cos(e.Dir), math.Sin(e.Dir)
		nx, ny := -dy, dx
		grow := math.Min(1, p*5)
		fade := 1 - math.Max(0, (p-0.6)/0.4)
		half := 14 * fade
		for t := 0.0; t < e.R*grow; t++ {
			for w := -half; w <= half; w++ {
				k := 1 - math.Abs(w)/math.Max(1, half)
				n := hash2(int(t), int(w), int(T*30))
				ch := "@#%*+:."[clampi(int((1-k)*7+float64(n&1)), 0, 6)]
				c := mix3(col{90, 30, 170}, col{255, 250, 255}, k*k)
				g.setC(round(e.X+dx*t+nx*w), round(cy+dy*t+ny*w*0.8), ch, c)
			}
		}
		g.light(e.X+dx*e.R*0.5, e.Y+dy*e.R*0.5, e.R*0.9, 2.5*fade, col{0.9, 0.7, 1.8}, col{140, 90, 220})
		g.shake = math.Max(g.shake, 0.9*fade)
	default:
		return false
	}
	return true
}
