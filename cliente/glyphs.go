package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Glifos próprios, além do ASCII da fonte: lâminas de capim inclinadas, para
// o capim ficar orgânico (só / \ | dá um capim duro), e pontinhos de flor.
const (
	glyphBladeShort = 128 // + 0..11: lâmina curta, de -55° a +55° da vertical
	glyphBladeLong  = 140 // + 0..11: lâmina longa
	glyphFlower     = 152 // + 0..3: ponto, ponto duplo, cruzinha, florzinha
	bladeSteps      = 12
)

// bladeGlyph escolhe a lâmina pela inclinação (-1 deita à esquerda, +1 à direita).
func bladeGlyph(lean float64, long bool) byte {
	i := clampi(int(math.Round((clamp(lean, -1, 1)+1)/2*float64(bladeSteps-1))), 0, bladeSteps-1)
	if long {
		return byte(glyphBladeLong + i)
	}
	return byte(glyphBladeShort + i)
}

func drawCustomGlyphs(atlas *ebiten.Image, cs int) {
	w := float32(math.Max(1, float64(cs)/6))
	at := func(code int) (float32, float32) { return float32(code & 15 * cs), float32(code >> 4 * cs) }
	for i := 0; i < bladeSteps; i++ {
		a := (-55 + 110*float64(i)/float64(bladeSteps-1)) * math.Pi / 180
		for k, l := range []float64{0.55, 0.92} {
			x, y := at(glyphBladeShort + k*bladeSteps + i)
			bx, by := x+float32(cs)/2-float32(math.Sin(a)*float64(cs)*0.18), y+float32(cs)-0.5
			// Lâmina curva: dois trechos, o de cima mais deitado.
			mx, my := bx+float32(math.Sin(a)*l*float64(cs)*0.5), by-float32(math.Cos(a)*l*float64(cs)*0.5)
			a2 := a * 1.35
			tx, ty := mx+float32(math.Sin(a2)*l*float64(cs)*0.5), my-float32(math.Cos(a2)*l*float64(cs)*0.5)
			vector.StrokeLine(atlas, bx, by, mx, my, w*1.2, color.White, true)
			vector.StrokeLine(atlas, mx, my, tx, ty, w*0.8, color.White, true)
		}
	}
	r := float32(math.Max(1, float64(cs)/7))
	for k := 0; k < 4; k++ {
		x, y := at(glyphFlower + k)
		cx, cy := x+float32(cs)/2, y+float32(cs)*0.55
		switch k {
		case 0:
			vector.FillCircle(atlas, cx, cy, r, color.White, true)
		case 1:
			vector.FillCircle(atlas, cx-r*1.4, cy, r*0.8, color.White, true)
			vector.FillCircle(atlas, cx+r*1.4, cy-r, r*0.8, color.White, true)
		case 2:
			vector.StrokeLine(atlas, cx-r*1.6, cy, cx+r*1.6, cy, w, color.White, true)
			vector.StrokeLine(atlas, cx, cy-r*1.6, cx, cy+r*1.6, w, color.White, true)
		default:
			for p := 0; p < 5; p++ {
				a := float64(p) * 2 * math.Pi / 5
				vector.FillCircle(atlas, cx+float32(math.Cos(a))*r*1.3, cy+float32(math.Sin(a))*r*1.3, r*0.7, color.White, true)
			}
		}
	}
}
