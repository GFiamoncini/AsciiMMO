package main

import (
	"math"
	"strconv"
	"unicode/utf8"
)

// noBG deixa o fundo padrão do terminal.
const noBG rgb = 1 << 24

type cell struct {
	ch     rune
	fg, bg rgb
}

// frame é a tela de uma sessão: uma grade de células coloridas que vira
// uma única string com escapes truecolor.
type frame struct {
	w, h  int
	cells []cell
}

func newFrame(w, h int) *frame {
	f := &frame{w: w, h: h, cells: make([]cell, w*h)}
	for i := range f.cells {
		f.cells[i] = cell{' ', 0xffffff, noBG}
	}
	return f
}

func (f *frame) in(x, y int) bool { return x >= 0 && y >= 0 && x < f.w && y < f.h }

func (f *frame) set(x, y int, ch rune, fg, bg rgb) {
	if f.in(x, y) {
		f.cells[y*f.w+x] = cell{ch, fg, bg}
	}
}

// glyph troca caractere e cor, mantendo o fundo que já estava.
func (f *frame) glyph(x, y int, ch rune, fg rgb) {
	if f.in(x, y) {
		c := &f.cells[y*f.w+x]
		c.ch, c.fg = ch, fg
	}
}

func (f *frame) text(x, y int, s string, fg, bg rgb) {
	for _, r := range s {
		f.set(x, y, r, fg, bg)
		x++
	}
}

func (f *frame) textRight(x, y int, s string, fg, bg rgb) {
	f.text(x-utf8.RuneCountInString(s), y, s, fg, bg)
}

func (f *frame) fill(x, y, w, h int, bg rgb) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			f.set(xx, yy, ' ', 0xffffff, bg)
		}
	}
}

// box desenha um painel preto com moldura fina.
func (f *frame) box(x, y, w, h int, border rgb) {
	f.fill(x, y, w, h, 0)
	for i := 1; i < w-1; i++ {
		f.set(x+i, y, '─', border, 0)
		f.set(x+i, y+h-1, '─', border, 0)
	}
	for i := 1; i < h-1; i++ {
		f.set(x, y+i, '│', border, 0)
		f.set(x+w-1, y+i, '│', border, 0)
	}
	f.set(x, y, '┌', border, 0)
	f.set(x+w-1, y, '┐', border, 0)
	f.set(x, y+h-1, '└', border, 0)
	f.set(x+w-1, y+h-1, '┘', border, 0)
}

// ticks é a barra feita de tracinhos, como no HUD de referência.
func (f *frame) ticks(x, y, w, cur, max int, full, empty rgb) {
	n := 0
	if max > 0 && cur > 0 {
		n = clamp((cur*w+max-1)/max, 0, w)
	}
	for i := 0; i < w; i++ {
		fg := empty
		if i < n {
			fg = full
		}
		f.set(x+i, y, '|', fg, 0)
	}
}

func (f *frame) bar(x, y, w, cur, max int, full, empty rgb) {
	n := 0
	if max > 0 && cur > 0 {
		n = clamp((cur*w+max-1)/max, 0, w)
	}
	for i := 0; i < w; i++ {
		bg := empty
		if i < n {
			bg = full
		}
		f.set(x+i, y, ' ', 0xffffff, bg)
	}
}

// ring desenha o trecho [a0, a1] de um círculo de raio r (em pixels), com o
// caractere acompanhando a inclinação da curva.
func (f *frame) ring(cx, cy int, r, a0, a1 float64, fg rgb) {
	n := int(math.Max(12, r*10*(a1-a0)/(2*math.Pi)))
	for i := 0; i <= n; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(n)
		sin, cos := math.Sincos(a)
		ch := '\\'
		switch {
		case math.Abs(cos) > 0.92:
			ch = '|'
		case math.Abs(sin) > 0.92:
			ch = '-'
		case sin*cos > 0:
			ch = '/'
		}
		f.glyph(cx+int(math.Round(2*r*cos)), cy+int(math.Round(r*sin)), ch, fg)
	}
}

func dim(c rgb, k float64) rgb {
	// Cada canal para em 255: sem isso, clarear uma cor forte (k > 1) estoura
	// o vermelho para o bit de cima e o amarelo de uma janela vira verde.
	r := math.Min(255, float64(c>>16&0xff)*k)
	g := math.Min(255, float64(c>>8&0xff)*k)
	b := math.Min(255, float64(c&0xff)*k)
	return rgb(r)<<16 | rgb(g)<<8 | rgb(b)
}

func appendColor(b []byte, prefix string, c rgb) []byte {
	b = append(b, prefix...)
	b = strconv.AppendUint(b, uint64(c>>16&0xff), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(c>>8&0xff), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(c&0xff), 10)
	return append(b, 'm')
}

// String só emite escape quando a cor muda, então trechos da mesma cor
// custam um byte por célula.
func (f *frame) String() string {
	const unset rgb = 1 << 25
	b := make([]byte, 0, f.w*f.h*8)
	for y := 0; y < f.h; y++ {
		fg, bg := unset, unset
		for _, c := range f.cells[y*f.w : (y+1)*f.w] {
			if c.bg != bg {
				bg = c.bg
				if bg == noBG {
					b = append(b, "\x1b[49m"...)
				} else {
					b = appendColor(b, "\x1b[48;2;", bg)
				}
			}
			if c.fg != fg && c.ch != ' ' {
				fg = c.fg
				b = appendColor(b, "\x1b[38;2;", fg)
			}
			b = utf8.AppendRune(b, c.ch)
		}
		b = append(b, "\x1b[0m"...)
		if y < f.h-1 {
			b = append(b, '\n')
		}
	}
	return string(b)
}
