package main

import (
	"math"
)

// rgb é uma cor 0xRRGGBB.
type rgb uint32

// O mundo é uma grade de células quadradas: cada célula é um "pixel" de pixel
// art, desenhado como um caractere colorido. O continente ocupa as linhas até
// overH; abaixo fica a cripta, isolada por rocha.
// mapScale afasta as regiões umas das outras: o continente tem o dobro do
// lado, e entre elas sobram ermos para explorar. As construções de cada
// região mantêm o tamanho.
const mapScale = 3 // era 2; em 15/10 ele achou o mapa pequeno (chegava rápido a Tiamath)

// bsF é quanto os biomas (deserto, gelo, pântano, selva, mar, montanhas, lago)
// crescem em relação à escala 2, para não virarem ilhas num mar de floresta.
// As construções (vila, cidadela, castelo, acampamento) mantêm o tamanho.
const bsF = float64(mapScale) / 2

const (
	worldW, worldH = 1536 * mapScale, deepY0 + deepH // continente, cripta/cavernas e Profundezas (depths.go)
	overH          = 1200 * mapScale
	cryptDY        = overH - 540 // deslocamento vertical da planta da cripta

	villageX, villageY = 390 * mapScale, 630 * mapScale
	hubX, hubY         = 840 * mapScale, 570 * mapScale  // Encruzilhada Queimada: o antigo covil, onde os caminhos se cruzam
	lairX, lairY       = 280 * mapScale, 215 * mapScale  // Pico de Tiamath, no alto das montanhas a noroeste
	graveX, graveY     = 390 * mapScale, 465 * mapScale  // centro da clareira do cemitério
	chapelX, chapelY   = graveX, graveY - 53             // meio da soleira da capela
	jungleX, jungleY   = 1230 * mapScale, 450 * mapScale // Selva de Marapá
	citadelX, citadelY = jungleX + 60, jungleY - 104     // pátio da Cidadela Asura, dentro da selva (mesma distância de antes)
	radiantX, radiantY = 780 * mapScale, 960 * mapScale  // Bastião Radiante
	tideX, tideY       = 1200 * mapScale, 915 * mapScale // Talássia
)

// Fogueira da vila.
const fireX, fireY = villageX + 4, villageY - 16

// Biomas de fora do núcleo: deserto ao norte, gelo a sudoeste, pântano a
// leste do covil e o acampamento orc no caminho do sul.
const (
	desertX, desertY = 800 * mapScale, 250 * mapScale  // Deserto de Ankar
	snowX, snowY     = 330 * mapScale, 985 * mapScale  // Picos de Gelo
	swampX, swampY   = 1050 * mapScale, 700 * mapScale // Pântano de Venor
	orcX, orcY       = 640 * mapScale, 770 * mapScale  // Acampamento Orc
)

func desertDist(px, py float64) float64 {
	return math.Hypot((px-desertX)/(280*bsF), (py-desertY)/(175*bsF))
}

func snowDist(px, py float64) float64 {
	return math.Hypot((px-snowX)/(266*bsF), (py-snowY)/(175*bsF))
}

func swampDist(px, py float64) float64 {
	return math.Hypot((px-swampX)/(182*bsF), (py-swampY)/(140*bsF))
}

const (
	groundGlyphs = "Q0acdq8o"
	leafGlyphs   = "&@$#%WMQ8KNRgbBD"
	stoneGlyphs  = "0Q8#$B@&"
	woodGlyphs   = "|[]#H"
)

var roofPal = []rgb{0xc96f34, 0xb05c28, 0x8a4519}

// Casas da vila, em células relativas ao centro: x, y, largura, altura.
var houses = [][4]int{
	{-110, -44, 44, 30}, {-50, -58, 38, 26}, {40, -62, 48, 30}, {-108, 30, 42, 30}, {60, 34, 46, 32},
}

