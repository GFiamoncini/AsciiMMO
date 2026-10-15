package main

import (
	"math"
)

// Paletas de copa: escuro e claro.
var (
	leafGreen  = [2]rgb{0x2a5644, 0xa6d8aa}
	leafJungle = [2]rgb{0x1f6a3a, 0xb8e86a}
	leafCoral  = [2]rgb{0x8a3a5a, 0xffb0c0}
	leafSnow   = [2]rgb{0x5a7a8a, 0xf0f6ff} // pinheiros nevados
	leafSwamp  = [2]rgb{0x2a3a26, 0x7a8a4a} // copas escuras e amareladas
)

// leaf devolve o caractere e a cor de uma célula de copa; os vãos escuros
// entre os tufos saem como espaço.
func leaf(x, y int, seed uint32, light float64, pal [2]rgb) (byte, rgb) {
	d := fbm(float64(x)/6, float64(y)/6, seed+60)
	if d < 0.34 {
		return ' ', 0
	}
	return pick(leafGlyphs, hash2(x, y, seed+61)), mix(pal[0], pal[1], clamp01((d-0.34)/0.5)*light)
}

func leafPal(x, y int) [2]rgb {
	if l, ok := newBiomeLeaf(float64(x), float64(y)); ok {
		return l
	}
	switch px, py := float64(x), float64(y); {
	case tideDist(px, py) < 1.15:
		return leafCoral
	case jungleDist(px, py) < 1.1:
		return leafJungle
	case snowDist(px, py) < 1.1:
		return leafSnow
	case swampDist(px, py) < 1.1:
		return leafSwamp
	}
	return leafGreen
}

func (wd *World) put(x, y int, ch byte, c rgb, solid, over bool) {
	if wd.inBounds(x, y) {
		i := y*wd.w + x
		wd.glyph[i], wd.color[i], wd.solid[i], wd.over[i], wd.water[i], wd.marsh[i] = ch, c, solid, over, false, 0
		if wd.glow != nil {
			wd.glow[i] = false
		}
	}
}

// sea transforma a célula em água: não se anda, e o cliente anima as ondas.
func (wd *World) sea(x, y int, seed uint32) {
	if wd.inBounds(x, y) {
		i := y*wd.w + x
		wd.glyph[i], wd.color[i] = '~', mix(0x1e3a7a, 0x3a6ab0, fbm(float64(x)/25, float64(y)/25, seed+201))
		wd.solid[i], wd.over[i], wd.water[i], wd.marsh[i] = true, false, true, 0
	}
}

