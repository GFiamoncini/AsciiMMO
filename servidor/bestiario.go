package main

import "time"

// Bestiário: toda criatura que o jogador vê de perto, mata ou doma entra no
// registro (BestSeen, BestKills e os bits das montarias domadas), com uma
// história curta, as fraquezas e as resistências. As fraquezas valem no
// combate: golpe do elemento fraco tira 50% a mais; do resistente, 40% a
// menos (o número que sobe sai amarelo ou cinza). Cada um é mostrado no
// diário (N), na aba Bestiário.

type beastInfo struct {
	Lore         string
	Weak, Resist []int
}

var bestiary = map[int]beastInfo{
	mobSkeleton:     {"Ossos do cemitério de Cinzalva, de pé de novo. Batem fraco, mas nunca vêm sozinhos.", []int{elemHoly, elemPhys}, []int{elemPoison, elemShadow}},
	mobWarrior:      {"Soldados mortos que ainda lembram a formação. Escudo velho, espada enferrujada.", []int{elemHoly}, []int{elemPoison, elemShadow}},
	mobLord:         {"O senhor da cripta. Comandava os mortos antes de morrer e continua comandando.", []int{elemHoly, elemFire}, []int{elemPoison, elemShadow, elemIce}},
	mobAsura:        {"Guardiãs de muitos braços da Cidadela. Cantam enquanto lutam.", []int{elemIce}, []int{elemFire}},
	mobTriton:       {"Povo do mar de Talássia, de pele fria e lança curta.", []int{elemBolt}, []int{elemIce, elemFire}},
	mobSentinel:     {"Estátuas do Bastião Radiante que andam quando alguém se aproxima.", []int{elemBolt, elemPhys}, []int{elemFire, elemPoison}},
	mobQueen:        {"A Rainha Asura, que dança no altar da selva. Mata rodando.", []int{elemIce, elemShadow}, []int{elemFire}},
	mobTyrant:       {"O Tirano das Marés: um tritão do tamanho de um barco.", []int{elemBolt}, []int{elemIce, elemFire}},
	mobArchon:       {"O Arconte guarda a luz do Bastião. Ninguém sabe a quem ele obedece.", []int{elemShadow}, []int{elemHoly, elemBolt}},
	mobRat:          {"Ratos das cavernas, gordos de restos. Mordem e fogem.", []int{elemFire, elemPhys}, nil},
	mobWolf:         {"Lobos das planícies. Caçam em bando e uivam de noite.", []int{elemFire}, nil},
	mobOrc:          {"Orcs do acampamento do leste. Força bruta e pouca paciência.", []int{elemArcane}, []int{elemPhys}},
	mobOrcWarrior:   {"Orc de armadura de couro e cutelo pesado.", []int{elemArcane, elemBolt}, []int{elemPhys}},
	mobOrcChief:     {"O chefe orc. Grita para os outros e bate mais forte que todos.", []int{elemArcane, elemBolt}, []int{elemPhys, elemFire}},
	mobScarab:       {"Escaravelhos do deserto que levam a areia para dentro da pirâmide.", []int{elemIce}, []int{elemFire}},
	mobMummy:        {"Mortos enfaixados de Ankar. O pano seco queima fácil.", []int{elemFire, elemHoly}, []int{elemPoison, elemShadow}},
	mobPharaoh:      {"O faraó que sonha acordado há mil anos sob a pirâmide.", []int{elemFire, elemHoly}, []int{elemPoison, elemShadow, elemIce}},
	mobIceWolf:      {"Lobos brancos dos Picos de Gelo. O frio não os incomoda.", []int{elemFire}, []int{elemIce}},
	mobYeti:         {"Bufa, ruge e arremessa gelo. Mora nas cavernas dos Picos.", []int{elemFire}, []int{elemIce}},
	mobFrostGiant:   {"Gigante de gelo dos picos mais altos. Cada passo racha o chão.", []int{elemFire, elemBolt}, []int{elemIce}},
	mobSpider:       {"Aranhas do pântano de Venor. O veneno delas paralisa.", []int{elemFire}, []int{elemPoison}},
	mobTroll:        {"Trolls do charco. Feridas fecham sozinhas, menos as de fogo.", []int{elemFire}, []int{elemPhys}},
	mobBat:          {"Morcegos das cavernas. Somem no escuro e voltam.", []int{elemBolt}, nil},
	mobDummy:        {"Boneco de treino do Vale do Despertar. Não revida.", nil, nil},
	mobCrawler:      {"Coisa cega que rasteja nas Profundezas e sente o calor dos vivos.", []int{elemFire, elemHoly}, []int{elemShadow}},
	mobMagmaGolem:   {"Pedra derretida que anda. Ferve tudo em volta.", []int{elemIce}, []int{elemFire, elemPoison}},
	mobFungusHusk:   {"Casca de fungo que já foi alguém. Solta esporos venenosos.", []int{elemFire}, []int{elemPoison}},
	mobIgnar:        {"Ignar, o Coração de Magma. O fogo das Profundezas tem um rosto.", []int{elemIce}, []int{elemFire, elemPoison}},
	mobApostle:      {"Apóstolos famintos rondam a Árvore Divina. Já foram homens que pediram demais.", []int{elemHoly}, []int{elemShadow}},
	mobGrael:        {"Grael, o Apóstolo Rei, guarda a clareira da Árvore. Ele não tem fome: tem sede.", []int{elemHoly}, []int{elemShadow, elemPoison}},
	mobSandScorpion: {"Escorpiões das Grutas de Arenito. A cauda envenena.", []int{elemIce}, []int{elemPoison}},
	mobIceWraith:    {"Espectro do Gelo Antigo: um lamento frio que corta.", []int{elemFire, elemHoly}, []int{elemIce, elemPhys}},
	mobRootHorror:   {"Raízes da Árvore Divina que aprenderam a caçar.", []int{elemFire}, []int{elemPoison}},
	mobDwarfGuard:   {"Guardiões de Dorrim. Morreram de pé e continuam de guarda.", []int{elemBolt}, []int{elemPhys}},
	mobVulkhar:      {"Vulkhar, o Flagelo das Profundezas: chicote de fogo e uma ponte só.", []int{elemIce, elemHoly}, []int{elemFire, elemShadow}},
	mobCrystalGolem: {"Golem de cristal das cavernas sob o lago. A luz atravessa, a lâmina não.", []int{elemPhys, elemBolt}, []int{elemArcane}},
}

