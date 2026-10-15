package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Cards da história em ASCII animado: a abertura (opening.go) e o card de
// cada capítulo novo. A arte é desenhada letra a letra, e cada letra tem o
// seu movimento conforme o que ela é: água corre, fogo e brasa tremem,
// estrela pisca, olho pulsa. Por cima vai uma camada de partículas (brasas
// subindo, neve caindo, pó, estrelas) e, em algumas telas, linhas que
// balançam (as cabeças de Tiamath).

// artFx diz o que se move numa arte.
type artFx struct {
	particles string  // "embers", "snow", "dust", "stars", "spores", "drops" ou ""
	sway      int     // as primeiras N linhas balançam de lado
	eyes      col     // cor do brilho dos "o" (olhos); zero = sem olhos
	water     col     // cor da água "~"
	fire      bool    // ( ) ^ tremem como chama
	pcol      col     // cor das partículas
	seed      float64 // muda o sorteio das partículas
}

// drawArt desenha a arte animada com a tinta tint, a partir de (x0, y0) com
// letras de tamanho sz, no tempo t (segundos) e com opacidade fade.
func (g *Game) drawArt(dst *ebiten.Image, art string, x0, y0, sz, t, fade float64, tint col, fx artFx) (w, h float64) {
	lines := strings.Split(strings.Trim(art, "\n"), "\n")
	cwid, lh := sz*0.6, sz*1.15
	aw := 0
	for _, l := range lines {
		aw = max(aw, len([]rune(l)))
	}
	w, h = float64(aw)*cwid, float64(len(lines))*lh
	put := func(ch string, x, y float64, c col, a float64) {
		a = clamp(a*fade, 0, 1)
		g.txt(dst, ch, x, y, color.RGBA{cb(c[0] * a), cb(c[1] * a), cb(c[2] * a), cb(255 * a)}, sz, false, alignLeft)
	}
	for row, l := range lines {
		dx := 0.0
		if row < fx.sway {
			dx = math.Round(math.Sin(t*1.3+float64(row)*0.5)*1.4) * cwid * 0.5
		}
		for colI, r := range []rune(l) {
			if r == ' ' {
				continue
			}
			x, y := x0+float64(colI)*cwid+dx, y0+float64(row)*lh
			c, a, ch := tint, 0.85+0.15*math.Sin(t*2.2+float64(row)*0.7), string(r)
			h := hash2(colI, row, 7)
			switch {
			case r == '~' && fx.water != (col{}):
				// a onda anda para a direita: o brilho corre pela linha
				k := 0.5 + 0.5*math.Sin(float64(colI)*0.7-t*3+float64(row))
				c, a = mix3(fx.water, white, k*0.35), 0.55+0.45*k
				if k > 0.85 {
					ch = "-"
				}
			case fx.fire && strings.ContainsRune("()^", r):
				k := 0.5 + 0.5*math.Sin(t*11+float64(h&63))
				c, a = mix3(col{255, 120, 40}, col{255, 220, 120}, k), 0.7+0.3*k
				if k > 0.8 && r != '^' {
					ch = "^"
				}
			case r == '*' || r == '.' || r == '+':
				// estrela/brasa/neve parada: pisca devagar, cada uma no seu ritmo
				k := 0.5 + 0.5*math.Sin(t*(1.5+float64(h&7)*0.4)+float64(h&255))
				a = 0.25 + 0.75*k
				if k > 0.92 {
					ch = "*"
				}
			case r == 'o' && fx.eyes != (col{}):
				k := 0.5 + 0.5*math.Sin(t*2.5)
				c, a = mix3(fx.eyes, white, 0.25*k), 0.6+0.4*k
			}
			put(ch, x, y, c, a)
		}
	}
	// Partículas: posição de cada uma é função do tempo (sem estado).
	if fx.particles != "" {
		n := int(w * h / 900)
		for i := 0; i < n; i++ {
			fi := float64(i) + fx.seed*97
			u, v := frac(math.Sin(fi*12.9898)*43758.5), frac(math.Sin(fi*78.233)*12345.6)
			sp := 0.5 + frac(math.Sin(fi*3.7)*999)
			var px, py float64
			ch, a := ".", 1.0
			switch fx.particles {
			case "embers": // sobem tremendo e apagam
				k := frac(v + t*0.12*sp)
				px, py = u*w+math.Sin(t*2+fi)*cwid*1.5, h*(1-k)
				a = math.Sin(k * math.Pi)
				if i%5 == 0 {
					ch = "'"
				}
			case "snow": // caem de lado
				k := frac(v + t*0.08*sp)
				px, py = frac(u+math.Sin(t*0.5+fi)*0.02+t*0.01)*w, h*k
				a, ch = 0.5+0.5*sp, []string{".", "*", "'"}[i%3]
			case "dust": // poeira que o vento leva
				k := frac(u + t*0.05*sp)
				px, py = w*k, v*h+math.Sin(t+fi)*lh*0.4
				a = 0.4 * math.Sin(k*math.Pi)
			case "spores": // esporos e vaga-lumes subindo devagar
				k := frac(v + t*0.04*sp)
				px, py = u*w+math.Sin(t*0.8+fi)*cwid*2, h*(1-k)
				a = math.Pow(math.Sin(k*math.Pi), 2) * (0.5 + 0.5*math.Sin(t*4+fi))
				ch = "*"
			case "drops": // chuva fina caindo reta
				k := frac(v + t*0.35*sp)
				px, py = u*w, h*k
				a, ch = 0.35, "|"
			case "stars":
				px, py = u*w, v*h*0.45
				a = 0.2 + 0.8*math.Pow(0.5+0.5*math.Sin(t*(1+sp)+fi), 3)
			}
			put(ch, x0+px, y0+py, fx.pcol, a)
		}
	}
	return w, h
}