// generate preenche caractere, cor e colisão de cada célula.
func (wd *World) generate(seed uint32) {
	n := wd.w * wd.h
	wd.seed = seed
	wd.glyph, wd.color = make([]byte, n), make([]rgb, n)
	wd.solid, wd.over, wd.water = make([]bool, n), make([]bool, n), make([]bool, n)
	wd.marsh = make([]byte, n)
	wd.glow = make([]bool, n)
	wd.buildElev(seed)

	// Cada célula só escreve nela mesma, então as linhas são geradas em paralelo.
	parallelRows(wd.h, func(y int) {
		for x := 0; x < wd.w; x++ {
			wd.genCell(x, y, seed)
		}
	})

	// Árvores soltas no campo aberto (e corais em Talássia), fora das clareiras.
	for gy := 24; gy < overH-24; gy += 26 {
		for gx := 24; gx < wd.w-24; gx += 26 {
			h := hash2(gx, gy, seed+70)
			x, y := gx+int(h%19)-9, gy+int(h>>8%19)-9
			px, py := float64(x), float64(y)
			f := forestAt(x, y, seed)
			coral := tideDist(px, py) < 1 && dist(px, py, tideX, tideY) > 40 && pathDist(px, py) > 14
			if (f > 0.3 && f < 0.74 || coral) && h>>16%100 < 55 && wd.fits(x, y) && castleDist(px, py) > 1.1 && !nearLandmark(px, py, 14) {
				r := 8 + int(h>>24%6)
				on, off := wd.stamp(x-r-3, y-2*r-r/2-1, x+r+3, y+1, func() { wd.tree(x, y, r, seed) })
				wd.addProp(propTree, x, y, on, off)
			}
			// Cactos no deserto, longe da pirâmide e dos caminhos.
			if desertDist(px, py) < 0.95 && dist(px, py, desertX, desertY) > 60 && pathDist(px, py) > 12 && h>>16%100 < 40 && wd.fits(x, y) {
				wd.cactus(x, y, seed)
			}
		}
	}

	wd.chapel(chapelX, chapelY, seed)

	// Fileiras de lápides dos dois lados do caminho; metade delas é cova ativa.
	for row := 0; row < 4; row++ {
		for _, off := range []int{-44, -30, -16, 16, 30, 44} {
			h := hash2(row, off, seed+90)
			if h%6 == 0 {
				continue
			}
			x, y := chapelX+off+int(h>>4%5)-2, chapelY+30+row*24+int(h>>8%5)-2
			if graveDist(float64(x), float64(y)) > 0.9 {
				continue
			}
			wd.tomb(x, y, h>>12%3 == 0, seed)
			if h>>16%3 == 0 {
				wd.spawns = append(wd.spawns, spawnPt{x, y + 5, int(h >> 20 % 4 / 3)})
			}
		}
	}

	for _, hs := range houses {
		wd.house(villageX+hs[0], villageY+hs[1], hs[2], hs[3], seed)
	}

	wd.lights = []light{{fireX, fireY, 70, 0}}
	wd.citadel(seed)
	wd.bastion(seed)
	wd.castle(seed)
	wd.buildLandmarks(seed)
	wd.outlands(seed)
	wd.scatter(mobAsura, citadelX, citadelY, 150, 84, 9, seed+300) // dentro da Selva de Marapá
	wd.scatter(mobTriton, tideX, tideY, 196, 56, 9, seed+301)
	wd.scatter(mobSentinel, radiantX, radiantY, 110, 44, 6, seed+302)
	wd.spawns = append(wd.spawns,
		spawnPt{citadelX, citadelY, mobQueen}, spawnPt{tideX, tideY, mobTyrant}, spawnPt{radiantX, radiantY + 6, mobArchon})

	wd.crypt(seed)
	wd.portals = []portal{
		{chapelX, chapelY + 2, cryptInX, cryptInY, true, "Cripta", false},
		{cryptOutX, cryptOutY, chapelX, chapelY + 14, false, "Saída", false},
	}
	wd.genDeep(seed)
	wd.buildPonds(seed) // lagoas novas, antes das rochas e bichos (pesca.go)
	wd.landscape(seed)
	wd.settleProps()
	wd.camps()
	wd.buildNewBiomes(seed) // inimigos dos biomas novos (biomas_novos.go)
	wd.villageDressing(seed)
	wd.buildMerchantPosts(seed) // cabana ou barraca para os mercadores de fora da vila (lojas.go)
	wd.houseMarks(seed)         // placas das casas à venda (casa.go)
	wd.buildSeals(seed)
	wd.buildIntro(seed)
	wd.buildDojo(seed)
	wd.buildDivineTree(seed)
	wd.buildRifts(seed)
	wd.buildGates(seed)
	wd.signposts(seed)
	wd.buildTravel()
}

