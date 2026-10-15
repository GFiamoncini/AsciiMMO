package main

import "math"

// Ruínas de Vaelor: um castelo em ruínas na margem norte do Lago Espelhado,
// visto de frente como as outras construções (2.5D): a fachada sobe a partir
// da linha de base e tapa o pátio de trás, como a copa das árvores. A base
// encosta na água, e o cliente espelha o castelo no lago.
//
// Peças: muralha da frente com ameias, portão em arco e janelas; quatro torres
// na frente (as dos cantos mais altas, a da direita desabada pela metade);
// muralha e torres do fundo; muralhas laterais vistas de cima, com uma brecha
// a oeste por onde se entra a pé; pátio de lajes rachadas, poço seco e
// entulho; cais de pedra diante do portão. Musgo na base, trepadeiras
// descendo do alto, tijolo à mostra onde a pedra caiu.

const (
	castleX  = lakeX                // meio da fachada
	castleBY = lakeY - lakeRY + 2   // linha de base da fachada da frente, na beira da água
	castleW  = 160                  // largura entre as muralhas laterais
	castleD  = 84                   // profundidade do pátio
	castleX0 = castleX - castleW/2  // muralha oeste
	castleX1 = castleX + castleW/2  // muralha leste
	castleYB = castleBY - castleD   // base da muralha do fundo
	castleH  = 50                   // altura da muralha da frente
	breachY  = castleYB + castleD/2 // meio da brecha da muralha oeste
	castleCY = (castleYB + castleBY) / 2
)

// castleDist vale menos de 1 dentro do terreno do castelo (com folga).
func castleDist(px, py float64) float64 {
	return math.Max(math.Abs(px-castleX)/(castleW/2+20), math.Abs(py-float64(castleYB+castleBY)/2+20)/(castleD/2+66))
}

