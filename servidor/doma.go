package main

import (
	"strconv"
	"strings"
	"time"
)

// Doma: além do nível, o bicho pede fôlego (stamina) e uma sequência de setas
// tocada a tempo. E diante do bicho começa a tentativa: o servidor sorteia a
// sequência (mais longa nos bichos mais bravos) e o cliente a mostra; o
// jogador repete com as setas (ou WASD). Acertou dentro do tempo: domado, e a
// stamina pedida é gasta. Errou ou demorou: gasta metade e o bicho se assusta
// e corre para longe.

type tameTry struct {
	steed int       // índice em wd.steeds
	kind  int       // tipo do bicho
	seq   string    // U, D, L, R
	until time.Time // fim do tempo
}

// Fôlego pedido e tamanho da sequência de cada montaria (na ordem de steedKinds).
var tameNeed = []struct{ Stamina, Len int }{
	steedHorse:   {30, 3},
	steedBoar:    {45, 4},
	steedOstrich: {55, 5},
	steedRaptor:  {75, 6},
}

const tameTime = 6 * time.Second

// startTame começa a doma do bicho s. Chamar com o lock.
func (wd *World) startTame(p *Player, si int, now time.Time) {
	s := wd.steeds[si]
	k, need := steedKinds[s.Kind], tameNeed[s.Kind]
	if p.Level < k.Level {
		wd.say(ChatLine{Text: p.Name + " precisa do nível " + strconv.Itoa(k.Level) + " para domar o " + k.Name + "."})
		return
	}
	if int(p.ST) < need.Stamina {
		wd.pop(p, "Sem fôlego para domar o "+k.Name+" (pede "+strconv.Itoa(need.Stamina)+" de stamina)", 0xc0d080, float64(p.X), float64(p.Y)-34, now)
		return
	}
	var b strings.Builder
	for i := 0; i < need.Len; i++ {
		b.WriteByte("UDLR"[wd.rng.Intn(4)])
	}
	p.tame = &tameTry{steed: si, kind: s.Kind, seq: b.String(), until: now.Add(tameTime + time.Duration(need.Len)*400*time.Millisecond)}
}

// Tame recebe a sequência que o jogador tocou.
func (wd *World) Tame(id int, seq string) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.tame == nil {
		return
	}
	now, t := wd.clock(), p.tame
	p.tame = nil
	if t.steed < 0 || t.steed >= len(wd.steeds) {
		return
	}
	s := wd.steeds[t.steed]
	k, need := steedKinds[s.Kind], tameNeed[s.Kind]
	if !s.Alive || dist(float64(p.X), float64(p.Y), s.X, s.Y) > steedReach*2 {
		return
	}
	if seq != t.seq || now.After(t.until) {
		p.ST = max(0, p.ST-float64(need.Stamina/2))
		// O bicho se assusta e dispara para longe de quem tentou.
		dx, dy := s.X-float64(p.X), s.Y-float64(p.Y)
		d := max(1, dist(0, 0, dx, dy))
		for step := 24.0; step > 0; step -= 4 {
			if nx, ny := s.X+dx/d*step, s.Y+dy/d*step*0.6; wd.fits(int(nx), int(ny)) {
				s.X, s.Y = nx, ny
				break
			}
		}
		wd.pop(p, "O "+k.Name+" se assustou e fugiu", 0xd09060, float64(p.X), float64(p.Y)-34, now)
		return
	}
	p.ST = max(0, p.ST-float64(need.Stamina))
	s.Alive, s.respawnAt = false, now.Add(steedRespawn)
	if p.Steeds&(1<<s.Kind) == 0 {
		wd.say(ChatLine{Text: p.Name + " domou um " + k.Name + "!"})
	}
	p.Steeds |= 1 << s.Kind
	p.Riding, p.LastSteed = s.Kind+1, s.Kind
}

// tameName é o nome do bicho que está sendo domado (para o cliente).
func tameName(t *tameTry) string {
	if t.kind >= 0 && t.kind < len(steedKinds) {
		return steedKinds[t.kind].Name
	}
	return "bicho"
}

// tameExpire larga a tentativa esquecida (cliente que não respondeu).
func (wd *World) tameExpire(p *Player, now time.Time) {
	if p.tame != nil && now.After(p.tame.until.Add(2*time.Second)) {
		p.tame = nil
	}
}
