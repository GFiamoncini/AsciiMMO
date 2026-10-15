package main

import (
	"asciimmo/proto"

	"fmt"
	"math"
	"time"
)

// Inimigos. Os chefes (Boss) têm pisão em área e largam uma arma só deles.
const (
	mobSkeleton = iota
	mobWarrior
	mobLord
	mobAsura
	mobTriton
	mobSentinel
	mobQueen
	mobTyrant
	mobArchon
	mobRat
	mobWolf
	mobOrc
	mobOrcWarrior
	mobOrcChief
	mobScarab
	mobMummy
	mobPharaoh
	mobIceWolf
	mobYeti
	mobFrostGiant
	mobSpider
	mobTroll
	mobBat
	mobDummy   // boneco de treino do prólogo: não anda nem bate
	mobCrawler // Profundezas (depths.go)
	mobMagmaGolem
	mobFungusHusk
	mobIgnar   // chefe das Profundezas, no coração sob o Pico
	mobApostle // Apóstolos e o Apóstolo Rei, na Árvore Divina (divina.go)
	mobGrael
	mobSandScorpion // Profundezas, rodada 2 (dorrim.go)
	mobIceWraith
	mobRootHorror
	mobDwarfGuard
	mobVulkhar      // o demônio do Abismo de Dorrim (vulkhar.go)
	mobCrystalGolem // Cavernas de Cristal (cristais.go)
)

type mobKind struct {
	Name       string
	HP         int
	Speed      float64 // células por tick
	Dmg, DmgSp int     // dano mínimo e variação
	Gold       int
	Respawn    time.Duration
	Boss       bool
	Drop       int // arma única do chefe
}

var mobKinds = []mobKind{
	mobSkeleton: {"Esqueleto", 60, 0.6, 4, 5, 8, 10 * time.Second, false, 0},
	mobWarrior:  {"Esqueleto Guerreiro", 140, 0.5, 8, 7, 20, 10 * time.Second, false, 0},
	mobLord:     {"Lorde da Cripta", 900, 0.45, 14, 10, 200, 45 * time.Second, true, itemLordBlade},
	mobAsura:    {"Asura", 120, 0.7, 7, 6, 16, 12 * time.Second, false, 0},
	mobTriton:   {"Tritão", 150, 0.55, 9, 6, 18, 12 * time.Second, false, 0},
	mobSentinel: {"Sentinela Radiante", 190, 0.5, 11, 7, 24, 12 * time.Second, false, 0},
	mobQueen:    {"Rainha Asura", 1500, 0.55, 16, 10, 300, 60 * time.Second, true, itemQueenFangs},
	mobTyrant:   {"Tirano das Marés", 1900, 0.45, 18, 10, 350, 60 * time.Second, true, itemTrident},
	mobArchon:   {"Arconte Radiante", 2400, 0.45, 20, 12, 400, 60 * time.Second, true, itemArchonHammer},

	mobRat:          {"Rato", 30, 0.75, 2, 3, 3, 10 * time.Second, false, 0},
	mobWolf:         {"Lobo", 70, 0.9, 5, 4, 8, 12 * time.Second, false, 0},
	mobOrc:          {"Orc", 110, 0.6, 7, 5, 14, 12 * time.Second, false, 0},
	mobOrcWarrior:   {"Orc Guerreiro", 170, 0.55, 10, 6, 22, 12 * time.Second, false, 0},
	mobOrcChief:     {"Chefe Orc", 1100, 0.5, 15, 9, 220, 60 * time.Second, true, itemOrcCleaver},
	mobScarab:       {"Escaravelho", 100, 0.7, 8, 5, 15, 12 * time.Second, false, 0},
	mobMummy:        {"Múmia", 180, 0.4, 12, 7, 22, 12 * time.Second, false, 0},
	mobPharaoh:      {"Faraó", 1700, 0.5, 17, 10, 320, 60 * time.Second, true, itemPharaohScepter},
	mobIceWolf:      {"Lobo do Gelo", 130, 0.95, 9, 6, 18, 12 * time.Second, false, 0},
	mobYeti:         {"Yeti", 240, 0.5, 14, 8, 28, 12 * time.Second, false, 0},
	mobFrostGiant:   {"Gigante de Gelo", 2100, 0.45, 19, 11, 380, 60 * time.Second, true, itemGlacialAxe},
	mobSpider:       {"Aranha Gigante", 140, 0.8, 10, 6, 18, 12 * time.Second, false, 0},
	mobTroll:        {"Troll do Pântano", 200, 0.5, 11, 7, 22, 12 * time.Second, false, 0},
	mobBat:          {"Morcego", 45, 1.0, 3, 4, 5, 10 * time.Second, false, 0},
	mobDummy:        {"Boneco de Treino", 40, 0, 0, 1, 0, 4 * time.Second, false, 0},
	mobCrawler:      {"Rastejante Cego", 230, 0.7, 13, 8, 26, 14 * time.Second, false, 0},
	mobMagmaGolem:   {"Golem de Magma", 420, 0.4, 19, 10, 40, 16 * time.Second, false, 0},
	mobFungusHusk:   {"Casca de Fungo", 260, 0.5, 14, 7, 30, 14 * time.Second, false, 0},
	mobIgnar:        {"Ignar, o Coração de Magma", 3600, 0.4, 26, 12, 600, 90 * time.Second, true, itemMagmaMaul},
	mobApostle:      {"Apóstolo Faminto", 560, 0.55, 21, 10, 50, 20 * time.Second, false, 0},
	mobGrael:        {"Grael, o Apóstolo Rei", 4400, 0.45, 29, 13, 700, 120 * time.Second, true, itemIronSlab},
	mobSandScorpion: {"Escorpião de Arenito", 240, 0.8, 14, 7, 28, 14 * time.Second, false, 0},
	mobIceWraith:    {"Espectro Gélido", 280, 0.7, 16, 8, 32, 14 * time.Second, false, 0},
	mobRootHorror:   {"Horror das Raízes", 330, 0.45, 17, 8, 34, 16 * time.Second, false, 0},
	mobDwarfGuard:   {"Guardião de Dorrim", 460, 0.4, 18, 9, 40, 18 * time.Second, false, 0},
	mobVulkhar:      {vulkharName, 6200, 0.45, 30, 14, 900, 180 * time.Second, true, itemVulkharWhip},
	mobCrystalGolem: {"Golem de Cristal", 540, 0.4, 20, 10, 46, 18 * time.Second, false, 0},
}

