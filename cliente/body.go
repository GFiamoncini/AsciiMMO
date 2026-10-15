package main

import (
	"math"
	"math/rand"
)

// Os personagens não são sprites fixos: cabeça, tronco, braços, pernas e arma
// são cápsulas posicionadas a cada quadro. Coordenadas locais em células, com
// (0, 0) nos pés e y para cima negativo.

type bodyStyle struct {
	// corpo de gente
	bone                             bool
	head                             string
	hair, hat, helm, plume           *col
	torso, torso2, belt, legs, arm   *col
	skin, boots, cape, chest, shield *col
	orb                              *col
	robe                             bool
	weapon                           string
	wlen, scale                      float64
	seed                             int
	aura                             bool
	embers                           bool // solta brasas e ilumina em volta (bichos de fogo)
	// bicho
	beast                       int
	fly, tail, ears, tusk, arch bool
	blen, br, bh, leg, neck, hr float64
	tailLen                     float64
	c1, c2, mane, eye           *col
}

func pc(r, g, b float64) *col { c := col{r, g, b}; return &c }

func cp(c col) *col { return &c }

var (
	steel = col{176, 184, 214}
	dark  = col{26, 20, 30}
	wood  = col{128, 88, 52}
	gold  = col{236, 196, 92}
	skinC = col{232, 196, 160}
	boneC = col{228, 184, 136}
)

// charScale encolhe todo personagem (gente, bichos, montarias) em relação ao
// cenário: casas, árvores e muralhas ficam proporcionais a quem anda nelas.
// Tiamath tem escala própria e continua do tamanho de um dragão.
const charScale = 0.62

func (s bodyStyle) sc() float64 {
	if s.scale == 0 {
		return charScale
	}
	return s.scale * charScale
}

func or(c *col, d col) col {
	if c == nil {
		return d
	}
	return *c
}

// body é o corpo em desenho: origem, corte no chão, clarão.
type body struct {
	ox, oy, clip int
	flash        float64
	tint         col
	seed         int
	mask         map[int]bool
	lit          bool // modo Mescla: desenho 2D com luz de volume, como o 3D
	edgeMaxY     int  // contorno vazio só até esta linha (montado: não apaga o bicho embaixo)
}

// bodyRamp são as letras do corpo: só as densas, do mais escuro ao mais claro.
// Com a rampa inteira do cenário (que começa em ".:;") e um sorteio por
// célula, o boneco virava uma sopa de letras e a silhueta se desmanchava.
const bodyRamp = "o0O8&%#@"

func bodyCode(l float64) byte {
	return bodyRamp[clampi(int(l*float64(len(bodyRamp))), 0, len(bodyRamp)-1)]
}

func (g *Game) px(e *body, lx, ly int, c col, l float64) {
	m := g.m
	wx, wy := e.ox+lx, e.oy+ly
	if wy > e.clip || wx < 0 || wy < 0 || wx >= m.w || wy >= m.h || g.hides(wx, wy, e.oy) {
		return
	}
	e.mask[wy*m.w+wx] = true
	n := hash2(lx, ly, e.seed)
	l = clamp(l+(float64(n&255)/255-0.5)*0.16, 0.25, 1.3)
	r, gg, b := capLuma(math.Min(255, c[0]*l), math.Min(255, c[1]*l), math.Min(255, c[2]*l), c)
	if e.flash > 0 {
		r += (e.tint[0] - r) * e.flash
		gg += (e.tint[1] - gg) * e.flash
		b += (e.tint[2] - b) * e.flash
	}
	g.setCell(wx, wy, bodyCode((0.299*r+0.587*gg+0.114*b)/255*1.15), r, gg, b)
}

// capLuma segura a luz do corpo abaixo do limiar do bloom: roupa e pele não
// brilham; só o que já é claro de nascença (orbe, lâmina, aura) passa disso.
func capLuma(r, gg, b float64, base col) (float64, float64, float64) {
	lum := 0.299*r + 0.587*gg + 0.114*b
	lim := math.Max(150, 0.299*base[0]+0.587*base[1]+0.114*base[2])
	if lum > lim {
		k := lim / lum
		r, gg, b = r*k, gg*k, b*k
	}
	return r, gg, b
}

// capsule preenche um segmento com raio, clareando o lado de cima e da
// esquerda; em volta fica um contorno vazio para a silhueta se destacar.
func (g *Game) capsule(e *body, ax, ay, bx, by, r float64, c col) {
	m := g.m
	vx, vy := bx-ax, by-ay
	L := vx*vx + vy*vy
	minX, maxX := int(math.Floor(math.Min(ax, bx)-r))-1, int(math.Ceil(math.Max(ax, bx)+r))+1
	minY, maxY := int(math.Floor(math.Min(ay, by)-r))-1, int(math.Ceil(math.Max(ay, by)+r))+1
	edge := (r + 1.3) * (r + 1.3)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			t := 0.0
			if L > 0 {
				t = clamp((float64(x-0)-ax)*vx/L+(float64(y)-ay)*vy/L, 0, 1)
			}
			dx, dy := float64(x)-ax-t*vx, float64(y)-ay-t*vy
			d2 := dx*dx + dy*dy
			if d2 > r*r {
				wx, wy := e.ox+x, e.oy+y
				if wy <= e.edgeMaxY && d2 <= edge && wy <= e.clip && wx >= 0 && wy >= 0 && wx < m.w && wy < m.h &&
					!g.hides(wx, wy, e.oy) && !e.mask[wy*m.w+wx] {
					g.setCell(wx, wy, ' ', 0, 0, 0)
				}
				continue
			}
			if e.lit {
				// Mescla: cada cápsula é um cilindro; a distância ao eixo dá a
				// normal, e a luz é a mesma do 3D (chave, contorno e brilho).
				rr := math.Max(r, 0.6)
				nx, ny := dx/rr, -dy/rr
				nz := math.Sqrt(math.Max(0, 1-nx*nx-ny*ny))
				g.pxLit(e, x, y, c, v3{nx, ny, nz})
				continue
			}
			g.px(e, x, y, c, 0.9-0.3*(dx+dy)/(2*math.Max(r, 1)))
		}
	}
}

