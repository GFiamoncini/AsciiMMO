package main

import (
	"math"
	"runtime"
	"sync"
)

// ---------- profundidade do ambiente (2.5D) ----------
// O mapa é plano: cada célula só tem caractere e cor. Para dar volume, tudo o
// que é alto (copa, telhado, parede, rocha) vira um campo de altura: a máscara
// do que é alto, borrada, sobe no meio das manchas e desce nas bordas. Desse
// relevo saem a normal e a luz, como no 3D dos personagens: copas abauladas,
// telhado com cumeeira, rocha com face clara e face escura. No chão em volta,
// a mesma máscara escurece o pé das coisas (oclusão de contato), e de dia
// sombras de nuvens passam devagar.

type depthBuf struct {
	w, h              int
	mask, tmp, hs, hl []float32
}

// valueNoise é ruído suave (0..1) em grade, interpolado.
func valueNoise(fx, fy float64, seed int) float64 {
	ix, iy := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(ix), fy-float64(iy)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	v := func(a, b int) float64 { return float64(hash2(a, b, seed)&1023) / 1023 }
	top := v(ix, iy) + (v(ix+1, iy)-v(ix, iy))*tx
	bot := v(ix, iy+1) + (v(ix+1, iy+1)-v(ix, iy+1))*tx
	return top + (bot-top)*ty
}

// depthPass aplica volume, oclusão e sombra de nuvem ao terreno já pintado.
// (lx, ly) é de onde vem a luz no plano da tela (x à direita, y para baixo).
func (g *Game) depthPass(buf []byte, cols, rows, x0, y0 int, lx, ly, strength, clouds, haze, T float64) {
	const R = 12 // margem: o borrão largo precisa ver fora da tela
	m := g.m
	d := &g.depth
	w, h := cols+2*R, rows+2*R
	if d.w != w || d.h != h {
		d.w, d.h = w, h
		d.mask, d.tmp, d.hs, d.hl = make([]float32, w*h), make([]float32, w*h), make([]float32, w*h), make([]float32, w*h)
	}
	for y := 0; y < h; y++ {
		wy := y0 + y - R
		for x := 0; x < w; x++ {
			wx, v := x0+x-R, float32(0)
			if wx >= 0 && wy >= 0 && wx < m.w && wy < m.h {
				j := wy*m.w + wx
				if f := m.flags[j]; f&2 != 0 || f&1 != 0 && f&4 == 0 && m.glyph[j] != ' ' {
					v = 1
				}
			}
			d.mask[y*w+x] = v
		}
	}
	// Dois relevos: um fino (raio 2), para o pé das coisas e o detalhe, e um
	// largo (raio 5 duas vezes, quase gaussiano), para o volume das manchas
	// grandes: copa inteira abaulada, telhado com cumeeira.
	boxBlur(d.mask, d.tmp, w, h, 2, true)
	boxBlur(d.tmp, d.hs, w, h, 2, false)
	boxBlur(d.mask, d.tmp, w, h, 5, true)
	boxBlur(d.tmp, d.hl, w, h, 5, false)
	boxBlur(d.hl, d.tmp, w, h, 5, true)
	boxBlur(d.tmp, d.hl, w, h, 5, false)
	hs, hl := d.hs, d.hl

	ln := math.Hypot(lx, ly)
	if ln > 0 {
		lx, ly = lx/ln, ly/ln
	}
	lz := 1.0
	n := math.Sqrt(lx*lx + ly*ly + lz*lz)
	lx, ly, lz = lx/n, ly/n, lz/n
	// Sombras de nuvem numa grade grossa (o ruído é suave; ver fogGrid).
	var cg *fogGrid
	if clouds > 0 {
		cg = newGridOf(x0, y0, cols, rows, func(wx, wy int) float64 {
			return valueNoise(float64(wx)/70+T*0.035, float64(wy)/42+T*0.012, 4242)
		})
	}
	parallelRows(rows, func(sy int) {
		for sx := 0; sx < cols; sx++ {
			wx, wy := x0+sx, y0+sy
			if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h {
				continue
			}
			i := (sy+R)*w + sx + R
			o := (sy*cols + sx) * 4
			f := 1.0
			if m.flags[wy*m.w+wx]&64 != 0 {
				continue // o que brilha não ganha sombra de volume nem neblina
			}
			if d.mask[i] > 0 {
				// Normal do relevo pela diferença central dos dois borrões.
				gx := float64(hl[i+1]-hl[i-1])*9 + float64(hs[i+1]-hs[i-1])*1.5
				gy := float64(hl[i+w]-hl[i-w])*9 + float64(hs[i+w]-hs[i-w])*1.5
				nx, ny := -gx, -gy
				k := math.Sqrt(nx*nx + ny*ny + 1)
				diff := (nx*lx + ny*ly + lz) / k
				f = (0.42 + 0.72*diff) * (0.8 + 0.35*float64(hl[i]))
			} else if hs[i] > 0 {
				f = 1 - 0.5*float64(hs[i]) // pé das coisas: o chão encostado escurece
			}
			if clouds > 0 && m.flags[wy*m.w+wx]&(2|4) == 0 { // nuvem só no chão: copa e telhado ficam com o volume
				c := cg.at(sx, sy)
				f *= 1 - clouds*clamp((c-0.55)*4, 0, 1)
			}
			f = 1 + (f-1)*strength
			// Neblina na altura: o alto das torres e das copas some aos poucos
			// no escuro, o que dá tamanho às coisas.
			fade := 0.0
			if haze > 0 && m.flags[wy*m.w+wx]&2 != 0 {
				if base := g.baseAt(wx, wy); base > wy {
					fade = haze * clamp(float64(base-wy-14)/40, 0, 1)
				}
			}
			if f == 1 && fade == 0 {
				continue
			}
			f *= 1 - 0.45*fade
			k := 0.35 * fade
			buf[o] = cb(float64(buf[o])*f*(1-k) + 34*k)
			buf[o+1] = cb(float64(buf[o+1])*f*(1-k) + 38*k)
			buf[o+2] = cb(float64(buf[o+2])*f*(1-k) + 56*k)
		}
	})
}