func frac(v float64) float64 { return v - math.Floor(v) }

// Arte de cada capítulo (I a VI), desenhada para o card que abre o capítulo.
var chapterArt = []struct {
	art  string
	tint col
	fx   artFx
	line string // narração curta embaixo do subtítulo
}{
	{`
          +                    _/\_                   +
         _|_       +          | [] |          +      _|_
          |       _|_    ____ |    | ____    _|_      |
     .    |   .    |    |    ||    ||    |    |   .   |    .
   __|__ _|_ __|__ |  __| [] || [] || [] |__  | __|__ _|_ __|__
  ^^^ ^^ ^^^ ^^ ^^^ ^^ ^^^ ^^^ ^^ ^^^ ^^ ^^^ ^^^ ^^ ^^^ ^^^ ^^`,
		col{200, 170, 140}, artFx{particles: "embers", fire: true, pcol: col{255, 150, 60}, seed: 1},
		"A capela foi construída em cima de alguma coisa. Os sinos tocam sozinhos."},
	{`
     \||/      ,;;;,        \\||//       ,;;;,      \||/
    ;;;;;;   ;;;;;;;;;    ;;;;;;;;;;   ;;;;;;;;;   ;;;;;;
     ;||;   ;;; o o ;;;     ;;||;;    ;;; o o ;;;   ;||;
      ||      ;;;;;;;        ||||       ;;;;;;;      ||
    __||__      ||||       __||||__       ||||     __||__
   ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~`,
		col{110, 170, 100}, artFx{particles: "spores", eyes: col{230, 200, 80}, water: col{60, 110, 90}, pcol: col{180, 240, 140}, seed: 2},
		"Quem entra na mata ouve o próprio nome cantado entre as folhas."},
	{`
                         \   |   /
                       ---  ( )  ---          )\
        __/\__           /  |  \             /  \__
   ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
   ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~`,
		col{200, 120, 110}, artFx{particles: "drops", water: col{170, 40, 40}, fire: true, pcol: col{140, 130, 160}, seed: 3},
		"No fundo do mar alguma coisa responde às rezas. E ela está com fome."},
	{`
              .                 /\                .
                               /  \
         *                    / [] \                    *
                             /      \
          ___               /   ()   \               ___
     ____/   \_____________/__________\_____________/   \____
    ..   ...  ....  ..  ...   ....   ..  ...  ....  ..   ...`,
		col{220, 180, 110}, artFx{particles: "dust", fire: true, pcol: col{220, 190, 130}, seed: 4},
		"Os escaravelhos levam a areia para dentro. A pirâmide cresce para baixo."},
	{`
                    /\                    /\
          /\       /**\        /\        /**\       /\
         /**\     /    \      /**\      /    \     /**\
        /    \   /      \    /    \    /      \   /    \
   ____/      \_/        \__/      \__/        \_/      \____
   .  .   .  .   .  .   .   .  .   .   .  .   .   .  .   .`,
		col{170, 200, 230}, artFx{particles: "snow", pcol: col{230, 240, 255}, seed: 5},
		"O frio não mata depressa. Ele espera você parar de andar."},
	{`
          __                                          __
        _/  \_          ____________________        _/  \_
       /  o   \_______/                      \_____/   o  \
       \      /       \     (  (  ()  )  )   /     \      /
        \____/          \____________________/       \____/
            \\\\  \\\\           ^^^^^^           ////  ////`,
		col{220, 80, 70}, artFx{particles: "embers", sway: 5, eyes: col{255, 210, 80}, fire: true, pcol: col{255, 120, 50}, seed: 6},
		"Cinco cabeças, um só ninho. E o ninho é o mundo inteiro."},
}

