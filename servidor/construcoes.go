package main

import (
	"math"
)

// cactus planta um cacto de braços com o pé em (cx, cy).
func (wd *World) cactus(cx, cy int, seed uint32) {
	for dy := -8; dy <= 0; dy++ {
		for dx := -3; dx <= 3; dx++ {
			trunk := dx >= -1 && dx <= 1
			arm := (dx == -3 || dx == 3) && dy >= -6 && dy <= -3 || (dx == -2 || dx == 2) && dy == -3
			if trunk || arm {
				wd.put(cx+dx, cy+dy, "#H|8"[hash2(dx, dy, seed+240)%4], dim(0x5aa050, 1.1-0.1*float64(dx+3)), dy >= -1 && trunk, false)
			}
		}
	}
}

// pillar ergue uma coluna com a base em (cx, cy); só o pé bloqueia.
func (wd *World) pillar(cx, cy, h int, c rgb, seed uint32) {
	for y := cy - h; y <= cy; y++ {
		for x := cx - 3; x <= cx+3; x++ {
			wd.put(x, y, pick(stoneGlyphs, hash2(x, y, seed+106)), dim(c, 1.05-0.12*float64(x-cx+3)), y >= cy-3, false)
		}
	}
}

// tier é um andar de construção vista de frente: beiral colorido em cima,
// parede de blocos embaixo.
func (wd *World) tier(x0, y0, w, h int, wall, eave rgb, seed uint32) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			l := (1.1 - 0.4*float64(x-x0)/float64(w)) * (0.85 + 0.3*rnd01(x, y, seed+110))
			ch, c := pick(stoneGlyphs, hash2(x, y, seed+111)), wall
			switch row := y - y0; {
			case row < 3:
				ch, c = "oo0o"[hash2(x, y, seed)%4], eave
			case row%4 == 0 || (x+row/4%2*4)%8 == 0:
				l, ch = l*0.55, '='
			}
			wd.put(x, y, ch, dim(c, math.Min(1.25, l)), true, false)
		}
	}
}

// citadel monta a Cidadela Asura no meio da selva: pagode de três andares ao
// norte, pátio de lajes, colunas em volta e duas lanternas na entrada.
func (wd *World) citadel(seed uint32) {
	for y := citadelY - 50; y <= citadelY+50; y++ {
		for x := citadelX - 50; x <= citadelX+50; x++ {
			if dist(float64(x), float64(y), citadelX, citadelY) > 46 {
				continue
			}
			ch, c := byte(' '), dim(mix(0x4a3a5e, 0x6a567e, fbm(float64(x)/18, float64(y)/18, seed+112)), 0.6+0.4*rnd01(x, y, seed+113))
			switch {
			case x%12 == 0 || y%8 == 0:
				ch, c = ':', dim(c, 0.6)
			case rnd01(x, y, seed+114) < 0.4:
				ch = pick(groundGlyphs, hash2(x, y, seed+115))
			}
			wd.put(x, y, ch, c, false, false)
		}
	}
	wd.tier(citadelX-34, citadelY-52, 68, 16, 0x8a6a9a, 0xd0a040, seed)
	wd.tier(citadelX-24, citadelY-66, 48, 14, 0x8a6a9a, 0xc04040, seed)
	wd.tier(citadelX-14, citadelY-78, 28, 12, 0x8a6a9a, 0xd0a040, seed)
	for k := 0.0; k < 8; k++ {
		a := k*math.Pi/4 + 0.4
		if math.Sin(a) > -0.5 { // o lado norte é do pagode
			wd.pillar(citadelX+int(40*math.Cos(a)), citadelY+int(40*math.Sin(a)), 9, 0xb090c0, seed)
		}
	}
	wd.lights = append(wd.lights, light{citadelX - 18, citadelY - 32, 45, 1}, light{citadelX + 18, citadelY - 32, 45, 1})
}