// boxBlur é a média numa janela de raio r que desliza por linha (ou coluna):
// custo fixo por célula, qualquer que seja o raio.
func boxBlur(src, dst []float32, w, h, r int, horiz bool) {
	n, stride, lines, step := w, 1, h, w
	if !horiz {
		n, stride, lines, step = h, w, w, 1
	}
	inv := 1 / float32(2*r+1)
	parallelRows(lines, func(l int) {
		base := l * step
		at := func(k int) float32 { return src[base+clampi(k, 0, n-1)*stride] }
		s := float32(0)
		for k := -r; k <= r; k++ {
			s += at(k)
		}
		for k := 0; k < n; k++ {
			dst[base+k*stride] = s * inv
			s += at(k+r+1) - at(k-r)
		}
	})
}

// groundRing desenha o anel de seleção no chão, como em Diablo e Age of
// Empires: uma elipse em volta dos pés, com a metade da frente mais viva que a
// de trás (a de trás fica atrás do corpo, que é desenhado por cima).
func (g *Game) groundRing(cx, cy, rx, ry float64, c col, k float64) {
	for y := int(math.Floor(cy - ry - 1)); y <= int(math.Ceil(cy+ry+1)); y++ {
		for x := int(math.Floor(cx - rx - 1)); x <= int(math.Ceil(cx+rx+1)); x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			e := math.Sqrt(dx*dx/(rx*rx) + dy*dy/(ry*ry))
			if e == 0 {
				continue
			}
			gx, gy := dx/(rx*rx)/e, dy/(ry*ry)/e
			gl := math.Hypot(gx, gy)
			if math.Abs((e-1)/gl) > 0.55 {
				continue
			}
			// O caractere segue a tangente da elipse.
			tx, ty := -gy, gx
			a := math.Atan2(ty, tx)
			if a < 0 {
				a += math.Pi
			}
			// '_' sai cortado e '-' some em célula pequena: a frente usa '='.
			ch := byte('-')
			switch {
			case a > math.Pi*3/8 && a < math.Pi*5/8:
				ch = '('
				if dx > 0 {
					ch = ')'
				}
			case a >= math.Pi/8 && a <= math.Pi*3/8:
				ch = '\\'
			case a >= math.Pi*5/8 && a <= math.Pi*7/8:
				ch = '/'
			}
			if dy > 0.3 && ch == '-' {
				ch = '='
			}
			kk := k
			if dy < 0 {
				kk *= 0.55 // metade de trás, mais apagada
			}
			sx, sy := x-g.x0, y-g.y0
			if sx < 0 || sy < 0 || sx >= g.cols || sy >= g.rows {
				continue
			}
			o := (sy*g.cols + sx) * 4
			g.buf[o] = cb(float64(g.buf[o])*(1-kk) + c[0]*kk)
			g.buf[o+1] = cb(float64(g.buf[o+1])*(1-kk) + c[1]*kk)
			g.buf[o+2] = cb(float64(g.buf[o+2])*(1-kk) + c[2]*kk)
			g.buf[o+3] = ch
		}
	}
}

