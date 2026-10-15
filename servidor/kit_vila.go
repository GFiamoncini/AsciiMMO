package main

import (
	"math"
	"strings"
)

// Mais peças do kit: cabana de mercador com placa, placas de direção nas
// trilhas e os objetos que enchem o chão da vila (caixotes, barris, potes,
// lenha, carroça, tapetes, cerca). Texto em placa usa só ASCII, porque o
// atlas do cliente não tem acentos.

// signText escreve o texto numa placa de madeira com moldura, centrada em
// (cx, y). solid diz se a placa bloqueia (as presas em poste, não).
func (wd *World) signText(cx, y int, text string, ink rgb, seed uint32) {
	wd.addRead("Placa", text, cx, y)
	text = asciiUpper(text)
	w := len(text) + 4
	x0 := cx - w/2
	for dy := -1; dy <= 1; dy++ {
		for dx := 0; dx < w; dx++ {
			ch, c := byte('='), dim(0x6a4626, 0.85+0.25*rnd01(x0+dx, y+dy, seed+900))
			switch {
			case dy == 0 && dx >= 2 && dx < w-2:
				ch, c = text[dx-2], ink
				if ch == ' ' {
					ch, c = '=', dim(0x6a4626, 0.9)
				}
			case dx == 0 || dx == w-1:
				ch, c = '|', 0x3a2414
			}
			wd.put(x0+dx, y+dy, ch, c, false, true)
		}
	}
}

// asciiUpper tira os acentos e põe em maiúsculas.
func asciiUpper(s string) string {
	r := strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o",
		"ú", "u", "ç", "c", "Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ç", "C", "—", "-", "·", "-")
	return strings.ToUpper(r.Replace(s))
}

// shopHut ergue a cabana do mercador com a frente aberta virada para o sul:
// telhado de palha por cima (passa por cima de quem entra), parede do fundo,
// dois esteios na frente, balcão, chão de tábuas e a placa pendurada. O
// mercador fica dentro, em (cx, by-6); devolve essa posição.
func (wd *World) shopHut(cx, by int, sign string, ink rgb, goods []rgb, seed uint32) (int, int) {
	const hw, d = 12, 14
	x0, x1, top := cx-hw, cx+hw, by-d
	wd.clearArea(x0-3, top-12, x1+3, by+4, seed)
	// Chão de tábuas.
	for y := top; y <= by; y++ {
		for x := x0; x <= x1; x++ {
			ch, c := byte('='), dim(0x6a4a2c, 0.5+0.3*rnd01(x, y, seed+901))
			if (x-x0)%5 == 0 {
				ch, c = ':', dim(c, 0.6)
			}
			wd.put(x, y, ch, c, false, false)
		}
	}
	// Parede do fundo e paredes laterais (vistas de cima, com espessura).
	for x := x0; x <= x1; x++ {
		for y := top - 6; y <= top; y++ {
			wd.put(x, y, pick("#H|", hash2(x, y, seed+902)), dim(0x7a5632, 0.8-0.04*float64(top-y)), y == top, y < top)
		}
	}
	for y := top; y <= by-2; y++ {
		for _, x := range []int{x0, x0 + 1, x1 - 1, x1} {
			wd.put(x, y, '|', dim(0x5a3c22, 0.8), true, false)
		}
	}
	// Balcão e mercadorias em cima dele.
	for x := x0 + 3; x <= x1-3; x++ {
		if x >= cx-2 && x <= cx+2 {
			continue // passagem para o mercador
		}
		wd.put(x, by-3, '=', dim(0x8a6440, 1.0), true, false)
		if len(goods) > 0 && rnd01(x, by, seed+903) < 0.7 {
			wd.put(x, by-4, pick("o*%&", hash2(x, by, seed+904)), goods[int(hash2(x, 1, seed+905))%len(goods)], false, true)
		}
	}
	// Telhado de palha: cobre tudo, mais claro na beira, com franja.
	for y := top - 12; y <= by-4; y++ {
		k := float64(y-(top-12)) / float64(by-4-(top-12))
		for x := x0 - 2; x <= x1+2; x++ {
			if y < top-6 && math.Abs(float64(x-cx)) > float64(hw+2)*(0.55+0.45*float64(y-(top-12))/6) {
				continue // cumeeira mais estreita em cima
			}
			ch := pick("/\\|", hash2(x, y, seed+906))
			if y == by-4 {
				ch = "VvYv"[hash2(x, y, seed+907)%4] // franja
			}
			wd.put(x, y, ch, dim(mix(0x6a5228, 0x8a6e36, rnd01(x, y, seed+908)), 0.6+0.35*k), false, true)
		}
	}
	// Esteios da frente.
	for _, x := range []int{x0, x1} {
		for y := by - 4; y <= by; y++ {
			wd.put(x, y, '|', 0x5a3c22, by-y < 2, by-y >= 2)
		}
	}
	// Placa pendurada à frente do telhado.
	wd.signText(cx, by-4, sign, ink, seed)
	wd.lights = append(wd.lights, light{x1 + 3, by + 2, 22, 1})
	return cx, by - 1 // na boca da cabana, atrás do balcão não daria para ver
}

