package main

import (
	"math"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

// Abertura no estilo do jogo: cada tela é uma cena montada a cada quadro numa
// grade de células (caractere + cor), desenhada pelo mesmo shader do mundo
// (atlas de glifos, fundo tingido, brilho). Três cenas: Cinzalva ardendo,
// Tiamath desperta entre relâmpagos, o vale de cinzas à noite com a fogueira e
// o Eremita. A narração fala "a língua velha" (inventada), palavra a palavra,
// no ritmo da legenda em português.

// sceneBuf é a grade da cena: 4 bytes por célula (r, g, b, glifo).
type sceneBuf struct {
	cols, rows int
	b          []byte
}

func (s *sceneBuf) set(x, y int, ch byte, c col) {
	if x < 0 || y < 0 || x >= s.cols || y >= s.rows {
		return
	}
	o := (y*s.cols + x) * 4
	s.b[o], s.b[o+1], s.b[o+2], s.b[o+3] = cb(c[0]), cb(c[1]), cb(c[2]), ch
}

func (s *sceneBuf) get(x, y int) (col, byte) {
	if x < 0 || y < 0 || x >= s.cols || y >= s.rows {
		return col{}, ' '
	}
	o := (y*s.cols + x) * 4
	return col{float64(s.b[o]), float64(s.b[o+1]), float64(s.b[o+2])}, s.b[o+3]
}

// light clareia em volta de (x, y) com a cor c (para fogo e olhos).
func (s *sceneBuf) light(x, y, r float64, c col, k float64) {
	for yy := int(y - r); yy <= int(y+r); yy++ {
		for xx := int(x - r*1.6); xx <= int(x+r*1.6); xx++ {
			d := math.Hypot((float64(xx)-x)/1.6, float64(yy)-y) / r
			if d >= 1 {
				continue
			}
			cc, ch := s.get(xx, yy)
			f := (1 - d) * (1 - d) * k
			if ch == ' ' {
				ch = ' '
			}
			s.set(xx, yy, ch, col{cc[0] + c[0]*f, cc[1] + c[1]*f, cc[2] + c[2]*f})
		}
	}
}

func noise1(x float64, seed int) float64 {
	i := math.Floor(x)
	f := x - i
	a := float64(hash2(int(i), 0, seed)&1023) / 1023
	b := float64(hash2(int(i)+1, 0, seed)&1023) / 1023
	f = f * f * (3 - 2*f)
	return a + (b-a)*f
}

// sceneFor monta a cena da página p no tempo t (segundos).
func sceneFor(s *sceneBuf, p int, t float64) (flash float64) {
	for i := range s.b {
		s.b[i] = 0
	}
	for i := 3; i < len(s.b); i += 4 {
		s.b[i] = ' '
	}
	W, H := s.cols, s.rows
	switch p {
	case 0: // Cinzalva ardendo
		for y := 0; y < H; y++ {
			k := float64(y) / float64(H)
			sky := mix3(col{14, 4, 6}, col{120, 30, 14}, math.Pow(k, 1.6))
			for x := 0; x < W; x++ {
				// fumaça correndo em faixas
				n := noise1(float64(x)/9+t*0.6+float64(y)*0.3, 3) * noise1(float64(y)/3+t*0.2, 4)
				if n > 0.42 && k < 0.7 {
					s.set(x, y, "~:.,"[hash2(x, y, 5)%4], mix3(sky, col{90, 70, 70}, n-0.3))
				} else if hash2(x, y, 9)%9 == 0 {
					s.set(x, y, '.', mix3(sky, col{60, 20, 10}, 0.5))
				} else {
					s.set(x, y, ' ', sky)
				}
			}
		}
		ground := H * 62 / 100
		x := 0
		for x < W { // casas: paredes, telhado de duas águas, janelas acesas
			hw := 6 + int(hash2(x, 1, 11)%8)
			hh := 5 + int(hash2(x, 2, 11)%7)
			top := ground - hh
			for xx := x; xx < x+hw && xx < W; xx++ {
				for yy := top; yy < ground; yy++ {
					c := mix3(col{70, 38, 30}, col{110, 60, 40}, float64(yy-top)/float64(hh))
					ch := "#H#="[hash2(xx, yy, 13)%4]
					if (yy-top)%3 == 1 && (xx-x)%3 == 1 && xx > x && xx < x+hw-1 {
						fl := 0.6 + 0.4*math.Sin(t*9+float64(xx*7+yy))
						ch, c = 'o', col{255 * fl, 150 * fl, 40 * fl}
					}
					s.set(xx, yy, ch, c)
				}
				roofH := hw / 2
				d := min(xx-x, x+hw-1-xx)
				for r := 0; r < min(d, roofH); r++ {
					s.set(xx, top-1-r, "/\\^"[hash2(xx, r, 14)%3], col{95, 45, 30})
				}
				// fogo no telhado
				if hash2(x, 3, 11)%3 != 0 {
					fh := int(3 + 5*noise1(float64(xx)*0.7+t*4, 7))
					for f := 0; f < fh; f++ {
						kk := float64(f) / float64(fh)
						fc := mix3(col{255, 240, 160}, col{200, 40, 10}, kk)
						s.set(xx, top-1-min(d, roofH)-f, "^*'\")("[(hash2(xx, f, int(t*14))%6)], fc)
					}
				}
			}
			x += hw + 1 + int(hash2(x, 4, 11)%3)
		}
		for yy := ground; yy < H; yy++ {
			for xx := 0; xx < W; xx++ {
				s.set(xx, yy, "_.,~"[hash2(xx, yy, 2)%4], mix3(col{90, 36, 20}, col{30, 12, 10}, float64(yy-ground)/float64(H-ground)))
			}
		}
		for i := 0; i < W/3; i++ { // brasas subindo
			fi := float64(i)
			k := frac(fi*0.137 + t*0.08*(0.5+frac(fi*0.71)))
			px := frac(fi*0.6180339) * float64(W)
			s.set(int(px+math.Sin(t*2+fi)*2), int(float64(ground)*(1-k)), "'.*"[i%3], col{255, 160 * (1 - k), 50})
		}
		for xx := 0; xx < W; xx += 9 {
			s.light(float64(xx), float64(ground-6), 9, col{120, 50, 10}, 0.9)
		}
	case 1: // Tiamath
		// relâmpago de tempos em tempos
		cyc := math.Mod(t, 4.2)
		if cyc < 0.12 || cyc > 0.25 && cyc < 0.32 {
			flash = 1
		}
		for y := 0; y < H; y++ {
			k := float64(y) / float64(H)
			sky := mix3(col{8, 8, 16}, col{40, 20, 30}, k)
			sky = mix3(sky, col{170, 170, 210}, flash*0.5)
			for x := 0; x < W; x++ {
				n := noise1(float64(x)/14+t*0.3, 21) * noise1(float64(y)/4-t*0.1, 22)
				if n > 0.35 && k < 0.5 {
					s.set(x, y, "~-"[hash2(x, y, 3)%2], mix3(sky, col{70, 70, 90}, n))
				} else {
					s.set(x, y, ' ', sky)
				}
			}
		}
		// montanha
		for x := 0; x < W; x++ {
			top := int(float64(H)*0.5 - 10*noise1(float64(x)/11, 30) - 6*math.Sin(float64(x)/float64(W)*math.Pi))
			for y := top; y < H; y++ {
				s.set(x, y, "#%&^"[hash2(x, y, 31)%4], mix3(col{48, 36, 44}, col{150, 140, 170}, flash*0.5))
			}
		}
		// corpo e os cinco pescoços
		bx, by := float64(W)/2, float64(H)*0.62
		body := col{170, 50, 34}
		for a := 0.0; a < math.Pi*2; a += 0.05 {
			for r := 0.0; r < 9; r++ {
				s.set(int(bx+math.Cos(a)*r*1.8), int(by+math.Sin(a)*r*0.7), "&%#"[int(r)%3], mix3(body, col{30, 8, 6}, r/9))
			}
		}
		for n := 0; n < 5; n++ {
			base := -math.Pi/2 + (float64(n)-2)*0.45
			hx, hy := bx, by-4
			for seg := 0; seg < 26; seg++ {
				sw := math.Sin(t*1.4+float64(n)*1.3+float64(seg)*0.25) * 0.05 * float64(seg)
				a := base + sw
				hx += math.Cos(a) * 1.7
				hy += math.Sin(a) * 1.05
				for w := -1; w <= 1; w++ {
					s.set(int(hx)+w, int(hy), "%#&"[(seg+w+3)%3], mix3(body, col{200, 60, 30}, float64(seg)/40))
				}
			}
			// cabeça e olhos
			for w := -2; w <= 2; w++ {
				s.set(int(hx)+w, int(hy)-1, '#', col{110, 30, 20})
				s.set(int(hx)+w, int(hy), '#', col{80, 20, 14})
			}
			eye := 0.6 + 0.4*math.Sin(t*3+float64(n))
			s.set(int(hx)-1, int(hy)-1, 'o', col{255 * eye, 220 * eye, 60})
			s.set(int(hx)+1, int(hy)-1, 'o', col{255 * eye, 220 * eye, 60})
			s.light(hx, hy-1, 4, col{120, 90, 20}, eye)
			if n == int(t/3)%5 && math.Mod(t, 3) < 1.2 { // uma cabeça cospe fogo
				for f := 1; f < 14; f++ {
					s.set(int(hx)+f*(n-2)/2, int(hy)+f/2, "*^'"[f%3], mix3(col{255, 230, 120}, col{220, 50, 10}, float64(f)/14))
				}
				s.light(hx, hy+4, 10, col{160, 60, 10}, 1)
			}
		}
		if flash > 0 { // o raio
			x := W/2 + int(hash2(int(t/4.2), 1, 7)%uint32(W/2)) - W/4
			for y := 0; y < H/2; y++ {
				x += int(hash2(x, y, 8)%3) - 1
				s.set(x, y, '|', col{255, 255, 255})
			}
		}
	case 2: // o vale à noite
		for y := 0; y < H; y++ {
			k := float64(y) / float64(H)
			sky := mix3(col{6, 8, 20}, col{20, 22, 40}, k)
			for x := 0; x < W; x++ {
				if hash2(x, y, 41)%37 == 0 && k < 0.5 {
					tw := 0.4 + 0.6*math.Pow(0.5+0.5*math.Sin(t*(1+float64(hash2(x, y, 3)%5)*0.4)+float64(x)), 3)
					s.set(x, y, ".*+"[hash2(x, y, 4)%3], col{200 * tw, 210 * tw, 255 * tw})
				} else {
					s.set(x, y, ' ', sky)
				}
			}
		}
		// lua
		mx, my := float64(W)*0.78, float64(H)*0.18
		for yy := -3; yy <= 3; yy++ {
			for xx := -6; xx <= 6; xx++ {
				if math.Hypot(float64(xx)/2, float64(yy)) <= 3 {
					s.set(int(mx)+xx, int(my)+yy, "Oo0"[hash2(xx, yy, 6)%3], col{150, 150, 132})
				}
			}
		}
		// montanhas em duas camadas
		for layer, cl := range []col{{56, 62, 100}, {34, 38, 62}} {
			for x := 0; x < W; x++ {
				top := int(float64(H)*(0.4+0.08*float64(layer)) - 12*noise1(float64(x)/(16-float64(layer)*5)+float64(layer)*3, 50+layer))
				for y := top; y < H; y++ {
					s.set(x, y, "^#"[min(layer, 1)], cl)
				}
			}
		}
		// chão do vale com capim balançando
		g0 := H * 60 / 100
		for y := g0; y < H; y++ {
			for x := 0; x < W; x++ {
				sway := math.Sin(t*1.5+float64(x)*0.3) * 0.5
				ch := "/|\\"[int(1+sway+0.5)%3]
				s.set(x, y, ch, mix3(col{60, 90, 64}, col{20, 30, 24}, float64(y-g0)/float64(H-g0)))
			}
		}
		// fogueira e o Eremita sentado ao lado
		fx, fy := W/2, g0+2
		for f := 0; f < 5; f++ {
			for w := -2 + f/2; w <= 2-f/2; w++ {
				kk := float64(f) / 5
				s.set(fx+w, fy-f, "^*(')"[hash2(w, f, int(t*12))%5], mix3(col{255, 240, 150}, col{210, 60, 10}, kk))
			}
		}
		s.light(float64(fx), float64(fy-1), 18, col{170, 90, 25}, 0.9+0.1*math.Sin(t*9))
		for _, c := range [][3]int{{-6, -3, 'O'}, {-6, -2, '#'}, {-7, -1, '/'}, {-5, -1, '\\'}, {-8, -2, '|'}} {
			s.set(fx+c[0], fy+c[1], byte(c[2]), col{170, 150, 120})
		}
		s.set(fx+7, fy, '_', col{120, 100, 90}) // você, deitado nas cinzas
		s.set(fx+8, fy, '_', col{120, 100, 90})
		s.set(fx+9, fy, 'o', col{170, 140, 120})
	}
	return flash
}

// ---------- a língua velha ----------

// oldWord transforma uma palavra em português numa palavra da língua velha:
// sempre a mesma para a mesma palavra, mais ou menos do mesmo tamanho. Nomes
// próprios (com maiúscula) ficam como estão.
func oldWord(w string) string {
	r := []rune(w)
	if len(r) == 0 {
		return w
	}
	if unicode.IsUpper(r[0]) && len(r) > 3 {
		return w
	}
	cons := []string{"k", "r", "v", "th", "d", "m", "sh", "g", "n", "z", "l", "kh"}
	vows := []string{"a", "o", "u", "e", "ai", "y", "ou"}
	h := uint32(2166136261)
	for _, c := range strings.ToLower(w) {
		h = (h ^ uint32(c)) * 16777619
	}
	n := max(1, (len(r)+1)/3)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(cons[h%uint32(len(cons))])
		h = h*1103515245 + 12345
		b.WriteString(vows[h%uint32(len(vows))])
		h = h*1103515245 + 12345
	}
	if h%3 == 0 {
		b.WriteString(cons[h%uint32(len(cons))])
	}
	return b.String()
}