// pxLit pinta uma célula do corpo com a luz de volume do modo Mescla.
func (g *Game) pxLit(e *body, lx, ly int, c col, n v3) {
	m := g.m
	wx, wy := e.ox+lx, e.oy+ly
	if wy > e.clip || wx < 0 || wy < 0 || wx >= m.w || wy >= m.h || g.hides(wx, wy, e.oy) {
		return
	}
	e.mask[wy*m.w+wx] = true
	// A luz vem do alto, da esquerda e da frente (a mesma do 3D).
	diff := math.Max(0, n.dot(v3{-0.55, 0.75, 0.6}.norm()))
	rim := math.Pow(1-n.z, 3)
	spec := math.Pow(math.Max(0, n.dot(v3{-0.32, 0.43, 0.84})), 18) * 0.18
	l := 0.24 + 1.1*diff
	r, gg, b := c[0]*l+95*rim+255*spec, c[1]*l+85*rim+255*spec, c[2]*l+120*rim+255*spec
	r, gg, b = capLuma(r, gg, b, c)
	if e.flash > 0 {
		r += (e.tint[0] - r) * e.flash
		gg += (e.tint[1] - gg) * e.flash
		b += (e.tint[2] - b) * e.flash
	}
	h := hash2(lx, ly, e.seed)
	k := 1 + (float64(h&15)/15-0.5)*0.08 // textura só na cor, bem de leve
	r, gg, b = r*k, gg*k, b*k
	g.setCell(wx, wy, bodyCode((0.299*r+0.587*gg+0.114*b)/255*1.2), r, gg, b)
}

// bodyDraw são os parâmetros de um corpo a desenhar.
type bodyDraw struct {
	style  *bodyStyle
	x, y   int
	T      float64
	phase  *float64 // fase do passo, ou nil parado
	hf     float64
	face   int
	act    *action
	sink   int
	flash  float64
	tint   col
	guard  bool
	wcol   *col
	lit    bool // desenho 2D com luz de volume (modo Mescla)
	seated bool // montado: sentado na sela, pernas pelos flancos
}

type action struct {
	anim string
	t    float64
	dir  float64
}

var anims = [][3]string{{"swing", "thrust", "spin"}, {"point", "point", "cast"}, {"thrust", "block", "thrust"}, {"swing", "cast", "cast"},
	{"point", "point", "cast"}, {"swing", "thrust", "spin"}, {"point", "thrust", "cast"}, {"swing", "cast", "spin"}}
var animDur = map[string]float64{"swing": 0.28, "thrust": 0.3, "spin": 0.5, "point": 0.3, "cast": 0.6, "block": 0.4}

var playerColors = []col{hexc(0xff6b8b), hexc(0xffa94d), hexc(0x8ce99a), hexc(0x66d9e8), hexc(0xb197fc), hexc(0xf783ac), hexc(0xffd43b), hexc(0x74c0fc)}

func playerColor(c int) col {
	return playerColors[(c%len(playerColors)+len(playerColors))%len(playerColors)]
}

// playerStyle monta o visual da classe na cor do jogador; a skin equipada
// tinge a roupa e as lendárias soltam brilho.
func (g *Game) playerStyle(cls, c, sk int) *bodyStyle {
	key := cls*100 + c%len(playerColors) + sk*10000
	if s, ok := g.styles[key]; ok {
		return s
	}
	p := playerColor(c)
	var st bodyStyle
	switch cls {
	case 0:
		st = bodyStyle{head: "hair", hair: pc(126, 88, 60), torso: cp(mix3(p, col{150, 110, 80}, 0.45)), belt: cp(wood), legs: pc(104, 110, 150),
			arm: cp(mix3(p, col{150, 110, 80}, 0.45)), weapon: "sword", wlen: 11}
	case 1:
		st = bodyStyle{head: "hat", hat: cp(mix3(p, col{70, 50, 150}, 0.6)), hair: pc(210, 210, 220), torso: cp(mix3(p, col{84, 66, 170}, 0.55)),
			torso2: cp(gold), robe: true, arm: cp(mix3(p, col{84, 66, 170}, 0.55)), weapon: "staff", wlen: 14, orb: pc(150, 190, 255)}
	case 2:
		st = bodyStyle{head: "helm", plume: cp(p), torso: cp(steel), torso2: cp(p), belt: pc(90, 96, 124), legs: pc(140, 148, 180), arm: cp(steel),
			skin: cp(steel), weapon: "lance", wlen: 20, shield: cp(mix3(p, steel, 0.35))}
	case 3:
		st = bodyStyle{head: "circlet", hair: pc(240, 214, 140), torso: pc(232, 232, 240), torso2: cp(gold), belt: cp(gold), legs: pc(204, 204, 220),
			arm: pc(232, 232, 240), cape: cp(p), weapon: "hammer", wlen: 10}
	case 4:
		st = bodyStyle{head: "hood", hair: pc(50, 90, 60), torso: cp(mix3(p, col{60, 110, 70}, 0.55)), belt: cp(wood), legs: pc(90, 80, 60),
			arm: cp(mix3(p, col{60, 110, 70}, 0.55)), weapon: "bow", wlen: 3}
	case 5:
		st = bodyStyle{head: "hood", hair: pc(44, 40, 56), torso: cp(mix3(p, col{60, 56, 76}, 0.7)), belt: pc(120, 40, 50), legs: pc(50, 48, 64),
			arm: cp(mix3(p, col{60, 56, 76}, 0.7)), cape: pc(36, 32, 48), weapon: "sword", wlen: 6}
	case 7: // Depravado: andrajos remendados, cabelo desgrenhado e um pedaço de pau
		st = bodyStyle{head: "hair", hair: pc(70, 60, 50), torso: cp(mix3(p, col{100, 86, 62}, 0.7)), torso2: pc(130, 60, 50), belt: pc(80, 60, 40),
			legs: pc(84, 74, 60), arm: cp(mix3(p, col{120, 100, 76}, 0.6)), boots: pc(60, 46, 34), weapon: "club", wlen: 9}
	default:
		st = bodyStyle{head: "hood", hair: pc(60, 30, 80), torso: cp(mix3(p, col{70, 36, 96}, 0.65)), torso2: pc(140, 255, 170), robe: true,
			arm: cp(mix3(p, col{70, 36, 96}, 0.65)), weapon: "staff", wlen: 13, orb: pc(140, 255, 170)}
	}
	st.seed = 40 + cls
	if st.boots == nil {
		st.boots = pc(96, 72, 56)
	}
	if st.skin == nil {
		st.skin = cp(skinC)
	}
	if sk > 0 && sk < len(g.info.Items) {
		it := g.info.Items[sk]
		s := hexc(it.Color)
		st.torso = cp(mix3(s, *st.torso, 0.25))
		st.arm = cp(mix3(s, *st.arm, 0.35))
		if !st.robe {
			st.legs = cp(mix3(s, or(st.legs, col{104, 110, 150}), 0.6))
		}
		st.cape = cp(mix3(s, col{}, 0.35))
		st.aura = it.Rarity == 3
	}
	g.styles[key] = &st
	return &st
}

