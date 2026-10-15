package main

import (
	"fmt"
	"image/color"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pausa e salvar no cliente (o servidor decide em servidor/pausa.go):
//   - Esc sem nada aberto, ou P, abre o menu de pausa: continuar, salvar agora,
//     opções, controles, voltar à tela de entrada;
//   - com a opção "Pausar ao abrir menus" (padrão ligada), mochila, loja, teia,
//     mapa grande, conversa etc. também pausam enquanto estiverem abertos;
//   - F5 salva na hora; cada gravação (automática, a cada 30 s, ao descansar
//     ou pedida) mostra "salvo" no canto.
// Sozinho no servidor, pausar para o mundo; com outros jogando, você fica
// parado e protegido, e o mundo segue para eles.

// wantPause diz se o jogador quer estar pausado agora.
func (g *Game) wantPause() bool {
	if g.pauseMenu {
		return true
	}
	if g.opts.NoPauseInMenus {
		return false
	}
	return g.openPanel != "" || g.bigMap || g.talkNPC >= 0 || g.introPage >= 0
}

// syncPause manda ao servidor quando a vontade de pausar muda.
func (g *Game) syncPause() {
	if g.c == nil || g.cur == nil {
		return
	}
	if me := g.cur.pmap[g.info.ID]; g.shopIdx == -2 && me != nil { // -painel loja
		if s := g.nearShop(me); s >= 0 {
			g.openShop(s)
			if g.shopWant == "compra" {
				g.shopView = "buy"
			}
		}
	}
	want := g.wantPause()
	if want != g.pauseSent {
		g.pauseSent = want
		v := 0
		if want {
			v = 1
		}
		g.send(proto.In{T: "pause", I: v})
		g.lastDir = "" // ao voltar, manda a direção de novo
	}
	if g.cur.Saves != g.savesSeen {
		if g.savesSeen > 0 || g.cur.Saves > 0 {
			g.savedAt = g.now()
		}
		g.savesSeen = g.cur.Saves
	}
}

// updatePauseMenu cuida das teclas com o menu aberto; true se ele tomou a entrada.
func (g *Game) updatePauseMenu() bool {
	if !g.pauseMenu {
		return false
	}
	switch {
	case justKey(ebiten.KeyEscape), justKey(ebiten.KeyP):
		g.pauseMenu = false
	case justKey(ebiten.KeyF5):
		g.saveNow()
	case justKey(ebiten.KeyO):
		g.pauseMenu, g.openPanel = false, "opts"
	case justKey(ebiten.KeyH):
		g.pauseMenu, g.showHelp = false, true
	case justClick():
		for _, r := range g.uiRects {
			if g.mouseX >= r.x && g.mouseX < r.x+r.w && g.mouseY >= r.y && g.mouseY < r.y+r.h && r.fn != nil {
				g.sfx("click", 0.4)
				r.fn()
				break
			}
		}
	}
	g.send(proto.In{T: "in"})
	g.lastDir = ""
	return true
}

func (g *Game) saveNow() {
	g.send(proto.In{T: "save"})
}

// leaveWorld volta à tela de entrada (desconecta; o servidor grava ao sair).
func (g *Game) leaveWorld() {
	g.pauseMenu = false
	if g.c != nil {
		g.c.close()
	}
}

// drawPause desenha o aviso de pausa e, se aberto, o menu.
func (g *Game) drawPause(dst *ebiten.Image, cw, chh float64) {
	now := g.now()
	// Aviso de gravação: um disquete em ASCII e "salvo", no canto de baixo.
	if t := now - g.savedAt; g.savedAt > 0 && t < 2500 {
		a := 1.0
		if t > 1800 {
			a = 1 - (t-1800)/700
		}
		c := color.RGBA{cb(150 * a), cb(220 * a), cb(150 * a), cb(255 * a)}
		x, y := cw-rightW-150, chh-30
		g.rect(dst, x-6, y-3, 132, 20, color.RGBA{0, 0, 0, cb(150 * a)})
		g.txt(dst, "[=] salvo", x, y, c, 13, true, alignLeft)
	}
	if g.cur == nil {
		return
	}
	if !g.pauseMenu {
		if g.cur.Paused && (g.openPanel != "" || g.bigMap) {
			// Pausado por janela aberta: um selo pequeno no alto.
			s := "pausado"
			if g.cur.Frozen {
				s = "pausado · o mundo parou"
			}
			tw := g.textW(s, 12)
			g.rect(dst, cw/2-tw/2-8, 84, tw+16, 18, color.RGBA{0, 0, 0, 170})
			g.txt(dst, s, cw/2, 86, rgbaHex(0xc8b890), 12, false, alignCenter)
		}
		return
	}
	g.rect(dst, 0, 0, cw, chh, color.RGBA{0, 0, 0, 150})
	w, h := 360.0, 330.0
	x, y := (cw-w)/2, (chh-h)/2
	g.panel(dst, x, y, w, h, cGold)
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, cGold, 14, true, alignCenter)
	}
	g.txt(dst, "-=< PAUSA >=-", cw/2, y+14, cGold, 18, true, alignCenter)
	sub := "O mundo parou."
	if !g.cur.Frozen {
		sub = "Há outros jogando: o mundo segue, você fica protegido."
	}
	for i, l := range g.wrapPx(sub, 12, w-30) {
		g.txt(dst, l, cw/2, y+42+float64(i)*16, cDim, 12, false, alignCenter)
	}
	rows := []struct {
		key, label string
		fn         func()
	}{
		{"Esc", "Continuar", func() { g.pauseMenu = false }},
		{"F5", "Salvar agora", func() { g.saveNow() }},
		{"O", "Opções", func() { g.pauseMenu = false; g.openPanel = "opts" }},
		{"H", "Controles", func() { g.pauseMenu = false; g.showHelp = true }},
		{"", "Voltar à tela de entrada", g.leaveWorld},
	}
	for i, r := range rows {
		ry := y + 84 + float64(i)*40
		hov := g.hover(x+20, ry, w-40, 32)
		bc := color.Color(cDim)
		if hov {
			bc = cGold
		}
		g.rect(dst, x+20, ry, w-40, 32, color.RGBA{14, 12, 20, 255})
		g.stroke(dst, x+20, ry, w-40, 32, bc, 1)
		if r.key != "" {
			g.keyTile(dst, r.key, x+32, ry+7)
		}
		tc := color.Color(cText)
		if hov {
			tc = cGold
		}
		g.txt(dst, r.label, x+90, ry+8, tc, 14, false, alignLeft)
		g.uiRects = append(g.uiRects, uiRect{x: x + 20, y: ry, w: w - 40, h: 32, fn: r.fn})
	}
	if t := now - g.savedAt; g.savedAt > 0 && t < 4000 {
		g.txt(dst, "Jogo salvo.", cw/2, y+h-24, rgbaHex(0x90e090), 12, false, alignCenter)
	} else {
		g.txt(dst, fmt.Sprintf("Salva sozinho a cada 30 s e ao descansar · %d nesta sessão", g.cur.Saves), cw/2, y+h-24, cDimmer, 11, false, alignCenter)
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}
