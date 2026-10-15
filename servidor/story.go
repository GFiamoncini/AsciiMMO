package main

import (
	"fmt"
	"strings"
	"time"
)

// A história: seis capítulos, um por bioma, até Tiamath no alto das
// montanhas. As missões principais formam uma corrente; as secundárias de
// cada capítulo quebram o selo que fecha a região seguinte.

// currency é o nome do dinheiro do jogo (pedido do usuário). Trocar aqui muda
// o nome em todo lugar, inclusive nos clientes, que recebem no init.
const currency = "bosta"

type chapter struct{ Title, Intro string }

var chapters = []chapter{
	{"Capítulo I — Cinzas sobre Cinzalva", "Os mortos do cemitério não descansam. Algo os chama de debaixo da capela."},
	{"Capítulo II — A Selva que Devora", "Na mata de Marapá, as asuras cantam para um deus que não tem nome."},
	{"Capítulo III — Marés de Sangue", "O mar de Talássia ficou vermelho. Os tritões rezam para o fundo."},
	{"Capítulo IV — Areias sem Nome", "Sob a pirâmide de Ankar, um faraó sonha acordado há mil anos."},
	{"Capítulo V — O Inverno Eterno", "Nos picos, o gelo guarda os ossos dos que subiram antes de você."},
	{"Capítulo VI — Tiamath", "No alto das montanhas, a mãe dos dragões abre um olho. O mundo inteiro é o ninho dela."},
}

// Missões: matar inimigos, descer à cripta ou chegar a uma região.
const (
	qKill  = iota // derrotar Need inimigos de um dos tipos em Targets
	qEnter        // descer à cripta
	qVisit        // chegar à região Targets[0]
)

// killDragon é o "tipo" de Tiamath nas missões de caça.
const killDragon = -1

type quest struct {
	Chapter     int // 1 a 6; a secundária fica disponível a partir dele
	Title, Text string
	Kind        int
	Targets     []int
	Need        int
	Gold        int
	Potions     int
	Level       bool  // sobe um nível ao concluir
	Unlock      []int // secundária: regiões cujo selo ela quebra
}

var quests = []quest{
	{1, "Ossos inquietos", "Os mortos se levantam no cemitério, ao norte de Cinzalva. Devolva-os à terra.", qKill, []int{mobSkeleton, mobWarrior}, 5, 40, 2, false, nil},
	{1, "A descida", "Desça à cripta pela porta da capela. Algo respira lá embaixo.", qEnter, nil, 1, 60, 1, true, nil},
	{1, "O Lorde da Cripta", "Destrua o Lorde que se ergue na última sala da cripta.", qKill, []int{mobLord}, 1, 200, 2, true, nil},
	{2, "Rumo à selva", "Siga a trilha a leste da Encruzilhada até a Selva de Marapá.", qVisit, []int{4}, 1, 60, 0, false, nil},
	{2, "Lâminas na mata", "Silencie as asuras que cantam em volta da cidadela.", qKill, []int{mobAsura}, 6, 150, 2, false, nil},
	{2, "A Rainha Asura", "Derrube a Rainha no pátio da cidadela antes que o canto termine.", qKill, []int{mobQueen}, 1, 300, 2, true, nil},
	{3, "Marés de sangue", "Os tritões da costa sudeste afogam quem chega perto. Afogue-os primeiro.", qKill, []int{mobTriton}, 6, 180, 2, false, nil},
	{3, "O Tirano das Marés", "Enfrente o Tirano no coração de Talássia.", qKill, []int{mobTyrant}, 1, 350, 2, true, nil},
	{4, "Areias sem nome", "Escaravelhos e múmias guardam o deserto ao norte. Abra caminho.", qKill, []int{mobScarab, mobMummy}, 8, 240, 2, false, nil},
	{4, "O Faraó", "Acorde o Faraó diante da pirâmide — e faça-o dormir de novo.", qKill, []int{mobPharaoh}, 1, 380, 2, true, nil},
	{5, "Presas no gelo", "Lobos do gelo e yetis rondam os picos a sudoeste.", qKill, []int{mobIceWolf, mobYeti}, 8, 260, 2, false, nil},
	{5, "O Gigante de Gelo", "Derrube o Gigante entre os blocos de gelo.", qKill, []int{mobFrostGiant}, 1, 420, 2, true, nil},
	{6, "A subida", "Suba as montanhas a noroeste até o Pico de Tiamath.", qVisit, []int{3}, 1, 200, 2, false, nil},
	{6, "Tiamath", "Mate Tiamath, a mãe dos dragões, no topo do mundo.", qKill, []int{killDragon}, 1, 1500, 3, true, nil},
}

