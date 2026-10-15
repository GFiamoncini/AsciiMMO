package main

import (
	"fmt"
	"image/color"
	"strings"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Doma (servidor/doma.go): com a tentativa aberta, o painel mostra as setas
// que o bicho pede e o tempo que resta; o jogador repete com as setas ou
// WASD. Errar uma seta já manda (e o bicho foge); completar manda a sequência.

var tameKeys = []struct {
	a, b ebiten.Key
	c    byte
}{{ebiten.KeyArrowUp, ebiten.KeyW, 'U'}, {ebiten.KeyArrowDown, ebiten.KeyS, 'D'}, {ebiten.KeyArrowLeft, ebiten.KeyA, 'L'}, {ebiten.KeyArrowRight, ebiten.KeyD, 'R'}}

// updateTame cuida das teclas durante a doma; true enquanto ela está aberta.
func (g *Game) updateTame() bool {
	t := g.cur.Tame
	if t == nil {
		g.tameBuf, g.tameSent = "", false
		return false
	}
	if g.tameSent {
		return true // esperando a resposta do servidor
	}
	if t.Left <= 0 { // o tempo acabou
		g.send(proto.In{T: "tame", Text: g.tameBuf + "-"})
		g.tameSent = true
		return true
	}
	if justKey(ebiten.KeyEscape) {
		g.send(proto.In{T: "tame", Text: "-"})
		g.tameSent = true
		return true
	}
	for _, k := range tameKeys {
		if !justKey(k.a) && !justKey(k.b) {
			continue
		}
		g.tameBuf += string(k.c)
		ok := strings.HasPrefix(t.Seq, g.tameBuf)
		if ok {
			g.sfx("click", 0.6)
		}
		if !ok || len(g.tameBuf) >= len(t.Seq) {
			g.send(proto.In{T: "tame", Text: g.tameBuf})
			g.tameSent = true
			if ok {
				g.sfx("horse", 0.6)
			} else {
				g.sfx("hurt", 0.4)
			}
		}
		break
	}
	return true
}

func (g *Game) drawTame(dst *ebiten.Image, cw, chh float64) {
	t := g.cur.Tame
	if t == nil {
		return
	}
	n := len(t.Seq)
	cell := 46.0
	w := float64(n)*(cell+8) + 40
	w = max(w, 380)
	h := 150.0
	x, y := (cw-w)/2, chh-h-120
	g.panel(dst, x, y, w, h, rgbaHex(0xc08040))
	g.txt(dst, "Domando: "+t.Name, cw/2, y+10, rgbaHex(0xffd090), 15, true, alignCenter)
	g.txt(dst, "repita as setas (ou WASD) antes do tempo acabar · Esc desiste", cw/2, y+30, cDim, 11, false, alignCenter)
	sx := cw/2 - float64(n)*(cell+8)/2 + 4
	arrows := map[byte]string{'U': "↑", 'D': "↓", 'L': "←", 'R': "→"}
	for i := 0; i < n; i++ {
		bx := sx + float64(i)*(cell+8)
		bg, fg := color.RGBA{16, 12, 10, 255}, color.Color(cText)
		switch {
		case i < len(g.tameBuf) && g.tameBuf[i] == t.Seq[i]:
			bg, fg = color.RGBA{40, 70, 30, 255}, cGreen
		case i < len(g.tameBuf):
			bg, fg = color.RGBA{80, 24, 24, 255}, rgbaHex(0xff8080)
		case i == len(g.tameBuf):
			bg = color.RGBA{50, 36, 20, 255}
			fg = cGold
		}
		g.rect(dst, bx, y+52, cell, cell, bg)
		g.stroke(dst, bx, y+52, cell, cell, rgbaHex(0x8a6440), 1)
		g.txt(dst, arrows[t.Seq[i]], bx+cell/2, y+60, fg, 26, true, alignCenter)
	}
	// Tempo que falta.
	k := clamp(t.Left/8, 0, 1)
	g.rect(dst, x+20, y+h-26, w-40, 8, color.RGBA{40, 30, 20, 255})
	g.rect(dst, x+20, y+h-26, (w-40)*k, 8, color.RGBA{220, 170, 70, 255})
	g.txt(dst, fmt.Sprintf("%.1f s", t.Left), x+w-20, y+h-42, cDim, 11, false, alignRight)
}
