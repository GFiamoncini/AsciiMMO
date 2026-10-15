package main

import "math"

// Profundezas, rodada 2: os Salões de Dorrim (ruínas anãs sob a
// Encruzilhada, com a forja apagada e guardiões de pedra), os inimigos de
// cada bioma e um segredo sob cada região: uma câmara atrás de parede
// rachada, ao lado do salão central, com um baú.

const dorrimName = "Salões de Dorrim"

const (
	dorrimX, dorrimY = hubX - 90, deepY0 + hubY + 50
	dorrimHW         = 44 // meia largura dos salões
	dorrimHH         = 26 // meia altura
)

// deepSecret: onde fica a câmara secreta de uma região de baixo (centro da
// câmara, relativo ao centro da região).
const secretDX, secretDY = 38, -10

// deepSecretRegions são as regiões de baixo que guardam câmara secreta.
var deepSecretRegions = []string{"Grutas de Arenito", "Gelo Antigo", "Jardim de Fungos", "Raízes de Marapá", dorrimName}

func regionByName(name string) (region, bool) {
	for _, r := range regions {
		if r.Name == name {
			return r, true
		}
	}
	return region{}, false
}

// carveDeep2 escava, na grade das Profundezas (antes de pintar), o que a
// rodada 2 precisa: os túneis das portas de Dorrim e o caminho até cada
// câmara secreta.
func carveDeep2(carve func(cx, cy, r float64), tunnel func(ax, ay, bx, by float64, k uint32)) {
	carveAbyss(carve)
	carveGates(carve, tunnel)
	cy := float64(dorrimY - deepY0)
	// As duas portas (leste e oeste) levam a duas fendas diferentes.
	east := nearestRift(float64(dorrimX+dorrimHW+3), cy, -1)
	west := nearestRift(float64(dorrimX-dorrimHW-3), cy, east)
	for side, door := range [][2]int{{dorrimHW + 3, east}, {-dorrimHW - 3, west}} {
		dx0, r := float64(dorrimX+door[0]), rifts[door[1]]
		carve(dx0, cy, 4)
		tunnel(dx0, cy, float64(r[0]), float64(r[1]), uint32(700+side))
	}
	for _, name := range deepSecretRegions {
		r, ok := regionByName(name)
		if !ok {
			continue
		}
		x, y := float64(r.X), float64(r.Y-deepY0)
		if name == dorrimName {
			continue // o cofre de Dorrim fica dentro dos salões (buildDorrim)
		}
		sx, sy := x+secretDX, y+secretDY
		carve(sx, sy, 7)
		for t := 0.0; t <= 1; t += 0.02 { // corredor estreito do salão à câmara
			carve(x+(sx-x)*t, y+(sy-y)*t, 2)
		}
	}
}

// nearestRift é a fenda mais perto de (x, y da superfície), fora a skip.
func nearestRift(x, y float64, skip int) int {
	bi, bd := 0, math.MaxFloat64
	for i, r := range rifts {
		if d := math.Hypot(float64(r[0])-x, float64(r[1])-y); d < bd && i != skip {
			bi, bd = i, d
		}
	}
	return bi
}