// sideQuests quebram os selos das regiões. Os índices das regiões são os de
// `regions` (terrain.go): 4 Selva, 0 Cidadela, 6 Talássia, 7 Deserto,
// 8 Picos de Gelo, 3 Pico de Tiamath.
var sideQuests = []quest{
	{2, "Trilha dos lobos", "Lobos famintos fecham a trilha da selva. Cace 5 deles a leste de Cinzalva.", qKill, []int{mobWolf}, 5, 100, 1, false, []int{4, 0}},
	{3, "O charco envenenado", "Limpe o Pântano de Venor: derrote 5 aranhas ou trolls. Só então a costa se abre.", qKill, []int{mobSpider, mobTroll}, 5, 140, 2, false, []int{6}},
	{4, "Sangue verde", "O Chefe Orc guarda a passagem do deserto. Derrube-o no acampamento.", qKill, []int{mobOrcChief}, 1, 220, 2, false, []int{7}},
	{5, "A luz que cega", "O Arconte do Bastião Radiante sustenta o selo dos picos. Apague a luz dele.", qKill, []int{mobArchon}, 1, 300, 2, false, []int{8}},
	{6, "Ecos das profundezas", "Nas cavernas, algo canta com a voz de Tiamath. Cale 6 morcegos ou ratos.", qKill, []int{mobBat, mobRat}, 6, 250, 2, false, []int{3}},
	// Profundezas: levam o jogador lá para baixo (sem selo).
	{4, "O que rasteja embaixo", "Desça por uma das fendas do continente e cace 6 Rastejantes Cegos nas Profundezas.", qKill, []int{mobCrawler}, 6, 300, 2, false, nil},
	{5, "A forja apagada", "Sob a Encruzilhada ficam os Salões de Dorrim. Os guardiões de pedra ainda vigiam a forja: quebre 5.", qKill, []int{mobDwarfGuard}, 5, 420, 2, false, nil},
	{6, "O coração que arde", "Sob o Pico de Tiamath, Ignar aquece a pedra até ela correr. Apague o Coração de Magma.", qKill, []int{mobIgnar}, 1, 700, 3, true, nil},
	{6, "O Flagelo das Profundezas", "Os anões de Dorrim cavaram fundo demais e acordaram Vulkhar. Atravesse a ponte do Abismo, ao sul dos salões, e devolva-o ao fogo.", qKill, []int{mobVulkhar}, 1, 1500, 3, true, nil},
}

const numSide = 9

// chapterOf é o capítulo em que o jogador está (7 depois do fim).
func (p *Player) chapterOf() int {
	if p.Quest >= len(quests) {
		return len(chapters) + 1
	}
	return quests[p.Quest].Chapter
}

// sealedBy devolve a secundária que ainda sela a região, ou -1.
func (p *Player) sealedBy(region int) int {
	for i, q := range sideQuests {
		for _, r := range q.Unlock {
			if r == region && p.SideDone&(1<<i) == 0 {
				return i
			}
		}
	}
	return -1
}

// canEnter diz se o jogador pode pisar em (x, y): uma região selada não deixa
// entrar quem está fora dela (quem já está dentro pode sair).
func (wd *World) canEnter(p *Player, x, y int, now time.Time) bool {
	if y >= overH {
		return true
	}
	for i, r := range regions {
		inside := dist(float64(x), float64(y), float64(r.X), float64(r.Y)) <= float64(r.R)
		here := dist(float64(p.X), float64(p.Y), float64(r.X), float64(r.Y)) <= float64(r.R)
		if !inside || here {
			continue
		}
		if s := p.sealedBy(i); s >= 0 {
			if now.Sub(p.sealAt) > 3*time.Second {
				p.sealAt = now
				wd.pop(p, fmt.Sprintf("Um selo antigo fecha %s — missão: %s", r.Name, sideQuests[s].Title), 0xc080ff, float64(p.X), float64(p.Y)-34, now)
			}
			return false
		}
	}
	return true
}

