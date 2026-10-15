package main

import "time"

// Cavernas de Cristal: nas Profundezas, sob o Lago Espelhado. Galerias mais
// largas, paredes cheias de cristais que brilham, Golens de Cristal.

var crystalX, crystalY = lakeX, deepY0 + lakeY

// crystalDist passa de 1 fora das Cavernas de Cristal (coordenadas da
// superfície).
func crystalDist(x, yy float64) float64 {
	return ((x-lakeX)/(260*bsF))*((x-lakeX)/(260*bsF)) + ((yy-lakeY)/(170*bsF))*((yy-lakeY)/(170*bsF))
}

// O cofre de cristal: um cofre de pedra no salão central, de porta selada.
// Três prismas na frente dele; quem quebra os três antes de o primeiro
// voltar a crescer (25 s) abre a porta por um tempo (propSeal volta a fechar).

func crystalVault() (cx, by int) { return crystalX, crystalY - 4 }

var crystalPrisms = [][2]int{{-14, 8}, {14, 8}, {0, 16}}

func (wd *World) buildCrystalVault(seed uint32) {
	cx, by := crystalVault()
	wd.vaultKind(cx, by, propSeal, seed+1800)
	for k, d := range crystalPrisms {
		x, y := crystalX+d[0], crystalY+d[1]
		on, off := wd.stamp(x-1, y-3, x+1, y, func() {
			for dy := 0; dy <= 3; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if abs(dx) == 1 && dy == 3 {
						continue
					}
					wd.put(x+dx, y-dy, "A^"[min(1, dy/3)], mix(0xb090ff, 0xf0e0ff, float64(dy)/3), dy == 0, dy > 0)
					wd.glow[(y-dy)*wd.w+x+dx] = true
				}
			}
		})
		wd.addProp(propPrism, x, y, on, off)
		_ = k
	}
}

// prismsBroken: depois de um prisma cair, se os três estão quebrados, abre o
// cofre de cristal (a porta volta a fechar sozinha, como as seladas).
func (wd *World) prismsBroken(p *Player, now time.Time) {
	for _, pr := range wd.props {
		if pr.Kind == propPrism && pr.Alive {
			return
		}
	}
	cx, by := crystalVault()
	for _, pr := range wd.props {
		if pr.Kind == propSeal && pr.X == cx && pr.Y == by && pr.Alive {
			pr.Alive, pr.respawnAt = false, now.Add(propDefs[propSeal].Respawn)
			wd.edit(pr.off)
			if p != nil {
				wd.say(ChatLine{Text: p.Name + " quebrou os três prismas: o cofre de cristal se abre!"})
			}
		}
	}
}
