package main

import (
	"fmt"
	"math"
	"time"
)

// ---------- fogo no capim ----------
// A bola de fogo e o meteoro incendeiam o capim. Cada célula queima um
// instante, vira cinza e pode passar o fogo para o capim vizinho. A chuva
// apaga. A cinza volta a ser capim depois de um tempo.

const (
	burnTime   = 1400 * time.Millisecond
	ashRegrow  = 4 * time.Minute
	maxBurning = 900
	spreadOdds = 55 // % de passar o fogo para cada vizinho
	flagFire   = 32
	flagGlow   = 64 // emite luz: janela acesa, cristal, lava; o cliente não escurece à noite
)

type burnCell struct {
	until time.Time
	orig  cellEdit // como era antes do fogo
}

type ashCell struct {
	at   time.Time
	orig cellEdit
	ash  cellEdit
}

// isGrass diz se a célula é capim que pega fogo.
func (wd *World) isGrass(i int) bool {
	if wd.solid[i] || wd.water[i] || wd.marsh[i] != 0 || wd.burn[i] || wd.glyph[i] == ' ' || wd.glyph[i] == '.' && wd.color[i] == ashColor {
		return false
	}
	c := wd.color[i]
	r, g, b := float64(c>>16&255), float64(c>>8&255), float64(c&255)
	return g > r*1.08 && g > b*1.05
}

const ashColor rgb = 0x3a3836

// ignite põe fogo no capim em volta de (x, y).
func (wd *World) ignite(x, y, r float64, now time.Time) {
	if wd.weather != wxClear {
		return // na chuva o capim molhado não pega
	}
	var cells []cellEdit
	for dy := -int(r); dy <= int(r); dy++ {
		for dx := -int(r); dx <= int(r); dx++ {
			cx, cy := int(x)+dx, int(y)+dy
			if !wd.inBounds(cx, cy) || math.Hypot(float64(dx), float64(dy)) > r || len(wd.burning) >= maxBurning {
				continue
			}
			if i := cy*wd.w + cx; wd.isGrass(i) {
				cells = append(cells, wd.light(i, now))
			}
		}
	}
	if len(cells) > 0 {
		wd.edit(cells)
	}
}

// light marca a célula em chamas e devolve como ela fica.
func (wd *World) light(i int, now time.Time) cellEdit {
	wd.burning[i] = burnCell{until: now.Add(burnTime + time.Duration(wd.rng.Intn(600))*time.Millisecond), orig: wd.cellAt(i)}
	wd.burn[i] = true
	return cellEdit{I: i, Ch: '^', C: 0xff7a20, F: wd.flagsAt(i)}
}

// stepFire queima, espalha e deixa a cinza voltar a ser capim.
func (wd *World) stepFire(now time.Time) {
	var cells []cellEdit
	rain := wd.weather != wxClear
	for i, b := range wd.burning {
		if now.Before(b.until) {
			continue
		}
		delete(wd.burning, i)
		wd.burn[i] = false
		ash := cellEdit{I: i, Ch: '.', C: ashColor, F: b.orig.F}
		cells = append(cells, ash)
		wd.ashes = append(wd.ashes, ashCell{now.Add(ashRegrow), b.orig, ash})
		if rain {
			continue
		}
		x, y := i%wd.w, i/wd.w
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}, {-2, 0}} {
			nx, ny := x+d[0], y+d[1]
			if !wd.inBounds(nx, ny) || len(wd.burning) >= maxBurning || wd.rng.Intn(100) >= spreadOdds {
				continue
			}
			if j := ny*wd.w + nx; wd.isGrass(j) {
				cells = append(cells, wd.light(j, now))
			}
		}
	}
	// Cinza que já pode voltar: só se ninguém mexeu na célula desde então.
	live := wd.ashes[:0]
	for _, a := range wd.ashes {
		if now.Before(a.at) {
			live = append(live, a)
			continue
		}
		if wd.cellAt(a.orig.I) == a.ash {
			cells = append(cells, a.orig)
		}
	}
	wd.ashes = live
	if len(cells) > 0 {
		wd.edit(cells)
	}
}

