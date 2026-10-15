package main

import (
	"fmt"
	"image/color"
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ---------- teia de habilidades (tecla K) ----------
// Um núcleo no meio, os ramos saindo dele como raios de uma teia de aranha e
// anéis de fio ligando os ramos. Cada nó mostra o ícone; a ficha ao lado
// explica o que ele faz e o que falta para aprender.

func (g *Game) webAdj() [][]int {
	if g.adj != nil && len(g.adj) == len(g.info.Web) {
		return g.adj
	}
	g.adj = make([][]int, len(g.info.Web))
	for i, n := range g.info.Web {
		for _, l := range n.Links {
			g.adj[i] = append(g.adj[i], l)
			g.adj[l] = append(g.adj[l], i)
		}
	}
	return g.adj
}

func (g *Game) rank(i int) int {
	if i == 0 {
		return 1
	}
	if i < len(g.cur.Tree) {
		return g.cur.Tree[i]
	}
	return 0
}

// webReqs lista os requisitos do nó e se cada um está cumprido.
func (g *Game) webReqs(i int, me *proto.Player) (ok bool, reqs []struct {
	text string
	ok   bool
}) {
	n := g.info.Web[i]
	add := func(t string, v bool) {
		reqs = append(reqs, struct {
			text string
			ok   bool
		}{t, v})
	}
	attr, av := g.mainAttr()
	linked := false
	for _, l := range g.webAdj()[i] {
		linked = linked || g.rank(l) > 0
	}
	add(fmt.Sprintf("Nível %d", n.Lv), me.Level >= n.Lv)
	add(fmt.Sprintf("%s %d (atributo da classe)", attr, n.Attr), av >= n.Attr)
	add("Ligado a um nó aprendido", linked)
	if n.Crypt {
		add("Ter descido à cripta", g.cur.Unl)
	}
	add("1 ponto de habilidade", g.cur.SPts > 0)
	ok = g.rank(i) < n.Max
	for _, r := range reqs {
		ok = ok && r.ok
	}
	return ok, reqs
}

func (g *Game) drawWeb(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	if len(g.info.Web) == 0 {
		return
	}
	w := math.Min(cw-rightW-24, 1060)
	h := math.Min(chh-150, 660)
	x0, y0 := math.Max(8, math.Round((cw-rightW-w)/2)), math.Max(8, math.Round((chh-90-h)/2))
	g.panel(dst, x0, y0, w, h, cGold)
	attr, av := g.mainAttr()
	g.txt(dst, fmt.Sprintf("Habilidades — %s · nível %d · %s %d · %d ponto(s) a gastar", g.info.Class, me.Level, attr,
		av, g.cur.SPts), x0+14, y0+10, cGold, 14, true, alignLeft)
	g.txt(dst, "Passe o mouse num nó para ver o que ele faz", x0+14, y0+30, cDim, 12, false, alignLeft)

	infoW := math.Min(330, w*0.36)
	webW := w - infoW - 20
	cx, cy := x0+webW/2+10, y0+52+(h-62)/2
	// Raio de cada anel: o primeiro já sai afastado do núcleo.
	// O quinto anel (golpe final) tem de caber abaixo do título.
	step := math.Min(webW/2-36, (h-62)/2-26) / 6.35
	ringR := func(ring int) float64 { return (0.9 + float64(ring)) * step }
	pos := func(i int) (float64, float64) {
		n := g.info.Web[i]
		if i == 0 {
			return cx, cy
		}
		a := n.Ang * math.Pi / 180
		return cx + math.Cos(a)*ringR(n.Ring), cy + math.Sin(a)*ringR(n.Ring)
	}
	line := func(ax, ay, bx, by float64, c color.Color, width float64) {
		vector.StrokeLine(dst, float32(ax*g.ui), float32(ay*g.ui), float32(bx*g.ui), float32(by*g.ui), float32(width*g.ui), c, true)
	}

	// Teia de fundo: raios a cada 30° e anéis de fio caído. Não muda de um
	// quadro para outro, então é desenhada uma vez numa imagem e reaproveitada.
	key := fmt.Sprint(w, h, g.ui, cx-x0, cy-y0, step)
	if g.webBG == nil || g.webBGKey != key {
		if g.webBG != nil {
			g.webBG.Deallocate()
		}
		g.webBG, g.webBGKey = ebiten.NewImage(int(w*g.ui)+1, int(h*g.ui)+1), key
		bl := func(ax, ay, bx, by float64) {
			vector.StrokeLine(g.webBG, float32((ax-x0)*g.ui), float32((ay-y0)*g.ui), float32((bx-x0)*g.ui), float32((by-y0)*g.ui),
				float32(g.ui), color.RGBA{70, 70, 92, 150}, true)
		}
		for a := 0.0; a < 360; a += 30 {
			r := a * math.Pi / 180
			bl(cx, cy, cx+math.Cos(r)*ringR(4)*1.08, cy+math.Sin(r)*ringR(4)*1.08)
		}
		for ring := 1; ring <= 4; ring++ {
			R := ringR(ring)
			for k := 0; k < 12; k++ {
				a1 := float64(k) * math.Pi / 6
				px, py := cx+math.Cos(a1)*R, cy+math.Sin(a1)*R
				for s := 1; s <= 6; s++ {
					t := float64(s) / 6
					a := a1 + math.Pi/6*t
					sag := 1 - 0.07*math.Sin(t*math.Pi) // o fio cede para o centro
					nx, ny := cx+math.Cos(a)*R*sag, cy+math.Sin(a)*R*sag
					bl(px, py, nx, ny)
					px, py = nx, ny
				}
			}
		}
	}
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x0*g.ui, y0*g.ui)
		dst.DrawImage(g.webBG, op)
	}
	// Nome de cada ramo, por fora da teia.
	for _, b := range g.info.Branches {
		a := b.Ang * math.Pi / 180
		pal := elemPal[clampi(b.Elem, 0, len(elemPal)-1)]
		g.txt(dst, b.Name, cx+math.Cos(a)*ringR(4)*1.3, cy+math.Sin(a)*ringR(4)*1.22-7, pal[2].rgba(), 13, true, alignCenter)
	}

	// Fios entre os nós: brilham na cor do elemento quando os dois lados
	// foram aprendidos; ficam claros quando levam a um nó que já dá para aprender.
	for i, n := range g.info.Web {
		ax, ay := pos(i)
		for _, l := range n.Links {
			bx, by := pos(l)
			c, wd := color.Color(color.RGBA{60, 60, 76, 255}), 1.5
			switch {
			case g.rank(i) > 0 && g.rank(l) > 0:
				c, wd = elemPal[clampi(n.Elem, 0, len(elemPal)-1)][2].rgba(), 3
			case g.rank(i) > 0 || g.rank(l) > 0:
				c, wd = color.RGBA{150, 150, 170, 200}, 2
			}
			if o := g.info.Web[l]; i > 0 && l > 0 && o.Ring == n.Ring {
				// Fio entre ramos: segue a curva do anel, como teia.
				a1, a2 := o.Ang*math.Pi/180, n.Ang*math.Pi/180
				d := math.Remainder(a2-a1, 2*math.Pi)
				R := ringR(n.Ring)
				px, py := bx, by
				for s := 1; s <= 16; s++ {
					a := a1 + d*float64(s)/16
					qx, qy := cx+math.Cos(a)*R, cy+math.Sin(a)*R
					line(px, py, qx, qy, c, wd)
					px, py = qx, qy
				}
				continue
			}
			line(ax, ay, bx, by, c, wd)
		}
	}

	// Nós: primeiro os ícones de todos, depois por cima os números de grau e
	// as telhas de atalho, para nenhum vizinho tapar a informação.
	g.webHover = -1
	pulse := 0.5 + 0.5*math.Sin(g.now()/220)
	sizeA := clamp(step*0.8, 24, 44)
	sizeOf := func(i int, n proto.WebNode) float64 {
		switch {
		case i == 0:
			return sizeA * 1.25
		case n.Skill >= 0:
			return sizeA
		}
		return sizeA * 0.78
	}
	for i, n := range g.info.Web {
		px, py := pos(i)
		size := sizeOf(i, n)
		r := g.rank(i)
		can, _ := g.webReqs(i, me)
		can = can && i != 0
		x, y := px-size/2, py-size/2
		if can {
			c := color.RGBA{120, 255, 140, uint8(90 + 150*pulse)}
			vector.StrokeCircle(dst, float32(px*g.ui), float32(py*g.ui), float32((size/2+5)*g.ui), float32(2*g.ui), c, true)
		}
		g.drawIcon(dst, n.Icon, n.Elem, x, y, size, r == 0 && i != 0)
		if r > 0 && i != 0 {
			g.stroke(dst, x-2, y-2, size+4, size+4, elemPal[clampi(n.Elem, 0, len(elemPal)-1)][3].rgba(), 1.5)
		}
		if g.hover(x, y, size, size) {
			g.webHover = i
		}
		if can {
			g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: size, h: size, act: &proto.In{T: "tree", I: i}})
		}
	}
	for i, n := range g.info.Web {
		px, py := pos(i)
		size := sizeOf(i, n)
		x, y := px-size/2, py-size/2
		r := g.rank(i)
		if n.Max > 1 { // grau, dentro do canto de baixo do nó
			lbl := fmt.Sprintf("%d/%d", r, n.Max)
			tw := g.textW(lbl, 10) + 4
			g.rect(dst, x+size-tw, y+size-12, tw, 12, color.RGBA{0, 0, 0, 220})
			g.txt(dst, lbl, x+size-tw/2, y+size-13, pickC(r > 0, cGold, cDim), 10, true, alignCenter)
		}
		if n.Skill >= 0 && r > 0 {
			for k, s := range g.cur.Bar {
				if s == n.Skill { // telha do atalho, no canto de cima
					g.rect(dst, x+size-12, y-3, 14, 14, color.RGBA{230, 190, 80, 255})
					g.txt(dst, fmt.Sprint(k+1), x+size-5, y-3, color.RGBA{20, 14, 8, 255}, 11, true, alignCenter)
				}
			}
		}
	}

	// Técnicas do Mestre Oren já aprendidas: fora da teia, numa faixa embaixo.
	g.techHover = -1
	tx := x0 + 14
	for t, tk := range g.info.Techs {
		if !g.knowsTech(t) || tk.Skill >= len(g.info.Book) {
			continue
		}
		if tx == x0+14 {
			g.txt(dst, "Técnicas", tx, y0+h-52, cDim, 11, true, alignLeft)
		}
		sk := g.info.Book[tk.Skill]
		g.drawIcon(dst, sk.Icon, sk.Elem, tx, y0+h-36, 28, false)
		for k, s := range g.cur.Bar {
			if s == tk.Skill {
				g.txt(dst, fmt.Sprint(k+1), tx+24, y0+h-40, cGold, 10, true, alignCenter)
			}
		}
		if g.hover(tx, y0+h-36, 28, 28) {
			g.techHover = tk.Skill
			g.txt(dst, sk.Name+" — 1 a 9 põe na barra", x0+14, y0+h-68, cGold, 12, false, alignLeft)
		}
		tx += 34
	}

	// Ficha do nó sob o mouse.
	fx, fy, fw := x0+w-infoW-12, y0+52, infoW
	g.rect(dst, fx, fy, fw, h-64, color.RGBA{8, 8, 14, 230})
	g.stroke(dst, fx, fy, fw, h-64, rgbaHex(0x3a3a48), 1)
	i := g.webHover
	if i < 0 {
		g.txt(dst, "Passe o mouse num nó da teia.", fx+12, fy+12, cDim, 13, false, alignLeft)
		yy := fy + 40
		yy += g.bullets(dst, []string{
			"Cada nível dá 1 ponto de habilidade e 3 de atributo",
			"Cada nó pede nível e pontos no atributo da classe",
			"Nós grandes ensinam golpes; os pequenos são passivos e sobem de grau",
			"Só se aprende um nó ligado por fio a outro já aprendido",
			"Tomo do Recomeço (herbalista) devolve os pontos da teia",
		}, fx+12, yy, fw-24, cDimmer, 12, 16)
		yy += 10
		g.txt(dst, "Teclas", fx+12, yy, cDim, 11, true, alignLeft)
		yy += 18
		for _, k := range [][2]string{{"Clique", "aprender o nó"}, {"1–9", "pôr o golpe sob o mouse na barra"}, {"K", "fechar"}} {
			w := g.keyTile(dst, k[0], fx+12, yy)
			g.txt(dst, k[1], fx+20+w, yy, cDimmer, 12, false, alignLeft)
			yy += 20
		}
	} else {
		n := g.info.Web[i]
		pal := elemPal[clampi(n.Elem, 0, len(elemPal)-1)]
		g.drawIcon(dst, n.Icon, n.Elem, fx+12, fy+12, 56, false)
		g.txt(dst, g.fit(n.Name, 15, fw-88), fx+78, fy+14, pal[2].rgba(), 15, true, alignLeft)
		elem := ""
		if n.Elem < len(g.info.Elems) {
			elem = g.info.Elems[n.Elem]
		}
		kind := "Passiva · " + elem
		switch {
		case i == 0:
			kind = "Núcleo"
		case n.Skill >= 0:
			kind = "Habilidade ativa · " + elem
		}
		g.txt(dst, g.fit(kind, 12, fw-88), fx+78, fy+34, cDim, 12, false, alignLeft)
		if n.Max > 1 {
			g.txt(dst, fmt.Sprintf("Grau %d de %d", g.rank(i), n.Max), fx+78, fy+50, cGold, 12, false, alignLeft)
		}
		yy := fy + 80
		for _, l := range g.wrapPx(n.Desc, 13, fw-24) {
			g.txt(dst, l, fx+12, yy, cText, 13, false, alignLeft)
			yy += 17
		}
		if n.Skill >= 0 && n.Skill < len(g.info.Book) {
			sk := g.info.Book[n.Skill]
			yy += 6
			g.txt(dst, fmt.Sprintf("Custo: %d de mana   ·   Recarga: %.1f s", sk.Cost, sk.CD), fx+12, yy, rgbaHex(0x7aa8ff), 12, false, alignLeft)
			yy += 18
		}
		if i != 0 {
			yy += 8
			g.txt(dst, "Requisitos", fx+12, yy, cGold, 13, true, alignLeft)
			yy += 18
			_, reqs := g.webReqs(i, me)
			for _, r := range reqs {
				c := color.Color(cGreen)
				if !r.ok {
					c = rgbaHex(0xd06060)
				}
				g.checkMark(dst, fx+14, yy+2, 10, r.ok)
				g.txt(dst, g.fit(r.text, 12, fw-40), fx+30, yy, c, 12, false, alignLeft)
				yy += 16
			}
			yy += 10
			can, _ := g.webReqs(i, me)
			switch {
			case g.rank(i) >= n.Max:
				g.txt(dst, "Aprendido.", fx+12, yy, cGreen, 13, true, alignLeft)
			case can:
				g.txt(dst, "Clique para aprender.", fx+12, yy, cGreen, 13, true, alignLeft)
			default:
				g.txt(dst, "Ainda trancado.", fx+12, yy, cDimmer, 13, true, alignLeft)
			}
			if n.Skill >= 0 && g.rank(i) > 0 {
				g.txt(dst, "Aperte 1 a 9 para pôr na barra.", fx+12, yy+18, cGold, 12, false, alignLeft)
			}
		}
	}
	// O resto da janela engole o clique, para não sair golpe por trás dela.
	g.uiRects = append(g.uiRects, uiRect{x: x0, y: y0, w: w, h: h, fn: func() {}})
}

