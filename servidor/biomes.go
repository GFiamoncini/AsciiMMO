package main

import "math"

// Elementos de paisagem que vieram depois dos biomas principais: relevo com
// falésias, rio e lago com ilha, pontes, cais, pântano venenoso, rochas,
// ossadas, cavernas no subsolo e os buracos que caem nelas.

// Rio: nasce no Lago Espelhado, a leste do covil, e desce até o mar do sul.
var riverPts = scalePts([][2]float64{{980, 468}, {966, 560}, {950, 650}, {962, 740}, {942, 830}, {956, 930}, {962, 1040}, {966, 1150}, {968, 1215}})

// riverBox é o retângulo do leito, para pular o cálculo longe dele.
var riverBox = func() (b [4]float64) {
	b = [4]float64{math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}
	for _, p := range riverPts {
		b[0], b[1], b[2], b[3] = math.Min(b[0], p[0]-40), math.Min(b[1], p[1]-40), math.Max(b[2], p[0]+40), math.Max(b[3], p[1]+40)
	}
	return
}()

func scalePts(pts [][2]float64) [][2]float64 {
	for i := range pts {
		pts[i][0], pts[i][1] = pts[i][0]*mapScale, pts[i][1]*mapScale
	}
	return pts
}

const (
	lakeX, lakeY     = 980 * mapScale, 428 * mapScale
	lakeRX, lakeRY   = 50 * mapScale, 28 * mapScale
	islandX, islandY = lakeX + 32, lakeY - 8
	riverHalf        = 5.0 // meia largura do rio, em células

	poisonX, poisonY = swampX + 35*mapScale, swampY + 65*mapScale // Charco Venenoso, no sul do pântano
	poisonR          = 40.0 * mapScale
)

// Cais: o ponto é a ponta do píer, onde se embarca (um no lago, um no rio).
var docks = [][2]int{{lakeX - lakeRX + 12, lakeY + 14}, {int(riverPts[5][0]) - 1, int(riverPts[5][1])}}

// Bocas de caverna no continente e onde cada uma leva no subsolo.
type caveMouth struct{ X, Y, TX, TY int }

var caveMouths = []caveMouth{
	{520 * mapScale, 540 * mapScale, 640, overH + 100},   // a leste da vila → Salão de Entrada
	{1010 * mapScale, 650 * mapScale, 1170, overH + 106}, // margem do pântano → Galeria Leste
}

// Cavernas: salões (centro e raios) ligados por túneis, em coordenadas do mundo.
var caveRooms = [][4]float64{
	{640, overH + 90, 38, 22},   // 0 Salão de Entrada
	{760, overH + 130, 52, 30},  // 1
	{890, overH + 90, 44, 26},   // 2
	{1030, overH + 150, 62, 38}, // 3 ninho dos morcegos
	{1170, overH + 100, 40, 24}, // 4 Galeria Leste
	{1320, overH + 180, 30, 20}, // 5 sala secreta, atrás da parede rachada
	{880, overH + 205, 26, 14},  // 6 gruta do buraco, isolada
}

var caveTunnels = [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}}

// O buraco no campo cai na gruta 6; a corda da gruta devolve ao lado dele.
const (
	holeX, holeY   = 560 * mapScale, 700 * mapScale
	ropeX, ropeY   = 866, overH + 200
	grottoX, grotY = 892, overH + 208
)

// Zonas do subsolo, para o minimapa saber o que mostrar.
var zones = []struct {
	Name       string
	X, Y, W, H int
}{
	{"Cripta", 110, overH + 96, 420, 120},
	{"Cavernas", 590, overH + 50, 790, 190},
	{"Profundezas", 0, deepY0, worldW, deepH},
}

func segDist(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	t := clamp01(((px-ax)*vx + (py-ay)*vy) / (vx*vx + vy*vy))
	return math.Hypot(px-ax-t*vx, py-ay-t*vy)
}

func riverDist(px, py float64, seed uint32) float64 {
	if px < riverBox[0] || px > riverBox[2] || py < riverBox[1] {
		return 1e9 // longe de todo o leito: evita o cálculo no resto do mapa
	}
	best := math.MaxFloat64
	for i := 0; i+1 < len(riverPts); i++ {
		a, b := riverPts[i], riverPts[i+1]
		best = math.Min(best, segDist(px, py, a[0], a[1], b[0], b[1]))
	}
	return best + (valueNoise(px/14, py/14, seed+410)-0.5)*4
}

func lakeDist(px, py float64, seed uint32) float64 {
	if math.Abs(px-lakeX) > 2*lakeRX || math.Abs(py-lakeY) > 2*lakeRY {
		return 9
	}
	return math.Hypot((px-lakeX)/lakeRX, (py-lakeY)/lakeRY) + (valueNoise(px/10, py/10, seed+411)-0.5)*0.15
}