// landscape monta o que veio depois: falésias, cais, rochas, ossadas,
// cavernas com as bocas e o buraco, e os bichos e ervas de cada bioma.
func (wd *World) landscape(seed uint32) {
	wd.cliffs(seed)
	for _, d := range docks {
		wd.dock(d[0], d[1], seed)
	}

	// Rochas quebráveis e ossadas espalhadas, com a cor de cada bioma.
	for gy := 30; gy < overH-30; gy += 34 {
		for gx := 30; gx < wd.w-30; gx += 34 {
			h := hash2(gx, gy, seed+520)
			x, y := gx+int(h%21)-10, gy+int(h>>8%21)-10
			px, py := float64(x), float64(y)
			if pathDist(px, py) < 12 || plazaDist(px, py) < 1.3 || graveDist(px, py) < 1.1 || nearFeature(px, py, 30) ||
				!wd.fits(x, y) || !wd.fits(x, y-4) {
				continue
			}
			desert, snow := desertDist(px, py) < 1, snowDist(px, py) < 1
			switch roll := h >> 16 % 100; {
			case roll < 22:
				tint := rgb(0x8a8478)
				if desert {
					tint = 0xb89868
				} else if snow {
					tint = 0xc8d0dc
				} else if wd.elevOf(x, y) > 0.6 {
					tint = 0x9a9080
				}
				r := 3 + int(h>>24%3)
				on, off := wd.stamp(x-r-3, y-r-1, x+r+3, y+1, func() { wd.boulder(x, y, r, tint, seed) })
				wd.addProp(propRock, x, y, on, off)
			case desert && roll < 40 || swampDist(px, py) < 1 && roll < 34 || roll < 26:
				wd.bones(x, y, roll%2 == 0, seed)
			}
		}
	}

	for _, m := range caveMouths {
		wd.mouth(m.X, m.Y, seed)
		wd.portals = append(wd.portals,
			portal{m.X, m.Y - 1, m.TX, m.TY, false, "Caverna", false},
			portal{m.TX - 26, m.TY - 4, m.X, m.Y + 8, false, "Saída", false})
	}
	wd.pit(holeX, holeY, seed)
	wd.portals = append(wd.portals,
		portal{holeX, holeY, grottoX, grotY, false, "", true},
		portal{ropeX, ropeY, holeX + 14, holeY + 10, false, "Corda", false})

	wd.caves(seed)
	// A sala secreta fica atrás de uma parede rachada no túnel que sai da galeria leste.
	a, b := caveRooms[4], caveRooms[5]
	mx, my := (a[0]+b[0])/2, (a[1]+b[1])/2
	on, off := wd.stamp(int(mx)-10, int(my)-10, int(mx)+10, int(my)+10, func() {
		for y := int(my) - 10; y <= int(my)+10; y++ {
			for x := int(mx) - 10; x <= int(mx)+10; x++ {
				if !wd.Walkable(x, y) || dist(float64(x), float64(y), mx, my) > 9 {
					continue
				}
				ch := pick(stoneGlyphs, hash2(x, y, seed+530))
				if hash2(x, y, seed+531)%4 == 0 {
					ch = '%'
				}
				wd.put(x, y, ch, dim(0x9a8470, 0.6+0.3*rnd01(x, y, seed+532)), true, false)
			}
		}
	})
	wd.addProp(propCrack, int(mx), int(my), on, off)

	// Inimigos das cavernas.
	wd.scatter(mobRat, 760, overH+130, 46, 0, 4, seed+330)
	wd.scatter(mobRat, 890, overH+90, 38, 0, 3, seed+331)
	wd.scatter(mobBat, 1030, overH+150, 55, 0, 6, seed+332)
	wd.scatter(mobBat, 640, overH+90, 30, 0, 2, seed+333)

	// Montarias selvagens, cada uma no seu bioma.
	wd.herd(steedHorse, villageX+220, villageY+90, 70, 3, seed+340)
	wd.herd(steedBoar, graveX-170, graveY+70, 60, 3, seed+341)
	wd.herd(steedOstrich, desertX-100, desertY+130, 80, 3, seed+342)
	wd.herd(steedRaptor, jungleX-100, jungleY+170, 80, 3, seed+343)

	// Ervas que curam o mal de cada bioma.
	wd.plant(herbAntidote, swampX, swampY, 170, 12, seed+350)
	wd.plant(herbEmber, snowX, snowY, 175, 12, seed+351)
	wd.plant(herbAloe, desertX, desertY+40, 180, 12, seed+352)
}

