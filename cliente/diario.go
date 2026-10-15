package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Diário (tecla N): duas abas.
//   - Missões: as principais em ordem (feitas com visto, a atual em dourado
//     com as etapas, as próximas escondidas) e as secundárias abertas e feitas.
//   - História: a abertura e cada capítulo já alcançado, com o texto e um botão
//     para rever o card animado (a abertura volta inteira, com a narração).

func (g *Game) drawJournal(dst *ebiten.Image, cw, chh float64) {
	w, h := math.Min(820, cw-rightW-40), math.Min(560, chh-60)
	x, y := math.Max(8, (cw-rightW-w)/2), math.Max(8, (chh-h)/2)
	g.panel(dst, x, y, w, h, cGold)
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		g.txt(dst, "+", p[0], p[1]-9, cGold, 14, true, alignCenter)
	}
	g.txt(dst, "-=< Diário >=-", x+w/2, y+10, cGold, 16, true, alignCenter)
	g.txt(dst, "N fecha", x+w-14, y+12, cDimmer, 11, false, alignRight)
	tabs := []struct{ key, label string }{{"quests", "Missões"}, {"story", "História"}}
	if g.journalTab == "" {
		g.journalTab = "quests"
	}
	for i, t := range tabs {
		tx := x + 16 + float64(i)*124
		on := g.journalTab == t.key
		g.rect(dst, tx, y+36, 116, 24, pickRGBA(on, color.RGBA{60, 46, 20, 255}, color.RGBA{14, 12, 18, 255}))
		g.stroke(dst, tx, y+36, 116, 24, pickRGBA(on || g.hover(tx, y+36, 116, 24), rgbaHex(0xd6b050), rgbaHex(0x4a3c22)), 1)
		g.txt(dst, t.label, tx+58, y+41, pickC(on, cGold, cText), 13, on, alignCenter)
		key := t.key
		g.uiRects = append(g.uiRects, uiRect{x: tx, y: y + 36, w: 116, h: 24, fn: func() { g.journalTab, g.journalScroll = key, 0 }})
	}
	top, bottom := y+72, y+h-14
	cy := top - g.journalScroll
	line := func(s string, c color.Color, size float64, indent float64) {
		for _, l := range g.wrapPx(s, size, w-40-indent) {
			if cy >= top-2 && cy < bottom-size {
				g.txt(dst, l, x+20+indent, cy, c, size, false, alignLeft)
			}
			cy += size + 5
		}
	}
	button := func(label string, fn func()) {
		bw := g.textW(label, 12) + 20
		if cy >= top-2 && cy < bottom-20 {
			hov := g.hover(x+20, cy, bw, 22)
			g.rect(dst, x+20, cy, bw, 22, color.RGBA{30, 24, 14, 255})
			g.stroke(dst, x+20, cy, bw, 22, pickRGBA(hov, rgbaHex(0xd6b050), rgbaHex(0x5a4a2a)), 1)
			g.txt(dst, label, x+20+bw/2, cy+4, pickC(hov, cGold, cText), 12, false, alignCenter)
			g.uiRects = append(g.uiRects, uiRect{x: x + 20, y: cy, w: bw, h: 22, fn: fn})
		}
		cy += 30
	}
	q := g.cur.Quest
	switch g.journalTab {
	case "quests":
		done := len(g.info.Quests)
		if q != nil {
			done = q.N - 1
			if q.Chapter == 0 {
				done = 0
			}
		}
		line("Missões principais", cGold, 14, 0)
		cy += 4
		if q != nil && q.Chapter == 0 {
			line("> Prólogo: "+q.Title, cGold, 13, 6)
			line(q.Text, cText, 12, 22)
		}
		for i, m := range g.info.Quests {
			switch {
			case i < done:
				if cy >= top-2 && cy < bottom-14 {
					g.checkMark(dst, x+24, cy+2, 10, true)
				}
				line(m.Title, cDim, 13, 18)
			case i == done && q != nil && q.Chapter > 0:
				line("> "+q.Title, cGold, 13, 6)
				line(q.Text, cText, 12, 22)
				for k, s := range q.Steps {
					mark := "·"
					c := color.Color(cDim)
					if s.Done {
						mark, c = "ok", cDimmer
					} else if k == q.Cur {
						mark, c = ">", cGold
					}
					txt := mark + " " + s.Text
					if k == q.Cur && s.Need > 1 {
						txt += fmt.Sprintf("  (%d/%d)", s.Prog, s.Need)
					}
					line(txt, c, 12, 30)
				}
			default:
				line("??? (capítulo "+roman(m.Chapter)+")", cDimmer, 12, 18)
			}
		}
		cy += 10
		line("Secundárias", rgbaHex(0xc080ff), 14, 0)
		cy += 4
		any := false
		for i, s := range g.info.Sides {
			if g.cur.Chapter < s.Chapter {
				continue
			}
			any = true
			if g.cur.SideDone&(1<<i) != 0 {
				if cy >= top-2 && cy < bottom-14 {
					g.checkMark(dst, x+24, cy+2, 10, true)
				}
				line(s.Title, cDim, 13, 18)
				continue
			}
			pr := 0
			if i < len(g.cur.SideProg) {
				pr = g.cur.SideProg[i]
			}
			line(fmt.Sprintf("%s  (%d/%d)", s.Title, pr, s.Need), cText, 13, 18)
			line(s.Text, cDim, 12, 30)
		}
		if !any {
			line("Nenhuma ainda.", cDimmer, 12, 18)
		}
	case "story":
		line("A abertura: Cinzalva ardendo, Tiamath desperta, o vale de cinzas.", cText, 13, 0)
		button("Rever a abertura", func() {
			g.openPanel, g.introPage, g.introAt, g.introReplay = "", 0, 0, true
		})
		for i := 1; i <= min(g.cur.Chapter, len(g.info.Chapters)); i++ {
			c := g.info.Chapters[i-1]
			line(c[0], cGold, 14, 0)
			line(c[1], cText, 12, 10)
			if i-1 < len(chapterArt) {
				line(chapterArt[i-1].line, cDim, 12, 10)
			}
			ch := i
			button("Rever o card", func() {
				g.openPanel, g.lastChapter, g.chapterAt = "", ch, g.now()
			})
		}
	}
	// Rolagem: roda do mouse ou setas, quando o conteúdo passa da altura.
	total := cy + g.journalScroll - top
	g.journalMax = math.Max(0, total-(bottom-top))
	if g.journalMax > 0 {
		k := g.journalScroll / g.journalMax
		g.rect(dst, x+w-8, top, 3, bottom-top, color.RGBA{40, 36, 30, 255})
		g.rect(dst, x+w-8, top+(bottom-top-30)*k, 3, 30, color.RGBA{214, 176, 80, 200})
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

func roman(n int) string {
	r := []string{"", "I", "II", "III", "IV", "V", "VI", "VII"}
	if n >= 0 && n < len(r) {
		return r[n]
	}
	return fmt.Sprint(n)
}

// journalKeys: rolar com setas e roda; Tab troca a aba.
func (g *Game) journalKeys(wheel float64) {
	if g.openPanel != "diario" {
		return
	}
	g.journalScroll -= wheel * 40
	if repeatKey(ebiten.KeyArrowDown) {
		g.journalScroll += 30
	}
	if repeatKey(ebiten.KeyArrowUp) {
		g.journalScroll -= 30
	}
	if justKey(ebiten.KeyTab) {
		if g.journalTab == "story" {
			g.journalTab = "quests"
		} else {
			g.journalTab = "story"
		}
		g.journalScroll = 0
	}
	g.journalScroll = clamp(g.journalScroll, 0, g.journalMax)
}