func islandDist(px, py float64) float64 { return math.Hypot((px-islandX)/18, (py-islandY)/11) }

// freshWater diz se a célula é rio ou lago (a ilha fica de fora).
func freshWater(px, py float64, seed uint32) bool {
	if islandDist(px, py) < 1 {
		return false
	}
	return lakeDist(px, py, seed) < 1 || riverDist(px, py, seed) < riverHalf
}

func poisonDist(px, py float64) float64 { return math.Hypot(px-poisonX, py-poisonY) / poisonR }

// elevAt é a altura do terreno, 0..1; acima de ~0.52 começam as terras altas.
func elevAt(px, py float64, seed uint32) float64 {
	return fbm(px/150, py/150, seed+400)*0.75 + valueNoise(px/40, py/40, seed+402)*0.25 + 0.55*clamp01(1.1-mountDist(px, py))
}

// mountDist vale menos de 1 na cordilheira de Tiamath, a parte mais alta do mapa.
func mountDist(px, py float64) float64 { return math.Hypot((px-lairX)/(260*bsF), (py-lairY)/(200*bsF)) }

const cliffSteps = 7.0 // degraus de altura; cada descida para o sul vira falésia

// buildElev calcula a altura de todo o continente uma vez só.
func (wd *World) buildElev(seed uint32) {
	wd.elev = make([]float32, wd.w*overH)
	parallelRows(overH, func(y int) {
		for x := 0; x < wd.w; x++ {
			wd.elev[y*wd.w+x] = float32(elevAt(float64(x), float64(y), seed))
		}
	})
}

func (wd *World) elevOf(x, y int) float64 {
	x, y = max(0, min(wd.w-1, x)), max(0, min(overH-1, y))
	return float64(wd.elev[y*wd.w+x])
}

// relief devolve o fator de luz da encosta (as voltadas para noroeste
// clareiam) e quanto a célula é de terra alta (vira rocha nua).
func (wd *World) relief(x, y int) (light, rock float64) {
	e1, e2 := wd.elevOf(x-3, y-3), wd.elevOf(x+3, y+3)
	light = 1 + math.Max(-0.35, math.Min(0.35, (e2-e1)*14))
	rock = clamp01((wd.elevOf(x, y) - 0.6) * 5)
	return
}

// keepClear diz onde o relevo e as rochas não podem fechar a passagem.
func keepClear(px, py float64, seed uint32) bool {
	return regionAt(int(px), int(py)) >= 0 || pathDist(px, py) < 12 || plazaDist(px, py) < 1.4 ||
		graveDist(px, py) < 1.3 || dist(px, py, hubX, hubY) < 100 || lakeDist(px, py, seed) < 1.25 ||
		riverDist(px, py, seed) < riverHalf+6 || nearFeature(px, py, 45)
}

// nearFeature diz se o ponto está perto de uma boca de caverna, do buraco ou de um cais.
func nearFeature(px, py, r float64) bool {
	for _, m := range caveMouths {
		if dist(px, py, float64(m.X), float64(m.Y)) < r {
			return true
		}
	}
	for _, d := range docks {
		if dist(px, py, float64(d[0]), float64(d[1])) < r {
			return true
		}
	}
	return dist(px, py, holeX, holeY) < r || castleDist(px, py) < 1+r/100 || nearLandmark(px, py, r)
}

// cliffs desenha as falésias: onde o terreno desce um degrau para o sul, a
// face de pedra ocupa as células de baixo e bloqueia a passagem.
func (wd *World) cliffs(seed uint32) {
	for y := 1; y < overH-8; y++ {
		for x := 2; x < wd.w-2; x++ {
			e := wd.elevOf(x, y)
			if e < 0.52 || math.Floor(wd.elevOf(x, y+1)*cliffSteps) >= math.Floor(e*cliffSteps) {
				continue
			}
			px, py := float64(x), float64(y)
			if keepClear(px, py, seed) || wd.solid[y*wd.w+x] || landAt(px, py, seed) < 0.1 {
				continue
			}
			h := 4 + int(hash2(x, 0, seed+420)%3)
			for k := 1; k <= h; k++ {
				i := (y+k)*wd.w + x
				if wd.water[i] {
					break
				}
				l := (1.05 - 0.12*float64(k)) * (0.8 + 0.3*rnd01(x, y+k, seed+421))
				ch := pick(stoneGlyphs, hash2(x, y+k, seed+422))
				if (y+k)%3 == 0 {
					ch, l = '=', l*0.7
				}
				wd.put(x, y+k, ch, dim(mix(0x6a5e52, 0x8a8070, rnd01(x>>2, y, seed+423)), l), true, false)
			}
		}
	}
}

