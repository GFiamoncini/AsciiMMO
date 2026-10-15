package main

import (
	"testing"
	"time"
)

// TestProfundezas: cada fenda desce e a escada sobe; as fendas, as fogueiras
// de baixo e o coração estão ligados por galerias; a lava queima, as Botas
// de Obsidiana protegem e o Manto de Cinzas alivia.
func TestProfundezas(t *testing.T) {
	t0 := time.Now()
	wd := NewWorld(1)
	t.Logf("mundo com Profundezas: %v, %dx%d", time.Since(t0), wd.w, wd.h)
	p := wd.spawn("fundo", 0)
	wd.players[p.ID] = p
	for i, r := range rifts {
		p.X, p.Y = r[0], r[1]
		wd.enter(p)
		if !inDeep(p.Y) || !wd.Walkable(p.X, p.Y) {
			t.Fatalf("fenda %d devia levar a chão das Profundezas: (%d,%d)", i, p.X, p.Y)
		}
		p.X, p.Y = r[0], deepY0+r[1]
		wd.enter(p)
		if inDeep(p.Y) || !wd.Walkable(p.X, p.Y) {
			t.Fatalf("a escada %d devia subir a chão livre: (%d,%d)", i, p.X, p.Y)
		}
	}
	// Galerias ligadas: busca em largura a partir da primeira fenda, com o
	// tamanho do personagem (fits: 5 de lado), não só célula livre. Antes os
	// corredores finos passavam aqui e o jogador ficava preso neles.
	seen := make([]bool, wd.w*wd.h)
	r0 := rifts[0]
	q := []int{(deepY0+r0[1]+7)*wd.w + r0[0]}
	seen[q[0]] = true
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x, y := i%wd.w, i/wd.w
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			j := ny*wd.w + nx
			if nx < 0 || ny < deepY0 || nx >= wd.w || ny >= wd.h || seen[j] || !wd.fits(nx, ny) {
				continue
			}
			seen[j] = true
			q = append(q, j)
		}
	}
	reach := func(x, y int) bool {
		for dy := -3; dy <= 3; dy++ {
			for dx := -3; dx <= 3; dx++ {
				if wd.inBounds(x+dx, y+dy) && seen[(y+dy)*wd.w+x+dx] {
					return true
				}
			}
		}
		return false
	}
	for i, r := range rifts {
		if !reach(r[0], deepY0+r[1]+7) {
			t.Errorf("a fenda %d não se liga às outras por baixo", i)
		}
	}
	if !reach(heartX, heartY) {
		t.Error("o coração (Ignar) não se liga às fendas")
	}
	for _, b := range wd.bonfires {
		if inDeep(b.Y) && !reach(b.X, b.Y) {
			t.Errorf("%s fora das galerias: (%d,%d)", b.Name, b.X, b.Y)
		}
	}
	boss := false
	for _, s := range wd.spawns {
		boss = boss || s.Kind == mobIgnar
	}
	if !boss || regions[regionAt(heartX, heartY)].Name != "Ventre de Fogo" {
		t.Fatal("Ignar devia morar no Ventre de Fogo")
	}
	// Lava.
	lava := -1
	for i := deepY0 * wd.w; i < len(wd.marsh); i++ {
		if wd.marsh[i] == lavaMarsh {
			lava = i
			break
		}
	}
	if lava < 0 || wd.flagsAt(lava)&flagLava == 0 {
		t.Fatal("devia haver lava, com a flag")
	}
	now := time.Now()
	p.X, p.Y = lava%wd.w, lava/wd.w
	p.HP = p.MaxHP
	wd.lavaStep(p, now)
	burn := p.MaxHP - p.HP
	p.HP, p.Skin = p.MaxHP, itemAshMantle
	wd.lavaStep(p, now)
	if burn <= 0 || p.MaxHP-p.HP >= burn {
		t.Fatalf("lava devia queimar (%d) e o manto aliviar (%d)", burn, p.MaxHP-p.HP)
	}
	p.HP, p.Boots = p.MaxHP, itemObsidianBoots
	wd.lavaStep(p, now)
	if p.HP != p.MaxHP {
		t.Fatal("com Botas de Obsidiana a lava não devia queimar")
	}
	for _, part := range []int{0, 1} {
		b, _ := wd.MapBlob(part)
		t.Logf("parte %d do mapa: %d KB", part, len(b)>>10)
	}
}