// genCell gera o chão, a água ou a mata de uma célula do continente.
func (wd *World) genCell(x, y int, seed uint32) {
	px, py := float64(x), float64(y)
	wd.glyph[y*wd.w+x] = ' '
	if y >= overH {
		wd.solid[y*wd.w+x] = true
		return
	}
	if landAt(px, py, seed) < 0 {
		wd.sea(x, y, seed)
		return
	}
	if freshWater(px, py, seed) {
		if pathDist(px, py) < 7 {
			wd.bridge(x, y, seed)
		} else {
			wd.sea(x, y, seed)
			wd.color[y*wd.w+x] = mix(wd.color[y*wd.w+x], 0x2a5a8a, 0.4)
		}
		return
	}
	// Poças de maré em Talássia, longe do centro e dos caminhos.
	if tideDist(px, py) < 0.95 && dist(px, py, tideX, tideY) > 34 && pathDist(px, py) > 12 &&
		fbm(px/14, py/14, seed+210) > 0.7 {
		wd.sea(x, y, seed)
		return
	}
	// Charcos do pântano: água parada e esverdeada.
	if pd := poisonDist(px, py); pathDist(px, py) > 12 && (swampDist(px, py) < 0.95 && fbm(px/16, py/16, seed+230) > 0.6 ||
		pd < 1 && fbm(px/11, py/11, seed+236) > 0.42-0.2*(1-pd)) {
		wd.bog(x, y, pd < 1, seed)
		return
	}
	if wd.newBiomeWater(x, y, px, py, seed) { // poças do brejo, canais do mangue (biomas_novos.go)
		return
	}
	wd.ground(x, y, seed)
	f := forestMod(px, py, forestAt(x, y, seed))
	// Trilhas de bicho cortam a mata fechada: sempre há por onde passar.
	trail := math.Abs(valueNoise(px/38, py/38, seed+77)-0.5) < 0.04 || math.Abs(valueNoise(px/29+7, py/29, seed+78)-0.5) < 0.03
	switch {
	case f > 0.74 && !trail:
		// Miolo da mata: fechado.
		ch, c := leaf(x, y, seed, 0.5+0.5*fbm(px/30, py/30, seed+62), leafPal(x, y))
		if ch != ' ' || f > 0.78 {
			wd.put(x, y, ch, c, true, false)
		}
	case f > 0.56:
		// Borda da mata e trilhas: vegetação rasteira, moitas de capim e
		// samambaia por onde se anda; só uma touceira aqui e ali bloqueia.
		k := clamp01((f - 0.56) / 0.18)
		h := hash2(x, y, seed+63)
		pal := leafPal(x, y)
		switch {
		case h%100 < uint32(2+3*k) && !trail:
			wd.put(x, y, pick("&%#", h>>8), dim(pal[0], 1.3+0.4*rnd01(x, y, seed+64)), true, false)
		case rnd01(x, y, seed+65) < 0.35+0.4*k:
			wd.put(x, y, pick(",;'\"`v", h>>12), mix(pal[0], pal[1], 0.25+0.45*rnd01(x, y, seed+66)), false, false)
		}
	}
}