// bog transforma a célula em charco: dá para andar, mas devagar; o venenoso
// ainda envenena quem pisa.
func (wd *World) bog(x, y int, poison bool, seed uint32) {
	i := y*wd.w + x
	px, py := float64(x), float64(y)
	c := mix(0x24402a, 0x3a5a30, fbm(px/12, py/12, seed+232))
	kind := byte(1)
	if poison {
		c, kind = mix(0x4a6a1a, 0x8aa02a, fbm(px/9, py/9, seed+233)), 2
	}
	wd.glyph[i], wd.color[i] = pick("~-~=.", hash2(x, y, seed+234)), dim(c, 0.7+0.3*rnd01(x, y, seed+235))
	wd.solid[i], wd.over[i], wd.water[i], wd.marsh[i] = false, false, false, kind
}

// bridge pinta a ponte de tábuas onde um caminho cruza o rio.
func (wd *World) bridge(x, y int, seed uint32) {
	l := 0.8 + 0.25*rnd01(x, y, seed+440)
	ch := byte('=')
	if y%3 == 0 {
		ch, l = '#', l*0.75
	}
	wd.put(x, y, ch, dim(0x9a7048, l), false, false)
}

// dock monta o píer de madeira que entra na água até a ponta (ex, ey).
func (wd *World) dock(ex, ey int, seed uint32) {
	// O píer vem da margem mais próxima: procura terra para o oeste.
	sx := ex
	for sx > ex-60 && wd.water[ey*wd.w+sx] {
		sx--
	}
	for y := ey - 2; y <= ey+1; y++ {
		for x := sx - 4; x <= ex+2; x++ {
			ch, c := byte('='), rgb(0x8a6440)
			if (x-sx)%6 == 0 {
				ch, c = '#', 0x5a3e26
			}
			wd.put(x, y, ch, dim(c, 0.8+0.3*rnd01(x, y, seed+450)), false, false)
		}
	}
	// Estacas nas pontas.
	for _, x := range []int{ex + 2, ex + 3} {
		for y := ey - 4; y <= ey-2; y++ {
			wd.put(x, y, '|', 0x6a4a2e, true, false)
		}
	}
}

// boulder desenha uma rocha com o pé em (cx, cy); a metade de baixo bloqueia.
func (wd *World) boulder(cx, cy, r int, tint rgb, seed uint32) {
	for dy := -r; dy <= 0; dy++ {
		for dx := -r - 2; dx <= r+2; dx++ {
			e := math.Hypot(float64(dx)/(float64(r)+2), float64(dy)/float64(r)) + (valueNoise(float64(cx+dx)/3, float64(cy+dy)/3, seed+460)-0.5)*0.3
			if e > 1 {
				continue
			}
			l := clamp01(0.95-0.35*float64(dx-dy)/float64(r+2)) * (0.8 + 0.3*rnd01(cx+dx, cy+dy, seed+461))
			ch := pick(stoneGlyphs, hash2(cx+dx, cy+dy, seed+462))
			if hash2(cx+dx, cy+dy, seed+463)%9 == 0 {
				ch = '%' // rachaduras
			}
			wd.put(cx+dx, cy+dy, ch, dim(tint, l+0.15), dy >= -r/2, false)
		}
	}
}

// bones espalha uma ossada: crânio e costelas; com carcass, ainda há o couro
// escuro de um bicho morto em volta.
func (wd *World) bones(cx, cy int, carcass bool, seed uint32) {
	if carcass {
		for dy := -3; dy <= 1; dy++ {
			for dx := -9; dx <= 9; dx++ {
				if math.Hypot(float64(dx)/9, float64(dy+1)/2.6) < 1 && hash2(cx+dx, cy+dy, seed+470)%3 != 0 {
					wd.put(cx+dx, cy+dy, pick("%&#o", hash2(dx, dy, seed+471)), dim(0x5a3a2a, 0.7+0.3*rnd01(dx, dy, seed+472)), false, false)
				}
			}
		}
	}
	for k := -2; k <= 2; k++ { // costelas
		wd.put(cx+k*2, cy-1, '(', 0xe8dcc0, false, false)
		wd.put(cx+k*2+1, cy-1, ')', 0xd8ccb0, false, false)
	}
	for x := cx - 5; x <= cx+5; x++ {
		wd.put(x, cy, '=', 0xc8bca0, false, false) // espinha
	}
	wd.put(cx+7, cy-1, 'O', 0xf0e6d0, false, false) // crânio
	wd.put(cx+8, cy-1, 'o', 0xd8ccb0, false, false)
}