var mobStyles = func() []bodyStyle {
	asura := bodyStyle{head: "hair", hair: pc(40, 20, 60), skin: pc(214, 160, 230), torso: pc(190, 50, 70), torso2: cp(gold), belt: cp(gold),
		legs: pc(90, 40, 90), arm: pc(214, 160, 230), boots: pc(60, 30, 60), weapon: "sword", wlen: 9, seed: 64}
	triton := bodyStyle{head: "helm", helm: pc(60, 170, 160), plume: pc(40, 220, 200), skin: pc(90, 200, 190), torso: pc(50, 130, 140),
		torso2: pc(200, 240, 230), legs: pc(40, 110, 120), arm: pc(90, 200, 190), boots: pc(30, 80, 90), weapon: "lance", wlen: 16, seed: 65}
	sentinel := bodyStyle{head: "helm", helm: pc(240, 220, 150), plume: pc(255, 255, 240), skin: pc(240, 220, 150), torso: pc(236, 226, 190),
		torso2: cp(gold), belt: cp(gold), legs: pc(200, 190, 150), arm: pc(236, 226, 190), boots: pc(160, 140, 90), weapon: "hammer", wlen: 9,
		shield: pc(240, 230, 200), seed: 66}
	orc := bodyStyle{head: "hair", hair: pc(40, 44, 36), skin: pc(112, 160, 90), torso: pc(120, 86, 54), belt: pc(60, 44, 30), legs: pc(86, 70, 50),
		arm: pc(112, 160, 90), boots: pc(56, 44, 34), weapon: "club", wlen: 8, seed: 72}
	mummy := bodyStyle{head: "hair", hair: pc(224, 214, 180), skin: pc(214, 204, 170), torso: pc(224, 214, 180), torso2: pc(150, 140, 110),
		legs: pc(204, 194, 160), arm: pc(224, 214, 180), boots: pc(180, 170, 140), weapon: "club", wlen: 1, seed: 76}
	b := cp(boneC)
	with := func(s bodyStyle, f func(*bodyStyle)) bodyStyle { f(&s); return s }
	return []bodyStyle{
		{bone: true, head: "skull", torso: b, arm: b, legs: b, skin: b, boots: b, weapon: "club", wlen: 7, seed: 61},
		{bone: true, head: "skullhelm", torso: b, arm: b, legs: b, skin: b, boots: b, chest: pc(132, 138, 168), weapon: "sword", wlen: 10, shield: pc(150, 70, 50), seed: 62},
		{bone: true, head: "skullcrown", torso: pc(206, 170, 226), arm: pc(206, 170, 226), legs: pc(206, 170, 226), skin: pc(206, 170, 226),
			boots: pc(206, 170, 226), chest: pc(96, 60, 140), cape: pc(110, 40, 130), weapon: "sword", wlen: 15, scale: 1.4, seed: 63},
		asura, triton, sentinel,
		with(asura, func(s *bodyStyle) {
			s.head, s.hair, s.cape, s.wlen, s.scale, s.seed = "circlet", pc(255, 120, 190), pc(200, 40, 110), 12, 1.5, 67
		}),
		with(triton, func(s *bodyStyle) { s.cape, s.wlen, s.scale, s.seed = pc(30, 110, 130), 22, 1.6, 68 }),
		with(sentinel, func(s *bodyStyle) { s.cape, s.wlen, s.scale, s.seed = pc(255, 250, 230), 12, 1.7, 69 }),
		{beast: 4, blen: 3, br: 1.8, bh: 1.5, c1: pc(124, 106, 98), c2: pc(156, 134, 124), tail: true, seed: 70},
		{beast: 4, blen: 5.5, br: 2.8, bh: 4.5, c1: pc(128, 128, 140), c2: pc(158, 158, 170), tail: true, ears: true, seed: 71},
		orc,
		with(orc, func(s *bodyStyle) {
			s.head, s.helm, s.plume, s.weapon, s.wlen, s.shield, s.seed = "helm", pc(120, 124, 140), pc(150, 40, 30), "sword", 10, pc(110, 70, 44), 73
		}),
		with(orc, func(s *bodyStyle) {
			s.head, s.helm, s.plume, s.cape, s.weapon, s.wlen, s.scale, s.seed = "helm", pc(90, 90, 100), pc(220, 60, 40), pc(150, 36, 28), "hammer", 11, 1.5, 74
		}),
		{beast: 6, blen: 3.5, br: 3.2, bh: 2, c1: pc(60, 96, 170), c2: pc(30, 44, 90), arch: true, seed: 75},
		mummy,
		with(mummy, func(s *bodyStyle) {
			s.head, s.hair, s.skin, s.torso, s.torso2, s.arm, s.cape = "circlet", pc(40, 70, 180), pc(214, 176, 120), pc(236, 196, 92), pc(40, 70, 180), pc(214, 176, 120), pc(40, 70, 180)
			s.weapon, s.wlen, s.orb, s.scale, s.seed = "staff", 14, pc(255, 230, 120), 1.5, 77
		}),
		{beast: 4, blen: 5.5, br: 2.9, bh: 4.5, c1: pc(208, 224, 240), c2: pc(240, 246, 255), tail: true, ears: true, seed: 78},
		with(mummy, func(s *bodyStyle) {
			s.hair, s.skin, s.torso, s.arm, s.legs, s.boots = pc(240, 244, 252), pc(200, 216, 236), pc(236, 240, 250), pc(236, 240, 250), pc(224, 230, 244), pc(200, 216, 236)
			s.weapon, s.wlen, s.scale, s.seed = "club", 8, 1.25, 79
		}),
		with(mummy, func(s *bodyStyle) {
			s.head, s.helm, s.plume, s.skin, s.torso, s.arm, s.legs, s.boots, s.cape = "helm", pc(170, 210, 245), pc(255, 255, 255), pc(150, 190, 235), pc(90, 124, 176), pc(150, 190, 235), pc(80, 104, 150), pc(60, 80, 120), pc(200, 230, 255)
			s.weapon, s.wlen, s.scale, s.seed = "hammer", 13, 1.8, 80
		}),
		{beast: 8, blen: 3, br: 3.6, bh: 3, c1: pc(56, 44, 66), c2: pc(130, 34, 44), arch: true, seed: 81},
		with(orc, func(s *bodyStyle) {
			s.hair, s.skin, s.arm, s.torso, s.legs, s.wlen, s.scale, s.seed = pc(50, 70, 44), pc(92, 132, 80), pc(92, 132, 80), pc(84, 72, 50), pc(70, 62, 44), 9, 1.1, 82
		}),
		{beast: 2, fly: true, blen: 0.8, br: 1.5, bh: 10, c1: pc(86, 70, 90), c2: pc(54, 42, 60), ears: true, hr: 1.1, seed: 83},
		// Boneco de treino: saco de palha num poste, cabelo de capim seco.
		{head: "hair", hair: pc(206, 178, 96), skin: pc(196, 166, 112), torso: pc(170, 138, 86), torso2: pc(110, 80, 50), belt: pc(120, 84, 50),
			arm: pc(196, 166, 112), legs: pc(120, 90, 56), boots: pc(90, 66, 40), weapon: "club", wlen: 0, seed: 84},
		// Profundezas. Rastejante Cego: bicho baixo, comprido, pálido, sem olhos.
		{beast: 6, blen: 6, br: 2.2, bh: 1.6, c1: pc(176, 168, 160), c2: pc(110, 100, 96), tail: true, arch: true, seed: 85},
		// Golem de Magma: grandalhão de basalto com brasa nas juntas.
		{head: "hair", hair: pc(60, 36, 30), skin: pc(70, 44, 36), torso: pc(64, 40, 34), torso2: pc(255, 120, 30), belt: pc(255, 90, 20),
			arm: pc(70, 44, 36), legs: pc(56, 34, 28), boots: pc(40, 24, 20), weapon: "club", wlen: 3, scale: 1.4, aura: true, embers: true, seed: 86},
		// Casca de Fungo: um morto tomado pelo mofo, com chapéu de cogumelo.
		with(mummy, func(s *bodyStyle) {
			s.head, s.hat, s.hair, s.skin, s.torso, s.arm, s.legs, s.boots = "hat", pc(150, 70, 60), pc(90, 120, 80), pc(110, 140, 96), pc(70, 96, 64), pc(110, 140, 96), pc(60, 80, 54), pc(40, 54, 36)
			s.weapon, s.wlen, s.seed = "club", 2, 87
		}),
		// Ignar, o Coração de Magma: o golem rei, com coroa de chamas e malho.
		{head: "circlet", hair: pc(255, 140, 40), skin: pc(80, 46, 36), torso: pc(70, 40, 32), torso2: pc(255, 160, 40), belt: pc(255, 110, 30),
			arm: pc(80, 46, 36), legs: pc(60, 36, 28), boots: pc(44, 26, 20), cape: pc(180, 40, 20), weapon: "hammer", wlen: 13, scale: 2, aura: true, embers: true, seed: 88},
		// Apóstolo Faminto: gente que virou carne demais; pele pálida, peito aberto.
		{head: "hair", hair: pc(40, 24, 30), skin: pc(214, 170, 160), torso: pc(200, 140, 130), torso2: pc(150, 30, 40), belt: pc(120, 30, 36),
			arm: pc(214, 170, 160), legs: pc(180, 120, 110), boots: pc(110, 60, 56), weapon: "club", wlen: 2, scale: 1.7, seed: 89},
		// Grael, o Apóstolo Rei: armadura negra com chifres, capa rubra e espadão.
		{head: "helm", helm: pc(40, 36, 44), plume: pc(220, 200, 170), skin: pc(70, 60, 70), torso: pc(44, 40, 50), torso2: pc(160, 30, 40),
			belt: pc(120, 100, 60), legs: pc(36, 32, 40), arm: pc(54, 48, 60), boots: pc(30, 26, 34), cape: pc(150, 20, 30), chest: pc(70, 64, 80),
			weapon: "sword", wlen: 18, scale: 2.2, aura: true, seed: 90},
		// Escorpião de Arenito: bicho baixo e largo, cor de areia, cauda erguida.
		{beast: 6, blen: 4, br: 3, bh: 1.8, c1: pc(196, 150, 80), c2: pc(120, 84, 40), tail: true, arch: true, seed: 91},
		// Espectro Gélido: um morto pálido de manto, quase transparente de tão claro.
		with(mummy, func(s *bodyStyle) {
			s.head, s.hair, s.skin, s.torso, s.torso2, s.arm, s.legs, s.boots = "hood", pc(170, 220, 250), pc(190, 230, 255), pc(150, 200, 236), pc(90, 150, 210), pc(190, 230, 255), pc(130, 180, 220), pc(110, 160, 210)
			s.robe, s.weapon, s.wlen, s.aura, s.seed = true, "staff", 12, true, 92
		}),
		// Horror das Raízes: tronco que anda, braços de galho.
		{head: "hair", hair: pc(70, 110, 50), skin: pc(110, 84, 54), torso: pc(96, 72, 46), torso2: pc(60, 100, 44), belt: pc(70, 50, 30),
			arm: pc(110, 84, 54), legs: pc(84, 62, 40), boots: pc(60, 44, 28), weapon: "club", wlen: 6, scale: 1.5, seed: 93},
		// Guardião de Dorrim: estátua de pedra com elmo de anão e martelo.
		{head: "helm", helm: pc(150, 156, 176), plume: pc(200, 180, 120), skin: pc(140, 144, 160), torso: pc(120, 124, 142), torso2: pc(200, 180, 120),
			belt: pc(200, 180, 120), legs: pc(110, 114, 130), arm: pc(140, 144, 160), boots: pc(90, 94, 110), weapon: "hammer", wlen: 9, scale: 1.3, seed: 94},
		// Vulkhar, o Flagelo das Profundezas: sombra e fogo, chifres em brasa, asas
		// negras (a capa), a lâmina-chicote acesa. O maior de todos depois de Tiamath.
		{head: "helm", helm: pc(70, 30, 22), plume: pc(255, 140, 40), skin: pc(110, 44, 30), torso: pc(84, 34, 24), torso2: pc(255, 150, 40),
			belt: pc(255, 110, 30), legs: pc(70, 30, 22), arm: pc(100, 40, 28), boots: pc(40, 18, 12), cape: pc(36, 14, 12), chest: pc(140, 50, 26),
			weapon: "sword", wlen: 24, scale: 2.8, aura: true, embers: true, seed: 95},
		// Golem de Cristal: corpo de quartzo violeta, cheio de brilhos.
		{head: "hair", hair: pc(200, 170, 255), skin: pc(170, 150, 230), torso: pc(140, 120, 210), torso2: pc(230, 210, 255), belt: pc(200, 180, 255),
			arm: pc(170, 150, 230), legs: pc(120, 100, 190), boots: pc(90, 76, 150), weapon: "club", wlen: 4, scale: 1.5, aura: true, seed: 96},
	}
}()

