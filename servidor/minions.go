package main

import (
	"math"
	"time"

	"asciimmo/proto"
)

// Servos: mortos erguidos pelo necromante. Seguem o dono, atacam o inimigo
// mais próximo dele e se desfazem quando o tempo acaba ou quando apanham
// demais. Os inimigos também os atacam (servem de escudo).

type minion struct {
	ID, Owner, Kind int
	X, Y            float64
	HP, MaxHP       int
	until, atkAt    time.Time
	bornAt, hurtAt  time.Time
	lo, sp          int // dano do golpe
}

const (
	minionReach = 6.0
	minionAggro = 70.0
	minionCD    = 900 * time.Millisecond
	maxMinions  = 8 // por dono
)

// raise ergue n servos em volta de (x, y) para o dono.
func (wd *World) raise(owner int, x, y float64, n int, life time.Duration, kind, hp, lo, sp int, now time.Time) {
	mine := 0
	for _, m := range wd.minions {
		if m.Owner == owner {
			mine++
		}
	}
	for k := 0; k < n; k++ {
		if mine >= maxMinions { // o mais velho se desfaz para dar lugar
			for i, m := range wd.minions {
				if m.Owner == owner {
					wd.minions = append(wd.minions[:i], wd.minions[i+1:]...)
					break
				}
			}
			mine--
		}
		a := float64(k)/float64(n)*2*math.Pi + 0.4
		mx, my := x+math.Cos(a)*10, y+math.Sin(a)*6
		if !wd.Walkable(int(mx), int(my)) {
			mx, my = x, y
		}
		wd.nextMinion++
		wd.minions = append(wd.minions, &minion{ID: wd.nextMinion, Owner: owner, Kind: kind, X: mx, Y: my, HP: hp, MaxHP: hp,
			until: now.Add(life), bornAt: now, lo: lo, sp: sp})
		wd.effects = append(wd.effects, Effect{Kind: fxGrave, X: mx, Y: my, R: 6, Start: now, Dur: 700 * time.Millisecond})
		mine++
	}
}

// stepMinions move e faz atacar os servos; tira os que acabaram.
func (wd *World) stepMinions(now time.Time) {
	live := wd.minions[:0]
	for _, s := range wd.minions {
		p := wd.players[s.Owner]
		if p == nil || now.After(s.until) || s.HP <= 0 {
			wd.effects = append(wd.effects, Effect{Kind: fxPuff, X: s.X, Y: s.Y, R: 8, Start: now, Dur: 500 * time.Millisecond})
			continue
		}
		live = append(live, s)
		if now.Sub(s.bornAt) < 600*time.Millisecond {
			continue // saindo da cova
		}
		// Alvo: o inimigo vivo mais perto do servo, sem se afastar muito do dono.
		var target *Mob
		best := minionAggro
		for _, m := range wd.mobs {
			if !m.Alive || now.Sub(m.bornAt) < mobRise || dist(m.X, m.Y, float64(p.X), float64(p.Y)) > 110 {
				continue
			}
			if d := dist(m.X, m.Y, s.X, s.Y); d < best {
				target, best = m, d
			}
		}
		switch {
		case target != nil && best > minionReach:
			wd.stepToward(&s.X, &s.Y, target.X, target.Y, 0.9)
		case target != nil && now.Sub(s.atkAt) > minionCD:
			s.atkAt = now
			wd.strikeOpt(s.Owner, now, s.lo, s.sp, around(target.X, target.Y, 1), hitOpt{Elem: elemShadow})
		case target == nil:
			// Sem luta: segue o dono e fica em volta dele.
			if dist(s.X, s.Y, float64(p.X), float64(p.Y)) > 14 {
				wd.stepToward(&s.X, &s.Y, float64(p.X), float64(p.Y), 1.1)
			}
			if dist(s.X, s.Y, float64(p.X), float64(p.Y)) > 160 { // ficou para trás: volta pela sombra
				s.X, s.Y = float64(p.X), float64(p.Y)+4
			}
		}
	}
	wd.minions = live
}

func (wd *World) stepToward(x, y *float64, tx, ty, speed float64) {
	dx, dy := tx-*x, ty-*y
	d := math.Hypot(dx, dy)
	if d < 1 {
		return
	}
	nx, ny := *x+dx/d*speed, *y+dy/d*speed
	switch {
	case wd.Walkable(int(nx), int(ny)):
		*x, *y = nx, ny
	case wd.Walkable(int(nx), int(*y)):
		*x = nx
	case wd.Walkable(int(*x), int(ny)):
		*y = ny
	}
}

// minionNear devolve o servo mais perto de (x, y) dentro de r, para os
// inimigos atacarem em vez do jogador.
func (wd *World) minionNear(x, y, r float64) *minion {
	var best *minion
	bd := r
	for _, s := range wd.minions {
		if d := dist(s.X, s.Y, x, y); d < bd {
			best, bd = s, d
		}
	}
	return best
}

// minionsOf converte os servos para o estado do cliente (vão como inimigos
// com Ally, para o desenho reaproveitar os esqueletos).
func (wd *World) minionsOf(now time.Time) []proto.Mob {
	var out []proto.Mob
	for _, s := range wd.minions {
		name := "Servo"
		if p := wd.players[s.Owner]; p != nil {
			name = "Servo de " + p.Name
		}
		out = append(out, proto.Mob{ID: -s.ID, Kind: s.Kind, Name: name, X: math.Round(s.X*10) / 10, Y: math.Round(s.Y*10) / 10,
			HP: s.HP, MaxHP: s.MaxHP, Age: since(now, s.bornAt), Hit: since(now, s.hurtAt), Atk: since(now, s.atkAt), Ally: true})
	}
	return out
}
