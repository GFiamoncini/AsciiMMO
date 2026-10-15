package main

import (
	"math"
	"time"
)

// Profundezas: um segundo continente embaixo do primeiro, na mesma escala.
// Fica numa faixa própria do mapa, abaixo da cripta e das cavernas (deepY0),
// e é alinhado 1:1 com a superfície: o ponto (x, y) lá em cima fica sobre
// (x, deepY0+y) aqui embaixo. Só há galerias sob a terra firme; sob o mar é
// rocha. O que está em cima decide o bioma de baixo (deepBiome): fogo e lava
// sob o Pico de Tiamath, arenito sob o deserto, gelo sob os picos, fungos sob
// o pântano e a selva, pedra no resto. Fendas na superfície (rifts) descem;
// escadas na pedra sobem. O cliente baixa esta faixa só quando alguém desce.

const (
	cryptH = 240            // altura da faixa da cripta e das cavernas
	deepY0 = overH + cryptH // primeira linha das Profundezas
	deepH  = overH          // mesma altura do continente

	lavaMarsh = 3 // wd.marsh: lava (queima quem pisa; Botas de Obsidiana protegem)
)

// Biomas das Profundezas.
const (
	dbStone = iota
	dbFire
	dbSand
	dbIce
	dbFungus
	dbCrystal // Cavernas de Cristal (cristais.go)
)

// rifts são as fendas que descem, na superfície (x, y). A câmara de chegada
// fica logo abaixo, em (x, deepY0+y).
var rifts = [][2]int{
	{villageX + 140, villageY + 70},
	{graveX - 90, graveY + 50},
	{hubX + 70, hubY - 60},
	{desertX - 90, desertY + 60},
	{snowX + 60, snowY + 80},
	{swampX + 40, swampY + 80},
	{jungleX - 110, jungleY + 90},
	{lairX + 150, lairY + 130},
	{treeX + treeRiftOffX, treeY + treeRiftOffY}, // as raízes da Árvore Divina
}

// Coração das Profundezas: a câmara de Ignar, sob o Pico de Tiamath.
var heartX, heartY = lairX, deepY0 + lairY

func inDeep(y int) bool { return y >= deepY0 }

func deepBiome(x, yy int) int {
	px, py := float64(x), float64(yy)
	switch {
	case crystalDist(px, py) < 1:
		return dbCrystal
	case mountDist(px, py) < 0.95:
		return dbFire
	case desertDist(px, py) < 1.05:
		return dbSand
	case snowDist(px, py) < 1.05:
		return dbIce
	case swampDist(px, py) < 1.1 || jungleDist(px, py) < 1:
		return dbFungus
	}
	return dbStone
}

// deepFloor diz se a célula (x, yy da superfície) é galeria: duas redes de
// túneis (as curvas de nível de dois ruídos) e salões onde um terceiro ruído
// sobe. Sob o mar, rocha.
func deepFloor(x, yy int, seed uint32) bool {
	px, py := float64(x), float64(yy)
	if landAt(px, py, seed) < 0.06 {
		return false
	}
	if math.Abs(fbm(px/90, py/90, seed+900)-0.5) < 0.02 {
		return true
	}
	if math.Abs(fbm(px/56, py/56, seed+902)-0.5) < 0.014 {
		return true
	}
	if crystalDist(px, py) < 1 { // as Cavernas de Cristal são mais abertas
		return fbm(px/48, py/48, seed+901) > 0.54
	}
	return fbm(px/48, py/48, seed+901) > 0.64
}

