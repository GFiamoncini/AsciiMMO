package main

import (
	"testing"
	"time"
)

// TestPausa: sozinho, pausar para o mundo (o relógio não anda); pausado, o
// inimigo não o vê, nada o fere e ele não anda nem golpeia.
func TestPausa(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("pausa", 0)
	p := wd.players[id]
	if wd.frozen() {
		t.Fatal("o mundo começou parado")
	}
	wd.SetPaused(id, true)
	if !wd.frozen() {
		t.Fatal("sozinho e pausado: o mundo devia parar")
	}
	// Inimigo encostado não o vê e não fere.
	m := wd.mobs[0]
	m.X, m.Y, m.Alive, m.bornAt = float64(p.X+1), float64(p.Y), true, time.Now().Add(-time.Hour)
	if tg, _ := wd.nearestFor(m.X, m.Y, time.Now()); tg == p {
		t.Fatal("pausado: o inimigo não devia vê-lo")
	}
	hp := p.HP
	wd.hurt(p, 50, "teste", time.Now())
	wd.hurtRaw(p, 50, "teste", time.Now())
	if p.HP != hp {
		t.Fatalf("pausado: levou dano (%d → %d)", hp, p.HP)
	}
	x := p.X
	wd.SetInput(id, 1, 0, false)
	wd.step(time.Now())
	if p.X != x {
		t.Fatal("pausado: andou")
	}
	// Com outro jogador ativo, o mundo segue.
	id2 := wd.Join("outro", 0)
	if wd.frozen() {
		t.Fatal("com alguém jogando o mundo não para")
	}
	wd.SetPaused(id, false)
	wd.SetInput(id, 1, 0, false)
	wd.step(time.Now())
	if p.X == x {
		t.Fatal("despausado: devia andar")
	}
	_ = id2
	// O relógio desconta o tempo parado.
	wd.pausedNs.Store(int64(time.Hour))
	if d := time.Since(wd.clock()); d < 59*time.Minute {
		t.Fatalf("relógio do mundo devia estar 1 h atrás, está %v", d)
	}
}

// TestSalvarPosicao: o save guarda onde o personagem estava, e ele volta lá.
func TestSalvarPosicao(t *testing.T) {
	dir := t.TempDir()
	wd := NewWorld(1)
	wd.saveDir = dir
	id, err := wd.JoinSaved("viajante", 0, "segredo-1", "")
	if err != nil {
		t.Fatal(err)
	}
	p := wd.players[id]
	p.Intro = len(prologue) // fora do prólogo
	x, y := villageX+60, villageY+20
	for !wd.fits(x, y) {
		x++
	}
	p.X, p.Y = x, y
	before := p.saves
	wd.SaveNow(id)
	if p.saves != before+1 {
		t.Fatal("salvar agora devia contar a gravação")
	}
	wd.Leave(id)
	id, err = wd.JoinSaved("viajante", 0, "segredo-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if q := wd.players[id]; q.X != x || q.Y != y {
		t.Fatalf("voltou em (%d,%d), esperava (%d,%d)", q.X, q.Y, x, y)
	}
}