// oldLine traduz a frase inteira, palavra a palavra (pontuação fica).
func oldLine(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		core := strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) })
		tail := strings.TrimLeftFunc(strings.TrimPrefix(w, strings.SplitN(w, core, 2)[0]), unicode.IsLetter)
		if core == "" {
			out = append(out, w)
			continue
		}
		ow := oldWord(core)
		if unicode.IsUpper([]rune(core)[0]) {
			ow = strings.ToUpper(ow[:1]) + ow[1:]
		}
		out = append(out, ow+tail)
	}
	return out
}

// speakWord: voz de velho (grave, rouca) falando a palavra da língua velha.
func speakWord(w string, pitch float64, final bool) wave {
	r := &rnd{uint32(len(w))*2654435761 + 7}
	var segs []wave
	lw := strings.ToLower(w)
	for i := 0; i < len(lw); i++ {
		c := lw[i]
		if v, ok := map[byte]byte{'a': 'a', 'e': 'e', 'i': 'i', 'o': 'o', 'u': 'u', 'y': 'y'}[c]; ok {
			p := pitch
			if final && i >= len(lw)-2 {
				p *= 0.85 // fim da frase: a voz cai
			}
			segs = append(segs, formantVoice(r, 0.11, func(t float64) float64 { return p * (1.02 - 0.06*t) }, string(v)+string(v), 0.25, 0.12, 0.01, 0.06))
			continue
		}
		switch c { // consoantes: sopro, estalo ou zumbido curtos
		case 's', 'z', 'h':
			segs = append(segs, noiseBurst(r, 0.05, 5000, 4000, 0.5, 0.005, 0.015, 2))
		case 'k', 't', 'd', 'g':
			segs = append(segs, noiseBurst(r, 0.025, 2500, 1500, 0.4, 0.001, 0.006, 1))
		case 'm', 'n', 'l', 'r', 'v':
			segs = append(segs, formantVoice(r, 0.05, func(float64) float64 { return pitch }, "uu", 0.3, 0.05, 0.005, 0.03))
		}
	}
	n := 0
	for _, s := range segs {
		n += len(s)
	}
	out := make(wave, 0, n+sampleRate/20)
	for _, s := range segs {
		out = append(out, s...)
	}
	out = append(out, make(wave, sampleRate/20)...)
	return out.fadeEnds().norm(0.7)
}