// genDeep escava as Profundezas: a grade de galerias, os túneis que ligam
// as fendas entre si e ao coração, as câmaras de chegada, e então pinta
// chão, paredes, lava, poças e brilhos de cada bioma.
func (wd *World) genDeep(seed uint32) {
	grid := make([]bool, wd.w*deepH)
	parallelRows(deepH, func(yy int) {
		for x := 0; x < wd.w; x++ {
			grid[yy*wd.w+x] = deepFloor(x, yy, seed)
		}
	})
	widenGalleries(grid, wd.w, deepH)
	carve := func(cx, cy, r float64) {
		for y := int(cy - r); y <= int(cy+r); y++ {
			for x := int(cx - r); x <= int(cx+r); x++ {
				if x > 2 && y > 2 && x < wd.w-3 && y < deepH-3 && math.Hypot(float64(x)-cx, float64(y)-cy) <= r {
					grid[y*wd.w+x] = true
				}
			}
		}
	}
	tunnel := func(ax, ay, bx, by float64, k uint32) {
		n := int(math.Hypot(bx-ax, by-ay) / 2)
		for i := 0; i <= n; i++ {
			t := float64(i) / float64(max(1, n))
			// curva suave para os lados, para não ser uma régua
			off := math.Sin(t*math.Pi) * (fbm(t*3, float64(k), seed+905) - 0.5) * 120
			nx, ny := -(by - ay), bx-ax
			l := math.Max(1, math.Hypot(nx, ny))
			carve(ax+(bx-ax)*t+nx/l*off, ay+(by-ay)*t+ny/l*off, 3)
		}
	}
	// As fendas, em volta do centro, ligadas em anel; e todas ao coração.
	for i, r := range rifts {
		carve(float64(r[0]), float64(r[1]), 14)
		nx := rifts[(i+1)%len(rifts)]
		tunnel(float64(r[0]), float64(r[1]), float64(nx[0]), float64(nx[1]), uint32(i))
	}
	// Um salão no centro de cada região de baixo, ligado à fenda mais perto.
	for _, rg := range regions {
		if !inDeep(rg.Y) || rg.Name == "Abismo de Dorrim" { // ao Abismo só se chega pela ponte
			continue
		}
		cx, cy := float64(rg.X), float64(rg.Y-deepY0)
		carve(cx, cy, 20)
		best, bd := rifts[0], math.MaxFloat64
		for _, r := range rifts {
			if d := math.Hypot(float64(r[0])-cx, float64(r[1])-cy); d < bd {
				best, bd = r, d
			}
		}
		tunnel(cx, cy, float64(best[0]), float64(best[1]), uint32(rg.X))
	}
	carveDeep2(carve, tunnel) // portas de Dorrim e corredores das câmaras secretas (dorrim.go)
	carve(float64(heartX), float64(heartY-deepY0), 46)
	last := rifts[len(rifts)-1]
	tunnel(float64(last[0]), float64(last[1]), float64(heartX), float64(heartY-deepY0), 99)

	floor := func(x, yy int) bool {
		return x >= 0 && x < wd.w && yy >= 0 && yy < deepH && grid[yy*wd.w+x]
	}
	parallelRows(deepH, func(yy int) {
		y := deepY0 + yy
		for x := 0; x < wd.w; x++ {
			i := y*wd.w + x
			b := deepBiome(x, yy)
			if floor(x, yy) {
				wd.deepFloorCell(x, y, i, b, seed)
				continue
			}
			wall := false
			for dy := -2; dy <= 6 && !wall; dy++ {
				wall = floor(x-2, yy+dy) || floor(x, yy+dy) || floor(x+2, yy+dy)
			}
			wd.glyph[i], wd.solid[i] = ' ', true
			if wall {
				wd.glyph[i] = pick(stoneGlyphs, hash2(x, y, seed+906))
				wd.color[i] = dim(deepWall[b], 0.45+0.35*rnd01(x, y, seed+907))
				if b == dbIce && rnd01(x, y, seed+908) < 0.03 || b == dbStone && rnd01(x, y, seed+908) < 0.006 || b == dbCrystal && rnd01(x, y, seed+908) < 0.09 {
					wd.glyph[i], wd.color[i], wd.glow[i] = '*', deepGlint[b], true // cristal na parede
				}
			}
		}
	})
	wd.buildDorrim(seed)
	wd.buildAbyss(seed) // a ponte e Vulkhar (vulkhar.go)
	wd.buildCrystalVault(seed)
	wd.deepSecrets(seed)
	wd.scatterDeep2(seed)
	// Câmara do coração: anel de lava em volta de uma ilha de basalto.
	for y := heartY - 46; y <= heartY+46; y++ {
		for x := heartX - 46; x <= heartX+46; x++ {
			d := math.Hypot(float64(x-heartX), float64(y-heartY))
			if d > 24 && d < 32+6*(fbm(float64(x)/9, float64(y)/9, seed+909)-0.5) {
				wd.lavaCell(x, y, seed)
			}
		}
	}
	// Uma ponte de basalto atravessa o anel, do lado de onde vem o túnel.
	for k := 20; k <= 40; k++ {
		for w := -2; w <= 2; w++ {
			i := (heartY+w)*wd.w + heartX + k
			wd.marsh[i], wd.glow[i], wd.solid[i] = 0, false, false
			wd.glyph[i], wd.color[i] = '=', dim(0x5a4038, 0.8)
		}
	}

	// Os de baixo: rastejantes por toda parte, golens no fogo, cascas de
	// fungo nos jardins, morcegos; Ignar no coração.
	for i, r := range rifts {
		cx, cy := r[0], deepY0+r[1]
		wd.scatter(mobCrawler, cx, cy, 160, 60, 6, seed+910+uint32(i))
		wd.scatter(mobBat, cx, cy, 100, 50, 2, seed+930+uint32(i))
		switch deepBiome(r[0], r[1]) {
		case dbFire:
			wd.scatter(mobMagmaGolem, cx, cy, 160, 30, 6, seed+950+uint32(i))
		case dbFungus:
			wd.scatter(mobFungusHusk, cx, cy, 150, 30, 6, seed+970+uint32(i))
		}
	}
	wd.scatter(mobMagmaGolem, heartX, heartY, 140, 50, 6, seed+990)
	wd.spawns = append(wd.spawns, spawnPt{heartX - 6, heartY, mobIgnar})
	wd.lights = append(wd.lights, light{heartX, heartY, 90, 0})
}

