package main

import (
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Gente com quem se conversa (o Eremita do prólogo): desenhada como um velho
// de manto e cajado; E perto dela abre a conversa, uma fala por vez.

var hermitStyle = bodyStyle{head: "hood", hair: pc(110, 100, 90), torso: pc(96, 84, 70), torso2: pc(150, 130, 90), robe: true,
	arm: pc(96, 84, 70), boots: pc(70, 56, 44), weapon: "staff", wlen: 14, orb: pc(255, 190, 90), seed: 98}

// O Mestre Oren: monge de manto cor de ferrugem, cabeça raspada, bastão.
var masterStyle = bodyStyle{head: "hair", hair: pc(196, 150, 110), skin: pc(196, 150, 110), torso: pc(170, 80, 40), torso2: pc(220, 160, 60), robe: true,
	arm: pc(196, 150, 110), boots: pc(90, 60, 40), weapon: "staff", wlen: 16, seed: 77}

func npcStyle(n proto.NPC) *bodyStyle {
	if n.Teach {
		return &masterStyle
	}
	return &hermitStyle
}

func (g *Game) nearNPC(me *proto.Player) int {
	for i, n := range g.info.NPCs {
		if math.Hypot(float64(n.X-me.X), float64(n.Y-me.Y)) <= 20 {
			return i
		}
	}
	return -1
}

func (g *Game) drawNPCs(me *proto.Player, T float64) {
	for i, n := range g.info.NPCs {
		face := 1
		if me.X < n.X {
			face = -1
		}
		g.op(func() {
			g.drawBody(bodyDraw{style: npcStyle(n), x: n.X, y: n.Y, T: T, hf: 1, face: face, tint: white, lit: g.mode == modeMix})
		})
		g.light(float64(n.X), float64(n.Y)-10, 26, 0.6, warm, col{60, 40, 10})
		tag := n.Name
		if g.cur.Quest != nil && g.cur.Quest.Chapter == 0 && g.cur.Quest.Cur == 0 && i == 0 {
			tag = "! " + n.Name // o prólogo pede para falar com ele
		}
		g.labels = append(g.labels, label{x: float64(n.X), y: float64(n.Y) - 31*charScale, text: tag, color: col{255, 220, 150}})
	}
}

// openTalk abre a conversa com o NPC i e avisa o servidor.
func (g *Game) openTalk(i int) {
	if g.info.NPCs[i].Teach && g.metMaster { // já conversou: vai direto ao terreiro
		g.openDojo()
		return
	}
	g.talkNPC, g.talkLine = i, 0
	g.send(proto.In{T: "talk", I: i})
}

// updateTalk passa as falas (E, Espaço, Enter, clique) e fecha com Esc. true
// enquanto a conversa está aberta (ela toma as teclas).
func (g *Game) updateTalk() bool {
	if g.talkNPC < 0 {
		return false
	}
	just := func(k ebiten.Key) bool { return justKey(k) }
	switch {
	case just(ebiten.KeyEscape):
		g.talkNPC = -1
	case just(ebiten.KeyE), just(ebiten.KeySpace), just(ebiten.KeyEnter), justClick():
		g.talkLine++
		if n := g.info.NPCs[g.talkNPC]; g.talkLine >= len(n.Lines) {
			g.talkNPC = -1
			if n.Teach {
				g.metMaster = true
				g.openDojo()
			}
		}
	}
	return true
}

func (g *Game) drawTalk(dst *ebiten.Image, cw, chh float64) {
	if g.talkNPC < 0 || g.talkNPC >= len(g.info.NPCs) {
		return
	}
	n := g.info.NPCs[g.talkNPC]
	w := math.Min(640, cw-rightW-40)
	lines := g.wrapPx(n.Lines[g.talkLine], 14, w-110)
	h := math.Max(96, float64(len(lines))*19+56)
	x, y := (cw-rightW-w)/2, chh-120-h
	g.panel(dst, x, y, w, h, rgbaHex(0xd6b050))
	pi := 1 // o velho de chapéu (retrato do mago)
	if n.Teach {
		pi = 0 // o mestre: retrato de lutador
	}
	g.drawPortrait(dst, pi, x+12, y+14, 64)
	g.txt(dst, n.Name, x+90, y+10, cGold, 14, true, alignLeft)
	for i, l := range lines {
		g.txt(dst, l, x+90, y+32+float64(i)*19, cText, 14, false, alignLeft)
	}
	more := "E / Espaço: continuar"
	if g.talkLine == len(n.Lines)-1 {
		more = "E / Espaço: encerrar"
	}
	g.txt(dst, more, x+w-12, y+h-20, cDim, 11, false, alignRight)
	g.txt(dst, itoaC(g.talkLine+1)+"/"+itoaC(len(n.Lines)), x+90, y+h-20, cDimmer, 11, false, alignLeft)
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

func itoaC(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func justKey(k ebiten.Key) bool { return inpututil.IsKeyJustPressed(k) }
func justClick() bool           { return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) }
