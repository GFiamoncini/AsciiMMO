package main

import (
	"image"
	"image/color"
	"math"
)

const tau = 2 * math.Pi

// col é uma cor RGB em 0..255, como float para as contas de luz.
type col [3]float64

var (
	white = col{255, 255, 255}
	red   = col{255, 60, 50}
)

func hexc(c int) col { return col{float64(c >> 16 & 255), float64(c >> 8 & 255), float64(c & 255)} }

func mix3(a, b col, t float64) col {
	return col{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

func (c col) rgba() color.RGBA {
	return color.RGBA{uint8(clamp(c[0], 0, 255)), uint8(clamp(c[1], 0, 255)), uint8(clamp(c[2], 0, 255)), 255}
}

func rgbaHex(c int) color.RGBA { return hexc(c).rgba() }

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func clampi(v, lo, hi int) int { return max(lo, min(hi, v)) }

// hash2 é o mesmo do servidor e do cliente web: o ruído tem de bater.
func hash2(x, y, s int) uint32 {
	h := uint32(int32(x))*374761393 + uint32(int32(y))*668265263 + uint32(int32(s))*2246822519
	h = (h ^ h>>13) * 1274126177
	return h ^ h>>16
}

func ease(t float64) float64 { return t * t * (3 - 2*t) }

func imageRect(w, h int) image.Rectangle { return image.Rect(0, 0, w, h) }

func imageRect2(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

func round(v float64) int { return int(math.Round(v)) }

func sign(v float64) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// Do mais apagado ao mais cheio: o brilho do "pixel" escolhe o caractere.
const ramp = ".:;~=+caoqdb0O8Q&$#@"

func rampCode(l float64) byte {
	return ramp[clampi(int(l*float64(len(ramp))), 0, len(ramp)-1)]
}

func slope(a float64) byte {
	s, c := math.Sin(a), math.Cos(a)
	switch {
	case math.Abs(c) > 0.92:
		return '-'
	case math.Abs(s) > 0.92:
		return '|'
	case s*c > 0:
		return '\\'
	}
	return '/'
}

// wrap quebra o texto em linhas de até max caracteres.
func wrap(s string, maxc int) []string {
	var out []string
	line := ""
	for _, w := range splitWords(s) {
		if line != "" && len([]rune(line+" "+w)) > maxc {
			out = append(out, line)
			line = w
		} else if line == "" {
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

func splitWords(s string) []string {
	var out []string
	cur := []rune{}
	for _, r := range s {
		if r == ' ' {
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = cur[:0]
			}
			continue
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	return out
}
