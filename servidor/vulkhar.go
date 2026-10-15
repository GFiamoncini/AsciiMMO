package main

import (
	"math"
	"time"
)

// Vulkhar, o Flagelo das Profundezas: o demônio de fogo e sombra que os anões
// de Dorrim acordaram cavando fundo demais. Mora no Abismo de Dorrim, ao sul
// dos salões: um salão cortado por um precipício, com uma ponte estreita de
// pedra. Além do pisão dos chefes, ele tem o chicote de chamas (marca uma
// linha no chão e, um instante depois, ela arde) e a aura de fogo (queima
// quem fica colado). Abaixo de metade da vida entra em fúria: chicote mais
// rápido e aura maior.

const (
	abyssX, abyssY = dorrimX, dorrimY + dorrimHH + 62 // centro do Abismo de Dorrim
	abyssR         = 46
	chasmHalf      = 6  // meia altura do precipício
	bridgeHalf     = 2  // meia largura da ponte
	vulkharHomeDY  = 26 // Vulkhar espera do outro lado da ponte

	whipLen     = 52.0
	whipWidth   = 5.0
	whipWarn    = 700 * time.Millisecond
	whipCD      = 5 * time.Second
	whipCDRage  = 3 * time.Second
	auraR       = 16.0
	auraRRage   = 24.0
	auraDmg     = 7
	auraEvery   = time.Second
	vulkharName = "Vulkhar, o Flagelo das Profundezas"
)

// demon é o estado dos golpes próprios de Vulkhar (um só no mundo).
type demon struct {
	whipArmed      bool
	whipAt, nextAt time.Time
	whipX, whipY   float64
	whipDir        float64
	auraAt         time.Time
	raged          bool
}

// carveAbyss abre, na grade, o corredor da porta sul de Dorrim e o salão.
func carveAbyss(carve func(cx, cy, r float64)) {
	x := float64(dorrimX)
	for y := float64(dorrimY + dorrimHH - deepY0); y <= float64(abyssY-abyssR+6-deepY0); y++ {
		carve(x, y, 3)
	}
	carve(x, float64(abyssY-deepY0), abyssR)
}

// buildAbyss pinta o precipício (rocha que não se pisa, com brasa lá no
// fundo), a ponte e a porta sul dos salões.
func (wd *World) buildAbyss(seed uint32) {
	for y := abyssY - chasmHalf; y <= abyssY+chasmHalf; y++ {
		for x := abyssX - abyssR - 4; x <= abyssX+abyssR+4; x++ {
			i := y*wd.w + x
			if !wd.inBounds(x, y) || wd.solid[i] && wd.glyph[i] == ' ' {
				continue // rocha de fora fica como está
			}
			if abs(x-abyssX) <= bridgeHalf {
				wd.put(x, y, '=', dim(0x8a8478, 0.7+0.2*rnd01(x, y, seed+1600)), false, false)
				wd.marsh[i] = 0
				continue
			}
			// O fundo: escuro, com uma brasa aqui e ali muito lá embaixo.
			ch, c := byte(' '), rgb(0x000000)
			glow := false
			switch v := rnd01(x, y, seed+1601); {
			case v < 0.05:
				ch, c, glow = '*', dim(0xff6020, 0.5+0.3*rnd01(x, y, seed+1602)), true // brasa que brilha lá no fundo
			case v < 0.25:
				ch, c = pick(".:", hash2(x, y, seed+1604)), dim(0x802010, 0.3+0.3*rnd01(x, y, seed+1602))
			}
			wd.put(x, y, ch, c, true, false)
			wd.marsh[i], wd.glow[i] = 0, glow
		}
	}
	// Bordas do precipício: lascas.
	for _, y := range []int{abyssY - chasmHalf - 1, abyssY + chasmHalf + 1} {
		for x := abyssX - abyssR; x <= abyssX+abyssR; x++ {
			if wd.Walkable(x, y) && abs(x-abyssX) > bridgeHalf {
				wd.put(x, y, pick("_~-", hash2(x, y, seed+1603)), dim(0xb09a84, 0.85), false, false)
			}
		}
	}
	// Porta sul dos salões.
	for x := dorrimX - 2; x <= dorrimX+2; x++ {
		y := dorrimY + dorrimHH
		wd.put(x, y, '.', dim(0x4a4c5c, 0.7), false, false)
	}
	wd.lights = append(wd.lights, light{abyssX, abyssY + vulkharHomeDY, 60, 0})
	wd.spawns = append(wd.spawns, spawnPt{abyssX, abyssY + vulkharHomeDY, mobVulkhar})
}

