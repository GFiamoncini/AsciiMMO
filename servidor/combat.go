package main

import (
	"asciimmo/proto"

	"fmt"
	"math"
	"time"
)

// bossRadius é a folga do dragão como alvo; os esqueletos usam mobRadius.
const (
	critOdds   = 12 // % de golpe crítico (o ladino tem mais)
	bossRadius = 18.0
	mobRadius  = 3.0
)

// Projéteis: raio da explosão e dano (mínimo e variação) de cada tipo.
const (
	projMissile  = proto.ProjMissile
	projFireball = proto.ProjFireball
	projArrow    = proto.ProjArrow
	projShadow   = proto.ProjShadow
	projIce      = proto.ProjIce
	projHoly     = proto.ProjHoly
)

var projDefs = []struct {
	R      float64
	Lo, Sp int
}{
	projMissile:  {5, 20, 13},
	projFireball: {16, 55, 31},
	projArrow:    {4, 22, 15},
	projShadow:   {6, 20, 14},
	projIce:      {5, 30, 16},
	projHoly:     {7, 35, 18},
}

// Proj é um projétil em voo. lo/sp/r, quando dados, trocam o dano e o raio
// padrão do tipo; opt leva o elemento e os efeitos.
type Proj struct {
	ID, Kind int
	X, Y     float64

	vx, vy float64 // células por tick
	left   float64 // alcance que ainda resta
	owner  int
	lo, sp int
	r      float64
	opt    hitOpt
}

// blast é uma explosão marcada para cair daqui a pouco (o meteoro).
type blast struct {
	at      time.Time
	x, y, r float64
	owner   int
	fx      int // efeito ao cair
	lo, sp  int // dano mínimo e variação
	opt     hitOpt
}

// As formas abaixo dizem se um alvo em (x, y), com folga pad, é atingido.
type shape func(x, y, pad float64) bool

func around(cx, cy, r float64) shape {
	return func(x, y, pad float64) bool { return dist(cx, cy, x, y) <= r+pad }
}

// arc é um leque de abertura half em torno de dir; quem está colado no
// atacante apanha de qualquer lado.
func arc(cx, cy, dir, r, half float64) shape {
	return func(x, y, pad float64) bool {
		d := dist(cx, cy, x, y)
		if d > r+pad {
			return false
		}
		return d < 6+pad || math.Abs(math.Remainder(math.Atan2(y-cy, x-cx)-dir, 2*math.Pi)) <= half
	}
}

// along é uma faixa de meia largura w entre dois pontos.
func along(ax, ay, bx, by, w float64) shape {
	return func(x, y, pad float64) bool {
		vx, vy := bx-ax, by-ay
		t := 0.0
		if l := vx*vx + vy*vy; l > 0 {
			t = clamp01(((x-ax)*vx + (y-ay)*vy) / l)
		}
		return math.Hypot(x-ax-t*vx, y-ay-t*vy) <= w+pad
	}
}

// strike fere sem elemento nem efeito (rochas, árvores, golpes simples).
func (wd *World) strike(owner int, now time.Time, lo, sp int, in shape) {
	wd.strikeOpt(owner, now, lo, sp, in, hitOpt{})
}

