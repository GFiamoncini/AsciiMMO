package main

import "testing"

// TestCasa: comprar na porta, guardar e tirar itens e dinheiro, e o baú
// voltar no save.
func TestCasa(t *testing.T) {
	dir := t.TempDir()
	wd := NewWorld(1)
	wd.saveDir = dir
	id, err := wd.JoinSaved("dono", 0, "seg-casa", "")
	if err != nil {
		t.Fatal(err)
	}
	p := wd.players[id]
	p.Intro = len(prologue)
	x, y := lotDoor(0)
	if !wd.fits(x, y) {
		t.Fatalf("a frente da porta da %s não é livre: (%d,%d)", lots[0].Name, x, y)
	}
	p.X, p.Y = x, y
	p.Gold = 10
	wd.BuyHouse(id, 0)
	if p.House != 0 {
		t.Fatal("comprou sem dinheiro")
	}
	p.Gold = lots[0].Price + 300
	wd.BuyHouse(id, 0)
	if p.House != 1 || p.Gold != 300 {
		t.Fatalf("compra: casa %d, ouro %d", p.House, p.Gold)
	}
	p.give(itemPotion, 5)
	wd.Stash(id, 0, itemPotion, 3)
	wd.Stash(id, 2, 0, 200)
	if p.count(itemPotion) != 2 || stashCount(p.Stash, itemPotion) != 3 || p.Gold != 100 || p.StashGold != 200 {
		t.Fatalf("guardar: mochila %d, baú %d, ouro %d/%d", p.count(itemPotion), stashCount(p.Stash, itemPotion), p.Gold, p.StashGold)
	}
	wd.Stash(id, 1, itemPotion, 1)
	wd.Stash(id, 3, 0, 50)
	wd.Leave(id)
	id, _ = wd.JoinSaved("dono", 0, "seg-casa", "")
	q := wd.players[id]
	if q.House != 1 || stashCount(q.Stash, itemPotion) != 2 || q.StashGold != 150 || q.Gold != 150 {
		t.Fatalf("depois de voltar: casa %d, baú %d, ouro %d/%d", q.House, stashCount(q.Stash, itemPotion), q.Gold, q.StashGold)
	}
	// Longe da porta o baú não abre.
	q.X += 40
	wd.Stash(id, 3, 0, 150)
	if q.StashGold != 150 {
		t.Fatal("tirou dinheiro longe da casa")
	}
}
