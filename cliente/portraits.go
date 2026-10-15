package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Retratos em pixel art (16×16) para o card do personagem, um por classe.
// Letras: . fundo  o contorno  k pele  s pele na sombra  e olho  h cabelo
// H chapéu/capuz/elmo  m metal  M metal escuro  c roupa  C roupa escura
// g dourado  w branco (barba, brilho)  r vermelho  v verde aceso.
var portraitArt = [][16]string{
	{ // Guerreiro: cabelo curto, cicatriz, ombreira de ferro
		"................",
		".....oooooo.....",
		"....ohhhhhhho...",
		"...ohhhhhhhhho..",
		"...ohkkkkkkkho..",
		"...okkkkkkkkko..",
		"...okeokkoeoko..",
		"...okkkkkkkrko..",
		"...oskkkkkkrko..",
		"....okkssskko...",
		"....oskkkkkso...",
		"..oooosssssoooo.",
		".ommmmocccommmo.",
		"oMmmmmocccommmMo",
		"oMMmmocccccommMo",
		"oooooooooooooooo",
	},
	{ // Mago: chapéu pontudo torto, barba branca comprida
		".........oo.....",
		"........oHHo....",
		".......oHHHo....",
		"......oHHHHo....",
		".....oHHgHHHo...",
		"..ooooHHHHHHoooo",
		".oHHHHHHHHHHHHHo",
		"..ooookkkkkoooo.",
		"....okeokeokso..",
		"....okkkkkkkso..",
		"....owwkkkwwwo..",
		"...owwwwwwwwwwo.",
		"...owwwwwwwwwwo.",
		"..occowwwwwwocco",
		".occccowwwwocccc",
		"oooooooooooooooo",
	},
	{ // Cavaleiro: elmo fechado com viseira e penacho
		"..........rr....",
		".........rrr....",
		".....oooooro....",
		"....ommmmmmmo...",
		"...ommmmmmmmmo..",
		"...ommmmmmmmMo..",
		"...oMoooooooMo..",
		"...oMoeeMeeoMo..",
		"...oMoooooooMo..",
		"...ommmmmmmmMo..",
		"...ommMmMmMmMo..",
		"....ommmmmmMo...",
		"..ooommmmmmMooo.",
		".occcommmmmocco.",
		"occccccmmmccccco",
		"oooooooooooooooo",
	},
	{ // Paladino: tiara dourada, cabelo loiro, halo
		"....wwwwwwww....",
		"...w........w...",
		".....oooooo.....",
		"....ohhhhhhho...",
		"...ogggggggggo..",
		"...ohkkkkkkkho..",
		"...okkkkkkkkko..",
		"...okeokkoeoko..",
		"...ohkkkkkkkho..",
		"...ohkkssskkho..",
		"....ohkkkkkho...",
		"..oooosssssoooo.",
		".owwwwogggowwwwo",
		"owwwwwogggowwwwo",
		"owwwwgggggggwwwo",
		"oooooooooooooooo",
	},
	{ // Arqueiro: capuz verde, olhar atento
		".....oooooo.....",
		"....oHHHHHHo....",
		"...oHHHHHHHHo...",
		"..oHHHHHHHHHHo..",
		"..oHHkkkkkkHHo..",
		"..oHkkkkkkkkHo..",
		"..oHkeokkoekHo..",
		"..oHkkkkkkkkHo..",
		"..oHHkkssskHHo..",
		"...oHHkkkkHHo...",
		"...oHHHsssHHo...",
		"..oHHHHHHHHHHo..",
		".oCCcccHHcccCCo.",
		"oCCcccccccccccCo",
		"oCccbbbbbbbcccCo",
		"oooooooooooooooo",
	},
	{ // Ladino: capuz escuro, máscara sobre a boca
		".....oooooo.....",
		"....oCCCCCCo....",
		"...oCCCCCCCCo...",
		"..oCCCCCCCCCCo..",
		"..oCCCCCCCCCCo..",
		"..oCCkkkkkkCCo..",
		"..oCkeokkoekCo..",
		"..oCkkkkkkkkCo..",
		"..oCCrrrrrrCCo..",
		"...oCrrrrrrCo...",
		"...oCCrrrrCCo...",
		"..oCCCCCCCCCCo..",
		".oCCCCCCCCCCCCo.",
		"oCCCrCCCCCCrCCCo",
		"oCCCCCCCCCCCCCCo",
		"oooooooooooooooo",
	},
	{ // Necromante: capuz roxo, rosto pálido, olhos verdes acesos
		".....oooooo.....",
		"....oHHHHHHo....",
		"...oHHHHHHHHo...",
		"..oHHHHHHHHHHo..",
		"..oHHHHHHHHHHo..",
		"..oHHsssssssHo..",
		"..oHsvoossovHo..",
		"..oHsssssssHHo..",
		"..oHHsoooosHHo..",
		"...oHHsssssHo...",
		"...oHHHHHHHHo...",
		"..oHHHHvHHHHHo..",
		".oHHHHHvHHHHHHo.",
		"oHHHHHvvvHHHHHHo",
		"oHHHHHHvHHHHHHHo",
		"oooooooooooooooo",
	},
	{ // Depravado: cabelo desgrenhado, barba por fazer, olho roxo, andrajos
		"...o.o..o.o.....",
		"..ohohhohhoho...",
		"..ohhhhhhhhhho..",
		"..ohhhhhhhhhho..",
		"...ohkkkkkkhho..",
		"...okkkkkkkkko..",
		"...okeokCCoeko..",
		"...okkkkCCkkko..",
		"...osksksksksk..",
		"....oskskskso...",
		"....osssssssso..",
		"..ooooosssooooo.",
		".occCcocccoCcco.",
		"occrcCcccccCrcco",
		"oCcccbbbbbbccCco",
		"oooooooooooooooo",
	},
}