// Caminhos entre as regiões.
var paths = [][4]float64{
	{chapelX, chapelY, villageX, villageY - 20},
	{villageX + 15, villageY, hubX - 60, hubY + 5},
	{hubX + 70, hubY - 15, citadelX - 40, citadelY + 30},
	{villageX + 10, villageY + 20, radiantX - 70, radiantY - 20},
	{radiantX + 70, radiantY, tideX - 70, tideY},
	{hubX + 20, hubY + 70, tideX - 60, tideY - 50},
	{hubX, hubY - 80, desertX, desertY + 70},
	{villageX - 20, villageY + 40, snowX + 50, snowY - 80},
	{hubX + 80, hubY + 30, swampX - 70, swampY - 20},
	{graveX - 70, graveY, lairX + 30, lairY + 100}, // trilha da subida às montanhas
	{villageX + 150, villageY + 140, orcX - 40, orcY - 10},
	{1623 * mapScale / 2, 760 * mapScale / 2, castleX0 - 4, breachY}, // trilha até a brecha das Ruínas de Vaelor
	{introGateX + 4, introGateY, villageX - 20, villageY + 40},       // do Vale do Despertar à vila
	{hubX - 60, hubY + 30, masterX + 20, masterY},                    // trilha até o Mestre Oren
}

// region dá nome a um pedaço do mapa, para o minimapa e as missões.
type region struct {
	Name    string
	X, Y, R int
}

var regions = []region{
	{"Cidadela Asura", citadelX, citadelY, 50},
	{"Vila", villageX, villageY, 80},
	{"Cemitério", graveX, graveY, 80},
	{"Pico de Tiamath", lairX, lairY, 55 * mapScale},
	{"Selva de Marapá", jungleX, jungleY, 140 * mapScale},
	{"Bastião Radiante", radiantX, radiantY, 120},
	{"Talássia", tideX, tideY, 105 * mapScale},
	{"Deserto de Ankar", desertX, desertY, 105 * mapScale},
	{"Picos de Gelo", snowX, snowY, 100 * mapScale},
	{"Pântano de Venor", swampX, swampY, 155 * mapScale / 2},
	{"Acampamento Orc", orcX, orcY, 60},
	{"Lago Espelhado", lakeX - lakeRX - 14, lakeY, 60 * mapScale},
	{"Encruzilhada Queimada", hubX, hubY, 85},
	{"Ruínas de Vaelor", castleX, castleCY, 135},
	{"Vale do Despertar", introX, introY, 80},
	{"Terreiro das Sequências", masterX, masterY, 26},
	// Profundezas: os biomas de baixo, cada um sob o que lhe dá nome.
	{"Ventre de Fogo", lairX, deepY0 + lairY, 130 * mapScale},
	{"Grutas de Arenito", desertX, deepY0 + desertY, 110 * mapScale},
	{"Gelo Antigo", snowX, deepY0 + snowY, 105 * mapScale},
	{"Jardim de Fungos", swampX, deepY0 + swampY, 85 * mapScale},
	{"Raízes de Marapá", jungleX, deepY0 + jungleY, 140 * mapScale},
	{"Raízes da Árvore Divina", treeX, deepY0 + treeY, 110},
	{"Árvore Divina", treeX, treeY - 30, 130},
	{dorrimName, dorrimX, dorrimY, 60},       // ruínas anãs nas Profundezas (dorrim.go)
	{"Abismo de Dorrim", abyssX, abyssY, 50}, // a ponte e Vulkhar (vulkhar.go)
	{"Cavernas de Cristal", crystalX, crystalY, 110 * mapScale},
}

// regionAt devolve o índice da região que contém o ponto, ou -1.
func regionAt(x, y int) int {
	for i, r := range regions {
		if dist(float64(x), float64(y), float64(r.X), float64(r.Y)) <= float64(r.R) {
			return i
		}
	}
	return -1
}

// spawnPt é o lugar de onde sai um inimigo.
type spawnPt struct{ X, Y, Kind int }

// light é uma fonte de luz: 0 fogueira, 1 tocha, 2 cristal (luz fria, sem chama).
type light struct{ X, Y, R, Kind int }

// portal leva quem pisa perto de (X, Y) para (TX, TY).
type portal struct {
	X, Y, TX, TY int
	Dungeon      bool // o destino é a cripta
	Label        string
	Fall         bool // buraco: quem cai se machuca
}

