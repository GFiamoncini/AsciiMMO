package main

import (
	"asciimmo/proto"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"math"
	"strings"
)

const (
	alignLeft = iota
	alignCenter
	alignRight
)

var (
	hpCol   = rgbaHex(0xe04848)
	mpCol   = rgbaHex(0x4a8cff)
	stCol   = rgbaHex(0x58d058)
	cText   = rgbaHex(0xe6e6e6)
	cDim    = rgbaHex(0x8a8a96)
	cDimmer = rgbaHex(0x6a6a76)
	cGold   = rgbaHex(0xffe066)
	cGreen  = rgbaHex(0x7dff7d)
	rarCols = []color.RGBA{rgbaHex(0xc8c8d0), rgbaHex(0x6aa8ff), rgbaHex(0xc880ff), rgbaHex(0xffb040)}
	rarName = []string{"comum", "raro", "épico", "lendário"}
	kindNam = []string{"consumível", "arma", "armadura", "elmo", "botas", "amuleto", "material", "erva", "dinheiro", "pergaminho", "chave", "mochila", "ferramenta", "anel"}
	wIcons  = []string{"/", "'", "|", "T", "?", ")", "W"}
)

// drawLabels segue a posição exata de cada personagem: os nomes deslizam e
// podem se sobrepor, como no vídeo de referência.
func (g *Game) drawLabels(dst *ebiten.Image, camL, camT float64) {
	cs := float64(g.cs)
	fs := clamp(cs*1.7, 10*g.ui, 16*g.ui)
	f := face(fs, true)
	for _, l := range g.labels {
		x := (l.x - camL) * cs
		bar := 0.0
		if l.hasBar {
			bar = math.Max(4*g.ui, fs*0.36)
		}
		y := (l.y-camT)*cs - bar - 2 - fs
		op := &text.DrawOptions{}
		op.PrimaryAlign = text.AlignCenter
		for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			op.GeoM.Reset()
			op.GeoM.Translate(x+d[0]*g.ui, y+d[1]*g.ui)
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 190})
			text.Draw(dst, l.text, f, op)
		}
		op.GeoM.Reset()
		op.GeoM.Translate(x, y)
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(l.color.rgba())
		text.Draw(dst, l.text, f, op)
		if l.skull || l.lv != "" {
			lx := x + text.Advance(l.text, f)/2 + 4*g.ui
			if l.skull {
				g.drawIcon(dst, "skull", 1, lx/g.ui, (y+fs*0.05)/g.ui, fs/g.ui, false)
			} else {
				op.PrimaryAlign = text.AlignStart
				for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					op.GeoM.Reset()
					op.GeoM.Translate(lx+d[0]*g.ui, y+d[1]*g.ui)
					op.ColorScale.Reset()
					op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 190})
					text.Draw(dst, l.lv, f, op)
				}
				op.GeoM.Reset()
				op.GeoM.Translate(lx, y)
				op.ColorScale.Reset()
				op.ColorScale.ScaleWithColor(l.lvCol.rgba())
				text.Draw(dst, l.lv, f, op)
			}
		}
		if l.hasBar && l.mhp > 0 {
			// Barra de vida: mais larga para quem tem mais vida; a cor cai do
			// verde ao vermelho conforme esvazia.
			w := clamp(math.Sqrt(float64(l.mhp))*4.5, 34, 160) * g.ui
			k := clamp(float64(l.hp)/float64(l.mhp), 0, 1)
			by := y + fs + 1
			vector.FillRect(dst, float32(x-w/2-g.ui), float32(by), float32(w+2*g.ui), float32(bar+2*g.ui), color.Black, false)
			vector.FillRect(dst, float32(x-w/2), float32(by+g.ui), float32(w), float32(bar), rgbaHex(0x3a1010), false)
			c := rgbaHex(0xe04838)
			if k > 0.5 {
				c = rgbaHex(0x58d058)
			} else if k > 0.25 {
				c = rgbaHex(0xe0c040)
			}
			vector.FillRect(dst, float32(x-w/2), float32(by+g.ui), float32(w*k), float32(bar), c, false)
		}
	}
}