// Paleta de cada classe: H chapéu, h cabelo, c roupa, C roupa escura.
var portraitPal = []map[byte]col{
	{'h': {120, 80, 50}, 'c': {150, 110, 80}, 'C': {100, 70, 50}},
	{'H': {80, 60, 160}, 'h': {220, 220, 230}, 'c': {84, 66, 170}, 'C': {50, 40, 110}},
	{'c': {120, 60, 60}, 'C': {80, 40, 40}},
	{'h': {240, 214, 140}, 'c': {232, 232, 240}, 'C': {180, 180, 200}},
	{'H': {60, 110, 70}, 'c': {90, 120, 70}, 'C': {50, 70, 40}},
	{'C': {50, 46, 64}, 'c': {70, 66, 90}},
	{'H': {70, 36, 96}, 'c': {80, 40, 110}, 'C': {50, 26, 70}},
	{'h': {70, 60, 50}, 'c': {110, 92, 66}, 'C': {74, 60, 44}},
}

var portraitBase = map[byte]col{
	'o': {14, 10, 18}, 'k': {232, 196, 160}, 's': {190, 150, 120}, 'e': {30, 24, 30},
	'm': {176, 184, 214}, 'M': {110, 116, 140}, 'g': {236, 196, 92}, 'w': {240, 240, 245},
	'r': {200, 50, 50}, 'v': {140, 255, 170}, 'b': {120, 82, 48}, 'h': {90, 70, 50},
	'H': {90, 60, 160}, 'c': {120, 100, 90}, 'C': {80, 66, 60},
}

var portraitCache = map[int]*ebiten.Image{}

func portrait(cls int) *ebiten.Image {
	cls = clampi(cls, 0, len(portraitArt)-1)
	if img, ok := portraitCache[cls]; ok {
		return img
	}
	px := make([]byte, 16*16*4)
	for y, row := range portraitArt[cls] {
		for x := 0; x < 16 && x < len(row); x++ {
			ch := row[x]
			if ch == '.' {
				continue
			}
			c, ok := portraitPal[cls][ch]
			if !ok {
				c = portraitBase[ch]
			}
			o := (y*16 + x) * 4
			px[o], px[o+1], px[o+2], px[o+3] = cb(c[0]), cb(c[1]), cb(c[2]), 255
		}
	}
	img := ebiten.NewImage(16, 16)
	img.WritePixels(px)
	portraitCache[cls] = img
	return img
}

// drawPortrait desenha o retrato da classe num quadrado de lado size, com um
// fundo escuro em degradê na cor da classe.
func (g *Game) drawPortrait(dst *ebiten.Image, cls int, x, y, size float64) {
	pal := portraitPal[clampi(cls, 0, len(portraitPal)-1)]
	bg := col{40, 34, 50}
	if c, ok := pal['c']; ok {
		bg = mix3(c, col{8, 6, 12}, 0.75)
	}
	g.rect(dst, x, y, size, size, bg.rgba())
	g.rect(dst, x, y+size*0.6, size, size*0.4, mix3(bg, col{0, 0, 0}, 0.4).rgba())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(size*g.ui/16, size*g.ui/16)
	op.GeoM.Translate(x*g.ui, y*g.ui)
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(portrait(cls), op)
	g.stroke(dst, x, y, size, size, rgbaHex(0xd6b050), 1.5)
}