// outlands monta o que há de construído nos biomas de fora: a pirâmide do
// deserto, o acampamento orc com paliçada e fogueira, e os blocos de gelo.
func (wd *World) outlands(seed uint32) {
	// Pirâmide em degraus, com a entrada escura na base.
	for i, w := range []int{96, 72, 48, 24} {
		wd.tier(desertX-w/2, desertY-22-i*13, w, 14, 0xd0a858, 0xf0d080, seed)
	}
	for y := desertY - 18; y <= desertY-9; y++ {
		for x := desertX - 4; x <= desertX+4; x++ {
			wd.put(x, y, ':', 0x2a2014, true, false)
		}
	}
	wd.lights = append(wd.lights, light{desertX - 16, desertY - 2, 40, 1}, light{desertX + 16, desertY - 2, 40, 1})

	// Acampamento: tendas em volta da fogueira e estacas fechando o círculo.
	for _, t := range [][2]int{{-40, -34}, {16, -40}, {-46, 8}, {24, 6}} {
		wd.tier(orcX+t[0], orcY+t[1], 24, 13, 0x7a5a3a, 0x9a3a2a, seed)
	}
	for k := 0.0; k < 14; k++ {
		a := k * math.Pi / 7
		if math.Abs(math.Cos(a)) < 0.95 || math.Sin(a) > 0 { // abre uma passagem a oeste
			wd.pillar(orcX+int(58*math.Cos(a)), orcY+int(56*math.Sin(a)), 7, 0x8a6a48, seed)
		}
	}
	wd.lights = append(wd.lights, light{orcX, orcY - 6, 60, 0})

	// Blocos de gelo que brilham em volta do trono do gigante.
	for k := 0.0; k < 6; k++ {
		a := k*math.Pi/3 + 0.5
		cx, cy := snowX+int(34*math.Cos(a)), snowY+int(26*math.Sin(a))
		for dy := -10; dy <= 0; dy++ {
			half := 3 - int(math.Abs(float64(dy+5))/2.6)
			for dx := -half; dx <= half; dx++ {
				wd.put(cx+dx, cy+dy, "@#%&"[hash2(dx, dy, seed+241)%4], mix(0xffffff, 0x7ac0ff, math.Abs(float64(dx))/3.5), dy >= -1, false)
			}
		}
		wd.lights = append(wd.lights, light{cx, cy - 5, 36, 2})
	}

	// Cada bioma com os seus inimigos; ratos e lobos rondam o entorno da vila.
	wd.scatter(mobWolf, villageX+520, villageY+240, 110, 20, 5, seed+311)
	wd.scatter(mobOrc, orcX, orcY, 52, 14, 4, seed+312)
	wd.scatter(mobOrcWarrior, orcX, orcY, 52, 14, 4, seed+313)
	wd.scatter(mobScarab, desertX, desertY+30, 196, 70, 6, seed+314)
	wd.scatter(mobMummy, desertX, desertY+30, 154, 56, 4, seed+315)
	wd.scatter(mobIceWolf, snowX, snowY, 190, 70, 5, seed+316)
	wd.scatter(mobYeti, snowX, snowY, 154, 63, 4, seed+317)
	wd.scatter(mobSpider, swampX, swampY, 182, 42, 4, seed+318)
	wd.scatter(mobTroll, swampX, swampY, 168, 42, 4, seed+319)
	wd.spawns = append(wd.spawns,
		spawnPt{orcX, orcY + 14, mobOrcChief}, spawnPt{desertX, desertY + 14, mobPharaoh}, spawnPt{snowX, snowY, mobFrostGiant})
}

// scatter espalha n inimigos em volta de (cx, cy), entre os raios rmin e r,
// só em chão livre.
func (wd *World) scatter(kind, cx, cy, r, rmin, n int, seed uint32) {
	for i, placed := 0, 0; i < n*40 && placed < n; i++ {
		h := hash2(i, kind, seed)
		x, y := cx+int(h%uint32(2*r))-r, cy+int(h>>12%uint32(2*r))-r
		if d := dist(float64(x), float64(y), float64(cx), float64(cy)); d > float64(r) || d < float64(rmin) || !wd.fits(x, y) {
			continue
		}
		wd.spawns = append(wd.spawns, spawnPt{x, y, kind})
		placed++
	}
}

