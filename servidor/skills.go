package main

import (
	"math"
	"time"
)

// Habilidades: cada classe tem um livro de golpes (o 0 é o básico, o 1 é o
// golpe de classe que já se sabe ao nascer) e uma árvore em teia: um núcleo
// no meio, ramos que saem dele e anéis que ligam os ramos. Cada nó pede
// nível e pontos no atributo da classe, custa 1 ponto de habilidade (um por
// nível) e só abre ligado a um nó já aprendido. Nós ativos ensinam golpes;
// passivos somam bônus por grau. O mago tem quatro escolas elementais.

// Elementos: dão a cor do ícone, o bônus das passivas e os efeitos.
const (
	elemPhys = iota
	elemFire
	elemIce
	elemBolt
	elemArcane
	elemShadow
	elemHoly
	elemPoison
	numElem
)

var elemNames = [numElem]string{"físico", "fogo", "gelo", "raio", "arcano", "sombra", "sagrado", "veneno"}

const (
	maxSkills = 32 // golpes por classe, no máximo (o Depravado junta os de várias)
	barSlots  = 9  // atalhos na barra (teclas 1 a 9); abrem com o nível (barCap)
)

// Requisitos por anel da teia: nível e pontos no atributo da classe.
var ringLv = [5]int{0, 2, 4, 7, 11}
var ringAttr = [5]int{0, 1, 3, 6, 10}

// hitOpt são os efeitos de um golpe além do dano.
type hitOpt struct {
	Elem         int
	Slow, Stun   float64 // segundos
	Dot          int     // dano por segundo
	DotFor       float64 // por quantos segundos
	Pct, Crit    int     // % de dano e de crítico a mais
	Ignite       float64 // raio de capim que pega fogo onde cai
	ignoreBuffed bool
}

type skillDef struct {
	Name, Icon, Desc string
	Elem             int
	Cost             int
	CD               time.Duration
	Anim             int // animação no cliente: 0 básico, 1 de classe, 2 grande
	do               func(c *cast)
}

// bonus é o que as passivas somam (por grau, na definição do nó; somado, no jogador).
type bonus struct {
	Basic, Skill, Def, CDR, MPRegen, MP, HP, Stam, Leech, Crit int
	Burn, Chain, Chill                                         int // fogo queima, raio salta mais, gelo prende mais (décimos de s)
	Minion                                                     int // servos do necromante: +30% de vida e +5 s por grau
	Elem                                                       [numElem]int
}

func (b *bonus) add(o bonus, n int) {
	b.Basic += o.Basic * n
	b.Skill += o.Skill * n
	b.Def += o.Def * n
	b.CDR += o.CDR * n
	b.MPRegen += o.MPRegen * n
	b.MP += o.MP * n
	b.HP += o.HP * n
	b.Stam += o.Stam * n
	b.Leech += o.Leech * n
	b.Crit += o.Crit * n
	b.Burn += o.Burn * n
	b.Chain += o.Chain * n
	b.Chill += o.Chill * n
	b.Minion += o.Minion * n
	for i := range b.Elem {
		b.Elem[i] += o.Elem[i] * n
	}
}

type webNode struct {
	Name, Desc, Icon string
	Elem             int
	Ring             int
	Ang              float64 // graus, 0 à direita, sentido horário (y para baixo)
	Links            []int
	Max, Lv, Attr    int
	Skill            int // golpe que ensina; -1 é passivo
	Branch           int // ramo do nó (-1 núcleo e golpe final)
	Crypt            bool
	Per              bonus
}

type branchDef struct {
	Name string
	Ang  float64
	Elem int
}

// ---------- montagem da teia ----------

type nodeSpec struct {
	skill        int // >= 0: ativo
	name, desc   string
	icon         string
	elem, max    int
	per          bonus
	crypt        bool
	sameAsBranch bool
}

func active(skill int) nodeSpec { return nodeSpec{skill: skill, max: 1} }
func crypt(skill int) nodeSpec  { return nodeSpec{skill: skill, max: 1, crypt: true} }
func passive(name, desc, icon string, elem, max int, per bonus) nodeSpec {
	return nodeSpec{skill: -1, name: name, desc: desc, icon: icon, elem: elem, max: max, per: per}
}

