package main

import (
	"math"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Zoom do mapa grande: a roda aproxima e afasta no ponto do cursor (até 8x),
// + e − também, 0 volta ao mapa inteiro; arrastar com o botão direito move a
// vista. Ao abrir, a vista começa no mapa inteiro.

// mapRect guarda onde o mapa grande foi desenhado (para o zoom no cursor).
type mapRect struct{ x, y, w, h float64 }

func (g *Game) zoomMap(wheel float64) {
	if wheel == 0 {
		return
	}
	old := math.Max(1, g.mapZoom)
	nz := clamp(old*math.Pow(1.25, wheel), 1, 8)
	r := g.mapArea
	if r.w > 0 && g.hover(r.x, r.y, r.w, r.h) {
		// mantém parado o ponto debaixo do cursor
		fx, fy := (g.mouseX-r.x)/r.w-0.5, (g.mouseY-r.y)/r.h-0.5
		g.mapCX += fx * g.mapFullW * (1/old - 1/nz)
		g.mapCY += fy * g.mapFullH * (1/old - 1/nz)
	}
	g.mapZoom = nz
}

// bigMapView devolve a parte do mapa a mostrar com o zoom e trata as teclas
// e o arrasto do mapa grande.
func (g *Game) bigMapView(full zoneRect, me *proto.Player, mx, my, mw, mh float64) zoneRect {
	g.mapArea = mapRect{mx, my, mw, mh}
	g.mapFullW, g.mapFullH = float64(full.w), float64(full.h)
	if g.mapZoom < 1 || g.mapZone != full.name {
		g.mapZoom, g.mapZone = 1, full.name
		g.mapCX, g.mapCY = float64(full.x)+float64(full.w)/2, float64(full.y)+float64(full.h)/2
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEqual), inpututil.IsKeyJustPressed(ebiten.KeyKPAdd):
		g.zoomMap(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyMinus), inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract):
		g.zoomMap(-1)
	case inpututil.IsKeyJustPressed(ebiten.Key0), inpututil.IsKeyJustPressed(ebiten.KeyKP0):
		g.mapZoom = 1
	}
	// Arrasto com o botão direito.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if g.mapDrag == nil {
			g.mapDrag = &[2]float64{g.mouseX, g.mouseY}
		} else {
			dx, dy := g.mouseX-g.mapDrag[0], g.mouseY-g.mapDrag[1]
			g.mapCX -= dx / mw * g.mapFullW / g.mapZoom
			g.mapCY -= dy / mh * g.mapFullH / g.mapZoom
			*g.mapDrag = [2]float64{g.mouseX, g.mouseY}
		}
	} else {
		g.mapDrag = nil
	}
	vw, vh := float64(full.w)/g.mapZoom, float64(full.h)/g.mapZoom
	g.mapCX = clamp(g.mapCX, float64(full.x)+vw/2, float64(full.x+full.w)-vw/2)
	g.mapCY = clamp(g.mapCY, float64(full.y)+vh/2, float64(full.y+full.h)-vh/2)
	// A imagem do minimapa é de 1 pixel a cada mm células: a vista anda nessa grade.
	x0 := int(g.mapCX-vw/2) / mm * mm
	y0 := int(g.mapCY-vh/2) / mm * mm
	w := max(mm*4, int(vw)/mm*mm)
	h := max(mm*4, int(vh)/mm*mm)
	return zoneRect{full.name, x0, y0, w, h}
}
