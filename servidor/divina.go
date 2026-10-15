package main

import (
	"math"
	"time"
)

// Midrath e a Árvore Divina (tema inspirado em Berserk, com nomes nossos).
// O continente se chama Midrath. No meio dele, entre a vila e a Encruzilhada,
// ergue-se a Árvore Divina: um tronco imenso de galhos em espiral, raízes que
// descem até as Profundezas. Em volta dela rondam os Apóstolos — gente que
// trocou a própria carne por poder — e Grael, o Apóstolo Rei, guarda a
// clareira. Dele saem a Lâmina de Ferro Bruto (grande demais para ser chamada
// de espada) e o Ovo Escarlate; sob a árvore, nas Profundezas, um baú guarda
// a Armadura do Abismo, que dá força a quem sangra e cobra em vida.

const (
	worldName        = "Midrath"
	treeX, treeY     = 620 * mapScale, 480 * mapScale
	treeR            = 64 // raio da copa
	abyssDrain       = 2  // vida por segundo que a Armadura do Abismo cobra
	abyssBonus       = 40 // % de dano a mais com menos de metade da vida
	eggCooldown      = 2 * time.Minute
	eggHealPct       = 30
	treeRiftOffX     = -34
	treeRiftOffY     = 50
	graelX, graelY   = treeX + 26, treeY + 60
	apostleRoamR     = 220
	apostleRoamRMin  = 90
	treeBonfireOffX  = 44
	treeBonfireOffY  = 34
	treeClearR       = 96
	treeTrunkHalfW   = 7
	treeTrunkHeight  = 46
	treeCrownCenterY = treeY - treeTrunkHeight - treeR/2
)

// buildDivineTree ergue a Árvore Divina: clareira, raízes, tronco torcido e
// a copa em espiral, com folhas que brilham de leve.
func (wd *World) buildDivineTree(seed uint32) {
	wd.clearArea(treeX-treeClearR, treeY-treeClearR-40, treeX+treeClearR, treeY+treeClearR/2, seed+1400)
	// Raízes: oito, saindo do pé e se dividindo pelo chão.
	for k := 0; k < 8; k++ {
		a := float64(k)/8*2*math.Pi + 0.3
		x, y := float64(treeX), float64(treeY)
		for s := 0; s < 70; s++ {
			a += (fbm(float64(s)/9, float64(k), seed+1401) - 0.5) * 0.3
			x += math.Cos(a)
			y += math.Sin(a) * 0.55
			w := max(1, 4-s/18)
			for d := -w; d <= w; d++ {
				cx, cy := int(x)+d, int(y)
				if wd.inBounds(cx, cy) && !wd.water[cy*wd.w+cx] {
					wd.put(cx, cy, pick("~=-", hash2(cx, cy, seed+1402)), dim(0x8a6a48, 0.55+0.3*rnd01(cx, cy, seed+1403)), s < 10, false)
				}
			}
		}
	}
	// Tronco: largo embaixo, torcido em espiral (os veios giram com a altura).
	for dy := 0; dy <= treeTrunkHeight; dy++ {
		y := treeY - dy
		hw := treeTrunkHalfW + max(0, 6-dy/3) - dy/16
		for dx := -hw; dx <= hw; dx++ {
			x := treeX + dx
			v := math.Sin(float64(dx)*0.7 + float64(dy)*0.35)
			c := dim(mix(0x5a4430, 0xc8b088, 0.5+0.5*v), 0.75+0.2*rnd01(x, y, seed+1404))
			wd.put(x, y, "|)(/"[hash2(x, y, seed+1405)%4], c, dy < 4, dy >= 4)
		}
	}
	// Copa: braços em espiral (galhos), folhas claras que brilham aqui e ali.
	for dy := -treeR; dy <= treeR; dy++ {
		for dx := -treeR - 20; dx <= treeR+20; dx++ {
			x, y := treeX+dx, treeCrownCenterY+dy
			r := math.Hypot(float64(dx)/1.3, float64(dy))
			if r > float64(treeR)+(valueNoise(float64(x)/6, float64(y)/6, seed+1406)-0.5)*14 || !wd.inBounds(x, y) {
				continue
			}
			a := math.Atan2(float64(dy), float64(dx)/1.3)
			arm := math.Sin(5*a - math.Log(r+1)*3.2) // cinco braços em espiral
			switch {
			case arm > 0.55:
				wd.put(x, y, pick("=/\\", hash2(x, y, seed+1407)), dim(0x9a7a54, 0.7+0.3*rnd01(x, y, seed+1408)), false, true)
			case arm > -0.2 || r < 14:
				c := mix(0xd8e0a0, 0xfff4c0, rnd01(x, y, seed+1409))
				ch := pick("&%@*", hash2(x, y, seed+1410))
				wd.put(x, y, ch, dim(c, 0.55+0.3*rnd01(x, y, seed+1411)), false, true)
				if rnd01(x, y, seed+1412) < 0.02 {
					wd.glow[y*wd.w+x] = true
				}
			}
		}
	}
	wd.lights = append(wd.lights, light{treeX, treeCrownCenterY, 120, 3}, light{treeX, treeY + 8, 60, 3})
	wd.signText(treeX, treeY+20, "A ARVORE DIVINA", 0xe8dca0, seed+1413)
	// Apóstolos rondam a clareira; Grael a guarda.
	wd.scatter(mobApostle, treeX, treeY, apostleRoamR, apostleRoamRMin, 5, seed+1414)
	wd.scatter(mobApostle, castleX, castleCY, 120, 40, 2, seed+1415)
	wd.spawns = append(wd.spawns, spawnPt{graelX, graelY, mobGrael})
}

// abyssStep: a Armadura do Abismo cobra vida a cada segundo (sem matar).
func (wd *World) abyssStep(p *Player, now time.Time) {
	if p.Skin == itemAbyssArmor && p.HP > p.MaxHP/5 {
		p.HP -= abyssDrain
	}
}

// abyssPct é o dano a mais que a armadura dá a quem já sangrou metade.
func (p *Player) abyssPct() int {
	if p.Skin == itemAbyssArmor && p.HP*2 < p.MaxHP {
		return abyssBonus
	}
	return 0
}

// eggCheck: com o Ovo Escarlate, quem cai abaixo de 1/4 da vida recupera
// parte dela (uma vez a cada eggCooldown).
func (wd *World) eggCheck(p *Player, now time.Time) {
	if p.Amulet != itemScarletEgg || p.HP <= 0 || p.HP*4 >= p.MaxHP || now.Before(p.eggReady) {
		return
	}
	p.HP = min(p.MaxHP, p.HP+p.MaxHP*eggHealPct/100)
	p.eggReady = now.Add(eggCooldown)
	wd.pop(p, "O Ovo Escarlate chora sangue", 0xff3040, float64(p.X), float64(p.Y)-44, now)
}