// buildDorrim pinta os salões: muralha de pedra talhada, piso de lajes,
// colunas, a forja apagada (com um fio de lava) e o cofre do rei atrás de
// uma parede rachada.
func (wd *World) buildDorrim(seed uint32) {
	x0, y0, x1, y1 := dorrimX-dorrimHW, dorrimY-dorrimHH, dorrimX+dorrimHW, dorrimY+dorrimHH
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			edge := x == x0 || x == x1 || y == y0 || y == y1 || x == x0+1 || x == x1-1
			door := (x <= x0+1 || x >= x1-1) && y >= dorrimY-2 && y <= dorrimY+2
			i := y*wd.w + x
			wd.marsh[i], wd.glow[i] = 0, false
			switch {
			case edge && !door:
				wd.put(x, y, pick("#H", hash2(x, y, seed+1500)), dim(0x8a90a8, 0.6+0.2*rnd01(x, y, seed+1501)), true, false)
			case (x-dorrimX)%12 == 0 && (y-dorrimY)%10 == 5 && abs(x-dorrimX) < dorrimHW-6:
				wd.put(x, y, 'O', 0x9aa0b8, true, false) // coluna
			default:
				ch := byte('.')
				if (x+y)%4 == 0 {
					ch = '+'
				}
				wd.put(x, y, ch, dim(0x4a4c5c, 0.6+0.25*rnd01(x, y, seed+1502)), false, false)
			}
		}
	}
	// Forja apagada no meio do lado norte: bigorna, um fio de lava e brasa.
	fx, fy := dorrimX, y0+5
	for dx := -4; dx <= 4; dx++ {
		wd.lavaCell(fx+dx, fy, seed)
	}
	wd.put(fx, fy+2, 'T', 0xb0b0c0, true, false) // bigorna
	wd.lights = append(wd.lights, light{fx, fy, 36, 0})
	// Estátuas dos reis nas duas portas.
	for _, sx := range []int{x0 + 4, x1 - 4} {
		for _, sy := range []int{dorrimY - 4, dorrimY + 4} {
			wd.put(sx, sy, '&', 0xc0b890, true, false)
		}
	}
	// Cofre do rei: sala no canto nordeste, fechada por parede rachada.
	vx0, vy0 := x1-14, y0+2
	cx, cy := vx0+5, vy0+8         // a rachadura, na parede de baixo do cofre
	for y := y0; y <= vy0+8; y++ { // até a muralha: sem fresta em cima
		wd.put(vx0, y, '#', dim(0x8a90a8, 0.7), true, false)
	}
	for x := vx0; x < x1-1; x++ {
		if abs(x-cx) > 2 { // o vão fica com o piso; a rachadura vem por cima
			wd.put(x, vy0+8, '#', dim(0x8a90a8, 0.7), true, false)
		}
	}
	on, off := wd.stamp(cx-2, cy, cx+2, cy, func() {
		for x := cx - 2; x <= cx+2; x++ {
			wd.put(x, cy, '%', dim(0x9a8470, 0.8), true, false)
		}
	})
	wd.addProp(propCrack, cx, cy, on, off)
}

// dorrimVault é onde fica o baú do cofre do rei.
const dorrimVaultX, dorrimVaultY = dorrimX + dorrimHW - 7, dorrimY - dorrimHH + 5

// deepSecrets fecha cada câmara secreta com uma parede rachada no corredor.
func (wd *World) deepSecrets(seed uint32) {
	for k, name := range deepSecretRegions {
		r, ok := regionByName(name)
		if !ok || name == dorrimName {
			continue
		}
		// a rachadura no meio do corredor, um pouco antes da câmara
		t := 0.62
		mx := r.X + int(float64(secretDX)*t)
		my := r.Y + int(float64(secretDY)*t)
		on, off := wd.stamp(mx-3, my-3, mx+3, my+3, func() {
			for y := my - 3; y <= my+3; y++ {
				for x := mx - 3; x <= mx+3; x++ {
					if wd.Walkable(x, y) {
						wd.put(x, y, pick("%#", hash2(x, y, seed+1510+uint32(k))), dim(0x9a8470, 0.75), true, false)
					}
				}
			}
		})
		wd.addProp(propCrack, mx, my, on, off)
	}
}

// scatterDeep2 põe os inimigos de cada bioma nas regiões de baixo e os
// guardiões em Dorrim.
func (wd *World) scatterDeep2(seed uint32) {
	kinds := map[string]int{"Grutas de Arenito": mobSandScorpion, "Gelo Antigo": mobIceWraith,
		"Raízes de Marapá": mobRootHorror, "Raízes da Árvore Divina": mobRootHorror, "Jardim de Fungos": mobFungusHusk, "Cavernas de Cristal": mobCrystalGolem}
	for i, r := range regions {
		if k, ok := kinds[r.Name]; ok {
			wd.scatter(k, r.X, r.Y, min(r.R, 200), 24, 7, seed+1520+uint32(i))
		}
	}
	wd.scatter(mobDwarfGuard, dorrimX, dorrimY, dorrimHW-4, 8, 6, seed+1540)
}
