package main

import "math"

// Kit de construção: peças que se combinam para montar castelo, vila e
// ruínas sem repetir desenho à mão. Regras do estilo:
//   - silhueta primeiro: ameias, telhados pontudos, torres de alturas
//     diferentes, chaminés e mastros com bandeira;
//   - textura com contraste: blocos claros e argamassa bem escura (vira vazio),
//     telha com sombra em cada fileira, madeira com veio;
//   - caractere pela face: parede vertical usa | = #, telhado e topo usam ^ : .,
//     e as beiradas usam / e \;
//   - vãos escuros: portas, janelas e arcos são buracos pretos (oclusão);
//   - quente contra frio: pedra azulada com fogo, janela acesa e estandarte vermelho;
//   - desgaste: musgo na base, hera descendo, blocos faltando.

// masonry pinta um bloco de alvenaria vista de frente. base é a linha do chão
// da parede (para o musgo e as fiadas), l a luz da face.
func (wd *World) masonry(x, y, base int, l float64, c rgb, solid, over bool, seed uint32) {
	row := base - y + 400
	ch := pick("#H8B&0", hash2(x, y, seed+800))
	mortar := row%4 == 0 || (x+row/4%2*4)%8 == 0
	if fbm(float64(x)/9, float64(y)/7, seed+801) > 0.72 {
		c = mix(0x9a5236, 0xb86a44, rnd01(x, y, seed+802)) // tijolo onde o reboco caiu
		mortar = row%2 == 0 || (x+row/2%2*2)%4 == 0
	}
	l *= 0.85 + 0.3*rnd01(x, y, seed+803)
	if up := base - y; up < 7 {
		c = mix(c, 0x3e6a3a, clamp01(float64(7-up)/7*fbm(float64(x)/5, float64(y)/5, seed+804)*1.2))
	}
	if mortar {
		ch, l = '.', l*0.4 // argamassa funda: quase vazio
	}
	if rnd01(x, y, seed+805) < 0.025 {
		ch, l = ' ', 0 // bloco que caiu
	}
	wd.put(x, y, ch, dim(c, math.Min(1.05, l)), solid, over)
}

// coneRoof põe um telhado cônico com a beirada em (cx, by), meia largura r e
// altura h: telhas ^ em fileiras com sombra embaixo de cada uma, luz de cone
// vindo da esquerda e as beiradas desenhadas com / e \.
func (wd *World) coneRoof(cx, by, r, h int, c rgb, over bool, seed uint32) {
	for k := 0; k <= h; k++ {
		y := by - k
		half := float64(r+1) * float64(h-k) / float64(h)
		for x := cx - int(half); x <= cx+int(half); x++ {
			t := float64(x-cx) / (half + 0.5)
			l := 0.55 + 0.5*(math.Sqrt(math.Max(0, 1-t*t))*0.6-t*0.5)
			ch := byte('^')
			if k%2 == 0 {
				ch, l = ':', l*0.6 // sombra sob a fileira de telhas
			}
			switch {
			case x == cx-int(half):
				ch, l = '/', l*1.1
			case x == cx+int(half):
				ch = '\\'
			}
			if k == 0 {
				ch, l = '=', 0.45 // beiral
			}
			wd.put(x, y, ch, dim(c, l*(0.9+0.2*rnd01(x, y, seed+810))), !over, over)
		}
	}
	// Remate no bico.
	wd.put(cx, by-h-1, '|', 0x3a3438, !over, over)
	wd.put(cx, by-h-2, '*', 0xd8b050, !over, over)
}

// gableRoof é o telhado de duas águas de uma casa vista de frente: empena em
// triângulo de (x0, by) a (x0+w-1, by), com h linhas.
func (wd *World) gableRoof(x0, by, w, h int, pal []rgb, seed uint32) {
	cx := float64(x0) + float64(w-1)/2
	for k := 0; k < h; k++ {
		y := by - k
		half := (float64(w)/2 + 2) * float64(h-k) / float64(h)
		for x := int(math.Ceil(cx - half)); x <= int(math.Floor(cx+half)); x++ {
			t := (float64(x) - cx) / (half + 0.5)
			l := 1.0 - 0.32*t // a água da esquerda pega sol
			ch, c := byte('^'), pal[(k/2)%len(pal)]
			if k%2 == 0 {
				ch, l = ':', l*0.62
			}
			switch {
			case float64(x)-(cx-half) < 1:
				ch, c, l = '/', 0x5a3a24, 0.9
			case cx+half-float64(x) < 1:
				ch, c, l = '\\', 0x5a3a24, 0.7
			}
			wd.put(x, y, ch, dim(c, l*(0.88+0.24*rnd01(x, y, seed+820))), true, false)
		}
	}
}

