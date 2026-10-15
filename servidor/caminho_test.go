package main

import (
	"testing"
	"time"
)

// TestCaminho: o inimigo dentro de uma parede em U, com o jogador do outro
// lado, tem de contornar e chegar perto (antes ele empacava encostado no
// fundo do U, como o chefe da cripta).
func TestCaminho(t *testing.T) {
	for _, kind := range []int{mobSkeleton, mobApostle} {
		caminhoU(t, kind)
	}
}

func caminhoU(t *testing.T, kind int) {
	wd := NewWorld(1)
	id := wd.Join("cami", 0)
	p := wd.players[id]
	// Um campo limpo de 90x60 bem longe de tudo, com o U aberto para o oeste.
	cx, cy := villageX+40, villageY+300
	for y := cy - 30; y <= cy+30; y++ {
		for x := cx - 45; x <= cx+45; x++ {
			wd.solid[y*wd.w+x] = false
		}
	}
	wall := func(x, y int) { wd.solid[y*wd.w+x] = true }
	for y := cy - 15; y <= cy+15; y++ {
		wall(cx+10, y) // fundo do U, entre o inimigo e o jogador
	}
	for x := cx - 20; x <= cx+10; x++ {
		wall(x, cy-15)
		wall(x, cy+15)
	}
	for _, m := range wd.mobs {
		m.Alive = false // só o nosso no mundo
		m.respawnAt = time.Now().Add(time.Hour)
	}
	m := wd.mobs[0]
	m.Kind, m.Alive, m.HP, m.MaxHP = kind, true, 999, 999
	m.X, m.Y, m.homeX, m.homeY = float64(cx), float64(cy), float64(cx), float64(cy)
	m.bornAt = time.Now().Add(-time.Hour)
	p.X, p.Y, p.HP, p.MaxHP = cx+25, cy, 99999, 99999
	now := time.Now()
	for i := 0; i < 20*40; i++ { // 40 s de jogo
		now = now.Add(time.Second / tickRate)
		wd.stepMobs(now)
		wd.pathBudget = pathPerTick
		if dist(m.X, m.Y, float64(p.X), float64(p.Y)) <= mobReach {
			return
		}
	}
	t.Fatalf("%s não contornou a parede: está em (%.0f,%.0f), o jogador em (%d,%d)", mobKinds[kind].Name, m.X, m.Y, p.X, p.Y)
}
