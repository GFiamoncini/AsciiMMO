package main

import (
	"fmt"
	"time"
)

// Renascimento dos inimigos e purificação das áreas.
//
// Inimigo comum volta em mobRespawn; chefe, em bossRespawn. Cada inimigo
// pertence à área da fogueira mais próxima (no mesmo andar: superfície ou
// subsolo). Quem descansa numa fogueira com a área toda limpa a purifica:
// ali ninguém renasce por purifyFor. Não é para sempre porque o mundo é
// compartilhado — outro jogador pode precisar daqueles inimigos na missão.
const (
	mobRespawn  = 3 * time.Minute
	bossRespawn = 10 * time.Minute
	purifyFor   = 30 * time.Minute
	zoneReach   = 420.0 // inimigo além disso de toda fogueira fica sem área
)

func init() {
	for i := range mobKinds {
		if mobKinds[i].Boss {
			mobKinds[i].Respawn = bossRespawn
		} else {
			mobKinds[i].Respawn = mobRespawn
		}
	}
}

// zone é a área de uma fogueira: os inimigos dela e até quando está purificada.
type zone struct {
	mobs   []*Mob
	purged time.Time
}

// assignZones liga cada inimigo à fogueira mais próxima do mesmo andar.
func (wd *World) assignZones() {
	wd.zones = make([]zone, len(wd.bonfires))
	for _, m := range wd.mobs {
		m.zone = -1
		best := zoneReach
		for i, b := range wd.bonfires {
			if (b.Y >= overH) != (m.homeY >= overH) {
				continue
			}
			if d := dist(m.homeX, m.homeY, float64(b.X), float64(b.Y)); d < best {
				best, m.zone = d, i
			}
		}
		if m.zone >= 0 {
			wd.zones[m.zone].mobs = append(wd.zones[m.zone].mobs, m)
		}
	}
}

// purged diz se a área do inimigo está purificada agora.
func (wd *World) purged(m *Mob, now time.Time) bool {
	if m.Kind == mobDummy {
		return false // os bonecos do prólogo voltam sempre, para o próximo novato
	}
	return m.zone >= 0 && m.zone < len(wd.zones) && now.Before(wd.zones[m.zone].purged)
}

// rest é o descanso na fogueira: recupera tudo, cura os status, acende a
// fogueira e, se a área estiver limpa, purifica.
func (wd *World) rest(p *Player, i int, now time.Time) {
	wd.questEvent(p, qRest, now)
	wd.giveTorch(p, now)
	b := wd.bonfires[i]
	p.Bonfire = i
	p.HP, p.MP, p.ST = p.MaxHP, p.MaxMP, float64(p.MaxST)
	p.status = [numStatus]time.Time{}
	wd.effects = append(wd.effects, Effect{Kind: fxHeal, X: float64(p.X), Y: float64(p.Y), R: 10, Start: now, Dur: 600 * time.Millisecond})
	wd.persist(p) // descansar grava, como num ponto de controle
	z := &wd.zones[i]
	alive := 0
	for _, m := range z.mobs {
		if m.Alive {
			alive++
		}
	}
	switch {
	case len(z.mobs) == 0:
		wd.pop(p, "Você descansou: "+b.Name, 0xffa040, float64(p.X), float64(p.Y)-40, now)
	case alive > 0:
		wd.pop(p, fmt.Sprintf("Você descansou. Ainda há %d inimigo(s) na área de %s.", alive, b.Name), 0xffa040, float64(p.X), float64(p.Y)-40, now)
	default:
		z.purged = now.Add(purifyFor)
		wd.say(ChatLine{Text: p.Name + " purificou a área de " + b.Name + "."})
		wd.pop(p, fmt.Sprintf("Área purificada por %d min", int(purifyFor.Minutes())), 0xfff0b0, float64(p.X), float64(p.Y)-40, now)
		wd.effects = append(wd.effects, Effect{Kind: fxHoly, X: float64(b.X), Y: float64(b.Y), R: 30, Start: now, Dur: 900 * time.Millisecond})
	}
}

// purgedFires lista as fogueiras com a área purificada agora (para os clientes).
func (wd *World) purgedFires(now time.Time) []int {
	var out []int
	for i, z := range wd.zones {
		if now.Before(z.purged) {
			out = append(out, i)
		}
	}
	return out
}

// PurgedFires é purgedFires com o lock.
func (wd *World) PurgedFires() []int {
	wd.mu.RLock()
	defer wd.mu.RUnlock()
	return wd.purgedFires(wd.clock())
}