// bastion monta o Bastião Radiante: salão de mármore ao norte e um anel de
// cristais que iluminam a praça.
func (wd *World) bastion(seed uint32) {
	wd.tier(radiantX-40, radiantY-78, 80, 24, 0xe0d8b8, 0xf0d060, seed)
	for x := radiantX - 34; x <= radiantX+34; x += 17 {
		wd.pillar(x, radiantY-52, 12, 0xf0ecd8, seed)
	}
	for k := 0.0; k < 8; k++ {
		a := k*math.Pi/4 + math.Pi/8
		cx, cy := radiantX+int(62*math.Cos(a)), radiantY+int(48*math.Sin(a))
		// Cristal: losango alto, branco no miolo e azulado nas bordas.
		for dy := -13; dy <= 0; dy++ {
			half := 2 - int(math.Abs(float64(dy+6))/3.3)
			for dx := -half; dx <= half; dx++ {
				wd.put(cx+dx, cy+dy, "@#%&"[hash2(dx, dy, seed+120)%4], mix(0xffffff, 0x8ad8ff, math.Abs(float64(dx))/2.5), dy >= -1, false)
				wd.lit(cx+dx, cy+dy)
			}
		}
		wd.lights = append(wd.lights, light{cx, cy - 6, 42, 2})
	}
}

// crypt escava a cripta: piso de lajes, paredes de pedra em volta (mais
// altas do lado norte, para dar profundidade), pilares, túmulos e tochas.
func (wd *World) crypt(seed uint32) {
	floor := func(x, y int) bool {
		y -= cryptDY
		for _, r := range cryptRooms {
			if x >= r[0] && x < r[0]+r[2] && y >= r[1] && y < r[1]+r[3] {
				return true
			}
		}
		return false
	}
	for y := overH + 80; y < deepY0; y++ {
		for x := 0; x < wd.w; x++ {
			if floor(x, y) {
				ch, c := byte(' '), dim(mix(0x34344c, 0x50506e, fbm(float64(x)/20, float64(y)/20, seed+100)), 0.6+0.4*rnd01(x, y, seed+101))
				switch {
				case x%12 == 0 || y%8 == 0:
					ch, c = ':', dim(c, 0.6)
				case rnd01(x, y, seed+102) < 0.4:
					ch = pick(groundGlyphs, hash2(x, y, seed+103))
				}
				wd.put(x, y, ch, c, false, false)
				continue
			}
			// Parede: há piso até 2 células acima ou até 7 abaixo.
			for dy := -2; dy <= 7; dy++ {
				if floor(x-2, y+dy) || floor(x, y+dy) || floor(x+2, y+dy) {
					l := 0.55 + 0.3*rnd01(x, y, seed+104)
					ch := pick(stoneGlyphs, hash2(x, y, seed+105))
					if y%4 == 0 || (x+y/4%2*4)%8 == 0 {
						l, ch = l*0.55, '='
					}
					if dy < 0 {
						l *= 0.5
					}
					wd.put(x, y, ch, dim(0x9c9abc, l), true, false)
					break
				}
			}
		}
	}

	for _, p := range [][2]int{{285, 690}, {325, 690}, {285, 722}, {325, 722}, {415, 680}, {475, 680}, {415, 722}, {475, 722}} {
		wd.pillar(p[0], p[1]+cryptDY, 9, 0xa4a2c4, seed)
	}
	for i, t := range [][2]int{{400, 660}, {430, 656}, {460, 656}, {490, 660}, {272, 672}, {338, 672}} {
		wd.tomb(t[0], t[1]+cryptDY, i%2 == 1, seed)
	}
	// Escada de saída, encostada na parede norte da primeira sala.
	for k := 0; k < 8; k++ {
		for x := cryptOutX - 7 + k/2; x <= cryptOutX+7-k/2; x++ {
			wd.put(x, cryptOutY-2-k, '=', dim(0xc8c8e4, 1-0.09*float64(k)), false, false)
		}
	}
	for _, t := range [][2]int{{150, 676}, {200, 676}, {275, 656}, {335, 656}, {405, 641}, {445, 641}, {485, 641}, {235, 695}, {370, 689}} {
		wd.lights = append(wd.lights, light{t[0], t[1] + cryptDY, 45, 1})
	}
	for _, sp := range cryptSpawns {
		wd.spawns = append(wd.spawns, spawnPt{sp.X, sp.Y + cryptDY, sp.Kind})
	}
}