// parallelRows roda fn(y) para y de 0 a n-1, em linhas intercaladas entre os
// núcleos. Só serve para laços em que cada linha escreve na sua parte.
func parallelRows(n int, fn func(y int)) {
	workers := min(runtime.NumCPU(), max(1, n/16))
	if workers <= 1 {
		for y := 0; y < n; y++ {
			fn(y)
		}
		return
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for y := w; y < n; y += workers {
				fn(y)
			}
		}(w)
	}
	wg.Wait()
}

// castShadows projeta no chão a sombra de tudo o que é alto, do tamanho da
// altura de cada célula acima da sua base: uma torre de 60 linhas deita uma
// sombra comprida, um caixote quase nenhuma. (sx, sy) é para onde a sombra
// anda por linha de altura; k é a força (0 de noite). Cada coluna deita uma
// faixa inteira, do pé até a ponta (antes só a ponta era marcada e a sombra
// saía pontilhada), clareando um pouco para a ponta. Escreve em g.shadeF.
func (g *Game) castShadows(cols, rows, x0, y0 int, sx, sy, k float64) {
	if k <= 0.02 {
		return
	}
	m := g.m
	const margin = 70 // estruturas fora da tela também fazem sombra dentro
	ln := math.Hypot(sx, sy)
	for wy := y0 - margin; wy < y0+rows; wy++ {
		for wx := x0 - margin; wx < x0+cols+margin; wx++ {
			if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h || m.flags[wy*m.w+wx]&2 == 0 {
				continue
			}
			base := g.baseAt(wx, wy)
			if base < 0 {
				base = wy + 2
			}
			h := math.Min(float64(base-wy), 48)
			if h < 2 {
				continue
			}
			steps := int(math.Ceil(h * ln))
			for t := 0; t <= steps; t++ {
				u := float64(t) / float64(max(1, steps))
				x := int(math.Round(float64(wx)+sx*h*u)) - x0
				y := int(math.Round(float64(base)+sy*h*u)) - y0
				if x < 0 || y < 0 || x >= cols || y >= rows {
					continue
				}
				f := float32(1 - 0.5*k*(1-0.45*u))
				if i := y*cols + x; f < g.shadeF[i] {
					g.shadeF[i] = f
				}
			}
		}
	}
}

// applyShade escurece o chão pela máscara de sombra (só o chão: o que é alto e
// o que brilha ficam como estão).
func (g *Game) applyShade(buf []byte, cols, rows, x0, y0 int) {
	m := g.m
	parallelRows(rows, func(sy int) {
		for sx := 0; sx < cols; sx++ {
			i := sy*cols + sx
			f := float64(g.shadeF[i])
			if f >= 1 {
				continue
			}
			wx, wy := x0+sx, y0+sy
			if wx < 0 || wy < 0 || wx >= m.w || wy >= m.h || m.flags[wy*m.w+wx]&(2|64) != 0 {
				continue
			}
			o := i * 4
			buf[o], buf[o+1], buf[o+2] = cb(float64(buf[o])*f), cb(float64(buf[o+1])*f), cb(float64(buf[o+2])*f*1.05)
		}
	})
}