// Montarias no bestiário (vistas e domadas).
var steedLore = []string{
	"Cavalos das planícies em volta da vila. Mansos para quem tem fôlego.",
	"Javalis da selva: teimosos, mas aguentam pancada (proteção montado).",
	"Avestruzes do deserto, rápidas e assustadas.",
	"Velociraptores das ruínas. Os mais rápidos, os mais bravos.",
}

// weakPct é quanto o elemento faz no bicho (100 normal, 150 fraco, 60 resistente).
func weakPct(kind, elem int) int {
	b, ok := bestiary[kind]
	if !ok {
		return 100
	}
	for _, e := range b.Weak {
		if e == elem {
			return 150
		}
	}
	for _, e := range b.Resist {
		if e == elem {
			return 60
		}
	}
	return 100
}

// seeBeasts registra o que está perto do jogador (uma vez a cada 2 s).
func (wd *World) seeBeasts(p *Player, now time.Time) {
	for _, m := range wd.mobs {
		if m.Alive && m.Kind < 64 && dist(m.X, m.Y, float64(p.X), float64(p.Y)) < 50 {
			p.BestSeen |= 1 << m.Kind
		}
	}
	for _, s := range wd.steeds {
		if s.Alive && dist(s.X, s.Y, float64(p.X), float64(p.Y)) < 50 {
			p.SteedSeen |= 1 << s.Kind
		}
	}
}

// countKill soma uma morte do tipo no bestiário de quem matou.
func (p *Player) countKill(kind int) {
	if kind < 0 || kind >= len(mobKinds) {
		return
	}
	for len(p.BestKills) < len(mobKinds) {
		p.BestKills = append(p.BestKills, 0)
	}
	p.BestKills[kind]++
	if kind < 64 {
		p.BestSeen |= 1 << kind
	}
}

// elemList dá o nome dos elementos (para o cliente).
func elemList(es []int) []string {
	var out []string
	for _, e := range es {
		out = append(out, elemNames[e])
	}
	return out
}
