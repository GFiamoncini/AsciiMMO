package main

import (
	"asciimmo/proto"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"math"
	"strings"
)

func itemIcon(it proto.Item) string {
	if it.Kind == 1 && it.WType >= 0 && it.WType < len(wIcons) {
		return wIcons[it.WType]
	}
	icons := []string{"!", "", "M", "^", "L", "o", "#", "*", "$", "?", "k"}
	if it.Kind >= 0 && it.Kind < len(icons) {
		return icons[it.Kind]
	}
	return "?"
}

// drawSheet é a tela de personagem: boneco com os espaços de equipamento em
// volta, mochila em grade, ficha do item sob o mouse e atributos.
func (g *Game) drawSheet(dst *ebiten.Image, cw, chh, lh float64) {
	const S, G, w, h = 58.0, 6.0, 700.0, 566.0
	x0, y0 := math.Max(8, math.Round((cw-rightW-w)/2)), math.Max(8, math.Round((chh-h)/2)-10)
	g.panel(dst, x0, y0, w, h, cGold)
	g.txt(dst, "Personagem — clique numa peça para equipar, usar ou tirar (I fecha)", x0+14, y0+10, cGold, 13, false, alignLeft)
	var hov *proto.Item
	worn := func(id int) bool {
		for _, v := range g.cur.Gear {
			if id > 0 && v == id {
				return true
			}
		}
		return false
	}
	cellS := func(x, y, S float64, id, n int, label string) {
		g.rect(dst, x, y, S, S, rgbaHex(0x0c0c14))
		if id < 0 || id >= len(g.info.Items) {
			g.stroke(dst, x+1, y+1, S-2, S-2, rgbaHex(0x3a3a48), 1)
			g.txt(dst, label, x+S/2, y+S/2-7, rgbaHex(0x4a4a58), 11, false, alignCenter)
			return
		}
		it := g.info.Items[id]
		g.stroke(dst, x+1, y+1, S-2, S-2, rarCols[clampi(it.Rarity, 0, 3)], 2)
		g.drawItemIcon(dst, it, x+S*0.07, y+S*0.07, S*0.86)
		if n > 1 {
			g.txt(dst, fmt.Sprintf("x%d", n), x+S-5, y+S-15, color.White, 11, false, alignRight)
		}
		if worn(id) && label == "" {
			g.txt(dst, "E", x+5, y+4, cGold, 11, false, alignLeft)
		}
		r := uiRect{x: x, y: y, w: S, h: S, act: &proto.In{T: "item", Item: id}}
		if label == "" { // na mochila, o botão direito joga fora
			r.alt = &proto.In{T: "drop", Item: id}
		}
		g.uiRects = append(g.uiRects, r)
		if g.hover(x, y, S, S) {
			hov = &it
		}
	}

	cell := func(x, y float64, id, n int, label string) { cellS(x, y, S, id, n, label) }

	// Boneco: uma silhueta atrás dos espaços.
	dx, top, rowH := x0+150, y0+44, S+12
	sil := rgbaHex(0x16161f)
	vector.FillCircle(dst, float32(dx*g.ui), float32((top+S/2)*g.ui), float32(40*g.ui), sil, true)
	g.rect(dst, dx-52, top+rowH*1.2-8, 104, S+30, sil)
	g.rect(dst, dx-118, top+rowH*1.2+4, 236, 26, sil)
	g.rect(dst, dx-44, top+rowH*2.4-20, 36, S+24, sil)
	g.rect(dst, dx+8, top+rowH*2.4-20, 36, S+24, sil)
	slots := []struct {
		name     string
		col, row float64
	}{{"arma", -1.3, 1.2}, {"elmo", 0, 0}, {"armadura", 0, 1.2}, {"botas", 0, 2.4}, {"amuleto", 1.3, 1.2}}
	for i, s := range slots {
		id := g.cur.Gear[i]
		if id == 0 {
			id = -1
		}
		cell(dx-S/2+s.col*(S+14), top+s.row*rowH, id, 1, s.name)
	}
	// Espaço de luz (lanterna ou tocha na mão): clicar acende/apaga; Z também.
	lx, ly := dx-S/2+1.3*(S+14), top+2.4*rowH
	lid, ln := -1, 0
	if ls := g.cur.LightSlot; ls > 0 {
		lid, ln = ls, max(1, g.owned(ls))
	}
	cell(lx, ly, lid, ln, "luz (Z)")
	if me := g.cur.pmap[g.info.ID]; me != nil && me.Light > 0 {
		g.txt(dst, "acesa", lx+S/2, ly+S+2, rgbaHex(0xffb050), 11, true, alignCenter)
	}

	y := top + rowH*2.4 + S + 14
	lineF := func(label, value string) {
		g.txt(dst, label, x0+20, y, cDim, 13, false, alignLeft)
		g.txt(dst, value, x0+290, y, cText, 13, false, alignRight)
		y += lh
	}
	lineF("Dano do golpe básico", fmt.Sprintf("%d%%", g.cur.Dmg[0]))
	lineF("Dano dos golpes de classe", fmt.Sprintf("%d%%", g.cur.Dmg[1]))
	lineF("Proteção", fmt.Sprintf("%d%%", g.cur.Def))
	y += 6
	pc := color.Color(cGold)
	if g.cur.Points > 0 {
		pc = cGreen
	}
	g.txt(dst, fmt.Sprintf("Atributos — %d ponto(s) a gastar", g.cur.Points), x0+20, y, pc, 13, false, alignLeft)
	y += lh
	for i, a := range g.info.Attrs {
		prefix, c := "   ", color.Color(cText)
		if g.cur.Points > 0 {
			prefix, c = "[+]", cGreen
			g.uiRects = append(g.uiRects, uiRect{x: x0 + 14, y: y - 2, w: 282, h: lh, act: &proto.In{T: "attr", I: i}})
		}
		g.txt(dst, fmt.Sprintf("%s %s %d", prefix, a[0], g.cur.Attr[i]), x0+20, y, c, 13, false, alignLeft)
		g.txt(dst, strings.SplitN(a[1], " (", 2)[0], x0+290, y, cDim, 13, false, alignRight)
		y += lh
	}

	// Mochila: grade de 10×5 por página; com mais espaços, setas mudam a página.
	const BS, BG, perPage = 31.0, 4.0, 50
	gx, gy := x0+330, y0+62
	bagCap := max(g.cur.BagCap, len(g.cur.Inv))
	pages := max(1, (bagCap+perPage-1)/perPage)
	g.bagPage = clampi(g.bagPage, 0, pages-1)
	g.txt(dst, fmt.Sprintf("Mochila — %d/%d", len(g.cur.Inv), bagCap), gx, y0+40, cGold, 13, false, alignLeft)
	if pages > 1 {
		px := gx + 10*(BS+BG) - 4
		g.txt(dst, fmt.Sprintf("<  %d/%d  >", g.bagPage+1, pages), px, y0+40, cText, 13, false, alignRight)
		g.uiRects = append(g.uiRects, uiRect{x: px - 92, y: y0 + 38, w: 22, h: 18, fn: func() { g.bagPage = max(0, g.bagPage-1) }})
		g.uiRects = append(g.uiRects, uiRect{x: px - 22, y: y0 + 38, w: 22, h: 18, fn: func() { g.bagPage = min(pages-1, g.bagPage+1) }})
	}
	for k := 0; k < perPage; k++ {
		i := g.bagPage*perPage + k
		if i >= bagCap {
			break
		}
		id, n := -1, 0
		if i < len(g.cur.Inv) {
			id, n = g.cur.Inv[i][0], g.cur.Inv[i][1]
		}
		cellS(gx+float64(k%10)*(BS+BG), gy+float64(k/10)*(BS+BG), BS, id, n, "")
	}

	// Ficha do item sob o mouse (texto quebrado na largura da caixa).
	fy, fw := gy+5*(BS+BG)+8, 10*(BS+BG)-BG
	bh := 7.4 * lh
	g.stroke(dst, gx, fy, fw, bh, rgbaHex(0x3a3a48), 1)
	tw := fw - 20
	if hov == nil {
		yy := fy + 8
		// Como usar a mochila, em lista.
		yy += g.bullets(dst, []string{"Mouse sobre um item: mostra o que ele faz", "Clique: usa, equipa ou acende", "Botão direito: joga fora", "E no canto: está equipado"}, gx+10, yy, tw, cDimmer, 12, 15)
		yy += 4
		g.txt(dst, "Raridade (cor da borda):", gx+10, yy, cDimmer, 12, false, alignLeft)
		yy += 15
		for i, n := range rarName {
			g.txt(dst, n, gx+10+float64(i)*(tw/4), yy, rarCols[i], 12, false, alignLeft)
		}
	} else {
		it := *hov
		var st []string
		switch {
		case it.Kind == proto.KindWeapon:
			fits := false
			for _, a := range g.info.Aff {
				fits = fits || a == it.WType
			}
			pct := it.Bonus * g.info.AffPct[1] / 100
			with := "sem"
			if fits {
				pct, with = it.Bonus*g.info.AffPct[0]/100, "com"
			}
			st = append(st, fmt.Sprintf("+%d%% de dano (%s afinidade da classe)", pct, with))
		case it.Kind == proto.KindPotion || it.Kind >= proto.KindMaterial:
			st = append(st, it.Desc)
		default:
			if it.Def > 0 {
				st = append(st, fmt.Sprintf("+%d%% de proteção", it.Def))
			}
			if it.Bonus > 0 {
				st = append(st, fmt.Sprintf("+%d%% de dano", it.Bonus))
			}
			if it.Magic > 0 {
				st = append(st, fmt.Sprintf("+%d%% de dano mágico (classes mágicas)", it.Magic))
			}
		}
		yy := fy + 8
		g.txt(dst, g.fit(it.Name, 13, tw), gx+10, yy, rarCols[clampi(it.Rarity, 0, 3)], 13, false, alignLeft)
		kind := ""
		if it.Kind == 1 && it.WType >= 0 && it.WType < len(g.info.WTypes) {
			kind = g.info.WTypes[it.WType]
		} else if it.Kind >= 0 && it.Kind < len(kindNam) {
			kind = kindNam[it.Kind]
		}
		g.txt(dst, rarName[clampi(it.Rarity, 0, 3)]+" · "+kind, gx+10, yy+lh, cDim, 13, false, alignLeft)
		yy += 2.2 * lh
		for _, s := range st {
			yy += g.txtWrap(dst, s, gx+10, yy, tw, cText, 13, lh)
		}
		if it.Kind != proto.KindPotion && yy < fy+bh-lh {
			lines := g.wrapPx("Origem: "+it.Desc, 12, tw)
			for _, l := range lines {
				if yy > fy+bh-lh {
					break
				}
				g.txt(dst, l, gx+10, yy, cDimmer, 12, false, alignLeft)
				yy += lh
			}
		}
	}
	// Montarias domadas: escolher qual usar e soltar.
	g.drawSteedList(dst, x0+14, y0+h-84, w-28)
	// O resto da janela engole o clique, para não sair golpe por trás dela.
	g.uiRects = append(g.uiRects, uiRect{x: x0, y: y0, w: w, h: h, fn: func() {}})
}