// ---------- barra de ações ----------

var buffNames = []struct{ icon, name string }{{"horn", "Grito de Guerra"}, {"eye", "Olho de Águia"}, {"vial", "Lâminas Envenenadas"}}

// drawActionBar é a barra de baixo: o golpe básico (Espaço/clique) e os cinco
// atalhos (1 a 9, abertos pelo nível), cada um com a telha da tecla, a recarga e o custo.
func (g *Game) drawActionBar(dst *ebiten.Image, me *proto.Player, cw, chh, lh float64) {
	if len(g.info.Book) == 0 {
		return
	}
	const S, G = 46.0, 7.0
	n := 1 + len(g.cur.Bar)
	total := float64(n)*S + float64(n-1)*G + 12
	x0, y0 := (cw-rightW-total)/2+rightW*0.5-rightW*0.5, chh-34-S
	x0 = math.Max(8, (cw-total)/2)
	g.barHover = -2
	tile := func(x float64, skill int, key string) {
		if skill < 0 || skill >= len(g.info.Book) {
			g.rect(dst, x, y0, S, S, color.RGBA{10, 10, 16, 200})
			g.stroke(dst, x, y0, S, S, rgbaHex(0x3a3a48), 1)
			g.txt(dst, "+", x+S/2, y0+S/2-9, cDimmer, 16, true, alignCenter)
			g.uiRects = append(g.uiRects, uiRect{x: x, y: y0, w: S, h: S, fn: func() { g.openPanel = "tree" }})
		} else {
			sk := g.info.Book[skill]
			cd := 0.0
			if skill < len(g.cur.CD2) {
				cd = g.cur.CD2[skill]
			}
			g.drawIcon(dst, sk.Icon, sk.Elem, x, y0, S, me.MP < sk.Cost)
			if cd > 0 && sk.CD > 0 {
				k := clamp(cd/sk.CD, 0, 1)
				g.rect(dst, x+1, y0+1+(S-2)*(1-k), S-2, (S-2)*k, color.RGBA{0, 0, 0, 170})
				g.txt(dst, fmt.Sprintf("%.1f", cd), x+S/2, y0+S/2-8, color.White, 13, true, alignCenter)
			}
			if sk.Cost > 0 {
				c := rgbaHex(0x8ab8ff)
				if me.MP < sk.Cost {
					c = rgbaHex(0xd06060)
				}
				g.txt(dst, fmt.Sprint(sk.Cost), x+S-3, y0+S-14, c, 11, true, alignRight)
			}
			if g.hover(x, y0, S, S) {
				g.barHover = skill
			}
		}
		// Telha da tecla, no canto de cima.
		kw := math.Max(16, g.textW(key, 11)+8)
		g.rect(dst, x-3, y0-7, kw, 15, color.RGBA{214, 176, 80, 255})
		g.stroke(dst, x-3, y0-7, kw, 15, color.RGBA{60, 40, 10, 255}, 1)
		g.txt(dst, key, x-3+kw/2, y0-7, color.RGBA{24, 16, 6, 255}, 11, true, alignCenter)
	}
	tile(x0, 0, "ESP")
	for i, s := range g.cur.Bar {
		x := x0 + 12 + float64(i+1)*(S+G)
		if i >= g.barN() { // atalho ainda trancado: mostra o nível que o abre
			g.rect(dst, x, y0, S, S, color.RGBA{6, 6, 10, 150})
			g.stroke(dst, x, y0, S, S, rgbaHex(0x26262e), 1)
			g.txt(dst, fmt.Sprintf("nv %d", barLevels[i-(proto.BarSlots-len(barLevels))]), x+S/2, y0+S/2-7, cDimmer, 11, false, alignCenter)
			continue
		}
		tile(x, s, fmt.Sprint(i+1))
	}

	// Reforços ativos, logo acima da barra.
	bx := x0
	for i, b := range buffNames {
		if i < len(g.cur.Buf) && g.cur.Buf[i] > 0 {
			g.drawIcon(dst, b.icon, []int{0, 0, 7}[i], bx, y0-40, 24, false)
			g.txt(dst, fmt.Sprintf("%.0fs", g.cur.Buf[i]), bx+12, y0-15, cText, 10, false, alignCenter)
			bx += 30
		}
	}
	if me.Guard {
		g.drawIcon(dst, "shield", 6, bx, y0-40, 24, false)
	}

	// Ficha do golpe sob o mouse, acima da barra.
	if g.barHover >= 0 && g.barHover < len(g.info.Book) && g.openPanel == "" {
		sk := g.info.Book[g.barHover]
		lines := g.wrapPx(sk.Desc, 12, 380)
		bw, bh := 400.0, 46+float64(len(lines))*16
		tx, ty := math.Min(math.Max(8, g.mouseX-bw/2), cw-bw-8), y0-bh-48
		g.panel(dst, tx, ty, bw, bh, elemPal[clampi(sk.Elem, 0, len(elemPal)-1)][1].rgba())
		g.txt(dst, sk.Name, tx+10, ty+8, elemPal[clampi(sk.Elem, 0, len(elemPal)-1)][2].rgba(), 14, true, alignLeft)
		g.txt(dst, fmt.Sprintf("%d MP · %.1f s", sk.Cost, sk.CD), tx+bw-10, ty+9, rgbaHex(0x8ab8ff), 12, false, alignRight)
		for i, l := range lines {
			g.txt(dst, l, tx+10, ty+30+float64(i)*16, cText, 12, false, alignLeft)
		}
	}
}

func pickC(on bool, a, b color.Color) color.Color {
	if on {
		return a
	}
	return b
}

// mainAttr devolve o nome e o valor do atributo que a teia pede (no
// Depravado, o mais alto que ele tiver).
func (g *Game) mainAttr() (string, int) {
	if g.info.MainAttr < 0 {
		best := 0
		for _, v := range g.cur.Attr {
			best = max(best, v)
		}
		return "melhor atributo", best
	}
	i := clampi(g.info.MainAttr, 0, len(g.cur.Attr)-1)
	return g.info.Attrs[i][0], g.cur.Attr[i]
}

// barLevels: níveis em que a barra abre mais um atalho (igual ao servidor).
var barLevels = []int{6, 10, 14, 18}

// barN é quantos atalhos já estão abertos (servidor antigo: 5).
func (g *Game) barN() int {
	if g.cur == nil || g.cur.BarN <= 0 {
		return 5
	}
	return min(g.cur.BarN, proto.BarSlots)
}