// TestArvoreDivina: a Árvore Divina com Grael e os Apóstolos; a Armadura do
// Abismo dá força a quem sangra e cobra vida; o Ovo Escarlate salva uma vez.
func TestArvoreDivina(t *testing.T) {
	wd := NewWorld(1)
	if i := regionAt(treeX, treeY); i < 0 || regions[i].Name != "Árvore Divina" {
		t.Fatal("a Árvore Divina devia ter região própria")
	}
	grael, apostles := false, 0
	for _, s := range wd.spawns {
		grael = grael || s.Kind == mobGrael
		if s.Kind == mobApostle {
			apostles++
		}
	}
	if !grael || apostles < 4 {
		t.Fatalf("Grael e os Apóstolos deviam rondar a árvore: %v %d", grael, apostles)
	}
	p := wd.spawn("guts", 0)
	now := time.Now()
	p.Skin, p.HP = itemAbyssArmor, p.MaxHP
	if p.abyssPct() != 0 {
		t.Fatal("com vida cheia a armadura não devia dar força")
	}
	wd.abyssStep(p, now)
	if p.HP != p.MaxHP-abyssDrain {
		t.Fatal("a armadura devia cobrar vida")
	}
	p.HP = p.MaxHP / 3
	if p.abyssPct() != abyssBonus {
		t.Fatal("abaixo de metade a armadura devia dar força")
	}
	p.Amulet, p.HP = itemScarletEgg, p.MaxHP/2
	wd.hurtRaw(p, p.MaxHP/2-5, "teste", now)
	if p.HP < p.MaxHP*eggHealPct/100 {
		t.Fatalf("o ovo devia devolver vida: %d", p.HP)
	}
	hp := p.HP
	wd.hurtRaw(p, hp-5, "teste", now)
	if p.HP != 5 {
		t.Fatal("o ovo só chora uma vez a cada 2 minutos")
	}
}

// deepReach é o chão das Profundezas que se alcança andando da primeira fenda.
func deepReach(wd *World) []bool {
	seen := make([]bool, wd.w*wd.h)
	r0 := rifts[0]
	q := []int{(deepY0+r0[1]+7)*wd.w + r0[0]}
	seen[q[0]] = true
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x, y := i%wd.w, i/wd.w
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			j := ny*wd.w + nx
			if nx < 0 || ny < deepY0 || nx >= wd.w || ny >= wd.h || seen[j] || wd.solid[j] {
				continue
			}
			seen[j] = true
			q = append(q, j)
		}
	}
	return seen
}

func near(wd *World, seen []bool, x, y, r int) bool {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if wd.inBounds(x+dx, y+dy) && seen[(y+dy)*wd.w+x+dx] {
				return true
			}
		}
	}
	return false
}

