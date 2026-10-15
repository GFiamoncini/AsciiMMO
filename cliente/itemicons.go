package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Ícones de itens e conquistas, no mesmo estilo dos de habilidade (12×12,
// icons.go): cada peça do set tem a sua figura (elmo, peitoral, botas,
// amuleto, anel), cada tipo de arma a sua, e a cor vem do próprio item; a
// raridade dá o brilho da moldura.

func init() {
	for k, v := range map[string][12]string{
		"chest": {
			"............",
			".oo......oo.",
			"o33o....o32o",
			"o334oooo332o",
			".o3344443320",
			"..o334432o..",
			"..o3a33a2o..",
			"..o333322o..",
			"..o3a3a22o..",
			"..o333222o..",
			"..o122211o..",
			"...oooooo...",
		},
		"helm": {
			"............",
			"....oooo....",
			"..oo3443oo..",
			".o33344332o.",
			".o33333332o.",
			"o333333322oo",
			"o3ooooooo2o.",
			"o3o.a..a.o2o",
			"o3o......o2o",
			"o33o....o32o",
			".ooo....ooo.",
			"............",
		},
		"boots": {
			"............",
			"..oooo......",
			"..o343o.....",
			"..o333o.....",
			"..oa3ao.....",
			"..o333o.....",
			"..o3332o....",
			"..o33322oo..",
			".o33333222o.",
			".o32222221o.",
			".oooooooooo.",
			"............",
		},
		"amulet": {
			"o.........o.",
			".o.......o..",
			"..o.....o...",
			"...o...o....",
			"....o.o.....",
			"....ooo.....",
			"...o343o....",
			"..o34a43o...",
			"..o3aaa3o...",
			"..o23a32o...",
			"...o222o....",
			"....ooo.....",
		},
		"ring": {
			"............",
			"....oaao....",
			"...oa44ao...",
			"....oaao....",
			"...ooooo....",
			"..o33333o...",
			".o3o...o3o..",
			".o3.....2o..",
			".o3.....2o..",
			".o2o...o2o..",
			"..o22222o...",
			"...ooooo....",
		},
		"potion": {
			"....oooo....",
			"....obbo....",
			".....oo.....",
			"....o..o....",
			"...o4...o...",
			"..o4.....o..",
			"..o333333o..",
			"..o344433o..",
			"..o333332o..",
			"..o222221o..",
			"...oooooo...",
			"............",
		},
		"herb": {
			"............",
			"......o.....",
			".....o3o....",
			"..o..o3o....",
			".o3o.o3o.o..",
			".o33oo3oo3o.",
			"..o33o3o33o.",
			"...o3333o...",
			"....o33o....",
			".....bo.....",
			".....bo.....",
			".....oo.....",
		},
		"log": {
			"............",
			"............",
			"..oooooooo..",
			".obbbbbbbbo.",
			"oobbbbbbbbo.",
			"o3ooooooooo.",
			"o232ooooooo.",
			"o2a2obbbbbbo",
			"o232obbbbbbo",
			"o3ooooooooo.",
			".oo.........",
			"............",
		},
		"stone": {
			"............",
			"............",
			"....oooo....",
			"...o3443o...",
			"..o334433o..",
			".o33333332o.",
			".o33322322o.",
			"o3332222221o",
			"o2222222211o",
			".oooooooooo.",
			"............",
			"............",
		},
		"coin": {
			"............",
			"...oooooo...",
			"..o344433o..",
			".o34oooo32o.",
			".o3o3443o2o.",
			".o3o3aa3o2o.",
			".o3o3aa3o2o.",
			".o3o2332o2o.",
			".o32oooo22o.",
			"..o222221o..",
			"...oooooo...",
			"............",
		},
		"tome": {
			"............",
			".ooooooooo..",
			".o3333333oo.",
			".o3a3a3a3ob.",
			".o33aaa33ob.",
			".o3a3a3a3ob.",
			".o3333333ob.",
			".o3333333ob.",
			".o2222222ob.",
			".ooooooooob.",
			"..bbbbbbbbb.",
			"............",
		},
		"key": {
			"............",
			"..oooo......",
			".o3443o.....",
			"o34oo43o....",
			"o3o..o3o....",
			"o34oo43o....",
			".o3333oooo..",
			"..oooo3333o.",
			"......o3oo..",
			"......o33o..",
			"......o3o...",
			".......o....",
		},
		"staff": {
			"........ooo.",
			".......o44ao",
			".......o4aao",
			"......ooaao.",
			".....obo....",
			"....obo.....",
			"...obo......",
			"..obo.......",
			".obo........",
			"obo.........",
			"oo..........",
			"............",
		},
		"bow": {
			"..oo........",
			"..obo.......",
			"..o.bo......",
			"..o..bo.....",
			"..o...bo....",
			"..sssssbo...",
			"..o...bo....",
			"..o..bo.....",
			"..o.bo......",
			"..obo.......",
			"..oo........",
			"............",
		},
		"claw": {
			"............",
			"o...o...o...",
			"o4..o4..o4..",
			".o3..o3..o3.",
			".o3..o3..o3.",
			"..o3.o3..o3.",
			"..o33o33o33.",
			"..o2222222o.",
			"..obbbbbbbo.",
			"...obbbbbo..",
			"....ooooo...",
			"............",
		},
		"trophy": {
			"............",
			".oooooooooo.",
			"oo34444433oo",
			"o.o344433o.o",
			"o.o333333o.o",
			".oo333332oo.",
			"...o3332o...",
			"....o32o....",
			"....o22o....",
			"...oooooo...",
			"..o222221o..",
			"..oooooooo..",
		},
		"map": {
			"............",
			"oooooooooooo",
			"o3333a33333o",
			"o333a3a3333o",
			"o33a333a333o",
			"o3a33333a33o",
			"o333333333ao",
			"o33333333a3o",
			"o333333aa33o",
			"o22222222222",
			"oooooooooooo",
			"............",
		},
		"chestbox": {
			"............",
			"..oooooooo..",
			".o33333333o.",
			"o3333aa3333o",
			"oooooaaooooo",
			"o2222aa2222o",
			"o2222aa2222o",
			"o2222222222o",
			"o1111111111o",
			"oooooooooooo",
			"............",
			"............",
		},
		"dragon": {
			"............",
			"o.........o.",
			"oo...oo..oo.",
			".o3o.o3oo3o.",
			"..o3oo33o...",
			"...o3333o...",
			"..o334a33o..",
			".o3333333o..",
			".o333oo33o..",
			"..ooo..oo...",
			"............",
			"............",
		},
	} {
		iconArt[k] = v
	}
}

var wIconArt = []string{"sword", "dagger", "spear", "hammer", "staff", "bow", "claw"}

// itemArtKey escolhe a figura do item pelo tipo, pela arma e pelo nome.
func itemArtKey(it proto.Item) string {
	switch it.Kind {
	case proto.KindPotion:
		return "potion"
	case proto.KindWeapon:
		if it.WType >= 0 && it.WType < len(wIconArt) {
			return wIconArt[it.WType]
		}
		return "sword"
	case proto.KindSkin:
		return "chest"
	case proto.KindHelm:
		return "helm"
	case proto.KindBoots:
		return "boots"
	case proto.KindAmulet:
		return "amulet"
	case proto.KindMaterial:
		if strings.HasPrefix(it.Name, "Madeira") {
			return "log"
		}
		return "stone"
	case proto.KindHerb:
		return "herb"
	case proto.KindMoney:
		return "coin"
	case proto.KindScroll:
		if strings.HasPrefix(it.Name, "Tônico") {
			return "potion"
		}
		return "tome"
	case proto.KindKey:
		return "key"
	case proto.KindRing:
		return "ring"
	case proto.KindBag:
		return "bag"
	case proto.KindTool:
		switch {
		case strings.HasPrefix(it.Name, "Lanterna"):
			return "lantern"
		case strings.HasPrefix(it.Name, "Tocha"):
			return "torch"
		}
		return "axe"
	}
	return "star"
}

// tintPal monta a paleta de um ícone a partir de uma cor só: escuro, médio,
// claro, brilho e um destaque dourado.
func tintPal(c col) [5]col {
	return [5]col{mix3(c, col{10, 8, 14}, 0.55), c, mix3(c, white, 0.35), mix3(c, white, 0.75), {240, 205, 100}}
}

var tintCache = map[string]*ebiten.Image{}

func tintedIcon(key string, c col) *ebiten.Image {
	ck := key + "|" + string(rune(int(c[0])<<16|int(c[1])<<8|int(c[2])))
	if img, ok := tintCache[ck]; ok {
		return img
	}
	art, ok := iconArt[key]
	if !ok {
		art = iconArt["star"]
	}
	pal := tintPal(c)
	px := make([]byte, 12*12*4)
	for y, row := range art {
		for x := 0; x < 12 && x < len(row); x++ {
			var cc col
			switch row[x] {
			case 'o':
				cc = col{14, 10, 18}
			case '0':
				cc = pal[1]
			case '1':
				cc = pal[0]
			case '2':
				cc = pal[1]
			case '3':
				cc = pal[2]
			case '4':
				cc = pal[3]
			case 'a':
				cc = pal[4]
			case 'b':
				cc = col{120, 82, 48}
			case 's':
				cc = col{190, 196, 214}
			default:
				continue
			}
			o := (y*12 + x) * 4
			px[o], px[o+1], px[o+2], px[o+3] = cb(cc[0]), cb(cc[1]), cb(cc[2]), 255
		}
	}
	img := ebiten.NewImage(12, 12)
	img.WritePixels(px)
	tintCache[ck] = img
	return img
}

// drawItemIcon desenha o ícone do item num quadrado de lado size.
func (g *Game) drawItemIcon(dst *ebiten.Image, it proto.Item, x, y, size float64) {
	g.drawTinted(dst, itemArtKey(it), hexc(it.Color), x, y, size, false)
}

func (g *Game) drawTinted(dst *ebiten.Image, key string, c col, x, y, size float64, dim bool) {
	img := tintedIcon(key, c)
	op := &ebiten.DrawImageOptions{}
	pad := size * 0.1
	op.GeoM.Scale((size-2*pad)*g.ui/12, (size-2*pad)*g.ui/12)
	op.GeoM.Translate((x+pad)*g.ui, (y+pad)*g.ui)
	op.Filter = ebiten.FilterNearest
	if dim {
		op.ColorScale.Scale(0.3, 0.3, 0.34, 1)
	}
	dst.DrawImage(img, op)
}

// Ícone e cor de cada conquista, na ordem do servidor (progress.go).
var achIcons = []struct {
	key string
	c   col
}{
	{"dagger", col{210, 60, 50}},    // Primeiro Sangue
	{"bow", col{150, 190, 90}},      // Caçador
	{"skull", col{200, 190, 170}},   // Exterminador
	{"map", col{210, 180, 120}},     // Explorador
	{"chestbox", col{200, 150, 70}}, // Caça-Tesouros
	{"star", col{255, 200, 80}},     // Lendário
	{"dragon", col{220, 90, 50}},    // Matador de Chefes
	{"shield", col{150, 170, 200}},  // Veterano
	{"crown", col{240, 200, 90}},    // Mestre
	{"trophy", col{255, 215, 90}},   // Herói
}

// drawAchievements é o painel de conquistas (J): uma grade de cartões com o
// ícone aceso nas obtidas e apagado nas que faltam.
func (g *Game) drawAchievements(dst *ebiten.Image, cw, chh float64) {
	n := len(g.info.Achs)
	cols := 2
	cardW, cardH := 300.0, 70.0
	rows := (n + cols - 1) / cols
	w, h := float64(cols)*cardW+float64(cols+1)*10, float64(rows)*(cardH+10)+50
	x0, y0 := math.Max(8, (cw-rightW-w)/2), math.Max(8, (chh-h)/2-20)
	have := 0
	for i := range g.info.Achs {
		if g.cur.Ach>>i&1 != 0 {
			have++
		}
	}
	g.panel(dst, x0, y0, w, h, cGold)
	g.txt(dst, fmt.Sprintf("Conquistas — %d/%d   (J fecha)", have, n), x0+14, y0+12, cGold, 14, true, alignLeft)
	for i, a := range g.info.Achs {
		cx, cy := x0+10+float64(i%cols)*(cardW+10), y0+40+float64(i/cols)*(cardH+10)
		got := g.cur.Ach>>i&1 != 0
		bg := color.RGBA{12, 11, 18, 255}
		if got {
			bg = color.RGBA{30, 24, 14, 255}
		}
		g.rect(dst, cx, cy, cardW, cardH, bg)
		ic := achIcons[i%len(achIcons)]
		frame := color.Color(rgbaHex(0x3a3a48))
		if got {
			frame = ic.c.rgba()
		}
		g.rect(dst, cx+8, cy+8, 54, 54, color.RGBA{8, 6, 12, 255})
		g.drawTinted(dst, ic.key, ic.c, cx+8, cy+8, 54, !got)
		g.stroke(dst, cx+8, cy+8, 54, 54, frame, 2)
		g.stroke(dst, cx, cy, cardW, cardH, frame, 1)
		nc := color.Color(cGold)
		if !got {
			nc = cDim
		}
		g.txt(dst, g.fit(a[0], 14, cardW-100), cx+72, cy+12, nc, 14, true, alignLeft)
		for k, l := range g.wrapPx(a[1], 12, cardW-84) {
			if k > 1 {
				break
			}
			g.txt(dst, l, cx+72, cy+32+float64(k)*15, pickC(got, cText, cDimmer), 12, false, alignLeft)
		}
		if got {
			g.checkMark(dst, cx+cardW-22, cy+10, 12, true)
		}
	}
	g.uiRects = append(g.uiRects, uiRect{x: x0, y: y0, w: w, h: h, fn: func() {}})
}

// drawStatusCard é o card do personagem, enxuto e em ASCII: retrato pequeno,
// "Nome · Classe  nv N" numa linha e as barras como texto (= cheio, · vazio)
// com o número ao lado; embaixo, a experiência e o dinheiro. Fundo escuro
// translúcido numa moldura dourada fina, com cantos "+" como os de um quadro em
// ASCII e o traço mais forte à esquerda. Devolve a altura usada.
func (g *Game) drawStatusCard(dst *ebiten.Image, me *proto.Player, x, y float64) float64 {
	cur := g.cur
	const w, h, ps = 286.0, 94.0, 40.0
	g.rect(dst, x, y, w, h, color.RGBA{0, 0, 0, 130})
	g.stroke(dst, x, y, w, h, color.RGBA{214, 176, 80, 120}, 1)
	g.rect(dst, x, y, 2, h, color.RGBA{214, 176, 80, 200})
	for _, c := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", c[0], c[1]-8, color.RGBA{236, 200, 110, 230}, 12, true, alignCenter)
	}
	g.drawPortrait(dst, me.Class, x+8, y+8, ps)
	if cur.Points > 0 || cur.SPts > 0 { // pontos a gastar: um "+" piscando no canto do retrato
		a := uint8(160 + 90*math.Sin(g.now()/200))
		g.txt(dst, "+", x+8+ps-2, y+2, color.RGBA{110, 230, 110, a}, 13, true, alignCenter)
	}

	tx := x + ps + 18
	name := g.fit(me.Name, 13, w*0.42)
	g.txt(dst, name, tx, y+6, cGold, 13, true, alignLeft)
	cx := tx + g.textW(name, 13)*1.06 + 6
	lv := fmt.Sprintf("nv %d", me.Level)
	g.txt(dst, lv, x+w-8, y+7, cGold, 12, false, alignRight)
	g.txt(dst, g.fit("· "+g.info.Class, 12, x+w-16-g.textW(lv, 12)-cx), cx, y+7, rgbaHex(0x9a8a70), 12, false, alignLeft)

	// Barra em texto: rótulo, = cheios na cor, · vazios apagados, e o número.
	const cells = 16
	bar := func(by float64, label string, v, mv int, full, dim color.RGBA) {
		k := clamp(float64(v)/float64(max(1, mv)), 0, 1)
		n := int(math.Round(k * cells))
		g.txt(dst, label, tx, by, rgbaHex(0x8a8a96), 11, false, alignLeft)
		bx := tx + 22
		g.txt(dst, strings.Repeat("=", n), bx, by, full, 11, true, alignLeft)
		g.txt(dst, strings.Repeat("·", cells-n), bx+g.textW(strings.Repeat("=", n), 11)*1.05, by, dim, 11, true, alignLeft)
		g.txt(dst, fmt.Sprintf("%d/%d", v, mv), x+w-8, by, rgbaHex(0xb8b0a0), 11, false, alignRight)
	}
	bar(y+24, "HP", me.HP, me.MaxHP, color.RGBA{230, 80, 70, 255}, color.RGBA{80, 34, 34, 255})
	bar(y+37, "MP", me.MP, me.MaxMP, color.RGBA{100, 150, 255, 255}, color.RGBA{34, 44, 90, 255})
	bar(y+50, "ST", cur.ST, cur.MaxST, color.RGBA{120, 210, 100, 255}, color.RGBA{36, 70, 34, 255})

	// Experiência: barra dourada com quanto tem, quanto pede o nível e
	// quanto falta; o dinheiro vai embaixo, à direita.
	xk := clamp(float64(cur.XP)/float64(max(1, cur.XPNeed)), 0, 1)
	g.txt(dst, "XP", tx, y+63, rgbaHex(0x8a8a96), 11, false, alignLeft)
	bx, bw := tx+22, x+w-8-(tx+22)
	g.rect(dst, bx, y+65, bw, 8, color.RGBA{40, 32, 16, 255})
	g.rect(dst, bx, y+65, bw*xk, 8, color.RGBA{214, 176, 80, 230})
	g.stroke(dst, bx, y+65, bw, 8, color.RGBA{120, 96, 50, 200}, 1)
	if gain := g.now() - g.xpAt; g.xpAt > 0 && gain < 1500 { // acabou de ganhar: a ponta brilha
		g.rect(dst, bx+bw*xk-3, y+64, 3, 10, color.RGBA{255, 240, 180, uint8(255 * (1 - gain/1500))})
	}
	g.txt(dst, fmt.Sprintf("%d / %d · faltam %d", cur.XP, cur.XPNeed, max(0, cur.XPNeed-cur.XP)), tx, y+77, rgbaHex(0x9a9080), 10, false, alignLeft)
	money := fmt.Sprintf("$ %d %s", me.Gold, g.info.Currency)
	g.txt(dst, money, x+w-8, y+77, rgbaHex(0xc8a860), 10, false, alignRight)
	return h
}

