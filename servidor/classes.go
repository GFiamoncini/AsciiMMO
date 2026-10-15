package main

import (
	"math"
	"time"
)

// class define vida, mana, o atributo da classe (que a teia pede), o livro de
// golpes e a teia de habilidades.
type class struct {
	Name     string
	HP, MP   int
	Magic    bool // o dano mágico das peças vale para a classe
	Attr     int  // atributo da classe: a teia pede pontos nele
	anyAttr  bool // a teia aceita o atributo mais alto (Depravado)
	Skills   []skillDef
	Branches []branchDef
	Web      []webNode
	Adj      [][]int
	techBase int // índice do livro onde começam as técnicas (techniques.go)
}

const ms = time.Millisecond

// guardFor protege (só passa 30% do dano) por d.
func guardFor(q *Player, now time.Time, d time.Duration) {
	if t := now.Add(d); t.After(q.guardUntil) {
		q.guardUntil = t
	}
}

var classes []class

// classBook monta o livro de cada classe (fica numa função para os golpes
// poderem usar o resto do jogo sem ciclo de inicialização).
func classBook() []class {
	return []class{
		// ---------------- Guerreiro ----------------
		{Name: "Guerreiro", HP: 150, MP: 40, Attr: attrStr, Skills: []skillDef{
			{"Golpe", "espada", "Golpe básico de espada num arco à frente (17 de alcance). Dano 25–45, mais 4 por nível; sem custo.", elemPhys, 0, 350 * ms, 0, func(c *cast) {
				c.fxAt(fxSlash, c.x, c.y, 13, 200)
				c.hit(25, 21, arc(c.x, c.y, c.dir, 17, 1.3), hitOpt{})
			}},
			{"Investida", "investida", "Avança até 26 células na mira e corta tudo no caminho. Dano 40–60. Bom para fechar distância ou fugir.", elemPhys, 10, 3 * time.Second, 1, func(c *cast) {
				c.wd.dash(c.p, c.dir, 26, 40, 21, c.now, hitOpt{})
			}},
			{"Redemoinho", "redemoinho", "Gira a espada e fere todos em volta (raio 24). Dano 70–110. O golpe que limpa grupos.", elemPhys, 15, 5 * time.Second, 2, func(c *cast) {
				c.fxAt(fxSpin, c.x, c.y, 24, 500)
				c.hit(70, 41, around(c.x, c.y, 24), hitOpt{})
			}},
			{"Golpe Esmagador", "esmagar", "Um golpe pesado num arco estreito (18 de alcance) que atordoa por 1,2 s. Dano 60–90. Chefes não ficam atordoados.", elemPhys, 18, 6 * time.Second, 2, func(c *cast) {
				c.fxAt(fxSlash, c.x, c.y, 15, 300)
				c.fxAt(fxCrack, c.x+c.cos*10, c.y+c.sin*10, 8, 500)
				c.hit(60, 31, arc(c.x, c.y, c.dir, 18, 0.9), hitOpt{Stun: 1.2})
			}},
			{"Grito de Guerra", "grito", "Um urro que inflama você e os aliados a até 40 células: +30% de dano por 8 s.", elemPhys, 15, 15 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 40, 0, 800)
				c.buffAllies(40, func(q *Player) { q.dmgUntil = c.now.Add(8 * time.Second) })
			}},
			{"Segundo Fôlego", "folego2", "Respira fundo no meio da luta e recupera 30% da vida máxima de uma vez.", elemHoly, 20, 20 * time.Second, 1, func(c *cast) {
				c.fxAt(fxHeal, c.x, c.y, 14, 600)
				c.p.HP = min(c.p.MaxHP, c.p.HP+c.p.MaxHP*30/100)
			}},
			{"Fenda", "fenda", "Crava a espada no chão e abre uma fenda em linha reta (50 células). Dano 45–65 e lentidão de 2 s em quem estiver nela.", elemPhys, 20, 7 * time.Second, 2, func(c *cast) {
				ex, ey := c.ahead(50)
				c.fxAt(fxCrack, c.x, c.y, 50, 700)
				c.hit(45, 21, along(c.x, c.y, ex, ey, 6), hitOpt{Slow: 2})
			}},
		}, Branches: []branchDef{{"FÚRIA", -90, elemPhys}, {"MURALHA", 30, elemHoly}, {"TÉCNICA", 150, elemArcane}}},

		// ---------------- Mago ----------------
		{Name: "Mago", HP: 85, MP: 110, Magic: true, Attr: attrMana, Skills: []skillDef{
			{"Míssil Arcano", "missil", "Dispara um projétil arcano rápido (95 de alcance) que explode no primeiro alvo. Dano 20–32, mais 4 por nível; sem custo.", elemArcane, 0, 450 * ms, 0, func(c *cast) {
				c.shoot(projMissile, 4, 95, 0)
			}},
			{"Bola de Fogo", "fogo", "Uma bola de fogo lenta (105 de alcance) que explode numa área de raio 16. Dano 55–85 e põe fogo no capim.", elemFire, 15, 2 * time.Second, 1, func(c *cast) {
				c.shoot(projFireball, 3, 105, 0)
			}},
			{"Meteoro", "meteoro", "Chama um meteoro a 45 células na mira. Depois de 0,7 s de aviso ele cai numa área de raio 26. Dano 140–200 e incêndio grande.", elemFire, 30, 6 * time.Second, 2, func(c *cast) {
				tx, ty := c.ahead(45)
				c.fxAt(fxMeteor, tx, ty, 26, 700)
				c.blastAt(700*ms, tx, ty, 26, fxBoom, 140, 61, hitOpt{Elem: elemFire, Ignite: 10})
			}},
			{"Muralha de Chamas", "muralha", "Ergue uma muralha de fogo de 36 células atravessada na mira, a 28 de distância. Por 4 s, queima quem estiver nela a cada meio segundo (14–22) e acende o capim.", elemFire, 22, 8 * time.Second, 2, func(c *cast) {
				cx, cy := c.ahead(28)
				pd := c.dir + math.Pi/2
				ax, ay, bx, by := cx-math.Cos(pd)*18, cy-math.Sin(pd)*18, cx+math.Cos(pd)*18, cy+math.Sin(pd)*18
				c.hazardFx(fxFlame, cx, cy, 18, pd, 4*time.Second)
				c.wd.ignite(cx, cy, 8, c.now)
				c.everyFor(4*time.Second, 500*ms, func(now time.Time) {
					c.wd.strikeOpt(c.id, now, 14, 9, along(ax, ay, bx, by, 4), hitOpt{Elem: elemFire})
				})
			}},
			{"Lança de Gelo", "gelo", "Atira uma lança de gelo veloz (100 de alcance). Dano 30–45 e lentidão de 2,5 s no alvo.", elemIce, 12, 1500 * ms, 1, func(c *cast) {
				c.shootOpt(projIce, 4.5, 100, 0, 30, 16, 5, hitOpt{Elem: elemIce, Slow: 2.5})
			}},
			{"Nova Congelante", "nova", "Uma onda de frio sai de você (raio 26) e congela todos por 2 s. Dano 40–60. Congelado não anda nem ataca.", elemIce, 25, 7 * time.Second, 2, func(c *cast) {
				c.fxAt(fxFrost, c.x, c.y, 26, 700)
				c.hit(40, 21, around(c.x, c.y, 26), hitOpt{Elem: elemIce, Stun: 2})
			}},
			{"Nevasca", "nevasca", "Uma nevasca cobre uma área de raio 30 a 40 células na mira por 5 s: a cada meio segundo, dano 12–18 e lentidão em quem estiver dentro.", elemIce, 40, 12 * time.Second, 2, func(c *cast) {
				tx, ty := c.ahead(40)
				c.hazardFx(fxBlizzard, tx, ty, 30, 0, 5*time.Second)
				c.everyFor(5*time.Second, 500*ms, func(now time.Time) {
					c.wd.strikeOpt(c.id, now, 12, 7, around(tx, ty, 30), hitOpt{Elem: elemIce, Slow: 1})
				})
			}},
			{"Raio", "raio", "Um raio salta do seu cajado para o inimigo mais perto na mira (até 70) e pula para mais 3 alvos próximos, perdendo 20% a cada salto. Dano 35–55.", elemBolt, 14, 2500 * ms, 1, func(c *cast) {
				c.wd.chain(c, 35, 21, 3+c.p.bon.Chain)
			}},
			{"Tempestade", "tempestade", "Uma tempestade sobre a área a 35 células na mira (raio 45) por 4 s: a cada 0,4 s um raio cai num inimigo dentro dela. Dano 40–60 por raio.", elemBolt, 35, 10 * time.Second, 2, func(c *cast) {
				tx, ty := c.ahead(35)
				c.hazardFx(fxStorm, tx, ty, 45, 0, 4*time.Second)
				c.everyFor(4*time.Second, 400*ms, func(now time.Time) {
					targets := c.mobsNear(tx, ty, 45)
					zx, zy := tx+float64(c.wd.rng.Intn(61)-30), ty+float64(c.wd.rng.Intn(41)-20)
					if len(targets) > 0 {
						m := targets[c.wd.rng.Intn(len(targets))]
						zx, zy = m.X, m.Y
					}
					c.wd.effects = append(c.wd.effects, Effect{Kind: fxZap, X: zx, Y: zy, R: 6, Start: now, Dur: 300 * ms})
					c.wd.strikeOpt(c.id, now, 40, 21, around(zx, zy, 5), hitOpt{Elem: elemBolt})
				})
			}},
			{"Piscar", "piscar", "Some num lampejo e reaparece até 40 células na mira, atravessando inimigos (não paredes). Não causa dano.", elemArcane, 15, 5 * time.Second, 1, func(c *cast) {
				c.wd.blink(c.p, c.dir, 40, c.now)
			}},
			{"Escudo Arcano", "escudo", "Uma bolha arcana envolve você por 5 s: só 30% do dano recebido passa.", elemArcane, 20, 14 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 10, 3, 800)
				guardFor(c.p, c.now, 5*time.Second)
			}},
			{"Obliterar", "obliterar", "O golpe proibido. Cinco selos arcanos se juntam em volta de você por 1,2 s (você fica parado e protegido) e então um feixe colossal de 130 células varre a mira com 28 de largura: dano 500–800 e atordoamento de 3 s em tudo que tocar. Gasta toda a mana que restar além do custo.", elemArcane, 80, 90 * time.Second, 2, func(c *cast) {
				c.p.MP = 0
				guardFor(c.p, c.now, 1300*ms)
				c.fxDir(fxSeal, c.x, c.y, 26, c.dir, 1200)
				ex, ey := c.ahead(130)
				x, y, dir, id := c.x, c.y, c.dir, c.id
				c.wd.hazards = append(c.wd.hazards, hazard{until: c.now.Add(1250 * ms), next: c.now.Add(1200 * ms), every: time.Second, tick: func(now time.Time) {
					c.wd.effects = append(c.wd.effects, Effect{Kind: fxOblit, X: x, Y: y, R: 130, Dir: dir, Start: now, Dur: 900 * ms})
					c.wd.strikeOpt(id, now, 500, 301, along(x, y, ex, ey, 14), hitOpt{Elem: elemArcane, Stun: 3})
				}})
			}},
		}, Branches: []branchDef{{"FOGO", -135, elemFire}, {"GELO", -45, elemIce}, {"RAIO", 45, elemBolt}, {"ARCANO", 135, elemArcane}}},

		// ---------------- Cavaleiro ----------------
		{Name: "Cavaleiro", HP: 180, MP: 40, Attr: attrVit, Skills: []skillDef{
			{"Estocada", "lanca", "Estoca com a lança em linha reta (26 de alcance). Dano 28–44, mais 4 por nível; sem custo.", elemPhys, 0, 450 * ms, 0, func(c *cast) {
				c.fxAt(fxThrust, c.x, c.y, 26, 200)
				ex, ey := c.ahead(26)
				c.hit(28, 17, along(c.x, c.y, ex, ey, 5), hitOpt{})
			}},
			{"Bloqueio", "bloqueio", "Ergue o escudo por 4 s: só 30% do dano recebido passa.", elemPhys, 10, 6 * time.Second, 1, func(c *cast) {
				guardFor(c.p, c.now, 4*time.Second)
			}},
			{"Carga", "carga", "Dispara em carga por até 46 células, ferindo tudo no caminho. Dano 70–100.", elemPhys, 15, 5 * time.Second, 2, func(c *cast) {
				c.wd.dash(c.p, c.dir, 46, 70, 31, c.now, hitOpt{})
			}},
			{"Lança Giratória", "giro", "Gira a lança em volta do corpo (raio 22). Dano 50–75.", elemPhys, 15, 5 * time.Second, 2, func(c *cast) {
				c.fxAt(fxSpin, c.x, c.y, 22, 450)
				c.hit(50, 26, around(c.x, c.y, 22), hitOpt{})
			}},
			{"Golpe de Escudo", "escudada", "Bate com o escudo à frente (14 de alcance) e atordoa por 1,5 s. Dano 30–45.", elemPhys, 12, 6 * time.Second, 1, func(c *cast) {
				c.fxAt(fxSlash, c.x, c.y, 11, 250)
				c.hit(30, 16, arc(c.x, c.y, c.dir, 14, 1.0), hitOpt{Stun: 1.5})
			}},
			{"Bastião", "bastiao", "Planta o estandarte: você e os aliados a até 40 células ficam protegidos por 5 s (só 30% do dano passa).", elemHoly, 25, 18 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 40, 1, 900)
				c.buffAllies(40, func(q *Player) { guardFor(q, c.now, 5*time.Second) })
			}},
			{"Investida Real", "real", "Uma carga curta (34 células) que atordoa por 1 s quem for atingido. Dano 55–80.", elemPhys, 20, 8 * time.Second, 2, func(c *cast) {
				c.wd.dash(c.p, c.dir, 34, 55, 26, c.now, hitOpt{Stun: 1})
			}},
		}, Branches: []branchDef{{"LANÇA", -90, elemPhys}, {"ESCUDO", 30, elemHoly}, {"CAVALARIA", 150, elemArcane}}},

		// ---------------- Paladino ----------------
		{Name: "Paladino", HP: 130, MP: 70, Magic: true, Attr: attrMana, Skills: []skillDef{
			{"Martelo", "martelo", "Golpe de martelo num arco à frente (17 de alcance). Dano 24–42, mais 4 por nível; sem custo.", elemPhys, 0, 400 * ms, 0, func(c *cast) {
				c.fxAt(fxSlash, c.x, c.y, 13, 250)
				c.hit(24, 19, arc(c.x, c.y, c.dir, 17, 1.5), hitOpt{})
			}},
			{"Cura", "cura", "Cura 45 de vida em você e em quem estiver a até 40 células.", elemHoly, 20, 5 * time.Second, 1, func(c *cast) {
				c.fxAt(fxHeal, c.x, c.y, 40, 600)
				c.wd.heal(c.x, c.y, 40, 45)
			}},
			{"Julgamento", "julgamento", "A luz cai em volta de você (raio 30): dano 90–130 nos inimigos e 25 de cura nos aliados.", elemHoly, 30, 7 * time.Second, 2, func(c *cast) {
				c.fxAt(fxHoly, c.x, c.y, 30, 600)
				c.hit(90, 41, around(c.x, c.y, 30), hitOpt{Elem: elemHoly})
				c.wd.heal(c.x, c.y, 30, 25)
			}},
			{"Luz Sagrada", "luz", "Arremessa uma esfera de luz (100 de alcance) que explode no primeiro alvo (raio 7). Dano 35–52.", elemHoly, 12, 2 * time.Second, 1, func(c *cast) {
				c.shootOpt(projHoly, 4, 100, 0, 35, 18, 7, hitOpt{Elem: elemHoly})
			}},
			{"Aura Protetora", "aura", "Uma aura dourada protege você e os aliados a até 40 células por 5 s (só 30% do dano passa) e cura 15 na hora.", elemHoly, 25, 18 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 40, 1, 900)
				c.buffAllies(40, func(q *Player) { guardFor(q, c.now, 5*time.Second); q.HP = min(q.MaxHP, q.HP+15) })
			}},
			{"Consagrar", "consagrar", "Consagra o chão onde você está (raio 26) por 4 s: a cada meio segundo, 10–15 de dano nos inimigos e 4 de cura nos aliados dentro.", elemHoly, 30, 10 * time.Second, 2, func(c *cast) {
				cx, cy := c.x, c.y
				c.hazardFx(fxConsecrate, cx, cy, 26, 0, 4*time.Second)
				c.everyFor(4*time.Second, 500*ms, func(now time.Time) {
					c.wd.strikeOpt(c.id, now, 10, 6, around(cx, cy, 26), hitOpt{Elem: elemHoly})
					c.wd.heal(cx, cy, 26, 4)
				})
			}},
			{"Martelo do Céu", "ceu", "Um martelo de luz cai a 40 células na mira depois de 0,6 s (raio 16). Dano 90–130 e atordoamento de 1,5 s.", elemHoly, 28, 8 * time.Second, 2, func(c *cast) {
				tx, ty := c.ahead(40)
				c.fxAt(fxHammer, tx, ty, 16, 600)
				c.blastAt(600*ms, tx, ty, 16, fxHoly, 90, 41, hitOpt{Elem: elemHoly, Stun: 1.5})
			}},
		}, Branches: []branchDef{{"MARTELO", -90, elemPhys}, {"FÉ", 30, elemHoly}, {"LUZ", 150, elemHoly}}},

		// ---------------- Arqueiro ----------------
		{Name: "Arqueiro", HP: 110, MP: 60, Attr: attrSta, Skills: []skillDef{
			{"Flecha", "flecha", "Uma flecha rápida (110 de alcance). Dano 22–36, mais 4 por nível; sem custo.", elemPhys, 0, 400 * ms, 0, func(c *cast) {
				c.shoot(projArrow, 6, 110, 0)
			}},
			{"Rajada", "rajada", "Três flechas em leque. Cada uma: dano 22–36.", elemPhys, 12, 3 * time.Second, 1, func(c *cast) {
				for _, off := range []float64{-0.18, 0, 0.18} {
					c.shoot(projArrow, 6, 100, off)
				}
			}},
			{"Chuva de Flechas", "chuva", "Uma chuva de flechas cai a 50 células na mira depois de 0,6 s (raio 22). Dano 80–120.", elemPhys, 25, 6 * time.Second, 2, func(c *cast) {
				tx, ty := c.ahead(50)
				c.fxAt(fxRain, tx, ty, 22, 600)
				c.blastAt(600*ms, tx, ty, 22, fxSpin, 80, 41, hitOpt{})
			}},
			{"Flecha Congelante", "flechagelo", "Uma flecha de ponta gelada: dano 30–45 e lentidão de 3 s.", elemIce, 10, 3 * time.Second, 1, func(c *cast) {
				c.shootOpt(projArrow, 6, 110, 0, 30, 16, 4, hitOpt{Elem: elemIce, Slow: 3})
			}},
			{"Rolamento", "rolamento", "Rola para trás até 20 células, para longe da mira. Não causa dano; ótimo para manter distância.", elemPhys, 8, 4 * time.Second, 1, func(c *cast) {
				c.wd.blink(c.p, c.dir+math.Pi, 20, c.now)
			}},
			{"Flecha Explosiva", "flechafogo", "Uma flecha com pólvora que explode no impacto (raio 14). Dano 55–85 e fogo no capim.", elemFire, 18, 6 * time.Second, 2, func(c *cast) {
				c.shootOpt(projArrow, 5, 110, 0, 55, 31, 14, hitOpt{Elem: elemFire, Ignite: 6})
			}},
			{"Olho de Águia", "olho", "Mira perfeita por 8 s: +25% de chance de crítico em todos os golpes.", elemPhys, 15, 16 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 10, 4, 800)
				c.p.critUntil = c.now.Add(8 * time.Second)
			}},
		}, Branches: []branchDef{{"PONTARIA", -90, elemPhys}, {"SOBREVIVÊNCIA", 30, elemHoly}, {"FLECHAS", 150, elemFire}}},

		// ---------------- Ladino ----------------
		{Name: "Ladino", HP: 115, MP: 50, Attr: attrSta, Skills: []skillDef{
			{"Corte", "adaga", "Um corte rápido e curto (15 de alcance). Dano 18–30, mais 4 por nível; sem custo. O ladino tem +10% de crítico.", elemPhys, 0, 250 * ms, 0, func(c *cast) {
				c.fxAt(fxSlash, c.x, c.y, 11, 150)
				c.hit(18, 13, arc(c.x, c.y, c.dir, 15, 1.2), hitOpt{})
			}},
			{"Passo Sombrio", "passo", "Avança pelas sombras até 30 células, cortando quem estiver no caminho. Dano 45–70.", elemShadow, 10, 3 * time.Second, 1, func(c *cast) {
				c.wd.dash(c.p, c.dir, 30, 45, 26, c.now, hitOpt{Elem: elemShadow})
			}},
			{"Leque de Lâminas", "leque", "Lâminas em volta de você (raio 22). Dano 60–95.", elemPhys, 15, 5 * time.Second, 2, func(c *cast) {
				c.fxAt(fxSpin, c.x, c.y, 22, 400)
				c.hit(60, 36, around(c.x, c.y, 22), hitOpt{})
			}},
			{"Apunhalar", "apunhalar", "Uma punhalada certeira bem de perto (10 de alcance): dano 70–100 e +35% de chance de crítico.", elemPhys, 12, 4 * time.Second, 1, func(c *cast) {
				c.fxAt(fxThrust, c.x, c.y, 10, 200)
				c.hit(70, 31, arc(c.x, c.y, c.dir, 10, 0.6), hitOpt{Crit: 35})
			}},
			{"Bomba de Fumaça", "fumaca", "Uma nuvem de fumaça em volta (raio 24) que deixa os inimigos lentos por 3 s. Dano leve (5–9).", elemShadow, 15, 10 * time.Second, 1, func(c *cast) {
				c.fxAt(fxSmoke, c.x, c.y, 24, 1200)
				c.hit(5, 5, around(c.x, c.y, 24), hitOpt{Elem: elemShadow, Slow: 3})
			}},
			{"Lâminas Envenenadas", "veneno", "Passa veneno nas lâminas por 10 s: cada golpe físico envenena (8 de dano por segundo por 4 s).", elemPoison, 15, 16 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 10, 2, 800)
				c.p.venomUntil = c.now.Add(10 * time.Second)
			}},
			{"Dança das Lâminas", "danca", "Um turbilhão de cortes em volta (raio 26). Dano 80–120.", elemPhys, 25, 9 * time.Second, 2, func(c *cast) {
				c.fxAt(fxSpin, c.x, c.y, 26, 600)
				c.fxAt(fxSpin, c.x, c.y, 18, 400)
				c.hit(80, 41, around(c.x, c.y, 26), hitOpt{})
			}},
		}, Branches: []branchDef{{"LÂMINAS", -90, elemPhys}, {"SOMBRAS", 30, elemShadow}, {"VENENOS", 150, elemPoison}}},

		// ---------------- Necromante ----------------
		{Name: "Necromante", HP: 90, MP: 100, Magic: true, Attr: attrMana, Skills: []skillDef{
			{"Seta Sombria", "seta", "Uma seta de sombra (95 de alcance) que explode no primeiro alvo. Dano 20–33, mais 4 por nível; sem custo.", elemShadow, 0, 450 * ms, 0, func(c *cast) {
				c.shoot(projShadow, 3.5, 95, 0)
			}},
			{"Dreno", "dreno", "Suga a vida em linha reta (40 células): dano 40–60 e você recupera 25 de vida.", elemShadow, 15, 3 * time.Second, 1, func(c *cast) {
				c.fxAt(fxThrust, c.x, c.y, 40, 300)
				ex, ey := c.ahead(40)
				c.hit(40, 21, along(c.x, c.y, ex, ey, 5), hitOpt{Elem: elemShadow})
				c.p.HP = min(c.p.MaxHP, c.p.HP+25)
				c.fxAt(fxHeal, c.x, c.y, 8, 400)
			}},
			{"Nova Sombria", "novasombra", "Uma explosão de sombra em volta (raio 30). Dano 110–160.", elemShadow, 30, 6 * time.Second, 2, func(c *cast) {
				c.fxAt(fxDark, c.x, c.y, 30, 600)
				c.hit(110, 51, around(c.x, c.y, 30), hitOpt{Elem: elemShadow})
			}},
			{"Lança de Osso", "osso", "Uma lança de osso atravessa tudo em linha reta (50 células). Dano 45–65 em cada alvo.", elemPhys, 14, 3 * time.Second, 1, func(c *cast) {
				ex, ey := c.ahead(50)
				c.fxAt(fxBone, c.x, c.y, 50, 400)
				c.hit(45, 21, along(c.x, c.y, ex, ey, 4), hitOpt{})
			}},
			{"Praga", "praga", "Uma nuvem de praga em volta (raio 28): dano 10–15 na hora e veneno de 8 por segundo por 6 s.", elemPoison, 20, 8 * time.Second, 2, func(c *cast) {
				c.fxAt(fxPlague, c.x, c.y, 28, 900)
				c.hit(10, 6, around(c.x, c.y, 28), hitOpt{Elem: elemPoison, Dot: 8, DotFor: 6})
			}},
			{"Pacto de Sangue", "pacto", "Troca sangue por poder: perde 15% da vida máxima e ganha 35% da mana máxima. Não mata (fica com 1 de vida no mínimo).", elemShadow, 0, 12 * time.Second, 1, func(c *cast) {
				c.fxDir(fxBuff, c.x, c.y, 10, 5, 700)
				c.p.HP = max(1, c.p.HP-c.p.MaxHP*15/100)
				c.p.MP = min(c.p.MaxMP, c.p.MP+c.p.MaxMP*35/100)
			}},
			{"Mão da Cova", "mao", "Mãos de ossos saem do chão a 40 células na mira depois de 0,5 s (raio 14): dano 70–100 e prendem por 2 s.", elemShadow, 28, 9 * time.Second, 2, func(c *cast) {
				tx, ty := c.ahead(40)
				c.fxAt(fxGrave, tx, ty, 14, 900)
				c.blastAt(500*ms, tx, ty, 14, fxDark, 70, 31, hitOpt{Elem: elemShadow, Stun: 2})
			}},
		}, Branches: []branchDef{{"MORTE", -90, elemShadow}, {"SANGUE", 30, elemShadow}, {"PRAGA", 150, elemPoison}}},
	}
}

