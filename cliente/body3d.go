package main

import (
	"math"
	"runtime"
	"sync"
)

// Personagens em 3D: o corpo é um modelo de verdade, feito de formas
// arredondadas (cápsulas, esferas, cones) presas a um esqueleto que anda e
// golpeia. Cada célula da grade lança um raio contra o modelo; onde acerta,
// a luz (chave, ambiente e contorno) dá o brilho e o brilho escolhe o
// caractere. É o mesmo princípio do jogo de referência: renderizar em 3D e
// converter os pixels em caracteres.

type v3 struct{ x, y, z float64 }

func (a v3) add(b v3) v3          { return v3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a v3) sub(b v3) v3          { return v3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a v3) mul(k float64) v3     { return v3{a.x * k, a.y * k, a.z * k} }
func (a v3) dot(b v3) float64     { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a v3) len() float64         { return math.Sqrt(a.dot(a)) }
func (a v3) norm() v3             { return a.mul(1 / math.Max(1e-9, a.len())) }
func (a v3) cross(b v3) v3        { return v3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x} }
func lerp3(a, b v3, t float64) v3 { return a.add(b.sub(a).mul(t)) }

// Formas do modelo.
const (
	shCapsule = iota // segmento a-b com raio r1
	shCone           // cone arredondado: raio r1 em a, r2 em b
	shBox            // caixa centrada em a, meias medidas em b, girada pelo eixo dir
)

type prim struct {
	kind   int
	a, b   v3
	r1, r2 float64
	mat    int
	// caixa: eixos locais
	ax, ay, az v3
}

type model struct {
	prims []prim
	mats  []col
	glow  []bool // material que brilha sozinho (orbe do cajado)
	minY  float64
	maxY  float64
	rad   float64 // raio horizontal máximo, para cortar raios que passam longe
}

func (m *model) mat(c col) int {
	m.mats = append(m.mats, c)
	m.glow = append(m.glow, false)
	return len(m.mats) - 1
}

func (m *model) capsule(a, b v3, r float64, mat int) {
	m.prims = append(m.prims, prim{kind: shCapsule, a: a, b: b, r1: r, mat: mat})
}

func (m *model) cone(a, b v3, ra, rb float64, mat int) {
	m.prims = append(m.prims, prim{kind: shCone, a: a, b: b, r1: ra, r2: rb, mat: mat})
}

func (m *model) box(c, half, dir, up v3, mat int) {
	z := dir.norm()
	x := up.cross(z).norm()
	y := z.cross(x)
	m.prims = append(m.prims, prim{kind: shBox, a: c, b: half, mat: mat, ax: x, ay: y, az: z})
}

func sdCapsule(p, a, b v3, r float64) float64 {
	pa, ba := p.sub(a), b.sub(a)
	h := clamp(pa.dot(ba)/math.Max(1e-9, ba.dot(ba)), 0, 1)
	return pa.sub(ba.mul(h)).len() - r
}

// sdRoundCone é o cone de pontas arredondadas (fórmula de Inigo Quilez).
func sdRoundCone(p, a, b v3, r1, r2 float64) float64 {
	ba := b.sub(a)
	l2 := ba.dot(ba)
	rr := r1 - r2
	a2 := l2 - rr*rr
	il2 := 1 / l2
	pa := p.sub(a)
	y := pa.dot(ba)
	z := y - l2
	xv := pa.mul(l2).sub(ba.mul(y))
	x2 := xv.dot(xv)
	y2 := y * y * l2
	z2 := z * z * l2
	k := math.Copysign(1, rr) * rr * rr * x2
	if math.Copysign(1, z)*a2*z2 > k {
		return math.Sqrt(x2+z2)*il2 - r2
	}
	if math.Copysign(1, y)*a2*y2 < k {
		return math.Sqrt(x2+y2)*il2 - r1
	}
	return (math.Sqrt(x2*a2*il2)+y*rr)*il2 - r1
}

func sdBox(p v3, pr *prim) float64 {
	d := p.sub(pr.a)
	q := v3{math.Abs(d.dot(pr.ax)) - pr.b.x, math.Abs(d.dot(pr.ay)) - pr.b.y, math.Abs(d.dot(pr.az)) - pr.b.z}
	out := v3{math.Max(q.x, 0), math.Max(q.y, 0), math.Max(q.z, 0)}.len()
	return out + math.Min(math.Max(q.x, math.Max(q.y, q.z)), 0) - 0.3
}

// sdf devolve a distância até as formas da lista e o material mais próximo.
func (m *model) sdf(p v3, list []int) (float64, int) {
	best, mat := 1e9, 0
	for _, i := range list {
		pr := &m.prims[i]
		var d float64
		switch pr.kind {
		case shCapsule:
			d = sdCapsule(p, pr.a, pr.b, pr.r1)
		case shCone:
			d = sdRoundCone(p, pr.a, pr.b, pr.r1, pr.r2)
		default:
			d = sdBox(p, pr)
		}
		if d < best {
			best, mat = d, pr.mat
		}
	}
	return best, mat
}

// ---------- pose ----------

