package main

import "math"

// Biomas novos da escala 3 (15/10/2026): com o mapa maior, os ermos entre as
// regiões viraram terras próprias, cada uma com chão, mata, água e dois
// inimigos:
//   - Brejo das Garças (sul): pântano grande de lama andável, poças e juncos;
//   - Mangue de Sal (costa leste): canais de água salgada entre raízes;
//   - Floresta Cinzenta (norte): mata morta, chão de cinza;
//   - Planalto dos Ventos (oeste): capim alto amarelo, pouca árvore, rochas;
//   - Charneca Roxa (nordeste): urze roxa em flor e moitas baixas.
// Ficam no fim de regions (os índices das missões não mudam) e cada uma tem
// fogueira (no fim de bonfiresBase) e trilha.

type newBiome struct {
	name     string
	x, y     float64
	rx, ry   float64
	ground   rgb     // cor do chão
	dens     float64 // quanto o chão leva caractere
	leaf     [2]rgb  // cor da mata
	forest   float64 // multiplica a mata (0 abre tudo, >1 fecha)
	water    float64 // fração de poças/canais (0 nenhuma)
	salt     bool    // canais de água de verdade (mangue), não lama
	mobs     [2]int
	fire     [2]int // deslocamento da fogueira em relação ao centro
	fireName string
}

var newBiomes = []newBiome{
	{"Brejo das Garças", 570 * mapScale, 1065 * mapScale, 260, 180, 0x4a5a34, 0.22, [2]rgb{0x3a5a2a, 0x6a8a3a}, 0.55, 0.42, false,
		[2]int{mobBullFrog, mobLeech}, [2]int{-40, -120}, "Fogueira do Brejo"},
	{"Mangue de Sal", 1420 * mapScale, 640 * mapScale, 200, 280, 0x4a4a30, 0.18, [2]rgb{0x2a4a32, 0x4a6a3a}, 1.15, 0.35, true,
		[2]int{mobMudCrab, mobCroc}, [2]int{-120, 0}, "Fogueira do Mangue"},
	{"Floresta Cinzenta", 545 * mapScale, 165 * mapScale, 300, 200, 0x3a3634, 0.3, [2]rgb{0x4a4440, 0x6a625a}, 1.1, 0, false,
		[2]int{mobCrow, mobWerewolf}, [2]int{0, 150}, "Fogueira da Floresta Cinzenta"},
	{"Planalto dos Ventos", 150 * mapScale, 560 * mapScale, 240, 360, 0x8a8a40, 0.34, [2]rgb{0x7a7a34, 0xa8a050}, 0.25, 0, false,
		[2]int{mobHarpy, mobCentaur}, [2]int{120, 40}, "Fogueira do Planalto"},
	{"Charneca Roxa", 1300 * mapScale, 140 * mapScale, 310, 190, 0x5a3a5a, 0.3, [2]rgb{0x5a3a6a, 0x8a5a8a}, 0.45, 0, false,
		[2]int{mobMoth, mobHeathWitch}, [2]int{-20, 140}, "Fogueira da Charneca"},
}

func init() {
	for _, b := range newBiomes {
		regions = append(regions, region{b.name, int(b.x), int(b.y), int(math.Max(b.rx, b.ry))})
		bonfiresBase = append(bonfiresBase, bonfire{b.fireName, int(b.x) + b.fire[0], int(b.y) + b.fire[1]})
	}
	// Trilhas: da fogueira mais perto de cada lugar conhecido até o bioma.
	paths = append(paths,
		[4]float64{graveX - 60, graveY - 120, 545 * mapScale, 165*mapScale + 150},         // cemitério → Floresta Cinzenta
		[4]float64{desertX + 200, desertY - 60, 1300*mapScale - 20, 140*mapScale + 140},   // deserto → Charneca Roxa
		[4]float64{villageX - 120, villageY - 40, 150*mapScale + 120, 560*mapScale + 40},  // vila → Planalto dos Ventos
		[4]float64{radiantX - 120, radiantY + 40, 570*mapScale - 40, 1065*mapScale - 120}, // Bastião → Brejo das Garças
		[4]float64{swampX + 200, swampY - 60, 1420*mapScale - 120, 640 * mapScale},        // pântano → Mangue de Sal
	)
}

