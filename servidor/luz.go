package main

import "time"

// Espaço de luz no equipamento: a lanterna ou a tocha fica "na mão" (LightSlot)
// em vez de solta na mochila. Clicar nela na mochila equipa e acende; a tecla
// de luz (msg "light") acende e apaga a equipada; e ela acende sozinha ao
// descer ao subsolo (e apaga sozinha ao voltar para o dia, se foi ela que
// acendeu). A tocha acesa continua queimando 5 min e a chuva apaga.

// equipLight põe a lanterna ou a tocha no espaço de luz e acende.
func (wd *World) equipLight(p *Player, it int, now time.Time) {
	if p.LightSlot == it && p.Light != lightNone {
		p.Light, p.autoLit = lightNone, false // clicar na acesa apaga
		return
	}
	p.LightSlot = it
	p.Light = lightNone
	wd.toggleLight(p, it, now)
	p.autoLit = false
}

// ToggleLight é a tecla de luz: acende ou apaga o que está no espaço de luz
// (ou, vazio, a lanterna da mochila, senão uma tocha).
func (wd *World) ToggleLight(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok {
		return
	}
	now := wd.clock()
	if p.Light != lightNone {
		p.Light, p.autoLit = lightNone, false
		return
	}
	it := p.lightItem()
	if it == 0 {
		wd.pop(p, "Sem lanterna nem tocha. Ysolde vende; a fogueira dá uma tocha.", 0xc0a070, float64(p.X), float64(p.Y)-34, now)
		return
	}
	p.LightSlot = it
	wd.toggleLight(p, it, now)
	p.autoLit = false
}

// lightItem é a luz que dá para acender agora: a do espaço, se ainda houver,
// senão a lanterna, senão uma tocha.
func (p *Player) lightItem() int {
	if p.LightSlot == itemLantern && p.count(itemLantern) > 0 || p.LightSlot == itemTorch && p.count(itemTorch) > 0 {
		return p.LightSlot
	}
	if p.count(itemLantern) > 0 {
		return itemLantern
	}
	if p.count(itemTorch) > 0 {
		return itemTorch
	}
	return 0
}

// autoLight acende a luz equipada ao entrar no subsolo e apaga ao sair para
// o dia (só a que ela mesma acendeu). Uma vez por segundo.
func (wd *World) autoLight(p *Player, now time.Time) {
	under := p.Y >= overH
	if under == p.wasUnder {
		return
	}
	p.wasUnder = under
	if p.LightSlot == 0 {
		return
	}
	if under && p.Light == lightNone {
		if it := p.lightItem(); it != 0 {
			wd.toggleLight(p, it, now)
			p.autoLit = p.Light != lightNone
		}
		return
	}
	day := !isNight(wd.timeOfDay(now.Sub(wd.start).Seconds()))
	if !under && p.autoLit && day && p.Light == lightLantern {
		p.Light, p.autoLit = lightNone, false
	}
}