var deepWall = [...]rgb{dbStone: 0x6a6070, dbFire: 0x5a3028, dbSand: 0xa08050, dbIce: 0x9ab8d8, dbFungus: 0x405a44, dbCrystal: 0x8070c0}
var deepGround = [...][2]rgb{dbStone: {0x24222c, 0x383444}, dbFire: {0x2c1a16, 0x40261c}, dbSand: {0x3a2e1c, 0x54422a},
	dbIce: {0x2a3644, 0x40566a}, dbFungus: {0x1c2a1e, 0x2c4030}, dbCrystal: {0x1c1a32, 0x2e2a54}}
var deepGlint = [...]rgb{dbStone: 0xb890ff, dbFire: 0xff8030, dbSand: 0xffd060, dbIce: 0x90f0ff, dbFungus: 0x9aff8a, dbCrystal: 0xd0b0ff}

// widenGalleries alarga as galerias naturais: as curvas de nível saem com 1 a
// 3 células de largura, e o personagem ocupa 5 de lado (fits), então ficava
// preso nos trechos finos. Cada célula de chão abre 3 para cada lado e 1 para
// cima e para baixo. Roda antes dos túneis e salões, que já nascem largos.
func widenGalleries(grid []bool, w, h int) {
	horiz := make([]bool, len(grid))
	parallelRows(h, func(y int) {
		row, out := grid[y*w:(y+1)*w], horiz[y*w:(y+1)*w]
		for x := range row {
			for dx := -3; dx <= 3; dx++ {
				if j := x + dx; j >= 0 && j < w && row[j] {
					out[x] = true
					break
				}
			}
		}
	})
	parallelRows(h, func(y int) {
		for x := 0; x < w; x++ {
			if x < 3 || x >= w-3 || y < 3 || y >= h-3 {
				continue // a borda do mundo fica fechada
			}
			grid[y*w+x] = horiz[(y-1)*w+x] || horiz[y*w+x] || horiz[(y+1)*w+x]
		}
	})
}