// strikeOpt fere o chefe e todo inimigo de pé dentro da forma. O dano vai de
// lo a lo+sp-1, mais o bônus de nível de quem bateu, vezes os bônus de dano
// (arma, Força, passivas, elemento, Grito de Guerra). Depois aplica os efeitos
// do golpe: lentidão, atordoamento, queimadura, veneno.
func (wd *World) strikeOpt(owner int, now time.Time, lo, sp int, in shape, o hitOpt) {
	p := wd.players[owner]
	dealt := 0
	if p != nil {
		// Passivas e reforços mexem nos efeitos.
		if o.Slow > 0 {
			o.Slow += float64(p.bon.Chill) / 10
		}
		if o.Elem == elemFire && p.bon.Burn > 0 && o.Dot == 0 {
			o.Dot, o.DotFor = 5*p.bon.Burn, 3
		}
		if o.Elem == elemPhys && now.Before(p.venomUntil) && o.Dot == 0 {
			o.Dot, o.DotFor = 8, 4
			o.Elem = elemPoison
		}
	}
	// dmg sorteia o dano num alvo em (x, y); às vezes sai crítico. Quem bateu
	// vê o número subir sobre o alvo.
	// dmgK leva também o % do elemento no alvo (fraqueza 150, resistência 60:
	// bestiario.go); dmg é o golpe em quem não tem fraqueza.
	var dmgK func(x, y float64, lv, weak int) int
	dmg := func(x, y float64, lv int) int { return dmgK(x, y, lv, 100) }
	dmgK = func(x, y float64, lv, weak int) int {
		d := lo + wd.rng.Intn(max(1, sp))
		odds := critOdds + o.Crit
		if p != nil {
			pct := p.dmgPct(p.act) + p.bon.Elem[clampInt(o.Elem, 0, numElem-1)] + o.Pct
			pct += p.abyssPct() // Armadura do Abismo, com menos de metade da vida
			if now.Before(p.dmgUntil) {
				pct += 30
			}
			d = (d + 4*p.Level) * pct / 100
			d = max(1, int(float64(d)*lvScale(p.Level, lv))) // diferença de nível
			odds += p.bon.Crit
			if p.Class == 5 {
				odds += 10
			}
			if now.Before(p.critUntil) {
				odds += 25
			}
		}
		crit := wd.rng.Intn(100) < odds
		if crit {
			d = d * 18 / 10
		}
		d = max(1, d*weak/100)
		dealt += d
		if p != nil {
			switch {
			case crit:
				wd.pop(p, fmt.Sprintf("CRÍTICO -%d", d), 0xffa030, x, y, now)
			case weak > 100:
				wd.pop(p, fmt.Sprintf("FRACO -%d", d), 0xffe040, x, y, now)
			case weak < 100:
				wd.pop(p, fmt.Sprintf("-%d resiste", d), 0x8a8a96, x, y, now)
			default:
				wd.pop(p, fmt.Sprintf("-%d", d), elemPopColor[clampInt(o.Elem, 0, numElem-1)], x, y, now)
			}
		}
		return d
	}
	// Sede de Sangue devolve em vida uma parte do dano causado.
	defer func() {
		if p != nil && p.bon.Leech > 0 {
			p.HP = min(p.MaxHP, p.HP+dealt*p.bon.Leech/100)
		}
	}()
	wd.hitProps(owner, now, lo+wd.rng.Intn(max(1, sp)), in)
	if p != nil && o.Elem != elemIce && o.Elem != elemShadow && o.Elem != elemHoly && o.Elem != elemPoison {
		wd.cutGrass(owner, float64(p.X), float64(p.Y), in, now) // golpes físicos, de fogo e raio cortam o capim
	}
	if b := &wd.boss; b.Alive && in(b.X, b.Y, bossRadius) && (p == nil || len(p.veilMissing()) == 0) { // o Véu protege Tiamath de quem não o derrubou
		wd.hitBoss(dmg(b.X, b.Y-40, levelDragon), now)
	}
	for _, m := range wd.mobs {
		if !m.Alive || now.Sub(m.bornAt) < mobRise || !in(m.X, m.Y, mobRadius) {
			continue
		}
		wd.damageMob(m, dmgK(m.X, m.Y-26, mobLv(m.Kind), weakPct(m.Kind, o.Elem)), p, now)
		if m.Alive {
			wd.afflictMob(m, o, owner, now)
		}
	}
}

// Cor do número de dano de cada elemento.
var elemPopColor = [numElem]int{0xf0f0f0, 0xffa060, 0xa0e0ff, 0xfff080, 0xd0a0ff, 0xb090d0, 0xfff0b0, 0x9ae070}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }

func (wd *World) heal(x, y, r float64, amount int) {
	for _, p := range wd.players {
		if dist(float64(p.X), float64(p.Y), x, y) <= r {
			p.HP = min(p.MaxHP, p.HP+amount)
		}
	}
}