// Montarias: a mesma família de desenho dos bichos, com sela para o cavaleiro.
// steedStyles: montarias com escala 1,6, para o cavaleiro sentado ficar
// proporcional (o bicho é maior que a pessoa).
var steedStyles = []bodyStyle{
	{scale: 1.6, beast: 4, blen: 4.5, br: 3.4, bh: 5, leg: 1.1, c1: pc(150, 104, 64), c2: pc(112, 76, 46), mane: pc(70, 46, 30), neck: 5, hr: 2.2, tail: true, ears: true, eye: pc(30, 20, 16), seed: 90},
	{scale: 1.6, beast: 4, blen: 4, br: 3.4, bh: 2.5, leg: 1.1, c1: pc(112, 86, 66), c2: pc(84, 62, 48), hr: 2.4, tusk: true, ears: true, eye: pc(30, 20, 16), seed: 91},
	{scale: 1.6, beast: 2, blen: 1.8, br: 3.4, bh: 8, leg: 0.8, c1: pc(52, 46, 52), c2: pc(232, 212, 190), mane: pc(220, 200, 180), neck: 8, hr: 1.5, tail: true, tailLen: 0.5, eye: pc(20, 20, 20), seed: 92},
	{scale: 1.6, beast: 2, blen: 3.4, br: 2.8, bh: 5, leg: 1.1, c1: pc(176, 126, 72), c2: pc(126, 88, 52), neck: 3, hr: 2.1, tail: true, tailLen: 1.8, eye: pc(255, 210, 60), seed: 93},
}