func (wd *World) deepFloorCell(x, y, i, b int, seed uint32) {
	g := deepGround[b]
	c := dim(mix(g[0], g[1], fbm(float64(x)/18, float64(y)/18, seed+911)), 0.6+0.4*rnd01(x, y, seed+912))
	wd.glyph[i], wd.color[i], wd.solid[i] = ' ', c, false
	if rnd01(x, y, seed+913) < 0.3 {
		wd.glyph[i] = pick(".,:;'`", hash2(x, y, seed+914))
	}
	pool := fbm(float64(x)/22, float64(y)/22, seed+915)
	switch {
	case b == dbFire && pool > 0.66:
		wd.lavaCell(x, y, seed)
	case b == dbFungus && pool > 0.7:
		wd.glyph[i], wd.color[i], wd.marsh[i] = '~', dim(0x60a040, 0.5+0.3*rnd01(x, y, seed+916)), 2
	case b == dbIce && pool > 0.68:
		wd.glyph[i], wd.color[i] = '-', dim(0xb0d8f0, 0.5) // gelo liso
	case rnd01(x, y, seed+917) < [...]float64{0.0015, 0.002, 0.002, 0.002, 0.012, 0.02}[b]:
		ch := byte('*')
		if b == dbFungus {
			ch = 'T' // cogumelo
		}
		wd.glyph[i], wd.color[i], wd.glow[i] = ch, deepGlint[b], true
	}
}

func (wd *World) lavaCell(x, y int, seed uint32) {
	i := y*wd.w + x
	wd.glyph[i] = pick("~~=~", hash2(x, y, seed+918))
	wd.color[i] = mix(0xc03010, 0xffa030, rnd01(x, y, seed+919))
	wd.solid[i], wd.marsh[i], wd.glow[i] = false, lavaMarsh, true
}

// buildRifts abre as fendas na superfície e põe as escadas embaixo.
func (wd *World) buildRifts(seed uint32) {
	for k, r := range rifts {
		x, y := r[0], r[1]
		wd.clearArea(x-10, y-10, x+10, y+12, seed+920+uint32(k))
		for dy := -3; dy <= 3; dy++ {
			for dx := -5; dx <= 5; dx++ {
				d := math.Hypot(float64(dx)/5, float64(dy)/3)
				switch {
				case d < 0.7:
					wd.put(x+dx, y+dy, ' ', 0x000000, false, false) // a boca escura
				case d < 1.1:
					wd.put(x+dx, y+dy, pick("/\\_", hash2(x+dx, y+dy, seed+921)), dim(0x6a5a4a, 0.8), false, false)
				}
			}
		}
		dyU := deepY0 + y
		for dx := -2; dx <= 2; dx++ { // degraus na pedra
			wd.put(x+dx, dyU, '=', dim(0x9a8a7a, 0.8), false, false)
		}
		wd.lights = append(wd.lights, light{x, dyU - 2, 30, 2})
		wd.portals = append(wd.portals,
			portal{x, y, x, dyU + 7, false, "Fenda · nível 15+", false},
			portal{x, dyU, x, y + 8, false, "Subida", false})
	}
}

// lavaStep: uma vez por segundo, quem está na lava se queima, a não ser que
// calce Botas de Obsidiana; o Manto de Cinzas divide a queimadura por quatro.
func (wd *World) lavaStep(p *Player, now time.Time) {
	if !wd.inBounds(p.X, p.Y) || wd.marsh[p.Y*wd.w+p.X] != lavaMarsh || p.Boots == itemObsidianBoots {
		return
	}
	dmg := 45
	if p.Skin == itemAshMantle {
		dmg /= 4
	}
	wd.afflict(p, stHeat, 3*time.Second, now)
	wd.hurtRaw(p, dmg, "na lava", now)
}