// tree desenha uma árvore com o pé do tronco em (cx, cy). A copa fica por
// cima de quem passa atrás; só o pé do tronco bloqueia.
func (wd *World) tree(cx, cy, r int, seed uint32) {
	pal := leafPal(cx, cy)
	for y := cy - r/2 - 2; y <= cy; y++ {
		for x := cx - 1; x <= cx+1; x++ {
			wd.put(x, y, pick(woodGlyphs, hash2(x, y, seed+71)), dim(0x6a4a34, 1-0.25*float64(x-cx+1)), y >= cy-1, false)
		}
	}
	ccy := cy - r - r/2
	for dy := -r; dy <= r; dy++ {
		for dx := -r - 2; dx <= r+2; dx++ {
			x, y := cx+dx, ccy+dy
			e := math.Hypot(float64(dx)/1.15, float64(dy))/float64(r) + (valueNoise(float64(x)/4, float64(y)/4, seed+72)-0.5)*0.35
			if e > 1 {
				continue
			}
			ch, c := leaf(x, y, seed, clamp01(0.75-0.35*float64(dx+dy)/float64(r)), pal)
			if ch != ' ' || e < 0.7 {
				wd.put(x, y, ch, c, false, true)
			}
		}
	}
}

// tomb desenha uma lápide (ou cruz) com a base em (cx, cy).
func (wd *World) tomb(cx, cy int, cross bool, seed uint32) {
	for dy := -6; dy <= 0; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if cross && dx != 0 && dy != -4 || !cross && dy == -6 && (dx == -2 || dx == 2) {
				continue
			}
			x, y := cx+dx, cy+dy
			wd.put(x, y, pick(stoneGlyphs, hash2(x, y, seed+91)), dim(0xa2a2bc, 0.95-0.12*float64(dx+2)), dy >= -1, false)
		}
	}
}

// house desenha uma casa vista de frente: telhado de duas águas em empena
// (telha com sombra em cada fileira, beiradas / e \), chaminé, parede em
// enxaimel (vigas escuras, reboco claro), escuro sob o beiral, porta de
// madeira e janelas acesas.
func (wd *World) house(x0, y0, w, h int, seed uint32) {
	roofH := h * 11 / 20
	wd.gableRoof(x0, y0+roofH-1, w, roofH, roofPal, seed)
	wd.chimney(x0+w*3/4-1, y0+roofH/2, 8, seed+74)
	// Janelinha redonda na empena.
	for dx := -1; dx <= 1; dx++ {
		wd.put(x0+w/2+dx, y0+roofH/2+1, 'o', 0xffc860, true, false)
		wd.lit(x0+w/2+dx, y0+roofH/2+1)
	}
	cx := x0 + w/2
	for y := y0 + roofH; y < y0+h; y++ {
		wr := y - y0 - roofH
		for x := x0; x < x0+w; x++ {
			cxr := x - cx
			t := float64(x-x0) / float64(w)
			ch, c, l := pick("#H", hash2(x, y, seed+73)), rgb(0xa89878), 1.0-0.3*t
			switch {
			case wr == 0:
				ch, c, l = ' ', 0, 0 // sombra funda sob o beiral
			case (x-x0)%7 == 0 || x == x0+w-1:
				ch, c = '|', 0x4e3620 // viga
			case wr == (h-roofH)/2:
				ch, c = '=', 0x4e3620
			}
			switch {
			case cxr >= -3 && cxr <= 3 && y >= y0+h-9:
				// Porta de madeira em arco, com tábuas e argola.
				ch, c, l = '|', 0x6a4422, 0.9
				if cxr == -3 || cxr == 3 || y == y0+h-9 && (cxr == -3+1 || cxr == 3-1) {
					ch, c, l = ' ', 0, 0
				}
				if cxr == 1 && y == y0+h-5 {
					ch, c = 'o', 0xd8b050
				}
			case wr >= 2 && wr <= 6 && (cxr >= -w/4-2 && cxr <= -w/4+2 || cxr >= w/4-2 && cxr <= w/4+2):
				ch, c, l = '#', 0xffd070, 1.0 // janela acesa
				if cxr == -w/4 || cxr == w/4 || wr == 4 {
					ch, c = '+', 0x3a2414 // cruz do caixilho
				}
				if wr == 2 && (cxr == -w/4-2 || cxr == -w/4+2 || cxr == w/4-2 || cxr == w/4+2) {
					ch, c, l = ' ', 0, 0
				}
			}
			wd.put(x, y, ch, dim(c, l*(0.88+0.24*rnd01(x, y, seed+75))), true, false)
			if c == 0xffd070 {
				wd.lit(x, y)
			}
		}
	}
}