// webSpecs: os quatro anéis de cada ramo, por classe, na ordem de Branches.
var webSpecs = [][][4]nodeSpec{
	{ // Guerreiro
		{pSharp, active(3), crypt(2), pMastery},
		{pSkin, active(5), pLeech, active(4)},
		{pFocus, pBreath, active(6), pCrit},
	},
	{ // Mago
		{passive("Brasa Viva", "O fogo pega mais fundo: +8% de dano de fogo por grau.", "brasa", elemFire, 3, bonus{Elem: [numElem]int{elemFire: 8}}),
			active(3), crypt(2),
			passive("Inferno", "Golpes de fogo deixam o alvo queimando: 5 de dano por segundo por 3 s, por grau.", "inferno", elemFire, 2, bonus{Burn: 1})},
		{active(4),
			passive("Frio Cortante", "+8% de dano de gelo e +0,5 s em toda lentidão que você causa, por grau.", "frio", elemIce, 3, bonus{Chill: 5, Elem: [numElem]int{elemIce: 8}}),
			active(5), active(6)},
		{active(7),
			passive("Condutor", "O raio salta para mais 1 alvo e ganha +6% de dano, por grau.", "condutor", elemBolt, 2, bonus{Chain: 1, Elem: [numElem]int{elemBolt: 6}}),
			active(8),
			passive("Sobrecarga", "Energia demais no corpo: +4% de chance de crítico por grau.", "sobrecarga", elemBolt, 3, bonus{Crit: 4})},
		{pFocus, active(9), active(10),
			passive("Mente Vasta", "+15 de mana máxima e +1 de mana por segundo, por grau.", "mente", elemArcane, 3, bonus{MP: 15, MPRegen: 1})},
	},
	{ // Cavaleiro
		{pSharp, active(3), crypt(2), pMastery},
		{pSkin, active(4), pVigor, active(5)},
		{pFocus, pBreath, active(6), pCrit},
	},
	{ // Paladino
		{pSharp, active(3), crypt(2), pMastery},
		{pSkin, active(4), pVigor, active(5)},
		{pFocus, pBreath, active(6),
			passive("Graça", "+8% de dano sagrado e +1 de mana por segundo, por grau.", "graca", elemHoly, 3, bonus{MPRegen: 1, Elem: [numElem]int{elemHoly: 8}})},
	},
	{ // Arqueiro
		{pSharp, active(3), crypt(2), pMastery},
		{pSkin, active(4), pVigor, active(6)},
		{pFocus, pBreath, active(5), pCrit},
	},
	{ // Ladino
		{pSharp, active(3), crypt(2), pCrit},
		{pSkin, active(4), pLeech, active(6)},
		{pFocus, active(5),
			passive("Toxinas", "+10% de dano de veneno e de sombra por grau.", "toxina", elemPoison, 3, bonus{Elem: [numElem]int{elemPoison: 10, elemShadow: 10}}),
			pMastery},
	},
	{ // Necromante
		{pSharp, active(3), crypt(2), pMastery},
		{pVigor, active(5), pLeech,
			passive("Mortalha", "+8% de dano de sombra e +4% de proteção, por grau.", "mortalha", elemShadow, 3, bonus{Def: 4, Elem: [numElem]int{elemShadow: 8}})},
		{pFocus, pBreath, active(4), active(6)},
	},
}

func init() {
	classes = classBook()
	extendClasses()
	d, ds := buildDepraved()
	classes = append(classes, d)
	webSpecs = append(webSpecs, ds)
	for i := range classes {
		c := &classes[i]
		c.Web = buildWeb(c, c.Branches, webSpecs[i])
		s, lv, at := capstoneOf(c)
		capstone(c, s, -90, lv, at)
		c.Adj = neighbors(c.Web)
	}
	addTechs()
}

// capstone acrescenta à teia o nó final da classe, num quinto anel acima de
// todos: liga aos quartos anéis de todos os ramos (basta um aprendido).
func capstone(c *class, skill int, ang float64, lv, attr int) {
	n := webNode{Ring: 5, Ang: ang, Max: 1, Lv: lv, Attr: attr, Skill: skill, Branch: -1}
	sk := c.Skills[skill]
	n.Name, n.Desc, n.Icon, n.Elem = sk.Name, sk.Desc, sk.Icon, sk.Elem
	for i, w := range c.Web {
		if w.Ring == 4 {
			n.Links = append(n.Links, i)
		}
	}
	c.Web = append(c.Web, n)
}