// pose3d monta o modelo de uma pessoa na pose do quadro: passo, golpe,
// respiração. O modelo olha para +z; quem gira é a câmera (yaw).
func pose3d(st *bodyStyle, phase *float64, act *action, T float64, wcol *col, guard, seated bool) *model {
	m := &model{}
	moving := phase != nil
	ph := 0.0
	if moving {
		ph = *phase
	}
	bob := math.Sin(T*2.4+float64(st.seed)) * 0.25
	if moving {
		bob = -math.Abs(math.Cos(ph)) * 0.7
	}
	matOr := func(c *col, d col) int {
		if c == nil {
			return m.mat(d)
		}
		return m.mat(*c)
	}
	skinM := matOr(st.skin, skinC)
	torsoM := matOr(st.torso, col{150, 120, 100})
	legsM := matOr(st.legs, col{104, 110, 150})
	armM := matOr(st.arm, col{150, 120, 100})
	bootsM := matOr(st.boots, col{96, 72, 56})
	darkM := m.mat(dark)

	lk := 1.0 // espessura dos membros: o esqueleto é só osso
	if st.bone {
		lk = 0.5
	}
	pelvis := v3{0, 12.5 + bob, 0}
	chest := v3{0, 19.5 + bob, 0}
	head := v3{0, 24.6 + bob, 0}

	// Pernas (ou a saia do manto).
	if seated {
		// Montado: coxas para a frente, canelas descendo pelos flancos.
		for side := -1.0; side <= 1; side += 2 {
			hip := pelvis.add(v3{side * 1.8, -0.3, 0})
			knee := hip.add(v3{side * 2.4, -0.6, 4.6})
			foot := knee.add(v3{side * 0.3, -5.6, -0.8})
			lm := legsM
			if st.robe {
				lm = torsoM
			}
			m.capsule(hip, knee, 1.5*lk, lm)
			m.capsule(knee, foot, 1.3*lk, lm)
			m.capsule(foot, foot.add(v3{0, -0.2, 1.4}), 0.85, bootsM)
		}
	} else if st.robe {
		sw := math.Sin(ph) * 0.6
		// Manto até o chão, abrindo em sino, com a barra mais escura.
		m.cone(pelvis.add(v3{0, 1.5, 0}), v3{sw * 0.6, 1.4, sw * 0.4}, 3.0, 5.6, torsoM)
		m.cone(v3{sw * 0.6, 1.3, sw * 0.4}, v3{sw * 0.6, 0.5, sw * 0.4}, 5.6, 5.8, m.mat(mix3(m.mats[torsoM], dark, 0.45)))
		m.capsule(v3{-1.3 + sw, 0.6, 0.8}, v3{-1.3 + sw, 0.6, 1.8}, 0.9, bootsM)
		m.capsule(v3{1.3 - sw, 0.6, 0.8}, v3{1.3 - sw, 0.6, 1.8}, 0.9, bootsM)
	} else {
		for side := -1.0; side <= 1; side += 2 {
			lp := ph
			if side > 0 {
				lp += math.Pi
			}
			a, k := 0.0, 0.12
			if moving {
				a = math.Sin(lp) * 0.6
				k = math.Max(0, math.Sin(lp+math.Pi/2)) * 0.95
			}
			hip := pelvis.add(v3{side * 1.6, -0.3, 0})
			knee := hip.add(v3{0, -math.Cos(a), math.Sin(a)}.mul(6))
			foot := knee.add(v3{0, -math.Cos(a - k), math.Sin(a - k)}.mul(5.8))
			if foot.y < 0.8 {
				foot.y = 0.8
			}
			m.capsule(hip, knee, 1.45*lk, legsM)
			m.capsule(knee, foot, 1.25*lk, legsM)
			m.capsule(foot.add(v3{0, -0.2, -0.3}), foot.add(v3{0, -0.2, 1.3}), 0.85, bootsM)
		}
	}

	// Tronco: largo nos ombros, fino na cintura; faixa e cinto por cima. No
	// esqueleto, coluna, costelas e bacia de osso.
	if st.bone {
		m.capsule(pelvis, chest.add(v3{0, 1.5, 0}), 0.7, torsoM)
		for k := 0; k < 3; k++ {
			ry := chest.y - 0.4 - float64(k)*1.6
			w := 2.8 - float64(k)*0.35
			m.capsule(v3{-w, ry, 0.4}, v3{0, ry - 0.5, 1.3}, 0.45, torsoM)
			m.capsule(v3{w, ry, 0.4}, v3{0, ry - 0.5, 1.3}, 0.45, torsoM)
		}
		m.capsule(pelvis.add(v3{-2.2, 0, 0}), pelvis.add(v3{2.2, 0, 0}), 0.9, torsoM)
		if st.chest != nil {
			m.box(chest.add(v3{0, -1.6, 1.2}), v3{2.8, 2.4, 0.5}, v3{0, 0, 1}, v3{0, 1, 0}, m.mat(*st.chest))
		}
	} else {
		m.cone(pelvis, chest, 2.5, 3.4, torsoM)
	}
	if st.torso2 != nil {
		m.capsule(pelvis.add(v3{0, 1, 2.1}), chest.add(v3{0, 0, 2.6}), 0.55, m.mat(*st.torso2))
	}
	if st.belt != nil {
		m.capsule(pelvis.add(v3{-2.6, 0.8, 0}), pelvis.add(v3{2.6, 0.8, 0}), 0.75, m.mat(*st.belt))
	}
	if st.cape != nil {
		sw := 0.0
		if moving {
			sw = math.Sin(ph) * 0.5
		}
		m.box(chest.add(v3{0, -5.5, -2.8 - sw*0.3}), v3{3.1, 6.2, 0.35}, v3{0, -0.15, -1}, v3{0, 1, 0}, m.mat(*st.cape))
	}
	m.capsule(chest.add(v3{0, 1, 0}), head.add(v3{0, -2.6, 0}), 1.1, skinM) // pescoço

	// Cabeça e o que vai nela.
	headM := skinM
	if st.head == "helm" {
		headM = matOr(st.helm, steel)
	}
	skull := len(st.head) >= 5 && st.head[:5] == "skull"
	m.capsule(head, head.add(v3{0, 0.4, 0}), 2.9, headM)
	switch st.head {
	case "skull", "skullhelm", "skullcrown":
		// Caveira: órbitas fundas que brilham fraco, maxilar à mostra.
		sock := m.mat(col{255, 120, 60})
		m.glow[sock] = true
		m.mats[sock] = col{140, 40, 20}
		m.capsule(head.add(v3{-1.1, 0.2, 2.4}), head.add(v3{-1.1, 0.2, 2.5}), 0.75, sock)
		m.capsule(head.add(v3{1.1, 0.2, 2.4}), head.add(v3{1.1, 0.2, 2.5}), 0.75, sock)
		m.box(head.add(v3{0, -2.4, 1.2}), v3{1.6, 0.6, 1.1}, v3{0, 0, 1}, v3{0, 1, 0}, headM)
		if st.head == "skullhelm" {
			m.cone(head.add(v3{0, 1.2, 0}), head.add(v3{0, 3.4, -0.3}), 3.2, 1.4, m.mat(col{132, 138, 168}))
		}
		if st.head == "skullcrown" {
			for _, dx := range []float64{-2, 0, 2} {
				m.cone(head.add(v3{dx, 2.4, 0.3}), head.add(v3{dx * 1.1, 5.2, 0.2}), 0.6, 0.15, m.mat(gold))
			}
		}
	case "helm":
		m.box(head.add(v3{0, 0.1, 2.6}), v3{1.8, 0.35, 0.3}, v3{0, 0, 1}, v3{0, 1, 0}, darkM) // viseira
		m.capsule(head.add(v3{0, 3, -0.5}), head.add(v3{0, 4, -3.2}), 0.9, matOr(st.plume, white))
	case "hood":
		m.cone(head.add(v3{0, 0.6, -0.8}), head.add(v3{0, 3.8, -2.2}), 3.4, 1.2, matOr(st.hair, col{60, 50, 70}))
	case "hat":
		hat := matOr(st.hat, col{90, 60, 160})
		// Chapéu de bruxo: aba larga e caída, copa alta que entorta para
		// trás no meio, faixa escura; barba longa e grisalha embaixo.
		// Aba: disco fino de raios e anel, caindo um pouco na borda.
		const spokes = 14
		for k := 0; k < spokes; k++ {
			a0, a1 := float64(k)*tau/spokes, float64(k+1)*tau/spokes
			rim0 := head.add(v3{6.2 * math.Cos(a0), 1.3, 6.2 * math.Sin(a0)})
			rim1 := head.add(v3{6.2 * math.Cos(a1), 1.3, 6.2 * math.Sin(a1)})
			m.capsule(head.add(v3{0, 2.3, 0}), rim0, 0.55, hat)
			m.capsule(rim0, rim1, 0.6, hat)
		}
		mid := head.add(v3{-0.3, 7.5, -0.9})
		m.cone(head.add(v3{0, 2.4, 0}), mid, 3.5, 1.9, hat)
		m.cone(mid, mid.add(v3{-2.2, 3.2, -3.4}), 1.9, 0.35, hat)
		m.cone(head.add(v3{0, 2.6, 0}), head.add(v3{0, 3.4, 0}), 3.6, 3.4, darkM) // faixa
		beard := m.mat(col{196, 196, 204})
		m.cone(head.add(v3{0, -1.2, 2.0}), head.add(v3{0, -7.2, 2.9}), 2.4, 0.5, beard)
		m.capsule(head.add(v3{-1.4, -0.8, 2.5}), head.add(v3{1.4, -0.8, 2.5}), 0.6, beard) // bigode
	default:
		hair := matOr(st.hair, col{90, 70, 50})
		m.capsule(head.add(v3{0, 1, -0.7}), head.add(v3{0, 1.6, -1.4}), 2.75, hair)
		if st.head == "circlet" {
			m.capsule(head.add(v3{-2.9, 1.2, 0.4}), head.add(v3{2.9, 1.2, 0.4}), 0.45, m.mat(gold))
		}
	}
	if st.head != "helm" && !skull {
		m.capsule(head.add(v3{-1.1, 0.3, 2.65}), head.add(v3{-1.1, 0.3, 2.66}), 0.45, darkM) // olhos
		m.capsule(head.add(v3{1.1, 0.3, 2.65}), head.add(v3{1.1, 0.3, 2.66}), 0.45, darkM)
	}

	// Braços: o esquerdo balança; o direito segura a arma e faz o golpe.
	shoulderL, shoulderR := chest.add(v3{-3.5, 0.6, 0}), chest.add(v3{3.5, 0.6, 0})
	swing := 0.0
	if moving {
		swing = math.Sin(ph) * 0.55
	}
	elbowL := shoulderL.add(v3{-0.4, -math.Cos(swing), -math.Sin(swing)}.mul(4.4))
	handL := elbowL.add(v3{0, -math.Cos(swing*1.4) * 0.9, -math.Sin(swing*1.4) + 0.3}.norm().mul(4.2))
	m.capsule(shoulderL, elbowL, 1.15*lk, armM)
	sleeve := func(a, b v3) { // no manto, a manga abre como sino até o pulso
		if st.robe {
			m.cone(a, b, 1.1, 2.0, armM)
		} else {
			m.capsule(a, b, 1.0*lk, armM)
		}
	}
	sleeve(elbowL, handL)
	m.capsule(handL, handL, 1.05, skinM)

	// Direção do braço da arma em repouso: um pouco à frente; no golpe, varre.
	armDir := v3{0.25, -math.Cos(-swing), math.Sin(-swing) + 0.25}.norm()
	wDir := v3{0.1, 0.85, 0.5}.norm()
	armLen, glow := 8.4, 0.0
	if act != nil {
		t := act.t
		switch act.anim {
		case "swing":
			th := -1.6 + 3.0*ease(t)
			armDir = v3{math.Sin(th), 0.05, math.Cos(th)}.norm()
			wDir = armDir
		case "spin":
			th := t * 2 * tau
			armDir = v3{math.Sin(th), 0, math.Cos(th)}
			wDir = armDir
		case "thrust":
			armDir, wDir = v3{0, 0.05, 1}, v3{0, 0, 1}
			armLen = 6.5 + 4*math.Sin(math.Pi*t)
		case "point":
			armDir, wDir, glow = v3{0, 0.1, 1}.norm(), v3{0, 0.2, 1}.norm(), 1-t
		case "cast":
			armDir, wDir, glow = v3{0.3, 1, 0.3}.norm(), v3{0, 1, 0.1}.norm(), math.Sin(math.Pi*t)
		case "block":
			armDir = v3{-0.3, 0, 1}.norm()
		}
	}
	elbowR := shoulderR.add(armDir.mul(armLen * 0.5)).add(v3{0, -0.6, 0})
	handR := shoulderR.add(armDir.mul(armLen))
	m.capsule(shoulderR, elbowR, 1.15*lk, armM)
	sleeve(elbowR, handR)
	m.capsule(handR, handR, 1.05, skinM)

	blade := col{220, 226, 245}
	if wcol != nil {
		blade = *wcol
	}
	tip := handR.add(wDir.mul(st.wlen))
	switch st.weapon {
	case "sword":
		m.capsule(handR, tip, 0.55, m.mat(blade))
		side := wDir.cross(v3{0, 0, 1})
		if side.len() < 0.1 {
			side = v3{1, 0, 0}
		}
		side = side.norm()
		m.capsule(handR.sub(side.mul(1.6)), handR.add(side.mul(1.6)), 0.5, m.mat(gold))
	case "staff":
		m.capsule(handR.sub(wDir.mul(5)), tip, 0.5, m.mat(wood))
		orb := matOr(st.orb, col{150, 190, 255})
		if wcol != nil {
			orb = m.mat(*wcol)
		}
		m.glow[orb] = true
		m.mats[orb] = mix3(m.mats[orb], white, 0.25+0.6*glow)
		m.capsule(tip, tip, 1.6+glow*0.8, orb)
	case "lance":
		m.capsule(handR.sub(wDir.mul(6)), tip, 0.55, m.mat(wood))
		m.cone(tip.sub(wDir.mul(3.5)), tip.add(wDir.mul(0.8)), 0.95, 0.15, m.mat(blade))
	case "hammer":
		m.capsule(handR, tip, 0.5, m.mat(wood))
		head := gold
		if wcol != nil {
			head = *wcol
		}
		m.box(tip, v3{2.3, 1.4, 1.4}, v3{1, 0, 0}, wDir, m.mat(mix3(head, white, 0.3*glow)))
	case "club":
		mid := lerp3(handR, tip, 0.5).add(v3{0.6, 0.4, 0})
		m.capsule(handR, mid, 0.5, m.mat(wood))
		m.cone(mid, tip, 0.6, 0.95, m.mat(col{110, 76, 44}))
	case "bow":
		up := v3{0, 1, 0}
		mid := handR.add(v3{0, 0, 1.2})
		m.capsule(mid, mid.add(up.mul(5.5)).add(v3{0, 0, -1.5}), 0.45, m.mat(wood))
		m.capsule(mid, mid.add(up.mul(-5.5)).add(v3{0, 0, -1.5}), 0.45, m.mat(wood))
		m.capsule(mid.add(v3{0, 5.4, -1.5}), mid.add(v3{0, -5.4, -1.5}), 0.15, m.mat(blade))
	default:
		m.capsule(handR, tip, 0.8, m.mat(col{200, 170, 130}))
	}
	if st.shield != nil {
		c := handL.add(v3{0, 1, 1.4})
		dir := v3{0, 0, 1}
		if guard || act != nil && act.anim == "block" {
			c = chest.add(v3{-0.5, -2, 4})
		}
		m.box(c, v3{2.6, 3.2, 0.35}, dir, v3{0, 1, 0}, m.mat(*st.shield))
	}
	m.minY, m.maxY, m.rad = -1, 38, 16
	return m
}