// chimney ergue uma chaminé de tijolo com o pé em (x, by) e topo aceso.
func (wd *World) chimney(x, by, h int, seed uint32) {
	for y := by - h; y <= by; y++ {
		for dx := 0; dx < 3; dx++ {
			ch, l := '#', 1.05-0.25*float64(dx)
			if (y+dx)%2 == 0 {
				ch, l = '=', l*0.6
			}
			wd.put(x+dx, y, byte(ch), dim(0x8a4a30, l*(0.85+0.3*rnd01(x+dx, y, seed+830))), true, false)
		}
	}
	for dx := -1; dx < 4; dx++ {
		wd.put(x+dx, by-h-1, '=', 0x5a4a44, true, false)
	}
}

// banner pendura um estandarte num mastro: o mastro sobe de (x, by) por h
// células e o pano (largura w, comprimento n) cai do alto, rasgado na ponta.
func (wd *World) banner(x, by, h, w, n int, c rgb, over bool, seed uint32) {
	for y := by - h; y <= by; y++ {
		wd.put(x, y, '|', 0x6a5440, !over && by-y < 2, over)
	}
	wd.put(x, by-h-1, 'o', 0xd8b050, false, over)
	for k := 0; k < n; k++ {
		for dx := 1; dx <= w; dx++ {
			// Ponta em V e rasgos sorteados.
			if k > n-3 && (k-(n-3)) > int(math.Abs(float64(dx)-float64(w+1)/2)) || rnd01(dx, k, seed+840) < 0.06 {
				continue
			}
			l := 1.05 - 0.14*float64(dx) - 0.02*float64(k)
			ch := byte('#')
			if k%3 == 1 {
				ch = '='
			}
			if k == 3 && dx == (w+1)/2 {
				ch, l = '+', 1.4 // brasão
			}
			wd.put(x+dx, by-h+k, ch, dim(c, l), false, true)
		}
	}
}

// brazier põe um braseiro de pé em (x, y): a bacia é desenhada aqui e a
// chama vem do cliente (luz tipo tocha).
func (wd *World) brazier(x, y int, high bool) {
	wd.put(x-2, y-2, '\\', 0x6a5a50, !high, high)
	wd.put(x-1, y-1, '_', 0x6a5a50, !high, high)
	wd.put(x, y-1, '_', 0x6a5a50, !high, high)
	wd.put(x+1, y-1, '_', 0x6a5a50, !high, high)
	wd.put(x+2, y-2, '/', 0x6a5a50, !high, high)
	wd.put(x, y, '|', 0x4a3e38, !high, high)
	wd.lights = append(wd.lights, light{x, y - 3, 30, 1})
}

// clearArea apaga o que houver (mata, rocha) e refaz o chão num retângulo.
func (wd *World) clearArea(x0, y0, x1, y1 int, seed uint32) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if !wd.inBounds(x, y) || y >= overH || wd.water[y*wd.w+x] {
				continue
			}
			wd.put(x, y, ' ', 0, false, false)
			wd.ground(x, y, seed)
		}
	}
}

// watchtower é uma torre de vigia isolada. Inteira: telhado cônico, mastro
// com estandarte e braseiro no pé. Arruinada: topo quebrado, entulho e hera.
func (wd *World) watchtower(cx, by int, ruined bool, seed uint32) {
	wd.clearArea(cx-22, by-84, cx+22, by+8, seed)
	h := 54
	if ruined {
		h = 32 + int(rnd01(cx, by, seed+850)*12)
	}
	stone := func(x, y, base int, l float64, solid, over bool) {
		wd.masonry(x, y, base, l, 0x8c8ca6, solid, over, seed)
	}
	noCrenel := func(x, top, base, y int) bool { return false }
	crenel := func(x, top, base, y int) bool { return y-top < 3 && (x+400)%5 >= 3 }
	if ruined {
		wd.tower(cx, by, 10, h, 0.95, false, seed+851, stone, noCrenel)
		for k := 0; k < 16; k++ {
			x := cx - 16 + int(rnd01(k, 1, seed+852)*32)
			y := by - 1 + int(rnd01(k, 2, seed+853)*5)
			if x < cx-10 || x > cx+10 || y > by {
				wd.put(x, y, pick(stoneGlyphs, hash2(k, 3, seed+854)), dim(0x8c8ca6, 0.4+0.3*rnd01(k, 4, seed)), rnd01(k, 5, seed) < 0.5, false)
			}
		}
		for x := cx - 10; x <= cx+10; x++ {
			if hash2(x, 0, seed+855)%3 == 0 {
				for y := by - h + 3; y < by-h+3+int(hash2(x, 1, seed+856)%14); y++ {
					wd.put(x, y, pick(leafGlyphs, hash2(x, y, seed+857)), dim(0x3f7a3a, 0.7+0.4*rnd01(x, y, seed)), false, true)
				}
			}
		}
		return
	}
	wd.tower(cx, by, 10, h, 1.0, false, seed+851, stone, crenel)
	wd.coneRoof(cx, by-h+2, 12, 18, 0x4a5a7a, true, seed+858)
	wd.banner(cx+12, by-h-2, 12, 5, 12, 0xb02a2a, true, seed+859)
	wd.brazier(cx-14, by+3, false)
}