// buildWeb monta a teia: o núcleo (0), quatro anéis por ramo e fios entre
// os ramos vizinhos no segundo anel.
func buildWeb(c *class, branches []branchDef, specs [][4]nodeSpec) []webNode {
	web := []webNode{{Name: "Núcleo — " + c.Name, Desc: "O começo do caminho. " + c.Skills[0].Name + " e " + c.Skills[1].Name + " já vêm aprendidos.",
		Icon: c.Skills[1].Icon, Elem: c.Skills[1].Elem, Max: 1, Skill: -1, Branch: -1}}
	ring2 := make([]int, len(branches))
	for b, br := range branches {
		prev := 0
		for k, s := range specs[b] {
			ring := k + 1
			n := webNode{Ring: ring, Ang: br.Ang, Links: []int{prev}, Max: s.max, Lv: ringLv[ring], Attr: ringAttr[ring], Skill: s.skill, Crypt: s.crypt, Per: s.per, Branch: b}
			if s.skill >= 0 {
				sk := c.Skills[s.skill]
				n.Name, n.Desc, n.Icon, n.Elem = sk.Name, sk.Desc, sk.Icon, sk.Elem
			} else {
				n.Name, n.Desc, n.Icon, n.Elem = s.name, s.desc, s.icon, s.elem
			}
			if s.crypt {
				n.Desc += " Só abre depois da primeira descida à cripta."
			}
			web = append(web, n)
			prev = len(web) - 1
			if ring == 2 {
				ring2[b] = prev
			}
		}
	}
	// Fios do anel: cada nó do segundo anel liga ao do ramo seguinte.
	for b := range branches {
		a, z := ring2[b], ring2[(b+1)%len(branches)]
		if len(branches) > 2 || b == 0 {
			web[z].Links = append(web[z].Links, a)
		}
	}
	return web
}

// neighbors é a lista de quem liga a cada nó (nos dois sentidos).
func neighbors(web []webNode) [][]int {
	out := make([][]int, len(web))
	for i, n := range web {
		for _, l := range n.Links {
			out[i] = append(out[i], l)
			out[l] = append(out[l], i)
		}
	}
	return out
}

// Passivas comuns, por grau.
var (
	pSharp   = passive("Golpe Afiado", "Treino de lâmina e de pontaria: +8% de dano no golpe básico por grau.", "afiado", elemPhys, 3, bonus{Basic: 8})
	pSkin    = passive("Pele Dura", "Calos e couro grosso: +4% de proteção contra todo dano por grau.", "pele", elemPhys, 3, bonus{Def: 4})
	pFocus   = passive("Foco", "A mente encurta a espera: -6% na recarga de todos os golpes por grau.", "foco", elemArcane, 3, bonus{CDR: 6})
	pBreath  = passive("Fôlego", "+1 de mana por segundo e +20% na volta da stamina por grau.", "folego", elemPhys, 3, bonus{MPRegen: 1, Stam: 20})
	pMastery = passive("Maestria", "Domínio da classe: +12% de dano em todos os golpes que não são o básico, por grau.", "maestria", elemPhys, 2, bonus{Skill: 12})
	pLeech   = passive("Sede de Sangue", "Cada golpe que acerta devolve em vida 2% do dano causado, por grau.", "sede", elemShadow, 2, bonus{Leech: 2})
	pVigor   = passive("Vigor", "+20 de vida máxima por grau.", "vigor", elemHoly, 3, bonus{HP: 20})
	pCrit    = passive("Olho Frio", "+4% de chance de crítico (dano x1,8) por grau.", "critico", elemPhys, 3, bonus{Crit: 4})
)

// ---------- execução ----------

// cast é o contexto de um golpe sendo usado.
type cast struct {
	wd               *World
	p                *Player
	id               int
	now              time.Time
	x, y, dir        float64
	cos, sin         float64
	sk               *skillDef
	ahead            func(d float64) (float64, float64)
	fxAt             func(kind int, ex, ey, r float64, ms int)
	fxDir            func(kind int, ex, ey, r, dir float64, ms int)
	hit              func(lo, sp int, in shape, o hitOpt)
	shoot            func(kind int, speed, reach, off float64)
	shootOpt         func(kind int, speed, reach, off float64, lo, sp int, r float64, o hitOpt)
	blastAt          func(after time.Duration, tx, ty, r float64, fx, lo, sp int, o hitOpt)
	everyFor         func(dur, every time.Duration, tick func(now time.Time))
	alliesInside     func(r float64) []*Player
	buffAllies       func(r float64, f func(q *Player))
	elemOf           func() hitOpt
	hitWithElemOf    func(lo, sp int, in shape)
	hazardFx         func(kind int, ex, ey, r, dir float64, dur time.Duration)
	mobsNear         func(x, y, r float64) []*Mob
	nearestMobInCone func(r, half float64) *Mob
}

