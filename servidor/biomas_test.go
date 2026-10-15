package main

import "testing"

// TestBiomasNovos: cada bioma da escala 3 é região, tem os dois inimigos
// dentro, fogueira em chão livre e chão andável no meio (o brejo é lama).
func TestBiomasNovos(t *testing.T) {
	wd := NewWorld(1)
	for _, b := range newBiomes {
		r := regionAt(int(b.x), int(b.y))
		if r < 0 || regions[r].Name != b.name {
			t.Errorf("%s não é região no centro", b.name)
		}
		for _, k := range b.mobs {
			n := 0
			for _, m := range wd.mobs {
				if m.Kind == k && regionAt(int(m.X), int(m.Y)) >= 0 && regions[regionAt(int(m.X), int(m.Y))].Name == b.name {
					n++
				}
			}
			if n < 3 {
				t.Errorf("%s: só %d %s", b.name, n, mobKinds[k].Name)
			}
		}
		found := false
		for _, f := range wd.bonfires {
			if f.Name == b.fireName {
				found = true
				if _, _, ok := wd.freeNear(f.X, f.Y, 6); !ok {
					t.Errorf("%s em lugar preso", f.Name)
				}
			}
		}
		if !found {
			t.Errorf("%s sem fogueira", b.name)
		}
	}
	if wd.bonfires[wd.introBonfire()].Name != "Fogueira do Despertar" {
		t.Error("fogueira do prólogo errada")
	}
}