// TestProfundezas2: os Salões de Dorrim se ligam às fendas e o cofre só abre
// quebrando a parede; cada câmara secreta tem baú alcançável depois de
// quebrar as rachaduras; os inimigos novos existem; Ignar fecha a secundária.
func TestProfundezas2(t *testing.T) {
	wd := NewWorld(1)
	seen := deepReach(wd)
	if !near(wd, seen, dorrimX, dorrimY, 3) {
		t.Fatal("os Salões de Dorrim deviam se ligar às fendas")
	}
	if near(wd, seen, dorrimVaultX, dorrimVaultY, 1) {
		t.Fatal("o cofre de Dorrim devia estar fechado antes de quebrar a parede")
	}
	broke := 0
	for _, pr := range wd.props {
		if pr.Kind == propCrack && inDeep(pr.Y) {
			pr.Alive = false
			wd.edit(pr.off)
			broke++
		}
	}
	if broke != 5 {
		t.Fatalf("esperava 5 paredes rachadas nas Profundezas, há %d", broke)
	}
	seen = deepReach(wd)
	for _, c := range chests {
		if inDeep(c.Y) && !near(wd, seen, c.X, c.Y, 2) {
			t.Errorf("baú em (%d,%d) inalcançável mesmo com as paredes quebradas", c.X, c.Y)
		}
	}
	count := map[int]int{}
	for _, s := range wd.spawns {
		count[s.Kind]++
	}
	for _, k := range []int{mobSandScorpion, mobIceWraith, mobRootHorror, mobDwarfGuard} {
		if count[k] < 4 {
			t.Errorf("%s: só %d", mobKinds[k].Name, count[k])
		}
	}
	p := wd.spawn("fundo", 0)
	wd.players[p.ID] = p
	p.Quest = len(quests) - 1 // capítulo 6
	now := time.Now()
	for _, m := range wd.mobs {
		if m.Kind == mobIgnar {
			m.bornAt = time.Time{}
			p.X, p.Y = int(m.X)+4, int(m.Y) // o crédito vai para quem está perto
			wd.mobDied(m, p, now)
		}
	}
	if p.SideDone&(1<<7) == 0 {
		t.Fatal("derrubar Ignar devia fechar 'O coração que arde'")
	}
}

// TestVulkhar: o Abismo tem precipício e ponte, Vulkhar mora do outro lado e
// se chega a ele andando; o chicote fere quem está na linha marcada e só
// ele; a aura queima quem está colado; abaixo de metade entra em fúria.
func TestVulkhar(t *testing.T) {
	wd := NewWorld(1)
	if wd.Walkable(abyssX+bridgeHalf+4, abyssY) || !wd.Walkable(abyssX, abyssY) {
		t.Fatal("o precipício devia barrar e a ponte deixar passar")
	}
	seen := deepReach(wd)
	if !near(wd, seen, abyssX, abyssY+vulkharHomeDY, 3) {
		t.Fatal("devia dar para chegar a Vulkhar pela ponte")
	}
	var m *Mob
	for _, x := range wd.mobs {
		if x.Kind == mobVulkhar {
			m = x
		}
	}
	if m == nil {
		t.Fatal("Vulkhar devia estar no mundo")
	}
	in := wd.spawn("na linha", 0)
	out := wd.spawn("de lado", 0)
	wd.players[in.ID], wd.players[out.ID] = in, out
	in.Level, out.Level = 30, 30
	in.recalc()
	out.recalc()
	in.X, in.Y = int(m.X)+30, int(m.Y)
	out.X, out.Y = int(m.X)+30, int(m.Y)+20
	in.HP, out.HP = in.MaxHP, out.MaxHP
	now := time.Now()
	k := mobKinds[mobVulkhar]
	wd.demon.auraAt = now.Add(time.Hour) // só o chicote, por ora
	if !wd.demonStep(m, k, in, 30, now) || !wd.demon.whipArmed {
		t.Fatal("o chicote devia ser armado")
	}
	in.X += 4 // andar um pouco na linha não salva
	wd.demonStep(m, k, in, 30, now.Add(whipWarn+time.Millisecond))
	if in.HP >= in.MaxHP || out.HP != out.MaxHP {
		t.Fatalf("o chicote devia ferir só quem está na linha: %d/%d e %d/%d", in.HP, in.MaxHP, out.HP, out.MaxHP)
	}
	// Aura.
	out.X, out.Y = int(m.X)+6, int(m.Y)
	out.HP = out.MaxHP
	wd.demon.auraAt = time.Time{}
	wd.demonStep(m, k, nil, 99, now.Add(2*time.Second))
	if out.HP >= out.MaxHP {
		t.Fatal("a aura devia queimar quem está colado")
	}
	m.HP = m.MaxHP / 3
	wd.demonStep(m, k, nil, 99, now.Add(3*time.Second))
	if !wd.demon.raged {
		t.Fatal("abaixo de metade devia entrar em fúria")
	}
}