// Cripta: salas e corredores (x, y, largura, altura) escavados na rocha, em
// coordenadas da planta; no mundo ficam cryptDY células abaixo.
var cryptRooms = [][4]int{
	{140, 680, 70, 50}, {210, 698, 50, 12}, {260, 660, 90, 80}, {350, 692, 40, 12}, {390, 645, 110, 100},
}

var cryptSpawns = []spawnPt{
	{303, 690, 0}, {325, 678, 1}, {300, 722, 0}, {335, 725, 1}, {280, 726, 0},
	{420, 670, 1}, {470, 670, 1}, {420, 728, 1}, {470, 728, 1}, {445, 700, 2},
}

const (
	cryptInX, cryptInY   = 178, 706 + cryptDY // onde se chega ao descer
	cryptOutX, cryptOutY = 158, 690 + cryptDY // escada de volta
)

func pathDist(px, py float64) float64 {
	best := math.MaxFloat64
	for _, s := range paths {
		vx, vy := s[2]-s[0], s[3]-s[1]
		t := clamp01(((px-s[0])*vx + (py-s[1])*vy) / (vx*vx + vy*vy))
		best = math.Min(best, math.Hypot(px-s[0]-t*vx, py-s[1]-t*vy))
	}
	return best
}

// As distâncias abaixo valem menos de 1 dentro de cada área.
func plazaDist(px, py float64) float64 {
	return math.Hypot((px-villageX)/95, (py-villageY)/62)
}

func graveDist(px, py float64) float64 {
	return math.Hypot((px-graveX)/64, (py-graveY)/74)
}

func jungleDist(px, py float64) float64 {
	return math.Hypot((px-jungleX)/(308*bsF), (py-jungleY)/(266*bsF))
}

func radiantDist(px, py float64) float64 {
	return math.Hypot((px-radiantX)/126, (py-radiantY)/102)
}

func tideDist(px, py float64) float64 {
	return math.Hypot((px-tideX)/(231*bsF), (py-tideY)/(182*bsF))
}

// landAt é positivo em terra firme e negativo no mar; perto de zero é praia.
// O continente é um retângulo de cantos bem arredondados, com a costa
// recortada por ruído.
func landAt(px, py float64, seed uint32) float64 {
	dx, dy := math.Abs(px-worldW/2)/(worldW*0.4785), math.Abs(py-overH/2)/(overH*0.468)
	return 1 - math.Sqrt(math.Sqrt(dx*dx*dx*dx+dy*dy*dy*dy)) + (fbm(px/70, py/70, seed+200)-0.5)*0.2
}

// forestAt passa de 0.6 onde há mata fechada. A mata aperta em volta do
// cemitério e na selva, e abre nas clareiras, nos caminhos e na costa.
func forestAt(x, y int, seed uint32) float64 {
	px, py := float64(x), float64(y)
	ge := graveDist(px, py) + (valueNoise(px/10, py/10, seed+52)-0.5)*0.2
	if ge < 1 || plazaDist(px, py) < 1.25 || dist(px, py, hubX, hubY) < 85 || mountDist(px, py) < 1.05 ||
		radiantDist(px, py) < 1.05 || tideDist(px, py) < 1.05 || dist(px, py, citadelX, citadelY) < 52 ||
		desertDist(px, py) < 1.05 || dist(px, py, orcX, orcY) < 62 || dist(px, py, snowX, snowY) < 40 ||
		pathDist(px, py) < 9+4*valueNoise(px/9, py/9, seed+51) || landAt(px, py, seed) < 0.07 ||
		lakeDist(px, py, seed) < 1.2 || dist(px, py, jungleX, jungleY) < 40 || riverDist(px, py, seed) < riverHalf+3 || nearFeature(px, py, 40) {
		return 0
	}
	f := fbm(px/45, py/45, seed+50) + 0.35*clamp01((1.9-ge)/0.4)
	if jd := jungleDist(px, py); jd < 1.1 {
		f += 0.2 * clamp01((1.1-jd)*4)
	}
	if sd := swampDist(px, py); sd < 1.1 {
		f += 0.12 * clamp01((1.1-sd)*4)
	}
	return f
}
