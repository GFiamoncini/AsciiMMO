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

// Descrição de cada classe no painel de detalhes da tela de entrada.
var classLore = []string{
	"Corpo a corpo, aguenta muito. Investida, redemoinho e o ramo de SANGUE.",
	"Magia de longe: fogo, gelo, raio e arcano. Frágil, mas devastador.",
	"Lança e escudo: bloqueia, avança e protege quem está perto.",
	"Martelo e luz: cura, consagra o chão e julga os mortos.",
	"Flechas de longe, rajadas e armadilhas; o olho da águia não erra.",
	"Adagas e sombras: some no escuro, envenena e golpeia pelas costas.",
	"Seta sombria, dreno e servos: os mortos lutam por você.",
	"Começa com um pedaço de pau e aprende de tudo: até 3 escolas.",
}

// drawJoin é a tela de entrada, no mesmo desenho das lojas: a cena do vale
// animada ao fundo, a moldura dourada com cantos +, duas abas (seus
// personagens e novo personagem) e, no novo, o painel de detalhes da classe.
func (g *Game) drawJoin(dst *ebiten.Image) {
	cw, chh := float64(g.W)/g.ui, float64(g.H)/g.ui
	if g.g != nil {
		g.drawScene(dst, 2, g.now()/1000)
		g.rect(dst, 0, 0, cw, chh, color.RGBA{0, 0, 0, 120})
	}
	w := math.Min(860, cw-40)
	h := math.Min(560, chh-30)
	x, y := (cw-w)/2, math.Max(10, (chh-h)/2)
	gold := rgbaHex(0xc08040)
	g.panel(dst, x, y, w, h, gold)
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, rgbaHex(0xe0a050), 14, true, alignCenter)
	}
	g.txt(dst, "-=< AsciiMMO >=-", x+w/2, y+12, rgbaHex(0xffd090), 20, true, alignCenter)
	g.txt(dst, "as terras de Midrath", x+w/2, y+38, cDim, 12, false, alignCenter)
	if g.joinTab == "" {
		g.joinTab = "new"
		if len(g.chars) > 0 {
			g.joinTab = "load"
		}
	}
	tabs := []struct{ key, label string }{{"load", fmt.Sprintf("Seus personagens (%d)", len(g.chars))}, {"new", "Novo personagem"}}
	for i, t := range tabs {
		tx := x + 16 + float64(i)*210
		on := g.joinTab == t.key
		g.rect(dst, tx, y+62, 200, 26, pickRGBA(on, color.RGBA{60, 40, 20, 255}, color.RGBA{16, 12, 10, 255}))
		g.stroke(dst, tx, y+62, 200, 26, pickRGBA(on || g.hover(tx, y+62, 200, 26), rgbaHex(0xe0a050), rgbaHex(0x4a3420)), 1)
		g.txt(dst, t.label, tx+100, y+67, pickC(on, cGold, cText), 13, on, alignCenter)
		key := t.key
		g.uiRects = append(g.uiRects, uiRect{x: tx, y: y + 62, w: 200, h: 26, fn: func() { g.joinTab = key }})
	}
	top := y + 100
	if g.joinTab == "load" {
		if len(g.chars) == 0 {
			msg := "Nenhum personagem neste PC ainda. Crie um na outra aba."
			if g.charsMsg != "" {
				msg = g.charsMsg
			}
			g.txt(dst, msg, x+20, top+6, cDimmer, 13, false, alignLeft)
		}
		for k, ch := range g.chars {
			ch := ch
			ry := top + float64(k)*56
			if ry > y+h-110 {
				break
			}
			hov := g.hover(x+16, ry, w-32, 50)
			g.rect(dst, x+16, ry, w-32, 50, pickRGBA(hov, color.RGBA{36, 26, 16, 255}, color.RGBA{16, 12, 10, 255}))
			g.stroke(dst, x+16, ry, w-32, 50, pickRGBA(hov, rgbaHex(0xe0a050), rgbaHex(0x4a3420)), 1)
			if ch.Class >= 0 {
				g.drawPortrait(dst, ch.Class, x+22, ry+5, 40)
			}
			if k < 9 {
				g.keyTile(dst, fmt.Sprint(k+1), x+70, ry+15)
			}
			cls := "?"
			if ch.Class >= 0 && ch.Class < len(classInfo) {
				cls = classInfo[ch.Class][0]
			}
			info := fmt.Sprintf("%s · nível %d · capítulo %s", cls, ch.Level, roman(min(ch.Chapter, 6)))
			if ch.Online {
				info += " · conectado agora"
			}
			g.txt(dst, ch.Name, x+100, ry+7, rgbaHex(0xffd090), 15, true, alignLeft)
			g.txt(dst, info, x+100, ry+28, cDim, 12, false, alignLeft)
			bx := x + w - 16 - 200
			load := func() {
				if g.c == nil {
					g.nameInput, g.pwInput, g.cls = ch.Name, "", ch.Class
					g.join()
				}
			}
			g.rect(dst, bx, ry+11, 96, 28, color.RGBA{30, 50, 24, 255})
			g.stroke(dst, bx, ry+11, 96, 28, rgbaHex(0x7dff7d), 1)
			g.txt(dst, "Carregar", bx+48, ry+17, cGreen, 13, true, alignCenter)
			g.uiRects = append(g.uiRects, uiRect{x: bx, y: ry + 11, w: 96, h: 28, fn: load})
			del, dc := "Excluir", rgbaHex(0xd06060)
			if g.delConfirm == ch.Name {
				del, dc = "Confirmar?", rgbaHex(0xff4040)
			}
			g.stroke(dst, bx+104, ry+11, 96, 28, dc, 1)
			g.txt(dst, del, bx+152, ry+17, dc, 12, false, alignCenter)
			g.uiRects = append(g.uiRects, uiRect{x: bx + 104, y: ry + 11, w: 96, h: 28, fn: func() {
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
		}
	} else {
		yy := top
		g.txt(dst, "Nome e senha (a senha abre o mesmo personagem em outro aparelho) · Tab troca o campo", x+20, yy, cDim, 11, false, alignLeft)
		yy += 18
		field := func(label, val string, fx, fw float64, focus, secret bool, fn func()) {
			border := color.Color(rgbaHex(0x5a4a2a))
			if focus {
				border = cGold
			}
			g.rect(dst, fx, yy, fw, 32, color.RGBA{16, 12, 10, 255})
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
		lw := w*0.56 - 24
		half := (lw - 10) / 2
		field("nome", g.nameInput, x+16, half, !g.focusPw, false, func() { g.focusPw = false })
		field("senha (opcional)", g.pwInput, x+26+half, half, g.focusPw, true, func() { g.focusPw = true })
		yy += 44
		// Classes em grade, com retrato; ← → ↑ ↓ ou F1–F8 escolhem.
		bw := (lw - 8) / 2
		for i, c := range classInfo {
			bx, by := x+16+float64(i%2)*(bw+8), yy+float64(i/2)*50
			on := i == g.cls
			hov := g.hover(bx, by, bw, 44)
			g.rect(dst, bx, by, bw, 44, pickRGBA(on, color.RGBA{48, 34, 18, 255}, color.RGBA{16, 12, 10, 255}))
			g.stroke(dst, bx, by, bw, 44, pickRGBA(on || hov, rgbaHex(0xe0a050), rgbaHex(0x4a3420)), 1)
			g.drawPortrait(dst, i, bx+4, by+4, 36)
			g.txt(dst, c[0], bx+48, by+6, pickC(on, cGold, cText), 13, on, alignLeft)
			g.txt(dst, g.fit(c[1], 11, bw-54), bx+48, by+24, cDim, 11, false, alignLeft)
			i := i
			g.uiRects = append(g.uiRects, uiRect{x: bx, y: by, w: bw, h: 44, fn: func() { g.cls = i }})
		}
		// Painel de detalhes da classe escolhida (como o detalhe do item na loja).
		dx, dw := x+16+lw+16, w-lw-48
		dh := 4*50 + 44 - 6.0
		g.rect(dst, dx, top+18, dw, dh+44, color.RGBA{12, 10, 14, 255})
		g.stroke(dst, dx, top+18, dw, dh+44, rgbaHex(0x4a3420), 1)
		c := clampi(g.cls, 0, len(classInfo)-1)
		g.drawPortrait(dst, c, dx+12, top+30, 64)
		g.txt(dst, classInfo[c][0], dx+88, top+36, rgbaHex(0xffd090), 17, true, alignLeft)
		g.txt(dst, classInfo[c][1], dx+88, top+60, cDim, 12, false, alignLeft)
		ly := top + 110
		if c < len(classLore) {
			for _, l := range g.wrapPx(classLore[c], 13, dw-24) {
				g.txt(dst, l, dx+12, ly, cText, 13, false, alignLeft)
				ly += 19
			}
		}
		// Entrar.
		ey := yy + 4*50 + 6
		hov := g.hover(x+16, ey, lw, 38)
		g.rect(dst, x+16, ey, lw, 38, color.RGBA{50, 34, 18, 255})
		g.stroke(dst, x+16, ey, lw, 38, pickRGBA(hov, rgbaHex(0xffd34a), rgbaHex(0xc08040)), 1)
		g.txt(dst, "Entrar em Midrath (Enter)", x+16+lw/2, ey+10, pickC(hov, cGold, cText), 15, true, alignCenter)
		g.uiRects = append(g.uiRects, uiRect{x: x + 16, y: ey, w: lw, h: 38, fn: func() {
			if g.c == nil {
				g.join()
			}
		}})
	}
	for i, l := range g.wrapPx(g.joinMsg, 12, w-40) {
		g.txt(dst, l, x+20, y+h-60+float64(i)*16, rgbaHex(0xe0b070), 12, false, alignLeft)
	}
	g.txt(dst, "Servidor: "+g.server+"   ·   F11 tela cheia", x+20, y+h-24, cDimmer, 11, false, alignLeft)
	// Sair do jogo (fecha a janela).
	qw := 130.0
	qx, qy := x+w-16-qw, y+h-34
	hov := g.hover(qx, qy, qw, 26)
	g.rect(dst, qx, qy, qw, 26, color.RGBA{40, 16, 16, 255})
	g.stroke(dst, qx, qy, qw, 26, pickRGBA(hov, rgbaHex(0xff6060), rgbaHex(0x8a3030)), 1)
	g.txt(dst, "Sair do jogo", qx+qw/2, qy+5, pickC(hov, rgbaHex(0xff8080), cText), 13, false, alignCenter)
	g.uiRects = append(g.uiRects, uiRect{x: qx, y: qy, w: qw, h: 26, fn: func() { g.quit = true }})
}