// Itens. O 0 é a poção, então Equip == 0 e Skin == 0 querem dizer "nada
// equipado". Armas vêm dos chefes; skins e as armas de baú, dos baús.
const (
	itemPotion = iota
	itemLordBlade
	itemQueenFangs
	itemTrident
	itemArchonHammer
	itemDragonClaw
	itemMidnightDagger
	itemSolarRelic
	itemTravelCloak
	itemCryptPlate
	itemAsuraGarb
	itemTideScales
	itemRadiantAegis
	itemDragonHide
	itemMarapaBow
	itemIronHelm
	itemAsuraCrown
	itemLeatherBoots
	itemTideGreaves
	itemCryptAmulet
	itemDragonEye
	itemOrcCleaver
	itemPharaohScepter
	itemGlacialAxe
	itemWood
	itemStone
	itemAntidote
	itemEmberRoot
	itemAloe
	itemLakeHelm
	itemDeepAmulet
	itemExplorerBoots
	itemCoins // montinho de dinheiro no chão; não vai para a mochila
	itemShortSword
	itemRustDagger
	itemHuntSpear
	itemMace
	itemAshStaff
	itemShortBow
	itemIronClaws
	itemForgetTonic // Tônico do Esquecimento: devolve os pontos de atributo
	itemRebirthTome // Tomo do Recomeço: devolve os pontos da teia de habilidades
	itemIronKey     // Chave de Ferro: abre um cofre trancado
	itemVaelorHelm  // do cofre das Ruínas de Vaelor
	itemGuardBoots  // do depósito da guarda da vila
	itemBagSmall    // ampliações da mochila (mercadores escondidos)
	itemBagMid
	itemBagBig
	itemStick   // pedaço de pau: a arma de início do Depravado
	itemLantern // ferramentas de ambiente (tools.go)
	itemTorch
	itemAxe
	itemObsidianBoots // Profundezas: andar na lava
	itemAshMantle     // Profundezas: lava e fogo queimam menos
	itemMagmaMaul     // arma de Ignar
	itemIronSlab      // Grael (divina.go)
	itemScarletEgg
	itemAbyssArmor
	itemGoldScarab // câmaras secretas das Profundezas (dorrim.go)
	itemIceCrown
	itemMyceliumBoots
	itemRootBow
	itemDorrimHelm
	itemDorrimHammer
	itemVulkharWhip // Vulkhar (vulkhar.go)
	itemHornCrown
	itemEchoStone    // Portões Rúnicos (portoes.go)
	itemCrystalHeart // cofre de cristal (cristais.go)
	itemPrismBlade
)

// bossGear é a peça de equipamento que cada chefe larga além da arma.
var bossGear = map[int]int{mobLord: itemCryptAmulet, mobQueen: itemAsuraCrown, mobTyrant: itemTideGreaves, mobIgnar: itemObsidianBoots, mobGrael: itemScarletEgg, mobVulkhar: itemHornCrown}

// commonGear sai, raramente, de qualquer inimigo comum.
var commonGear = []int{itemIronHelm, itemLeatherBoots}

const gearOdds = 5 // % de chance de um inimigo comum largar uma peça comum

// Tipos de arma. Cada classe tem afinidade com alguns: com eles o bônus da
// arma rende mais; com os outros, menos.
const (
	wSword = iota
	wDagger
	wSpear
	wHammer
	wStaff
	wBow
	wClaw
)

var weaponNames = []string{"espada", "adaga", "lança", "martelo", "cajado", "arco", "garra"}