func (wd *World) reward(p *Player, q quest, now time.Time) {
	p.Gold += q.Gold
	if q.Potions > 0 {
		p.give(itemPotion, q.Potions)
	}
	if q.Level {
		wd.levelUp(p)
	}
	wd.pop(p, fmt.Sprintf("Missão concluída: %s  +%d %s", q.Title, q.Gold, currency), 0xffe066, float64(p.X), float64(p.Y)-40, now)
	wd.effects = append(wd.effects, Effect{Kind: fxHoly, X: float64(p.X), Y: float64(p.Y), R: 14, Start: now, Dur: 600 * time.Millisecond})
}

// creditSide soma um abate à secundária i e a conclui se fechou.
func (wd *World) creditSide(p *Player, i int, now time.Time) {
	q := sideQuests[i]
	if p.SideProg[i]++; p.SideProg[i] >= q.Need {
		p.SideDone |= 1 << i
		wd.reward(p, q, now)
		var names []string
		for _, r := range q.Unlock {
			names = append(names, regions[r].Name)
		}
		if len(names) > 0 { // as das Profundezas não quebram selo
			wd.say(ChatLine{Text: p.Name + " quebrou o selo: " + strings.Join(names, " e ") + " estão abertos."})
		}
	}
}

func has(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// oldQuest converte o índice da corrente antiga (17 missões, antes dos
// capítulos) para a nova, e diz quais selos o jogador já teria quebrado.
func oldQuest(q int) (int, uint) {
	conv := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 8, 8, 8, 8, 9, 10, 11, 12}
	if q >= len(conv) {
		return len(quests), 1<<5 - 1 // as cinco secundárias que existiam então
	}
	var done uint
	for i, need := range []int{3, 6, 12, 14, 16} { // onde a corrente antiga já tinha passado por cada região
		if q >= need {
			done |= 1 << i
		}
	}
	return conv[q], done
}

// ---------- recompensas ----------

type reward struct{ XP, Money int }

var mobReward = map[int]reward{
	mobSkeleton: {12, 6}, mobWarrior: {25, 12}, mobLord: {250, 150},
	mobAsura: {22, 12}, mobTriton: {26, 14}, mobSentinel: {32, 18},
	mobQueen: {400, 250}, mobTyrant: {480, 300}, mobArchon: {560, 350},
	mobRat: {6, 3}, mobWolf: {14, 7}, mobOrc: {20, 10}, mobOrcWarrior: {30, 16}, mobOrcChief: {300, 200},
	mobScarab: {22, 12}, mobMummy: {32, 18}, mobPharaoh: {520, 320},
	mobIceWolf: {26, 14}, mobYeti: {40, 22}, mobFrostGiant: {600, 380},
	mobSpider: {24, 12}, mobTroll: {34, 18}, mobBat: {8, 4}, mobDummy: {3, 0},
	mobCrawler: {44, 24}, mobFungusHusk: {48, 26}, mobMagmaGolem: {60, 32}, mobIgnar: {900, 600}, mobApostle: {70, 40}, mobGrael: {1100, 700}, mobSandScorpion: {46, 24}, mobIceWraith: {54, 30}, mobRootHorror: {56, 30}, mobDwarfGuard: {62, 34}, mobVulkhar: {1800, 1200}, mobCrystalGolem: {66, 36},
}

// dropCoins larga um montinho de dinheiro no chão.
func (wd *World) dropCoins(n int, x, y float64, owner int, life time.Duration, now time.Time) {
	if n <= 0 {
		return
	}
	wd.nextLoot++
	wd.loot = append(wd.loot, Loot{ID: wd.nextLoot, Item: itemCoins, X: int(x) + wd.rng.Intn(9) - 4, Y: int(y) + wd.rng.Intn(7) - 3,
		Owner: owner, N: n, until: now.Add(life)})
}

// pop é um texto que sobe sobre a cabeça de um jogador: XP, dinheiro, aviso.
type pop struct {
	ID    int
	Text  string
	Color int
	X, Y  float64
	Kind  int // 0 texto, 1 morte
	at    time.Time
}

func (wd *World) pop(p *Player, text string, color int, x, y float64, now time.Time) {
	wd.popKind(p, text, color, x, y, 0, now)
}

func (wd *World) popKind(p *Player, text string, color int, x, y float64, kind int, now time.Time) {
	wd.nextPop++
	live := p.pops[:0]
	for _, q := range p.pops {
		if now.Sub(q.at) < 3*time.Second {
			live = append(live, q)
		}
	}
	p.pops = append(live, pop{ID: wd.nextPop, Text: text, Color: color, X: x, Y: y, Kind: kind, at: now})
}

