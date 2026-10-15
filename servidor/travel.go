package main

import (
	"math"
	"time"

	"asciimmo/proto"
)

// Viagem rápida: as fogueiras e as placas de direção são pontos de viagem.
// Cada um é descoberto ao passar perto; de um ponto descoberto se viaja para
// qualquer outro já descoberto (tecla V no cliente).

type travelPt struct {
	Name string
	X, Y int
	Kind int // 0 fogueira, 1 placa
}

const (
	discoverR = 24.0
	travelR   = 24.0
)

func (wd *World) buildTravel() {
	wd.travel = wd.travel[:0]
	for _, b := range wd.bonfires {
		if b.Name == "Fogueira do Despertar" {
			continue // fora da viagem: os índices das placas ficam como estavam (saves)
		}
		wd.travel = append(wd.travel, travelPt{b.Name, b.X, b.Y + 6, 0})
	}
	wd.travel = append(wd.travel, wd.signs...)
}

// discover marca os pontos perto do jogador.
func (wd *World) discover(p *Player, now time.Time) {
	for i, t := range wd.travel {
		if i < 64 && p.Found&(1<<i) == 0 && dist(float64(p.X), float64(p.Y), float64(t.X), float64(t.Y)) <= discoverR {
			p.Found |= 1 << i
			wd.pop(p, "Ponto de viagem: "+t.Name, 0x9ad0ff, float64(p.X), float64(p.Y)-34, now)
		}
	}
}

// atTravel devolve o ponto de viagem descoberto em que o jogador está, ou -1.
func (wd *World) atTravel(p *Player) int {
	for i, t := range wd.travel {
		if i < 64 && p.Found&(1<<i) != 0 && dist(float64(p.X), float64(p.Y), float64(t.X), float64(t.Y)) <= travelR {
			return i
		}
	}
	return -1
}

// Travel leva o jogador ao ponto i, se ele estiver num ponto descoberto e o
// destino também já tiver sido descoberto.
func (wd *World) Travel(id, i int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.boat || i < 0 || i >= len(wd.travel) || i >= 64 || p.Found&(1<<i) == 0 || wd.atTravel(p) < 0 {
		return
	}
	now := wd.clock()
	t := wd.travel[i]
	if r := regionAt(t.X, t.Y); r >= 0 && wd.veilBlocks(p, regions[r], now) {
		return // o Véu de Cinzas também barra a viagem para dentro do Pico
	}
	x, y, ok := wd.freeNear(t.X, t.Y+4, 20)
	if !ok {
		return
	}
	wd.effects = append(wd.effects, Effect{Kind: fxBlink, X: float64(p.X), Y: float64(p.Y), R: 10, Start: now, Dur: 500 * time.Millisecond})
	p.X, p.Y = x, y
	p.Riding = 0
	wd.effects = append(wd.effects, Effect{Kind: fxBlink, X: float64(x), Y: float64(y), R: 10, Dir: 1, Start: now, Dur: 500 * time.Millisecond})
	wd.pop(p, "Viajou até "+t.Name, 0x9ad0ff, float64(x), float64(y)-34, now)
}

func (wd *World) travelInfo() []proto.TravelPt {
	var out []proto.TravelPt
	for _, t := range wd.travel {
		out = append(out, proto.TravelPt{Name: t.Name, X: t.X, Y: t.Y, Kind: t.Kind})
	}
	return out
}

// signName dá nome ao ponto de viagem de uma placa: a trilha e para onde ela vai.
func signName(dest int, x, y int) string {
	from := "Ermos"
	best := math.MaxFloat64
	for i, r := range regions {
		if i == dest {
			continue
		}
		if d := dist(float64(x), float64(y), float64(r.X), float64(r.Y)); d < best {
			from, best = r.Name, d
		}
	}
	return "Placa: " + from + " → " + regions[dest].Name
}