var weaponType = map[int]int{
	itemLordBlade: wSword, itemQueenFangs: wDagger, itemTrident: wSpear, itemArchonHammer: wHammer,
	itemDragonClaw: wClaw, itemMidnightDagger: wDagger, itemSolarRelic: wStaff, itemMarapaBow: wBow,
	itemOrcCleaver: wSword, itemPharaohScepter: wStaff, itemGlacialAxe: wHammer,
	itemShortSword: wSword, itemRustDagger: wDagger, itemHuntSpear: wSpear, itemMace: wHammer, itemAshStaff: wStaff,
	itemShortBow: wBow, itemIronClaws: wClaw, itemStick: wHammer, itemMagmaMaul: wHammer, itemIronSlab: wSword, itemRootBow: wBow, itemDorrimHammer: wHammer, itemVulkharWhip: wSword, itemPrismBlade: wSword,
}

// affinity lista os tipos de arma de cada classe, na ordem de classes.
var affinity = [][]int{
	{wSword, wHammer}, // Guerreiro
	{wStaff},          // Mago
	{wSpear, wSword},  // Cavaleiro
	{wHammer, wSpear}, // Paladino
	{wBow, wDagger},   // Arqueiro
	{wDagger, wClaw},  // Ladino
	{wStaff, wClaw},   // Necromante
	{wSword, wDagger, wSpear, wHammer, wStaff, wBow, wClaw}, // Depravado: qualquer coisa serve
}

const (
	affinePct  = 130 // % do bônus da arma com afinidade
	clumsyPct  = 70  // % sem afinidade
	seasonDays = 2   // dias do jogo em cada estação
)

// affine diz se a classe tem afinidade com a arma.
func affine(class, it int) bool {
	for _, t := range affinity[class] {
		if t == weaponType[it] {
			return true
		}
	}
	return false
}

// weaponBonus é o bônus de dano da arma equipada, já pesado pela afinidade.
func (p *Player) weaponBonus() int {
	if p.Equip == 0 {
		return 0
	}
	if affine(p.Class, p.Equip) {
		return items[p.Equip].Bonus * affinePct / 100
	}
	return items[p.Equip].Bonus * clumsyPct / 100
}

// seasonOf devolve a estação do dia dado: 0 primavera, 1 verão, 2 outono, 3 inverno.
func seasonOf(day int) int { return (day - 1) / seasonDays % 4 }

const (
	kindPotion   = proto.KindPotion
	kindWeapon   = proto.KindWeapon
	kindSkin     = proto.KindSkin
	kindHelm     = proto.KindHelm
	kindBoots    = proto.KindBoots
	kindAmulet   = proto.KindAmulet
	kindMaterial = proto.KindMaterial
	kindHerb     = proto.KindHerb
	kindMoney    = proto.KindMoney
	kindScroll   = proto.KindScroll
	kindKey      = proto.KindKey
	kindBag      = proto.KindBag
	kindTool     = proto.KindTool
	kindFish     = proto.KindFish
)

// Raridade: quanto mais alta, mais forte o item e mais difícil sair do baú.
const (
	rarCommon = iota
	rarRare
	rarEpic
	rarLegend
)

type item struct {
	Name   string
	Kind   int
	Rarity int
	Bonus  int // arma: % de dano a mais
	Def    int // skin: % do dano recebido que ela absorve
	Magic  int // skin: % de dano a mais nas classes mágicas
	Heal   int
	Color  rgb
	Desc   string // de onde vem
}

