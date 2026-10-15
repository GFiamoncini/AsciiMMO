package main

import (
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Abertura da história: na primeira entrada de um personagem no prólogo, três
// telas em ASCII (Cinzalva ardendo, Tiamath de cinco cabeças, o vale onde ele
// acorda) com o texto surgindo letra a letra. Espaço passa, Esc pula. Visto
// uma vez por personagem (arquivo-marca na pasta de configuração).

type openingPage struct {
	art  string
	text string
	tint col
	fx   artFx // o que se move na tela (historia.go)
}

var openingPages = []openingPage{
	{art: `
                 (    )        (      )         (
          )     (  ) (   )   (   ) (     )   (   )
         (  )  ) (    )  ( )  (  )  ( )  ( ) ) (
            |>        _/\_        |>         _/\_
     __/\__ ||  ____ /    \  __   ||  /\__  /    \
    |  []  ||| |[] ||  []  ||[]|  ||| | []||| [] |
    |______|||_|___||______||__|__|||_|___|||____|_
    ^^^ ^^ ^^^^ ^ ^^^ ^^^ ^^^ ^^ ^^^ ^^^ ^^ ^^^ ^^`,
		text: "Midrath. Por três noites a fumaça cobriu o céu de Cinzalva. Os telhados ardem, e no cemitério os mortos já não descansam.",
		tint: col{255, 140, 60}, fx: artFx{particles: "embers", fire: true, pcol: col{255, 150, 60}, seed: 11}},
	{art: `
          __          __         __          __          __
         /oo\        /oo\       /oo\        /oo\        /oo\
         \  /_      _\  /       \  /       _\  /_      _\  /
          ) \ \    / /  (        )(        ) /  \ \    / / (
           \ \ \__/ /  / \______/  \______/ /    \ \__/ / /
            \ \____/  /                      \    \____/ /
             \_______/   T  I  A  M  A  T  H  \_________/
                    \__________________________/
                       /// ///       \\\ \\\`,
		text: "Nas montanhas do noroeste, Tiamath — a mãe dos dragões, de cinco cabeças — abriu os olhos. O fogo dela chama os mortos.",
		tint: col{220, 80, 70}, fx: artFx{particles: "embers", sway: 7, eyes: col{255, 210, 80}, pcol: col{255, 110, 50}, seed: 12}},
	{art: `
               .        *          .          *        .
          *         .        _/\_      .           *
                ,;,     .   (    )        ,;,
          ,;,  ;;;;;       (  )(  )      ;;;;;   ,;,
         ;;;;;  |||    ___   )(    ___    |||   ;;;;;
          |||   |||   /   \ (  )  /   \   |||    |||
        ~~~~~~~~~~~~~~~~~~~~ \/ ~~~~~~~~~~~~~~~~~~~~~~
             o_                 ___
            /|  ...            (___) <- o Eremita`,
		text: "E você acorda num vale de cinzas, sem lembrar do caminho até ali. Junto ao fogo, um velho espera.",
		tint: col{230, 200, 140}, fx: artFx{particles: "stars", water: col{90, 110, 150}, fire: true, pcol: col{230, 230, 255}, seed: 13}},
}

func openingMark(name string) string {
	return filepath.Join(configDir(), "abertura-"+strings.ToLower(name)+".visto")
}

// maybeOpening abre a abertura se o personagem está no começo do prólogo e
// ainda não a viu.
func (g *Game) maybeOpening(name string, chapter, cur int) {
	if g.introPage >= 0 || g.introDone || chapter != 0 || cur != 0 {
		return
	}
	g.introDone = true
	if _, err := os.Stat(openingMark(name)); err == nil {
		return
	}
	g.introPage, g.introAt = 0, 0
	g.introName = name
}

func (g *Game) closeOpening() {
	g.introPage = -1
	os.WriteFile(openingMark(g.introName), []byte("1"), 0o644)
}

// updateIntro avança a abertura; true enquanto ela está na tela.
func (g *Game) updateIntro() bool {
	if g.introPage < 0 {
		return false
	}
	g.introAt += 1.0 / 60
	switch {
	case justKey(ebiten.KeyEscape):
		g.closeOpening()
	case justKey(ebiten.KeySpace), justKey(ebiten.KeyEnter), justKey(ebiten.KeyE), justClick():
		pg := openingPages[g.introPage]
		if n := len([]rune(pg.text)); g.introAt*introCPS < float64(n) {
			g.introAt = float64(n) / introCPS // primeiro toque completa o texto
			break
		}
		g.introPage++
		g.introAt = 0
		g.sfx("warn", 0.5)
		if g.introPage >= len(openingPages) {
			g.closeOpening()
		}
	}
	return true
}

// introCPS: letras por segundo da legenda (no ritmo da narração).
const introCPS = 22.0

// wordsTyped diz quantas palavras do texto já começaram a aparecer.
func wordsTyped(text string, n int) int {
	r := []rune(text)
	w, in := 0, false
	for i := 0; i < n && i < len(r); i++ {
		sp := r[i] == ' '
		if !sp && !in {
			w++
		}
		in = !sp
	}
	return w
}

// narrate fala cada palavra da língua velha quando ela começa na legenda.
func (g *Game) narrate(text string, n int) {
	if g.introVoicePage != g.introPage {
		g.introVoicePage, g.introSpoken = g.introPage, 0
	}
	old := oldLine(text)
	k := wordsTyped(text, n)
	for g.introSpoken < k && g.introSpoken < len(old) {
		w := old[g.introSpoken]
		g.introSpoken++
		if g.snd != nil && g.introSpoken > k-2 { // depois de pular o texto, só a última
			g.snd.playWave(speakWord(w, 92, g.introSpoken == len(old)), 0.9)
		}
	}
}

func (g *Game) drawOpening(dst *ebiten.Image, cw, chh float64) {
	if g.introPage < 0 {
		return
	}
	pg := openingPages[g.introPage]
	fade := math.Min(1, g.introAt/0.8)
	// A cena, na grade do jogo (abertura_cena.go), escurecida no começo.
	g.drawScene(dst, g.introPage, g.introAt)
	g.rect(dst, 0, 0, cw, chh, color.RGBA{0, 0, 0, uint8(255 * (1 - fade))})
	// Faixa da legenda embaixo.
	g.rect(dst, 0, chh*0.72, cw, chh*0.28, color.RGBA{0, 0, 0, 170})
	n := int(g.introAt * introCPS)
	r := []rune(pg.text)
	if n > len(r) {
		n = len(r)
	}
	g.narrate(pg.text, n)
	tw := math.Min(760, cw-80)
	// A língua velha, apagada, em cima; a legenda em português embaixo.
	old := oldLine(pg.text)
	shown := wordsTyped(pg.text, n)
	oy := chh * 0.745
	for i, l := range g.wrapPx(strings.Join(old[:min(shown, len(old))], " "), 14, tw) {
		g.txt(dst, l, (cw-tw)/2, oy+float64(i)*20, rgbaHex(0xa08a60), 14, false, alignLeft)
	}
	for i, l := range g.wrapPx(string(r[:n]), 17, tw) {
		g.txt(dst, l, (cw-tw)/2, chh*0.82+float64(i)*26, cText, 17, false, alignLeft)
	}
	g.txt(dst, itoaC(g.introPage+1)+" / "+itoaC(len(openingPages))+"    ESP: continuar    ESC: pular", cw/2, chh-40, cDimmer, 12, false, alignCenter)
}

func (g *Game) pmapMe(s *proto.State) *proto.Player {
	for i := range s.Players {
		if s.Players[i].ID == g.info.ID {
			return &s.Players[i]
		}
	}
	return nil
}

// drawDeepWait cobre a tela enquanto as Profundezas baixam.
func (g *Game) drawDeepWait(dst *ebiten.Image, cw, chh float64) {
	if g.cur == nil || g.info == nil {
		return
	}
	me := g.cur.pmap[g.info.ID]
	if me == nil || !g.deepWaiting(me.Y) {
		return
	}
	g.rect(dst, 0, 0, cw, chh, rgbaHex(0x050406))
	dots := strings.Repeat(".", int(g.now()/400)%4)
	g.txt(dst, "Descendo às Profundezas"+dots, cw/2, chh/2-10, cGold, 16, true, alignCenter)
}
