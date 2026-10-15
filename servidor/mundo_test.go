package main

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestBiomas(t *testing.T) {
	wd := NewWorld(1)
	count := map[int]int{}
	for _, m := range wd.mobs {
		count[m.Kind]++
	}
	// Todo tipo de inimigo existe no mundo, e os comuns em número que dê caça.
	for k, def := range mobKinds {
		if min := map[bool]int{true: 1, false: 4}[def.Boss]; count[k] < min {
			t.Errorf("%s: só %d no mundo", def.Name, count[k])
		}
	}
	// Dá para chegar ao centro de cada região andando: o chão ali é livre.
	for _, r := range regions {
		free := false
		for dy := -12; dy <= 12 && !free; dy++ {
			for dx := -12; dx <= 12 && !free; dx++ {
				free = wd.fits(r.X+dx, r.Y+dy)
			}
		}
		if !free {
			t.Errorf("%s: sem chão livre no centro", r.Name)
		}
	}
	// As missões só pedem inimigos que existem.
	for _, q := range quests {
		for _, k := range q.Targets {
			if q.Kind == qKill && k != killDragon && count[k] == 0 {
				t.Errorf("missão %q pede um inimigo que não nasce", q.Title)
			}
		}
	}
}

func TestNatureza(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ivo", 0)
	p := wd.players[id]
	now := time.Now()

	// Árvore derrubada: some do mapa, larga madeira e vira mudança para os clientes.
	var tree *prop
	for _, pr := range wd.props {
		if pr.Kind == propTree && len(pr.on) > 0 && wd.fits(pr.X+6, pr.Y) {
			tree = pr
			break
		}
	}
	if tree == nil {
		t.Fatal("nenhuma árvore quebrável com chão livre ao lado")
	}
	ver := wd.mapVer
	tree.HP = 1
	p.X, p.Y, p.FaceX, p.FaceY, p.ready = tree.X+6, tree.Y, -1, 0, [maxSkills]time.Time{}
	wd.Attack(id)
	if tree.Alive || wd.mapVer < ver+1 || wd.cellAt(tree.off[0].I) != tree.off[0] {
		t.Fatalf("árvore: viva=%v versão %d→%d", tree.Alive, ver, wd.mapVer)
	}
	if cells, _, reload := wd.editsSince(ver); reload || len(cells) < len(tree.off) {
		t.Fatalf("mudanças: %d células, reload=%v", len(cells), reload)
	}
	wood := false
	for _, l := range wd.loot {
		wood = wood || l.Item == itemWood
	}
	if !wood {
		t.Fatal("árvore devia largar madeira")
	}
	wd.stepProps(now.Add(propDefs[propTree].Respawn + time.Second))
	if tree.Alive {
		t.Fatal("não devia renascer com o jogador em cima")
	}
	p.X += 40
	wd.stepProps(now.Add(propDefs[propTree].Respawn + time.Minute))
	if !tree.Alive || wd.cellAt(tree.on[0].I) != tree.on[0] {
		t.Fatal("árvore devia voltar")
	}

	// Cais: embarca, navega para oeste e desce ao encostar na margem.
	d := docks[0]
	p.X, p.Y = d[0]-4, d[1]
	wd.Interact(id)
	if !p.boat || !wd.floats(p.X, p.Y) {
		t.Fatalf("devia estar de barco na água: barco=%v em (%d,%d)", p.boat, p.X, p.Y)
	}
	for i := 0; i < 200 && p.boat; i++ {
		wd.sail(p, -1, 0, boatSpeed)
	}
	if p.boat || !wd.fits(p.X, p.Y) {
		t.Fatalf("devia ter descido em terra: barco=%v em (%d,%d)", p.boat, p.X, p.Y)
	}

	// Ilha: não tem cais. O barco fica atracado onde se desceu e E perto dele
	// embarca de novo (antes, quem descia na ilha ficava preso).
	p.X, p.Y, p.boat = islandX, islandY+22, true
	if !wd.floats(p.X, p.Y) {
		t.Fatalf("ponto de partida ao sul da ilha devia ser água: (%d,%d)", p.X, p.Y)
	}
	for i := 0; i < 200 && p.boat; i++ {
		wd.sail(p, 0, -1, boatSpeed)
	}
	if p.boat || islandDist(float64(p.X), float64(p.Y)) > 1.2 || !p.Moored {
		t.Fatalf("devia ter descido na ilha com o barco atracado: barco=%v em (%d,%d) atracado=%v", p.boat, p.X, p.Y, p.Moored)
	}
	for _, d := range docks {
		if dist(float64(p.X), float64(p.Y), float64(d[0]), float64(d[1])) <= dockReach {
			t.Fatal("a ilha não devia ter cais por perto; o teste perderia o sentido")
		}
	}
	wd.Interact(id)
	if !p.boat || !wd.floats(p.X, p.Y) || p.Moored {
		t.Fatalf("E devia embarcar de novo no barco atracado: barco=%v em (%d,%d)", p.boat, p.X, p.Y)
	}
	for i := 0; i < 200 && p.boat; i++ {
		wd.sail(p, 0, 1, boatSpeed)
	}
	if p.boat || islandDist(float64(p.X), float64(p.Y)) < 1.5 {
		t.Fatalf("devia ter saído da ilha: barco=%v em (%d,%d)", p.boat, p.X, p.Y)
	}

	// Montaria: o cavalo doma no nível 1; o raptor pede nível 8.
	find := func(kind int) *Steed {
		for _, s := range wd.steeds {
			if s.Kind == kind && s.Alive {
				return s
			}
		}
		t.Fatalf("sem montaria do tipo %d", kind)
		return nil
	}
	for k := range steedKinds {
		find(k)
	}
	rap := find(steedRaptor)
	p.X, p.Y = int(rap.X), int(rap.Y)
	wd.Interact(id)
	if p.Steeds != 0 || !rap.Alive {
		t.Fatal("raptor não devia aceitar nível 1")
	}
	horse := find(steedHorse)
	p.X, p.Y = int(horse.X), int(horse.Y)
	p.ST = 0
	wd.Interact(id)
	if p.tame != nil {
		t.Fatal("sem fôlego a doma não devia começar")
	}
	p.ST = float64(p.MaxST)
	wd.Interact(id)
	if p.tame == nil || len(p.tame.seq) != tameNeed[steedHorse].Len {
		t.Fatalf("doma do cavalo devia pedir %d setas: %+v", tameNeed[steedHorse].Len, p.tame)
	}
	wd.Tame(id, "X") // errou: o cavalo foge e a stamina cai
	if p.Steeds != 0 || p.ST >= float64(p.MaxST) || !horse.Alive {
		t.Fatalf("errar a sequência: donos=%b st=%v", p.Steeds, p.ST)
	}
	p.X, p.Y, p.ST = int(horse.X), int(horse.Y), float64(p.MaxST)
	wd.Interact(id)
	wd.Tame(id, p.tame.seq)
	if p.Steeds&(1<<steedHorse) == 0 || p.Riding != steedHorse+1 || horse.Alive {
		t.Fatalf("cavalo: donos=%b montado=%d", p.Steeds, p.Riding)
	}
	x0 := p.X
	wd.SetInput(id, 1, 0, false)
	wd.step(now)
	wd.SetInput(id, 0, 0, false)
	if p.X-x0 != walkSpeed+steedKinds[steedHorse].Speed && wd.fits(x0+walkSpeed+steedKinds[steedHorse].Speed, p.Y) {
		t.Fatalf("a cavalo devia andar %d, andou %d", walkSpeed+steedKinds[steedHorse].Speed, p.X-x0)
	}
	wd.Ride(id)
	if p.Riding != 0 {
		t.Fatal("X devia desmontar")
	}
	wd.Ride(id)
	if p.Riding != steedHorse+1 {
		t.Fatal("X devia montar de novo no cavalo")
	}

	// Charco venenoso envenena; a erva cura e protege.
	poison := -1
	for i, m := range wd.marsh {
		if m == 2 && wd.fits(i%wd.w, i/wd.w) {
			poison = i
			break
		}
	}
	if poison < 0 {
		t.Fatal("nenhum charco venenoso pisável")
	}
	p.X, p.Y, p.HP = poison%wd.w, poison/wd.w, p.MaxHP
	wd.environ(p, now)
	if !now.Before(p.status[stPoison]) || p.HP != p.MaxHP-statusDmg[stPoison] {
		t.Fatalf("veneno: até %v, HP %d/%d", p.status[stPoison], p.HP, p.MaxHP)
	}
	p.give(itemAntidote, 1)
	wd.UseItem(id, itemAntidote)
	if !p.status[stPoison].IsZero() || p.count(itemAntidote) != 0 {
		t.Fatal("antídoto devia curar o veneno")
	}
	wd.environ(p, time.Now())
	if !p.status[stPoison].IsZero() {
		t.Fatal("logo depois do antídoto o charco não devia envenenar")
	}

	// Frio nos picos, longe do fogo.
	p.X, p.Y = snowX+60, snowY+40
	wd.environ(p, now)
	if !now.Before(p.status[stCold]) && !wd.nearFire(p) {
		t.Fatal("os picos de gelo deviam gelar")
	}

	// Ervas existem nos três biomas.
	kinds := map[int]bool{}
	for _, h := range wd.herbs {
		kinds[h.Kind] = true
	}
	if len(kinds) != 3 {
		t.Fatalf("ervas de %d tipos", len(kinds))
	}
}

