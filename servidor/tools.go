package main

import (
	"math"
	"time"
)

// Ferramentas de ambiente: lanterna, tocha e machado.
//   - Lanterna: luz em volta do jogador; funciona com chuva. Comprada ou achada.
//   - Tocha: luz mais forte e quente, mas dura 5 minutos e a chuva apaga.
//     Pega-se de graça nas fogueiras acesas (E descansa e dá uma tocha).
//   - Machado: na mochila, os golpes derrubam árvores e moitas muito mais
//     rápido e rendem mais madeira.
// Luz acende/apaga clicando no item (UseItem). Com tempestade, quem carrega
// metal (arma de metal equipada) atrai raios; raio que cai em árvore a
// incendeia.

const (
	lightNone = iota
	lightTorch
	lightLantern
)

const torchLife = 5 * time.Minute

// toggleLight acende ou apaga a luz do item. Chamar com o lock.
func (wd *World) toggleLight(p *Player, it int, now time.Time) {
	want := lightLantern
	if it == itemTorch {
		want = lightTorch
	}
	if p.Light == want {
		p.Light = lightNone
		return
	}
	if want == lightTorch && wd.raining(p) {
		wd.pop(p, "A chuva não deixa a tocha acender. Use a lanterna.", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
		return
	}
	if want == lightTorch {
		p.take(itemTorch) // a tocha acesa queima: sai da mochila
		p.torchUntil = now.Add(torchLife)
	}
	p.Light = want
}

// raining diz se chove onde o jogador está (fora do deserto e do subsolo).
func (wd *World) raining(p *Player) bool {
	return (wd.weather == wxRain || wd.weather == wxStorm) && p.Y < overH && desertDist(float64(p.X), float64(p.Y)) >= 1
}

// stepLight apaga a tocha que acabou ou que tomou chuva. Uma vez por segundo.
func (wd *World) stepLight(p *Player, now time.Time) {
	if p.Light != lightTorch {
		return
	}
	switch {
	case now.After(p.torchUntil):
		p.Light = lightNone
		wd.pop(p, "A tocha se apagou.", 0xc08040, float64(p.X), float64(p.Y)-34, now)
	case wd.raining(p):
		p.Light = lightNone
		wd.pop(p, "A chuva apagou a tocha.", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
	}
}

// giveTorch dá uma tocha a quem descansa numa fogueira e não tem nenhuma.
func (wd *World) giveTorch(p *Player, now time.Time) {
	if p.count(itemTorch) == 0 && p.give(itemTorch, 1) {
		wd.pop(p, "Você pegou uma tocha na fogueira.", 0xffa040, float64(p.X), float64(p.Y)-48, now)
	}
}

// metallic diz se o jogador carrega metal à mão (arma de metal equipada).
func (p *Player) metallic() bool {
	if p.Equip == 0 {
		return false
	}
	switch weaponType[p.Equip] {
	case wSword, wDagger, wSpear, wHammer, wClaw:
		return p.Equip != itemStick
	}
	return false
}

// treeDamage é quanto um golpe tira de árvore ou moita: com machado, muito mais.
func (p *Player) chopMult() int {
	if p != nil && p.count(itemAxe) > 0 {
		return 4
	}
	return 1
}

// strikeTree faz um raio cair numa árvore perto de (x, y): ela pega fogo e
// cai, e o capim em volta acende.
func (wd *World) strikeTree(x, y float64, now time.Time) bool {
	var best *prop
	bd := 90.0
	for _, pr := range wd.props {
		if pr.Kind != propTree || !pr.Alive {
			continue
		}
		if d := dist(float64(pr.X), float64(pr.Y), x, y); d < bd {
			best, bd = pr, d
		}
	}
	if best == nil {
		return false
	}
	bx, by := float64(best.X), float64(best.Y)
	wd.effects = append(wd.effects, Effect{Kind: fxBolt, X: bx, Y: by - 14, R: boltRadius, Start: now, Dur: 700 * time.Millisecond})
	best.Alive, best.respawnAt = false, now.Add(propDefs[propTree].Respawn)
	wd.edit(best.off)
	wd.effects = append(wd.effects, Effect{Kind: fxBoom, X: bx, Y: by - 8, R: 10, Dir: projFireball, Start: now, Dur: 600 * time.Millisecond})
	wd.ignite(bx, by, 9, now)
	return true
}

// boltTarget decide onde cai o raio da tempestade: em quem carrega metal (às
// vezes), numa árvore perto de alguém, ou no chão perto de alguém.
func (wd *World) boltTarget(p *Player, now time.Time) (float64, float64, bool) {
	if p.metallic() && wd.rng.Intn(100) < 35 {
		wd.pop(p, "O metal na sua mão atraiu o raio!", 0xd0e0ff, float64(p.X), float64(p.Y)-40, now)
		return float64(p.X), float64(p.Y), false
	}
	a, r := wd.rng.Float64()*2*math.Pi, 15+wd.rng.Float64()*75
	x, y := float64(p.X)+math.Cos(a)*r, float64(p.Y)+math.Sin(a)*r*0.6
	if wd.rng.Intn(100) < 40 && wd.strikeTree(x, y, now) {
		return 0, 0, true
	}
	return x, y, false
}
