package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// Loja como conversa: ao apertar E diante do balcão, o mercador cumprimenta e
// pergunta o que você quer; as opções são Comprar, Vender, Conversar e Até
// logo (teclas 1 a 4 ou clique). Comprar e Vender abrem a lista à esquerda e
// os detalhes do item à direita: descrição, números, comparação com o que
// está equipado, quanto você já tem, e os botões de quantidade. ↑/↓ escolhem,
// Enter compra ou vende, Esc volta à conversa e depois fecha.

// Falas de cada mercador: cumprimentos e conversa (sem spoiler da história).
var shopLines = map[string]struct{ greet, talk []string }{
	"Morvan, o ferreiro": {
		[]string{"Aço bom não pergunta de onde vem o inimigo. O que vai ser?", "Quer algo que corte, ou algo que aguente?", "A forja não apaga desde que os mortos acordaram. Diga."},
		[]string{"Os esqueletos do cemitério usam ferro velho. Ferro velho quebra; o meu, não.", "Uma Chave de Ferro abre os cofres da guarda. Não me pergunte como sei.", "Com um machado na mochila você derruba árvore em quatro golpes. Madeira vira jangada."}},
	"Ysolde, a herbalista": {
		[]string{"Ervas, poções, um pouco de luz. Do que precisa?", "Você está pálido. Sente-se, me diga o que procura.", "Cuidado com o que traz nas mãos: algumas folhas mordem."},
		[]string{"Erva-antídoto para o pântano, raiz-brasa para o frio, babosa para o deserto. Não confunda.", "A lanterna não apaga na chuva. A tocha, sim, mas a fogueira dá outra.", "O Tônico do Esquecimento devolve os seus pontos. Esquecer também é remédio."}},
	"A Velha do Charco": {
		[]string{"Hihi. Mais um vivo no charco. O que a velha tem para você?", "Não pise na água verde, criança. Venha comprar.", "Ervas daqui curam o que as de lá matam."},
		[]string{"As aranhas não gostam de fogo. Nem eu, mas eu já sou velha.", "Quem morre no charco fica no charco. Leve antídoto.", "O sapo canta quando chove. Quando ele cala, corra."}},
	"O Eremita da Ilha": {
		[]string{"Pouca gente rema até aqui. O que te trouxe?", "A ilha é pequena, mas a mochila pode ser maior.", "Senta. O lago escuta."},
		[]string{"O baú desta ilha guarda um elmo que o lago devolveu.", "Do alto das torres de Vaelor dá para ver a ilha. Do fundo do lago, também.", "Quem carrega mais, anda mais devagar? Não com as minhas bolsas."}},
	"Mascate das Sombras": {
		[]string{"Psst. Ninguém te seguiu? Então negocie.", "Achou a parede rachada, hein. Esperto. O que quer?", "Aqui não tem preço de vila. Tem preço de quem achou."},
		[]string{"Há outras paredes rachadas. Bata nas que soam ocas.", "A Chave de Ferro abre portas que voltam a trancar. Seja rápido.", "Eu vendo bolsas porque quem desce aqui sempre volta carregado."}},
	"Velho Guardião de Vaelor": {
		[]string{"O castelo caiu, mas eu ainda guardo. O que deseja, viajante?", "Poucos atravessam o fosso. Fale.", "Os mortos do pátio não me incomodam mais. Você, talvez."},
		[]string{"A brecha a oeste é por onde a trilha entra. O portão nunca mais abriu.", "Há um cofre no pátio. A chave você já sabe onde comprar.", "As janelas frias do castelo não apagam. Ninguém sabe quem as acende."}},
	"Durgo, o anão do fundo": {
		[]string{"Hm. Gente de cima. Veio comprar ou veio morrer na lava?", "Botas que pisam fogo, manto que segura o calor. Escolha.", "Dorrim caiu, mas a forja de Durgo arde."},
		[]string{"A lava tira 45 por segundo de quem não tem Botas de Obsidiana.", "Os Salões de Dorrim têm um cofre atrás de uma parede rachada.", "As Pedras de Eco acendem os portões rúnicos. Eu vendo algumas."}},
}