// mouth ergue um morro de pedra com a entrada escura da caverna em (cx, cy).
func (wd *World) mouth(cx, cy int, seed uint32) {
	for dy := -18; dy <= 0; dy++ {
		for dx := -26; dx <= 26; dx++ {
			e := math.Hypot(float64(dx)/26, float64(dy)/18) + (valueNoise(float64(cx+dx)/5, float64(cy+dy)/5, seed+480)-0.5)*0.25
			if e > 1 {
				continue
			}
			l := clamp01(1-0.3*float64(dx-dy)/26) * (0.75 + 0.3*rnd01(cx+dx, cy+dy, seed+481))
			wd.put(cx+dx, cy+dy, pick(stoneGlyphs, hash2(cx+dx, cy+dy, seed+482)), dim(0x7a7268, l), true, false)
		}
	}
	// Boca em arco, preta por dentro, com a soleira livre para pisar.
	for dy := -10; dy <= 0; dy++ {
		for dx := -6; dx <= 6; dx++ {
			if dy < -6 && math.Hypot(float64(dx), float64(dy+6)) > 6 {
				continue
			}
			wd.put(cx+dx, cy+dy, ' ', 0, dy < -2, false)
		}
	}
}

// pit cava um buraco no chão: borda de terra e o fundo escuro.
func (wd *World) pit(cx, cy int, seed uint32) {
	for dy := -5; dy <= 5; dy++ {
		for dx := -9; dx <= 9; dx++ {
			d := math.Hypot(float64(dx)/9, float64(dy)/5)
			switch {
			case d < 0.7:
				wd.put(cx+dx, cy+dy, ' ', 0, false, false)
			case d < 1:
				wd.put(cx+dx, cy+dy, pick(":.;", hash2(dx, dy, seed+490)), dim(0x5a4630, 0.6+0.3*rnd01(dx, dy, seed+491)), false, false)
			}
		}
	}
}

// caveFloor diz se a célula do subsolo é chão de caverna.
func caveFloor(x, y int, seed uint32) bool {
	px, py := float64(x), float64(y)
	n := (valueNoise(px/7, py/7, seed+500) - 0.5)
	for _, r := range caveRooms {
		if math.Hypot((px-r[0])/r[2], (py-r[1])/r[3])+n*0.35 < 1 {
			return true
		}
	}
	for _, t := range caveTunnels {
		a, b := caveRooms[t[0]], caveRooms[t[1]]
		if segDist(px, py, a[0], a[1], b[0], b[1]) < 4+n*3 {
			return true
		}
	}
	return false
}

// caves escava as cavernas: chão de terra escura, paredes de rocha marrom
// (mais altas do lado norte), cogumelos que brilham e as saídas.
func (wd *World) caves(seed uint32) {
	// O chão é calculado uma vez numa grade; as paredes só consultam a grade.
	const gx0, gx1 = 556, 1404
	gy0 := overH + 30
	gw := gx1 - gx0
	grid := make([]bool, gw*(deepY0-gy0))
	for y := gy0; y < deepY0; y++ {
		for x := gx0; x < gx1; x++ {
			grid[(y-gy0)*gw+x-gx0] = caveFloor(x, y, seed)
		}
	}
	floor := func(x, y int) bool {
		return x >= gx0 && x < gx1 && y >= gy0 && y < deepY0 && grid[(y-gy0)*gw+x-gx0]
	}
	for y := overH + 40; y < deepY0-4; y++ {
		for x := 560; x < 1400; x++ {
			if floor(x, y) {
				ch, c := byte(' '), dim(mix(0x2a2420, 0x3e342a, fbm(float64(x)/16, float64(y)/16, seed+501)), 0.6+0.4*rnd01(x, y, seed+502))
				if rnd01(x, y, seed+503) < 0.35 {
					ch = pick(".,:;'`", hash2(x, y, seed+504))
				}
				wd.put(x, y, ch, c, false, false)
				continue
			}
			for dy := -2; dy <= 7; dy++ {
				if floor(x-2, y+dy) || floor(x, y+dy) || floor(x+2, y+dy) {
					l := 0.5 + 0.35*rnd01(x, y, seed+505)
					if dy < 0 {
						l *= 0.5
					}
					wd.put(x, y, pick(stoneGlyphs, hash2(x, y, seed+506)), dim(0x8a7460, l), true, false)
					break
				}
			}
		}
	}
	// Cogumelos luminosos espalhados pelos salões.
	for i, r := range caveRooms {
		for k := 0; k < 3; k++ {
			h := hash2(i, k, seed+510)
			x, y := int(r[0])+int(h%uint32(r[2]))-int(r[2])/2, int(r[1])+int(h>>8%uint32(r[3]))-int(r[3])/2
			if !wd.Walkable(x, y) {
				continue
			}
			wd.put(x, y, 'T', 0x9aff8a, false, false)
			wd.put(x+1, y, 't', 0x6ad07a, false, false)
			wd.lights = append(wd.lights, light{x, y, 30, 3})
		}
	}
	// Corda pendurada na gruta do buraco.
	for k := 0; k < 9; k++ {
		wd.put(ropeX, ropeY-2-k, '|', 0xb08a5a, false, false)
	}
}