func (g *Game) drawHUD(dst *ebiten.Image, me *proto.Player) {
	cw, chh := float64(g.W)/g.ui, float64(g.H)/g.ui
	ch, lh := g.textW("M", 13), 17.0
	cur := g.cur

	pw := 286.0
	y := 8 + g.drawStatusCard(dst, me, 8, 8) + 4
	g.drawModeIcon(dst, 8+286+6, 8) // modo de combate (pesca.go)

	// Status negativos e a proteção das ervas, com o tempo que falta.
	stc := []color.RGBA{rgbaHex(0x9aff6a), rgbaHex(0x8ac8ff), rgbaHex(0xffb050)}
	cure := []string{"antídoto", "raiz-brasa", "babosa"}
	type line struct {
		s string
		c color.Color
	}
	var sts []line
	for i, s := range cur.Sts {
		if s > 0 && i < len(g.info.Status) {
			sts = append(sts, line{fmt.Sprintf("%s %ds — use %s", g.info.Status[i], int(math.Ceil(s)), cure[i]), stc[i]})
		}
	}
	for i, s := range cur.Imm {
		if s > 0 && cur.Sts[i] == 0 {
			sts = append(sts, line{fmt.Sprintf("protegido (%s) %ds", cure[i], int(s)), cDim})
		}
	}
	if len(sts) > 0 {
		g.panel(dst, 8, y, pw, (float64(len(sts))+0.8)*lh, nil)
		for i, l := range sts {
			g.txt(dst, l.s, 8+ch, y+lh*(0.4+float64(i)), l.c, 13, false, alignLeft)
		}
		y += (float64(len(sts)) + 0.8) * lh
	}
	g.veilVignette(dst, me)
	// A borda da tela ganha a cor do mal que está agindo.
	for i, s := range cur.Sts {
		if s > 0 {
			g.vignette(dst, []color.RGBA{{90, 200, 40, 70}, {90, 150, 255, 70}, {255, 140, 40, 60}}[i])
			break
		}
	}

	// O que a tecla E faz aqui, escrito acima da cabeça.
	if hint := g.interactHint(me); hint != "" && g.meScreen != nil {
		hx := g.meScreen[0] / g.ui
		mult := 1.0
		if me.Ride > 0 {
			mult = 2
		}
		hy := g.meScreen[1]/g.ui - 64 - 4*float64(g.cs)/g.ui*mult
		tw := g.textW(hint, 13) * 1.05
		g.rect(dst, hx-tw/2-6, hy-3, tw+12, lh+4, color.RGBA{0, 0, 0, 180})
		g.txt(dst, hint, hx, hy, cGold, 13, true, alignCenter)
	}

	// Minimapa embaixo do card e dos status; depois os outros jogadores em fichas.
	y = g.drawMiniLeft(dst, me, 8, y, pw)
	g.drawPartyChips(dst, me, 8, y, pw)

	if g.opts.ShowFPS {
		g.txt(dst, fmt.Sprintf("FPS %d", g.fps), cw-12, chh-22, cGreen, 13, false, alignRight)
	}
	g.drawQuestArrow(dst, me, cw, chh)
	g.trackHeading(me)
	g.drawCompass(dst, me, cw)
	g.drawSealInput(dst, cw, chh)
	g.drawTame(dst, cw, chh)
	g.drawTalk(dst, cw, chh)
	g.drawRight(dst, me, cw, ch, lh)
	switch g.openPanel {
	case "sheet":
		g.drawSheet(dst, cw, chh, lh)
	case "shop":
		g.drawShop(dst, me, cw, chh, lh)
	case "fogo":
		g.drawBonfirePanel(dst, me, cw, chh)
	case "diario":
		g.drawJournal(dst, cw, chh)
	case "casa":
		g.drawHouse(dst, me, cw, chh)
	case "opts":
		g.drawOptions(dst, cw, chh, lh)
	case "tree":
		g.drawWeb(dst, me, cw, chh)
	case "ach":
		g.drawAchievements(dst, cw, chh)
	case "travel":
		g.drawTravel(dst, me, cw, chh)
	case "dojo":
		g.drawDojo(dst, me, cw, chh)
	case "gate":
		g.drawGate(dst, me, cw, chh)
	case "read":
		g.drawRead(dst, cw, chh)
	}
	if g.bigMap {
		// O mapa grande é uma janela à parte: escurece tudo atrás (nada de
		// menu aparecendo misturado) e tem zoom (roda, + e −) e arrasto
		// (botão direito) — mapa_zoom.go.
		g.rect(dst, 0, 0, cw, chh, color.RGBA{0, 0, 0, 215})
		full := g.mapSrc(me)
		k := math.Min((cw-80)/float64(full.w), (chh-150)/float64(full.h))
		mw, mh := float64(full.w)*k, float64(full.h)*k
		mx, my := (cw-mw)/2, (chh-mh)/2+10
		src := g.bigMapView(full, me, mx, my, mw, mh)
		g.panel(dst, mx-6, my-28, mw+12, mh+34, cGold)
		g.txt(dst, "Mapa (M fecha) · roda/+/− zoom, botão direito arrasta, 0 volta · "+g.goalText(me), mx, my-22, cGold, 13, false, alignLeft)
		g.drawMap(dst, me, mx, my, mw, mh, src, true)
		if g.mapZoom > 1.01 {
			g.txt(dst, fmt.Sprintf("zoom %.1fx", g.mapZoom), mx+mw-6, my-22, cDim, 12, false, alignRight)
		}
		g.bigMapClick(mx, my, mw, mh, src) // clicar: caminhar até lá (pesca.go)
	}

	// Barra de ações: golpe básico e os cinco atalhos (webtree.go).
	g.drawActionBar(dst, me, cw, chh, lh)

	// Chat: as falas dos jogadores aparecem por 12 s; os avisos do sistema
	// ("entrou no mundo", "obteve…") ficam no registro, aberto com L, que
	// conta quantos chegaram desde a última olhada. As dicas ficam no H.
	now := g.now()
	var show []logLine
	scrolled := 0
	if g.showLog {
		g.logScroll = clampi(g.logScroll, 0, max(0, len(g.chatLog)-16))
		scrolled = g.logScroll
		end := len(g.chatLog) - scrolled
		show = g.chatLog[max(0, end-16):end]
		g.logSeen = len(g.chatLog)
	} else {
		for _, l := range g.chatLog {
			if l.from != "" && now-l.at < 12000 {
				show = append(show, l)
			}
		}
		show = show[max(0, len(show)-5):]
	}
	cy := chh - 48 - float64(len(show))*lh
	if g.chatting {
		cy -= 20
	}
	if g.showLog {
		g.panel(dst, 4, cy-lh-6, math.Min(640, cw*0.5), float64(len(show)+1)*lh+10, cDim)
		head := "Registro (L fecha · ↑/↓, roda e PgUp/PgDn rolam)"
		if scrolled > 0 {
			head = fmt.Sprintf("Registro · %d mais novas abaixo (End volta ao fim)", scrolled)
		}
		g.txt(dst, head, 8+ch, cy-lh-2, cGold, 12, true, alignLeft)
	}
	for _, l := range show {
		who, body := "*", " "+l.text
		wc, bc := color.Color(cDim), color.Color(cDim)
		if l.from != "" {
			who, body, wc, bc = l.from, ": "+l.text, playerColor(l.color).rgba(), cText
		}
		if !g.showLog {
			g.rect(dst, 8, cy-2, g.textW(who+body, 13)+2*ch, lh, color.RGBA{0, 0, 0, 165})
		}
		g.txt(dst, who, 8+ch, cy, wc, 13, false, alignLeft)
		g.txt(dst, body, 8+ch+g.textW(who, 13), cy, bc, 13, false, alignLeft)
		cy += lh
	}
	if g.chatting {
		w := math.Min(560, cw*0.7)
		g.panel(dst, 8, chh-40, w, 30, cGold)
		g.txt(dst, string(g.chatBuf)+"_", 16, chh-33, cText, 15, false, alignLeft)
		return
	}
	unseen := 0
	for _, l := range g.chatLog[min(g.logSeen, len(g.chatLog)):] {
		if l.from == "" {
			unseen++
		}
	}
	// Linha de baixo: telhas H e L com o que fazem.
	reg := "Registro"
	if unseen > 0 && !g.showLog {
		reg = fmt.Sprintf("Registro (%d novos)", unseen)
	}
	tw := 22 + g.textW("Controles", 13) + 40 + g.textW(reg, 13) + 16
	if !g.opts.HideMenu {
		g.rect(dst, 6, chh-26, tw, 22, color.RGBA{0, 0, 0, 165})
		tx := 10.0
		tx += g.keyTile(dst, "H", tx, chh-23) + 5
		g.txt(dst, "Controles", tx, chh-23, pickC(g.showHelp, cGold, cDim), 13, false, alignLeft)
		tx += g.textW("Controles", 13) + 18
		tx += g.keyTile(dst, "L", tx, chh-23) + 5
		g.txt(dst, reg, tx, chh-23, pickC(unseen > 0 && !g.showLog, cGold, cDim), 13, false, alignLeft)
	}
	if g.showHelp {
		// Lista de controles: a tecla numa telha à esquerda, o que faz ao lado,
		// em grupos com título.
		type key struct{ k, what string }
		groups := []struct {
			title string
			keys  []key
		}{
			{"Movimento", []key{{"WASD", "mover"}, {"Shift", "correr"}, {"X", "montar e descer"}}},
			{"Combate", []key{{"ESP", "golpe básico (ou clique)"}, {"1–9", "atalhos da barra (abrem com o nível)"}, {"F · R", "atalhos 1 e 2"},
				{"Direito", "atalho 1"}, {"Q", "beber poção"}}},
			{"Mundo", []key{{"E", "interagir · descansar · tocar runas"}, {"V", "viagem rápida (fogueira ou placa)"}, {"Z", "acender/apagar a luz"}, {"G", "pescar (com a vara)"}, {"B", "modo: pacifista · defensivo · agressivo"}, {"Enter", "falar"}}},
			{"Janelas", []key{{"I", "personagem e mochila"}, {"K", "teia de habilidades"}, {"J", "conquistas"},
				{"M", "mapa"}, {"N", "diário (missões e história)"}, {"L", "registro"}, {"O", "opções"}, {"P · Esc", "pausa"}, {"F5", "salvar"}, {"H", "fecha esta ajuda"}}},
		}
		const rowH, keyW = 19.0, 64.0
		n := 0
		for _, gr := range groups {
			n += len(gr.keys) + 1
		}
		pw := 420.0
		ph := float64(n)*rowH + float64(len(groups)-1)*6 + 40
		px, py := 8.0, chh-30-ph
		g.panel(dst, px, py, pw, ph, cGold)
		g.txt(dst, "Controles", px+12, py+8, cGold, 14, true, alignLeft)
		yy := py + 32
		for gi, gr := range groups {
			if gi > 0 {
				yy += 6
			}
			g.txt(dst, gr.title, px+12, yy, cDim, 11, true, alignLeft)
			yy += rowH
			for _, k := range gr.keys {
				g.rect(dst, px+16, yy-1, keyW, 16, color.RGBA{214, 176, 80, 255})
				g.stroke(dst, px+16, yy-1, keyW, 16, color.RGBA{60, 40, 10, 255}, 1)
				g.txt(dst, k.k, px+16+keyW/2, yy, color.RGBA{24, 16, 6, 255}, 11, true, alignCenter)
				g.txt(dst, g.fit(k.what, 13, pw-keyW-40), px+keyW+28, yy, cText, 13, false, alignLeft)
				yy += rowH
			}
		}
	}
}