// TestPortoes: o de Cinzalva acende de chegar perto e dá pedras; os outros só
// com a runa certa; atravessar entre acesos gasta uma Pedra de Eco; tudo
// volta no save. As Cavernas de Cristal existem, com Golens.
func TestPortoes(t *testing.T) {
	wd := NewWorld(1)
	wd.saveDir = t.TempDir()
	id, _ := wd.JoinSaved("viajante", 0, "segredo", "")
	p := wd.players[id]
	now := time.Now()
	for i, g := range wd.gates {
		x, y, ok := wd.freeNear(g.X, g.Y+4, 14)
		if !ok || !wd.Walkable(x, y) {
			t.Fatalf("%s sem chão em volta", g.Name)
		}
		_ = i
	}
	g0, g1 := wd.gates[0], wd.gates[1]
	p.X, p.Y = g0.X, g0.Y+3
	wd.gateFree(p, now)
	if !p.gateLit(0) || p.count(itemEchoStone) != 3 {
		t.Fatalf("o portão de Cinzalva devia acender e dar 3 pedras: %b %d", p.Gates, p.count(itemEchoStone))
	}
	p.X, p.Y = g1.X, g1.Y+3
	wd.GateRune(id, 1, "UUUU")
	if p.gateLit(1) {
		t.Fatal("runa errada não devia acender")
	}
	wd.GateRune(id, 1, g1.Seq)
	if !p.gateLit(1) {
		t.Fatal("a runa certa devia acender")
	}
	wd.GateGo(id, 0)
	if dist(float64(p.X), float64(p.Y), float64(g0.X), float64(g0.Y)) > 16 || p.count(itemEchoStone) != 2 {
		t.Fatalf("devia atravessar até Cinzalva gastando uma pedra: (%d,%d) %d", p.X, p.Y, p.count(itemEchoStone))
	}
	wd.persist(p)
	wd.Leave(id)
	id, _ = wd.JoinSaved("viajante", 0, "segredo", "")
	if wd.players[id].Gates != 3 {
		t.Fatalf("o save devia guardar os portões: %b", wd.players[id].Gates)
	}
	if i := regionAt(crystalX, crystalY); i < 0 || regions[i].Name != "Cavernas de Cristal" {
		t.Fatal("devia haver Cavernas de Cristal")
	}
	golems := 0
	for _, s := range wd.spawns {
		if s.Kind == mobCrystalGolem {
			golems++
		}
	}
	if golems < 4 {
		t.Fatalf("poucos Golens de Cristal: %d", golems)
	}
	seen := deepReach(wd)
	for _, g := range wd.gates {
		if inDeep(g.Y) && !near(wd, seen, g.X, g.Y+3, 4) {
			t.Errorf("%s fora das galerias", g.Name)
		}
	}
}

// TestCofreDeCristal: o cofre só abre quando os três prismas estão quebrados.
func TestCofreDeCristal(t *testing.T) {
	wd := NewWorld(1)
	cx, by := crystalVault()
	var door *prop
	var prisms []*prop
	for _, pr := range wd.props {
		switch {
		case pr.Kind == propSeal && pr.X == cx && pr.Y == by:
			door = pr
		case pr.Kind == propPrism:
			prisms = append(prisms, pr)
		}
	}
	if door == nil || len(prisms) != 3 {
		t.Fatalf("devia haver a porta do cofre e 3 prismas: %v %d", door != nil, len(prisms))
	}
	now := time.Now()
	for k, pr := range prisms {
		pr.Alive = false
		wd.edit(pr.off)
		wd.prismsBroken(nil, now)
		if open := !door.Alive; open != (k == 2) {
			t.Fatalf("com %d prismas quebrados a porta aberta = %v", k+1, open)
		}
	}
	if !wd.Walkable(cx, by) {
		t.Fatal("a porta aberta devia deixar passar")
	}
}