var items = []item{
	itemPotion:         {"Poção de Vida", kindPotion, rarCommon, 0, 0, 0, 60, 0xe04058, "Recupera 60 de vida"},
	itemLordBlade:      {"Lâmina do Lorde", kindWeapon, rarRare, 15, 0, 0, 0, 0xb080ff, "Lorde da Cripta"},
	itemQueenFangs:     {"Presas da Rainha", kindWeapon, rarEpic, 25, 0, 0, 0, 0xff6ab0, "Rainha Asura"},
	itemTrident:        {"Tridente das Marés", kindWeapon, rarEpic, 30, 0, 0, 0, 0x50e0d0, "Tirano das Marés"},
	itemArchonHammer:   {"Martelo do Arconte", kindWeapon, rarLegend, 40, 0, 0, 0, 0xffe070, "Arconte Radiante"},
	itemDragonClaw:     {"Garra do Dragão", kindWeapon, rarLegend, 55, 0, 0, 0, 0xff7030, "Dragão"},
	itemMidnightDagger: {"Adaga da Meia-Noite", kindWeapon, rarRare, 18, 0, 0, 0, 0x7080ff, "Baú da cripta, à noite"},
	itemSolarRelic:     {"Relíquia Solar", kindWeapon, rarLegend, 45, 0, 0, 0, 0xfff0a0, "Baú do bastião, de dia"},
	itemTravelCloak:    {"Manto do Viajante", kindSkin, rarCommon, 0, 6, 0, 0, 0x9aa878, "Baú do fundo da cripta"},
	itemCryptPlate:     {"Couraça da Cripta", kindSkin, rarRare, 0, 14, 0, 0, 0x8a88b0, "Baú da cripta, de dia"},
	itemAsuraGarb:      {"Traje Asura", kindSkin, rarEpic, 0, 10, 18, 0, 0xe04a90, "Baú da cidadela, à noite"},
	itemTideScales:     {"Escamas das Marés", kindSkin, rarEpic, 0, 20, 6, 0, 0x40c8c0, "Baú de Talássia, de dia"},
	itemRadiantAegis:   {"Égide Radiante", kindSkin, rarLegend, 0, 26, 22, 0, 0xfff4c0, "Baú do bastião, à noite"},
	itemDragonHide:     {"Pele de Dragão", kindSkin, rarLegend, 0, 32, 12, 0, 0xe05a28, "Dragão"},
	itemMarapaBow:      {"Arco de Marapá", kindWeapon, rarRare, 20, 0, 0, 0, 0x90e060, "Baú da cidadela, de dia"},
	itemIronHelm:       {"Elmo de Ferro", kindHelm, rarCommon, 0, 4, 0, 0, 0xa0a8c0, "Inimigos comuns"},
	itemAsuraCrown:     {"Coroa Asura", kindHelm, rarEpic, 0, 6, 12, 0, 0xff8ad0, "Rainha Asura"},
	itemLeatherBoots:   {"Botas de Couro", kindBoots, rarCommon, 0, 3, 0, 0, 0xa07848, "Inimigos comuns"},
	itemTideGreaves:    {"Grevas das Marés", kindBoots, rarRare, 0, 8, 0, 0, 0x50c8c0, "Tirano das Marés"},
	itemCryptAmulet:    {"Amuleto da Cripta", kindAmulet, rarRare, 6, 0, 6, 0, 0xb090ff, "Lorde da Cripta"},
	itemDragonEye:      {"Olho do Dragão", kindAmulet, rarLegend, 12, 4, 12, 0, 0xffb040, "Dragão"},
	itemOrcCleaver:     {"Cutelo Orc", kindWeapon, rarRare, 18, 0, 0, 0, 0xc06040, "Chefe Orc"},
	itemPharaohScepter: {"Cetro do Faraó", kindWeapon, rarEpic, 28, 0, 0, 0, 0xf0d060, "Faraó"},
	itemGlacialAxe:     {"Machado Glacial", kindWeapon, rarEpic, 30, 0, 0, 0, 0x90d8ff, "Gigante de Gelo"},
	itemWood:           {"Madeira", kindMaterial, rarCommon, 0, 0, 0, 0, 0xb08050, "Árvores derrubadas; 6 viram jangada"},
	itemStone:          {"Pedra", kindMaterial, rarCommon, 0, 0, 0, 0, 0xa8a4a0, "Rochas quebradas"},
	itemAntidote:       {"Erva-antídoto", kindHerb, rarCommon, 0, 0, 0, 10, 0x8ae060, "Cura o veneno (pântano)"},
	itemEmberRoot:      {"Raiz-brasa", kindHerb, rarCommon, 0, 0, 0, 10, 0xff8a40, "Aquece contra o frio (gelo)"},
	itemAloe:           {"Babosa", kindHerb, rarCommon, 0, 0, 0, 10, 0x60d0a0, "Refresca contra o calor (deserto)"},
	itemLakeHelm:       {"Elmo do Lago", kindHelm, rarRare, 0, 6, 6, 0, 0x6ab8e0, "Baú da ilha do lago"},
	itemDeepAmulet:     {"Amuleto das Profundezas", kindAmulet, rarEpic, 8, 4, 8, 0, 0x9a7aff, "Sala secreta das cavernas"},
	itemExplorerBoots:  {"Botas do Explorador", kindBoots, rarRare, 0, 6, 0, 0, 0xc0a070, "Gruta do buraco"},
	itemCoins:          {"Bosta", kindMoney, rarCommon, 0, 0, 0, 0, 0xc09040, "Dinheiro do jogo"},
	itemShortSword:     {"Espada Curta", kindWeapon, rarCommon, 8, 0, 0, 0, 0xc8c8d8, "Ferreiro da vila"},
	itemRustDagger:     {"Adaga Enferrujada", kindWeapon, rarCommon, 6, 0, 0, 0, 0xb08070, "Ferreiro da vila"},
	itemHuntSpear:      {"Lança de Caça", kindWeapon, rarCommon, 8, 0, 0, 0, 0xc0b090, "Ferreiro da vila"},
	itemMace:           {"Maça de Ferro", kindWeapon, rarCommon, 9, 0, 0, 0, 0x9aa0b0, "Ferreiro da vila"},
	itemAshStaff:       {"Cajado de Freixo", kindWeapon, rarCommon, 8, 0, 0, 0, 0xa0c080, "Ferreiro da vila"},
	itemShortBow:       {"Arco Curto", kindWeapon, rarCommon, 8, 0, 0, 0, 0xb09060, "Ferreiro da vila"},
	itemIronClaws:      {"Garras de Ferro", kindWeapon, rarCommon, 8, 0, 0, 0, 0xa8a8b8, "Ferreiro da vila"},
	itemForgetTonic:    {"Tônico do Esquecimento", kindScroll, rarRare, 0, 0, 0, 0, 0x9ad0c0, "Devolve todos os pontos de atributo para redistribuir"},
	itemRebirthTome:    {"Tomo do Recomeço", kindScroll, rarRare, 0, 0, 0, 0, 0xc8a0ff, "Devolve todos os pontos da teia de habilidades"},
	itemIronKey:        {"Chave de Ferro", kindKey, rarCommon, 0, 0, 0, 0, 0xc0b8a0, "Abre um cofre trancado (some ao usar)"},
	itemVaelorHelm:     {"Elmo de Vaelor", kindHelm, rarEpic, 0, 9, 8, 0, 0xa8b0d8, "Cofre das Ruínas de Vaelor"},
	itemGuardBoots:     {"Botas da Guarda", kindBoots, rarRare, 0, 7, 0, 0, 0x9a8060, "Depósito da guarda da vila"},
	itemBagSmall:       {"Bolsa de Couro", kindBag, rarCommon, 10, 0, 0, 0, 0xa07848, "+10 espaços na mochila"},
	itemBagMid:         {"Mochila de Explorador", kindBag, rarRare, 20, 0, 0, 0, 0x8a9a5a, "+20 espaços na mochila"},
	itemBagBig:         {"Alforje Encantado", kindBag, rarEpic, 30, 0, 0, 0, 0xb080e0, "+30 espaços na mochila"},
	itemStick:          {"Pedaço de Pau", kindWeapon, rarCommon, 3, 0, 0, 0, 0x8a6a44, "Achado no chão. Melhor que nada"},
	itemLantern:        {"Lanterna", kindTool, rarRare, 0, 0, 0, 0, 0xffd070, "Luz em volta de você; funciona na chuva. Clique para acender ou apagar"},
	itemTorch:          {"Tocha", kindTool, rarCommon, 0, 0, 0, 0, 0xff8a30, "Luz forte por 5 minutos; a chuva apaga. Pegue de graça nas fogueiras"},
	itemAxe:            {"Machado", kindTool, rarRare, 0, 0, 0, 0, 0xb0b8c8, "Na mochila, derruba árvores e moitas 4x mais rápido e rende mais madeira"},
	itemObsidianBoots:  {"Botas de Obsidiana", kindBoots, rarRare, 0, 7, 0, 0, 0x403848, "Calçadas, a lava não queima. Durgo, nas Profundezas, ou Ignar"},
	itemAshMantle:      {"Manto de Cinzas", kindSkin, rarRare, 0, 12, 0, 0, 0x8a7a70, "Vestido, a lava e o fogo queimam só um quarto. Durgo, nas Profundezas"},
	itemMagmaMaul:      {"Malho de Magma", kindWeapon, rarEpic, 34, 0, 0, 0, 0xff6020, "Ignar, o Coração de Magma"},
	itemIronSlab:       {"Lâmina de Ferro Bruto", kindWeapon, rarEpic, 42, 0, 0, 0, 0x8a8a90, "Grael, o Apóstolo Rei. Grande demais para ser chamada de espada"},
	itemScarletEgg:     {"Ovo Escarlate", kindAmulet, rarEpic, 0, 5, 0, 0, 0xc02030, "Um ovo de carne com um rosto fora do lugar. Abaixo de 1/4 da vida, chora sangue e devolve 30% dela (a cada 2 min). Grael"},
	itemAbyssArmor:     {"Armadura do Abismo", kindSkin, rarEpic, 0, 18, 0, 0, 0x2a2a34, "+40% de dano com menos de metade da vida, mas cobra 2 de vida por segundo. Baú sob a Árvore Divina, nas Profundezas"},
	itemGoldScarab:     {"Escaravelho Dourado", kindAmulet, rarRare, 0, 6, 8, 0, 0xffd060, "Câmara secreta das Grutas de Arenito"},
	itemIceCrown:       {"Coroa do Gelo Antigo", kindHelm, rarRare, 0, 10, 0, 0, 0x90f0ff, "Câmara secreta do Gelo Antigo"},
	itemMyceliumBoots:  {"Botas de Micélio", kindBoots, rarRare, 0, 8, 0, 0, 0x9aff8a, "Câmara secreta do Jardim de Fungos"},
	itemRootBow:        {"Arco de Raiz Viva", kindWeapon, rarEpic, 30, 0, 0, 0, 0x7ab060, "Câmara secreta das Raízes de Marapá"},
	itemDorrimHelm:     {"Elmo de Dorrim", kindHelm, rarEpic, 0, 11, 0, 0, 0xb0b8d0, "Cofre do rei, nos Salões de Dorrim (de dia)"},
	itemDorrimHammer:   {"Martelo de Dorrim", kindWeapon, rarEpic, 36, 0, 0, 0, 0xc0c8e0, "Cofre do rei, nos Salões de Dorrim (de noite)"},
	itemVulkharWhip:    {"Chicote de Vulkhar", kindWeapon, rarEpic, 46, 0, 0, 0, 0xff5018, "Vulkhar, o Flagelo das Profundezas"},
	itemHornCrown:      {"Coroa de Chifres", kindHelm, rarEpic, 0, 14, 10, 0, 0x5a1a14, "Vulkhar, o Flagelo das Profundezas"},
	itemEchoStone:      {"Pedra de Eco", kindMaterial, rarCommon, 0, 0, 0, 0, 0xb8a0ff, "Atravessar um Portão Rúnico gasta uma. Cai nas Profundezas; Ysolde e Durgo vendem"},
	itemCrystalHeart:   {"Coração de Cristal", kindAmulet, rarEpic, 0, 6, 16, 0, 0xd0b0ff, "Cofre de cristal, nas Cavernas de Cristal (de dia)"},
	itemPrismBlade:     {"Lâmina Prismática", kindWeapon, rarEpic, 40, 0, 0, 0, 0xe0d0ff, "Cofre de cristal, nas Cavernas de Cristal (de noite)"},
}