// interactHint espelha a ordem do servidor: domar, colher, embarcar, jangada.
func (g *Game) interactHint(me *proto.Player) string {
	if me.Boat {
		return "encoste na margem para descer do barco"
	}
	near := func(x, y, r float64) bool { return math.Hypot(x-float64(me.X), y-float64(me.Y)) <= r }
	for _, s := range g.cur.Steeds {
		if !near(s.X, s.Y, 9) || s.Kind >= len(g.info.SteedK) {
			continue
		}
		k := g.info.SteedK[s.Kind]
		if me.Level >= k.Level {
			return fmt.Sprintf("[E] domar %s (+%d de velocidade)", k.Name, k.Speed)
		}
		return fmt.Sprintf("%s: precisa do nível %d", k.Name, k.Level)
	}
	for _, h := range g.cur.Herbs {
		if near(h.X, h.Y, 7) && h.Kind < len(g.info.HerbK) {
			return "[E] colher " + g.info.Items[g.info.HerbK[h.Kind]].Name
		}
	}
	for _, d := range g.info.Doors {
		m := g.m
		if !near(float64(d[0]), float64(d[1]), 9) || d[1] < 0 || d[1] >= m.h || m.flags[d[1]*m.w+d[0]]&1 == 0 {
			continue // longe, ou a porta já está aberta
		}
		for i, it := range g.info.Items {
			if it.Kind == proto.KindKey {
				for _, s := range g.cur.Inv {
					if s[0] == i {
						return "[E] destrancar com a Chave de Ferro"
					}
				}
			}
		}
		return "Trancada: o ferreiro da vila vende a Chave de Ferro"
	}
	if i := g.nearSeal(me); i >= 0 && !g.sealOpen(g.info.Seals[i]) {
		if g.info.Seals[i].Kind == 1 {
			return "[E] tocar as runas · " + g.info.Seals[i].Hint
		}
		return g.info.Seals[i].Hint + "  (Enter para falar)"
	}
	if l := g.nearLot(me); l >= 0 {
		if g.cur.House == l+1 {
			return "[E] sua casa: abrir o baú"
		}
		return fmt.Sprintf("[E] %s · à venda por %d", g.info.Houses[l].Name, g.info.Houses[l].Price)
	}
	if t := g.nearTravel(me); t >= 0 {
		if g.info.Travel[t].Kind == 0 {
			return "[E] descansar  ·  [V] viagem rápida"
		}
		return "[V] viagem rápida"
	}
	if mo := g.cur.Moor; mo != nil && near(float64(mo[0]), float64(mo[1]), 16) {
		return "[E] embarcar no seu barco"
	}
	for _, d := range g.info.Docks {
		if near(float64(d[0]), float64(d[1]), 10) {
			return "[E] embarcar"
		}
	}
	wood := 0
	for i, it := range g.info.Items {
		if it.Kind == proto.KindMaterial {
			for _, s := range g.cur.Inv {
				if s[0] == i {
					wood = s[1]
				}
			}
			break
		}
	}
	if wood >= g.info.Raft {
		m := g.m
		for dy := -6; dy <= 6; dy++ {
			for dx := -6; dx <= 6; dx++ {
				x, y := me.X+dx, me.Y+dy
				if x >= 0 && y >= 0 && x < m.w && y < m.h && m.flags[y*m.w+x]&4 != 0 {
					return fmt.Sprintf("[E] montar jangada (%d madeiras)", g.info.Raft)
				}
			}
		}
	}
	if r := g.nearRead(me); r >= 0 {
		return "[E] ler: " + g.info.Reads[r].Title
	}
	if g.cur.Owned != 0 && me.Ride == 0 {
		return "[X] montar"
	}
	return ""
}