func TestSubsolo(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ida", 0)
	p := wd.players[id]

	// Cada boca de caverna leva para dentro, em chão livre, e a saída devolve.
	for _, m := range caveMouths {
		p.X, p.Y, p.Riding = m.X, m.Y+2, 0
		wd.enter(p)
		if p.Y < overH || !wd.fits(p.X, p.Y) {
			t.Fatalf("boca (%d,%d): chegou em (%d,%d)", m.X, m.Y, p.X, p.Y)
		}
		p.X, p.Y = m.TX-26, m.TY-2
		wd.enter(p)
		if p.Y >= overH || !wd.fits(p.X, p.Y) {
			t.Fatalf("saída da caverna (%d,%d) levou a (%d,%d)", m.TX, m.TY, p.X, p.Y)
		}
	}

	// Cair no buraco machuca e leva à gruta; a corda devolve ao campo.
	p.X, p.Y, p.HP = holeX, holeY+1, p.MaxHP
	wd.enter(p)
	if p.Y < overH || p.HP != p.MaxHP-12 || !wd.fits(p.X, p.Y) {
		t.Fatalf("buraco: em (%d,%d) com HP %d", p.X, p.Y, p.HP)
	}
	p.X, p.Y = ropeX, ropeY+2
	wd.enter(p)
	if p.Y >= overH || !wd.fits(p.X, p.Y) {
		t.Fatalf("a corda devia devolver ao campo: (%d,%d)", p.X, p.Y)
	}

	// A parede rachada fecha a sala secreta até ser quebrada.
	var crack *prop
	for _, pr := range wd.props {
		if pr.Kind == propCrack {
			crack = pr
		}
	}
	if crack == nil || len(crack.on) == 0 || wd.Walkable(crack.X, crack.Y) {
		t.Fatal("devia haver uma parede rachada fechando o túnel")
	}
	crack.HP = 1
	p.X, p.Y, p.ready = crack.X+8, crack.Y, [maxSkills]time.Time{}
	wd.Use(id, 0, func() *float64 { a := math.Pi; return &a }())
	if crack.Alive || !wd.Walkable(crack.X, crack.Y) {
		t.Fatal("a parede rachada devia abrir")
	}

	// Ratos e morcegos moram nas cavernas, não na porta da vila.
	for _, m := range wd.mobs {
		if (m.Kind == mobRat || m.Kind == mobBat) && m.Y < overH {
			t.Errorf("%s fora da caverna em (%v,%v)", mobKinds[m.Kind].Name, m.X, m.Y)
		}
	}
}