func saddleH(st *bodyStyle) float64 { return (st.bh + st.br*1.2) * st.sc() }

// drawBeast desenha um bicho com as patas em (b.x, b.y): corpo deitado,
// cabeça para o lado em que anda, patas alternando no passo.
func (g *Game) drawBeast(b bodyDraw) {
	st := b.style
	s, f, T := st.sc(), float64(b.face), b.T
	lx, ly := 0.0, 0.0
	if b.act != nil {
		k := math.Sin(math.Pi*b.act.t) * 3
		lx, ly = math.Cos(b.act.dir)*k, math.Sin(b.act.dir)*k
	}
	e := &body{ox: b.x, oy: b.y + b.sink, clip: 1 << 30, flash: b.flash, tint: b.tint, seed: st.seed, mask: map[int]bool{}, lit: b.lit, edgeMaxY: 1 << 30}
	if b.sink > 0 {
		e.clip = b.y
	}
	C := func(ax, ay, bx, by, r float64, c col) {
		g.capsule(e, ax*s+lx, ay*s+ly, bx*s+lx, by*s+ly, r*s, c)
	}
	moving := b.phase != nil
	ph := 0.0
	if moving {
		ph = *b.phase
	}
	L := st.blen
	pairs := st.beast / 2
	if st.fly {
		pairs = 0
	}
	cy := -st.bh - st.br
	switch {
	case st.fly:
		cy += math.Sin(T*6+float64(st.seed)) * 1.5
	case moving:
		cy += math.Abs(math.Sin(ph*2)) * 0.5
	default:
		cy += math.Sin(T*2.4+float64(st.seed)) * 0.3
	}
	c1, c2 := or(st.c1, col{120, 100, 90}), or(st.c2, col{150, 130, 120})
	if st.fly {
		for dx := -3; dx <= 3; dx++ {
			g.setCell(b.x+dx, b.y, ' ', 0, 0, 0)
		}
		flap := math.Sin(T*16+float64(st.seed)) * 2.5
		for _, side := range []float64{-1, 1} {
			C(side*1, cy, side*5, cy-1.5-flap, 0.7, c2)
			C(side*5, cy-1.5-flap, side*8, cy+1-flap*0.6, 0.5, c2)
		}
	}
	if st.tail {
		tl := st.tailLen
		if tl == 0 {
			tl = 1
		}
		C(-f*L, cy, -f*(L+3.5*tl), cy-2+math.Sin(T*5)*1.2, 0.6, c1)
	}
	for i := 0; i < pairs; i++ {
		px := 0.0
		if pairs > 1 {
			px = -L + 2*L*float64(i)/float64(pairs-1)
		}
		for side := 0; side < 2; side++ {
			sw := 0.0
			if moving {
				sw = math.Sin(ph*1.5 + float64(i)*2.1 + float64(side)*math.Pi)
			}
			lift := math.Max(0, -sw) * 1.2
			if st.arch {
				out := px*1.5 + 2*(0.6+float64(side)*0.5)
				if px < 0 {
					out = px*1.5 - 2*(0.6+float64(side)*0.5)
				}
				C(px, cy, out, cy-st.br*0.7, 0.5, c2)
				C(out, cy-st.br*0.7, out*1.25+sw, -0.4-lift, 0.5, c2)
			} else {
				lx2 := px
				if pairs == 1 {
					lx2 += []float64{-0.9, 0.9}[side]
				}
				legR := st.leg
				if legR == 0 {
					legR = 0.8
				}
				C(lx2, cy, lx2+sw*2*b.hf, -0.5-lift, legR, c2)
			}
		}
	}
	C(-L, cy, L, cy, st.br, c1)
	hxp, hy := f*(L+st.br*0.5), cy-st.br*0.35-st.neck
	if st.neck > 0 {
		C(f*L*0.8, cy-st.br*0.3, hxp, hy, st.br*0.42, or(st.mane, c1))
	}
	hr := st.hr
	if hr == 0 {
		hr = st.br * 0.75
	}
	C(hxp, hy, hxp+f*st.br*0.5, hy+0.4, hr, c2)
	C(hxp+f*st.br*0.5, hy-0.3, hxp+f*st.br*0.5, hy-0.3, 0.5, or(st.eye, col{255, 70, 50}))
	if st.tusk {
		C(hxp+f*(hr+0.6), hy+0.9, hxp+f*(hr+1.6), hy-0.4, 0.4, col{240, 232, 210})
	}
	if st.ears {
		C(hxp-1.2, hy-st.br*0.8, hxp-1.2, hy-st.br*1.2, 0.6, c1)
		C(hxp+1.2, hy-st.br*0.8, hxp+1.2, hy-st.br*1.2, 0.6, c1)
	}
}