// ground pinta o chão: escuro, com caracteres ralos, e a cor de cada região
// (dourado na praça, areia na costa, mármore no bastião, verde vivo na selva).
func (wd *World) ground(x, y int, seed uint32) {
	px, py := float64(x), float64(y)
	i := y*wd.w + x
	col := mix(0x2c2c48, 0x4c4a72, fbm(px/40, py/40, seed))
	dens := 0.16 + 0.22*fbm(px/18, py/18, seed+3)
	// Pasto: manchas largas de capim denso (o cliente desenha as lâminas).
	if g := fbm(px/60, py/60, seed+11); g > 0.4 {
		k := math.Min(0.9, (g-0.4)*3.2)
		col, dens = mix(col, mix(0x2f6e52, 0x4a8a4a, fbm(px/9, py/9, seed+12)), k), dens+0.4*k
	}
	if p := plazaDist(px, py); p < 1.2 {
		k := math.Min(1, (1.2-p)*2)
		col, dens = mix(col, 0x7a6436, k*0.8), dens+0.15*k
	}
	if d := dist(px, py, hubX, hubY); d < 85 {
		col = mix(col, 0x5a2a26, math.Min(1, (85-d)/30)*0.7)
	}
	if jd := jungleDist(px, py); jd < 1.1 {
		k := clamp01((1.1 - jd) * 4)
		col, dens = mix(col, 0x2a7a48, k*0.7), dens+0.12*k
		// Flores da selva.
		if h := hash2(x, y, seed+220); k > 0.5 && h%90 == 0 {
			wd.glyph[i], wd.color[i] = '*', []rgb{0xff6ab0, 0xffd24a, 0xff8a3a}[h>>8%3]
			return
		}
	}
	if td := tideDist(px, py); td < 1.1 {
		k := clamp01((1.1 - td) * 4)
		col, dens = mix(col, 0xc2a878, k*0.85), dens+0.1*k
	}
	if dd := desertDist(px, py); dd < 1.1 {
		// Areia clara, com as dunas marcadas em faixas.
		k := clamp01((1.1 - dd) * 4)
		wave := 0.85 + 0.15*math.Sin(px/9+py/5+4*fbm(px/40, py/40, seed+231))
		col, dens = mix(col, dim(0xd8b860, wave), k*0.9), dens+0.18*k
	}
	if sd := snowDist(px, py); sd < 1.1 {
		k := clamp01((1.1 - sd) * 4)
		col, dens = mix(col, 0xdfe8f5, k*0.9), dens+0.28*k
	}
	if sd := swampDist(px, py); sd < 1.1 {
		k := clamp01((1.1 - sd) * 4)
		col, dens = mix(col, 0x3a5a34, k*0.8), dens+0.1*k
	}
	if c, d, done := wd.newBiomeGround(x, y, px, py, col, dens, seed); done {
		return
	} else {
		col, dens = c, d
	}
	if l := landAt(px, py, seed); l < 0.07 {
		col, dens = mix(col, 0xc2a878, clamp01((0.07-l)*30)), dens+0.1
	}
	if rd := radiantDist(px, py); rd < 1.05 {
		// Mármore claro, com as juntas das lajes sempre desenhadas.
		k := clamp01((1.05 - rd) * 6)
		col, dens = mix(col, mix(0xb0a070, 0xe8e0c0, fbm(px/16, py/16, seed+221)), k), dens+0.3*k
		if k > 0.8 && (x%14 == 0 || y%10 == 0) {
			wd.glyph[i], wd.color[i] = ':', dim(col, 0.5)
			return
		}
	}
	if md := mountDist(px, py); md < 1.2 {
		k := clamp01((1.2 - md) * 3)
		col, dens = mix(col, 0x3a3436, 0.8*k), dens+0.2*k
		if md < 0.4 && fbm(px/8, py/8, seed+600) > 0.7 {
			wd.glyph[i], wd.color[i] = '~', mix(0xff4818, 0xffa040, rnd01(x, y, seed+601))
			wd.glow[i] = true
			return
		}
	}
	if y < overH {
		light, rock := wd.relief(x, y)
		col, dens = dim(mix(col, 0x6a6052, rock*0.7), light), dens+0.15*rock
	}
	// Trilha sutil: só chão um pouco mais gasto, que some em alguns trechos
	// (tomados pelo mato). O caminho é para ser descoberto, não seguido.
	if pd := pathDist(px, py); pd < 5+3*valueNoise(px/9, py/9, seed+21) {
		wear := valueNoise(px/16, py/16, seed+22)
		k := clamp01((0.75-wear)*2.5) * clamp01(1.2-pd/7)
		col, dens = mix(col, 0x5a5872, 0.4*k), dens+0.1*k
	}
	if rnd01(x, y, seed+13) < dens {
		wd.glyph[i] = pick(groundGlyphs, hash2(x, y, seed)>>8)
		wd.color[i] = dim(col, 0.6+0.4*rnd01(x, y, seed+14))
	}
}

