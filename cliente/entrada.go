package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"math"
	"strings"
)

var classInfo = [][2]string{{"Guerreiro", "espada · investida"}, {"Mago", "míssil · bola de fogo"}, {"Cavaleiro", "lança · bloqueio"},
	{"Paladino", "martelo · cura"}, {"Arqueiro", "flechas · rajada"}, {"Ladino", "adagas · passo sombrio"}, {"Necromante", "seta sombria · dreno"},
	{"Depravado", "pedaço de pau · até 3 escolas"}}

func (g *Game) drawJoin(dst *ebiten.Image) {
	cw, chh := float64(g.W)/g.ui, float64(g.H)/g.ui
	w := math.Min(520, cw*0.9)
	rowsN := len(g.chars)
	h := 560 + float64(max(1, rowsN))*34
	x, y := (cw-w)/2, math.Max(10, (chh-h)/2)
	g.panel(dst, x, y, w, h, nil)
	g.txt(dst, "AsciiMMO", x+28, y+18, cText, 22, true, alignLeft)
	g.txt(dst, "as terras de Midrath", x+w-28, y+26, cDim, 12, false, alignRight)

	// Personagens deste PC: carregar ou excluir.
	yy := y + 60
	g.txt(dst, "Seus personagens", x+28, yy, cGold, 13, true, alignLeft)
	yy += 22
	if len(g.chars) == 0 {
		msg := "Nenhum ainda neste PC."
		if g.charsMsg != "" {
			msg = g.charsMsg
		}
		g.txt(dst, msg, x+28, yy+6, cDimmer, 12, false, alignLeft)
		yy += 34
	}
	for _, ch := range g.chars {
		ch := ch
		g.rect(dst, x+28, yy, w-56, 30, color.RGBA{14, 12, 18, 255})
		cls := "?"
		if ch.Class >= 0 && ch.Class < len(classInfo) {
			cls = classInfo[ch.Class][0]
		}
		info := fmt.Sprintf("%s · nível %d · cap. %d", cls, ch.Level, min(ch.Chapter, 6))
		if ch.Online {
			info += " · conectado"
		}
		g.txt(dst, ch.Name, x+38, yy+7, cText, 14, true, alignLeft)
		g.txt(dst, info, x+38+g.textW(ch.Name, 14)*1.1+12, yy+8, cDim, 12, false, alignLeft)
		// Botões à direita: Carregar e Excluir (este pede confirmação).
		bx := x + w - 28 - 180
		g.stroke(dst, bx, yy+3, 84, 24, rgbaHex(0x7dff7d), 1)
		g.txt(dst, "Carregar", bx+42, yy+7, cGreen, 12, false, alignCenter)
		g.uiRects = append(g.uiRects, uiRect{x: bx, y: yy + 3, w: 84, h: 24, fn: func() {
			if g.c == nil {
				g.nameInput, g.pwInput, g.cls = ch.Name, "", ch.Class
				g.join()
			}
		}})
		del, dc := "Excluir", rgbaHex(0xd06060)
		if g.delConfirm == ch.Name {
			del, dc = "Confirmar?", rgbaHex(0xff4040)
		}
		g.stroke(dst, bx+92, yy+3, 88, 24, dc, 1)
		g.txt(dst, del, bx+136, yy+7, dc, 12, false, alignCenter)
		g.uiRects = append(g.uiRects, uiRect{x: bx + 92, y: yy + 3, w: 88, h: 24, fn: func() {
			if g.delConfirm != ch.Name {
				g.delConfirm = ch.Name
				return
			}
			g.delConfirm = ""
			go func() {
				if err := deleteChar(g.server, ch.Name); err != nil {
					g.joinMsg = "Não consegui excluir: " + err.Error()
				} else {
					g.joinMsg = ch.Name + " foi para a lixeira do servidor."
				}
				g.loadChars()
			}()
		}})
		yy += 34
	}

	// Novo personagem (ou um seu de outro aparelho, com a senha).
	yy += 10
	g.txt(dst, "Novo personagem — ou um seu de outro aparelho, com a senha", x+28, yy, cGold, 13, true, alignLeft)
	yy += 24
	field := func(label, val string, fx, fw float64, focus, secret bool, fn func()) {
		border := color.Color(rgbaHex(0x55555f))
		if focus {
			border = cGold
		}
		g.stroke(dst, fx, yy, fw, 32, border, 1)
		show := val
		if secret {
			show = strings.Repeat("•", len([]rune(val)))
		}
		if focus && int(g.now()/500)%2 == 0 {
			show += "_"
		}
		if val == "" && !focus {
			g.txt(dst, label, fx+10, yy+8, cDimmer, 14, false, alignLeft)
		} else {
			g.txt(dst, show, fx+10, yy+8, cText, 14, false, alignLeft)
		}
		g.uiRects = append(g.uiRects, uiRect{x: fx, y: yy, w: fw, h: 32, fn: fn})
	}
	half := (w - 56 - 10) / 2
	field("nome", g.nameInput, x+28, half, !g.focusPw, false, func() { g.focusPw = false })
	field("senha (opcional)", g.pwInput, x+38+half, half, g.focusPw, true, func() { g.focusPw = true })
	yy += 42
	bw := (w - 56 - 8) / 2
	for i, c := range classInfo {
		bx, by := x+28+float64(i%2)*(bw+8), yy+float64(i/2)*46
		border, tc := color.Color(rgbaHex(0xcfcfcf)), color.Color(cText)
		if i == g.cls {
			border, tc = cGold, cGold
		}
		g.rect(dst, bx, by, bw, 40, color.Black)
		g.stroke(dst, bx, by, bw, 40, border, 1)
		g.txt(dst, c[0], bx+10, by+5, tc, 13, false, alignLeft)
		g.txt(dst, c[1], bx+10, by+22, cDim, 11, false, alignLeft)
		i := i
		g.uiRects = append(g.uiRects, uiRect{x: bx, y: by, w: bw, h: 40, fn: func() { g.cls = i }})
	}
	ey := yy + 4*46
	g.rect(dst, x+28, ey, w-56, 36, color.Black)
	g.stroke(dst, x+28, ey, w-56, 36, rgbaHex(0xcfcfcf), 1)
	g.txt(dst, "Entrar (Enter)", x+w/2, ey+9, cText, 15, false, alignCenter)
	g.uiRects = append(g.uiRects, uiRect{x: x + 28, y: ey, w: w - 56, h: 36, fn: func() {
		if g.c == nil {
			g.join()
		}
	}})
	for i, l := range wrap(g.joinMsg, int((w-56)/g.textW("M", 12))) {
		g.txt(dst, l, x+28, ey+48+float64(i)*16, cDim, 12, false, alignLeft)
	}
	g.txt(dst, "A senha deixa abrir o mesmo personagem no navegador e no PC. Tab troca de campo.", x+28, y+h-42, cDimmer, 11, false, alignLeft)
	g.txt(dst, "Servidor: "+g.server+"   ·   F11 tela cheia", x+28, y+h-24, cDimmer, 11, false, alignLeft)
}