// sceneGPU é um segundo desenhador de grade (mesmo shader e atlas do mundo),
// para a cena não brigar com o tamanho da grade do mundo.
func (g *Game) sceneGPU() *gpu {
	if g.sceneG == nil {
		cp := *g.g // mesmo shader; grade e atlas próprios (o atlas é do tamanho da célula)
		cp.cells, cp.cellW, cp.cellH, cp.atlas, cp.atlasCell = nil, 0, 0, nil, 0
		g.sceneG = &cp
	}
	return g.sceneG
}

// drawScene desenha a cena da página na tela toda.
func (g *Game) drawScene(dst *ebiten.Image, page int, t float64) {
	cs := max(6, int(float64(g.H)/90))
	cols, rows := g.W/cs+1, g.H/cs+1
	if g.scene == nil || g.scene.cols != cols || g.scene.rows != rows {
		g.scene = &sceneBuf{cols: cols, rows: rows, b: make([]byte, cols*rows*4)}
	}
	flash := sceneFor(g.scene, page, t)
	if flash > 0 && t-g.thunderAt > 1.5 {
		g.thunderAt = t
		g.sfx("thunder", 0.6)
	}
	sg := g.sceneGPU()
	sg.buildAtlas(cs)
	sg.draw(dst, g.scene.b, cols, rows, 0, 0, cs, 0.5)
}
