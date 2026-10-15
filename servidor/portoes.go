package main

import (
	"strings"
	"time"

	"asciimmo/proto"
)

// Portões Rúnicos: arcos de pedra antigos espalhados por Midrath e pelas
// Profundezas. Cada um tem uma runa (sequência de setas); quem a toca diante
// do portão (E) o acende, e a runa do próximo portão da corrente fica
// gravada no que acabou de acender. Portões acesos se ligam entre si: E num
// deles abre a lista dos outros, e atravessar gasta uma Pedra de Eco. O
// primeiro, o de Cinzalva, já nasce sem runa e dá três pedras a quem o acende.
// Player.Gates guarda os acesos (bits; vai no save e no estado).

const (
	gateReach = 12.0 // distância para usar o portão
	maxGates  = 32
)

type gate struct {
	Name string
	X, Y int
	Seq  string // runa (U D L R); vazia = acende sem runa
}

// gatesBase: a ordem é a da corrente (acender o i revela a runa do i+1) e
// também o índice nos bits do jogador: só acrescentar no fim.
var gatesBase = []gate{
	{"Portão de Cinzalva", villageX - 110, villageY - 70, ""},
	{"Portão de Marapá", jungleX - 270, jungleY + 90, "URDL"}, // junto à Fogueira da Selva, longe da cidadela
	{"Portão das Areias", desertX + 130, desertY - 50, "LLUR"},
	{"Portão das Marés", tideX - 120, tideY + 70, "DURD"},
	{"Portão dos Picos", snowX - 110, snowY + 10, "RULDR"},
	{"Portão da Árvore Divina", treeX + 84, treeY + 24, "UDLRU"}, // Grael ronda perto: fica para o fim da superfície
	{"Portão de Dorrim", dorrimX + dorrimHW + 26, dorrimY, "LDLDU"},
	{"Portão dos Cristais", crystalX + 34, crystalY + 14, "RRDLU"},
	{"Portão do Coração", heartX - 92, heartY, "DDURLU"},
}

// carveGates abre, na grade das Profundezas, um patamar em cada portão de
// baixo e um túnel até a fenda mais perto.
func carveGates(carve func(cx, cy, r float64), tunnel func(ax, ay, bx, by float64, k uint32)) {
	for i, g := range gatesBase {
		if !inDeep(g.Y) {
			continue
		}
		x, y := float64(g.X), float64(g.Y-deepY0)
		carve(x, y, 9)
		r := rifts[nearestRift(x, y, -1)]
		tunnel(x, y, float64(r[0]), float64(r[1]), uint32(800+i))
	}
}

// buildGates ergue os arcos: dois pilares, a verga e a runa no meio.
func (wd *World) buildGates(seed uint32) {
	wd.gates = append([]gate(nil), gatesBase...)
	for i := range wd.gates {
		g := &wd.gates[i]
		if !inDeep(g.Y) {
			wd.clearArea(g.X-12, g.Y-14, g.X+12, g.Y+8, seed+1700+uint32(i))
		}
		stone := dim(0x9a94a8, 0.85)
		for dy := 0; dy <= 7; dy++ {
			for _, dx := range []int{-5, 5} {
				wd.put(g.X+dx, g.Y-dy, pick("H#", hash2(g.X+dx, g.Y-dy, seed+1710)), stone, dy < 2, dy >= 2)
			}
		}
		for dx := -6; dx <= 6; dx++ {
			wd.put(g.X+dx, g.Y-8, '=', stone, false, true)
		}
		wd.put(g.X, g.Y-8, '*', 0xb8a0ff, false, true)
		wd.glow[(g.Y-8)*wd.w+g.X] = true
		for dx := -4; dx <= 4; dx++ { // soleira
			wd.put(g.X+dx, g.Y+1, '_', dim(0x8a8498, 0.7), false, false)
		}
	}
}

func (p *Player) gateLit(i int) bool { return i >= 0 && i < maxGates && p.Gates&(1<<i) != 0 }

// atGate é o portão onde o jogador está, ou -1.
func (wd *World) atGate(p *Player) int {
	for i, g := range wd.gates {
		if dist(float64(p.X), float64(p.Y), float64(g.X), float64(g.Y)) <= gateReach {
			return i
		}
	}
	return -1
}

// GateRune: o jogador tocou seq diante do portão i para acendê-lo.
func (wd *World) GateRune(id, i int, seq string) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || i < 0 || i >= len(wd.gates) || wd.atGate(p) != i || p.gateLit(i) || len(seq) > 12 {
		return
	}
	now := wd.clock()
	g := wd.gates[i]
	if strings.ToUpper(seq) != g.Seq {
		wd.pop(p, "A runa não responde", 0xff7060, float64(p.X), float64(p.Y)-40, now)
		return
	}
	wd.lightGate(p, i, now)
}

func (wd *World) lightGate(p *Player, i int, now time.Time) {
	p.Gates |= 1 << i
	g := wd.gates[i]
	wd.effects = append(wd.effects, Effect{Kind: fxNova, X: float64(g.X), Y: float64(g.Y - 4), R: 24, Start: now, Dur: 900 * time.Millisecond})
	wd.pop(p, g.Name+" acendeu", 0xb8a0ff, float64(p.X), float64(p.Y)-40, now)
	if i == 0 {
		p.give(itemEchoStone, 3)
	}
	if n := i + 1; n < len(wd.gates) {
		wd.say(ChatLine{Text: "Na pedra do " + g.Name + " está gravada a runa do " + wd.gates[n].Name + ": " + runeArrows(wd.gates[n].Seq)})
	}
}

// GateGo: atravessa do portão aceso onde o jogador está até o portão aceso j.
func (wd *World) GateGo(id, j int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.boat || j < 0 || j >= len(wd.gates) || !p.gateLit(j) {
		return
	}
	i := wd.atGate(p)
	if i < 0 || i == j || !p.gateLit(i) {
		return
	}
	now := wd.clock()
	if p.count(itemEchoStone) == 0 {
		wd.pop(p, "Falta uma Pedra de Eco", 0xff7060, float64(p.X), float64(p.Y)-40, now)
		return
	}
	g := wd.gates[j]
	x, y, ok := wd.freeNear(g.X, g.Y+4, 14)
	if !ok {
		return
	}
	p.take(itemEchoStone)
	wd.effects = append(wd.effects, Effect{Kind: fxBlink, X: float64(p.X), Y: float64(p.Y), R: 12, Start: now, Dur: 600 * time.Millisecond})
	p.X, p.Y, p.Riding = x, y, 0
	wd.effects = append(wd.effects, Effect{Kind: fxBlink, X: float64(x), Y: float64(y), R: 12, Dir: 1, Start: now, Dur: 600 * time.Millisecond})
	wd.pop(p, "Atravessou até o "+g.Name, 0xb8a0ff, float64(x), float64(y)-34, now)
}

// gateFree: o portão sem runa acende só de chegar perto (o de Cinzalva).
func (wd *World) gateFree(p *Player, now time.Time) {
	if i := wd.atGate(p); i >= 0 && wd.gates[i].Seq == "" && !p.gateLit(i) {
		wd.lightGate(p, i, now)
	}
}

func runeArrows(seq string) string {
	r := strings.NewReplacer("U", "↑ ", "D", "↓ ", "L", "← ", "R", "→ ")
	return strings.TrimSpace(r.Replace(seq))
}

func (wd *World) gateInfo() []proto.Gate {
	var out []proto.Gate
	for _, g := range wd.gates {
		out = append(out, proto.Gate{Name: g.Name, X: g.X, Y: g.Y, Seq: g.Seq})
	}
	return out
}
