package main

import (
	"strings"
	"testing"
	"time"
)

func TestCombate(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ana", 0)
	p := wd.players[id]

	wd.Attack(id)
	if wd.boss.HP != bossMaxHP {
		t.Fatalf("ataque de longe não devia acertar: HP %d", wd.boss.HP)
	}

	p.X, p.Y, p.ready = lairX+8, lairY, [maxSkills]time.Time{}
	wd.Attack(id)
	if wd.boss.HP >= bossMaxHP {
		t.Fatal("ataque de perto devia tirar vida do chefe")
	}

	// O pisão arma com o jogador por perto e, um segundo depois, machuca.
	now := time.Now()
	wd.step(now)
	if !wd.boss.armed {
		t.Fatal("chefe devia armar o pisão")
	}
	wd.step(now.Add(1100 * time.Millisecond))
	// Nível 1 contra Tiamath (caveira): o pisão fere ou mata (e manda à fogueira).
	if p.HP >= p.MaxHP && dist(float64(p.X), float64(p.Y), lairX, lairY) < 30 {
		t.Fatal("pisão devia ferir o jogador dentro do raio")
	}

	p.X, p.Y = lairX+8, lairY
	wd.boss.HP = 1
	p.ready = [maxSkills]time.Time{}
	wd.Attack(id)
	if wd.boss.Alive || p.Level != 2 || p.Gold != 500 {
		t.Fatalf("morte do chefe: alive=%v level=%d gold=%d", wd.boss.Alive, p.Level, p.Gold)
	}
}

func TestEsqueletos(t *testing.T) {
	wd := NewWorld(1)
	if len(wd.mobs) == 0 {
		t.Fatal("o cemitério devia ter esqueletos")
	}
	id := wd.Join("ana", 0)
	p, m := wd.players[id], wd.mobs[0]
	p.X, p.Y = int(m.homeX)+4, int(m.homeY)
	m.bornAt = time.Time{} // já fora da cova

	// Depois de sair da cova, o esqueleto ao lado bate no jogador.
	now := time.Now().Add(2 * mobRise)
	wd.step(now)
	if p.HP >= p.MaxHP {
		t.Fatal("esqueleto ao lado devia ferir o jogador")
	}

	// Ao cair, o esqueleto dá XP na hora e larga o dinheiro da tabela no chão.
	m.HP, p.XP = 1, 0
	wd.Attack(id)
	if m.Alive || p.XP != mobReward[m.Kind].XP && p.Level == 1 {
		t.Fatalf("morte do esqueleto: alive=%v XP=%d", m.Alive, p.XP)
	}
	var coins *Loot
	for i := range wd.loot {
		if wd.loot[i].Item == itemCoins {
			coins = &wd.loot[i]
		}
	}
	if coins == nil || coins.N <= 0 {
		t.Fatal("esqueleto devia largar dinheiro")
	}
	n := coins.N
	p.X, p.Y = coins.X, coins.Y
	wd.stepLoot(time.Now())
	if p.Gold != n {
		t.Fatalf("pegar o dinheiro: tem %d, esperava %d", p.Gold, n)
	}
	wd.step(time.Now().Add(mobKinds[m.Kind].Respawn + time.Second))
	if !m.Alive || m.HP != m.MaxHP {
		t.Fatal("esqueleto devia renascer na cova")
	}
}

