package main

import (
	"math"
	"time"
)

// Quarto ramo e golpe final das classes além do mago: dois golpes novos no
// ramo (índices 7 e 8 do livro) e um golpe final (9) no quinto anel, ligado
// aos quartos anéis de todos os ramos. Os ramos passam a quatro, em X.

type classExt struct {
	branch       branchDef
	spec         [4]nodeSpec
	skills       []skillDef // 7, 8 e o golpe final 9
	capLv, capAt int
}

var branchAngles = []float64{-135, -45, 45, 135}

func classExtensions() map[string]classExt {
	return map[string]classExt{
		"Guerreiro": {branchDef{"SANGUE", 0, elemPhys},
			[4]nodeSpec{pVigor, active(7), pLeech, active(8)},
			[]skillDef{
				{"Execução", "esmagar", "Um golpe de cima para baixo num arco curto (16 de alcance) que mira quem já está ferido: dano 80–120, e o dobro em alvos com menos de 30% de vida.", elemPhys, 20, 6 * time.Second, 2, func(c *cast) {
					c.fxAt(fxSlash, c.x, c.y, 14, 300)
					c.fxAt(fxCrack, c.x+c.cos*8, c.y+c.sin*8, 6, 500)
					for _, m := range c.mobsNear(c.x, c.y, 18) {
						pct := 0
						if m.HP*100 < m.MaxHP*30 {
							pct = 100
						}
						c.hit(80, 41, func(x, y, pad float64) bool { return x == m.X && y == m.Y }, hitOpt{Pct: pct})
					}
				}},
				{"Salto Devastador", "dash", "Salta até 30 células na mira e cai com o peso de tudo: dano 60–90 e atordoamento de 1,5 s num raio de 18 em volta de onde cair.", elemPhys, 22, 9 * time.Second, 2, func(c *cast) {
					x0, y0, x, y := c.wd.slide(c.p, c.dir, 30, c.now)
					c.fxDir(fxThrust, x0, y0, dist(x0, y0, x, y), c.dir, 300)
					c.fxAt(fxBlast, x, y, 18, 500)
					c.wd.strikeOpt(c.id, c.now, 60, 31, around(x, y, 18), hitOpt{Stun: 1.5})
					c.wd.enter(c.p)
				}},
				{"Fúria do Titã", "horn", "O golpe final do guerreiro. Por 10 s: +30% de dano, só 30% do dano recebido passa, e uma onda de choque sai de você na hora (raio 34, dano 120–180).", elemPhys, 40, 90 * time.Second, 2, func(c *cast) {
					c.fxDir(fxBuff, c.x, c.y, 40, 0, 1000)
					c.fxAt(fxBlast, c.x, c.y, 34, 600)
					c.p.dmgUntil = c.now.Add(10 * time.Second)
					guardFor(c.p, c.now, 10*time.Second)
					c.hit(120, 61, around(c.x, c.y, 34), hitOpt{})
				}},
			}, 18, 14},
		"Cavaleiro": {branchDef{"HONRA", 0, elemHoly},
			[4]nodeSpec{pVigor, active(7), pSkin, active(8)},
			[]skillDef{
				{"Juramento", "cross", "Renova o voto: recupera 25% da vida máxima e fica protegido por 3 s (só 30% do dano passa).", elemHoly, 18, 14 * time.Second, 1, func(c *cast) {
					c.fxAt(fxHeal, c.x, c.y, 14, 600)
					c.p.HP = min(c.p.MaxHP, c.p.HP+c.p.MaxHP/4)
					guardFor(c.p, c.now, 3*time.Second)
				}},
				{"Lança Trovejante", "spear", "Arremessa a lança em linha reta (70 células): atravessa tudo, dano 60–90 e atordoamento de 1 s.", elemBolt, 20, 7 * time.Second, 2, func(c *cast) {
					ex, ey := c.ahead(70)
					c.fxDir(fxChain, c.x, c.y-14, 70, c.dir, 400)
					c.hit(60, 31, along(c.x, c.y, ex, ey, 5), hitOpt{Elem: elemBolt, Stun: 1})
				}},
				{"Carga dos Mil Cascos", "dash", "O golpe final do cavaleiro. Uma carga de 70 células que nada para: dano 200–300 e atordoamento de 2 s em tudo no caminho.", elemPhys, 45, 90 * time.Second, 2, func(c *cast) {
					c.wd.dash(c.p, c.dir, 70, 200, 101, c.now, hitOpt{Stun: 2})
				}},
			}, 18, 14},
		"Paladino": {branchDef{"JUSTIÇA", 0, elemHoly},
			[4]nodeSpec{pVigor, active(7), pFocus, active(8)},
			[]skillDef{
				{"Espada da Fé", "sword", "Uma lâmina de luz corta em linha reta (45 células): dano 55–80 e cura você em 15.", elemHoly, 16, 4 * time.Second, 1, func(c *cast) {
					ex, ey := c.ahead(45)
					c.fxAt(fxThrust, c.x, c.y, 45, 300)
					c.hit(55, 26, along(c.x, c.y, ex, ey, 6), hitOpt{Elem: elemHoly})
					c.p.HP = min(c.p.MaxHP, c.p.HP+15)
				}},
				{"Escudo Divino", "halo", "Uma bolha de luz em quem estiver a até 40 células: 6 s de proteção (só 30% do dano passa) e 30 de cura.", elemHoly, 30, 20 * time.Second, 1, func(c *cast) {
					c.fxDir(fxBuff, c.x, c.y, 40, 1, 900)
					c.buffAllies(40, func(q *Player) { guardFor(q, c.now, 6*time.Second); q.HP = min(q.MaxHP, q.HP+30) })
				}},
				{"Ira dos Céus", "hammer", "O golpe final do paladino. Cinco martelos de luz caem em volta da mira, um a cada 0,3 s (raio 16 cada): dano 110–160 e atordoamento de 1 s em cada impacto.", elemHoly, 45, 90 * time.Second, 2, func(c *cast) {
					tx, ty := c.ahead(36)
					for k := 0; k < 5; k++ {
						a := float64(k) * 2 * math.Pi / 5
						hx, hy := tx+math.Cos(a)*18, ty+math.Sin(a)*12
						after := time.Duration(600+300*k) * time.Millisecond
						c.fxAt(fxHammer, hx, hy, 16, int(after/time.Millisecond))
						c.blastAt(after, hx, hy, 16, fxHoly, 110, 51, hitOpt{Elem: elemHoly, Stun: 1})
					}
				}},
			}, 18, 14},
		"Arqueiro": {branchDef{"CAÇA", 0, elemPoison},
			[4]nodeSpec{pCrit, active(7), pBreath, active(8)},
			[]skillDef{
				{"Armadilha de Espinhos", "crack", "Espalha espinhos onde você está (raio 22) por 6 s: a cada meio segundo, dano 8–12 e lentidão em quem pisar.", elemPoison, 15, 10 * time.Second, 1, func(c *cast) {
					cx, cy := c.x, c.y
					c.hazardFx(fxPlague, cx, cy, 22, 0, 6*time.Second)
					c.everyFor(6*time.Second, 500*ms, func(now time.Time) {
						c.wd.strikeOpt(c.id, now, 8, 5, around(cx, cy, 22), hitOpt{Elem: elemPoison, Slow: 1})
					})
				}},
				{"Flecha Perfurante", "arrow", "Uma flecha que atravessa tudo em linha reta (120 células): dano 70–100 em cada alvo.", elemPhys, 18, 5 * time.Second, 1, func(c *cast) {
					ex, ey := c.ahead(120)
					c.fxDir(fxChain, c.x, c.y-14, 120, c.dir, 300)
					c.hit(70, 31, along(c.x, c.y, ex, ey, 3), hitOpt{})
				}},
				{"Tempestade de Flechas", "rain", "O golpe final do arqueiro. Por 4 s, flechas cobrem uma área de raio 40 a 50 células na mira: a cada 0,25 s, dano 25–40 em tudo dentro.", elemPhys, 45, 90 * time.Second, 2, func(c *cast) {
					tx, ty := c.ahead(50)
					c.hazardFx(fxRain, tx, ty, 40, 0, 4*time.Second)
					c.everyFor(4*time.Second, 250*ms, func(now time.Time) {
						c.wd.strikeOpt(c.id, now, 25, 16, around(tx, ty, 40), hitOpt{})
					})
				}},
			}, 18, 14},
		"Ladino": {branchDef{"ASSASSINO", 0, elemShadow},
			[4]nodeSpec{pCrit, active(7), pLeech, active(8)},
			[]skillDef{
				{"Adagas Arremessadas", "fan", "Três adagas em leque (90 de alcance). Cada uma: dano 25–40.", elemPhys, 10, 3 * time.Second, 1, func(c *cast) {
					for _, off := range []float64{-0.2, 0, 0.2} {
						c.shootOpt(projArrow, 6, 90, off, 25, 16, 3, hitOpt{})
					}
				}},
				{"Sumiço", "smoke", "Some numa nuvem de fumaça: recua 25 células, fica protegido por 2 s e ganha +25% de crítico por 5 s.", elemShadow, 15, 12 * time.Second, 1, func(c *cast) {
					c.fxAt(fxSmoke, c.x, c.y, 14, 900)
					c.wd.blink(c.p, c.dir+math.Pi, 25, c.now)
					guardFor(c.p, c.now, 2*time.Second)
					c.p.critUntil = c.now.Add(5 * time.Second)
				}},
				{"Mil Cortes", "spiral", "O golpe final do ladino. Por 1,6 s você vira um borrão de lâminas: a cada 0,2 s, cortes em tudo num raio de 26 (dano 30–50, +30% de crítico).", elemShadow, 40, 90 * time.Second, 2, func(c *cast) {
					guardFor(c.p, c.now, 1600*ms)
					p := c.p
					c.everyFor(1600*ms, 200*ms, func(now time.Time) {
						x, y := float64(p.X), float64(p.Y)
						c.wd.effects = append(c.wd.effects, Effect{Kind: fxSpin, X: x, Y: y, R: 26, Start: now, Dur: 200 * ms})
						c.wd.strikeOpt(c.id, now, 30, 21, around(x, y, 26), hitOpt{Elem: elemShadow, Crit: 30})
					})
				}},
			}, 18, 14},
		"Necromante": {branchDef{"MORTOS", 0, elemShadow},
			[4]nodeSpec{
				passive("Ossos Firmes", "Seus servos duram mais e aguentam mais: +30% de vida e +5 s de duração por grau.", "bone", elemShadow, 3, bonus{Minion: 1}),
				active(7), pVigor, active(8)},
			[]skillDef{
				{"Erguer Mortos", "skull", "Ergue 2 esqueletos da terra em volta de você por 25 s. Eles seguem você, atacam o inimigo mais próximo (dano 15–25) e atraem os golpes dos inimigos. No máximo 8 servos por vez.", elemShadow, 25, 10 * time.Second, 2, func(c *cast) {
					hp, life := c.p.minionHP(120), c.p.minionLife(25*time.Second)
					c.wd.raise(c.id, c.x, c.y, 2, life, mobSkeleton, hp, 15, 11, c.now)
				}},
				{"Colheita de Almas", "drop", "Arranca a vida de todos os inimigos num raio de 32: dano 35–55 em cada um, e você recupera 12 de vida por inimigo atingido.", elemShadow, 22, 7 * time.Second, 2, func(c *cast) {
					c.fxAt(fxDark, c.x, c.y, 32, 700)
					n := len(c.mobsNear(c.x, c.y, 32))
					c.hit(35, 21, around(c.x, c.y, 32), hitOpt{Elem: elemShadow})
					c.p.HP = min(c.p.MaxHP, c.p.HP+12*n)
					if n > 0 {
						c.fxAt(fxHeal, c.x, c.y, 10, 500)
					}
				}},
				{"Legião dos Mortos", "hand", "O golpe final do necromante. A terra se abre e 6 guerreiros esqueletos se erguem por 30 s (dano 30–45 cada), enquanto uma onda de morte drena todos os inimigos num raio de 40 (dano 80–120).", elemShadow, 60, 90 * time.Second, 2, func(c *cast) {
					c.fxAt(fxGrave, c.x, c.y, 30, 1200)
					c.fxAt(fxDark, c.x, c.y, 40, 800)
					hp, life := c.p.minionHP(260), c.p.minionLife(30*time.Second)
					c.wd.raise(c.id, c.x, c.y, 6, life, mobWarrior, hp, 30, 16, c.now)
					c.hit(80, 41, around(c.x, c.y, 40), hitOpt{Elem: elemShadow})
				}},
			}, 18, 14},
	}
}

func (p *Player) minionHP(base int) int { return base * (10 + 3*p.bon.Minion) / 10 }
func (p *Player) minionLife(d time.Duration) time.Duration {
	return d + time.Duration(p.bon.Minion)*5*time.Second
}

// extendClasses põe os ramos novos e os golpes finais nas classes.
func extendClasses() {
	ext := classExtensions()
	for i := range classes {
		c := &classes[i]
		e, ok := ext[c.Name]
		if !ok {
			continue
		}
		c.Skills = append(c.Skills, e.skills...)
		c.Branches = append(c.Branches, e.branch)
		for k := range c.Branches {
			c.Branches[k].Ang = branchAngles[k]
		}
		webSpecs[i] = append(webSpecs[i], e.spec)
	}
}

// capstoneOf diz qual golpe é o final de cada classe (o último do livro).
func capstoneOf(c *class) (skill, lv, attr int) {
	if e, ok := classExtensions()[c.Name]; ok {
		return len(c.Skills) - 1, e.capLv, e.capAt
	}
	if c.Name == "Mago" {
		return 11, 18, 14 // Obliterar
	}
	return len(c.Skills) - 1, 18, 14
}