// archRow ergue uma fileira de arcos de pedra (claustro ou aqueduto em
// ruínas) com a base em by, a partir de x0: n vãos de largura span.
func (wd *World) archRow(x0, by, n, span, h int, seed uint32) {
	x1 := x0 + n*span + 3
	wd.clearArea(x0-6, by-h-10, x1+6, by+4, seed)
	for x := x0; x <= x1; x++ {
		// Topo quebrado: alguns trechos desabaram até a metade.
		top := by - h + int(math.Max(0, (fbm(float64(x)/6, 11, seed+860)-0.5)*40))
		for y := top; y <= by; y++ {
			k := (x - x0) % span
			if k > 3 && x < x1-3 {
				// Vão do arco: retângulo embaixo, meio círculo em cima.
				cxa := float64(span+3) / 2
				ry := by - h*2/3
				d := math.Abs(float64(k) - cxa)
				if y < ry {
					d = math.Hypot(float64(k)-cxa, float64(y-ry)*1.2)
				}
				if d < cxa-2.2 {
					continue
				}
				if d < cxa-1.2 {
					wd.put(x, y, '@', dim(0xb4b4cc, 0.9), by-y < 2, by-y >= 2) // aduelas do arco
					continue
				}
			}
			wd.masonry(x, y, by, 1.05-0.3*float64(x-x0)/float64(x1-x0), 0x9a96a8, by-y < 2, by-y >= 2, seed)
		}
	}
	for k := 0; k < 20; k++ {
		x := x0 + int(rnd01(k, 7, seed+861)*float64(x1-x0))
		wd.put(x, by+1+int(rnd01(k, 8, seed+862)*3), pick(stoneGlyphs, hash2(k, 9, seed)), dim(0x9a96a8, 0.5), rnd01(k, 10, seed) < 0.4, false)
	}
}

// stoneCircle planta um círculo de menires numa clareira, com a pedra do
// meio caída.
func (wd *World) stoneCircle(cx, cy int, seed uint32) {
	wd.clearArea(cx-50, cy-56, cx+50, cy+30, seed)
	for k := 0; k < 9; k++ {
		a := float64(k)/9*2*math.Pi + 0.3
		x, y := cx+int(36*math.Cos(a)), cy+int(22*math.Sin(a))
		h := 16 + int(hash2(k, 0, seed+870)%10)
		if k == 4 {
			h = 5 // tombada
		}
		for dy := -h; dy <= 0; dy++ {
			half := 3
			if dy < -h+2 {
				half = 2
			}
			for dx := -half; dx <= half; dx++ {
				l := 1.1 - 0.18*float64(dx+half)
				ch := pick("#H8&", hash2(x+dx, y+dy, seed+871))
				if (dy+h)%3 == 0 {
					ch, l = ':', l*0.6
				}
				wd.put(x+dx, y+dy, ch, dim(mix(0x7a7a88, 0x4e6a46, clamp01(float64(dy+4)/4)*0.5), l), dy >= -1, dy < -1)
			}
		}
	}
	// Laje do altar, com marcas acesas.
	for dy := -2; dy <= 1; dy++ {
		for dx := -6; dx <= 6; dx++ {
			ch, c := byte('='), dim(0x7a7a88, 0.9-0.05*float64(dy+2))
			if dy == -1 && dx%3 == 0 {
				ch, c = '*', 0x8ad8ff
			}
			wd.put(cx+dx, cy+dy, ch, c, true, false)
			if ch == '*' {
				wd.lit(cx+dx, cy+dy)
			}
		}
	}
	wd.lights = append(wd.lights, light{cx, cy - 2, 40, 2})
}

// Marcos espalhados pelo mapa: torres de vigia (inteira na vila, arruinadas
// nas trilhas), o claustro em ruínas perto do cemitério e o círculo de
// menires. Árvores, rochas e falésias ficam longe deles.
var landmarks = []struct {
	X, Y, R int
	Kind    string
}{
	{villageX + 135, villageY - 40, 24, "vigia"},
	{hubX - 40, hubY - 210, 24, "vigia-ruina"},
	{770, 1560, 24, "vigia-ruina"},
	{690, 760, 24, "vigia-ruina"},
	{graveX + 100, graveY - 30, 50, "arcos"},
	{villageX - 260, villageY - 120, 50, "menires"},
	{villageX + 178, villageY - 67, 24, "cofre"},
}

