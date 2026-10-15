package main

import (
	"image"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// savePNG grava a tela num arquivo (opção -print, usada para conferir o visual).
func savePNG(screen *ebiten.Image, path string) {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	if f, err := os.Create(path); err == nil {
		png.Encode(f, img)
		f.Close()
	}
}