func TestClasses(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("mia", 1)
	p, m := wd.players[id], wd.mobs[0]
	if p.MaxHP != classes[1].HP || p.MaxMP != classes[1].MP {
		t.Fatalf("mago entrou com HP %d e MP %d", p.MaxHP, p.MaxMP)
	}
	m.bornAt = time.Time{} // já fora da cova
	p.X, p.Y = int(m.X)-40, int(m.Y)
	for x := p.X; x <= int(m.X); x++ { // linha de tiro limpa, sem lápide no meio
		wd.solid[p.Y*wd.w+x] = false
	}

	// O golpe da cripta só existe depois de descer.
	wd.Use(id, 2, nil)
	if p.MP != p.MaxMP {
		t.Fatal("golpe trancado não devia gastar MP")
	}

	// A bola de fogo voa na direção da mira e explode no esqueleto.
	aim := 0.0
	wd.Use(id, 1, &aim)
	if len(wd.projs) != 1 {
		t.Fatalf("esperava 1 projétil, há %d", len(wd.projs))
	}
	now := time.Now()
	for i := 0; i < 40 && len(wd.projs) > 0; i++ {
		wd.stepProjs(now)
	}
	if len(wd.projs) != 0 || m.HP >= m.MaxHP {
		t.Fatalf("bola de fogo devia acertar: projéteis=%d HP=%d/%d", len(wd.projs), m.HP, m.MaxHP)
	}

	// Pisar no portal da capela leva à cripta e libera o terceiro golpe.
	p.X, p.Y = chapelX, chapelY+8
	for i := 0; i < 4; i++ {
		wd.Move(id, 0, -1)
	}
	if !p.Unlocked || p.Y < overH {
		t.Fatalf("devia estar na cripta com o golpe liberado: y=%d unlocked=%v", p.Y, p.Unlocked)
	}
	if !wd.fits(p.X, p.Y) {
		t.Fatal("chegou à cripta dentro da parede")
	}
	// Agora o Meteoro pode ser aprendido: ramo do fogo, anéis 1, 2 e 3.
	p.Level, p.Attr[attrMana], p.SkillPts = 9, 8, 3
	for _, n := range learnPath(p, 2) {
		wd.SpendTree(id, n)
	}
	if !p.knows(2) {
		t.Fatal("devia ter aprendido o Meteoro na teia")
	}
	p.MP = p.MaxMP
	wd.Use(id, 2, &aim)
	if p.MP != p.MaxMP-classes[1].Skills[2].Cost || len(wd.blasts) != 1 {
		t.Fatal("meteoro devia sair depois de liberado")
	}

	// Todo esqueleto da cripta nasce em chão livre, e a saída devolve à capela.
	for _, sp := range cryptSpawns {
		if !wd.fits(sp.X, sp.Y+cryptDY) {
			t.Errorf("cova da cripta em parede: %+v", sp)
		}
	}
	p.X, p.Y = cryptOutX, cryptOutY+6
	for i := 0; i < 4; i++ {
		wd.Move(id, 0, -1)
	}
	if p.Y >= overH {
		t.Fatalf("a escada devia levar de volta à capela: y=%d", p.Y)
	}

	// O bloqueio do cavaleiro corta o dano.
	kid := wd.Join("kai", 2)
	k := wd.players[kid]
	wd.Use(kid, 1, nil)
	wd.hurt(k, 100, "no teste", time.Now())
	if k.HP != k.MaxHP-30 {
		t.Fatalf("bloqueio devia deixar passar 30 de dano, passou %d", k.MaxHP-k.HP)
	}
}

func TestFogoInvasaoCritico(t *testing.T) {
	wd := NewWorld(1)
	wd.wxForced, wd.weather = wxClear, wxClear
	now := time.Now()

	// Fogo: acha capim, incendeia, vira cinza e espalha; a cinza volta a ser capim.
	grass := -1
	for y := villageY + 150; y < villageY+400 && grass < 0; y++ {
		for x := villageX - 200; x < villageX+200; x++ {
			if i := y*wd.w + x; wd.isGrass(i) {
				grass = i
				break
			}
		}
	}
	if grass < 0 {
		t.Fatal("nenhum capim perto da vila")
	}
	orig := wd.cellAt(grass)
	wd.ignite(float64(grass%wd.w), float64(grass/wd.w), 3, now)
	if !wd.burn[grass] || wd.flagsAt(grass)&flagFire == 0 {
		t.Fatal("o capim devia pegar fogo")
	}
	wd.stepFire(now.Add(3 * time.Second))
	if wd.burn[grass] || wd.glyph[grass] != '.' || wd.color[grass] != ashColor {
		t.Fatal("o capim queimado devia virar cinza")
	}
	wd.stepFire(now.Add(ashRegrow + time.Minute))
	if wd.cellAt(grass) != orig && !wd.burn[grass] {
		t.Fatal("a cinza devia voltar a ser capim")
	}
	// Na chuva não pega.
	wd.weather = wxRain
	burning := len(wd.burning)
	wd.ignite(float64(grass%wd.w), float64(grass/wd.w), 3, now)
	if len(wd.burning) != burning {
		t.Fatal("na chuva o capim não devia pegar fogo")
	}
	wd.weather = wxClear

	// Invasão: numa noite sorteada os mortos aparecem; caídos todos, acaba.
	wd.dayLen = 100
	night := wd.start.Add(45 * time.Second) // tod 0.8
	for k := 0; k < 20 && !wd.inv.active; k++ {
		wd.inv.lastNight = 0
		wd.stepInvasion(night)
	}
	if !wd.inv.active || len(wd.inv.mobs) == 0 {
		t.Fatal("a invasão devia começar numa noite")
	}
	n := len(wd.mobs)
	for _, m := range wd.inv.mobs {
		m.Alive = false
	}
	wd.stepInvasion(night)
	if wd.inv.active || len(wd.mobs) != n-invaders && len(wd.mobs) >= n {
		t.Fatal("com todos os invasores caídos a invasão devia acabar e eles sumirem")
	}

	// Crítico: em muitos golpes alguns saem críticos e o atacante vê os números.
	id := wd.Join("cris", 0)
	p := wd.players[id]
	b := &wd.boss
	b.HP = 1 << 30
	p.X, p.Y = int(b.X)+8, int(b.Y)
	crit := 0
	for k := 0; k < 200; k++ {
		p.ready = [maxSkills]time.Time{}
		wd.Attack(id)
		for _, pp := range p.pops {
			if strings.HasPrefix(pp.Text, "CRÍTICO") {
				crit++
			}
		}
		p.pops = nil
	}
	if crit == 0 || crit > 100 {
		t.Fatalf("críticos em 200 golpes: %d", crit)
	}
}

