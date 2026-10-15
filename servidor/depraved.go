package main

import "time"

// O Depravado: sem mestre e sem escola. A teia dele tem um ramo emprestado
// de cada classe (o primeiro ramo de cada uma, com os golpes e as passivas
// dela), e ele pode seguir qualquer um — mas no máximo três escolas ao mesmo
// tempo (o Tomo do Recomeço desfaz a escolha). Começa com um pedaço de pau.
// O atributo pedido pela teia é o mais alto que ele tiver (anyAttr).

const (
	classDepraved = 7
	maxSchools    = 3 // ramos (escolas) diferentes que o Depravado pode seguir
)

func depravedClass() class {
	return class{Name: "Depravado", HP: 120, MP: 70, Magic: true, Attr: attrStr, anyAttr: true, Skills: []skillDef{
		{"Paulada", "hammer", "Desce o pedaço de pau (ou o que tiver na mão) num arco à frente (16 de alcance). Dano 22–38, mais 4 por nível; sem custo.", elemPhys, 0, 380 * ms, 0, func(c *cast) {
			c.fxAt(fxSlash, c.x, c.y, 12, 220)
			c.hit(22, 17, arc(c.x, c.y, c.dir, 16, 1.4), hitOpt{})
		}},
		{"Uivo Desvairado", "horn", "Um berro sem sentido que assusta tudo em volta (raio 26): dano 15–25 e lentidão de 2 s. Ninguém sabe se é coragem ou loucura.", elemShadow, 10, 5 * time.Second, 1, func(c *cast) {
			c.fxDir(fxBuff, c.x, c.y, 26, 5, 600)
			c.hit(15, 11, around(c.x, c.y, 26), hitOpt{Elem: elemShadow, Slow: 2})
		}},
	}}
}

// buildDepraved monta o livro e a teia do Depravado a partir das outras
// classes, já prontas (chamar depois de estendê-las).
func buildDepraved() (class, [][4]nodeSpec) {
	d := depravedClass()
	var specs [][4]nodeSpec
	for i := 0; i < classDepraved; i++ {
		src := &classes[i]
		sp := webSpecs[i][0] // o primeiro ramo da classe
		remap := map[int]int{}
		for k := range sp {
			if s := sp[k].skill; s >= 0 {
				if _, ok := remap[s]; !ok {
					remap[s] = len(d.Skills)
					d.Skills = append(d.Skills, src.Skills[s])
				}
				sp[k].skill = remap[s]
			}
		}
		specs = append(specs, sp)
		d.Branches = append(d.Branches, branchDef{src.Branches[0].Name + " (" + src.Name + ")", 0, src.Branches[0].Elem})
	}
	// Golpe final: o caos de quem não segue ninguém.
	d.Skills = append(d.Skills, skillDef{"Caos Depravado", "burst", "O golpe final do Depravado. Explode tudo o que aprendeu numa onda sem forma (raio 36): dano 150–230 de um elemento ao acaso, atordoamento de 1,5 s, e você recupera 20% da vida.", elemArcane, 50, 90 * time.Second, 2, func(c *cast) {
		elems := []int{elemFire, elemIce, elemBolt, elemArcane, elemShadow, elemHoly, elemPoison}
		e := elems[c.wd.rng.Intn(len(elems))]
		c.fxAt(fxDark, c.x, c.y, 36, 800)
		c.fxAt(fxFrost, c.x, c.y, 30, 700)
		c.hit(150, 81, around(c.x, c.y, 36), hitOpt{Elem: e, Stun: 1.5})
		c.p.HP = min(c.p.MaxHP, c.p.HP+c.p.MaxHP/5)
	}})
	n := len(d.Branches)
	for k := range d.Branches {
		d.Branches[k].Ang = -90 + 360*(float64(k)+0.5)/float64(n)
	}
	return d, specs
}

// schools conta os ramos em que o jogador já tem algum nó aprendido.
func (p *Player) schools() map[int]bool {
	out := map[int]bool{}
	for i, n := range p.web() {
		if i < len(p.Web) && p.Web[i] > 0 && n.Branch >= 0 {
			out[n.Branch] = true
		}
	}
	return out
}

// classAttr é quanto o jogador tem no atributo que a teia pede.
func (p *Player) classAttr() int {
	c := classes[p.Class]
	if !c.anyAttr {
		return p.Attr[c.Attr]
	}
	best := 0
	for _, v := range p.Attr {
		best = max(best, v)
	}
	return best
}