// demonStep são os golpes próprios de Vulkhar. Devolve true enquanto ele
// está ocupado com um (não anda nem bate normalmente).
func (wd *World) demonStep(m *Mob, k mobKind, target *Player, d float64, now time.Time) bool {
	dm := &wd.demon
	rage := m.HP*2 < m.MaxHP
	if rage && !dm.raged {
		dm.raged = true
		wd.effects = append(wd.effects, Effect{Kind: fxBlast, X: m.X, Y: m.Y, R: 40, Start: now, Dur: 800 * time.Millisecond})
		for _, p := range wd.near(m.X, m.Y) {
			wd.pop(p, "Vulkhar abre as asas de fogo!", 0xff6020, m.X, m.Y-60, now)
		}
	}
	if !rage {
		dm.raged = false // renasceu ou curou
	}
	// Aura: queima quem está colado.
	r := auraR
	if rage {
		r = auraRRage
	}
	if now.After(dm.auraAt) {
		dm.auraAt = now.Add(auraEvery)
		for _, p := range wd.players {
			if dist(float64(p.X), float64(p.Y), m.X, m.Y) <= r {
				dmg := auraDmg + mobLv(m.Kind) - p.Level/2
				if p.Skin == itemAshMantle {
					dmg /= 4
				}
				wd.hurtRaw(p, max(1, dmg), "nas chamas de Vulkhar", now)
			}
		}
	}
	// Chicote: arma, espera, arde.
	if dm.whipArmed {
		if now.After(dm.whipAt) {
			dm.whipArmed = false
			cd := whipCD
			if rage {
				cd = whipCDRage
			}
			dm.nextAt = now.Add(cd)
			ex, ey := dm.whipX+math.Cos(dm.whipDir)*whipLen, dm.whipY+math.Sin(dm.whipDir)*whipLen
			wd.effects = append(wd.effects, Effect{Kind: fxFlame, X: (dm.whipX + ex) / 2, Y: (dm.whipY + ey) / 2, R: whipLen / 2, Dir: dm.whipDir, Start: now, Dur: 600 * time.Millisecond})
			for _, p := range wd.players {
				if segDist(float64(p.X), float64(p.Y), dm.whipX, dm.whipY, ex, ey) <= whipWidth {
					dmg := max(1, int(float64(k.Dmg+10+wd.rng.Intn(k.DmgSp))*lvScale(mobLv(m.Kind), p.Level)))
					wd.hurt(p, dmg, "pelo chicote de Vulkhar", now)
					wd.afflict(p, stHeat, 4*time.Second, now)
				}
			}
		}
		return true
	}
	if target != nil && d < whipLen && d > 10 && now.After(dm.nextAt) {
		dm.whipArmed, dm.whipAt = true, now.Add(whipWarn)
		dm.whipX, dm.whipY = m.X, m.Y
		dm.whipDir = math.Atan2(float64(target.Y)-m.Y, float64(target.X)-m.X)
		wd.effects = append(wd.effects, Effect{Kind: fxWhip, X: m.X, Y: m.Y, R: whipLen, Dir: dm.whipDir, Start: now, Dur: whipWarn})
		return true
	}
	return false
}