// ---------- câmera e desenho ----------

// A câmera olha de frente, inclinada para baixo, como a vista do mundo.
const camPitch = 0.42

var (
	camFwd = v3{0, -math.Sin(camPitch), -math.Cos(camPitch)} // para onde a câmera olha
	camUp  = v3{0, math.Cos(camPitch), -math.Sin(camPitch)}
	keyL   = v3{-0.55, 0.75, 0.6}.norm() // luz do alto, à esquerda e à frente
)

type r3cell struct {
	dx, dy  int
	code    byte
	r, g, b float64
}

// r3res é um modelo já desenhado, esperando para entrar na grade.
type r3res struct {
	x, y           int
	dx0, dy0, w, h int
	hits           []bool
	cells          []r3cell
}

// renderModel desenha o modelo com os pés em (x, y), girado em yaw (0 olha
// para a câmera, π/2 para a direita da tela). Só lê o estado do jogo, então
// vários modelos podem ser desenhados ao mesmo tempo, um por núcleo.
func (g *Game) renderModel(m *model, x, y int, yaw, scale, flash float64, tint col, seed, clip int) *r3res {
	cy, sy := math.Cos(-yaw), math.Sin(-yaw)
	toModel := func(p v3) v3 { return v3{p.x*cy + p.z*sy, p.y, -p.x*sy + p.z*cy} }
	toWorld := func(n v3) v3 { return v3{n.x*cy - n.z*sy, n.y, n.x*sy + n.z*cy} }
	fwd := toModel(camFwd)

	// Cada forma projetada na tela vira um retângulo (u, v); um raio só testa
	// as formas cujo retângulo o contém.
	type box2 struct{ u0, u1, v0, v1 float64 }
	boxes := make([]box2, len(m.prims))
	all := box2{1e9, -1e9, 1e9, -1e9}
	proj := func(q v3) (float64, float64) { w := toWorld(q); return w.x, w.dot(camUp) }
	for i, pr := range m.prims {
		r := math.Max(pr.r1, pr.r2)
		ends := []v3{pr.a, pr.b}
		if pr.kind == shBox {
			r = pr.b.len() + 0.3
			ends = []v3{pr.a}
		}
		bx := box2{1e9, -1e9, 1e9, -1e9}
		for _, q := range ends {
			u, v := proj(q)
			bx.u0, bx.u1 = math.Min(bx.u0, u-r), math.Max(bx.u1, u+r)
			bx.v0, bx.v1 = math.Min(bx.v0, v-r), math.Max(bx.v1, v+r)
		}
		boxes[i] = bx
		all.u0, all.u1 = math.Min(all.u0, bx.u0), math.Max(all.u1, bx.u1)
		all.v0, all.v1 = math.Min(all.v0, bx.v0), math.Max(all.v1, bx.v1)
	}
	res := &r3res{x: x, y: y}
	res.dx0, res.dy0 = int(math.Floor(all.u0*scale))-2, int(math.Floor(-all.v1*scale))-2
	res.w, res.h = int(math.Ceil(all.u1*scale))+2-res.dx0+1, int(math.Ceil(-all.v0*scale))+2-res.dy0+1
	if res.w <= 0 || res.h <= 0 || res.w*res.h > 400*300 {
		return res
	}
	res.hits = make([]bool, res.w*res.h)
	mm := g.m
	t0, t1 := math.Max(0, 40-m.rad*1.6), 40+m.rad*1.6
	list := make([]int, 0, len(m.prims))
	// Modelo enorme (Tiamath): um raio a cada duas células, pintando o bloco
	// 2×2 inteiro; a diferença não se vê e o custo cai a um quarto.
	step := 1
	if m.rad*scale > 50 {
		step = 2
	}
	for j := 0; j < res.h; j += step {
		dy := res.dy0 + j
		wy := y + dy
		if wy > clip || wy-g.y0 < 0 || wy-g.y0 >= g.rows {
			continue
		}
		for i := 0; i < res.w; i += step {
			dx := res.dx0 + i
			wx := x + dx
			if wx-g.x0 < 0 || wx-g.x0 >= g.cols || wx < 0 || wy < 0 || wx >= mm.w || wy >= mm.h || g.hides(wx, wy, y) {
				continue // fora da tela ou atrás de uma copa
			}
			// A célula (dx, dy) corresponde a (dx, -dy) em unidades do modelo.
			u, v := (float64(dx)+0.5)/scale, (-float64(dy)+0.5)/scale
			list = list[:0]
			for k, bx := range boxes {
				if u >= bx.u0 && u <= bx.u1 && v >= bx.v0 && v <= bx.v1 {
					list = append(list, k)
				}
			}
			if len(list) == 0 {
				continue
			}
			o := toModel(v3{u, 0, 0}.add(camUp.mul(v)).sub(camFwd.mul(40)))
			t, hit, mat := t0, false, 0
			for it := 0; it < 56 && t < t1; it++ {
				d, mt := m.sdf(o.add(fwd.mul(t)), list)
				if d < 0.04 {
					hit, mat = true, mt
					break
				}
				t += math.Max(d, 0.05)
			}
			if !hit {
				continue
			}
			p := o.add(fwd.mul(t))
			const e = 0.08
			k1, _ := m.sdf(p.add(v3{e, -e, -e}), list)
			k2, _ := m.sdf(p.add(v3{-e, -e, e}), list)
			k3, _ := m.sdf(p.add(v3{-e, e, -e}), list)
			k4, _ := m.sdf(p.add(v3{e, e, e}), list)
			n := toWorld(v3{k1 - k2 - k3 + k4, -k1 - k2 + k3 + k4, -k1 + k2 - k3 + k4}.norm())
			for bj := j; bj < min(j+step, res.h); bj++ {
				for bi := i; bi < min(i+step, res.w); bi++ {
					res.hits[bj*res.w+bi] = true
				}
			}
			c := m.mats[mat]
			var r, gg, b float64
			if m.glow[mat] {
				r, gg, b = c[0], c[1], c[2]
			} else {
				diff := math.Max(0, n.dot(keyL))
				rim := math.Pow(1-math.Max(0, n.dot(camFwd.mul(-1))), 3) // luz de contorno, como no vídeo
				spec := math.Pow(math.Max(0, n.dot(keyL.add(camFwd.mul(-1)).norm())), 18) * 0.35
				l := 0.42 + 0.85*diff
				r, gg, b = c[0]*l+90*rim+255*spec, c[1]*l+80*rim+255*spec, c[2]*l+110*rim+255*spec
			}
			if flash > 0 {
				r += (tint[0] - r) * flash
				gg += (tint[1] - gg) * flash
				b += (tint[2] - b) * flash
			}
			hn := hash2(dx, dy, seed)
			// Dentro do corpo o caractere é sempre cheio; a luz muda a cor e a densidade.
			lum := 0.35 + (0.299*r+0.587*gg+0.114*b)/255*0.75 + (float64(hn&15)/15-0.5)*0.1
			for bj := 0; bj < step && j+bj < res.h; bj++ {
				for bi := 0; bi < step && i+bi < res.w; bi++ {
					res.cells = append(res.cells, r3cell{dx + bi, dy + bj, rampCode(lum), r, gg, b})
				}
			}
		}
	}
	return res
}

