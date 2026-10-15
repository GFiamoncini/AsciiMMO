package main

import (
	"math"
	"strings"

	"asciimmo/proto"
)

// Leituras: tudo que tem texto no mapa (placas, postes de direção, pedras
// rúnicas, inscrições) se registra aqui ao ser construído, e o cliente abre
// numa janela com E. No mapa as letras são células minúsculas; a janela é
// onde se lê de verdade.

func (wd *World) addRead(title, text string, x, y int) {
	wd.reads = append(wd.reads, proto.Read{Title: title, Text: text, X: x, Y: y})
}

// dirArrow é a seta do rumo a (ângulo em radianos, y para baixo).
func dirArrow(a float64) string {
	s, c := math.Sin(a), math.Cos(a)
	switch {
	case math.Abs(s) > 0.8 && s < 0:
		return "↑"
	case math.Abs(s) > 0.8:
		return "↓"
	case c < 0:
		return "←"
	}
	return "→"
}

func signpostText(dests []string, angs []float64) string {
	var lines []string
	for i, d := range dests {
		lines = append(lines, dirArrow(angs[i])+" "+d)
	}
	return strings.Join(lines, "\n")
}
