package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"math"
	"strings"
)

const rightW = 300.0

var faceCache = map[[2]int]*text.GoTextFace{}

func face(size float64, bold bool) *text.GoTextFace {
	k := [2]int{int(size * 10), 0}
	src := monoSrc
	if bold {
		k[1], src = 1, monoBoldSrc
	}
	if f, ok := faceCache[k]; ok {
		return f
	}
	f := &text.GoTextFace{Source: src, Size: size}
	faceCache[k] = f
	return f
}

// txt escreve com o topo do texto em y.
func (g *Game) txt(dst *ebiten.Image, s string, x, y float64, c color.Color, size float64, bold bool, align int) {
	if s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = []text.Align{text.AlignStart, text.AlignCenter, text.AlignEnd}[align]
	op.GeoM.Translate(x*g.ui, y*g.ui)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face(size*g.ui, bold), op)
}

func (g *Game) textW(s string, size float64) float64 {
	return text.Advance(s, face(size*g.ui, false)) / g.ui
}

func (g *Game) rect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x*g.ui), float32(y*g.ui), float32(w*g.ui), float32(h*g.ui), c, false)
}

func (g *Game) stroke(dst *ebiten.Image, x, y, w, h float64, c color.Color, width float64) {
	vector.StrokeRect(dst, float32((x+0.5)*g.ui), float32((y+0.5)*g.ui), float32((w-1)*g.ui), float32((h-1)*g.ui), float32(width*g.ui), c, false)
}

func (g *Game) panel(dst *ebiten.Image, x, y, w, h float64, border color.Color) {
	g.rect(dst, x, y, w, h, color.Black)
	if border == nil {
		border = rgbaHex(0xcfcfcf)
	}
	g.stroke(dst, x, y, w, h, border, 1)
}

func (g *Game) hover(x, y, w, h float64) bool {
	return g.mouseX >= x && g.mouseX < x+w && g.mouseY >= y && g.mouseY < y+h
}

// stat é a barra de tracinhos, na cor de cada recurso: vida vermelha, mana azul, stamina verde.
func (g *Game) stat(dst *ebiten.Image, x, y float64, label string, v, maxv int, ch float64, full color.RGBA) {
	g.txt(dst, label, x, y, cText, 13, false, alignLeft)
	n := 0
	if maxv > 0 && v > 0 {
		n = clampi(int(math.Ceil(float64(v)*14/float64(maxv))), 0, 14)
	}
	for i := 0; i < 14; i++ {
		c := rgbaHex(0x2e2a2c)
		if i < n {
			c = full
		}
		g.txt(dst, "|", x+float64(3+i)*ch, y, c, 13, false, alignLeft)
	}
	g.txt(dst, fmt.Sprintf("%d/%d", v, maxv), x+18*ch, y, cText, 13, false, alignLeft)
}

// vignette pinta a borda da tela com a cor dada, sumindo para o centro.
func (g *Game) vignette(dst *ebiten.Image, c color.RGBA) {
	w, h := float64(g.W), float64(g.H)
	for k := 0; k < 12; k++ {
		t := float64(k) / 12
		a := uint8(float64(c.A) * (1 - t) * (1 - t))
		cc := color.RGBA{uint8(float64(c.R) * float64(a) / 255), uint8(float64(c.G) * float64(a) / 255), uint8(float64(c.B) * float64(a) / 255), a}
		in := t * math.Min(w, h) * 0.18
		vector.StrokeRect(dst, float32(in), float32(in), float32(w-2*in), float32(h-2*in), float32(math.Min(w, h)*0.018), cc, false)
	}
}

// colorScale converte uma cor para a escala de cor do Ebitengine.
func colorScale(c color.Color) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	return cs
}

// wrapPx quebra o texto em linhas que cabem em width (largura real da fonte).
func (g *Game) wrapPx(s string, size, width float64) []string {
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		try := w
		if line != "" {
			try = line + " " + w
		}
		if line != "" && g.textW(try, size)*1.04 > width {
			out = append(out, line)
			line = w
			continue
		}
		line = try
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// fit corta o texto com "…" para caber em width.
func (g *Game) fit(s string, size, width float64) string {
	if g.textW(s, size)*1.04 <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && g.textW(string(r)+"…", size)*1.04 > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// txtWrap escreve o texto quebrado em width e devolve a altura usada.
func (g *Game) txtWrap(dst *ebiten.Image, s string, x, y, width float64, c color.Color, size, lh float64) float64 {
	lines := g.wrapPx(s, size, width)
	for i, l := range lines {
		g.txt(dst, l, x, y+float64(i)*lh, c, size, false, alignLeft)
	}
	return float64(len(lines)) * lh
}

// keyTile desenha a telha de uma tecla (dourada, texto escuro) e devolve a
// largura usada.
func (g *Game) keyTile(dst *ebiten.Image, k string, x, y float64) float64 {
	w := math.Max(18, g.textW(k, 11)*1.05+10)
	g.rect(dst, x, y, w, 16, color.RGBA{214, 176, 80, 255})
	g.stroke(dst, x, y, w, 16, color.RGBA{60, 40, 10, 255}, 1)
	g.txt(dst, k, x+w/2, y+1, color.RGBA{24, 16, 6, 255}, 11, true, alignCenter)
	return w
}

// bullets escreve uma lista com marcador, quebrando cada item na largura, e
// devolve a altura usada.
func (g *Game) bullets(dst *ebiten.Image, items []string, x, y, width float64, c color.Color, size, lh float64) float64 {
	y0 := y
	for _, it := range items {
		g.txt(dst, "•", x, y, cGold, size, true, alignLeft)
		y += g.txtWrap(dst, it, x+12, y, width-12, c, size, lh) + 3
	}
	return y - y0
}