// composite põe o modelo desenhado na grade, com um contorno vazio em volta
// da silhueta para destacar do chão.
func (g *Game) composite(r *r3res) {
	if r == nil || r.hits == nil {
		return
	}
	for _, c := range r.cells {
		g.setCell(r.x+c.dx, r.y+c.dy, c.code, c.r, c.g, c.b)
	}
	mm := g.m
	for j := 0; j < r.h; j++ {
		for i := 0; i < r.w; i++ {
			if r.hits[j*r.w+i] {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				ni, nj := i+d[0], j+d[1]
				if ni >= 0 && ni < r.w && nj >= 0 && nj < r.h && r.hits[nj*r.w+ni] {
					wx, wy := r.x+r.dx0+i, r.y+r.dy0+j
					if wx >= 0 && wy >= 0 && wx < mm.w && wy < mm.h && !g.hides(wx, wy, r.y) {
						g.setCell(wx, wy, ' ', 0, 0, 0)
					}
					break
				}
			}
		}
	}
}

// ---------- fila do quadro ----------
// Os personagens entram na fila na ordem de desenho (de trás para a frente).
// Os modelos 3D são desenhados em paralelo; depois a fila roda em ordem.

type r3job struct {
	m                 *model
	x, y, seed, clip  int
	yaw, scale, flash float64
	tint              col
	res               *r3res
}