func (wd *World) newCast(p *Player, sk *skillDef, dir float64, now time.Time) *cast {
	c := &cast{wd: wd, p: p, id: p.ID, now: now, x: float64(p.X), y: float64(p.Y), dir: dir, cos: math.Cos(dir), sin: math.Sin(dir), sk: sk}
	c.ahead = func(d float64) (float64, float64) { return c.x + c.cos*d, c.y + c.sin*d }
	c.fxDir = func(kind int, ex, ey, r, d float64, ms int) {
		wd.effects = append(wd.effects, Effect{Kind: kind, X: ex, Y: ey, R: r, Dir: d, Start: now, Dur: time.Duration(ms) * time.Millisecond})
	}
	c.fxAt = func(kind int, ex, ey, r float64, ms int) { c.fxDir(kind, ex, ey, r, dir, ms) }
	c.hazardFx = func(kind int, ex, ey, r, d float64, dur time.Duration) {
		wd.effects = append(wd.effects, Effect{Kind: kind, X: ex, Y: ey, R: r, Dir: d, Start: now, Dur: dur})
	}
	c.elemOf = func() hitOpt { return hitOpt{Elem: sk.Elem} }
	c.hit = func(lo, sp int, in shape, o hitOpt) { wd.strikeOpt(c.id, now, lo, sp, in, o) }
	c.hitWithElemOf = func(lo, sp int, in shape) { c.hit(lo, sp, in, c.elemOf()) }
	c.shootOpt = func(kind int, speed, reach, off float64, lo, sp int, r float64, o hitOpt) {
		wd.nextProj++
		wd.projs = append(wd.projs, Proj{ID: wd.nextProj, Kind: kind, X: c.x, Y: c.y,
			vx: math.Cos(dir+off) * speed, vy: math.Sin(dir+off) * speed, left: reach, owner: c.id, lo: lo, sp: sp, r: r, opt: o})
	}
	c.shoot = func(kind int, speed, reach, off float64) {
		o := c.elemOf()
		if kind == projFireball {
			o.Ignite = 6
		}
		c.shootOpt(kind, speed, reach, off, 0, 0, 0, o)
	}
	c.blastAt = func(after time.Duration, tx, ty, r float64, fx, lo, sp int, o hitOpt) {
		wd.blasts = append(wd.blasts, blast{now.Add(after), tx, ty, r, c.id, fx, lo, sp, o})
	}
	c.everyFor = func(dur, every time.Duration, tick func(now time.Time)) {
		wd.hazards = append(wd.hazards, hazard{until: now.Add(dur), next: now.Add(every / 2), every: every, tick: tick})
	}
	c.alliesInside = func(r float64) []*Player {
		var out []*Player
		for _, q := range wd.players {
			if dist(float64(q.X), float64(q.Y), c.x, c.y) <= r {
				out = append(out, q)
			}
		}
		return out
	}
	c.buffAllies = func(r float64, f func(q *Player)) {
		for _, q := range c.alliesInside(r) {
			f(q)
		}
	}
	c.mobsNear = func(x, y, r float64) []*Mob {
		var out []*Mob
		for _, m := range wd.mobs {
			if m.Alive && now.Sub(m.bornAt) >= mobRise && dist(m.X, m.Y, x, y) <= r {
				out = append(out, m)
			}
		}
		return out
	}
	c.nearestMobInCone = func(r, half float64) *Mob {
		var best *Mob
		bd := math.MaxFloat64
		for _, m := range c.mobsNear(c.x, c.y, r) {
			d := dist(m.X, m.Y, c.x, c.y)
			if d > 24 && math.Abs(math.Remainder(math.Atan2(m.Y-c.y, m.X-c.x)-dir, 2*math.Pi)) > half {
				continue
			}
			if d < bd {
				best, bd = m, d
			}
		}
		return best
	}
	return c
}