// herbCure diz que status cada erva cura e por quanto tempo protege depois.
var herbCure = map[int]struct {
	Status int
	Immune time.Duration
}{
	itemAntidote:  {stPoison, time.Minute},
	itemEmberRoot: {stCold, 2 * time.Minute},
	itemAloe:      {stHeat, 2 * time.Minute},
}

const (
	baseBag     = 50  // espaços da mochila de início
	maxBag      = 140 // teto com todas as ampliações
	lootReach   = 6.0
	lootLife    = 90 * time.Second
	shareRange  = 110.0 // quem está até aqui divide o saque e o crédito da missão
	potionOdds  = 25    // % de chance de um inimigo comum largar poção
	bossStompR  = 20.0
	bossStompCD = 4 * time.Second
)

type invSlot struct{ Item, N int }

// Loot é um item caído no chão. Com Owner diferente de zero, só aquele
// jogador vê e pega: é assim que cada um recebe a sua arma do chefe.
type Loot struct {
	ID, Item, X, Y, Owner int
	N                     int // dinheiro: quanto há no montinho
	by                    int // quem largou (não recolhe de volta logo)
	byUntil               time.Time
	until                 time.Time
}

func (p *Player) count(it int) int {
	for _, s := range p.inv {
		if s.Item == it {
			return s.N
		}
	}
	return 0
}

