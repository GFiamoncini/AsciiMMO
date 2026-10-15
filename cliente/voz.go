package main

import "strings"

// Voz dos NPCs: a fala surge letra a letra (voiceCPS) e cada palavra é dita na
// "língua velha" (abertura_cena.go) quando aparece, no tom de quem fala. Vale
// para as conversas (E no NPC) e para as falas dos mercadores na loja.

const voiceCPS = 24.0 // letras por segundo das falas faladas

// Tom de cada um (Hz da voz); quem não está aqui fala com voz média.
var voicePitch = map[string]float64{
	"Eremita":                  92,
	"O Eremita":                92,
	"Mestre Oren":              108,
	"Morvan, o ferreiro":       82,
	"Ysolde, a herbalista":     190,
	"A Velha do Charco":        215,
	"O Eremita da Ilha":        98,
	"Mascate das Sombras":      135,
	"Velho Guardião de Vaelor": 90,
	"Durgo, o anão do fundo":   72,
}

type utterance struct {
	text   string
	start  float64 // ms
	spoken int
	pitch  float64
}

func pitchOf(who string) float64 {
	for k, v := range voicePitch {
		if strings.HasPrefix(who, k) || strings.HasPrefix(k, who) {
			return v
		}
	}
	return 120
}

// say começa uma fala nova de who (substitui a anterior).
func (g *Game) say(who, text string) {
	g.voice = &utterance{text: text, start: g.now(), pitch: pitchOf(who)}
}

// typed é quantas letras da fala atual já apareceram (e fala as palavras novas).
func (g *Game) typed() int {
	u := g.voice
	if u == nil {
		return 1 << 30
	}
	n := int((g.now() - u.start) / 1000 * voiceCPS)
	r := len([]rune(u.text))
	if n > r {
		n = r
	}
	old := oldLine(u.text)
	k := wordsTyped(u.text, n)
	for u.spoken < k && u.spoken < len(old) {
		w := old[u.spoken]
		u.spoken++
		if g.snd != nil && u.spoken > k-2 {
			g.snd.playWave(speakWord(w, u.pitch, u.spoken == len(old)), 0.8)
		}
	}
	return n
}

// voiceDone diz se a fala atual já apareceu inteira; complete a completa.
func (g *Game) voiceDone() bool {
	u := g.voice
	return u == nil || g.typed() >= len([]rune(u.text))
}

func (g *Game) voiceComplete() {
	if u := g.voice; u != nil {
		u.start = g.now() - float64(len([]rune(u.text)))/voiceCPS*1000 - 1
		u.spoken = len(oldLine(u.text)) // pular não fala tudo de uma vez
	}
}