func (g *Game) op(f func()) { g.ops = append(g.ops, f) }

func (g *Game) enqueue3d(m *model, x, y int, yaw, scale, flash float64, tint col, seed, clip int) {
	j := &r3job{m: m, x: x, y: y, seed: seed, clip: clip, yaw: yaw, scale: scale, flash: flash, tint: tint}
	g.jobs = append(g.jobs, j)
	g.op(func() { g.composite(j.res) })
}

func (g *Game) flush3d() {
	if len(g.jobs) > 0 {
		var wg sync.WaitGroup
		next := make(chan *r3job, len(g.jobs))
		for _, j := range g.jobs {
			next <- j
		}
		close(next)
		for w := 0; w < min(runtime.NumCPU(), len(g.jobs)); w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range next {
					j.res = g.renderModel(j.m, j.x, j.y, j.yaw, j.scale, j.flash, j.tint, j.seed, j.clip)
				}
			}()
		}
		wg.Wait()
	}
	for _, f := range g.ops {
		f()
	}
	g.jobs, g.ops = g.jobs[:0], g.ops[:0]
}

// yawFor acompanha para onde o personagem anda (ou golpeia), girando aos
// poucos, sem saltos.
func (g *Game) yawFor(key string, vx, vy float64, act *action, face int, dt float64) float64 {
	cur, ok := g.yaws[key]
	target := cur
	switch {
	case act != nil:
		target = math.Atan2(math.Cos(act.dir), math.Sin(act.dir))
	case math.Abs(vx)+math.Abs(vy) > 0.05:
		target = math.Atan2(vx, vy)
	case !ok:
		target = float64(face) * math.Pi / 2
	}
	if !ok {
		cur = target
	}
	cur += math.Remainder(target-cur, tau) * math.Min(1, dt*12)
	g.yaws[key] = cur
	return cur
}