// give põe n unidades do item na mochila; falha se não houver espaço.
func (p *Player) give(it, n int) bool {
	for i := range p.inv {
		if p.inv[i].Item == it {
			p.inv[i].N += n
			return true
		}
	}
	if len(p.inv) >= p.bagCap() {
		return false
	}
	p.inv = append(p.inv, invSlot{it, n})
	// A primeira peça de cada espaço já entra equipada.
	if s := p.slot(items[it].Kind); s != nil && *s == 0 {
		*s = it
	}
	return true
}

func (p *Player) take(it int) {
	for i := range p.inv {
		if p.inv[i].Item == it {
			if p.inv[i].N--; p.inv[i].N <= 0 {
				p.inv = append(p.inv[:i], p.inv[i+1:]...)
			}
			return
		}
	}
}

// UseItem bebe a poção ou equipa/desequipa a peça no espaço dela.
func (wd *World) UseItem(id, it int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || it < 0 || it >= len(items) || p.count(it) == 0 {
		return
	}
	def := items[it]
	if s := p.slot(def.Kind); s != nil {
		if *s == it {
			*s = 0
		} else {
			*s = it
		}
		return
	}
	if c, ok := herbCure[it]; ok {
		now := wd.clock()
		p.status[c.Status], p.immune[c.Status] = time.Time{}, now.Add(c.Immune)
		p.HP = min(p.MaxHP, p.HP+def.Heal)
		p.take(it)
		wd.effects = append(wd.effects, Effect{Kind: fxHeal, X: float64(p.X), Y: float64(p.Y), R: 8, Start: now, Dur: 500 * time.Millisecond})
		return
	}
	if def.Kind == kindTool {
		if it == itemLantern || it == itemTorch {
			wd.equipLight(p, it, wd.clock()) // vai para o espaço de luz e acende (luz.go)
		}
		return
	}
	if def.Kind == kindScroll {
		now := wd.clock()
		switch it {
		case itemForgetTonic:
			p.resetAttrs()
			wd.pop(p, "Atributos devolvidos: redistribua em I", 0x9ad0c0, float64(p.X), float64(p.Y)-34, now)
		case itemRebirthTome:
			p.resetWeb()
			wd.pop(p, "Teia devolvida: aprenda de novo em K", 0xc8a0ff, float64(p.X), float64(p.Y)-34, now)
		}
		p.take(it)
		wd.effects = append(wd.effects, Effect{Kind: fxBuff, X: float64(p.X), Y: float64(p.Y), R: 12, Dir: 3, Start: now, Dur: 800 * time.Millisecond})
		return
	}
	switch {
	case def.Kind != kindPotion:
	case p.HP < p.MaxHP:
		p.HP = min(p.MaxHP, p.HP+def.Heal)
		p.take(it)
		wd.effects = append(wd.effects, Effect{Kind: fxHeal, X: float64(p.X), Y: float64(p.Y), R: 8, Start: wd.clock(), Dur: 500 * time.Millisecond})
	}
}