func (wd *World) castle(seed uint32) {
	stoneC := rgb(0x9c9cb6)
	// stone pinta um bloco de alvenaria da fachada (kit.go): argamassa funda,
	// musgo perto do chão, tijolo onde o reboco caiu; l é a luz da face.
	stone := func(x, y, base int, l float64, solid, over bool) {
		wd.masonry(x, y, base, l, stoneC, solid, over, seed+700)
	}
	// top dá a altura da ruína em cada coluna: o alto da muralha falha aos
	// pedaços, e merlões de 4 com vãos de 3 viram ameias.
	merlon := func(x, top, base, y int) bool {
		return y-top < 4 && (x+400)%7 >= 4 // vão entre os merlões
	}
	vine := func(x, top, y int) bool {
		h := hash2(x, 0, seed+705)
		return h%9 == 0 && y-top < 6+int(h>>8%22)
	}
	leafAt := func(x, y int) {
		wd.put(x, y, pick(leafGlyphs, hash2(x, y, seed+706)), dim(0x3f7a3a, 0.7+0.45*rnd01(x, y, seed+707)), wd.solid[y*wd.w+x], wd.over[y*wd.w+x])
	}

	// 1. Terreno limpo: nada de mata, rocha ou água dentro do castelo.
	for y := castleYB - 80; y <= castleBY; y++ {
		for x := castleX0 - 16; x <= castleX1+16; x++ {
			wd.put(x, y, ' ', 0, false, false)
			wd.ground(x, y, seed)
		}
	}
	// Fosso: a água encosta na base da fachada, para o reflexo.
	for y := castleBY + 1; y <= castleBY+6; y++ {
		for x := castleX0 - 16; x <= castleX1+16; x++ {
			wd.sea(x, y, seed)
		}
	}

	// 2. Pátio: lajes rachadas, capim nas juntas.
	for y := castleYB + 1; y < castleBY; y++ {
		for x := castleX0 + 7; x <= castleX1-7; x++ {
			c := dim(mix(0x45455a, 0x5e5c74, fbm(float64(x)/16, float64(y)/16, seed+710)), 0.6+0.4*rnd01(x, y, seed+711))
			ch := byte(' ')
			switch {
			case fbm(float64(x)/7, float64(y)/7, seed+712) > 0.62:
				ch, c = pick("'\",;", hash2(x, y, seed+713)), dim(0x3c7046, 0.7+0.4*rnd01(x, y, seed+714))
			case x%10 == 0 || y%6 == 0:
				ch, c = ':', dim(c, 0.6)
			case rnd01(x, y, seed+715) < 0.35:
				ch = pick(groundGlyphs, hash2(x, y, seed+716))
			}
			wd.put(x, y, ch, c, false, false)
		}
	}

	// 3. Muralha e torres do fundo: tudo bloqueia (atrás é fora do castelo).
	for x := castleX0; x <= castleX1; x++ {
		top := castleYB - 24 + int(math.Max(0, (fbm(float64(x)/11, 3, seed+720)-0.5)*30))
		for y := top; y <= castleYB; y++ {
			if merlon(x, top, castleYB, y) {
				continue
			}
			stone(x, y, castleYB, 0.75-0.2*float64(x-castleX0)/castleW, true, false)
		}
	}
	wd.tower(castleX0+10, castleYB+2, 11, 52, 0.75, true, seed+721, stone, merlon)
	wd.tower(castleX1-10, castleYB+2, 11, 46, 0.75, true, seed+722, stone, merlon)

	// 4. Muralhas laterais, vistas de cima: o caminho de ronda claro, o
	// parapeito de fora mais alto e escuro. A oeste, a brecha.
	for y := castleYB; y <= castleBY; y++ {
		for _, side := range []int{-1, 1} {
			edge := castleX0
			if side > 0 {
				edge = castleX1
			}
			for k := 0; k <= 6; k++ {
				x := edge - side*k
				if side < 0 && y > breachY-9 && y < breachY+9 {
					// Brecha: entulho espalhado; dá para passar pelo meio.
					if rnd01(x, y, seed+730) < 0.35 && math.Abs(float64(y-breachY)) > 3 {
						wd.put(x, y, pick(stoneGlyphs, hash2(x, y, seed+731)), dim(stoneC, 0.5+0.4*rnd01(x, y, seed+732)), true, false)
					} else {
						wd.put(x, y, pick(".,:", hash2(x, y, seed+733)), dim(0x6a6a80, 0.6), false, false)
					}
					continue
				}
				l := 1.0
				ch := pick(stoneGlyphs, hash2(x, y, seed+734))
				if k <= 1 {
					l = 0.6
					if (y+400)%7 < 3 {
						ch, l = '#', 0.8 // merlões do parapeito
					}
				} else if y%4 == 0 {
					ch, l = '=', 0.6
				}
				wd.put(x, y, ch, dim(stoneC, l*(0.85+0.3*rnd01(x, y, seed+735))), true, false)
			}
		}
	}

	// 5. Coisas do pátio: poço seco e montes de entulho.
	wx, wy := castleX+30, castleYB+26
	for dy := -5; dy <= 5; dy++ {
		for dx := -9; dx <= 9; dx++ {
			d := math.Hypot(float64(dx)/1.7, float64(dy))
			switch {
			case d < 3:
				wd.put(wx+dx, wy+dy, ' ', 0, true, false)
			case d < 5:
				wd.put(wx+dx, wy+dy, pick(stoneGlyphs, hash2(dx, dy, seed+740)), dim(stoneC, 1.1-0.08*float64(dx+dy)), true, false)
			}
		}
	}
	for i := 0; i < 9; i++ {
		h := hash2(i, 0, seed+741)
		rx, ry := castleX0+14+int(h%uint32(castleW-28)), castleYB+6+int(h>>10%uint32(castleD-14))
		for dy := -2; dy <= 0; dy++ {
			for dx := -3 - dy; dx <= 3+dy; dx++ {
				if rnd01(rx+dx, ry+dy, seed+742) < 0.75 {
					wd.put(rx+dx, ry+dy, pick(stoneGlyphs, hash2(rx+dx, ry+dy, seed+743)), dim(stoneC, 0.55+0.4*rnd01(rx+dx, ry+dy, seed+744)), dy == 0, false)
				}
			}
		}
	}

	// 6. Muralha da frente: só as 3 linhas de baixo bloqueiam; o resto passa
	// por cima de quem anda no pátio. Portão em arco no meio, janelas em arco
	// dos lados, e a brecha grande à direita, onde o alto desabou.
	gateHalf, gateTop := 9, castleBY-16
	for x := castleX0; x <= castleX1; x++ {
		ruin := int(math.Max(0, (fbm(float64(x)/8, 7, seed+750)-0.52)*26))
		if x > castleX+30 && x < castleX+54 {
			ruin = 32 + int(rnd01(x, 0, seed+751)*6) // a brecha
		}
		top := castleBY - castleH + ruin
		for y := top; y <= castleBY; y++ {
			solid, over := castleBY-y < 3, castleBY-y >= 3
			dx := x - castleX
			// Portão: vão escuro, com a grade enferrujada só no alto do arco.
			ad := math.Abs(float64(dx))
			if y < gateTop {
				ad = math.Hypot(float64(dx), float64(y-gateTop))
			}
			if ad <= float64(gateHalf) {
				ch, c := byte(' '), rgb(0)
				if y < gateTop+2 && (dx%3 == 0 || y%3 == 0) {
					ch, c = '#', 0x4a3a30
				}
				wd.put(x, y, ch, c, false, castleBY-y >= 3)
				continue
			}
			if ad <= float64(gateHalf)+2 {
				wd.put(x, y, '@', dim(0xb4b4cc, 1.05-0.02*float64(dx)), solid, over)
				continue
			}
			if merlon(x, top, castleBY, y) {
				continue
			}
			// Janelas em arco em dois andares, duas de cada lado do portão; as
			// de cima, menores. Vãos escuros, que é o que dá fundo à parede.
			win := false
			for _, wxo := range []int{-58, -36, 36, 58} {
				if x > castleX+30 && x < castleX+54 {
					break
				}
				for fl, wy := range []int{castleBY - 20, castleBY - 38} {
					hh := 10 - 3*fl
					ddx, ddy := float64(x-castleX-wxo), float64(y-wy)
					if ddy > -float64(hh) && ddy <= 0 && math.Abs(ddx) <= 2 || ddy <= -float64(hh) && math.Hypot(ddx, ddy+float64(hh)) <= 2.4 {
						win = true
					}
				}
			}
			if win {
				wd.put(x, y, ':', 0x1a1c30, solid, over)
				continue
			}
			stone(x, y, castleBY, 1.08-0.3*float64(x-castleX0)/castleW, solid, over)
			if vine(x, top, y) {
				leafAt(x, y)
			}
		}
		// Entulho da brecha caído no fosso.
		if x > castleX+28 && x < castleX+56 && rnd01(x, 1, seed+752) < 0.5 {
			wd.put(x, castleBY+1+int(rnd01(x, 2, seed+753)*2), pick(stoneGlyphs, hash2(x, 3, seed+754)), dim(stoneC, 0.55), true, false)
		}
	}

	// 7. Torres da frente: as dos cantos avançam sobre a água; a da direita
	// desabou pela metade.
	wd.tower(castleX0, castleBY+3, 13, 74, 1.05, false, seed+760, stone, merlon)
	wd.tower(castleX1, castleBY+3, 13, 40, 0.9, false, seed+761, stone, merlon)
	wd.tower(castleX-gateHalf-9, castleBY+1, 7, 64, 1.0, false, seed+762, stone, merlon)
	wd.tower(castleX+gateHalf+9, castleBY+1, 7, 60, 0.95, false, seed+763, stone, merlon)
	for k := 0; k < 14; k++ {
		x := castleX1 - 16 + int(rnd01(k, 4, seed+764)*32)
		wd.put(x, castleBY+4+int(rnd01(k, 5, seed+765)*3), pick(stoneGlyphs, hash2(k, 6, seed+766)), dim(stoneC, 0.5), true, false)
	}
	// Telhado cônico só nas torres que ficaram de pé; estandartes vermelhos
	// rasgados no alto; braseiros acesos no cais (fogo quente contra a pedra
	// fria, refletido no fosso).
	slate := rgb(0x4a5a7a)
	wd.coneRoof(castleX-gateHalf-9, castleBY+1-64+3, 9, 16, slate, true, seed+767)
	wd.coneRoof(castleX0+10, castleYB+2-52+3, 13, 20, slate, false, seed+768)
	wd.banner(castleX0+1, castleBY+3-74+4, 10, 5, 15, 0xb02a2a, true, seed+769)
	wd.banner(castleX+gateHalf+10, castleBY+1-60+4, 8, 4, 12, 0x9a2424, true, seed+770)

	// No alto: braseiros nas torres (quentes) e duas janelas da torre alta
	// com luz fria de assombração; tudo isso se espelha no fosso à noite.
	wd.brazier(castleX+gateHalf+9, castleBY+1-60+12, true)
	wd.brazier(castleX0+5, castleBY+3-74+14, true)
	for _, up := range []int{34, 50} {
		for dy := 0; dy < 4; dy++ {
			for dx := -1; dx <= 1; dx++ {
				wd.put(castleX0+dx-4, castleBY+3-up+dy, '#', 0x9ad8ff, false, true)
				wd.lit(castleX0+dx-4, castleBY+3-up+dy)
			}
		}
		wd.lights = append(wd.lights, light{castleX0 - 4, castleBY + 3 - up + 2, 22, 2})
	}
	// 8. Cais de lajes diante do portão, sobre o fosso.
	for y := castleBY + 1; y <= castleBY+9; y++ {
		for x := castleX - gateHalf - 2; x <= castleX+gateHalf+2; x++ {
			ch, l := byte('='), 0.8-0.04*float64(y-castleBY)
			if x == castleX-gateHalf-2 || x == castleX+gateHalf+2 {
				ch, l = '#', 0.6
			}
			wd.put(x, y, ch, dim(0x8a8498, l*(0.85+0.3*rnd01(x, y, seed+770))), false, false)
		}
	}
	for _, lx := range []int{castleX - gateHalf, castleX + gateHalf} {
		wd.brazier(lx, castleBY+8, false)
	}

	// 9. Cofre trancado no canto do pátio (Chave de Ferro, do ferreiro).
	wd.vault(vaults[0][0], vaults[0][1], seed+790)

	// 10. Quem assombra o pátio.
	wd.scatter(mobWarrior, castleX, castleCY, 60, 0, 4, seed+780)
	wd.scatter(mobSkeleton, castleX, castleCY, 66, 0, 4, seed+781)
}