// hazard é um efeito que dura e bate de tempos em tempos (muralha de fogo,
// nevasca, tempestade, consagração).
type hazard struct {
	until, next time.Time
	every       time.Duration
	tick        func(now time.Time)
}

func (wd *World) stepHazards(now time.Time) {
	live := wd.hazards[:0]
	for _, h := range wd.hazards {
		for !now.Before(h.next) && h.next.Before(h.until) {
			h.tick(h.next)
			h.next = h.next.Add(h.every)
		}
		if now.Before(h.until) {
			live = append(live, h)
		}
	}
	wd.hazards = live
}

// ---------- efeitos nos inimigos ----------

// afflict aplica lentidão, atordoamento e dano contínuo num inimigo.
func (wd *World) afflictMob(m *Mob, o hitOpt, owner int, now time.Time) {
	if o.Slow > 0 {
		if t := now.Add(time.Duration(o.Slow * float64(time.Second))); t.After(m.slowUntil) {
			m.slowUntil = t
		}
	}
	if o.Stun > 0 && !mobKinds[m.Kind].Boss {
		if t := now.Add(time.Duration(o.Stun * float64(time.Second))); t.After(m.stunUntil) {
			m.stunUntil = t
		}
	}
	if o.Dot > 0 && o.DotFor > 0 {
		m.dotDmg, m.dotOwner, m.dotElem = o.Dot, owner, o.Elem
		m.dotUntil = now.Add(time.Duration(o.DotFor * float64(time.Second)))
		if m.dotNext.Before(now) {
			m.dotNext = now.Add(time.Second)
		}
	}
}

// damageMob tira vida de um inimigo e cuida da morte.
func (wd *World) damageMob(m *Mob, d int, p *Player, now time.Time) {
	m.HP -= d
	m.hurtAt = now
	if m.HP <= 0 {
		wd.mobDied(m, p, now)
	}
}

// tickDot aplica o dano contínuo (queimadura, veneno) uma vez por segundo.
func (wd *World) tickDot(m *Mob, now time.Time) {
	if m.dotDmg == 0 || now.After(m.dotUntil) {
		m.dotDmg = 0
		return
	}
	if now.Before(m.dotNext) {
		return
	}
	m.dotNext = m.dotNext.Add(time.Second)
	p := wd.players[m.dotOwner]
	if p != nil {
		c := 0xff9040
		if m.dotElem == elemPoison {
			c = 0x8ae060
		}
		wd.pop(p, "-"+itoa(m.dotDmg), c, m.X, m.Y-26, now)
	}
	wd.damageMob(m, m.dotDmg, p, now)
}

// mobStatus resume os efeitos de um inimigo em bits para o cliente pintar.
func mobStatus(m *Mob, now time.Time) int {
	st := 0
	if now.Before(m.slowUntil) {
		st |= 1
	}
	if now.Before(m.stunUntil) {
		st |= 2
	}
	if m.dotDmg > 0 && now.Before(m.dotUntil) {
		if m.dotElem == elemPoison {
			st |= 8
		} else {
			st |= 4
		}
	}
	return st
}

// ---------- jogador: teia, barra e bônus ----------

func (p *Player) web() []webNode { return classes[p.Class].Web }

// knows diz se o jogador sabe o golpe i do livro da classe.
func (p *Player) knows(i int) bool {
	c := classes[p.Class]
	if i < 0 || i >= len(c.Skills) {
		return false
	}
	if i <= 1 {
		return true
	}
	if i >= c.techBase {
		return p.knowsTech(i - c.techBase)
	}
	for n, nd := range c.Web {
		if nd.Skill == i && n < len(p.Web) && p.Web[n] > 0 {
			return true
		}
	}
	return false
}