// signpost põe uma placa de direção num poste: uma tábua por destino, com a
// seta para o lado dele. dirs: texto e ângulo (radianos) de cada destino.
func (wd *World) signpost(x, y int, dests []string, angs []float64, seed uint32) {
	wd.addRead("Placa de direção", signpostText(dests, angs), x, y)
	for dy := 0; dy <= 3+2*len(dests); dy++ {
		wd.put(x, y-dy, '|', dim(0x5a3c22, 0.9), dy < 2, dy >= 2)
	}
	for i, name := range dests {
		a := angs[i]
		txt := asciiUpper(name)
		left := math.Cos(a) < 0
		switch {
		case math.Abs(math.Sin(a)) > 0.8 && math.Sin(a) < 0:
			txt = "^ " + txt
		case math.Abs(math.Sin(a)) > 0.8:
			txt = "v " + txt
		case left:
			txt = "< " + txt
		default:
			txt += " >"
		}
		by := y - 4 - 2*i
		w := len(txt) + 2
		bx := x - w + 1
		if !left {
			bx = x
		}
		for dx := 0; dx < w; dx++ {
			ch, c := byte('='), dim(0x7a5632, 0.8+0.25*rnd01(bx+dx, by, seed+910))
			if dx >= 1 && dx <= len(txt) && txt[dx-1] != ' ' {
				ch, c = txt[dx-1], 0xc8b48c
			}
			wd.put(bx+dx, by, ch, c, false, true)
		}
	}
}

// ---------- objetos do chão ----------

func (wd *World) crate(x, y int, seed uint32) {
	for dy := -3; dy <= 0; dy++ {
		for dx := -2; dx <= 2; dx++ {
			ch, l := byte('#'), 1.05-0.12*float64(dx+2)
			if dy == -3 {
				ch, l = '=', 1.2 // tampa, pega luz
			} else if dx == -2 || dx == 2 || dy == 0 {
				ch, l = '|', 0.7
			}
			wd.put(x+dx, y+dy, ch, dim(0x9a7040, l), dy == 0, dy < 0)
		}
	}
}

func (wd *World) barrel(x, y int, seed uint32) {
	for dy := -4; dy <= 0; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if (dy == -4 || dy == 0) && (dx == -2 || dx == 2) {
				continue
			}
			t := float64(dx) / 2.5
			l := 0.6 + 0.5*(math.Sqrt(1-t*t)*0.8-t*0.4)
			ch, c := byte('|'), rgb(0x8a5a30)
			if dy == -3 || dy == -1 {
				ch, c = '=', 0x5a5a68 // aros de ferro
			}
			if dy == -4 {
				ch, l = 'o', l*1.2
			}
			wd.put(x+dx, y+dy, ch, dim(c, l), dy == 0, dy < 0)
		}
	}
}

func (wd *World) pots(x, y int, seed uint32) {
	for k := 0; k < 3; k++ {
		px := x + k*3 - 3
		c := mix(0xb06a40, 0xc8a070, rnd01(px, y, seed+920))
		wd.put(px, y-2, 'o', dim(c, 1.1), false, true)
		wd.put(px, y-1, '@', c, false, true)
		wd.put(px, y, 'U', dim(c, 0.7), true, false)
	}
}

func (wd *World) woodpile(x, y int, seed uint32) {
	for dy := -2; dy <= 0; dy++ {
		for dx := -3 - dy; dx <= 3+dy; dx++ {
			wd.put(x+dx, y+dy, "oO0"[hash2(x+dx, y+dy, seed+921)%3], dim(0x9a6a3a, 0.8+0.3*rnd01(x+dx, y+dy, seed+922)), dy == 0, dy < 0)
		}
	}
}

func (wd *World) cart(x, y int, seed uint32) {
	for dx := -5; dx <= 5; dx++ {
		wd.put(x+dx, y-3, '=', dim(0x8a6440, 1.0), false, true)
		wd.put(x+dx, y-2, '#', dim(0x7a5632, 0.8), false, true)
	}
	for _, wx := range []int{x - 4, x + 3} { // rodas
		wd.put(wx, y-1, '(', 0x5a3c22, true, false)
		wd.put(wx+1, y-1, ')', 0x5a3c22, true, false)
		wd.put(wx, y, 'o', 0x3a2818, true, false)
	}
	wd.put(x+6, y-2, '-', 0x6a4a2c, false, true)
	wd.put(x+7, y-2, '-', 0x6a4a2c, false, true)
	for k := 0; k < 4; k++ { // carga: sacos e abóboras
		wd.put(x-3+k*2, y-4, "Oo@o"[k], []rgb{0xc8b080, 0xe08030, 0xc8b080, 0x80a040}[k], false, true)
	}
}

