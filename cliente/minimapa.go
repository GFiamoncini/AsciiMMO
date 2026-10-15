package main

import (
	"asciimmo/proto"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const mm = proto.MiniScale

// miniPixel pega a cor mais viva de um bloco de células, para caminhos e
// construções não sumirem.
func (g *Game) miniPixel(mx, my int, d []byte) {
	m := g.m
	var r, gg, b, cnt, water, solid float64
	for dy := 0; dy < mm; dy++ {
		for dx := 0; dx < mm; dx++ {
			i := (my*mm+dy)*m.w + mx*mm + dx
			if i >= m.w*m.h {
				continue
			}
			if m.flags[i]&4 != 0 {
				water++
			}
			if m.flags[i]&1 != 0 {
				solid++
			}
			if m.glyph[i] != ' ' {
				r, gg, b, cnt = r+float64(m.color[i*3]), gg+float64(m.color[i*3+1]), b+float64(m.color[i*3+2]), cnt+1
			}
		}
	}
	k := 0.0
	if cnt > 0 {
		k = (0.35 + 0.65*cnt/(mm*mm)) * 1.25 / cnt
	}
	switch {
	case water > 8:
		d[0], d[1], d[2] = 30, 58, 120
	case cnt == 0 && solid > 8:
		d[0], d[1], d[2] = 6, 6, 10
	default:
		d[0], d[1], d[2] = cb(14+r*k), cb(14+gg*k), cb(22+b*k)
	}
	d[3] = 255
}

func (g *Game) buildMinimap() {
	w, h := g.m.w/mm, g.m.h/mm
	g.miniPix = make([]byte, w*h*4)
	for my := 0; my < h; my++ {
		for mx := 0; mx < w; mx++ {
			g.miniPixel(mx, my, g.miniPix[(my*w+mx)*4:])
		}
	}
	if g.mini != nil {
		g.mini.Deallocate()
	}
	g.mini = ebiten.NewImage(w, h)
	g.mini.WritePixels(g.miniPix)
	g.saveFog()
	g.fog = nil
}

func (g *Game) fogPath() string {
	return filepath.Join(configDir(), "nevoa_"+strings.ToLower(g.charName)+".bin")
}

// A névoa do mapa: uma célula do minimapa só aparece depois de o jogador
// passar perto. Fica guardada neste PC, por personagem, e também no save do
// servidor (vem no init); as duas se somam.
func (g *Game) initFog() {
	w, h := g.m.w/mm, g.m.h/mm
	g.fog = make([]byte, w*h)
	if raw, err := os.ReadFile(g.fogPath()); err == nil {
		for i := range g.fog {
			if i>>3 < len(raw) {
				g.fog[i] = raw[i>>3] >> (i & 7) & 1
			}
		}
	}
	if g.info != nil && g.info.Fog != "" {
		if srv := unpackFog(g.info.Fog); len(srv) == (w*h+7)/8 {
			for i := range g.fog {
				g.fog[i] |= srv[i>>3] >> (i & 7) & 1
			}
			g.fogDirty = true
		}
	}
	g.fogPix = make([]byte, w*h*4)
	for i, f := range g.fog {
		if f == 0 {
			g.fogPix[i*4], g.fogPix[i*4+1], g.fogPix[i*4+2], g.fogPix[i*4+3] = 5, 5, 10, 255
		}
	}
	if g.fogImg != nil {
		g.fogImg.Deallocate()
	}
	g.fogImg = ebiten.NewImage(w, h)
	g.fogImg.WritePixels(g.fogPix)
	g.fogLast = -1
}

func (g *Game) reveal(me *proto.Player, now float64) {
	w, h := g.m.w/mm, g.m.h/mm
	mx, my, R := me.X/mm, me.Y/mm, 28
	if my*w+mx != g.fogLast {
		g.fogLast = my*w + mx
		for dy := -R; dy <= R; dy++ {
			for dx := -R; dx <= R; dx++ {
				x, y := mx+dx, my+dy
				if dx*dx+dy*dy > R*R || x < 0 || y < 0 || x >= w || y >= h || g.fog[y*w+x] != 0 {
					continue
				}
				i := y*w + x
				g.fog[i] = 1
				g.fogPix[i*4], g.fogPix[i*4+1], g.fogPix[i*4+2], g.fogPix[i*4+3] = 0, 0, 0, 0
				g.fogDirty, g.fogImgDirty = true, true
			}
		}
	}
	if g.fogImgDirty {
		g.fogImg.WritePixels(g.fogPix)
		g.fogImgDirty = false
	}
	if g.fogDirty && now-g.fogAt > 5000 {
		g.fogAt, g.fogDirty = now, false
		raw := make([]byte, (len(g.fog)+7)/8)
		for i, f := range g.fog {
			raw[i>>3] |= f << (i & 7)
		}
		os.WriteFile(g.fogPath(), raw, 0o644)
	}
}

type zoneRect struct {
	name       string
	x, y, w, h int
}

// mapSrc: lá fora o mapa mostra o continente; no subsolo, a zona do jogador.
func (g *Game) mapSrc(me *proto.Player) zoneRect {
	if me.Y < g.info.OverH {
		return zoneRect{"", 0, 0, g.m.w, g.info.OverH}
	}
	for _, z := range g.info.Zones {
		if me.X >= z.X && me.X < z.X+z.W && me.Y >= z.Y && me.Y < z.Y+z.H {
			return zoneRect{z.Name, z.X, z.Y, z.W, z.H}
		}
	}
	if len(g.info.Zones) > 0 {
		z := g.info.Zones[0]
		return zoneRect{z.Name, z.X, z.Y, z.W, z.H}
	}
	return zoneRect{"Subsolo", 0, g.info.OverH, g.m.w, g.m.h - g.info.OverH}
}

func (g *Game) seen(wx, wy int) bool {
	i := (wy/mm)*(g.m.w/mm) + wx/mm
	return i >= 0 && i < len(g.fog) && g.fog[i] != 0
}

// drawMap desenha o mapa (pequeno ou grande) com a névoa e os pontos de
// interesse já descobertos; no grande entram os nomes das regiões.
func (g *Game) drawMap(dst *ebiten.Image, me *proto.Player, x, y, mw, mh float64, src zoneRect, big bool) {
	for _, img := range []*ebiten.Image{g.mini, g.fogImg} {
		sub := img.SubImage(imageRect2(src.x/mm, src.y/mm, (src.x+src.w)/mm, (src.y+src.h)/mm)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(mw*g.ui/float64(sub.Bounds().Dx()), mh*g.ui/float64(sub.Bounds().Dy()))
		op.GeoM.Translate(x*g.ui, y*g.ui)
		dst.DrawImage(sub, op)
	}
	dot := func(wx, wy float64, c color.Color, size float64) {
		if wx < float64(src.x) || wy < float64(src.y) || wx > float64(src.x+src.w) || wy > float64(src.y+src.h) {
			return
		}
		g.rect(dst, x+(wx-float64(src.x))/float64(src.w)*mw-size/2, y+(wy-float64(src.y))/float64(src.h)*mh-size/2, size, size, c)
	}
	s := 0.0
	if big {
		s = 2
	}
	for _, p := range g.info.Portals {
		if g.seen(p.X, p.Y) {
			dot(float64(p.X), float64(p.Y), rgbaHex(0xb9a8ff), 4+s)
		}
	}
	for i, c := range g.info.Chests {
		if g.seen(c[0], c[1]) {
			cc := rgbaHex(0xffd070)
			if i < len(g.cur.Chests) && g.cur.Chests[i] {
				cc = rgbaHex(0x6a6a76)
			}
			dot(float64(c[0]), float64(c[1]), cc, 4+s)
		}
	}
	for _, mb := range g.cur.Mobs {
		if mb.Boss && g.seen(int(mb.X), int(mb.Y)) {
			dot(mb.X, mb.Y, rgbaHex(0xff5ad0), 5+s)
		}
	}
	if b := g.cur.Boss; b.Alive && g.seen(int(b.X), int(b.Y)) {
		dot(b.X, b.Y, rgbaHex(0xff4030), 6+s)
	}
	for _, p := range g.cur.Players {
		if p.ID != me.ID {
			dot(float64(p.X), float64(p.Y), playerColor(p.Color).rgba(), 4+s)
		}
	}
	dot(float64(me.X), float64(me.Y), color.Black, 7+s)
	dot(float64(me.X), float64(me.Y), cGold, 5+s)

	// Missões: "!" dourado na principal, "?" lilás nas secundárias, com um
	// anel do tamanho da área; no subsolo, o mapa de cima mostra a entrada.
	pulse := 0.5 + 0.5*math.Sin(g.now()/250)
	for _, mk := range g.cur.Marks {
		wx, wy, ok, entr := g.markPos(mk, me)
		if !ok || wx < float64(src.x) || wy < float64(src.y) || wx > float64(src.x+src.w) || wy > float64(src.y+src.h) {
			continue
		}
		px, py := x+(wx-float64(src.x))/float64(src.w)*mw, y+(wy-float64(src.y))/float64(src.h)*mh
		c, sym := rgbaHex(0xffd34a), "!"
		if mk.Side {
			c, sym = rgbaHex(0xc9a0ff), "?"
		}
		r := math.Max(5+s, float64(mk.R)/float64(src.w)*mw)
		if entr {
			r = 5 + s
		}
		ring := c
		ring.A = uint8(120 + 120*pulse)
		vector.StrokeCircle(dst, float32(px*g.ui), float32(py*g.ui), float32(r*g.ui), float32(1.5*g.ui), ring, true)
		g.txt(dst, sym, px, py-8-s, c, 13+s*2, true, alignCenter)
		if big {
			title := mk.Title
			if entr {
				title += " (entrada)"
			}
			tw := g.textW(title, 12)
			g.rect(dst, px-tw/2-3, py+r+2, tw+6, 15, color.RGBA{0, 0, 0, 190})
			g.txt(dst, title, px, py+r+3, c, 12, false, alignCenter)
		}
	}
	if !big || src.y != 0 {
		return
	}
	for ri, r := range g.info.Regions {
		if !g.seen(r.X, r.Y) {
			continue
		}
		tx, ty := x+float64(r.X)/float64(src.w)*mw, y+(float64(r.Y)-float64(r.R)*0.5)/float64(src.h)*mh
		name, nc := r.Name, color.Color(color.White)
		if g.sealedBy(ri) >= 0 {
			name, nc = r.Name+" (selado)", rgbaHex(0xc080ff)
		}
		tw := g.textW(name, 13)
		g.rect(dst, tx-tw/2-3, ty-2, tw+6, 17, color.RGBA{0, 0, 0, 180})
		g.txt(dst, name, tx, ty, nc, 13, true, alignCenter)
	}
}

// unpackFog abre o explorado que o servidor manda (gzip + base64).
func unpackFog(s string) []byte {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b, _ := io.ReadAll(zr)
	return b
}

// saveFog grava a névoa agora (antes de descartá-la ao reconectar ou ao
// recarregar o mapa, para não perder os últimos segundos explorados).
func (g *Game) saveFog() {
	if g.fog == nil || g.charName == "" {
		return
	}
	raw := make([]byte, (len(g.fog)+7)/8)
	for i, f := range g.fog {
		raw[i>>3] |= f << (i & 7)
	}
	os.WriteFile(g.fogPath(), raw, 0o644)
	g.fogDirty = false
}

// markPos diz onde mostrar uma missão para quem está em me: o lugar dela ou,
// se ela fica no subsolo e o jogador está em cima, a entrada. ok é falso
// quando o lugar está no outro andar sem entrada a mostrar.
func (g *Game) markPos(mk proto.Mark, me *proto.Player) (x, y float64, ok, entrance bool) {
	under := me.Y >= g.info.OverH
	switch {
	case mk.Under && !under:
		return float64(mk.EX), float64(mk.EY), true, true
	case mk.Under == under:
		return float64(mk.X), float64(mk.Y), true, false
	}
	return 0, 0, false, false
}

// drawQuestArrow aponta, na borda da tela, para a missão principal quando ela
// está fora de vista. Só a seta, pulsando: nome e distância já estão no painel
// da missão e na bússola.
func (g *Game) drawQuestArrow(dst *ebiten.Image, me *proto.Player, cw, chh float64) {
	if g.meScreen == nil || !g.opts.QuestArrow {
		return
	}
	for _, mk := range g.cur.Marks {
		if mk.Side {
			continue
		}
		wx, wy, ok, entr := g.markPos(mk, me)
		if !ok {
			return
		}
		dx, dy := wx-float64(me.X), wy-float64(me.Y)
		d := math.Hypot(dx, dy)
		cs := float64(g.cs) / g.ui
		if d < float64(mk.R)*0.6 || math.Abs(dx)*cs < cw*0.42 && math.Abs(dy)*cs < chh*0.4 {
			return // já dá para ver
		}
		a := math.Atan2(dy, dx)
		cx, cy := g.meScreen[0]/g.ui, g.meScreen[1]/g.ui
		// Ponto na borda da área útil, na direção do alvo.
		mx, my := cw*0.5-rightW*0.5-40, chh*0.5-70
		t := math.Min(mx/math.Max(1e-6, math.Abs(math.Cos(a))), my/math.Max(1e-6, math.Abs(math.Sin(a))))
		px, py := clamp(cx+math.Cos(a)*t, 40, cw-rightW-40), clamp(cy+math.Sin(a)*t, 60, chh-110)
		pulse := 0.6 + 0.4*math.Sin(g.now()/200)
		c := color.RGBA{255, 211, 74, uint8(200 * pulse)}
		var path vector.Path
		tip, back, side := 16.0, 6.0, 9.0
		path.MoveTo(float32((px+math.Cos(a)*tip)*g.ui), float32((py+math.Sin(a)*tip)*g.ui))
		path.LineTo(float32((px-math.Cos(a)*back+math.Cos(a+math.Pi/2)*side)*g.ui), float32((py-math.Sin(a)*back+math.Sin(a+math.Pi/2)*side)*g.ui))
		path.LineTo(float32((px-math.Cos(a)*back*0.2)*g.ui), float32((py-math.Sin(a)*back*0.2)*g.ui))
		path.LineTo(float32((px-math.Cos(a)*back+math.Cos(a-math.Pi/2)*side)*g.ui), float32((py-math.Sin(a)*back+math.Sin(a-math.Pi/2)*side)*g.ui))
		path.Close()
		vector.FillPath(dst, &path, &vector.FillOptions{}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(c)})
		// Só a seta: o objetivo já está no painel da missão e na bússola.
		_ = entr
		return
	}
}