// learnPath devolve os nós da teia, do núcleo até o que ensina o golpe skill,
// na ordem em que precisam ser aprendidos.
func learnPath(p *Player, skill int) []int {
	w := p.web()
	target := -1
	for i, n := range w {
		if n.Skill == skill {
			target = i
		}
	}
	var path []int
	for i := target; i > 0; i = w[i].Links[0] {
		path = append([]int{i}, path...)
	}
	return path
}

// A teia: requisitos, atalhos, golpes elementais e os efeitos nos inimigos.
func TestTeiaEElementos(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ana", 1)
	p := wd.players[id]
	if p.Bar[0] != 1 || !p.knows(0) || !p.knows(1) || p.knows(4) {
		t.Fatalf("mago novo: barra %v, sabe básico %v, classe %v, gelo %v", p.Bar, p.knows(0), p.knows(1), p.knows(4))
	}
	// Todo nó de toda classe liga ao núcleo por um caminho e tem descrição.
	for ci, c := range classes {
		if len(c.Skills) > maxSkills {
			t.Fatalf("%s tem golpes demais", c.Name)
		}
		for i, n := range c.Web {
			if n.Desc == "" || n.Icon == "" {
				t.Errorf("%s: nó %d (%s) sem descrição ou ícone", c.Name, i, n.Name)
			}
			for j := i; j > 0; j = c.Web[j].Links[0] {
				if c.Web[j].Links[0] >= j {
					t.Fatalf("%s: nó %d não chega ao núcleo", c.Name, i)
				}
			}
		}
		taught := map[int]bool{0: true, 1: true}
		for _, n := range c.Web {
			if n.Skill >= 0 {
				taught[n.Skill] = true
			}
		}
		if len(taught) != c.techBase || len(c.Skills) != c.techBase+len(techs) { // as técnicas vêm do Mestre, não da teia
			t.Errorf("classe %d: %d golpes, a teia ensina %d", ci, len(c.Skills), len(taught))
		}
	}

	// Nível baixo trava; com nível e Mana abre, e o golpe vai para a barra.
	path := learnPath(p, 4) // Lança de Gelo: primeiro anel do gelo
	p.SkillPts = 5
	wd.SpendTree(id, path[0])
	if p.knows(4) {
		t.Fatal("nível 1 não devia aprender o primeiro anel")
	}
	p.Level, p.Attr[attrMana] = 2, 1
	wd.SpendTree(id, path[0])
	if !p.knows(4) || p.Bar[1] != 4 {
		t.Fatalf("Lança de Gelo: sabe %v, barra %v", p.knows(4), p.Bar)
	}
	wd.SetBar(id, 0, 4) // trocar de atalho troca os dois
	if p.Bar[0] != 4 || p.Bar[1] != 1 {
		t.Fatalf("troca de atalhos: %v", p.Bar)
	}

	// A lança de gelo deixa o esqueleto lento; a nova congela.
	m := wd.mobs[0]
	m.bornAt, m.HP = time.Time{}, 100000
	p.X, p.Y = int(m.X)-30, int(m.Y)
	for x := p.X; x <= int(m.X); x++ {
		wd.solid[p.Y*wd.w+x] = false
	}
	aim := 0.0
	wd.Use(id, 4, &aim)
	now := time.Now()
	for i := 0; i < 40 && len(wd.projs) > 0; i++ {
		wd.stepProjs(now)
	}
	if !now.Before(m.slowUntil) {
		t.Fatal("a lança de gelo devia deixar o alvo lento")
	}
	p.Level, p.Attr[attrMana], p.SkillPts, p.MP = 9, 8, 5, p.MaxMP
	for _, n := range learnPath(p, 5) {
		wd.SpendTree(id, n)
	}
	p.X, p.Y = int(m.X)-10, int(m.Y)
	wd.Use(id, 5, &aim)
	if !time.Now().Before(m.stunUntil) || mobStatus(m, time.Now())&2 == 0 {
		t.Fatal("a nova congelante devia congelar o esqueleto")
	}
	x0, y0 := m.X, m.Y
	wd.stepMobs(time.Now())
	if m.X != x0 || m.Y != y0 {
		t.Fatal("congelado não devia andar")
	}

	// Tomo do Recomeço devolve os pontos; Tônico do Esquecimento, os atributos.
	spent := 0
	for _, r := range p.Web[1:] {
		spent += r
	}
	pts := p.SkillPts
	p.resetWeb()
	if p.SkillPts != pts+spent || p.knows(4) || p.Bar[0] != 1 {
		t.Fatalf("recomeço: pontos %d (esperava %d), sabe gelo %v, barra %v", p.SkillPts, pts+spent, p.knows(4), p.Bar)
	}
	attrs := p.Attr[attrMana] + p.Points
	p.resetAttrs()
	if p.Points != attrs || p.Attr[attrMana] != 0 {
		t.Fatalf("esquecimento: pontos %d, mana %d", p.Points, p.Attr[attrMana])
	}
}