func (g *Game) drawSteed(kind, x, y, face int, phase *float64, T float64) {
	g.drawBeast(bodyDraw{style: &steedStyles[clampi(kind, 0, len(steedStyles)-1)], x: x, y: y, face: face, phase: phase, hf: 1, T: T})
}

// drawBody desenha um corpo de gente com os pés em (b.x, b.y).
func (g *Game) drawBody(b bodyDraw) {
	st := b.style
	s, f, act, T := st.sc(), float64(b.face), b.act, b.T
	lx, ly := 0.0, 0.0
	if act != nil && (act.anim == "swing" || act.anim == "thrust") {
		k := math.Sin(math.Pi*act.t) * 2.5
		lx, ly = math.Cos(act.dir)*k, math.Sin(act.dir)*k
	}
	e := &body{ox: b.x, oy: b.y + b.sink, clip: 1 << 30, flash: b.flash, tint: b.tint, seed: st.seed, mask: map[int]bool{}, lit: b.lit, edgeMaxY: 1 << 30}
	if b.seated {
		e.edgeMaxY = b.y + int(-12*st.sc()) // da sela para baixo é o bicho
	}
	if b.sink > 0 {
		e.clip = b.y
	}
	C := func(ax, ay, bx, by, r float64, c col) {
		g.capsule(e, ax*s+lx, ay*s+ly, bx*s+lx, by*s+ly, r*s, c)
	}
	moving := b.phase != nil
	// A respiração e o sobe-desce do passo andam em células inteiras: meia
	// célula de deslocamento troca o caractere do corpo todo a cada quadro e o
	// personagem parece piscar.
	ph, sw, bob := 0.0, 0.0, math.Round(math.Sin(T*1.6+float64(st.seed))*0.6)
	if moving {
		ph = *b.phase
		sw = math.Sin(ph)
		bob = -math.Round(math.Abs(math.Cos(ph)) * 0.8)
	}
	hipY, neckY, headY := -12.0, -21+bob, -25+bob
	// Membros grossos o bastante para não sumir: com a escala dos personagens
	// (charScale), um raio de 1,4 virava uma célula e o braço piscava.
	limb := 1.75
	if st.bone {
		limb = 1.05
	}
	torso, arm, legs, skin, boots := or(st.torso, col{150, 120, 100}), or(st.arm, col{150, 120, 100}), or(st.legs, col{104, 110, 150}), or(st.skin, skinC), or(st.boots, col{96, 72, 56})

	if st.cape != nil {
		C(-f*1.5, neckY+1, -f*(2.5+sw*0.8), -5, 2.6, *st.cape)
	}
	sh := func(i float64) (float64, float64) { return i * 3.5, neckY + 1.8 }
	offx, offy := sh(-f)
	oa := 0.12
	if moving {
		oa = -sw * 0.6
	}
	ohx, ohy := offx-f*math.Sin(oa)*8, offy+math.Cos(oa)*8
	C(offx, offy, ohx, ohy, limb*0.9, arm)
	C(ohx, ohy, ohx, ohy, 1.15, skin)

	if b.seated {
		// Montado: coxas para a frente sobre a sela, canelas descendo pelo
		// flanco; no manto, só a barra cai dos dois lados.
		for _, i := range []float64{-f, f} {
			hxp := i * 1.4
			kx, ky := hxp+f*3.4, hipY+1.2
			fx, fy := kx-f*0.6, hipY+7
			if st.robe {
				C(hxp, hipY+0.5, kx, ky, 2.2, torso)
				C(kx, ky, fx, fy-1.5, 1.6, torso)
			} else {
				C(hxp, hipY, kx, ky, limb*1.05, legs)
				C(kx, ky, fx, fy, limb*0.9, legs)
			}
			C(fx, fy+0.2, fx+f*1.3, fy+0.2, 1.15, boots)
		}
	} else if st.robe {
		for y := hipY; y <= -1; y++ {
			k := (y - hipY) / (-1 - hipY)
			half, sway := 3.1+2.7*k, sw*0.9*k
			C(-half+sway, y, half+sway, y, 0.6, torso)
		}
		C(-2+sw, -0.4, -1+sw, -0.4, 1.05, boots)
		C(1-sw, -0.4, 2-sw, -0.4, 1.05, boots)
	} else {
		for _, i := range []float64{-f, f} {
			lp := ph
			if i != f {
				lp += math.Pi
			}
			swing, lift := 0.0, 0.0
			if moving {
				swing, lift = math.Sin(lp), math.Max(0, math.Cos(lp))*2.4
			}
			hxp := i * 1.6
			fx, fy := hxp+swing*3.6*b.hf, -lift-0.6
			kx, ky := (hxp+fx)/2+f*lift*0.6, (hipY+fy)/2-lift*0.3
			C(hxp, hipY, kx, ky, limb*1.05, legs)
			C(kx, ky, fx, fy, limb*0.9, legs)
			C(fx, fy+0.2, fx+f*1.5, fy+0.2, 1.15, boots)
		}
	}

	if st.bone {
		C(0, hipY, 0, neckY, 0.8, torso)
		for _, y := range []float64{-20, -18, -16} {
			C(-3, y+bob, 3, y+bob, 0.55, torso)
		}
		C(-2, hipY, 2, hipY, 0.9, torso)
		if st.chest != nil {
			C(0, -19.5+bob, 0, -16.5+bob, 2.7, *st.chest)
		}
	} else {
		C(0, hipY+bob*0.5, 0, neckY, 3.3, torso)
		// Ombros largos e cintura mais fina: a silhueta em V lê como gente
		// mesmo pequena (antes o tronco era um tubo só).
		C(-2.6, neckY+1.6, 2.6, neckY+1.6, 1.9, torso)
		if st.torso2 != nil {
			C(0, hipY+1, 0, neckY+1, 0.7, *st.torso2)
		}
		if st.belt != nil {
			C(-3, hipY+0.6, 3, hipY+0.6, 0.6, *st.belt)
		}
	}

	// Cabeça.
	skull := len(st.head) >= 5 && st.head[:5] == "skull"
	headC := skin
	if st.head == "helm" {
		headC = or(st.helm, steel)
	}
	C(0, headY, 0, headY, 3.7, headC)
	switch {
	case skull:
		C(-1.5+f*0.5, headY-0.3, -1.5+f*0.5, headY-0.3, 0.8, dark)
		C(1.5+f*0.5, headY-0.3, 1.5+f*0.5, headY-0.3, 0.8, dark)
		C(-1.2, headY+2.7, 1.2, headY+2.7, 0.5, dark)
		if st.head == "skullhelm" {
			C(-2.6, headY-2.4, 2.6, headY-2.4, 1.1, col{132, 138, 168})
		}
		if st.head == "skullcrown" {
			for _, dx := range []float64{-2.6, 0, 2.6} {
				C(dx, headY-3, dx, headY-5.4, 0.6, gold)
			}
		}
	case st.head == "helm":
		C(-1.6+f*0.9, headY+0.3, 1.6+f*0.9, headY+0.3, 0.5, dark)
		C(0, headY-3.6, -f*3, headY-5, 0.9, or(st.plume, white))
	default:
		C(f*0.7-1.3, headY+0.2, f*0.7-1.3, headY+0.2, 0.5, dark)
		C(f*0.7+1.3, headY+0.2, f*0.7+1.3, headY+0.2, 0.5, dark)
		hair := or(st.hair, col{90, 70, 50})
		if st.head == "hat" {
			hat := or(st.hat, col{90, 60, 160})
			C(-5, headY-2, 5, headY-2, 0.7, hat)
			for k := 0.0; k < 8; k++ {
				h := 3.1 * (1 - k/8)
				C(-h-f*k*0.3, headY-3-k, h-f*k*0.3, headY-3-k, 0.6, hat)
			}
		} else {
			C(-1.6, headY-2, 1.6, headY-2, 2, hair)
			C(-f*2.7, headY-1, -f*2.7, headY+1.6, 0.9, hair)
			if st.head == "circlet" {
				C(-2.8, headY-1.6, 2.8, headY-1.6, 0.5, gold)
			}
		}
	}

	// Braço da arma: em repouso a arma fica erguida; no golpe ela varre,
	// estoca, aponta ou sobe, conforme a animação.
	wsx, wsy := sh(f)
	armA, wA, armLen, glow := math.Pi/2-f*0.35, -math.Pi/2+f*0.3, 7.5, 0.0
	if moving {
		armA = math.Pi/2 - f*(0.35+sw*0.4)
	}
	if act != nil {
		t := act.t
		switch act.anim {
		case "swing":
			armA = act.dir + f*(-1.5+2.9*ease(t))
			wA = armA
		case "spin":
			armA = act.dir + f*t*2*tau
			wA = armA
		case "thrust":
			armA, wA, armLen = act.dir, act.dir, 5+5*math.Sin(math.Pi*t)
		case "point":
			armA, wA, glow = act.dir, act.dir, 1-t
		case "cast":
			armA, wA, glow = -math.Pi/2+f*0.5, -math.Pi/2+f*0.1, math.Sin(math.Pi*t)
		}
	}
	hx, hy := wsx+math.Cos(armA)*armLen, wsy+math.Sin(armA)*armLen
	wc, wsn := math.Cos(wA), math.Sin(wA)
	tx, ty := hx+wc*st.wlen, hy+wsn*st.wlen
	blade := col{236, 240, 255}
	if b.wcol != nil {
		blade = *b.wcol
	}
	switch st.weapon {
	case "sword":
		C(hx, hy, tx, ty, 0.75, blade)
		C(hx-wsn*1.7, hy+wc*1.7, hx+wsn*1.7, hy-wc*1.7, 0.55, gold)
	case "staff":
		C(hx-wc*5, hy-wsn*5, tx, ty, 0.6, wood)
		orb := or(st.orb, col{150, 190, 255})
		if b.wcol != nil {
			orb = *b.wcol
		}
		C(tx, ty, tx, ty, 1.7+glow, mix3(orb, white, 0.3+0.7*glow+0.1*math.Sin(T*2)))
	case "lance":
		C(hx-wc*6, hy-wsn*6, tx, ty, 0.65, wood)
		C(tx-wc*4, ty-wsn*4, tx, ty, 0.9, blade)
	case "hammer":
		C(hx, hy, tx, ty, 0.6, wood)
		head := gold
		if b.wcol != nil {
			head = *b.wcol
		}
		C(tx-wsn*2.3, ty+wc*2.3, tx+wsn*2.3, ty-wc*2.3, 1.7, mix3(head, white, 0.4+0.6*glow))
	case "club": // pedaço de pau: galho torto, mais grosso na ponta, com um nó
		mx, my := (hx+tx)/2+wsn*0.8, (hy+ty)/2-wc*0.8
		C(hx, hy, mx, my, 0.6, wood)
		C(mx, my, tx, ty, 0.85, mix3(wood, col{70, 50, 30}, 0.3))
		C(tx, ty, tx, ty, 1.1, col{110, 76, 44})
	case "bow":
		fx, fy := hx+wc*2.5, hy+wsn*2.5
		C(hx-wsn*6, hy+wc*6, fx, fy, 0.6, wood)
		C(hx+wsn*6, hy-wc*6, fx, fy, 0.6, wood)
		C(hx-wsn*6, hy+wc*6, hx+wsn*6, hy-wc*6, 0.3, blade)
	default:
		C(hx, hy, tx, ty, 0.85, col{200, 170, 130})
	}
	C(wsx, wsy, hx, hy, limb*0.9, arm)
	C(hx, hy, hx, hy, 1.15, skin)

	if st.shield != nil {
		if b.guard || act != nil && act.anim == "block" {
			C(f*2, -16+bob, f*2, -14+bob, 5.2, mix3(*st.shield, white, 0.35))
		} else {
			C(ohx, ohy-1, ohx, ohy-1, 3.4, *st.shield)
		}
	}

	// Rastro da lâmina e brilho da magia, por cima de tudo.
	cx, cy := float64(b.x)+wsx*s+lx, float64(b.y)+wsy*s+ly
	reach := (armLen + st.wlen) * s
	if act != nil {
		switch act.anim {
		case "swing":
			a0 := act.dir - f*1.5
			for k := 0; k < 3; k++ {
				g.ring(cx, cy, reach-float64(k)*1.5, a0, armA, mix3(col{255, 255, 255}, col{120, 130, 170}, float64(k)/3+act.t*0.4))
			}
		case "spin":
			for k := 0; k < 3; k++ {
				g.ring(cx, cy, reach-float64(k)*1.5, armA-f*2.6, armA, mix3(col{255, 255, 255}, col{150, 160, 200}, float64(k)/3))
			}
		case "thrust":
			g.streak(cx, cy, cx+math.Cos(act.dir)*(reach+4), cy+math.Sin(act.dir)*(reach+4), col{220, 226, 255})
		}
	}
	if b.wcol != nil && rand.Float64() < 0.12 {
		g.spawn("spark", float64(b.x)+tx*s, float64(b.y)+ty*s, (rand.Float64()-0.5)*8, -4-rand.Float64()*6, 0.4)
	}
	if glow > 0 && rand.Float64() < 0.6 {
		kind := "arcane"
		if st.weapon == "hammer" {
			kind = "holy"
		}
		g.spawn(kind, float64(b.x)+tx*s, float64(b.y)+ty*s, (rand.Float64()-0.5)*14, (rand.Float64()-0.5)*14, 0.4)
	}
	if glow > 0 {
		g.light(float64(b.x)+tx*s, float64(b.y)+ty*s, 26, glow*0.8, col{0.4, 0.5, 1.3}, col{20, 30, 90})
	}
}