func clipOf(b bodyDraw) int {
	if b.sink > 0 {
		return b.y
	}
	return 1 << 30
}

// drawBody3D põe na fila o corpo de gente em 3D (e o rastro do golpe).
func (g *Game) drawBody3D(b bodyDraw, yaw float64) {
	m := pose3d(b.style, b.phase, b.act, b.T, b.wcol, b.guard, b.seated)
	g.enqueue3d(m, b.x, b.y+b.sink, yaw, b.style.sc(), b.flash, b.tint, b.style.seed, clipOf(b))
	if b.act != nil && b.act.anim == "swing" {
		act := *b.act
		g.op(func() {
			cx, cy := float64(b.x), float64(b.y)-17*b.style.sc()
			for k := 0; k < 3; k++ {
				g.ring(cx, cy, 16*b.style.sc()-float64(k)*1.5, act.dir-1.5, act.dir-1.5+2.9*ease(act.t), mix3(col{255, 255, 255}, col{120, 130, 170}, float64(k)/3+act.t*0.4))
			}
		})
	}
}

// drawBeast3D põe na fila um bicho (ou montaria) em 3D.
func (g *Game) drawBeast3D(b bodyDraw, yaw float64) {
	m := pose3dBeast(b.style, b.phase, b.act, b.T)
	g.enqueue3d(m, b.x, b.y+b.sink, yaw, b.style.sc(), b.flash, b.tint, b.style.seed, clipOf(b))
}

