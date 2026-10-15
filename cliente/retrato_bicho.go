package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Retrato das criaturas no bestiário: o próprio desenho do jogo (o mesmo
// drawBody/drawBeast dos inimigos), feito numa grade pequena e guardado numa
// imagem por criatura. Para isso o quadro do mundo (g.buf, g.x0...) é trocado
// por um buffer pequeno enquanto o bicho é desenhado, e uma terceira instância
// do shader (com atlas do tamanho da célula do retrato) pinta a imagem.

type creatureKey struct {
	kind  int
	steed bool
	size  int
}

func (g *Game) creatureImage(kind int, steed bool, size int) *ebiten.Image {
	k := creatureKey{kind, steed, size}
	if img, ok := g.creatures[k]; ok {
		return img
	}
	if g.creatures == nil {
		g.creatures = map[creatureKey]*ebiten.Image{}
	}
	if g.g == nil || g.m == nil {
		return nil
	}
	var st *bodyStyle
	tall := 30.0
	if !steed {
		st = &mobStyles[clampi(kind, 0, len(mobStyles)-1)]
		tall = 30 * st.sc()
		if st.beast > 0 {
			tall = (st.bh + 2*st.br + 6) * st.sc()
		}
	} else {
		ss := &steedStyles[clampi(kind, 0, len(steedStyles)-1)]
		tall = (ss.bh + 2*ss.br + 6 + ss.neck) * ss.sc()
	}
	cs := clampi(int(float64(size)/(tall+2)), 2, 8)
	cols, rows := size/cs, size/cs
	buf := make([]byte, cols*rows*4)
	for i := 0; i < cols*rows; i++ {
		buf[i*4], buf[i*4+1], buf[i*4+2], buf[i*4+3] = 14, 12, 20, ' '
	}
	// Troca o quadro do mundo pelo do retrato e desenha o bicho de perfil.
	sb, sc, sr, sx0, sy0 := g.buf, g.cols, g.rows, g.x0, g.y0
	ox, oy := 4, 4 // canto do mapa (água): nada "por cima" esconde o desenho
	g.buf, g.cols, g.rows, g.x0, g.y0 = buf, cols, rows, ox, oy
	fx, fy := ox+cols/2, oy+rows-max(2, rows/10)
	b := bodyDraw{style: st, x: fx, y: fy, T: 1, hf: 1, face: 1, tint: white, lit: true}
	switch {
	case steed:
		ph := 0.0
		g.drawSteed(kind, fx, fy, 1, &ph, 1)
	case st.beast > 0:
		g.drawBeast(b)
	default:
		g.drawBody(b)
	}
	g.buf, g.cols, g.rows, g.x0, g.y0 = sb, sc, sr, sx0, sy0
	if g.portG == nil {
		cp := *g.g
		cp.cells, cp.cellW, cp.cellH, cp.atlas, cp.atlasCell = nil, 0, 0, nil, 0
		g.portG = &cp
	}
	img := ebiten.NewImage(cols*cs, rows*cs)
	g.portG.buildAtlas(cs)
	g.portG.draw(img, buf, cols, rows, 0, 0, cs, 0.4)
	g.creatures[k] = img
	return img
}

// drawCreature põe o retrato em (x, y), num quadrinho de lado s (unidades da interface).
func (g *Game) drawCreature(dst *ebiten.Image, kind int, steed bool, x, y, s float64) {
	g.rect(dst, x, y, s, s, color.RGBA{14, 12, 20, 255})
	g.stroke(dst, x, y, s, s, rgbaHex(0x5a4a2a), 1)
	img := g.creatureImage(kind, steed, int(s*g.ui))
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x*g.ui+(s*g.ui-float64(img.Bounds().Dx()))/2, y*g.ui+(s*g.ui-float64(img.Bounds().Dy()))/2)
	dst.DrawImage(img, op)
}