func TestPurificacao(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("zeca", 0)
	p := wd.players[id]
	now := time.Now()

	// A cripta tem fogueira e os esqueletos de lá pertencem a ela.
	crypt := -1
	for i, b := range wd.bonfires {
		if b.Name == "Fogueira da Cripta" {
			crypt = i
		}
	}
	if crypt < 0 || len(wd.zones[crypt].mobs) == 0 || wd.bonfires[crypt].Y < overH {
		t.Fatal("a cripta devia ter fogueira com os esqueletos dela")
	}
	b := wd.bonfires[crypt]
	p.X, p.Y = b.X, b.Y+2

	// Descansar com inimigo vivo não purifica.
	wd.Interact(id)
	if now.Before(wd.zones[crypt].purged) {
		t.Fatal("não devia purificar com inimigo vivo")
	}
	// Limpa a área, descansa: ninguém renasce enquanto durar a purificação.
	for _, m := range wd.zones[crypt].mobs {
		m.Alive, m.respawnAt = false, now
	}
	p.HP = 1
	wd.Interact(id)
	if !time.Now().Before(wd.zones[crypt].purged) || p.HP != p.MaxHP || p.Bonfire != crypt {
		t.Fatalf("descanso: purificada até %v, HP %d, fogueira %d", wd.zones[crypt].purged, p.HP, p.Bonfire)
	}
	wd.stepMobs(now.Add(bossRespawn + time.Minute))
	for _, m := range wd.zones[crypt].mobs {
		if m.Alive {
			t.Fatal("área purificada não devia ter renascimento")
		}
	}
	wd.stepMobs(now.Add(purifyFor + time.Minute))
	alive := 0
	for _, m := range wd.zones[crypt].mobs {
		if m.Alive {
			alive++
		}
	}
	if alive == 0 {
		t.Fatal("depois da purificação os inimigos deviam voltar")
	}
	if mobKinds[mobSkeleton].Respawn != mobRespawn || mobKinds[mobLord].Respawn != bossRespawn {
		t.Fatal("tempos de renascimento errados")
	}
}

