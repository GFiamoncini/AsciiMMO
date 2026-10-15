package main

import "time"

// Esperar na fogueira: quem descansa pode deixar o tempo passar até uma hora
// do dia (amanhecer, meio-dia, anoitecer, meia-noite) ou até o clima virar.
// O mundo é um só, então vale para todos: o relógio do dia anda (todShift
// soma frações de dia) e o aviso vai para o chat. Com o clima fixado pela
// linha de comando (-weather), ele não muda.

var waitTargets = []struct {
	tod  float64
	name string
}{{0.27, "o amanhecer"}, {0.5, "o meio-dia"}, {0.76, "o anoitecer"}, {0.0, "a meia-noite"}}

// atBonfire diz se o jogador está junto a uma fogueira.
func (wd *World) atBonfire(p *Player) bool {
	for _, b := range wd.bonfires {
		if dist(float64(p.X), float64(p.Y), float64(b.X), float64(b.Y)) <= bonfireReach {
			return true
		}
	}
	return false
}

// Wait faz o tempo passar até a hora i de waitTargets.
func (wd *World) Wait(id, i int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || i < 0 || i >= len(waitTargets) || !wd.atBonfire(p) {
		return
	}
	now := wd.clock()
	tod := wd.timeOfDay(now.Sub(wd.start).Seconds())
	d := waitTargets[i].tod - tod
	for d <= 0.01 {
		d++ // sempre para a frente: a próxima vez que essa hora chegar
	}
	wd.todShift += d
	wd.say(ChatLine{Text: p.Name + " esperou junto à fogueira até " + waitTargets[i].name + "."})
}

// WaitWeather faz o clima virar: sorteia outro tempo, diferente do atual.
func (wd *World) WaitWeather(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || !wd.atBonfire(p) {
		return
	}
	now := wd.clock()
	if wd.wxForced >= 0 {
		wd.pop(p, "O tempo está fixo neste servidor.", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
		return
	}
	old := wd.weather
	for wd.weather == old {
		wd.weather = []int{wxClear, wxRain, wxStorm, wxFog}[wd.rng.Intn(4)]
	}
	wd.wxUntil = now.Add(time.Duration(180+wd.rng.Intn(180)) * time.Second)
	wd.todShift += 0.04 // esperar o tempo virar também passa umas horas
	wd.say(ChatLine{Text: p.Name + " esperou junto à fogueira o tempo virar."})
}

// dayOf é o dia do jogo (a partir de 1) no instante t (segundos desde o início).
func (wd *World) dayOf(t float64) int {
	return int(t/wd.dayLen+0.35+wd.todShift) + 1
}