func (wd *World) near(x, y float64) []*Player {
	var out []*Player
	for _, p := range wd.players {
		if dist(float64(p.X), float64(p.Y), x, y) <= shareRange {
			out = append(out, p)
		}
	}
	return out
}

func (wd *World) drop(it int, x, y float64, owner int, now time.Time) {
	wd.nextLoot++
	wd.loot = append(wd.loot, Loot{ID: wd.nextLoot, Item: it, X: int(x) + wd.rng.Intn(13) - 6, Y: int(y) + wd.rng.Intn(9) - 4, Owner: owner, until: now.Add(lootLife)})
}

// dropFor larga o item uma vez para cada jogador por perto que ainda não o
// tem; cada um só vê e pega o seu.
func (wd *World) dropFor(it int, x, y float64, now time.Time) {
	for _, p := range wd.near(x, y) {
		if p.count(it) == 0 {
			wd.drop(it, x, y, p.ID, now)
		}
	}
}

// bossDrop larga a arma do chefe para cada jogador por perto e duas poções
// para quem chegar primeiro.
func (wd *World) bossDrop(it int, x, y float64, now time.Time) {
	wd.dropFor(it, x, y, now)
	wd.drop(itemPotion, x, y, 0, now)
	wd.drop(itemPotion, x, y, 0, now)
}

// mobDied cuida do que um inimigo deixa ao cair: ouro para quem bateu, saque
// no chão e crédito de missão para o grupo.
func (wd *World) mobDied(m *Mob, killer *Player, now time.Time) {
	k := mobKinds[m.Kind]
	m.HP, m.Alive, m.armed, m.respawnAt = 0, false, false, now.Add(k.Respawn)
	if killer != nil {
		killer.countKill(m.Kind) // bestiário (bestiario.go)
	}
	if killer != nil {
		killer.Kills++
	}
	// A experiência da tabela vai para todos por perto; o dinheiro cai no chão.
	rw := mobReward[m.Kind]
	for _, p := range wd.near(m.X, m.Y) {
		if bit, ok := bossBit[m.Kind]; ok {
			p.Bosses |= 1 << bit
		}
		if mobKinds[m.Kind].Boss {
			first := p.Slain&(1<<uint(m.Kind)) == 0
			p.Slain |= 1 << uint(m.Kind)
			if first {
				wd.purifyNews(p, m.Kind) // purificou a região (veu.go)
			}
		}
		wd.pop(p, fmt.Sprintf("+%d XP (%d%%)", rw.XP, max(1, rw.XP*100/xpNeed(p.Level))), 0x9ad0ff, m.X, m.Y-20, now)
		wd.gainXP(p, rw.XP)
	}
	wd.dropCoins(rw.Money*(85+wd.rng.Intn(31))/100, m.X, m.Y, 0, lootLife, now)
	wd.effects = append(wd.effects, Effect{Kind: fxPuff, X: m.X, Y: m.Y, R: 10, Start: now, Dur: 600 * time.Millisecond})
	if inDeep(int(m.Y)) && wd.rng.Intn(100) < 12 { // Pedra de Eco: dos de baixo
		wd.drop(itemEchoStone, m.X, m.Y, 0, now)
	}
	switch roll := wd.rng.Intn(100); {
	case m.Kind == mobDummy: // boneco de palha não larga nada
	case k.Boss:
		wd.say(ChatLine{Text: k.Name + " foi derrotado!"})
		wd.bossDrop(k.Drop, m.X, m.Y, now)
		if gear, ok := bossGear[m.Kind]; ok {
			wd.dropFor(gear, m.X, m.Y, now)
		}
	case roll < potionOdds:
		wd.drop(itemPotion, m.X, m.Y, 0, now)
	case roll < potionOdds+gearOdds:
		wd.drop(commonGear[wd.rng.Intn(len(commonGear))], m.X, m.Y, 0, now)
	}
	wd.credit(m.Kind, m.X, m.Y, now)
}