// checkMark desenha um visto (ok) ou um x, já que a fonte não tem ✓ e ✗.
func (g *Game) checkMark(dst *ebiten.Image, x, y, s float64, ok bool) {
	u := float32(g.ui)
	line := func(ax, ay, bx, by float64, c color.Color) {
		vector.StrokeLine(dst, float32(ax)*u, float32(ay)*u, float32(bx)*u, float32(by)*u, 2*u, c, true)
	}
	if ok {
		c := color.RGBA{110, 230, 110, 255}
		line(x, y+s*0.55, x+s*0.38, y+s*0.9, c)
		line(x+s*0.38, y+s*0.9, x+s, y+s*0.1, c)
		return
	}
	c := color.RGBA{220, 90, 90, 255}
	line(x+s*0.1, y+s*0.1, x+s*0.9, y+s*0.9, c)
	line(x+s*0.9, y+s*0.1, x+s*0.1, y+s*0.9, c)
}

func init() {
	iconArt["bag"] = [12]string{
		"....oooo....",
		"...o....o...",
		"...o....o...",
		".oooooooooo.",
		"o3333333333o",
		"o34ooooo333o",
		"o33oaaao333o",
		"o333ooo3332o",
		"o3333333322o",
		"o2222222221o",
		".oooooooooo.",
		"............",
	}
}