// Ruínas de Vaelor e o kit de construção: dá para entrar pelo cais e pelo
// portão e pela brecha oeste; a fachada passa por cima do pátio; janelas
// acesas brilham; clarear uma cor não estoura o canal.
func TestRuinasEKit(t *testing.T) {
	wd := NewWorld(1)
	if r := regionAt(castleX, castleCY); r < 0 || regions[r].Name != "Ruínas de Vaelor" {
		t.Fatalf("região do castelo: %d", r)
	}
	for _, p := range [][2]int{{castleX, castleBY + 5}, {castleX, castleBY - 1}, {castleX, castleBY - 30}, {castleX0 + 3, breachY}} {
		if !wd.Walkable(p[0], p[1]) {
			t.Errorf("castelo: (%d,%d) deveria ser transitável", p[0], p[1])
		}
	}
	i := (castleBY-20)*wd.w + castleX - 40
	if !wd.over[i] || wd.solid[i] {
		t.Errorf("fachada a 20 linhas da base devia passar por cima sem bloquear")
	}
	if j := (castleBY-1)*wd.w + castleX - 40; !wd.solid[j] {
		t.Errorf("o pé da fachada devia bloquear")
	}
	glow := 0
	hs := houses[0]
	for y := villageY + hs[1]; y < villageY+hs[1]+hs[3]; y++ {
		for x := villageX + hs[0]; x < villageX+hs[0]+hs[2]; x++ {
			if wd.glow[y*wd.w+x] {
				glow++
			}
		}
	}
	if glow == 0 || wd.flagsAt((villageY+hs[1])*wd.w)&flagGlow != 0 {
		t.Errorf("janelas acesas da casa: %d células com brilho", glow)
	}
	if c := dim(0xffd070, 1.3); c>>16 != 0xff || c > 0xffffff {
		t.Errorf("dim estourou o canal: %x", uint32(c))
	}
}

func TestEscolherESoltarMontaria(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ivo", 0)
	p := wd.players[id]
	p.Steeds = 0b101
	wd.PickSteed(id, 2)
	if p.Riding != 3 || p.LastSteed != 2 {
		t.Fatalf("devia montar a montaria 2: riding %d", p.Riding)
	}
	wd.PickSteed(id, 1)
	if p.Riding != 3 {
		t.Fatal("não devia montar uma que não domou")
	}
	wd.ReleaseSteed(id, 2)
	if p.Riding != 0 || p.Steeds != 0b001 {
		t.Fatalf("soltar: riding %d, domadas %b", p.Riding, p.Steeds)
	}
}

// Portas seladas: a palavra certa dita diante do Portão do Amigo abre; a
// sequência lida nas pedras abre a Cripta dos Ecos; errado não abre.
func TestPortasSeladas(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("tom", 0)
	p := wd.players[id]
	a, e := seals[0], seals[1]
	if wd.Walkable(a.X, a.BY) || wd.Walkable(e.X, e.BY) {
		t.Fatal("as portas seladas deviam bloquear")
	}
	if len(e.Clues) != len(e.Seq) {
		t.Fatalf("pedras rúnicas: %d para %d setas", len(e.Clues), len(e.Seq))
	}
	p.X, p.Y = a.X, a.BY+5
	wd.Say(id, "abre-te sésamo")
	if wd.Walkable(a.X, a.BY) {
		t.Fatal("palavra errada não devia abrir")
	}
	wd.Say(id, "Amigo")
	if !wd.Walkable(a.X, a.BY) {
		t.Fatal("dizer amigo devia abrir o portão")
	}
	p.X, p.Y = e.X, e.BY+5
	wd.Unseal(id, "UUUUU")
	if wd.Walkable(e.X, e.BY) {
		t.Fatal("sequência errada não devia abrir")
	}
	wd.Unseal(id, e.Seq)
	if !wd.Walkable(e.X, e.BY) {
		t.Fatal("a sequência das pedras devia abrir a cripta")
	}
}

// TestLeituras: o que tem texto no mapa se lê com E — as cinco pedras da
// Cripta dos Ecos (com a seta por extenso), a inscrição da porta da palavra,
// as placas e os postes de direção.
func TestLeituras(t *testing.T) {
	wd := NewWorld(1)
	stones, posts, signs, inscr := 0, 0, 0, false
	for _, r := range wd.reads {
		switch {
		case strings.HasPrefix(r.Title, "Pedra rúnica"):
			stones++
			if !strings.ContainsAny(r.Text, "↑↓←→") {
				t.Errorf("%s sem a seta: %q", r.Title, r.Text)
			}
		case r.Title == "Placa de direção":
			posts++
		case r.Title == "Placa":
			signs++
		case r.Title == "Portão do Amigo":
			inscr = strings.Contains(r.Text, "AMIGO")
		}
	}
	if stones != 5 || posts == 0 || signs == 0 || !inscr {
		t.Fatalf("leituras: %d pedras, %d postes, %d placas, inscrição %v", stones, posts, signs, inscr)
	}
}