func (g *Game) shopLine(s proto.Shop, talk bool, k int) string {
	l, ok := shopLines[s.Name]
	if !ok {
		return "O que vai ser?"
	}
	list := l.greet
	if talk {
		list = l.talk
	}
	return list[k%len(list)]
}

// openShop abre a conversa com o mercador i.
func (g *Game) openShop(i int) {
	g.openPanel, g.shopIdx, g.shopView, g.shopSel = "shop", i, "menu", 0
	g.shopSay, g.shopSayAt = g.shopLine(g.info.Shops[i], false, int(g.now()/1000)), g.now()
	g.say(g.info.Shops[i].Name, g.shopSay)
}

// stackable diz se faz sentido comprar ou vender vários de uma vez.
func stackable(it proto.Item) bool {
	switch it.Kind {
	case proto.KindPotion, proto.KindHerb, proto.KindMaterial, proto.KindTool, proto.KindKey:
		return true
	}
	return strings.Contains(it.Name, "Pedra de Eco") || strings.Contains(it.Name, "Tocha")
}

// shopItems são as linhas da lista atual: (item, preço, quantos você tem).
func (g *Game) shopItems(s proto.Shop) [][3]int {
	var out [][3]int
	if g.shopView == "buy" {
		for _, st := range s.Stock {
			out = append(out, [3]int{st[0], st[1], g.owned(st[0])})
		}
		return out
	}
	for _, inv := range g.cur.Inv {
		if inv[0] >= 0 && inv[0] < len(g.info.Items) && g.info.Items[inv[0]].Kind != proto.KindMoney {
			out = append(out, [3]int{inv[0], g.sellPrice(s, inv[0]), inv[1]})
		}
	}
	return out
}

func (g *Game) owned(id int) int {
	for _, inv := range g.cur.Inv {
		if inv[0] == id {
			return inv[1]
		}
	}
	return 0
}

// shopDo compra ou vende n unidades do item selecionado.
func (g *Game) shopDo(n int) {
	s := g.info.Shops[g.shopIdx]
	items := g.shopItems(s)
	if g.shopSel < 0 || g.shopSel >= len(items) {
		return
	}
	row := items[g.shopSel]
	me := g.cur.pmap[g.info.ID]
	if g.shopView == "buy" {
		if me == nil || me.Gold < row[1] {
			g.shopSay, g.shopSayAt = "Isso custa mais do que você tem.", g.now()
			return
		}
		n = min(n, me.Gold/max(1, row[1]))
		for k := 0; k < n; k++ {
			g.send(proto.In{T: "buy", I: g.shopIdx, Item: row[0]})
		}
		g.sfx("coin", 0.6)
		g.shopSay, g.shopSayAt = "Bom negócio.", g.now()
		return
	}
	n = min(n, row[2])
	for k := 0; k < n; k++ {
		g.send(proto.In{T: "sell", I: g.shopIdx, Item: row[0]})
	}
	g.sfx("coin", 0.6)
	g.shopSay, g.shopSayAt = "Vou achar um uso para isso.", g.now()
}

// updateShopKeys cuida das teclas com a loja aberta; true se as tomou.
func (g *Game) updateShopKeys() bool {
	if g.openPanel != "shop" || g.shopIdx < 0 || g.shopIdx >= len(g.info.Shops) {
		return false
	}
	s := g.info.Shops[g.shopIdx]
	switch g.shopView {
	case "menu", "talk":
		switch {
		case justKey(ebiten.KeyDigit1):
			g.shopView, g.shopSel = "buy", 0
		case justKey(ebiten.KeyDigit2):
			g.shopView, g.shopSel = "sell", 0
		case justKey(ebiten.KeyDigit3):
			g.shopTalk++
			g.shopView, g.shopSay, g.shopSayAt = "talk", g.shopLine(s, true, g.shopTalk), g.now()
		case justKey(ebiten.KeyDigit4), justKey(ebiten.KeyEscape), justKey(ebiten.KeyE):
			g.openPanel = ""
		}
	default:
		n := len(g.shopItems(s))
		switch {
		case justKey(ebiten.KeyEscape), justKey(ebiten.KeyBackspace):
			g.shopView = "menu"
		case justKey(ebiten.KeyE):
			g.openPanel = ""
		case repeatKey(ebiten.KeyArrowUp), repeatKey(ebiten.KeyW):
			g.shopSel = max(0, g.shopSel-1)
		case repeatKey(ebiten.KeyArrowDown), repeatKey(ebiten.KeyS):
			g.shopSel = min(n-1, g.shopSel+1)
		case justKey(ebiten.KeyTab):
			if g.shopView == "buy" {
				g.shopView = "sell"
			} else {
				g.shopView = "buy"
			}
			g.shopSel = 0
		case justKey(ebiten.KeyEnter), justKey(ebiten.KeyNumpadEnter):
			g.shopDo(1)
		}
	}
	return true
}

