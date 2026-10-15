package main

import (
	"time"

	"asciimmo/proto"
)

// Prólogo: o Vale do Despertar, a sudoeste de Cinzalva. Todo personagem novo
// acorda aqui. O Eremita conta a história (Tiamath acordou e o fogo dela
// chama os mortos) e explica o básico; bonecos de palha servem de treino; a
// fogueira ensina a descansar; a trilha a leste leva à vila. O prólogo tem as
// suas etapas, fora da corrente de missões (Player.Intro).

const (
	introX, introY = villageX - 330, villageY + 300
	introR         = 70 // raio do vale (elipse 70 × 46)
	introGateX     = introX + introR
	introGateY     = introY
)

// Etapas do prólogo. qTalk = falar com o Eremita; qRest = descansar na fogueira.
const (
	qTalk = 10 + iota
	qRest
)

var prologue = []step{
	{"Fale com o Eremita (E perto dele)", qTalk, nil, 1},
	{"Derrote 3 bonecos de treino", qKill, []int{mobDummy}, 3},
	{"Descanse na fogueira do vale (E)", qRest, nil, 1},
	{"Siga a trilha a leste até Cinzalva", qVisit, []int{1}, 1},
}

// npc é alguém com quem se conversa (E): nome, lugar e o que diz.
type npc struct {
	Name  string
	X, Y  int
	Lines []string
}

var hermit = npc{"O Eremita", introX - 18, introY - 8, []string{
	"Acordou, enfim. Faz três noites que a fumaça não deixa o sol nascer direito.",
	"Lá no alto das montanhas do noroeste, Tiamath — a mãe dos dragões, de cinco cabeças — abriu os olhos.",
	"O fogo dela chama os mortos. O cemitério de Cinzalva já não descansa, e cada noite a vila arde um pouco mais.",
	"Você não vai durar lá fora sem saber o básico. WASD anda, Shift corre. Espaço ou o clique golpeiam.",
	"Os números de 1 a 5 usam os golpes da sua barra. K abre a teia de habilidades: cada nível dá um ponto para aprender algo novo.",
	"I mostra a mochila e o que você veste. Itens se usam com um clique; o botão direito joga fora.",
	"Fogueiras curam e guardam o lugar onde você volta se cair. E descansa nelas, e V leva às que você já achou.",
	"Ali estão os bonecos de palha. Mostre que sabe bater. Depois descanse na fogueira e siga a trilha a leste, até a vila.",
	"No meio de Midrath ainda está de pé a Árvore Divina. Dizem que as raízes dela descem até as Profundezas — e que gente faminta ronda a sombra dela. Não chegue perto cedo demais.",
	"Quando estiver pronto... o caminho até Tiamath passa por seis terras. Vá.",
}}

// curIntro diz se o jogador ainda está no prólogo.
func (p *Player) inPrologue() bool { return p.Intro < len(prologue) }

// buildIntro ergue o vale: chão limpo, mureta de pedra em elipse com a
// passagem a leste, a fogueira, o baú, os bonecos e as placas.
func (wd *World) buildIntro(seed uint32) {
	wd.clearArea(introX-introR-8, introY-60, introX+introR+8, introY+56, seed)
	// Mureta baixa de pedra (bloqueia), aberta a leste.
	for k := 0; k < 720; k++ {
		a := float64(k) / 720 * 2 * 3.14159265
		x, y := introX+int(float64(introR)*cos(a)), introY+int(46*sin(a))
		if cos(a) > 0.93 { // a passagem
			continue
		}
		for dy := -1; dy <= 0; dy++ {
			ch := pick("#H8&", hash2(x, y+dy, seed+1200))
			wd.put(x, y+dy, ch, dim(0x8a8a9a, 0.75+0.3*rnd01(x, y+dy, seed+1201)), true, false)
		}
	}
	// Placas que ensinam o caminho.
	wd.signText(introX-30, introY+22, "ESP / CLIQUE: GOLPE", 0xc8b48c, seed+1202)
	wd.signText(introX+20, introY+30, "K: HABILIDADES  I: MOCHILA", 0xc8b48c, seed+1203)
	wd.signText(introX+44, introY-16, "E: INTERAGIR  V: VIAGEM", 0xc8b48c, seed+1204)
	wd.signpost(introGateX+10, introGateY+12, []string{"Vila"}, []float64{0}, seed+1205)
	// Bonecos de treino (inimigos que não atacam nem andam).
	for _, d := range [][2]int{{introX + 8, introY + 4}, {introX + 22, introY - 6}, {introX + 30, introY + 12}, {introX + 14, introY + 20}} {
		wd.spawns = append(wd.spawns, spawnPt{d[0], d[1], mobDummy})
	}
}

// introSpawn é onde o personagem novo acorda: perto do Eremita.
func (wd *World) introSpawn() (int, int) {
	if x, y, ok := wd.freeNear(hermit.X+10, hermit.Y+6, 20); ok {
		return x, y
	}
	return introX, introY
}

// Talk é a conversa com o Eremita (o cliente mostra as falas).
func (wd *World) Talk(id, i int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok && i == 0 && dist(float64(p.X), float64(p.Y), float64(hermit.X), float64(hermit.Y)) <= 20 {
		wd.questEvent(p, qTalk, wd.clock())
	}
}

// finishPrologue fecha o prólogo: recompensa, a fogueira da vila vira a de
// renascer, e começa o Capítulo I.
func (wd *World) finishPrologue(p *Player, now time.Time) {
	p.Gold += 30
	p.give(itemPotion, 2)
	p.Bonfire = 0
	wd.pop(p, "Prólogo concluído — começa o Capítulo I", 0xffe066, float64(p.X), float64(p.Y)-40, now)
	wd.say(ChatLine{Text: p.Name + " deixou o Vale do Despertar."})
}

func (wd *World) npcInfo() []proto.NPC {
	return []proto.NPC{{Name: hermit.Name, X: hermit.X, Y: hermit.Y, Lines: hermit.Lines},
		{Name: master.Name, X: master.X, Y: master.Y, Lines: master.Lines, Teach: true}}
}

// prologueQuest é o prólogo no formato de missão, para o painel do cliente.
func prologueQuest(p *Player) *proto.Quest {
	q := &proto.Quest{N: 0, Of: len(quests), Title: "O Despertar", Text: "Você acorda num vale de cinzas. Um velho espera junto ao fogo.",
		Prog: p.QProg, Need: prologue[p.Intro].Need, Chapter: 0, Cur: p.Intro}
	for i, s := range prologue {
		qs := proto.QStep{Text: s.Text, Need: s.Need, Done: i < p.Intro}
		if i == p.Intro {
			qs.Prog = p.QProg
		}
		q.Steps = append(q.Steps, qs)
	}
	return q
}

// introBonfire é o índice da Fogueira do Despertar.
func (wd *World) introBonfire() int {
	for i, b := range wd.bonfires {
		if b.Name == "Fogueira do Despertar" {
			return i
		}
	}
	return 0
}
