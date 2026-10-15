package main

// Pontos de venda dos mercadores fora da vila. Morvan e Ysolde já têm cabana
// na vila (kit_vila.go); os outros ficavam soltos no mapa. A Velha do Charco
// ganha uma cabana igual; os de lugar apertado (a ilha, o pátio do castelo, a
// sala secreta da caverna e o fundo) ganham uma barraca: balcão, toldo de
// pano listrado em postes, mercadoria em cima e placa. Nada da barraca é
// sólido, para nunca fechar um corredor, a passagem de um pátio ou um baú.

type merchantPost struct {
	name, sign string
	ink        rgb
	cloth      [2]rgb // listras do toldo
	goods      []rgb
	hut        bool // cabana inteira (precisa de espaço) em vez de barraca
}

var merchantPosts = []merchantPost{
	{"A Velha do Charco", "Ervas do Charco", 0x90c060, [2]rgb{}, []rgb{0x8ae060, 0x60d0a0, 0xb070c0}, true},
	{"O Eremita da Ilha", "Eremita", 0xd0c090, [2]rgb{0x6a5a3a, 0x9a8a5a}, []rgb{0xe04058, 0xc8a060}, false},
	{"Velho Guardião de Vaelor", "Guardião", 0xb0b8d8, [2]rgb{0x2a3a6a, 0x8a8aa0}, []rgb{0xc8c8d8, 0xd0a050}, false},
	{"Mascate das Sombras", "Mascate", 0xa080d0, [2]rgb{0x3a2050, 0x6a4a80}, []rgb{0xb8c0d8, 0xd0a050, 0x8a60c0}, false},
	{"Durgo, o anão do fundo", "Forja de Durgo", 0xff9040, [2]rgb{0x5a2a1a, 0x8a4a2a}, []rgb{0x2a2a30, 0xff7030, 0xc8c8d8}, false},
}

func (wd *World) buildMerchantPosts(seed uint32) {
	for k, mp := range merchantPosts {
		i := -1
		for j := range wd.shops {
			if wd.shops[j].Name == mp.name {
				i = j
			}
		}
		if i < 0 {
			continue
		}
		x, y := wd.shops[i].X, wd.shops[i].Y
		s := seed + 960 + uint32(k)*7
		if mp.hut {
			mx, my := wd.shopHut(x, y+1, mp.sign, mp.ink, mp.goods, s)
			wd.moveShop(mp.name, mx, my)
			continue
		}
		wd.stall(x, y, mp, s)
	}
	wd.settleProps() // árvores que a cabana cobriu deixam de ser quebráveis ali
}

// stall monta a barraca com o mercador em (x, y), de frente para o sul.
func (wd *World) stall(x, y int, mp merchantPost, seed uint32) {
	const hw = 8
	// put transforma a célula em chão: na água não se põe nada (a ilha do
	// Eremita continua só de barco).
	put := func(x, y int, ch byte, c rgb, solid, over bool) {
		if wd.inBounds(x, y) && !wd.water[y*wd.w+x] {
			wd.put(x, y, ch, c, solid, over)
		}
	}
	under := y >= overH
	wood, counter := rgb(0x5a3c22), rgb(0x8a6440)
	if under {
		wood, counter = 0x4a4450, 0x6a6470 // no subsolo: pedra
	}
	// Balcão na frente do mercador, com a mercadoria em cima.
	for dx := -hw + 1; dx <= hw-1; dx++ {
		put(x+dx, y+3, '=', dim(counter, 0.85+0.15*rnd01(x+dx, y, seed)), false, false)
		if len(mp.goods) > 0 && rnd01(x+dx, y+2, seed+1) < 0.6 && (dx < -1 || dx > 1) {
			put(x+dx, y+2, pick("o*%&", hash2(x+dx, y, seed+2)), mp.goods[int(hash2(dx, 3, seed+3))%len(mp.goods)], false, true)
		}
	}
	// Postes e toldo listrado por cima do mercador.
	for _, dx := range []int{-hw, hw} {
		for yy := y - 11; yy <= y+3; yy++ {
			put(x+dx, yy, '|', dim(wood, 0.9), false, yy < y+2) // nada da barraca bloqueia
		}
	}
	for yy := y - 14; yy <= y-10; yy++ {
		for dx := -hw - 1; dx <= hw+1; dx++ {
			c := mp.cloth[((dx+hw+1)/2)%2]
			ch := byte('=')
			if yy == y-10 {
				ch = "VvYv"[hash2(x+dx, yy, seed+4)%4] // franja
			}
			put(x+dx, yy, ch, dim(c, 0.65+0.3*float64(yy-(y-14))/4), false, true)
		}
	}
	wd.signText(x, y-15, mp.sign, mp.ink, seed+5)
	wd.lights = append(wd.lights, light{x + hw + 2, y + 2, 20, 1})
}
