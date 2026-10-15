package main

import (
	"testing"
	"time"
)

// TestVeu: sem os cinco guardiões o Pico de Tiamath não deixa entrar; com
// eles caídos, deixa; e cada guardião purifica a sua região.
func TestVeu(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("subida", 0)
	p := wd.players[id]
	var pico region
	for _, r := range regions {
		if r.Name == veilRegion {
			pico = r
		}
	}
	// Um passo de fora para dentro da borda.
	x, y := pico.X, pico.Y+pico.R+1
	now := time.Now()
	if wd.canEnter(p, x, y-2, now) {
		t.Fatal("o Véu devia barrar o Pico sem os guardiões")
	}
	p.X, p.Y = int(wd.boss.X), int(wd.boss.Y)+30
	if tg, _ := wd.veilTarget(wd.boss.X, wd.boss.Y, now); tg != nil {
		t.Fatal("sem o Véu caído Tiamath não devia enxergar ninguém")
	}
	for _, v := range veilBosses {
		p.Slain |= 1 << uint(v.kind)
	}
	p.SideDone = ^uint(0) // o selo antigo da secundária também
	if tg, _ := wd.veilTarget(wd.boss.X, wd.boss.Y, now); tg != p {
		t.Fatal("com o Véu caído Tiamath devia enxergar o jogador")
	}
	if p.purified() != 1<<len(veilBosses)-1 {
		t.Fatalf("purificadas: %b", p.purified())
	}
	if !wd.canEnter(p, x, y-2, now) {
		t.Fatal("com os cinco guardiões caídos o Véu devia abrir")
	}
}

// veilDown dá ao jogador os cinco guardiões do Véu (para os testes de Tiamath).
func veilDown(p *Player) {
	for _, v := range veilBosses {
		p.Slain |= 1 << uint(v.kind)
	}
}