// ---------- mercadores ----------

const (
	shopArms = iota
	shopHerbs
)

type merchant struct {
	Name  string
	X, Y  int
	Kind  int
	Stock [][2]int // item, preço
}

const shopReach = 14.0

var merchantsBase = []merchant{
	{"Morvan, o ferreiro", villageX - 40, villageY + 12, shopArms, [][2]int{{itemShortSword, 120}, {itemRustDagger, 90}, {itemHuntSpear, 130},
		{itemMace, 140}, {itemAshStaff, 120}, {itemShortBow, 120}, {itemIronClaws, 130}, {itemIronKey, 150}, {itemAxe, 220}}},
	{"Ysolde, a herbalista", villageX + 44, villageY + 12, shopHerbs, [][2]int{{itemPotion, 40}, {itemAntidote, 30}, {itemEmberRoot, 30}, {itemAloe, 30},
		{itemForgetTonic, 300}, {itemRebirthTome, 400}, {itemLantern, 260}, {itemTorch, 15}, {itemEchoStone, 60}}},
	{"A Velha do Charco", swampX - 120, swampY - 60, shopHerbs, [][2]int{{itemPotion, 45}, {itemAntidote, 20}, {itemEmberRoot, 35}, {itemAloe, 35},
		{itemForgetTonic, 280}, {itemRebirthTome, 380}}},
	// Escondidos pelo mapa: vendem ampliações da mochila.
	{"O Eremita da Ilha", islandX - 8, islandY, shopHerbs, [][2]int{{itemBagSmall, 400}, {itemPotion, 50}}},
	{"Mascate das Sombras", 1312, overH + 182, shopHerbs, [][2]int{{itemBagMid, 900}, {itemBagSmall, 450}, {itemIronKey, 120}}},
	{"Velho Guardião de Vaelor", castleX - 40, castleCY, shopHerbs, [][2]int{{itemBagBig, 1600}, {itemBagMid, 950}}},
	// Nas Profundezas, perto da fenda a leste da vila: o que deixa andar na lava.
	{"Durgo, o anão do fundo", villageX + 152, deepY0 + villageY + 74, shopArms, [][2]int{{itemAshMantle, 700}, {itemObsidianBoots, 1100},
		{itemPotion, 60}, {itemLantern, 300}, {itemTorch, 20}, {itemEchoStone, 50}}},
}

// Buy compra um item do mercador, se o jogador estiver perto e tiver dinheiro.
func (wd *World) Buy(id, shop, it int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || shop < 0 || shop >= len(wd.shops) {
		return
	}
	m := wd.shops[shop]
	if dist(float64(p.X), float64(p.Y), float64(m.X), float64(m.Y)) > shopReach {
		return
	}
	now := wd.clock()
	for _, s := range m.Stock {
		if s[0] != it {
			continue
		}
		if p.Gold < s[1] {
			wd.pop(p, fmt.Sprintf("Faltam %d de %s", s[1]-p.Gold, currency), 0xd06060, float64(p.X), float64(p.Y)-34, now)
			return
		}
		if items[it].Kind == kindBag { // ampliação: vale na hora, não ocupa espaço
			if p.bagCap()+items[it].Bonus > maxBag {
				wd.pop(p, fmt.Sprintf("A mochila já está no máximo (%d)", maxBag), 0xd06060, float64(p.X), float64(p.Y)-34, now)
				return
			}
			p.Bag += items[it].Bonus
			p.Gold -= s[1]
			wd.pop(p, fmt.Sprintf("Mochila: %d espaços  −%d %s", p.bagCap(), s[1], currency), 0xe0b060, float64(p.X), float64(p.Y)-34, now)
			return
		}
		if !p.give(it, 1) {
			wd.pop(p, "Mochila cheia", 0xd06060, float64(p.X), float64(p.Y)-34, now)
			return
		}
		p.Gold -= s[1]
		wd.pop(p, fmt.Sprintf("%s  −%d %s", items[it].Name, s[1], currency), 0xe0b060, float64(p.X), float64(p.Y)-34, now)
		return
	}
}

// ---------- fogueiras ----------

type bonfire struct {
	Name string
	X, Y int
}