// drawRight é a coluna da direita: minimapa, região e hora, missão e menu.
func (g *Game) drawRight(dst *ebiten.Image, me *proto.Player, cw, ch, lh float64) {
	x := cw - 8 - rightW
	under := me.Y >= g.info.OverH
	// O minimapa foi para a esquerda e o relógio para baixo da bússola
	// (hud_layout.go): aqui ficam a missão e o menu.
	_ = under
	y := 8.0

	q := g.cur.Quest
	lines := []string{"A história terminou. Tiamath está morta."}
	extra := 0.6
	// Subobjetivos: cada um com as suas linhas (o texto quebra na largura).
	type stepRow struct {
		lines      []string
		done, cur  bool
		prog, need int
	}
	var steps []stepRow
	stepH := 0.0
	if q != nil {
		lines, extra = wrap(q.Text, int((rightW-2*ch)/ch)), 2.6
		for i, s := range q.Steps {
			r := stepRow{wrap(s.Text, int((rightW-5*ch)/ch)), s.Done, i == q.Cur, s.Prog, s.Need}
			steps = append(steps, r)
			stepH += float64(len(r.lines))
		}
		if len(steps) > 0 {
			extra += stepH + 0.4
		}
	}
	// Secundárias ativas: do capítulo atual para trás, ainda não concluídas.
	type side struct {
		title    string
		prog, of int
	}
	var sides []side
	for i, s := range g.info.Sides {
		if g.cur.Chapter >= s.Chapter && g.cur.SideDone&(1<<i) == 0 {
			pr := 0
			if i < len(g.cur.SideProg) {
				pr = g.cur.SideProg[i]
			}
			sides = append(sides, side{s.Title, pr, s.Need})
		}
	}
	if len(sides) > 0 {
		extra += 1.2 + float64(len(sides))
	}
	g.panel(dst, x, y, rightW, (float64(len(lines))+extra)*lh, nil)
	if len(sides) > 0 {
		sy := y + (float64(len(lines))+extra-float64(len(sides))-0.9)*lh
		g.txt(dst, "Secundárias (quebram selos)", x+ch, sy, rgbaHex(0xc080ff), 13, false, alignLeft)
		for i, s := range sides {
			g.txt(dst, s.title, x+ch, sy+lh*float64(i+1), cText, 13, false, alignLeft)
			g.txt(dst, fmt.Sprintf("%d/%d", s.prog, s.of), x+rightW-ch, sy+lh*float64(i+1), cGreen, 13, false, alignRight)
		}
	}
	if q != nil {
		chap := fmt.Sprintf("Missão %d/%d", q.N, q.Of)
		if q.Chapter == 0 {
			chap = "Prólogo"
		}
		if q.Chapter >= 1 && q.Chapter <= len(g.info.Chapters) {
			chap = strings.SplitN(g.info.Chapters[q.Chapter-1][0], " — ", 2)[0] + " · " + chap
		}
		g.txt(dst, chap, x+ch, y+lh*0.3, cDim, 13, false, alignLeft)
		g.txt(dst, q.Title, x+ch, y+lh*1.3, cGold, 13, false, alignLeft)
		for i, l := range lines {
			g.txt(dst, l, x+ch, y+lh*(2.3+float64(i)), cText, 13, false, alignLeft)
		}
		sy := y + lh*(2.6+float64(len(lines)))
		for _, s := range steps {
			c := color.Color(cText)
			switch {
			case s.done:
				c = cDimmer
				g.checkMark(dst, x+ch, sy+3, 9, true)
			case s.cur:
				c = cGold
				g.txt(dst, ">", x+ch+2, sy, cGold, 13, true, alignLeft)
			default:
				g.txt(dst, "·", x+ch+3, sy, cDimmer, 13, true, alignLeft)
				c = cDim
			}
			for k, l := range s.lines {
				g.txt(dst, l, x+3.2*ch, sy+lh*float64(k), c, 13, false, alignLeft)
			}
			if s.cur && s.need > 1 {
				g.txt(dst, fmt.Sprintf("%d/%d", s.prog, s.need), x+rightW-ch, sy, cGreen, 13, false, alignRight)
			}
			sy += lh * float64(len(s.lines))
		}
	} else {
		g.txt(dst, lines[0], x+ch, y+lh*0.3, cGreen, 13, false, alignLeft)
	}
	y += (float64(len(lines)) + extra) * lh

	// Menu: as teclas dos painéis, com aviso quando há ponto para gastar.
	potions := 0
	for _, s := range g.cur.Inv {
		if s[0] == 0 {
			potions = s[1]
		}
	}
	pick := func(on bool, a, b color.Color) color.Color {
		if on {
			return a
		}
		return b
	}
	// Menu em lista: telha da tecla + nome; verde quando há algo para gastar,
	// dourado quando a janela está aberta. Clicar abre a janela.
	type entry struct {
		k, name string
		c       color.Color
		fn      func()
	}
	entries := []entry{
		{"I", "Personagem e mochila", pick(g.cur.Points > 0, cGreen, pick(g.openPanel == "sheet", cGold, cText)), func() { g.togglePanel("sheet") }},
		{"K", "Teia de habilidades", pick(g.cur.SPts > 0, cGreen, pick(g.openPanel == "tree", cGold, cText)), func() { g.togglePanel("tree") }},
		{"J", "Conquistas", pick(g.openPanel == "ach", cGold, cText), func() { g.togglePanel("ach") }},
		{"M", "Mapa", pick(g.bigMap, cGold, cText), func() { g.bigMap = !g.bigMap }},
		{"Q", fmt.Sprintf("Poção  (%d)", potions), pick(potions > 0, rgbaHex(0xff8090), cDimmer), func() { g.useItem(0) }},
	}
	if g.opts.HideMenu {
		return // menu de teclas escondido nas opções
	}
	menuH := float64(len(entries))*19 + 8
	g.panel(dst, x, y, rightW, menuH, nil)
	for i, e := range entries {
		ry := y + 5 + float64(i)*19
		if g.hover(x+2, ry-1, rightW-4, 18) {
			g.rect(dst, x+2, ry-1, rightW-4, 18, color.RGBA{50, 40, 24, 200})
		}
		g.keyTile(dst, e.k, x+ch, ry)
		g.txt(dst, e.name, x+ch+28, ry, e.c, 13, false, alignLeft)
		g.uiRects = append(g.uiRects, uiRect{x: x + 2, y: ry - 1, w: rightW - 4, h: 18, fn: e.fn})
	}
	y += menuH

}
