package main

import (
	"fmt"
	"time"
)

// Missões com objetivo e subobjetivos: o objetivo é o título e o texto da
// missão; os subobjetivos são etapas em ordem (chegar a um lugar, descer à
// cripta, derrotar inimigos). A missão fecha quando a última etapa fecha.
// Missão sem etapas definidas vira uma etapa só, com os próprios campos.

type step struct {
	Text    string
	Kind    int
	Targets []int
	Need    int
}

// Índices das regiões (terrain.go): 0 Cidadela, 2 Cemitério, 3 Pico de
// Tiamath, 4 Selva, 6 Talássia, 7 Deserto, 8 Picos de Gelo, 12 Encruzilhada.
var questSteps = map[string][]step{
	"Ossos inquietos": {
		{"Chegue ao cemitério, ao norte de Cinzalva", qVisit, []int{2}, 1},
		{"Derrote os mortos que se levantam (esqueletos ou guerreiros)", qKill, []int{mobSkeleton, mobWarrior}, 5},
	},
	"A descida": {
		{"Volte ao cemitério e encontre a capela", qVisit, []int{2}, 1},
		{"Desça à cripta pela porta da capela", qEnter, nil, 1},
	},
	"O Lorde da Cripta": {
		{"Abra caminho pelos guerreiros da cripta", qKill, []int{mobWarrior}, 3},
		{"Destrua o Lorde da Cripta na última sala", qKill, []int{mobLord}, 1},
	},
	"Rumo à selva": {
		{"Atravesse a Encruzilhada Queimada", qVisit, []int{12}, 1},
		{"Entre na Selva de Marapá", qVisit, []int{4}, 1},
	},
	"Lâminas na mata": {
		{"Encontre a Cidadela Asura no coração da selva", qVisit, []int{0}, 1},
		{"Silencie as asuras que cantam", qKill, []int{mobAsura}, 6},
	},
	"A Rainha Asura": {
		{"Derrube a Rainha no pátio da cidadela", qKill, []int{mobQueen}, 1},
	},
	"Marés de sangue": {
		{"Siga para a costa de Talássia", qVisit, []int{6}, 1},
		{"Afogue os tritões", qKill, []int{mobTriton}, 6},
	},
	"O Tirano das Marés": {
		{"Enfrente o Tirano no coração de Talássia", qKill, []int{mobTyrant}, 1},
	},
	"Areias sem nome": {
		{"Chegue ao Deserto de Ankar", qVisit, []int{7}, 1},
		{"Abra caminho entre escaravelhos e múmias", qKill, []int{mobScarab, mobMummy}, 8},
	},
	"O Faraó": {
		{"Acorde e derrote o Faraó diante da pirâmide", qKill, []int{mobPharaoh}, 1},
	},
	"Presas no gelo": {
		{"Suba aos Picos de Gelo", qVisit, []int{8}, 1},
		{"Cace lobos do gelo e yetis", qKill, []int{mobIceWolf, mobYeti}, 8},
	},
	"O Gigante de Gelo": {
		{"Derrube o Gigante entre os blocos de gelo", qKill, []int{mobFrostGiant}, 1},
	},
	"A subida": {
		{"Atravesse a Encruzilhada rumo às montanhas", qVisit, []int{12}, 1},
		{"Chegue ao Pico de Tiamath", qVisit, []int{3}, 1},
	},
	"Tiamath": {
		{"Enfrente Tiamath no topo do mundo", qKill, []int{killDragon}, 1},
	},
}

// stepsOf devolve as etapas da missão.
func stepsOf(q quest) []step {
	if s, ok := questSteps[q.Title]; ok {
		return s
	}
	return []step{{q.Text, q.Kind, q.Targets, q.Need}}
}

// curStep é a etapa em que o jogador está na missão principal.
func (p *Player) curStep() (step, bool) {
	if p.inPrologue() {
		return prologue[p.Intro], true
	}
	if p.Quest >= len(quests) {
		return step{}, false
	}
	s := stepsOf(quests[p.Quest])
	if p.QStep >= len(s) {
		return step{}, false
	}
	return s[p.QStep], true
}

// advance soma n à etapa atual; fechada a etapa, passa à próxima; fechada a
// última, conclui a missão.
func (wd *World) advance(p *Player, n int, now time.Time) {
	if p.inPrologue() {
		s := prologue[p.Intro]
		if p.QProg += n; p.QProg < s.Need {
			return
		}
		p.Intro, p.QProg = p.Intro+1, 0
		if p.inPrologue() {
			wd.pop(p, "Objetivo concluído: "+s.Text, 0xc8e0a0, float64(p.X), float64(p.Y)-40, now)
			wd.catchStep(p, now)
		} else {
			wd.finishPrologue(p, now)
		}
		return
	}
	if p.Quest >= len(quests) {
		return
	}
	q := quests[p.Quest]
	steps := stepsOf(q)
	if p.QStep >= len(steps) {
		p.QStep = len(steps) - 1
	}
	s := steps[p.QStep]
	if p.QProg += n; p.QProg < s.Need {
		return
	}
	p.QStep, p.QProg = p.QStep+1, 0
	if p.QStep < len(steps) {
		wd.pop(p, "Objetivo concluído: "+s.Text, 0xc8e0a0, float64(p.X), float64(p.Y)-40, now)
		wd.catchStep(p, now)
		return
	}
	wd.reward(p, q, now)
	wd.say(ChatLine{Text: p.Name + " concluiu a missão \"" + q.Title + "\"."})
	p.Quest, p.QProg, p.QStep = p.Quest+1, 0, 0
	wd.catchStep(p, now)
}

// catchStep fecha na hora a etapa nova se ela já está cumprida: o jogador já
// está na região pedida, ou já derrubou o chefe pedido.
func (wd *World) catchStep(p *Player, now time.Time) {
	s, ok := p.curStep()
	if !ok {
		return
	}
	switch {
	case s.Kind == qVisit && inRegion(p, s.Targets[0]):
		wd.advance(p, s.Need-p.QProg, now)
	case s.Kind == qKill && s.Need == 1:
		for _, k := range s.Targets {
			if p.slew(k) {
				wd.say(ChatLine{Text: fmt.Sprintf("%s já tinha cumprido: %s.", p.Name, s.Text)})
				wd.advance(p, 1, now)
				return
			}
		}
	}
}

func inRegion(p *Player, r int) bool {
	g := regions[r]
	return dist(float64(p.X), float64(p.Y), float64(g.X), float64(g.Y)) <= float64(g.R)
}

// credit conta um abate para todo jogador por perto cuja etapa atual (ou
// secundária ativa) pede esse tipo.
func (wd *World) credit(kind int, x, y float64, now time.Time) {
	for _, p := range wd.near(x, y) {
		wd.catchStep(p, now) // já está na região pedida? fecha antes de contar o abate
		if s, ok := p.curStep(); ok && s.Kind == qKill && has(s.Targets, kind) {
			wd.advance(p, 1, now)
		}
		for i, q := range sideQuests {
			if p.SideDone&(1<<i) != 0 || p.chapterOf() < q.Chapter || !has(q.Targets, kind) {
				continue
			}
			wd.creditSide(p, i, now)
		}
	}
}

// questEvent fecha etapas de entrar na cripta ou de chegar a uma região.
func (wd *World) questEvent(p *Player, kind int, now time.Time) {
	s, ok := p.curStep()
	if !ok || s.Kind != kind {
		return
	}
	if kind == qVisit && !inRegion(p, s.Targets[0]) {
		return
	}
	wd.advance(p, 1, now)
}