// slide anda até n células na direção dada, parando em obstáculo ou selo, e
// devolve de onde saiu e onde parou.
func (wd *World) slide(p *Player, dir float64, n int, now time.Time) (float64, float64, float64, float64) {
	x0, y0 := float64(p.X), float64(p.Y)
	x, y := x0, y0
	for ; n > 0; n-- {
		nx, ny := x+math.Cos(dir), y+math.Sin(dir)
		if !wd.fits(int(math.Round(nx)), int(math.Round(ny))) || !wd.canEnter(p, int(math.Round(nx)), int(math.Round(ny)), now) {
			break
		}
		x, y = nx, ny
	}
	p.X, p.Y = int(math.Round(x)), int(math.Round(y))
	return x0, y0, x, y
}

// dash corre até n células na direção dada, parando em obstáculo, e fere
// quem estiver no caminho.
func (wd *World) dash(p *Player, dir float64, n, lo, sp int, now time.Time, o hitOpt) {
	x0, y0, x, y := wd.slide(p, dir, n, now)
	wd.effects = append(wd.effects, Effect{Kind: fxThrust, X: x0, Y: y0, R: dist(x0, y0, x, y), Dir: dir, Start: now, Dur: 300 * time.Millisecond})
	wd.strikeOpt(p.ID, now, lo, sp, along(x0, y0, x, y, 8), o)
	wd.enter(p)
}

// blink some e reaparece adiante, sem ferir ninguém.
func (wd *World) blink(p *Player, dir float64, n int, now time.Time) {
	x0, y0, x, y := wd.slide(p, dir, n, now)
	wd.effects = append(wd.effects,
		Effect{Kind: fxBlink, X: x0, Y: y0, R: 8, Start: now, Dur: 400 * time.Millisecond},
		Effect{Kind: fxBlink, X: x, Y: y, R: 8, Dir: 1, Start: now, Dur: 400 * time.Millisecond})
	wd.enter(p)
}

// chain é o raio que salta: acerta o mais perto na mira e pula para os
// próximos ainda não atingidos, perdendo força a cada salto.
func (wd *World) chain(c *cast, lo, sp, jumps int) {
	m := c.nearestMobInCone(70, 0.6)
	px, py := c.x, c.y-14
	if m == nil {
		ex, ey := c.ahead(55)
		c.fxDir(fxChain, px, py, 55, c.dir, 300)
		wd.strikeOpt(c.id, c.now, lo, sp, around(ex, ey, 4), hitOpt{Elem: elemBolt})
		return
	}
	hit := map[*Mob]bool{}
	pct := 0
	for k := 0; m != nil && k <= jumps; k++ {
		hit[m] = true
		d := dist(px, py, m.X, m.Y-14)
		c.fxDir(fxChain, px, py, d, math.Atan2(m.Y-14-py, m.X-px), 350)
		wd.strikeOpt(c.id, c.now, lo, sp, around(m.X, m.Y, 1), hitOpt{Elem: elemBolt, Pct: pct})
		pct -= 20
		px, py = m.X, m.Y-14
		var next *Mob
		best := 34.0
		for _, n := range c.mobsNear(m.X, m.Y, 34) {
			if dd := dist(n.X, n.Y, m.X, m.Y); !hit[n] && dd < best {
				next, best = n, dd
			}
		}
		m = next
	}
}

// enter teleporta quem pisou num portal. A primeira descida à cripta libera
// o nó da teia com o golpe da cripta.
func (wd *World) enter(p *Player) {
	for _, pt := range wd.portals {
		if dist(float64(p.X), float64(p.Y), float64(pt.X), float64(pt.Y)) > 4 {
			continue
		}
		p.X, p.Y, p.Riding = pt.TX, pt.TY, 0
		if pt.Fall {
			wd.say(ChatLine{Text: p.Name + " caiu num buraco!"})
			wd.hurtRaw(p, 12, "pela queda", wd.clock())
		}
		if pt.Dungeon {
			wd.questEvent(p, qEnter, wd.clock())
		}
		if pt.Dungeon && !p.Unlocked {
			p.Unlocked = true
			wd.say(ChatLine{Text: p.Name + " desceu à cripta: " + classes[p.Class].Skills[2].Name + " já pode ser aprendido na teia (K)."})
		}
		return
	}
}