// pose3dBeast monta um bicho de quatro, seis ou oito patas (ou que voa): corpo
// deitado ao longo de +z, cabeça na frente, patas alternando no passo.
func pose3dBeast(st *bodyStyle, phase *float64, act *action, T float64) *model {
	m := &model{}
	c1 := m.mat(or(st.c1, col{120, 100, 90}))
	c2 := m.mat(or(st.c2, col{150, 130, 120}))
	eye := m.mat(or(st.eye, col{255, 70, 50}))
	m.glow[eye] = true
	moving := phase != nil
	ph := 0.0
	if moving {
		ph = *phase
	}
	L, r, h := st.blen, st.br, st.bh
	cy := h + r
	switch {
	case st.fly:
		cy += math.Sin(T*6+float64(st.seed)) * 1.5
	case moving:
		cy += math.Abs(math.Sin(ph*2)) * 0.5
	default:
		cy += math.Sin(T*2.4+float64(st.seed)) * 0.3
	}
	lunge := 0.0
	if act != nil {
		lunge = math.Sin(math.Pi*act.t) * 3
	}
	m.capsule(v3{0, cy, -L + lunge}, v3{0, cy + 0.3, L + lunge}, r, c1)
	pairs := st.beast / 2
	if st.fly {
		pairs = 0
		flap := math.Sin(T*16+float64(st.seed)) * 0.7
		for side := -1.0; side <= 1; side += 2 {
			tip := v3{side * 8 * math.Cos(flap), cy + 8*math.Sin(flap), 0}
			sh := v3{side * 1, cy, 0}
			m.capsule(sh, tip, 0.35, c2)
			bone := tip.sub(sh)
			m.box(lerp3(sh, tip, 0.5).add(v3{0, -0.4, -1}), v3{bone.len() / 2, 0.25, 1.8}, v3{0, 0, 1}, v3{0, 0, 1}.cross(bone.norm()), c2)
		}
	}
	legR := st.leg
	if legR == 0 {
		legR = 0.8
	}
	for i := 0; i < pairs; i++ {
		z := 0.0
		if pairs > 1 {
			z = -L + 2*L*float64(i)/float64(pairs-1)
		}
		for side := -1.0; side <= 1; side += 2 {
			sw := 0.0
			if moving {
				sw = math.Sin(ph*1.5 + float64(i)*2.1 + (side+1)/2*math.Pi)
			}
			lift := math.Max(0, -sw) * 1.2
			x := side * r * 0.6
			top := v3{x, cy - r*0.3, z + lunge}
			if st.arch {
				knee := v3{x * 3, cy + 0.8, z*1.3 + lunge}
				m.capsule(top, knee, 0.5, c2)
				m.capsule(knee, v3{x*4 + sw*side, 0.4 + lift, z*1.5 + lunge}, 0.45, c2)
			} else {
				m.capsule(top, v3{x * 1.05, 0.5 + lift, z + sw*1.6 + lunge}, legR, c2)
			}
		}
	}
	hz, hy := L+r*0.5+lunge, cy+r*0.35+st.neck
	if st.neck > 0 {
		m.capsule(v3{0, cy + r*0.3, L*0.8 + lunge}, v3{0, hy, hz}, r*0.42, m.mat(or(st.mane, *st.c1)))
	}
	hr := st.hr
	if hr == 0 {
		hr = r * 0.75
	}
	m.capsule(v3{0, hy, hz}, v3{0, hy - 0.4, hz + r*0.5}, hr, c2)
	for side := -1.0; side <= 1; side += 2 {
		m.capsule(v3{side * hr * 0.5, hy + 0.3, hz + r*0.5 + hr*0.6}, v3{side * hr * 0.5, hy + 0.3, hz + r*0.5 + hr*0.61}, 0.35, eye)
		if st.ears {
			m.capsule(v3{side * 1.1, hy + hr*0.7, hz}, v3{side * 1.2, hy + hr*1.3, hz - 0.3}, 0.5, c1)
		}
		if st.tusk {
			m.capsule(v3{side * 0.8, hy - 0.6, hz + r*0.5 + hr*0.5}, v3{side * 1.0, hy + 0.6, hz + r*0.5 + hr*1.1}, 0.3, m.mat(col{240, 232, 210}))
		}
	}
	if st.tail {
		tl := st.tailLen
		if tl == 0 {
			tl = 1
		}
		m.capsule(v3{0, cy, -L + lunge}, v3{math.Sin(T*5) * 1.2, cy + 2, -L - 3.5*tl + lunge}, 0.6, c1)
	}
	m.rad = L + 12
	return m
}

