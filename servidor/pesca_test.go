package main

import (
	"testing"
	"time"
)

// TestPesca: as quatro lagoas existem como água, cada água é reconhecida, a
// pesca pega um peixe na mordida e falha cedo, e o peixe se vende.
func TestPesca(t *testing.T) {
	wd := NewWorld(1)
	if len(wd.ponds) != len(pondsBase) {
		t.Fatalf("lagoas: %d de %d", len(wd.ponds), len(pondsBase))
	}
	for _, pd := range wd.ponds {
		if w := wd.waterAt(int(pd.x), int(pd.y)); w != pd.water {
			t.Errorf("%s: centro reconhecido como água %d", pd.name, w)
		}
	}
	if w := wd.waterAt(int(lakeX), int(lakeY)+8); w != waterLake {
		t.Errorf("Lago Espelhado reconhecido como %d", w)
	}
	id := wd.Join("pescador", 0)
	p := wd.players[id]
	pd := wd.ponds[0]
	// Margem da Lagoa da Vila: anda do centro para fora até caber o personagem.
	x, y := int(pd.x), int(pd.y)
	for !wd.fits(x, y) {
		y++
	}
	p.X, p.Y = x, y
	wd.FishCast(id)
	if p.fish != nil {
		t.Fatal("sem vara não devia pescar")
	}
	p.give(itemFishRod, 1)
	wd.FishCast(id)
	if p.fish == nil || p.fish.water != waterPondVila {
		t.Fatalf("lançar: %+v", p.fish)
	}
	wd.Reel(id) // cedo demais
	if p.fish != nil || p.FishKinds != 0 {
		t.Fatal("puxar antes da mordida devia espantar")
	}
	got := false
	for try := 0; try < 20 && !got; try++ {
		wd.FishCast(id)
		p.fish.biteAt = time.Now().Add(-100 * time.Millisecond)
		p.fish.until = time.Now().Add(time.Second)
		wd.Reel(id)
		got = p.FishKinds != 0
	}
	if !got || p.FishBodies != 1<<waterPondVila {
		t.Fatalf("pescar na mordida: peixes %b águas %b", p.FishKinds, p.FishBodies)
	}
	k := -1
	for i := range fishes {
		if p.count(itemFish0+i) > 0 {
			k = i
		}
	}
	if k < 0 || sellPrice(&wd.shops[0], itemFish0+k) != fishes[k].Price {
		t.Fatalf("preço do peixe %d", k)
	}
}

// TestBestiario: matar registra; fraqueza tira mais.
func TestBestiario(t *testing.T) {
	if weakPct(mobSkeleton, elemHoly) <= 100 || weakPct(mobSkeleton, elemPoison) >= 100 || weakPct(mobRat, elemIce) != 100 {
		t.Fatal("fraquezas do bestiário")
	}
	for k := range mobKinds {
		if _, ok := bestiary[k]; !ok {
			t.Errorf("%s sem verbete no bestiário", mobKinds[k].Name)
		}
	}
	wd := NewWorld(1)
	id := wd.Join("caçador", 0)
	p := wd.players[id]
	m := wd.mobs[0]
	wd.mobDied(m, p, time.Now())
	if len(p.BestKills) <= m.Kind || p.BestKills[m.Kind] != 1 || p.BestSeen&(1<<m.Kind) == 0 {
		t.Fatalf("morte no bestiário: %v %b", p.BestKills, p.BestSeen)
	}
}

// TestModoECaminho: o agressivo golpeia sozinho quem está perto; o caminho
// até um ponto leva o jogador até lá.
func TestModoECaminho(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("andarilho", 0)
	p := wd.players[id]
	for _, m := range wd.mobs {
		m.Alive, m.respawnAt = false, time.Now().Add(time.Hour)
	}
	m := wd.mobs[0]
	m.Kind, m.Alive, m.HP, m.MaxHP = mobSkeleton, true, 500, 500
	m.X, m.Y, m.bornAt = float64(p.X+6), float64(p.Y), time.Now().Add(-time.Hour)
	wd.autoFight(p, time.Now())
	if m.HP != 500 {
		t.Fatal("pacifista não devia golpear sozinho")
	}
	p.Mode = modeAggressive
	wd.autoFight(p, time.Now())
	if m.HP == 500 && len(wd.projs) == 0 && len(wd.effects) == 0 {
		t.Fatal("agressivo devia golpear quem está perto")
	}
	m.Alive = false
	// Caminho: da vila até a frente da casa à venda.
	gx, gy := lotDoor(1)
	wd.SetGoal(id, gx, gy)
	if p.goal == nil {
		t.Fatal("devia achar caminho até a casa")
	}
	now := time.Now()
	for i := 0; i < 20*60 && p.goal != nil; i++ {
		now = now.Add(time.Second / tickRate)
		wd.step(now)
	}
	if dist(float64(p.X), float64(p.Y), float64(gx), float64(gy)) > 6 {
		t.Fatalf("não chegou: está em (%d,%d), destino (%d,%d)", p.X, p.Y, gx, gy)
	}
}
