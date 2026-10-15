package main

import (
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestMissoesEItens(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ana", 0)
	p := wd.players[id]

	// kill põe o jogador ao lado do inimigo, já de pé, e o derruba com um golpe.
	kill := func(m *Mob) {
		t.Helper()
		m.bornAt, m.HP = time.Time{}, 1
		p.X, p.Y, p.ready = int(m.X)+3, int(m.Y), [maxSkills]time.Time{}
		wd.Attack(id)
		if m.Alive {
			t.Fatalf("%s devia ter caído", mobKinds[m.Kind].Name)
		}
	}
	find := func(kinds ...int) *Mob {
		t.Helper()
		for _, m := range wd.mobs {
			for _, k := range kinds {
				if m.Alive && m.Kind == k {
					return m
				}
			}
		}
		t.Fatalf("nenhum inimigo vivo dos tipos %v", kinds)
		return nil
	}

	// Missão 1: cinco esqueletos rendem ouro e poções e abrem a missão seguinte.
	for i := 0; i < 5; i++ {
		kill(find(mobSkeleton, mobWarrior))
	}
	if p.Quest != 1 || p.Gold < quests[0].Gold || p.count(itemPotion) < quests[0].Potions {
		t.Fatalf("missão 1: quest=%d gold=%d poções=%d", p.Quest, p.Gold, p.count(itemPotion))
	}

	n := p.count(itemPotion)
	p.HP = 10
	wd.UseItem(id, itemPotion)
	if p.HP != 10+items[itemPotion].Heal || p.count(itemPotion) != n-1 {
		t.Fatalf("poção: HP=%d, restam %d de %d", p.HP, p.count(itemPotion), n)
	}

	// Missão 2: descer à cripta. Missão 3: o Lorde, que larga a lâmina dele.
	p.X, p.Y = chapelX, chapelY+2
	wd.enter(p)
	if p.Quest != 2 || p.Level < 2 {
		t.Fatalf("descida: quest=%d level=%d", p.Quest, p.Level)
	}
	for i := 0; i < 3; i++ { // etapa 1: os guerreiros da cripta
		kill(find(mobWarrior))
	}
	lord := find(mobLord)
	kill(lord)
	if p.Quest != 3 {
		t.Fatalf("o Lorde devia fechar a missão 3: quest=%d", p.Quest)
	}
	var blade *Loot
	for i := range wd.loot {
		if wd.loot[i].Item == itemLordBlade && wd.loot[i].Owner == id {
			blade = &wd.loot[i]
		}
	}
	if blade == nil {
		t.Fatal("o Lorde devia largar a lâmina para quem estava perto")
	}
	p.X, p.Y = blade.X, blade.Y
	wd.stepLoot(time.Now())
	if p.count(itemLordBlade) != 1 || p.Equip != itemLordBlade {
		t.Fatalf("lâmina: tem %d, equipada=%d", p.count(itemLordBlade), p.Equip)
	}
	wd.UseItem(id, itemLordBlade)
	if p.Equip != 0 {
		t.Fatal("usar a arma equipada devia desequipar")
	}

	// Quem já tem a arma do chefe não recebe outra.
	wd.loot = nil
	wd.bossDrop(itemLordBlade, float64(p.X), float64(p.Y), time.Now())
	for _, l := range wd.loot {
		if l.Item == itemLordBlade {
			t.Fatal("arma repetida no saque")
		}
	}

	// Todo chefe nasce em chão livre, e o dia gira com o relógio do mundo.
	for _, m := range wd.mobs {
		if mobKinds[m.Kind].Boss && !wd.fits(int(m.X), int(m.Y)) {
			t.Errorf("%s nasceu preso", mobKinds[m.Kind].Name)
		}
	}
	wd.dayLen = 100
	if a, b := wd.timeOfDay(0), wd.timeOfDay(50); a != 0.35 || math.Abs(b-0.85) > 1e-9 {
		t.Fatalf("hora do dia: %v e %v", a, b)
	}
}

func TestProgresso(t *testing.T) {
	wd := NewWorld(1)
	wd.saveDir = t.TempDir()
	id, err := wd.JoinSaved("lia", 6, "segredo", "")
	if err != nil {
		t.Fatal(err)
	}
	p := wd.players[id]
	if classes[p.Class].Name != "Necromante" || p.MaxST != baseST {
		t.Fatalf("classe %q, stamina %d", classes[p.Class].Name, p.MaxST)
	}

	// Experiência sobe de nível e rende pontos; os pontos viram atributos.
	wd.gainXP(p, xpNeed(1)+xpNeed(2))
	if p.Level != 3 || p.Points != 2*pointsPerLevel || p.SkillPts != 2 {
		t.Fatalf("nível %d, %d pontos, %d de árvore", p.Level, p.Points, p.SkillPts)
	}
	hp, base := p.MaxHP, p.dmgPct(0)
	wd.SpendAttr(id, attrVit)
	wd.SpendAttr(id, attrStr)
	wd.SpendAttr(id, attrSta)
	if p.MaxHP != hp+vitHP || p.dmgPct(0) != base+strDmg || p.MaxST != baseST+staST || p.Points != 3 {
		t.Fatalf("atributos: HP %d→%d, dano %d→%d, stamina %d, pontos %d", hp, p.MaxHP, base, p.dmgPct(0), p.MaxST, p.Points)
	}

	// Teia: o primeiro anel pede 1 ponto no atributo da classe (Mana, no
	// necromante); o segundo anel só abre ligado a um nó aprendido.
	wd.SpendTree(id, 1)
	if p.Web[1] != 0 {
		t.Fatal("sem Mana o nó do primeiro anel não devia abrir")
	}
	wd.SpendAttr(id, attrMana)
	wd.SpendTree(id, 2)
	if p.Web[2] != 0 {
		t.Fatal("nó sem ligação a um aprendido não devia aceitar ponto")
	}
	wd.SpendTree(id, 1)
	if p.Web[1] != 1 || p.dmgPct(0) != base+strDmg+8 || p.dmgPct(1) != base+strDmg {
		t.Fatalf("Golpe Afiado: grau %d, básico %d%%, classe %d%%", p.Web[1], p.dmgPct(0), p.dmgPct(1))
	}

	// Skin: protege e, em classe mágica, soma dano.
	p.give(itemRadiantAegis, 1)
	if p.Skin != itemRadiantAegis || p.defPct() != items[itemRadiantAegis].Def || p.dmgPct(1) != base+strDmg+items[itemRadiantAegis].Magic {
		t.Fatalf("skin: %d, proteção %d%%, dano %d%%", p.Skin, p.defPct(), p.dmgPct(1))
	}
	p.HP = p.MaxHP
	wd.hurt(p, 100, "no teste", time.Now())
	if lost := p.MaxHP - p.HP; lost != 100-items[itemRadiantAegis].Def {
		t.Fatalf("com a égide devia perder %d, perdeu %d", 100-items[itemRadiantAegis].Def, lost)
	}

	// Correr gasta stamina e anda mais.
	x0, st := p.X, p.ST
	wd.SetInput(id, 1, 0, true)
	wd.step(time.Now())
	if p.ST >= st || p.X-x0 != walkSpeed+1 {
		t.Fatalf("corrida: andou %d, stamina %v→%v", p.X-x0, st, p.ST)
	}
	wd.SetInput(id, 0, 0, false)

	// Baú: abre uma vez por dia do jogo e conta para a conquista.
	c := chests[0]
	p.X, p.Y = c.X, c.Y
	gold := p.Gold
	wd.openChests(p, time.Now())
	wd.openChests(p, time.Now())
	if p.Chests != 1 || p.Gold <= gold {
		t.Fatalf("baú: abriu %d vez(es), ouro %d→%d", p.Chests, gold, p.Gold)
	}
	for i, c := range chests {
		if !wd.fits(c.X, c.Y) {
			t.Errorf("baú %d fica em lugar inacessível", i)
		}
	}
	wd.achieve(p, time.Now())
	if p.Ach&(1<<5) == 0 {
		t.Fatal("ter a égide devia dar a conquista Lendário")
	}

	// Salvar e voltar: mesmo nome e segredo recuperam tudo; outro segredo é recusado.
	lvl, ach, kept := p.Level, p.Ach, p.Gold
	wd.Leave(id)
	if _, err := wd.JoinSaved("Lia", 0, "outro", ""); err == nil {
		t.Fatal("segredo errado devia ser recusado")
	}
	id, err = wd.JoinSaved("Lia", 0, "segredo", "")
	if err != nil {
		t.Fatal(err)
	}
	p = wd.players[id]
	if p.Level != lvl || p.Ach != ach || p.Gold != kept || p.Class != 6 || p.Skin != itemRadiantAegis ||
		p.Attr[attrVit] != 1 || p.Web[1] != 1 || p.count(itemRadiantAegis) != 1 {
		t.Fatalf("save não voltou igual: %+v", *p)
	}
	if _, err := wd.JoinSaved("lia", 0, "segredo", ""); err == nil {
		t.Fatal("personagem já conectado devia ser recusado")
	}
}

func TestAfinidadeEEstacoes(t *testing.T) {
	// A mesma espada rende mais na mão do guerreiro que na do mago.
	g, m := &Player{Class: 0, Equip: itemLordBlade}, &Player{Class: 1, Equip: itemLordBlade}
	bonus := items[itemLordBlade].Bonus
	if g.weaponBonus() != bonus*affinePct/100 || m.weaponBonus() != bonus*clumsyPct/100 || (&Player{}).weaponBonus() != 0 {
		t.Fatalf("afinidade: guerreiro %d, mago %d", g.weaponBonus(), m.weaponBonus())
	}
	// Toda arma tem tipo e toda classe tem ao menos uma afinidade.
	for i, it := range items {
		if _, ok := weaponType[i]; ok != (it.Kind == kindWeapon) {
			t.Errorf("%s: tipo de arma inconsistente", it.Name)
		}
	}
	if len(affinity) != len(classes) {
		t.Fatalf("%d afinidades para %d classes", len(affinity), len(classes))
	}
	// As estações giram de seasonDays em seasonDays e voltam à primavera.
	for day, want := range map[int]int{1: 0, seasonDays: 0, seasonDays + 1: 1, 3*seasonDays + 1: 3, 4*seasonDays + 1: 0} {
		if got := seasonOf(day); got != want {
			t.Errorf("dia %d: estação %d, esperava %d", day, got, want)
		}
	}
	// Cada região tem os seus inimigos: nenhum comum nasce na região de outro.
	wd := NewWorld(1)
	home := map[int]int{mobAsura: 4, mobTriton: 6, mobSentinel: 5}
	for _, m := range wd.mobs {
		if want, ok := home[m.Kind]; ok {
			if r := regionAt(int(m.X), int(m.Y)); r != want && !(m.Kind == mobAsura && r == 0) {
				t.Errorf("%s nasceu na região %d", mobKinds[m.Kind].Name, r)
			}
		}
	}
}

func TestEquipamento(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("eva", 1) // maga: o bônus mágico das peças vale para ela
	p := wd.players[id]
	base := p.dmgPct(1)

	// Cada peça vai para o seu espaço e a primeira já entra equipada.
	for _, it := range []int{itemIronHelm, itemLeatherBoots, itemCryptAmulet, itemTravelCloak, itemSolarRelic} {
		p.give(it, 1)
	}
	if p.Helm != itemIronHelm || p.Boots != itemLeatherBoots || p.Amulet != itemCryptAmulet || p.Skin != itemTravelCloak || p.Equip != itemSolarRelic {
		t.Fatalf("espaços: %+v", []int{p.Equip, p.Helm, p.Skin, p.Boots, p.Amulet})
	}
	wantDef := items[itemIronHelm].Def + items[itemLeatherBoots].Def + items[itemTravelCloak].Def
	amulet := items[itemCryptAmulet]
	if p.defPct() != wantDef || p.dmgPct(1) != base+p.weaponBonus()+amulet.Bonus+amulet.Magic {
		t.Fatalf("proteção %d (esperava %d), dano %d", p.defPct(), wantDef, p.dmgPct(1))
	}

	// Uma segunda peça do mesmo espaço fica na mochila até ser escolhida;
	// usar a que está equipada tira.
	p.give(itemAsuraCrown, 1)
	if p.Helm != itemIronHelm {
		t.Fatal("peça nova não devia trocar a equipada sozinha")
	}
	wd.UseItem(id, itemAsuraCrown)
	if p.Helm != itemAsuraCrown {
		t.Fatal("usar a coroa devia equipá-la")
	}
	wd.UseItem(id, itemAsuraCrown)
	if p.Helm != 0 || p.count(itemAsuraCrown) != 1 {
		t.Fatal("usar a peça equipada devia só tirá-la")
	}
}

func TestHistoria(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("rui", 0)
	p := wd.players[id]
	now := time.Now()

	// Região selada não deixa entrar quem está fora dela.
	sel := regions[4] // Selva de Marapá, selada pela "Trilha dos lobos"
	if p.sealedBy(4) != 0 {
		t.Fatalf("a selva devia estar selada pela secundária 0, está por %d", p.sealedBy(4))
	}
	edge := float64(sel.R) + 1
	if wd.canEnter(p, sel.X, sel.Y, now) || !wd.canEnter(p, sel.X+int(edge), sel.Y+int(edge), now) {
		t.Fatal("selo: devia barrar o centro da selva e liberar fora dela")
	}
	// A secundária só conta a partir do capítulo dela; concluída, quebra o selo.
	p.Quest = 3 // capítulo 2
	for _, m := range wd.mobs {
		if m.Kind == mobWolf && p.SideDone&1 == 0 {
			wd.credit(mobWolf, m.X, m.Y, now)
			p.X, p.Y = int(m.X), int(m.Y)
			wd.credit(mobWolf, m.X, m.Y, now)
		}
	}
	for i := 0; i < 5 && p.SideDone&1 == 0; i++ {
		wd.credit(mobWolf, float64(p.X), float64(p.Y), now)
	}
	if p.SideDone&1 == 0 || p.sealedBy(4) >= 0 || !wd.canEnter(p, sel.X, sel.Y, now) {
		t.Fatalf("Trilha dos lobos devia abrir a selva: done=%b prog=%v", p.SideDone, p.SideProg)
	}

	// Mercador: compra com dinheiro, recusa sem.
	shop := wd.shops[0]
	p.X, p.Y, p.Gold = shop.X, shop.Y+4, 10
	it, price := shop.Stock[0][0], shop.Stock[0][1]
	wd.Buy(id, 0, it)
	if p.count(it) != 0 || p.Gold != 10 {
		t.Fatal("sem dinheiro não devia comprar")
	}
	p.Gold = price + 5
	wd.Buy(id, 0, it)
	if p.count(it) != 1 || p.Gold != 5 {
		t.Fatalf("compra: tem %d, sobrou %d", p.count(it), p.Gold)
	}

	// Fogueira: acende ao encostar; morrer leva até ela e deixa o dinheiro no chão.
	b := wd.bonfires[2]
	p.X, p.Y = b.X, b.Y+4
	wd.tendFire(p, now)
	if p.Bonfire != 2 {
		t.Fatalf("fogueira acesa: %d", p.Bonfire)
	}
	p.X, p.Y, p.Gold = villageX, villageY, 77
	wd.hurtRaw(p, p.HP+10, "no teste", now)
	if dist(float64(p.X), float64(p.Y), float64(b.X), float64(b.Y)) > 30 || p.Gold != 0 {
		t.Fatalf("renasceu em (%d,%d), longe da fogueira (%d,%d); ouro %d", p.X, p.Y, b.X, b.Y, p.Gold)
	}
	found := false
	for _, l := range wd.loot {
		found = found || l.Item == itemCoins && l.N == 77 && l.Owner == id
	}
	if !found {
		t.Fatal("o dinheiro devia ficar caído onde ele morreu, só para ele")
	}

	// Tiamath mora no alto das montanhas, num lugar alcançável.
	if mountDist(lairX, lairY) > 0.01 || regions[3].Name != "Pico de Tiamath" {
		t.Fatal("Tiamath devia estar no pico das montanhas")
	}
	if len(quests) != 14 || quests[len(quests)-1].Chapter != 6 || len(chapters) != 6 {
		t.Fatal("a história devia ter 6 capítulos e terminar em Tiamath")
	}
	for i, q := range sideQuests {
		for _, k := range q.Targets {
			n := 0
			for _, m := range wd.mobs {
				if m.Kind == k {
					n++
				}
			}
			if n == 0 {
				t.Errorf("secundária %d pede inimigo que não existe", i)
			}
		}
	}
}

func TestPersonagens(t *testing.T) {
	wd := NewWorld(1)
	wd.saveDir = t.TempDir()

	// Criado num aparelho, aparece na lista dele e não na de outro.
	id, err := wd.JoinSaved("Teo", 2, "pc-1", "")
	if err != nil {
		t.Fatal(err)
	}
	wd.Leave(id)
	if l := wd.Chars("pc-1"); len(l) != 1 || l[0].Name != "teo" || l[0].Class != 2 {
		t.Fatalf("lista do pc-1: %+v", l)
	}
	if l := wd.Chars("pc-2"); len(l) != 0 {
		t.Fatalf("lista do pc-2 devia vir vazia: %+v", l)
	}
	// Outro aparelho sem senha é recusado; o dono define uma senha ao entrar.
	if _, err := wd.JoinSaved("Teo", 0, "pc-2", ""); err == nil {
		t.Fatal("outro aparelho sem senha devia ser recusado")
	}
	id, _ = wd.JoinSaved("Teo", 0, "pc-1", "segredo")
	wd.Leave(id)
	if _, err := wd.JoinSaved("Teo", 0, "pc-2", "errada"); err == nil {
		t.Fatal("senha errada devia ser recusada")
	}
	id, err = wd.JoinSaved("Teo", 0, "pc-2", "segredo")
	if err != nil || wd.players[id].Class != 2 {
		t.Fatalf("com a senha devia abrir o mesmo personagem: %v", err)
	}
	wd.Leave(id)
	// Agora o pc-2 também vê o personagem, e consegue excluir.
	if l := wd.Chars("pc-2"); len(l) != 1 || !l[0].Password {
		t.Fatalf("lista do pc-2 depois da senha: %+v", l)
	}
	if err := wd.DeleteChar("pc-3", "teo"); err == nil {
		t.Fatal("aparelho estranho não devia excluir")
	}
	if err := wd.DeleteChar("pc-2", "teo"); err != nil {
		t.Fatal(err)
	}
	if l := wd.Chars("pc-1"); len(l) != 0 {
		t.Fatal("excluído não devia aparecer")
	}
	if trash, _ := filepath.Glob(filepath.Join(wd.saveDir, "lixeira", "teo-*.json")); len(trash) != 1 {
		t.Fatal("o save excluído devia ir para a lixeira")
	}
	// O nome fica livre de novo.
	if _, err := wd.JoinSaved("Teo", 1, "pc-9", ""); err != nil {
		t.Fatal(err)
	}
}

// Quem derruba um chefe antes de a missão dele chegar não fica travado
// esperando o chefe renascer: a missão fecha sozinha quando chega a vez dela.
func TestChefeAntesDaHora(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("rui", 0)
	p := wd.players[id]
	queenQ := -1
	for i, q := range quests {
		if has(q.Targets, mobQueen) {
			queenQ = i
		}
	}
	p.Quest = queenQ - 1 // ainda caçando as asuras
	var queen *Mob
	for _, m := range wd.mobs {
		if m.Kind == mobQueen {
			queen = m
		}
	}
	p.X, p.Y = int(queen.X), int(queen.Y)+6
	now := time.Now()
	wd.mobDied(queen, p, now)
	if p.Quest != queenQ-1 || !p.slew(mobQueen) {
		t.Fatalf("matar a Rainha cedo não muda a missão atual, mas fica lembrado: missão %d, lembra %v", p.Quest, p.slew(mobQueen))
	}
	for q := p.Quest; p.Quest == q; { // termina as etapas das asuras
		wd.advance(p, 99, now)
	}
	wd.catchUp(p, now)
	if p.Quest != queenQ+1 {
		t.Fatalf("a missão da Rainha devia fechar sozinha: está na %d, esperava %d", p.Quest, queenQ+1)
	}
}

// Cofre: a porta bloqueia; sem chave não abre; com a Chave de Ferro abre,
// gasta a chave, e o baú lá dentro fica ao alcance. Depois volta a trancar.
func TestCofreEChave(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("leo", 0)
	p := wd.players[id]
	cx, by := vaults[1][0], vaults[1][1]
	if wd.Walkable(cx, by) {
		t.Fatal("porta fechada devia bloquear")
	}
	p.X, p.Y = cx, by+5
	wd.Interact(id)
	if wd.Walkable(cx, by) {
		t.Fatal("sem chave a porta não abre")
	}
	p.give(itemIronKey, 1)
	wd.Interact(id)
	if !wd.Walkable(cx, by) || p.count(itemIronKey) != 0 {
		t.Fatalf("com chave devia abrir e gastar a chave: anda=%v chaves=%d", wd.Walkable(cx, by), p.count(itemIronKey))
	}
	// Vender e jogar fora.
	p.give(itemPotion, 2)
	gold := p.Gold
	m := wd.shops[1]
	p.X, p.Y = m.X, m.Y+4
	wd.Sell(id, 1, itemPotion)
	if p.Gold <= gold || p.count(itemPotion) != 1 {
		t.Fatalf("venda: ouro %d→%d, poções %d", gold, p.Gold, p.count(itemPotion))
	}
	n := len(wd.loot)
	wd.Drop(id, itemPotion)
	if p.count(itemPotion) != 0 || len(wd.loot) != n+1 {
		t.Fatal("jogar fora devia tirar da mochila e pôr no chão")
	}
	wd.stepLoot(time.Now())
	if p.count(itemPotion) != 0 {
		t.Fatal("quem largou não devia recolher de volta na hora")
	}
}

// Mochila: começa com 50; as ampliações dos mercadores escondidos somam
// espaços na hora, sem ocupar a mochila, até o teto.
func TestMochila(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("bia", 0)
	p := wd.players[id]
	if p.bagCap() != baseBag {
		t.Fatalf("mochila inicial %d", p.bagCap())
	}
	for i, m := range wd.shops {
		if !wd.Walkable(m.X, m.Y) {
			t.Errorf("mercador %d (%s) em lugar inacessível", i, m.Name)
		}
	}
	hermit := -1
	for i, m := range wd.shops {
		if m.Name == "O Eremita da Ilha" {
			hermit = i
		}
	}
	m := wd.shops[hermit]
	p.X, p.Y, p.Gold = m.X, m.Y+3, 10000
	n := len(p.inv)
	wd.Buy(id, hermit, itemBagSmall)
	if p.bagCap() != baseBag+10 || len(p.inv) != n {
		t.Fatalf("depois da bolsa: %d espaços, %d itens", p.bagCap(), len(p.inv))
	}
	for i := 0; i < 20; i++ {
		wd.Buy(id, hermit, itemBagSmall)
	}
	if p.bagCap() != maxBag {
		t.Fatalf("teto da mochila: %d", p.bagCap())
	}
}

// Missões em etapas e viagem rápida entre fogueiras e placas descobertas.
func TestEtapasEViagem(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("eli", 0)
	p := wd.players[id]
	if s, ok := p.curStep(); !ok || s.Kind != qVisit {
		t.Fatalf("primeira etapa devia ser chegar ao cemitério: %+v", s)
	}
	g := regions[2]
	p.X, p.Y = g.X, g.Y
	wd.questEvent(p, qVisit, time.Now())
	if p.QStep != 1 || p.Quest != 0 {
		t.Fatalf("chegar ao cemitério devia abrir a etapa 2: missão %d etapa %d", p.Quest, p.QStep)
	}
	if len(wd.travel) < len(wd.bonfires)+4 {
		t.Fatalf("pontos de viagem: %d (fogueiras %d)", len(wd.travel), len(wd.bonfires))
	}
	a, b := wd.travel[0], wd.travel[len(wd.travel)-1]
	p.X, p.Y = a.X, a.Y
	wd.Travel(id, len(wd.travel)-1)
	if p.X == b.X && p.Y == b.Y {
		t.Fatal("não devia viajar para ponto não descoberto")
	}
	wd.discover(p, time.Now())
	p.Found |= 1 << (len(wd.travel) - 1)
	wd.Travel(id, len(wd.travel)-1)
	if dist(float64(p.X), float64(p.Y), float64(b.X), float64(b.Y)) > 25 {
		t.Fatalf("devia ter viajado para %s: está em (%d,%d)", b.Name, p.X, p.Y)
	}
}

// Ferramentas: a tocha não acende na chuva e a lanterna sim; o machado
// derruba árvore bem mais rápido; quem tem espada atrai raio na tempestade.
func TestFerramentas(t *testing.T) {
	wd := NewWorld(1)
	id := wd.Join("ana", 0)
	p := wd.players[id]
	p.give(itemTorch, 1)
	p.give(itemLantern, 1)
	wd.weather = wxRain
	wd.UseItem(id, itemTorch)
	if p.Light != lightNone {
		t.Fatal("tocha não devia acender na chuva")
	}
	wd.UseItem(id, itemLantern)
	if p.Light != lightLantern {
		t.Fatal("lanterna devia acender na chuva")
	}
	wd.weather = wxClear
	wd.UseItem(id, itemTorch)
	if p.Light != lightTorch || p.count(itemTorch) != 0 {
		t.Fatalf("tocha acesa: luz %d, tochas %d", p.Light, p.count(itemTorch))
	}
	wd.weather = wxRain
	wd.stepLight(p, time.Now())
	if p.Light != lightNone {
		t.Fatal("a chuva devia apagar a tocha")
	}
	if p.chopMult() != 1 {
		t.Fatal("sem machado o golpe é normal")
	}
	p.give(itemAxe, 1)
	if p.chopMult() != 4 {
		t.Fatal("com machado devia derrubar 4x mais rápido")
	}
	p.Equip = itemShortSword
	if !p.metallic() {
		t.Fatal("espada devia contar como metal")
	}
	p.Equip = itemStick
	if p.metallic() {
		t.Fatal("pedaço de pau não é metal")
	}
}

// Prólogo: personagem novo acorda no vale; falar com o Eremita, derrubar 3
// bonecos, descansar na fogueira e chegar à vila fecha o prólogo; o save
// guarda a etapa, e save antigo (sem prólogo) já começa depois dele.
func TestPrologo(t *testing.T) {
	wd := NewWorld(1)
	wd.saveDir = t.TempDir()
	id, err := wd.JoinSaved("novato", 0, "segredo", "")
	if err != nil {
		t.Fatal(err)
	}
	p := wd.players[id]
	if !p.inPrologue() || regionAt(p.X, p.Y) < 0 || regions[regionAt(p.X, p.Y)].Name != "Vale do Despertar" {
		t.Fatalf("devia acordar no vale no prólogo: intro %d em (%d,%d)", p.Intro, p.X, p.Y)
	}
	now := time.Now()
	wd.Talk(id, 0)
	if p.Intro != 1 {
		t.Fatalf("falar com o Eremita devia abrir a etapa 2: %d", p.Intro)
	}
	n := 0
	for _, m := range wd.mobs {
		if m.Kind == mobDummy && n < 3 {
			m.bornAt = time.Time{}
			p.X, p.Y = int(m.X)+3, int(m.Y)
			wd.mobDied(m, p, now)
			n++
		}
	}
	if n < 3 || p.Intro != 2 {
		t.Fatalf("3 bonecos (%d achados) deviam abrir a etapa 3: %d", n, p.Intro)
	}
	wd.rest(p, len(wd.bonfires)-1, now)
	if p.Intro != 3 {
		t.Fatalf("descansar devia abrir a etapa 4: %d", p.Intro)
	}
	wd.persist(p)
	wd.Leave(id)
	id, _ = wd.JoinSaved("novato", 0, "segredo", "")
	p = wd.players[id]
	if p.Intro != 3 || regions[regionAt(p.X, p.Y)].Name != "Vale do Despertar" {
		t.Fatalf("quem volta no meio do prólogo devia reaparecer no vale: intro %d em (%d,%d)", p.Intro, p.X, p.Y)
	}
	v := regions[1]
	p.X, p.Y = v.X, v.Y
	wd.questEvent(p, qVisit, now)
	if p.inPrologue() || p.Bonfire != 0 || p.Quest != 0 {
		t.Fatalf("chegar à vila devia fechar o prólogo: intro %d fogueira %d missão %d", p.Intro, p.Bonfire, p.Quest)
	}
	// O save guardou a etapa 3 (antes de chegar à vila).
	wd.Leave(id)
	id, _ = wd.JoinSaved("novato", 0, "segredo", "")
	if wd.players[id].Intro != len(prologue) {
		t.Fatalf("o save devia voltar com o prólogo concluído: %d", wd.players[id].Intro)
	}
}

// TestMestreDasSequencias: a técnica só vem com a sequência certa, nível e
// dinheiro; aprendida, entra no livro e na barra, funciona e volta no save.
func TestMestreDasSequencias(t *testing.T) {
	wd := NewWorld(1)
	wd.saveDir = t.TempDir()
	id, err := wd.JoinSaved("aluno", 2, "segredo", "")
	if err != nil {
		t.Fatal(err)
	}
	p := wd.players[id]
	if !wd.fits(masterX, masterY) {
		t.Fatalf("o terreiro do mestre devia ser chão livre")
	}
	p.X, p.Y = masterX+4, masterY
	p.Level, p.Gold = 1, 1000
	wd.Learn(id, 0, techs[0].Seq)
	if p.knowsTech(0) {
		t.Fatal("aprendeu abaixo do nível")
	}
	p.Level = 20
	wd.Learn(id, 0, "UUU")
	if p.knowsTech(0) || p.Gold != 1000 {
		t.Fatal("sequência errada não devia ensinar nem cobrar")
	}
	wd.Learn(id, 2, techs[2].Seq)
	s := classes[p.Class].techBase + 2
	if !p.knowsTech(2) || p.Gold != 1000-techs[2].Price || !p.knows(s) {
		t.Fatalf("devia aprender o Redemoinho: tech %b ouro %d", p.Tech, p.Gold)
	}
	onBar := false
	for _, b := range p.Bar {
		onBar = onBar || b == s
	}
	if !onBar {
		t.Fatalf("a técnica devia ir para a barra: %v", p.Bar)
	}
	p.X, p.Y = masterX+60, masterY
	p.MP = p.MaxMP
	wd.Use(id, s, nil)
	if p.ready[s].IsZero() {
		t.Fatal("a técnica devia poder ser usada")
	}
	wd.persist(p)
	wd.Leave(id)
	id, _ = wd.JoinSaved("aluno", 2, "segredo", "")
	if !wd.players[id].knowsTech(2) {
		t.Fatal("o save devia guardar a técnica")
	}
	for _, c := range classes {
		if len(c.Skills) > maxSkills {
			t.Fatalf("%s tem %d golpes (máximo %d)", c.Name, len(c.Skills), maxSkills)
		}
	}
}

// TestBarraPorNivel: a barra começa com 5 atalhos e abre até 9 com o nível;
// um save com a barra antiga (5) volta com o resto vazio.
func TestBarraPorNivel(t *testing.T) {
	wd := NewWorld(1)
	wd.saveDir = t.TempDir()
	id, _ := wd.JoinSaved("barra", 0, "segredo", "")
	p := wd.players[id]
	p.Level = 1
	if p.barCap() != 5 {
		t.Fatalf("nível 1 devia ter 5 atalhos: %d", p.barCap())
	}
	wd.SetBar(id, 6, 0)
	if p.Bar[6] != -1 {
		t.Fatal("atalho 7 ainda trancado não devia aceitar golpe")
	}
	p.Level = 18
	wd.SetBar(id, 8, 0)
	if p.barCap() != 9 || p.Bar[8] != 0 {
		t.Fatalf("nível 18 devia abrir os 9: %d, %v", p.barCap(), p.Bar)
	}
	var sv saveData
	sv.Bar, sv.Web, sv.SkillVer = []int{1, 0, -1, -1, -1}, []int{1}, 1
	q := wd.spawn("velho", 0)
	q.apply(sv)
	if q.Bar[0] != 1 || q.Bar[1] != 0 || q.Bar[5] != -1 || q.Bar[8] != -1 {
		t.Fatalf("barra antiga devia voltar com o resto vazio: %v", q.Bar)
	}
}