// Nível dos inimigos: igual não muda o dano; acima, bate mais e apanha menos;
// a partir da caveira, um golpe comum passa da vida de um personagem novo.
func TestNivelDosInimigos(t *testing.T) {
	if lvScale(5, 5) != 1 || lvScale(8, 5) <= 1 || lvScale(2, 5) >= 1 || lvScale(1, 20) < minDmgFactor {
		t.Fatalf("escala: igual %v, acima %v, abaixo %v, muito abaixo %v", lvScale(5, 5), lvScale(8, 5), lvScale(2, 5), lvScale(1, 20))
	}
	k := mobKinds[mobFrostGiant]
	if hit := float64(k.Dmg) * lvScale(mobLv(mobFrostGiant), 1); hit < float64(classes[1].HP) {
		t.Fatalf("Gigante de Gelo (caveira) contra mago nível 1: golpe de %.0f, vida %d", hit, classes[1].HP)
	}
	for kind := range mobKinds {
		if mobLv(kind) < 1 {
			t.Errorf("%s sem nível", mobKinds[kind].Name)
		}
	}
}

// Necromante: Erguer Mortos cria servos que ferem o inimigo por perto e se
// desfazem quando o tempo acaba; toda classe tem 4 ramos e um golpe final.
func TestServosENovosRamos(t *testing.T) {
	for _, c := range classes {
		last := c.Web[len(c.Web)-1]
		if len(c.Branches) < 4 || last.Ring != 5 || last.Skill != c.techBase-1 {
			t.Errorf("%s: %d ramos, último nó anel %d golpe %d", c.Name, len(c.Branches), last.Ring, last.Skill)
		}
	}
	wd := NewWorld(1)
	id := wd.Join("nyx", 6)
	p := wd.players[id]
	p.Level, p.Attr[attrMana], p.SkillPts = 10, 8, 4
	for _, n := range learnPath(p, 7) {
		wd.SpendTree(id, n)
	}
	if !p.knows(7) {
		t.Fatal("devia aprender Erguer Mortos")
	}
	m := wd.mobs[0]
	m.bornAt, m.HP = time.Time{}, 100000
	p.X, p.Y = int(m.X)-20, int(m.Y)
	p.MP = p.MaxMP
	wd.Use(id, 7, nil)
	if len(wd.minions) != 2 {
		t.Fatalf("esperava 2 servos, há %d", len(wd.minions))
	}
	now := time.Now()
	hp := m.HP
	for i := 0; i < 200 && m.HP == hp; i++ {
		now = now.Add(50 * time.Millisecond)
		wd.stepMinions(now)
	}
	if m.HP == hp {
		t.Fatal("os servos deviam ferir o esqueleto")
	}
	wd.stepMinions(now.Add(time.Minute))
	if len(wd.minions) != 0 {
		t.Fatal("os servos deviam se desfazer quando o tempo acaba")
	}
}

// Depravado: começa com o pedaço de pau na mão, aprende de qualquer escola
// com o atributo mais alto, mas no máximo de três ao mesmo tempo.
func TestDepravado(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("zeca", classDepraved)
	p := wd.players[id]
	if classes[p.Class].Name != "Depravado" || p.Equip != itemStick {
		t.Fatalf("classe %q, arma %d", classes[p.Class].Name, p.Equip)
	}
	c := classes[p.Class]
	if len(c.Branches) != classDepraved {
		t.Fatalf("o Depravado devia ter um ramo por classe: %d", len(c.Branches))
	}
	p.Level, p.Attr[attrSta], p.SkillPts = 10, 5, 20 // Stamina alta vale para qualquer escola
	first := map[int]int{}
	for i, n := range c.Web {
		if n.Ring == 1 {
			first[n.Branch] = i
		}
	}
	for b := 0; b < 4; b++ {
		wd.SpendTree(id, first[b])
	}
	if len(p.schools()) != maxSchools || p.Web[first[3]] != 0 {
		t.Fatalf("devia parar em %d escolas: segue %d, quarta=%d", maxSchools, len(p.schools()), p.Web[first[3]])
	}
}
