package main

import "time"

// Cortar vegetação, à moda de Zelda: os golpes cortam o capim e as moitas no
// alcance; o capim vira restolho e volta a crescer em alguns minutos, e às
// vezes larga alguma coisa (umas moedas, uma erva, madeira das moitas).

const (
	grassRegrow = 3 * time.Minute
	cutScan     = 30 // meia largura do quadrado varrido em volta de quem golpeia
)

type regrow struct {
	at   time.Time
	cell cellEdit
}

// cuttable diz se a célula é capim ou moita que se corta (não árvore, casa,
// água ou caminho de pedra).
func (wd *World) cuttable(i int) (grass, bush bool) {
	if wd.over[i] || wd.water[i] || wd.marsh[i] != 0 || wd.glyph[i] == ' ' || wd.glyph[i] == '.' {
		return false, false
	}
	c := wd.color[i]
	r, g, b := int(c>>16&0xff), int(c>>8&0xff), int(c&0xff)
	green := g > r*108/100 && g > b*105/100
	if !green {
		return false, false
	}
	if wd.solid[i] {
		// Moita: as touceiras soltas da mata rasteira ("&%#" isoladas).
		return false, wd.glyph[i] == '&' || wd.glyph[i] == '%' || wd.glyph[i] == '#'
	}
	return true, false
}

// cutGrass corta a vegetação dentro da forma, perto de (x, y). Chamar com o lock.
func (wd *World) cutGrass(owner int, x, y float64, in shape, now time.Time) {
	var cells []cellEdit
	x0, y0 := int(x)-cutScan, int(y)-cutScan
	bushes := 0
	for cy := y0; cy <= y0+2*cutScan; cy++ {
		for cx := x0; cx <= x0+2*cutScan; cx++ {
			if !wd.inBounds(cx, cy) || cy >= overH || !in(float64(cx), float64(cy), 0) {
				continue
			}
			i := cy*wd.w + cx
			grass, bush := wd.cuttable(i)
			if !grass && !bush {
				continue
			}
			old := wd.cellAt(i)
			wd.regrows = append(wd.regrows, regrow{now.Add(grassRegrow + time.Duration(wd.rng.Intn(60))*time.Second), old})
			stub := cellEdit{I: i, Ch: '.', C: dim(old.C, 0.55), F: 0}
			cells = append(cells, stub)
			if bush {
				bushes++
			}
		}
	}
	if len(cells) == 0 {
		return
	}
	wd.edit(cells)
	wd.effects = append(wd.effects, Effect{Kind: fxGrass, X: x, Y: y, R: float64(len(cells)), Start: now, Dur: 400 * time.Millisecond})
	// Achados: a cada ~25 tufos, uma chance de umas moedas ou uma erva.
	if len(cells) >= 6 && wd.rng.Intn(100) < min(60, len(cells)*2) {
		switch roll := wd.rng.Intn(10); {
		case roll < 6:
			wd.dropCoins(2+wd.rng.Intn(5), x, y, 0, lootLife, now)
		case roll < 8:
			wd.drop(herbItem[wd.rng.Intn(len(herbItem))], x, y, 0, now)
		case roll < 9:
			wd.drop(itemPotion, x, y, 0, now)
		default: // na mata, às vezes uma ferramenta esquecida
			if bushes > 0 && wd.rng.Intn(100) < 25 {
				wd.drop(itemAxe, x, y, 0, now)
			} else {
				wd.drop(itemTorch, x, y, 0, now)
			}
		}
	}
	for k := 0; k < bushes*wd.players[owner].chopMult()/4; k++ {
		wd.drop(itemWood, x, y, 0, now)
	}
}

// stepRegrow devolve o capim cortado quando chega a hora (se ninguém estiver
// em cima de uma moita que voltaria a bloquear).
func (wd *World) stepRegrow(now time.Time) {
	if len(wd.regrows) == 0 {
		return
	}
	var back []cellEdit
	keep := wd.regrows[:0]
	for _, r := range wd.regrows {
		if now.Before(r.at) || wd.glyph[r.cell.I] != '.' {
			if now.Before(r.at) {
				keep = append(keep, r)
			}
			continue
		}
		if r.cell.F&flagSolid != 0 {
			x, y := r.cell.I%wd.w, r.cell.I/wd.w
			busy := false
			for _, p := range wd.players {
				if abs(p.X-x) < 3 && abs(p.Y-y) < 2 {
					busy = true
				}
			}
			if busy {
				r.at = now.Add(10 * time.Second)
				keep = append(keep, r)
				continue
			}
		}
		back = append(back, r.cell)
	}
	wd.regrows = keep
	if len(back) > 0 {
		wd.edit(back)
	}
}