// newBiomeAt devolve o bioma novo em (px, py) e a distância normalizada
// (menos de 1 é dentro), ou -1.
func newBiomeAt(px, py float64) (int, float64) {
	for i, b := range newBiomes {
		if math.Abs(px-b.x) > b.rx*1.3 || math.Abs(py-b.y) > b.ry*1.3 {
			continue
		}
		if d := math.Hypot((px-b.x)/b.rx, (py-b.y)/b.ry); d < 1.15 {
			return i, d
		}
	}
	return -1, 9
}

// forestMod abre ou fecha a mata dentro dos biomas novos.
func forestMod(px, py, f float64) float64 {
	i, d := newBiomeAt(px, py)
	if i < 0 {
		return f
	}
	k := clamp01((1.1 - d) * 3)
	return f * (1 + (newBiomes[i].forest-1)*k)
}

// newBiomeWater faz as poças do brejo e os canais do mangue (antes do chão).
func (wd *World) newBiomeWater(x, y int, px, py float64, seed uint32) bool {
	i, d := newBiomeAt(px, py)
	if i < 0 || d > 0.95 || pathDist(px, py) < 10 {
		return false
	}
	b := newBiomes[i]
	if b.water <= 0 {
		return false
	}
	n := fbm(px/18, py/14, seed+1600+uint32(i))
	if n < 1-b.water {
		return false
	}
	if b.salt {
		wd.sea(x, y, seed)
		wd.color[y*wd.w+x] = mix(0x2a4a5a, 0x3a6a6a, fbm(px/9, py/9, seed+1610))
		return true
	}
	wd.bog(x, y, false, seed) // lama andável e lenta
	return true
}

// newBiomeGround pinta o chão do bioma (chamado de ground).
func (wd *World) newBiomeGround(x, y int, px, py float64, col rgb, dens float64, seed uint32) (rgb, float64, bool) {
	i, d := newBiomeAt(px, py)
	if i < 0 || d >= 1.1 {
		return col, dens, false
	}
	b := newBiomes[i]
	k := clamp01((1.1 - d) * 4)
	col, dens = mix(col, mix(b.ground, dim(b.ground, 1.3), fbm(px/20, py/20, seed+1620)), k*0.85), dens+(b.dens-0.15)*k
	idx := y*wd.w + x
	h := hash2(x, y, seed+1621)
	switch {
	case b.name == "Charneca Roxa" && k > 0.6 && h%23 == 0: // urze em flor
		wd.glyph[idx], wd.color[idx] = pick("*\"'", h>>8), []rgb{0xc070d0, 0xe090e0, 0xa050b0}[h>>12%3]
		return col, dens, true
	case b.name == "Planalto dos Ventos" && k > 0.6 && h%5 == 0: // capim alto
		wd.glyph[idx], wd.color[idx] = pick("|/\\\"", h>>8), dim(0xc8b860, 0.7+0.3*rnd01(x, y, seed+1622))
		return col, dens, true
	case b.name == "Floresta Cinzenta" && k > 0.6 && h%41 == 0: // galho caído e osso
		wd.glyph[idx], wd.color[idx] = pick("~-/", h>>8), 0x6a5e54
		return col, dens, true
	case b.name == "Brejo das Garças" && k > 0.6 && h%17 == 0: // junco
		wd.glyph[idx], wd.color[idx] = pick("|!", h>>8), dim(0x7a9a4a, 0.8+0.2*rnd01(x, y, seed+1623))
		return col, dens, true
	}
	return col, dens, false
}

// newBiomeLeaf é a cor da mata no bioma novo, se houver.
func newBiomeLeaf(px, py float64) ([2]rgb, bool) {
	if i, d := newBiomeAt(px, py); i >= 0 && d < 1.05 {
		return newBiomes[i].leaf, true
	}
	return [2]rgb{}, false
}

// buildNewBiomes espalha os inimigos (depois da paisagem).
func (wd *World) buildNewBiomes(seed uint32) {
	for i, b := range newBiomes {
		cx, cy := int(b.x), int(b.y)
		r := int(math.Min(b.rx, b.ry))
		wd.scatter(b.mobs[0], cx, cy, r, 30, 8, seed+1700+uint32(i)*3)
		wd.scatter(b.mobs[1], cx, cy, r, 50, 5, seed+1701+uint32(i)*3)
	}
}