// rug estende um tapete de padrão losangular no chão (não bloqueia).
func (wd *World) rug(x0, y0, w, h int, a, b rgb, seed uint32) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			ch, c := byte(':'), a
			dx, dy := x-x0-w/2, y-y0-h/2
			switch {
			case x == x0 || x == x0+w-1 || y == y0 || y == y0+h-1:
				ch, c = '#', dim(b, 0.8)
			case (abs(dx)+abs(dy*2))%6 < 2:
				ch, c = '+', b
			}
			wd.put(x, y, ch, dim(c, 0.75+0.2*rnd01(x, y, seed+930)), false, false)
		}
	}
}

func (wd *World) fence(x0, y, n int, seed uint32) {
	for k := 0; k < n; k++ {
		x := x0 + k
		ch := byte('-')
		if k%4 == 0 {
			ch = '|'
			wd.put(x, y-1, 'i', dim(0x7a5632, 0.9), false, true)
		}
		wd.put(x, y, ch, dim(0x7a5632, 0.85+0.2*rnd01(x, y, seed+931)), true, false)
	}
}

// village enche a vila: cabanas dos mercadores com placa, tapetes, caixotes,
// barris, potes, lenha, uma carroça e cerca nas hortas.
func (wd *World) villageDressing(seed uint32) {
	vx, vy := villageX, villageY
	mx, my := wd.shopHut(vx-58, vy+18, "Ferreiro", 0xc89a50, []rgb{0xb8c0d8, 0xc8c8d8, 0x9aa0b0}, seed+940)
	wd.moveShop("Morvan, o ferreiro", mx, my)
	hx, hy := wd.shopHut(vx+64, vy+18, "Herbalista", 0x80b060, []rgb{0x8ae060, 0xe04058, 0xff8a40, 0x60d0a0}, seed+941)
	wd.moveShop("Ysolde, a herbalista", hx, hy)

	wd.rug(vx-14, vy+2, 12, 6, 0x8a2a3a, 0xe0b060, seed)
	wd.rug(vx+20, vy-6, 10, 5, 0x2a4a7a, 0xd0a050, seed+1)
	for _, c := range [][2]int{{vx - 80, vy + 8}, {vx - 84, vy + 12}, {vx + 88, vy + 4}} {
		wd.crate(c[0], c[1], seed)
	}
	for _, b := range [][2]int{{vx - 74, vy + 14}, {vx + 84, vy + 12}, {vx + 30, vy - 30}} {
		wd.barrel(b[0], b[1], seed)
	}
	wd.pots(vx-30, vy+26, seed)
	wd.pots(vx+44, vy-28, seed+2)
	wd.woodpile(vx-24, vy-30, seed)
	wd.woodpile(vx+12, vy+40, seed+3)
	wd.cart(vx+100, vy-20, seed)
	wd.fence(vx-118, vy+68, 30, seed)
	wd.fence(vx+56, vy+72, 34, seed)
}

func (wd *World) moveShop(name string, x, y int) {
	for i := range wd.shops {
		if wd.shops[i].Name == name {
			wd.shops[i].X, wd.shops[i].Y = x, y
		}
	}
}

// Placas de direção: no fim de cada trilha que sai de um lugar conhecido,
// um poste com o nome da região para onde ela leva.
func (wd *World) signposts(seed uint32) {
	for i, p := range paths {
		ax, ay, bx, by := p[0], p[1], p[2], p[3]
		d := dist(ax, ay, bx, by)
		if d < 120 {
			continue
		}
		dest := regionAt(int(bx), int(by))
		if dest < 0 {
			continue
		}
		a := math.Atan2(by-ay, bx-ax)
		// Procura, a partir de 40 células da saída, um lugar livre ao lado da trilha.
	try:
		for _, along := range []float64{40, 55, 70, 85, 100, 120} {
			for _, side := range []float64{1, -1} {
				t := along / d
				sx, sy := ax+(bx-ax)*t, ay+(by-ay)*t
				px, py := int(sx-math.Sin(a)*11*side), int(sy+math.Cos(a)*7*side)
				if !wd.fits(px, py) || !wd.fits(px, py-2) || plazaDist(float64(px), float64(py)) < 1.5 ||
					nearLandmark(float64(px), float64(py), 20) || !wd.clearFor(px-24, py-8, px+24, py) {
					continue
				}
				wd.signpost(px, py, []string{regions[dest].Name}, []float64{a}, seed+uint32(i)*3)
				wd.signs = append(wd.signs, travelPt{signName(dest, px, py), px, py, 1})
				break try
			}
		}
	}
}

// clearFor diz se o retângulo não tem nada construído (nem alto nem sólido),
// para uma placa não cair em cima de casa, cabana ou árvore.
func (wd *World) clearFor(x0, y0, x1, y1 int) bool {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if !wd.inBounds(x, y) {
				return false
			}
			if i := y*wd.w + x; wd.over[i] || wd.solid[i] && !wd.water[i] {
				return false
			}
		}
	}
	return true
}
