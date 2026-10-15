package main

// Nível dos inimigos, à moda de The Witcher 3: cada tipo tem um nível, que o
// cliente mostra ao lado do nome. A diferença de nível pesa nos dois sentidos:
// bem acima de você, o inimigo apanha menos e bate muito mais; a partir de
// skullGap níveis acima, o cliente mostra uma caveira no lugar do número e um
// golpe dele pode matar de uma vez.

const (
	skullGap     = 6  // níveis acima do jogador a partir dos quais vira caveira
	levelDragon  = 25 // Tiamath
	minDmgFactor = 0.4
)

var mobLevel = map[int]int{
	mobSkeleton: 1, mobDummy: 1, mobRat: 2, mobWarrior: 3, mobBat: 3, mobWolf: 4, mobLord: 6,
	mobAsura: 6, mobOrc: 7, mobSpider: 8, mobQueen: 9, mobOrcWarrior: 9, mobTroll: 10,
	mobTriton: 10, mobOrcChief: 11, mobScarab: 12, mobMummy: 13, mobTyrant: 13,
	mobSentinel: 14, mobIceWolf: 15, mobPharaoh: 16, mobYeti: 17, mobArchon: 17, mobFrostGiant: 20,
	mobCrawler: 16, mobFungusHusk: 17, mobMagmaGolem: 19, mobIgnar: 22, mobApostle: 18, mobGrael: 23, mobSandScorpion: 16, mobRootHorror: 17, mobIceWraith: 18, mobDwarfGuard: 18, mobVulkhar: 27, mobCrystalGolem: 20,
	mobBullFrog: 5, mobLeech: 7, mobHarpy: 8, mobMudCrab: 9, mobCrow: 10, mobMoth: 11, mobCentaur: 12, mobCroc: 13, mobWerewolf: 15, mobHeathWitch: 16,
}

func mobLv(kind int) int {
	if kind == killDragon {
		return levelDragon
	}
	if lv, ok := mobLevel[kind]; ok {
		return lv
	}
	return 1
}

// lvScale é o multiplicador de dano de quem tem nível atk batendo em quem
// tem nível def.
func lvScale(atk, def int) float64 {
	switch d := atk - def; {
	case d >= skullGap:
		return 3 + 0.5*float64(d-skullGap) // caveira: um golpe pode matar
	case d > 0:
		return 1 + 0.15*float64(d)
	default:
		return max(minDmgFactor, 1+0.06*float64(d))
	}
}