// drawDragon3D põe Tiamath na fila: a mãe dos dragões, com cinco cabeças em
// pescoços longos que serpenteiam cada um no seu ritmo (uma por elemento:
// fogo, raio, veneno, gelo e sombra), corpo de escamas bronze, duas caudas e
// asas enormes com dedos e membrana.
func (g *Game) drawDragon3D(x, y int, yaw, T, flash float64) {
	m := &model{}
	body := m.mat(col{176, 132, 64})
	belly := m.mat(col{226, 196, 130})
	wing := m.mat(col{120, 82, 40})
	bone := m.mat(col{214, 176, 112})
	horn := m.mat(col{230, 220, 196})
	b := math.Sin(T*2.4) * 1.4 // sobe e desce com o bater das asas
	base := v3{0, 22 + b, 0}
	m.capsule(base.add(v3{0, 0, -10}), base.add(v3{0, 2, 8}), 7.5, body)
	m.capsule(base.add(v3{0, -4, -6}), base.add(v3{0, -3, 7}), 5, belly)

	// Cinco pescoços saindo dos ombros em leque; cada cabeça na cor do seu
	// elemento, com chifres, olhos acesos e a boca que abre de vez em quando.
	heads := []struct {
		scale col
		glow  col
	}{{col{200, 50, 40}, col{255, 200, 80}}, {col{60, 110, 210}, col{200, 230, 255}}, {col{176, 132, 64}, col{255, 230, 120}},
		{col{70, 150, 60}, col{200, 255, 120}}, {col{50, 40, 60}, col{220, 120, 255}}}
	order := []int{4, 1, 2, 3, 0} // da esquerda para a direita: sombra, raio, a do meio (bronze), veneno, fogo
	for k := 0; k < 5; k++ {
		hc := heads[order[k]]
		neckM := m.mat(mix3(col{176, 132, 64}, hc.scale, 0.55))
		headM := m.mat(hc.scale)
		eye := m.mat(hc.glow)
		m.glow[eye] = true
		spread := (float64(k) - 2) / 2 // -1..1
		ph := T*1.1 + float64(k)*1.3
		root := base.add(v3{spread * 6, 4, 7})
		prev, pr := root, 3.6
		var hd v3
		for s := 1; s <= 5; s++ {
			t := float64(s) / 5
			// O pescoço sobe e abre para o lado, ondulando.
			wave := math.Sin(ph+t*2.6) * 2.4 * t
			p := root.add(v3{spread*22*t + wave, 26*t - 6*t*t*math.Abs(spread), 4*t + math.Cos(ph*0.8+t*2)*2*t})
			r := 3*(1-t) + 1.4
			m.cone(prev, p, pr, r, neckM)
			prev, pr, hd = p, r, p
		}
		face := v3{spread * 0.5, -0.15, 1}.norm()
		open := math.Max(0, math.Sin(T*1.7+float64(k)*1.9)) * 1.6
		m.capsule(hd, hd.add(face.mul(5.5)), 2.3, headM)                                                     // crânio e focinho
		m.capsule(hd.add(v3{0, -1.6 - open, 1}), hd.add(v3{0, -2 - open, 1}.add(face.mul(4.5))), 1.2, headM) // mandíbula
		for side := -1.0; side <= 1; side += 2 {
			m.capsule(hd.add(v3{side * 1.3, 1, 1.6}), hd.add(v3{side * 1.3, 1, 1.7}), 0.6, eye)
			m.cone(hd.add(v3{side * 1.2, 1.6, -0.5}), hd.add(v3{side*2.2 + spread*1.5, 6, -4}), 0.7, 0.12, horn)
		}
	}

	// Duas caudas, afinando e serpenteando em oposição.
	for _, side := range []float64{-1, 1} {
		prev, pr := base.add(v3{side * 2.5, 0, -9}), 5.0
		for k := 1; k <= 6; k++ {
			t := float64(k) / 6
			pt := base.add(v3{side*(3+6*t) + math.Sin(T*2+float64(k)*0.8+side)*3*t, -9 * t, -9 - 32*t})
			r := 5*(1-t) + 0.5
			m.cone(prev, pt, pr, r, body)
			prev, pr = pt, r
		}
		m.cone(prev, prev.add(v3{side * 2, -1, -4}), 1.4, 0.2, horn) // ponta de espinho
	}
	// Patas penduradas (está voando).
	for _, z := range []float64{-6, 6} {
		for side := -1.0; side <= 1; side += 2 {
			hip := base.add(v3{side * 5.5, -3.5, z})
			knee := hip.add(v3{side * 1.5, -7, z * 0.1})
			m.capsule(hip, knee, 1.9, body)
			m.capsule(knee, knee.add(v3{0, -4.5, 2.2}), 1.4, body)
		}
	}
	// Asas: quatro dedos em leque e a membrana entre o ombro e cada dedo.
	flap := math.Sin(T*2.4)*0.6 + 0.15
	for side := -1.0; side <= 1; side += 2 {
		sh := base.add(v3{side * 6, 6, 2})
		elbow := sh.add(v3{side * 15 * math.Cos(flap), 15 * math.Sin(flap), -2})
		m.capsule(sh, elbow, 1.6, bone)
		for f := 0; f < 4; f++ {
			a := flap - 0.3 - float64(f)*0.4
			tip := elbow.add(v3{side * 28 * math.Cos(a), 28 * math.Sin(a), -4 - float64(f)*6})
			m.capsule(elbow, tip, 0.7, bone)
			mid := lerp3(sh, tip, 0.5).add(v3{0, -1.5, 0})
			dir := tip.sub(sh).norm()
			m.box(mid, v3{tip.sub(sh).len() / 2, 0.3, 5}, v3{0, 0, 1}, v3{0, 0, 1}.cross(dir), wing)
		}
	}
	m.rad = 60
	g.enqueue3d(m, x, y, yaw, 1.7, flash, white, 99, 1<<30) // um kaiju: bem maior que tudo
}

// drawShowcase é a vitrine do teste 3D: as sete classes em 3D numa fileira
// (andando, golpeando e paradas, girando devagar) e, embaixo, as mesmas em 2D.
func (g *Game) drawShowcase(T, dt float64) {
	gap := float64(g.cols) / 8
	row3, rowM, row2 := g.y0+g.rows*38/100, g.y0+g.rows*67/100, g.y0+g.rows*96/100
	// Palco escuro: o cenário recua para os bonecos se destacarem.
	for i := 0; i+3 < len(g.buf); i += 4 {
		g.buf[i], g.buf[i+1], g.buf[i+2] = g.buf[i]/5, g.buf[i+1]/5, g.buf[i+2]/5
	}
	g.labels = g.labels[:0]
	for i := 0; i < 7; i++ {
		st := g.playerStyle(i, i, 0)
		x := g.x0 + int(gap*float64(i+1))
		ph := T*7 + float64(i)
		var phase *float64
		var act *action
		switch i % 3 {
		case 0:
			phase = &ph
		case 1:
			anim := anims[i][0]
			act = &action{anim, math.Mod(T*1.2+float64(i)*0.3, 1), 0.4}
		}
		yaw := T*0.7 + float64(i)*0.9
		b := bodyDraw{style: st, x: x, y: row3, T: T, phase: phase, hf: 1, face: 1, tint: white, act: act}
		g.drawBody3D(b, yaw)
		bm := b
		bm.y, bm.lit = rowM, true
		g.op(func() { g.drawBody(bm) })
		b2 := b
		b2.y = row2
		g.op(func() { g.drawBody(b2) })
		g.labels = append(g.labels, label{x: float64(x), y: float64(row3) - 31, text: classInfo[i][0] + " · 3D", color: col{170, 170, 180}})
		g.labels = append(g.labels, label{x: float64(x), y: float64(rowM) - 31, text: classInfo[i][0] + " · Mescla", color: col{255, 224, 102}})
		g.labels = append(g.labels, label{x: float64(x), y: float64(row2) - 31, text: classInfo[i][0] + " · 2D", color: col{170, 170, 180}})
	}
}

// drawSteedAny põe a montaria na fila, em 3D ou no desenho antigo.
func (g *Game) drawSteedAny(kind int, b bodyDraw, key string, p pos, dt float64) {
	st := &steedStyles[clampi(kind, 0, len(steedStyles)-1)]
	if g.mode >= modeMix {
		sb := b
		sb.style, sb.act, sb.sink, sb.flash = st, nil, 0, 0
		g.drawBeast3D(sb, g.yawFor(key, p.vx, p.vy, nil, b.face, dt))
		return
	}
	g.op(func() { g.drawSteed(kind, b.x, b.y, b.face, b.phase, b.T) })
}