// fixWeb ajusta o tamanho das listas da teia e da barra à classe.
func (p *Player) fixWeb() {
	w := p.web()
	if len(p.Web) != len(w) {
		nw := make([]int, len(w))
		copy(nw, p.Web)
		p.Web = nw
	}
	p.Web[0] = 1 // o núcleo
	for i := range p.Web {
		p.Web[i] = min(max(0, p.Web[i]), w[i].Max)
	}
	for i, s := range p.Bar {
		if s >= 0 && !p.knows(s) {
			p.Bar[i] = -1
		}
	}
	if p.bar0 {
		return
	}
	p.bar0 = true
	if p.Bar == [barSlots]int{} { // barra nunca montada: só o golpe de classe
		p.Bar = emptyBar(1, -1)
	}
}

// recalcBonus soma as passivas aprendidas.
func (p *Player) recalcBonus() {
	p.bon = bonus{}
	for i, n := range p.web() {
		if i < len(p.Web) && p.Web[i] > 0 && n.Skill < 0 {
			p.bon.add(n.Per, p.Web[i])
		}
	}
}

// canLearn diz se o nó i pode receber mais um grau, e por que não.
func (p *Player) canLearn(i int) (bool, string) {
	w := p.web()
	if i <= 0 || i >= len(w) {
		return false, "nó inválido"
	}
	n := w[i]
	switch {
	case p.Web[i] >= n.Max:
		return false, "já está no máximo"
	case p.SkillPts <= 0:
		return false, "sem pontos de habilidade"
	case p.Level < n.Lv:
		return false, "nível baixo"
	case p.classAttr() < n.Attr:
		return false, attrNames[classes[p.Class].Attr] + " baixa"
	case n.Crypt && !p.Unlocked:
		return false, "desça à cripta primeiro"
	case classes[p.Class].anyAttr && n.Branch >= 0 && p.Web[i] == 0 && !p.schools()[n.Branch] && len(p.schools()) >= maxSchools:
		return false, "já segue três escolas"
	}
	for _, l := range classes[p.Class].Adj[i] {
		if p.Web[l] > 0 {
			return true, ""
		}
	}
	return false, "ligue a um nó aprendido"
}

// SpendTree aprende mais um grau do nó i da teia (o nome ficou do tempo da
// árvore antiga; o navegador e o terminal ainda chamam assim).
func (wd *World) SpendTree(id, i int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok {
		return
	}
	p.fixWeb()
	if can, _ := p.canLearn(i); !can {
		return
	}
	p.SkillPts--
	p.Web[i]++
	p.recalc()
	if s := p.web()[i].Skill; s >= 0 {
		for k, b := range p.Bar {
			if b == s {
				return
			}
			if b < 0 && k < p.barCap() {
				p.Bar[k] = s
				return
			}
		}
	}
}

// SetBar põe o golpe skill no atalho slot (ou limpa com -1). Se o golpe já
// estava em outro atalho, os dois trocam.
func (wd *World) SetBar(id, slot, skill int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || slot < 0 || slot >= p.barCap() || skill >= 0 && !p.knows(skill) {
		return
	}
	if skill < 0 {
		p.Bar[slot] = -1
		return
	}
	for k, b := range p.Bar {
		if b == skill {
			p.Bar[k] = p.Bar[slot]
		}
	}
	p.Bar[slot] = skill
}

// resetWeb devolve os pontos da teia (Tomo do Recomeço).
func (p *Player) resetWeb() {
	for i := 1; i < len(p.Web); i++ {
		p.SkillPts += p.Web[i]
		p.Web[i] = 0
	}
	p.Bar = emptyBar(1, -1)
	p.recalc()
}

// resetAttrs devolve os pontos de atributo (Tônico do Esquecimento).
func (p *Player) resetAttrs() {
	for i := range p.Attr {
		p.Points += p.Attr[i]
		p.Attr[i] = 0
	}
	p.recalc()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// barLevels são os níveis em que a barra ganha mais um atalho (5 de início,
// até 9).
var barLevels = []int{6, 10, 14, 18}

// barCap é quantos atalhos o jogador já tem.
func (p *Player) barCap() int {
	n := barSlots - len(barLevels)
	for _, lv := range barLevels {
		if p.Level >= lv {
			n++
		}
	}
	return n
}

// emptyBar é a barra com os dois primeiros atalhos dados e o resto vazio.
func emptyBar(a, b int) [barSlots]int {
	var bar [barSlots]int
	for i := range bar {
		bar[i] = -1
	}
	bar[0], bar[1] = a, b
	return bar
}