// dragonClass diz que parte do dragão cobre o ponto (px, py), em unidades de
// duas células relativas ao centro do corpo; 0 é fora. t anima asas e cauda.
const (
	dgNone = iota
	dgBody
	dgSpine
	dgHead
	dgEye
	dgWing
	dgBone
)

func dragonClass(px, py, t float64) int {
	ax := math.Abs(px)
	if math.Abs(ax-1) < 0.6 && math.Abs(py+10.5) < 0.6 {
		return dgEye
	}
	if ax < 0.6 && py > -7 && py < 6 {
		return dgSpine
	}
	if math.Hypot(px/2.6, (py+10)/3) < 1 {
		return dgHead
	}
	if ax < 1.6 && py > -8 && py < -4 {
		return dgBody
	}
	if math.Hypot(px/4.2, py/7) < 1 {
		return dgBody
	}
	if py >= 6 && py <= 19 {
		k := (py - 6) / 13
		if math.Abs(px-3.5*math.Sin(k*4+t*2)*k) < 2.2*(1-k)+0.5 {
			return dgBody
		}
	}
	flap := math.Sin(t * 3)
	span := 18 + 2.5*flap
	if ax >= 3 && ax <= 3+span {
		s := (ax - 3) / span
		top := -6 - 5*s + 2*flap*s
		bottom := 6 - 15*s - 1.2*math.Abs(math.Sin(s*math.Pi*3)) + 2*flap*s
		if py > top && py < bottom {
			if py < top+1 || math.Abs(math.Mod(s*3, 1)-0.5) > 0.43 {
				return dgBone
			}
			return dgWing
		}
	}
	return dgNone
}

// camps põe fogueiras e mercadores em chão livre perto de onde foram pensados.
// A fogueira da vila e a do acampamento já existem como luz.
func (wd *World) camps() {
	wd.bonfires = append([]bonfire(nil), bonfiresBase...)
	for i := range wd.bonfires {
		b := &wd.bonfires[i]
		if x, y, ok := wd.freeNear(b.X, b.Y, 40); ok {
			b.X, b.Y = x, y
		}
		if !wd.lightNear(b.X, b.Y, 12) { // a da vila e a do acampamento já são luz
			wd.lights = append(wd.lights, light{b.X, b.Y, 50, 0})
		}
	}
	wd.shops = append([]merchant(nil), merchantsBase...)
	for i := range wd.shops {
		if x, y, ok := wd.freeNear(wd.shops[i].X, wd.shops[i].Y, 30); ok {
			wd.shops[i].X, wd.shops[i].Y = x, y
		}
	}
}

// lightNear diz se já há uma fogueira (luz de chama) perto do ponto.
func (wd *World) lightNear(x, y int, r float64) bool {
	for _, l := range wd.lights {
		if l.Kind == 0 && dist(float64(l.X), float64(l.Y), float64(x), float64(y)) <= r {
			return true
		}
	}
	return false
}