var bonfiresBase = []bonfire{
	{"Fogueira de Cinzalva", fireX, fireY},
	{"Fogueira do Cemitério", graveX + 80, graveY + 40},
	{"Fogueira da Encruzilhada", hubX - 50, hubY + 40},
	{"Fogueira da Selva", jungleX - 300, jungleY + 60},
	{"Fogueira da Costa", tideX - 240, tideY - 60},
	{"Fogueira das Areias", desertX - 40, desertY + 240},
	{"Fogueira dos Picos", snowX + 200, snowY - 120},
	{"Fogueira da Subida", lairX + 70, lairY + 170},
	{"Fogueira da Cripta", cryptInX + 20, cryptInY},
	{"Fogueira das Cavernas", 640, overH + 96},
	{"Fogueira do Acampamento", orcX, orcY - 6},
	{"Fogueira do Despertar", introX - 4, introY + 8}, // as de cima mantêm o índice (saves)
	{"Fogueira do Fundo", villageX + 132, deepY0 + villageY + 78},
	{"Fogueira das Raízes", jungleX - 100, deepY0 + jungleY + 98},
	{"Fogueira do Coração", lairX + 140, deepY0 + lairY + 138},
	{"Fogueira da Árvore Divina", treeX + treeBonfireOffX, treeY + treeBonfireOffY},
}

const bonfireReach = 12.0

// tendFire acende a fogueira em que o jogador encostou: é onde ele volta ao cair.
func (wd *World) tendFire(p *Player, now time.Time) {
	for i, b := range wd.bonfires {
		if p.Bonfire != i && dist(float64(p.X), float64(p.Y), float64(b.X), float64(b.Y)) <= bonfireReach {
			p.Bonfire = i
			wd.pop(p, "Chama acesa: "+b.Name, 0xffa040, float64(p.X), float64(p.Y)-40, now)
			wd.effects = append(wd.effects, Effect{Kind: fxHoly, X: float64(b.X), Y: float64(b.Y), R: 12, Start: now, Dur: 600 * time.Millisecond})
			return
		}
	}
}

// respawnAt devolve onde o jogador renasce: na última fogueira acesa.
func (wd *World) respawnAt(p *Player) (int, int) {
	if p.Bonfire > 0 && p.Bonfire < len(wd.bonfires) {
		b := wd.bonfires[p.Bonfire]
		if x, y, ok := wd.freeNear(b.X, b.Y+8, 20); ok {
			return x, y
		}
	}
	return wd.spawnPoint()
}

// freeNear procura chão livre perto de (x, y), em anéis até r.
func (wd *World) freeNear(x, y, r int) (int, int, bool) {
	for d := 0; d <= r; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if max(abs(dx), abs(dy)) == d && wd.fits(x+dx, y+dy) {
					return x + dx, y + dy, true
				}
			}
		}
	}
	return 0, 0, false
}

// slainDragon é o bit de Tiamath em Player.Slain (os tipos comuns usam o índice).
const slainDragon = uint64(1) << 63

// slew diz se o jogador já derrubou o chefe do tipo dado (também pelos bits
// antigos de Bosses, para saves de antes).
func (p *Player) slew(kind int) bool {
	if kind == killDragon {
		return p.Slain&slainDragon != 0 || p.Bosses&(1<<bossBit[killDragon]) != 0
	}
	if kind < 0 || kind >= 63 || !mobKinds[kind].Boss {
		return false
	}
	if bit, ok := bossBit[kind]; ok && p.Bosses&(1<<bit) != 0 {
		return true
	}
	return p.Slain&(1<<uint(kind)) != 0
}

// bossDone diz se a missão é "derrote o chefe X" e o jogador já o derrubou.
func (p *Player) bossDone(q quest) bool {
	if q.Kind != qKill || q.Need != 1 {
		return false
	}
	for _, k := range q.Targets {
		if p.slew(k) {
			return true
		}
	}
	return false
}

// catchUp fecha as missões de chefe que o jogador já cumpriu antes da hora:
// quem derrubou a Rainha ainda caçando as asuras não fica esperando ela
// renascer quando a missão dela chega. Vale para as secundárias também.
func (wd *World) catchUp(p *Player, now time.Time) {
	wd.catchStep(p, now) // etapa de chefe já derrubado, ou de região onde já está
	for i, q := range sideQuests {
		if p.SideDone&(1<<i) == 0 && p.chapterOf() >= q.Chapter && p.bossDone(q) {
			p.SideProg[i] = q.Need - 1
			for _, k := range q.Targets {
				if p.slew(k) {
					wd.creditSide(p, i, now)
					break
				}
			}
		}
	}
}