// Use dispara o golpe i do livro da classe (0 é o básico). aim é o ângulo da
// mira, quando o cliente tem uma; sem ela vale a direção do andar.
func (wd *World) Use(id, i int, aim *float64) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok {
		wd.use(p, i, aim, wd.clock())
	}
}

// use solta o golpe i do jogador (com o lock); true se saiu.
func (wd *World) use(p *Player, i int, aim *float64, now time.Time) bool {
	if !p.knows(i) || p.paused {
		return false
	}
	sk := &classes[p.Class].Skills[i]
	if p.MP < sk.Cost || now.Before(p.ready[i]) {
		return false
	}
	dir := math.Atan2(float64(p.FaceY), float64(p.FaceX))
	if aim != nil && !math.IsNaN(*aim) && !math.IsInf(*aim, 0) {
		dir = *aim
	}
	p.ready[i], p.MP = now.Add(sk.CD*time.Duration(100-min(45, p.bon.CDR))/100), p.MP-sk.Cost
	p.act, p.actAt, p.actDir = i, now, dir
	sk.do(wd.newCast(p, sk, dir, now))
	return true
}

// Attack é o golpe básico na direção em que o jogador anda.
func (wd *World) Attack(id int) { wd.Use(id, 0, nil) }

// touches diz se há um alvo vivo encostado no ponto.
func (wd *World) touches(x, y float64, now time.Time) bool {
	if b := &wd.boss; b.Alive && dist(b.X, b.Y, x, y) < bossRadius {
		return true
	}
	for _, m := range wd.mobs {
		if m.Alive && now.Sub(m.bornAt) >= mobRise && dist(m.X, m.Y, x, y) < 5 {
			return true
		}
	}
	return false
}

// stepProjs voa os projéteis célula a célula, solta as explosões marcadas e
// roda os efeitos que duram (muralha de fogo, nevasca, tempestade...).
func (wd *World) stepProjs(now time.Time) {
	live := wd.projs[:0]
	for _, pr := range wd.projs {
		speed := math.Hypot(pr.vx, pr.vy)
		n := int(math.Ceil(speed))
		hit := false
		for i := 0; i < n && !hit; i++ {
			pr.X, pr.Y, pr.left = pr.X+pr.vx/float64(n), pr.Y+pr.vy/float64(n), pr.left-speed/float64(n)
			hit = pr.left <= 0 || !wd.Walkable(int(pr.X), int(pr.Y)) || wd.touches(pr.X, pr.Y, now)
		}
		if !hit {
			live = append(live, pr)
			continue
		}
		d := projDefs[pr.Kind]
		lo, sp, r := d.Lo, d.Sp, d.R
		if pr.lo > 0 {
			lo, sp = pr.lo, pr.sp
		}
		if pr.r > 0 {
			r = pr.r
		}
		look := float64(pr.Kind)
		if pr.Kind == projArrow && pr.opt.Elem == elemFire {
			look = projFireball // a flecha explosiva explode como bola de fogo
		}
		wd.effects = append(wd.effects, Effect{Kind: fxBoom, X: pr.X, Y: pr.Y, R: r, Dir: look, Start: now, Dur: 350 * time.Millisecond})
		if pr.opt.Ignite > 0 {
			wd.ignite(pr.X, pr.Y, pr.opt.Ignite, now)
		}
		wd.strikeOpt(pr.owner, now, lo, sp, around(pr.X, pr.Y, r), pr.opt)
	}
	wd.projs = live

	due := wd.blasts[:0]
	for _, b := range wd.blasts {
		if now.Before(b.at) {
			due = append(due, b)
			continue
		}
		wd.effects = append(wd.effects, Effect{Kind: b.fx, X: b.x, Y: b.y, R: b.r, Dir: projFireball, Start: now, Dur: 500 * time.Millisecond})
		if b.opt.Ignite > 0 {
			wd.ignite(b.x, b.y, b.opt.Ignite, now)
		}
		wd.strikeOpt(b.owner, now, b.lo, b.sp, around(b.x, b.y, b.r), b.opt)
	}
	wd.blasts = due
	wd.stepHazards(now)
}
