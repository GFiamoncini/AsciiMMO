package main

import (
	"fmt"
	"strings"
	"time"
)

// O Véu de Cinzas: o Pico de Tiamath fica fechado até os cinco guardiões da
// história caírem (Lorde da Cripta, Rainha Asura, Tirano das Marés, Faraó e
// Gigante de Gelo). Cada um que cai purifica a sua região: lá para de cair
// cinza e o ar volta ao normal (o cliente desenha; o estado leva os bits em
// "pure"). Antes, dava para subir ao Pico sem fazer nada da história.

var veilBosses = []struct {
	kind   int
	region string
}{
	{mobLord, "Cemitério"},
	{mobQueen, "Selva de Marapá"},
	{mobTyrant, "Talássia"},
	{mobPharaoh, "Deserto de Ankar"},
	{mobFrostGiant, "Picos de Gelo"},
}

const veilRegion = "Pico de Tiamath"

// purified são os bits (na ordem de veilBosses) das regiões já purificadas.
func (p *Player) purified() int {
	b := 0
	for i, v := range veilBosses {
		if p.Slain&(1<<uint(v.kind)) != 0 {
			b |= 1 << i
		}
	}
	return b
}

// veilMissing são os guardiões que ainda faltam.
func (p *Player) veilMissing() []string {
	var out []string
	for _, v := range veilBosses {
		if p.Slain&(1<<uint(v.kind)) == 0 {
			out = append(out, mobKinds[v.kind].Name)
		}
	}
	return out
}

// veilBlocks diz se o Véu barra o jogador de entrar em (x, y) e avisa.
func (wd *World) veilBlocks(p *Player, r region, now time.Time) bool {
	if r.Name != veilRegion {
		return false
	}
	miss := p.veilMissing()
	if len(miss) == 0 {
		return false
	}
	if now.Sub(p.sealAt) > 3*time.Second {
		p.sealAt = now
		wd.pop(p, fmt.Sprintf("O Véu de Cinzas fecha o Pico. Faltam: %s", strings.Join(miss, ", ")), 0xd0a090, float64(p.X), float64(p.Y)-34, now)
	}
	return true
}

// purifyNews avisa quem derrubou um guardião que a região foi purificada.
func (wd *World) purifyNews(p *Player, kind int) {
	for _, v := range veilBosses {
		if v.kind == kind {
			left := len(p.veilMissing())
			msg := fmt.Sprintf("%s purificou %s.", p.Name, v.region)
			if left == 0 {
				msg += " O Véu de Cinzas sobre o Pico de Tiamath se desfez."
			} else {
				msg += fmt.Sprintf(" Faltam %d guardiões para o Véu cair.", left)
			}
			wd.say(ChatLine{Text: msg})
		}
	}
}

// veilTarget é o alvo de Tiamath: o mais perto entre quem já pode estar no Pico.
func (wd *World) veilTarget(x, y float64, now time.Time) (*Player, float64) {
	var best *Player
	bd := 1e18
	for _, p := range wd.players {
		if p.paused || len(p.veilMissing()) > 0 {
			continue
		}
		if d := dist(float64(p.X), float64(p.Y), x, y); d < bd {
			best, bd = p, d
		}
	}
	return best, bd
}

// keepInPeak segura Tiamath dentro do Pico.
func (wd *World) keepInPeak() {
	b := &wd.boss
	for _, r := range regions {
		if r.Name != veilRegion {
			continue
		}
		dx, dy := b.X-float64(r.X), b.Y-float64(r.Y)
		if d := dist(0, 0, dx, dy); d > float64(r.R)-20 {
			k := (float64(r.R) - 20) / d
			b.X, b.Y = float64(r.X)+dx*k, float64(r.Y)+dy*k
		}
	}
}