// burnDamage fere, uma vez por segundo, quem está em cima do fogo.
func (wd *World) burnDamage(now time.Time) {
	for _, p := range wd.players {
		if wd.inBounds(p.X, p.Y) && wd.burn[p.Y*wd.w+p.X] {
			wd.hurtRaw(p, 4, "pelo fogo", now)
		}
	}
	for _, m := range wd.mobs {
		if m.Alive && wd.inBounds(int(m.X), int(m.Y)) && wd.burn[int(m.Y)*wd.w+int(m.X)] {
			if m.HP -= 6; m.HP <= 0 {
				wd.mobDied(m, nil, now)
			}
		}
	}
}

// ---------- invasão noturna ----------
// Em metade das noites, os mortos sobem em volta de Cinzalva e a fogueira da
// vila se apaga. Acaba quando todos caem (e a vila resiste) ou ao amanhecer.

const invaders = 8

type invasion struct {
	active    bool
	mobs      []*Mob
	lastNight int // última noite em que já se sorteou a invasão
}

func (wd *World) stepInvasion(now time.Time) {
	t := now.Sub(wd.start).Seconds()
	tod, day := wd.timeOfDay(t), wd.dayOf(t)
	inv := &wd.inv
	if inv.active {
		alive := 0
		for _, m := range inv.mobs {
			if m.Alive {
				alive++
			}
		}
		switch {
		case alive == 0:
			wd.say(ChatLine{Text: "Cinzalva resistiu à noite. A fogueira voltou a arder."})
			for _, p := range wd.near(villageX, villageY) {
				p.Gold += 100
				wd.pop(p, fmt.Sprintf("A vila resistiu: +100 %s", currency), 0xffe066, float64(p.X), float64(p.Y)-40, now)
			}
			wd.endInvasion()
		case !isNight(tod):
			wd.say(ChatLine{Text: "O sol nasce e os mortos voltam ao pó."})
			wd.endInvasion()
		}
		return
	}
	if !isNight(tod) || inv.lastNight == day {
		return
	}
	inv.lastNight = day
	if wd.rng.Intn(100) >= 50 {
		return
	}
	inv.active = true
	for k := 0; k < invaders; k++ {
		a := float64(k) / invaders * 2 * math.Pi
		x, y, ok := wd.freeNear(villageX+int(math.Cos(a)*120), villageY+int(math.Sin(a)*80), 30)
		if !ok {
			continue
		}
		kind := mobSkeleton
		if k%3 == 0 {
			kind = mobWarrior
		}
		hp := mobKinds[kind].HP
		wd.nextMob++
		m := &Mob{ID: wd.nextMob, Kind: kind, X: float64(x), Y: float64(y), HP: hp, MaxHP: hp, Alive: true, homeX: float64(villageX), homeY: float64(villageY),
			bornAt: now, zone: -1, temp: true}
		inv.mobs = append(inv.mobs, m)
		wd.mobs = append(wd.mobs, m)
	}
	wd.say(ChatLine{Text: "Os mortos invadem Cinzalva! A fogueira da vila se apagou."})
}

// endInvasion tira do mundo os invasores (vivos ou caídos).
func (wd *World) endInvasion() {
	gone := map[*Mob]bool{}
	for _, m := range wd.inv.mobs {
		gone[m] = true
	}
	live := wd.mobs[:0]
	for _, m := range wd.mobs {
		if !gone[m] {
			live = append(live, m)
		}
	}
	wd.mobs = live
	wd.inv.active, wd.inv.mobs = false, nil
}

// ---------- furtividade ----------

// nearestFor acha o jogador que o inimigo percebe primeiro. O ladino no
// escuro (noite ou subsolo) é notado como se estivesse ao dobro da distância.
func (wd *World) nearestFor(x, y float64, now time.Time) (*Player, float64) {
	night := isNight(wd.timeOfDay(now.Sub(wd.start).Seconds()))
	var target *Player
	best := math.MaxFloat64
	for _, p := range wd.players {
		if p.paused {
			continue // pausado: o inimigo não o vê
		}
		d := dist(float64(p.X), float64(p.Y), x, y)
		if p.Class == 5 && (night || p.Y >= overH) {
			d *= 2
		}
		if d < best {
			target, best = p, d
		}
	}
	return target, best
}