// stepLoot recolhe o que cada jogador pisou e some com o saque velho.
func (wd *World) stepLoot(now time.Time) {
	live := wd.loot[:0]
	for _, l := range wd.loot {
		if now.After(l.until) {
			continue
		}
		taken := false
		for _, p := range wd.players {
			if (l.Owner == 0 || l.Owner == p.ID) && l.Item == itemCoins && dist(float64(p.X), float64(p.Y), float64(l.X), float64(l.Y)) <= lootReach {
				p.Gold += l.N
				wd.pop(p, fmt.Sprintf("+%d %s", l.N, currency), 0xe0b060, float64(l.X), float64(l.Y)-10, now)
				taken = true
				break
			}
			if l.Item != itemCoins && (l.by != p.ID || now.After(l.byUntil)) && (l.Owner == 0 || l.Owner == p.ID) && dist(float64(p.X), float64(p.Y), float64(l.X), float64(l.Y)) <= lootReach && p.give(l.Item, 1) {
				if items[l.Item].Kind != kindPotion {
					wd.say(ChatLine{Text: p.Name + " obteve " + items[l.Item].Name + "!"})
				}
				taken = true
				break
			}
		}
		if !taken {
			live = append(live, l)
		}
	}
	wd.loot = live
}

// bossStomp é o golpe em área dos chefes: avisa no chão e, um instante
// depois, fere quem não saiu. Devolve true enquanto o chefe está ocupado.
func (wd *World) bossStomp(m *Mob, k mobKind, d float64, now time.Time) bool {
	if m.armed {
		if now.After(m.stompAt) {
			m.armed, m.nextStomp = false, now.Add(bossStompCD)
			wd.effects = append(wd.effects, Effect{Kind: fxBlast, X: m.stompX, Y: m.stompY, R: bossStompR, Start: now, Dur: 500 * time.Millisecond})
			for _, p := range wd.players {
				if dist(float64(p.X), float64(p.Y), m.stompX, m.stompY) <= bossStompR {
					wd.hurt(p, max(1, int(float64(k.Dmg+8+wd.rng.Intn(k.DmgSp))*lvScale(mobLv(m.Kind), p.Level))), "por "+k.Name, now)
				}
			}
		}
		return true
	}
	if d < 30 && now.After(m.nextStomp) {
		m.armed, m.stompAt, m.stompX, m.stompY = true, now.Add(900*time.Millisecond), m.X, m.Y
		wd.effects = append(wd.effects, Effect{Kind: fxWarn, X: m.X, Y: m.Y, R: bossStompR, Start: now, Dur: 900 * time.Millisecond})
		return true
	}
	return false
}

// timeOfDay vai de 0 a 1: 0 é meia-noite, 0.5 é meio-dia. O mundo começa de manhã.
func (wd *World) timeOfDay(t float64) float64 {
	return math.Mod(t/wd.dayLen+0.35+wd.todShift, 1)
}

// unequipLast tira a peça do corpo se for a última unidade dela.
func (p *Player) unequipLast(it int) {
	if s := p.slot(items[it].Kind); s != nil && *s == it && p.count(it) <= 1 {
		*s = 0
	}
}

// Drop joga fora uma unidade do item: ele cai no chão, perto, e qualquer um
// pode pegar (quem largou só depois de alguns segundos).
func (wd *World) Drop(id, it int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || it < 0 || it >= len(items) || p.count(it) == 0 {
		return
	}
	now := wd.clock()
	p.unequipLast(it)
	p.take(it)
	wd.nextLoot++
	wd.loot = append(wd.loot, Loot{ID: wd.nextLoot, Item: it, X: p.X + p.FaceX*6, Y: p.Y + 2, by: id, byUntil: now.Add(8 * time.Second), until: now.Add(lootLife)})
	wd.pop(p, "Jogou fora: "+items[it].Name, 0xa0a0b0, float64(p.X), float64(p.Y)-34, now)
}

// sellPrice é quanto o mercador paga pelo item: metade do preço de venda
// dele, se ele vende; senão, pelo tipo e pela raridade.
func sellPrice(m *merchant, it int) int {
	for _, s := range m.Stock {
		if s[0] == it {
			return max(1, s[1]/2)
		}
	}
	d := items[it]
	switch d.Kind {
	case kindPotion:
		return 15
	case kindMaterial:
		return 3
	case kindHerb:
		return 10
	case kindScroll:
		return 100
	case kindMoney:
		return 0
	case kindFish:
		return fishPrice(it)
	}
	return []int{30, 90, 200, 400}[clampInt(d.Rarity, 0, 3)]
}

// Sell vende uma unidade do item a um mercador por perto.
func (wd *World) Sell(id, shop, it int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || shop < 0 || shop >= len(wd.shops) || it < 0 || it >= len(items) || p.count(it) == 0 {
		return
	}
	m := wd.shops[shop]
	if dist(float64(p.X), float64(p.Y), float64(m.X), float64(m.Y)) > shopReach {
		return
	}
	price := sellPrice(&m, it)
	if price <= 0 {
		return
	}
	p.unequipLast(it)
	p.take(it)
	p.Gold += price
	wd.pop(p, fmt.Sprintf("Vendeu %s: +%d %s", items[it].Name, price, currency), 0xffd070, float64(p.X), float64(p.Y)-34, wd.clock())
}

// bagCap é quantos espaços a mochila do jogador tem.
func (p *Player) bagCap() int { return baseBag + p.Bag }