// chapel desenha a capela de pedra com o meio da soleira em (cx, by):
// corpo, empena, torre com cruz, rosácea, janelas e a porta em arco.
func (wd *World) chapel(cx, by int, seed uint32) {
	const bw, bh, roofH = 56, 46, 28
	x0, top := cx-bw/2, by-bh
	stone := func(x, y int, k float64) {
		l := (1.05 - 0.4*float64(x-x0)/bw) * k * (0.85 + 0.3*rnd01(x, y, seed+80))
		ch := pick(stoneGlyphs, hash2(x, y, seed+81))
		// Juntas dos blocos: uma fiada a cada quatro linhas, desencontrada.
		if row := y - top + 400; row%4 == 0 || (x+row/4%2*4)%8 == 0 {
			l, ch = l*0.55, '='
		}
		wd.put(x, y, ch, dim(0x9c9abc, math.Min(1.3, l)), true, false)
	}

	for y := top; y <= by; y++ {
		for x := x0; x < x0+bw; x++ {
			stone(x, y, 1)
		}
	}
	for k := 1; k <= roofH; k++ {
		half := (bw/2 + 3) * (roofH - k) / roofH
		for x := cx - half; x <= cx+half; x++ {
			if x-(cx-half) < 2 || cx+half-x < 2 {
				wd.put(x, top-k, '#', 0xc8c8e4, true, false)
			} else {
				stone(x, top-k, 0.85)
			}
		}
	}

	towerTop := top - roofH - 12
	for y := towerTop; y <= top-roofH+8; y++ {
		for x := cx - 6; x <= cx+6; x++ {
			stone(x, y, 0.95)
			if x >= cx-1 && x <= cx+1 && y >= towerTop+4 && y <= towerTop+11 {
				wd.put(x, y, ':', 0x26305a, true, false)
			}
		}
	}
	for k := 1; k <= 7; k++ {
		for x := cx - 7 + k; x <= cx+7-k; x++ {
			wd.put(x, towerTop-k, '%', dim(0x5c5a84, 1.1-0.05*float64(x-cx+7)), true, false)
		}
	}
	for k := 0; k <= 9; k++ {
		wd.put(cx, towerTop-7-k, '#', 0xd8d8f0, true, false)
	}
	for dx := -2; dx <= 2; dx++ {
		wd.put(cx+dx, towerTop-13, '#', 0xd8d8f0, true, false)
	}

	for dy := -7; dy <= 7; dy++ {
		for dx := -7; dx <= 7; dx++ {
			switch d := math.Hypot(float64(dx), float64(dy)); {
			case d < 5:
				wd.put(cx+dx, top-8+dy, "o*+o"[hash2(dx, dy, seed+82)%4], mix(0x2c3a6a, 0x6a4a8a, rnd01(dx, dy, seed+83)), true, false)
			case d < 6.3:
				wd.put(cx+dx, top-8+dy, '@', 0xd0d0ea, true, false)
			}
		}
	}
	for _, wx := range []int{cx - 18, cx + 18} {
		for y := top + 12; y <= top+30; y++ {
			for dx := -3; dx <= 3; dx++ {
				if y < top+15 && math.Hypot(float64(dx), float64(y-top-15)) > 3.4 {
					continue
				}
				ch, c := byte(':'), rgb(0x26305a)
				if dx == -3 || dx == 3 || y == top+30 || y < top+15 && math.Hypot(float64(dx), float64(y-top-15)) > 2.4 {
					ch, c = '#', 0xc4c4e0
				}
				wd.put(wx+dx, y, ch, c, true, false)
			}
		}
	}

	arch := by - 18
	for y := by - 28; y <= by; y++ {
		for dx := -10; dx <= 10; dx++ {
			d := math.Abs(float64(dx))
			if y < arch {
				d = math.Hypot(float64(dx), float64(y-arch))
			}
			switch {
			case d <= 7.5:
				ch, c := byte(' '), rgb(0)
				if dx%3 == 0 {
					ch, c = '|', 0x2a2438
				}
				wd.put(cx+dx, y, ch, c, true, false)
			case d <= 9.6:
				wd.put(cx+dx, y, '@', 0xd4d4ee, true, false)
			}
		}
	}
	for k := 1; k <= 3; k++ {
		for dx := -9 - k; dx <= 9+k; dx++ {
			wd.put(cx+dx, by+k, '=', dim(0x8684aa, 1-0.12*float64(k)), false, false)
		}
	}
}