func init() {
	iconArt["steed"] = [12]string{
		"............",
		".........oo.",
		"........o33o",
		".......o333o",
		".oooooo3332o",
		"o33333333oo.",
		"o333333332o.",
		"o33333333o..",
		".o3o..o3o...",
		".o3o..o3o...",
		".o2o..o2o...",
		".ooo..ooo...",
	}
}

// drawSteedList mostra as montarias: as domadas com Montar/Descer e Soltar;
// as que faltam, apagadas, com o nível pedido.
func (g *Game) drawSteedList(dst *ebiten.Image, x, y, w float64) {
	me := g.cur.pmap[g.info.ID]
	if me == nil {
		return
	}
	g.txt(dst, "Montarias — escolha qual usar (X monta e desce)", x, y, cGold, 13, false, alignLeft)
	n := len(g.info.SteedK)
	cw := (w - float64(n-1)*8) / float64(n)
	for k, sk := range g.info.SteedK {
		cx, cy := x+float64(k)*(cw+8), y+20
		owned := g.cur.Owned&(1<<k) != 0
		riding := me.Ride == k+1
		bg := color.RGBA{12, 11, 18, 255}
		if riding {
			bg = color.RGBA{34, 28, 14, 255}
		}
		g.rect(dst, cx, cy, cw, 58, bg)
		g.stroke(dst, cx, cy, cw, 58, pickC(riding, cGold, rgbaHex(0x3a3a48)), 1)
		c := col{150, 120, 90}
		if k < len(steedStyles) && steedStyles[k].c1 != nil {
			c = *steedStyles[k].c1
		}
		g.drawTinted(dst, steedIcons[k%len(steedIcons)], c, cx+4, cy+4, 40, !owned)
		g.txt(dst, g.fit(sk.Name, 12, cw-52), cx+48, cy+5, pickC(owned, cText, cDimmer), 12, true, alignLeft)
		if !owned {
			g.txt(dst, g.fit(fmt.Sprintf("dome no nível %d", sk.Level), 11, cw-52), cx+48, cy+22, cDimmer, 11, false, alignLeft)
			continue
		}
		g.txt(dst, g.fit(fmt.Sprintf("+%d vel · +%d%% def", sk.Speed, sk.Def), 11, cw-52), cx+48, cy+21, cDim, 11, false, alignLeft)
		btn := func(bx float64, label string, act *proto.In, hot color.Color) {
			bw := g.textW(label, 11) + 10
			hc := hot
			if g.hover(bx, cy+38, bw, 16) {
				hc = cGold
			}
			g.stroke(dst, bx, cy+38, bw, 16, hc, 1)
			g.txt(dst, label, bx+bw/2, cy+39, hc, 11, false, alignCenter)
			g.uiRects = append(g.uiRects, uiRect{x: bx, y: cy + 38, w: bw, h: 16, act: act})
		}
		if riding {
			btn(cx+48, "Descer", &proto.In{T: "ride"}, cGreen)
		} else {
			btn(cx+48, "Montar", &proto.In{T: "steed", I: k}, cGreen)
		}
		// Soltar pede um segundo clique em até 3 s (não tem volta).
		label, hot := "Soltar", color.Color(rgbaHex(0xd06060))
		ask := g.releaseAsk == k+1 && g.now()-g.releaseAt < 3000
		if ask {
			label, hot = "Confirmar?", color.Color(rgbaHex(0xff5040))
		}
		bw := g.textW(label, 11) + 10
		bx := cx + cw - bw - 4
		if g.hover(bx, cy+38, bw, 16) {
			hot = cGold
		}
		g.stroke(dst, bx, cy+38, bw, 16, hot, 1)
		g.txt(dst, label, bx+bw/2, cy+39, hot, 11, false, alignCenter)
		kk := k
		g.uiRects = append(g.uiRects, uiRect{x: bx, y: cy + 38, w: bw, h: 16, fn: func() {
			if ask {
				g.send(proto.In{T: "release", I: kk})
				g.releaseAsk = 0
				return
			}
			g.releaseAsk, g.releaseAt = kk+1, g.now()
		}})
	}
}