// tower ergue uma torre redonda vista de frente com a base em (cx, by) e meia
// largura r: luz de cilindro, seteiras, ameias no alto e o topo quebrado.
// Só a base bloqueia, a não ser que solid peça a torre inteira.
func (wd *World) tower(cx, by, r, h int, k float64, solid bool, seed uint32,
	stone func(x, y, base int, l float64, solid, over bool), merlon func(x, top, base, y int) bool) {
	for x := cx - r; x <= cx+r; x++ {
		t := float64(x-cx) / (float64(r) + 0.5)
		// Cilindro: a luz vem da esquerda; as bordas escurecem.
		l := k * (0.62 + 0.55*(math.Sqrt(math.Max(0, 1-t*t))*0.7-t*0.45))
		top := by - h + int(math.Max(0, (fbm(float64(x)/4, float64(h), seed)-0.48)*18))
		for y := top; y <= by; y++ {
			if merlon(x, top, by, y) {
				continue
			}
			up := by - y
			// Seteiras: fendas escuras em duas alturas.
			if x == cx && (up > 14 && up < 21 || up > h-22 && up < h-15) {
				wd.put(x, y, ':', 0x14141f, solid || up < 4, !solid && up >= 4)
				continue
			}
			// Cinta de pedra mais clara a cada 12 linhas.
			ll := l
			if up%12 == 11 {
				ll *= 1.25
			}
			stone(x, y, by, ll, solid || up < 4, !solid && up >= 4)
		}
	}
}