func nearLandmark(px, py, r float64) bool {
	for _, l := range landmarks {
		if dist(px, py, float64(l.X), float64(l.Y)) < float64(l.R)+r {
			return true
		}
	}
	return false
}

func (wd *World) buildLandmarks(seed uint32) {
	for i, l := range landmarks {
		s := seed + 900 + uint32(i)*7
		switch l.Kind {
		case "vigia":
			wd.watchtower(l.X, l.Y, false, s)
		case "vigia-ruina":
			wd.watchtower(l.X, l.Y, true, s)
		case "arcos":
			wd.archRow(l.X-42, l.Y, 5, 17, 40, s)
		case "cofre":
			wd.vault(vaults[1][0], vaults[1][1], s)
		case "menires":
			wd.stoneCircle(l.X, l.Y, s)
		}
	}
}

// lit marca a célula como fonte de luz (janela acesa, cristal): o cliente
// não a escurece com a noite.
func (wd *World) lit(x, y int) {
	if wd.inBounds(x, y) && wd.glow != nil {
		wd.glow[y*wd.w+x] = true
	}
}

// Cofres: salas de pedra trancadas com um baú dentro. O centro da parede da
// frente fica em (cx, by); o baú, a 8 células para dentro.
var vaults = [][2]int{
	{castleX + 52, castleYB + 28},   // pátio das Ruínas de Vaelor
	{villageX + 178, villageY - 58}, // depósito da guarda, ao lado da torre de vigia
}

// vault ergue o cofre: paredes de alvenaria vistas de cima (duas de
// espessura), chão de lajes, e na frente uma fachada baixa com a porta de
// madeira reforçada, que é um objeto: fechada ela bloqueia; aberta, vira vão.
func (wd *World) vault(cx, by int, seed uint32) { wd.vaultKind(cx, by, propDoor, seed) }

// vaultKind ergue o cofre com a porta do tipo dado (porta de chave ou selo).
func (wd *World) vaultKind(cx, by, doorKind int, seed uint32) {
	const hw, d = 13, 18 // meia largura e profundidade
	x0, x1, top := cx-hw, cx+hw, by-d
	wd.clearArea(x0-4, top-4, x1+4, by+4, seed)
	stoneC := rgb(0x9a98b0)
	for y := top; y <= by; y++ {
		for x := x0; x <= x1; x++ {
			wall := x-x0 < 2 || x1-x < 2 || y-top < 2
			switch {
			case by-y < 6: // fachada da frente: base bloqueia, o alto passa por cima
				wd.masonry(x, y, by, 1.05-0.3*float64(x-x0)/float64(2*hw), stoneC, by-y < 2, by-y >= 2, seed)
			case wall:
				ch, l := pick(stoneGlyphs, hash2(x, y, seed+880)), 0.95
				if (x+y)%4 == 0 {
					ch, l = '=', 0.6
				}
				wd.put(x, y, ch, dim(stoneC, l*(0.85+0.3*rnd01(x, y, seed+881))), true, false)
			default:
				ch, c := byte(' '), dim(rgb(0x4a4a5c), 0.7+0.3*rnd01(x, y, seed+882))
				if x%6 == 0 || y%5 == 0 {
					ch, c = ':', dim(c, 0.6)
				}
				wd.put(x, y, ch, c, false, false)
			}
		}
	}
	// Vão da porta aberta: escuro em cima, chão embaixo.
	for y := by - 5; y <= by; y++ {
		for x := cx - 2; x <= cx+2; x++ {
			if by-y < 2 {
				wd.put(x, y, ':', 0x3a3a48, false, false)
			} else {
				wd.put(x, y, ' ', 0, false, true)
			}
		}
	}
	// A porta fechada é o objeto: madeira com faixas de ferro e a fechadura.
	on, off := wd.stamp(cx-2, by-5, cx+2, by, func() {
		for y := by - 5; y <= by; y++ {
			for x := cx - 2; x <= cx+2; x++ {
				ch, c := byte('|'), dim(0x7a4e2a, 0.85+0.25*rnd01(x, y, seed+883))
				if (by-y)%3 == 1 {
					ch, c = '=', 0x5a5a68
				}
				if x == cx && y == by-2 {
					ch, c = 'o', 0xd8b050
				}
				wd.put(x, y, ch, c, by-y < 2, by-y >= 2)
			}
		}
	})
	wd.addProp(doorKind, cx, by, on, off)
	wd.lights = append(wd.lights, light{cx - hw - 2, by + 2, 26, 1})
}
