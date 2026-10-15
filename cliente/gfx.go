package main

import (
	"bytes"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
)

// A tela é uma grade de células: cada uma tem uma cor e um caractere. A grade
// sobe para a placa de vídeo como uma imagem (RGB = cor, A = código do
// caractere) e o shader desenha cada pixel buscando o caractere no atlas.
const gridShader = `//kage:unit pixels

package main

var View vec3 // deslocamento x e y em pixels e lado da célula
var Grid vec2 // colunas e linhas
var Bloom float // força do brilho (0 desliga)
var Tint float  // quanto da cor da célula pinta o fundo entre os glifos

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	bg := vec3(0.035, 0.035, 0.06)
	p := dstPos.xy + View.xy
	c := floor(p / View.z)
	if c.x < 0 || c.y < 0 || c.x >= Grid.x || c.y >= Grid.y {
		return vec4(bg, 1)
	}
	f := (p - c*View.z) / View.z
	d := imageSrc0At(imageSrc0Origin() + c + vec2(0.5))
	code := floor(d.a*255 + 0.5)
	g := vec2(mod(code, 16), floor(code/16))
	a := imageSrc1At(imageSrc1Origin() + (g+f)*View.z).a
	// O vão entre os traços não é preto: leva um tom escuro da cor da célula,
	// e a imagem fica "pintada" sem perder o aspecto de texto.
	col := mix(bg+d.rgb*Tint, d.rgb, a)
	// Brilho: o que é claro (fogo, magia, luz) solta um halo nas células em
	// volta, como a luz vazando no vídeo de referência.
	if Bloom > 0 {
		glow := vec3(0)
		pc := p / View.z
		for oy := -2; oy <= 2; oy++ {
			for ox := -2; ox <= 2; ox++ {
				nc := c + vec2(float(ox), float(oy))
				if nc.x < 0 || nc.y < 0 || nc.x >= Grid.x || nc.y >= Grid.y {
					continue
				}
				n := imageSrc0At(imageSrc0Origin() + nc + vec2(0.5))
				lum := dot(n.rgb, vec3(0.3, 0.59, 0.11))
				// Só o que é muito claro e quente ou saturado brilha (fogo, magia, luz), não a folhagem.
				sat := max(n.r, max(n.g, n.b)) - min(n.r, min(n.g, n.b))
				if lum > 0.62 && (sat > 0.35 || lum > 0.85) {
					dd := pc - (nc + vec2(0.5))
					glow += n.rgb * (lum - 0.62) * exp(-dot(dd, dd)/1.6)
				}
			}
		}
		col += glow * Bloom
	}
	return vec4(col, 1)
}
`

var (
	monoSrc, monoBoldSrc *text.GoTextFaceSource
)

func init() {
	var err error
	if monoSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF)); err != nil {
		panic(err)
	}
	if monoBoldSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gomonobold.TTF)); err != nil {
		panic(err)
	}
}

type gpu struct {
	shader       *ebiten.Shader
	cells        *ebiten.Image
	cellW, cellH int
	atlas        *ebiten.Image
	atlasCell    int
	verts        []ebiten.Vertex
	idx          []uint16
}

func newGPU() *gpu {
	s, err := ebiten.NewShader([]byte(gridShader))
	if err != nil {
		panic(err)
	}
	return &gpu{shader: s, idx: []uint16{0, 1, 2, 1, 3, 2}, verts: make([]ebiten.Vertex, 4)}
}

// buildAtlas redesenha os caracteres no tamanho exato da célula a cada passo
// de zoom, para saírem nítidos sem filtro. Os códigos 33–126 são o ASCII da
// fonte; de 128 em diante vêm glifos desenhados aqui: lâminas de capim em 12
// inclinações (curtas e longas) e pontinhos de flor (glyphs.go).
func (g *gpu) buildAtlas(cs int) {
	if cs == g.atlasCell {
		return
	}
	g.atlasCell = cs
	if g.atlas != nil {
		g.atlas.Deallocate()
	}
	g.atlas = ebiten.NewImageWithOptions(imageRect(16*cs, 16*cs), &ebiten.NewImageOptions{Unmanaged: true})
	face := &text.GoTextFace{Source: monoBoldSrc, Size: float64(max(6, int(float64(cs)*1.2+0.5)))}
	for code := 33; code < 127; code++ {
		x, y := code&15*cs, code>>4*cs
		cell := g.atlas.SubImage(imageRect2(x, y, x+cs, y+cs)).(*ebiten.Image)
		op := &text.DrawOptions{}
		op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter
		op.GeoM.Translate(float64(x)+float64(cs)/2, float64(y)+float64(cs)/2+float64(cs)*0.03)
		text.Draw(cell, string(rune(code)), face, op)
	}
	drawCustomGlyphs(g.atlas, cs)
}

// draw sobe a grade e pinta a tela inteira com o shader.
func (g *gpu) draw(screen *ebiten.Image, buf []byte, cols, rows int, offX, offY float64, cs int, bloom float64) {
	if g.cells == nil || g.cellW != cols || g.cellH != rows {
		if g.cells != nil {
			g.cells.Deallocate()
		}
		g.cells = ebiten.NewImageWithOptions(imageRect(cols, rows), &ebiten.NewImageOptions{Unmanaged: true})
		g.cellW, g.cellH = cols, rows
	}
	g.cells.WritePixels(buf)
	w, h := float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())
	for i, p := range [][2]float32{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		g.verts[i] = ebiten.Vertex{DstX: p[0], DstY: p[1], SrcX: p[0], SrcY: p[1], ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
	}
	op := &ebiten.DrawTrianglesShaderOptions{
		Uniforms: map[string]any{"View": []float32{float32(offX), float32(offY), float32(cs)}, "Grid": []float32{float32(cols), float32(rows)}, "Bloom": float32(bloom), "Tint": float32(0.13)},
		Images:   [4]*ebiten.Image{g.cells, g.atlas},
	}
	screen.DrawTrianglesShader(g.verts, g.idx, g.shader, op)
}