// drawChapterCard é o card que abre cada capítulo: arte animada numa moldura,
// o número, o título surgindo letra a letra e a narração. Dura 9 s e some
// sozinho; o jogo continua por trás.
func (g *Game) drawChapterCard(dst *ebiten.Image, cw, chh, now float64) {
	ch := g.lastChapter
	const dur = 9000.0
	t := (now - g.chapterAt) / dur
	if g.showcase || g.openPanel != "" || g.bigMap || g.chapterAt <= 0 || t >= 1 || ch < 1 || ch > len(g.info.Chapters) {
		return
	}
	a := math.Min(1, math.Min(t*6, (1-t)*4))
	secs := (now - g.chapterAt) / 1000
	c := g.info.Chapters[ch-1]
	ca := chapterArt[min(ch-1, len(chapterArt)-1)]

	w := math.Min(760, cw-40)
	h := 330.0
	x, y := (cw-w)/2, chh*0.12
	g.rect(dst, x, y, w, h, color.RGBA{4, 3, 8, uint8(215 * a)})
	gold := color.RGBA{cb(214 * a), cb(176 * a), cb(80 * a), cb(220 * a)}
	g.stroke(dst, x, y, w, h, gold, 1)
	g.stroke(dst, x+4, y+4, w-8, h-8, color.RGBA{cb(120 * a), cb(96 * a), cb(50 * a), cb(160 * a)}, 1)
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, gold, 14, true, alignCenter)
	}

	// Número do capítulo, como um selo: "- = < III > = -".
	roman := []string{"I", "II", "III", "IV", "V", "VI", "VII"}[min(ch-1, 6)]
	g.txt(dst, "-=< "+roman+" >=-", cw/2, y+12, gold, 15, true, alignCenter)

	// Arte animada, centralizada.
	sz := 14.0
	lines := strings.Split(strings.Trim(ca.art, "\n"), "\n")
	aw := 0
	for _, l := range lines {
		aw = max(aw, len([]rune(l)))
	}
	if float64(aw)*sz*0.6 > w-40 {
		sz = (w - 40) / float64(aw) / 0.6
	}
	g.drawArt(dst, ca.art, cw/2-float64(aw)*sz*0.6/2, y+40, sz, secs, a, ca.tint, ca.fx)

	// Título e subtítulo surgindo letra a letra; narração por último.
	title := c[0]
	if i := strings.Index(title, "—"); i >= 0 {
		title = strings.TrimSpace(title[i+len("—"):])
	}
	ty := y + 48 + float64(len(lines))*sz*1.15 + 14
	typed := func(s string, start, speed float64) string {
		r := []rune(s)
		n := clampi(int((secs-start)*speed), 0, len(r))
		return string(r[:n])
	}
	g.txt(dst, typed(title, 0.3, 30), cw/2, ty, color.RGBA{cb(235 * a), cb(212 * a), cb(170 * a), cb(255 * a)}, 26, true, alignCenter)
	for i, l := range g.wrapPx(typed(c[1], 1.0, 45), 14, w-60) {
		g.txt(dst, l, cw/2, ty+38+float64(i)*20, color.RGBA{cb(170 * a), cb(160 * a), cb(165 * a), cb(255 * a)}, 14, false, alignCenter)
	}
	if ca.line != "" {
		g.txt(dst, typed(ca.line, 3.2, 40), cw/2, y+h-26, color.RGBA{cb(140 * a), cb(120 * a), cb(110 * a), cb(255 * a)}, 12, false, alignCenter)
	}
}