// Um ícone por montaria, na ordem do servidor: cavalo, javali, avestruz, velociraptor.
var steedIcons = []string{"horse", "boar", "ostrich", "raptor"}

func init() {
	iconArt["horse"] = [12]string{
		"........oo..",
		".......o3ao.",
		"......o333o.",
		".....oa33o3o",
		"o...oa33o.o.",
		"oo.o3333o...",
		"o3o3333333o.",
		".o33333333o.",
		"..o3oooo3o..",
		"..o3o..o3o..",
		"..o2o..o2o..",
		"..ooo..ooo..",
	}
	iconArt["boar"] = [12]string{
		"............",
		"............",
		"..oooooooo..",
		".o33333333o.",
		"o3333333333o",
		"o33333333oo3o",
		"o333333334o3",
		".o3333333ao.",
		"..o3oo3o4o..",
		"..o2o.o2o...",
		"..ooo.ooo...",
		"............",
	}
	iconArt["ostrich"] = [12]string{
		".......oo...",
		"......o4ao..",
		"......o3o...",
		"......o3o...",
		"......o3o...",
		"..ooooo3o...",
		".o333333o...",
		"o33333333o..",
		".o333333o...",
		"...o3o3o....",
		"...o2o2o....",
		"..oo.oo.....",
	}
	iconArt["raptor"] = [12]string{
		"........ooo.",
		".......o34ao",
		"......o33ooo",
		"......o3o4..",
		"o....o33o...",
		"oo..o333o...",
		".o3o33333o..",
		"..o3333o3o..",
		"...o33o.o...",
		"...o3o......",
		"..o3o3o.....",
		"..oo.oo.....",
	}
}

func init() {
	iconArt["lantern"] = [12]string{
		"....oooo....",
		"...o....o...",
		"....oooo....",
		"...osssso...",
		"..os4aa4so..",
		"..os4aa4so..",
		"..osa44aso..",
		"..os4aa4so..",
		"...osssso...",
		"....oooo....",
		"............",
		"............",
	}
	iconArt["torch"] = [12]string{
		".....a......",
		"....a4a.....",
		"...a434a....",
		"...a333a....",
		"....o3o.....",
		"....obo.....",
		"....obo.....",
		"....obo.....",
		"....obo.....",
		"....obo.....",
		"....ooo.....",
		"............",
	}
	iconArt["axe"] = [12]string{
		"..oooo......",
		".o3444o.....",
		"o34333oo....",
		"o3333obo....",
		".o333obo....",
		"..ooobo.....",
		"....obo.....",
		"...obo......",
		"...obo......",
		"..obo.......",
		"..obo.......",
		"..oo........",
	}
}
