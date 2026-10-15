package main

import (
	"time"

	"asciimmo/proto"
)

// Técnicas: golpes que qualquer classe aprende com o Mestre das Sequências,
// num terreiro de pedra a sudoeste da Encruzilhada. O mestre mostra a
// sequência de setas (como um golpe de Mortal Kombat) e some com ela; quem
// repetir certo, com nível e dinheiro, aprende. Vão para o fim do livro de
// cada classe (a partir de class.techBase) e para a barra como os outros.

const (
	masterX, masterY = hubX - 120, hubY + 70
)

type tech struct {
	skillDef
	Seq   string // U D L R
	Lv    int
	Price int
}

var techs = []tech{
	{skillDef{"Rolamento", "dash", "Rola 18 células na mira, rápido demais para ser acertado: meio segundo protegido (só 30% do dano passa). Serve para sair do cerco.", elemPhys, 8, 3 * time.Second, 0, func(c *cast) {
		x0, y0, x, y := c.wd.slide(c.p, c.dir, 18, c.now)
		c.fxDir(fxSmoke, x0, y0, dist(x0, y0, x, y), c.dir, 300)
		guardFor(c.p, c.now, 500*time.Millisecond)
		c.wd.enter(c.p)
	}}, "DRR", 3, 40},
	{skillDef{"Brado de Guerra", "horn", "Um grito que acende o sangue: +30% de dano por 8 s, para você e quem estiver a até 30 células.", elemPhys, 15, 25 * time.Second, 1, func(c *cast) {
		c.fxDir(fxBuff, c.x, c.y, 30, 0, 800)
		c.buffAllies(30, func(q *Player) { q.dmgUntil = c.now.Add(8 * time.Second) })
	}}, "DUU", 5, 80},
	{skillDef{"Redemoinho", "spiral", "Gira com a arma estendida: dano 45–75 em tudo num raio de 20 e empurra o cerco para longe (lentidão de 1,5 s).", elemPhys, 18, 8 * time.Second, 2, func(c *cast) {
		c.fxAt(fxSpin, c.x, c.y, 20, 500)
		c.hit(45, 31, around(c.x, c.y, 20), hitOpt{Slow: 1.5})
	}}, "LDRU", 8, 150},
	{skillDef{"Punho do Dragão", "hand", "Um gancho que sobe em chamas e corre 30 células à frente: dano 70–110 de fogo e atordoamento de 1,2 s.", elemFire, 24, 10 * time.Second, 2, func(c *cast) {
		ex, ey := c.ahead(30)
		c.fxDir(fxFlame, c.x, c.y, 30, c.dir, 500)
		c.hit(70, 41, along(c.x, c.y, ex, ey, 6), hitOpt{Elem: elemFire, Stun: 1.2, Ignite: 4})
	}}, "DRDRU", 12, 300},
	{skillDef{"Golpe de Misericórdia", "skull", "Termina o que começou: quem estiver à frente (16 de alcance) com menos de 20% da vida cai na hora — menos chefes, que só levam dano 90–140.", elemShadow, 30, 20 * time.Second, 2, func(c *cast) {
		c.fxAt(fxDark, c.x+c.cos*8, c.y+c.sin*8, 14, 600)
		for _, m := range c.mobsNear(c.x, c.y, 18) {
			pct := 0
			if !mobKinds[m.Kind].Boss && m.HP*5 < m.MaxHP {
				pct = 5000 // acaba com ele
			}
			c.hit(90, 51, func(x, y, pad float64) bool { return x == m.X && y == m.Y }, hitOpt{Elem: elemShadow, Pct: pct})
		}
	}}, "RLRDUL", 16, 600},
}

// addTechs põe as técnicas no fim do livro de todas as classes.
func addTechs() {
	for i := range classes {
		c := &classes[i]
		c.techBase = len(c.Skills)
		for _, t := range techs {
			c.Skills = append(c.Skills, t.skillDef)
		}
	}
}

func (p *Player) knowsTech(t int) bool { return p.Tech&(1<<t) != 0 }

var master = npc{"Mestre Oren", masterX, masterY, []string{
	"Hm. Mais um que veio aprender a bater sem pensar. Bom. Pensar é lento.",
	"Eu não ensino golpes de classe. Ensino o que serve a qualquer mão: rolar, gritar, girar, socar, terminar.",
	"Eu mostro a sequência uma vez. Você repete com as setas — rápido, sem errar. Errou? Começa de novo, o chão é paciente.",
	"Cada técnica pede nível e um punhado de bosta. O mestre também come.",
}}

// buildDojo ergue o terreiro do mestre: chão de pedra, postes de treino e
// uma placa.
func (wd *World) buildDojo(seed uint32) {
	wd.clearArea(masterX-22, masterY-16, masterX+22, masterY+14, seed+1300)
	for y := masterY - 10; y <= masterY+8; y++ {
		for x := masterX - 16; x <= masterX+16; x++ {
			if (x+y)%2 == 0 {
				wd.put(x, y, '.', dim(0x9a9088, 0.45+0.15*rnd01(x, y, seed+1301)), false, false)
			}
		}
	}
	for _, d := range [][2]int{{masterX - 12, masterY - 6}, {masterX + 12, masterY - 6}, {masterX - 12, masterY + 5}, {masterX + 12, masterY + 5}} {
		for k := 0; k < 3; k++ {
			wd.put(d[0], d[1]-k, "I|T"[k], dim(0x8a6a44, 0.9), k == 0, k > 0)
		}
	}
	wd.signText(masterX, masterY+11, "TERREIRO DAS SEQUENCIAS", 0xc8b48c, seed+1302)
	wd.lights = append(wd.lights, light{masterX - 12, masterY - 8, 24, 2}, light{masterX + 12, masterY - 8, 24, 2})
}

// Learn: o jogador repetiu seq diante do mestre para aprender a técnica t.
func (wd *World) Learn(id, t int, seq string) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || t < 0 || t >= len(techs) || dist(float64(p.X), float64(p.Y), masterX, masterY) > 30 {
		return
	}
	now := wd.clock()
	tk := techs[t]
	msg, color := "", 0xff7060
	switch {
	case p.knowsTech(t):
		msg = "Você já sabe " + tk.Name
	case p.Level < tk.Lv:
		msg = "Volte no nível " + itoa(tk.Lv)
	case p.Gold < tk.Price:
		msg = "Faltam " + itoa(tk.Price-p.Gold) + " de bosta"
	case seq != tk.Seq:
		msg = "Sequência errada — o mestre balança a cabeça"
	default:
		p.Gold -= tk.Price
		p.Tech |= 1 << t
		msg, color = "Aprendeu "+tk.Name+"!", 0xffe066
		s := classes[p.Class].techBase + t
		for k, b := range p.Bar {
			if b < 0 && k < p.barCap() {
				p.Bar[k] = s
				break
			}
		}
		wd.say(ChatLine{Text: p.Name + " aprendeu " + tk.Name + " com o Mestre Oren."})
	}
	wd.pop(p, msg, color, float64(p.X), float64(p.Y)-40, now)
}

func techInfo(c *class) []proto.Tech {
	var out []proto.Tech
	for i, t := range techs {
		out = append(out, proto.Tech{Skill: c.techBase + i, Seq: t.Seq, Lv: t.Lv, Price: t.Price})
	}
	return out
}