// drawShop desenha a loja: conversa ou lista com detalhes.
func (g *Game) drawShop(dst *ebiten.Image, me *proto.Player, cw, chh, lh float64) {
	if g.shopIdx < 0 || g.shopIdx >= len(g.info.Shops) || g.nearShop(me) != g.shopIdx {
		g.openPanel = ""
		return
	}
	s := g.info.Shops[g.shopIdx]
	w, h := math.Min(860, cw-rightW-30), 470.0
	x, y := math.Max(8, (cw-rightW-w)/2), math.Max(8, (chh-h)/2-10)
	g.panel(dst, x, y, w, h, rgbaHex(0xc08040))
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, rgbaHex(0xe0a050), 14, true, alignCenter)
	}
	g.txt(dst, s.Name, x+16, y+12, rgbaHex(0xffd090), 16, true, alignLeft)
	g.txt(dst, fmt.Sprintf("$ %d %s", me.Gold, g.info.Currency), x+w-16, y+14, rgbaHex(0xc8a860), 13, false, alignRight)

	// A fala do mercador, surgindo letra a letra, num balão.
	say := []rune(g.shopSay)
	if g.voice == nil || g.voice.text != g.shopSay {
		g.say(s.Name, g.shopSay) // fala nova do mercador: com voz (voz.go)
	}
	n := min(len(say), g.typed())
	g.rect(dst, x+16, y+38, w-32, 44, color.RGBA{24, 18, 14, 255})
	g.stroke(dst, x+16, y+38, w-32, 44, rgbaHex(0x6a4a2a), 1)
	for i, l := range g.wrapPx("“"+string(say[:n])+"”", 14, w-60) {
		if i < 2 {
			g.txt(dst, l, x+28, y+46+float64(i)*18, rgbaHex(0xf0e0c0), 14, false, alignLeft)
		}
	}
	top := y + 96
	if g.shopView == "menu" || g.shopView == "talk" {
		opts := []struct {
			key, label, hint string
			fn               func()
		}{
			{"1", "Comprar", fmt.Sprintf("%d itens à venda", len(s.Stock)), func() { g.shopView, g.shopSel = "buy", 0 }},
			{"2", "Vender", "o que está na sua mochila", func() { g.shopView, g.shopSel = "sell", 0 }},
			{"3", "Conversar", "ouvir o que ele sabe", func() {
				g.shopTalk++
				g.shopView, g.shopSay, g.shopSayAt = "talk", g.shopLine(s, true, g.shopTalk), g.now()
			}},
			{"4", "Até logo", "fechar (Esc)", func() { g.openPanel = "" }},
		}
		for i, o := range opts {
			oy := top + float64(i)*52
			hov := g.hover(x+16, oy, w*0.55, 44)
			bc := color.Color(rgbaHex(0x4a3420))
			if hov {
				bc = rgbaHex(0xe0a050)
			}
			g.rect(dst, x+16, oy, w*0.55, 44, color.RGBA{16, 12, 10, 255})
			g.stroke(dst, x+16, oy, w*0.55, 44, bc, 1)
			g.keyTile(dst, o.key, x+28, oy+12)
			g.txt(dst, o.label, x+60, oy+8, pickC(hov, cGold, cText), 15, true, alignLeft)
			g.txt(dst, o.hint, x+60, oy+26, cDim, 11, false, alignLeft)
			g.uiRects = append(g.uiRects, uiRect{x: x + 16, y: oy, w: w * 0.55, h: 44, fn: o.fn})
		}
		// Ao lado, a mercadoria em vitrine (os primeiros itens, com ícone).
		vx := x + 32 + w*0.55
		g.txt(dst, "No balcão", vx, top, cGold, 12, true, alignLeft)
		for i, st := range s.Stock {
			if i >= 8 {
				break
			}
			it := g.info.Items[st[0]]
			iy := top + 20 + float64(i)*26
			g.drawItemIcon(dst, it, vx, iy, 22)
			g.txt(dst, g.fit(it.Name, 12, w*0.45-110), vx+28, iy+4, rarCols[clampi(it.Rarity, 0, 3)], 12, false, alignLeft)
			g.txt(dst, fmt.Sprint(st[1]), x+w-20, iy+4, cGold, 12, false, alignRight)
		}
		g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
		return
	}

	// Abas.
	tabs := []struct {
		view, label string
	}{{"buy", "Comprar"}, {"sell", "Vender"}, {"menu", "< Voltar"}}
	for i, t := range tabs {
		tx := x + 16 + float64(i)*120
		on := g.shopView == t.view
		g.rect(dst, tx, top, 112, 24, pickRGBA(on, color.RGBA{60, 40, 20, 255}, color.RGBA{16, 12, 10, 255}))
		g.stroke(dst, tx, top, 112, 24, pickRGBA(on || g.hover(tx, top, 112, 24), rgbaHex(0xe0a050), rgbaHex(0x4a3420)), 1)
		g.txt(dst, t.label, tx+56, top+5, pickC(on, cGold, cText), 13, on, alignCenter)
		view := t.view
		g.uiRects = append(g.uiRects, uiRect{x: tx, y: top, w: 112, h: 24, fn: func() { g.shopView, g.shopSel = view, 0 }})
	}
	g.txt(dst, "↑/↓ escolhe · Enter "+map[string]string{"buy": "compra", "sell": "vende"}[g.shopView]+" · Tab troca · Esc volta", x+w-16, top+6, cDimmer, 11, false, alignRight)

	items := g.shopItems(s)
	g.shopSel = clampi(g.shopSel, 0, max(0, len(items)-1))
	lw, rowH := w*0.52, 30.0
	ly := top + 34
	maxRows := int((y + h - 14 - ly) / rowH)
	first := clampi(g.shopSel-maxRows+1, 0, max(0, len(items)-maxRows))
	if len(items) == 0 {
		msg := "O mercador não tem nada agora."
		if g.shopView == "sell" {
			msg = "Sua mochila está vazia."
		}
		g.txt(dst, msg, x+24, ly+8, cDimmer, 13, false, alignLeft)
	}
	for k := first; k < len(items) && k < first+maxRows; k++ {
		row := items[k]
		it := g.info.Items[row[0]]
		ry := ly + float64(k-first)*rowH
		if g.hover(x+16, ry, lw, rowH-2) {
			g.shopSel = k
		}
		sel := k == g.shopSel
		g.rect(dst, x+16, ry, lw, rowH-2, pickRGBA(sel, color.RGBA{48, 34, 22, 255}, color.RGBA{14, 12, 18, 255}))
		if sel {
			g.stroke(dst, x+16, ry, lw, rowH-2, rgbaHex(0xe0a050), 1)
		}
		g.drawItemIcon(dst, it, x+19, ry+1, rowH-4)
		name := it.Name
		if row[2] > 0 && g.shopView == "sell" && row[2] > 1 {
			name += fmt.Sprintf(" x%d", row[2])
		}
		g.txt(dst, g.fit(name, 13, lw-110), x+52, ry+7, rarCols[clampi(it.Rarity, 0, 3)], 13, false, alignLeft)
		pc := color.Color(cGold)
		if g.shopView == "buy" && me.Gold < row[1] {
			pc = rgbaHex(0xd06060)
		} else if g.shopView == "sell" {
			pc = rgbaHex(0x9ae070)
		}
		g.txt(dst, fmt.Sprint(row[1]), x+16+lw-10, ry+7, pc, 13, true, alignRight)
		kk := k
		g.uiRects = append(g.uiRects, uiRect{x: x + 16, y: ry, w: lw, h: rowH - 2, fn: func() { g.shopSel = kk }})
	}

	// Detalhes do item escolhido.
	dx, dw := x+32+lw, w-lw-48
	g.rect(dst, dx, ly, dw, y+h-14-ly, color.RGBA{12, 10, 14, 255})
	g.stroke(dst, dx, ly, dw, y+h-14-ly, rgbaHex(0x4a3420), 1)
	if g.shopSel < len(items) {
		row := items[g.shopSel]
		it := g.info.Items[row[0]]
		g.drawItemIcon(dst, it, dx+12, ly+12, 44)
		g.txt(dst, g.fit(it.Name, 15, dw-80), dx+66, ly+14, rarCols[clampi(it.Rarity, 0, 3)], 15, true, alignLeft)
		g.txt(dst, []string{"comum", "incomum", "raro", "lendário"}[clampi(it.Rarity, 0, 3)], dx+66, ly+34, cDim, 11, false, alignLeft)
		yy := ly + 66
		line := func(s string, c color.Color) {
			for _, l := range g.wrapPx(s, 12, dw-24) {
				g.txt(dst, l, dx+12, yy, c, 12, false, alignLeft)
				yy += 16
			}
		}
		if it.Desc != "" {
			line(it.Desc, cText)
			yy += 6
		}
		if it.Kind == proto.KindWeapon {
			line(fmt.Sprintf("Dano: +%d%% (%s)", it.Bonus, g.info.WTypes[clampi(it.WType, 0, len(g.info.WTypes)-1)]), cText)
			if cur := g.cur.Gear[0]; cur > 0 && cur < len(g.info.Items) {
				d := it.Bonus - g.info.Items[cur].Bonus
				c, sgn := rgbaHex(0x9ae070), "+"
				if d < 0 {
					c, sgn = rgbaHex(0xd06060), ""
				}
				line(fmt.Sprintf("Comparado à sua %s: %s%d%%", g.info.Items[cur].Name, sgn, d), c)
			}
		} else if it.Def > 0 {
			line(fmt.Sprintf("Proteção: +%d%%", it.Def), cText)
		}
		if it.Magic > 0 {
			line(fmt.Sprintf("Magia: +%d%%", it.Magic), cText)
		}
		have := g.owned(row[0])
		line(fmt.Sprintf("Você tem: %d", have), cDim)
		verb := "Comprar"
		if g.shopView == "sell" {
			verb = "Vender"
		}
		qty := []int{1}
		if stackable(it) {
			qty = append(qty, 5)
		}
		if g.shopView == "sell" && row[2] > 1 {
			qty = append(qty, row[2])
		}
		by := y + h - 60
		bx := dx + 12
		for _, q := range qty {
			label := verb
			if q > 1 {
				label = fmt.Sprintf("%s %d", verb, q)
				if g.shopView == "sell" && q == row[2] {
					label = verb + " todos"
				}
			}
			bw := g.textW(label, 13) + 24
			ok := g.shopView == "sell" || me.Gold >= row[1]
			bc := color.Color(rgbaHex(0x6a4a2a))
			if ok && g.hover(bx, by, bw, 30) {
				bc = cGold
			}
			g.rect(dst, bx, by, bw, 30, pickRGBA(ok, color.RGBA{50, 34, 18, 255}, color.RGBA{24, 20, 20, 255}))
			g.stroke(dst, bx, by, bw, 30, bc, 1)
			g.txt(dst, label, bx+bw/2, by+7, pickC(ok, cText, cDimmer), 13, true, alignCenter)
			qq := q
			g.uiRects = append(g.uiRects, uiRect{x: bx, y: by, w: bw, h: 30, fn: func() { g.shopDo(qq) }})
			bx += bw + 8
		}
		total := row[1]
		g.txt(dst, fmt.Sprintf("%d %s cada", total, g.info.Currency), dx+dw-12, by+8, rgbaHex(0xc8a860), 12, false, alignRight)
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

func pickRGBA(ok bool, a, b color.RGBA) color.RGBA {
	if ok {
		return a
	}
	return b
}
