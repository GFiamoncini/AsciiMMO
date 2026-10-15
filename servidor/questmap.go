package main

import (
	"math"

	"asciimmo/proto"
)

// Onde fazer cada missão, para o mapa: a região a visitar, a porta da cripta
// ou o lugar onde os inimigos-alvo se concentram. Se o lugar fica no subsolo,
// vai junto a entrada (porta da capela ou boca de caverna) para o mapa de cima.

type questSpot struct {
	X, Y, R, EX, EY int
	Under           bool
}

// spotOf calcula o lugar da missão uma vez, na criação do mundo.
func (wd *World) spotOf(q step) questSpot {
	var s questSpot
	switch q.Kind {
	case qEnter:
		s = questSpot{X: chapelX, Y: chapelY, R: 14}
	case qVisit:
		r := regions[q.Targets[0]]
		s = questSpot{X: r.X, Y: r.Y, R: r.R}
	default:
		var pts [][2]int
		for _, sp := range wd.spawns {
			for _, k := range q.Targets {
				if sp.Kind == k {
					pts = append(pts, [2]int{sp.X, sp.Y})
				}
			}
		}
		if has(q.Targets, killDragon) {
			pts = append(pts, [2]int{lairX, lairY})
		}
		s = cluster(pts)
	}
	if s.Y >= overH {
		s.Under = true
		s.EX, s.EY = chapelX, chapelY // cripta: pela capela
		if s.X >= zones[1].X && s.X < zones[1].X+zones[1].W {
			best := math.MaxFloat64
			for _, m := range caveMouths { // cavernas: a boca mais perto do alvo
				if d := dist(float64(m.TX), float64(m.TY), float64(s.X), float64(s.Y)); d < best {
					best, s.EX, s.EY = d, m.X, m.Y
				}
			}
		}
	}
	return s
}

// cluster acha o ponto com mais vizinhos (a maior concentração) e devolve o
// centro e o raio desse grupo.
func cluster(pts [][2]int) questSpot {
	if len(pts) == 0 {
		return questSpot{X: villageX, Y: villageY, R: 20}
	}
	const near = 140.0
	best, bi := -1, 0
	for i, a := range pts {
		n := 0
		for _, b := range pts {
			if dist(float64(a[0]), float64(a[1]), float64(b[0]), float64(b[1])) <= near {
				n++
			}
		}
		if n > best {
			best, bi = n, i
		}
	}
	a := pts[bi]
	var sx, sy, n float64
	for _, b := range pts {
		if dist(float64(a[0]), float64(a[1]), float64(b[0]), float64(b[1])) <= near {
			sx, sy, n = sx+float64(b[0]), sy+float64(b[1]), n+1
		}
	}
	cx, cy := sx/n, sy/n
	r := 24.0
	for _, b := range pts {
		if d := dist(cx, cy, float64(b[0]), float64(b[1])); d <= near {
			r = math.Max(r, d+10)
		}
	}
	return questSpot{X: int(cx), Y: int(cy), R: int(math.Min(r, 120))}
}

func (wd *World) buildSpots() {
	wd.mainSpots = make([][]questSpot, len(quests))
	for i, q := range quests {
		for _, s := range stepsOf(q) {
			wd.mainSpots[i] = append(wd.mainSpots[i], wd.spotOf(s))
		}
	}
	wd.sideSpots = make([]questSpot, len(sideQuests))
	for i, q := range sideQuests {
		wd.sideSpots[i] = wd.spotOf(stepsOf(q)[0])
	}
}

// marksFor lista as marcas do jogador: a missão principal atual e as
// secundárias já disponíveis e ainda não feitas.
func (wd *World) marksFor(p *Player) []proto.Mark {
	var out []proto.Mark
	add := func(s questSpot, title string, side bool) {
		out = append(out, proto.Mark{X: s.X, Y: s.Y, R: s.R, EX: s.EX, EY: s.EY, Under: s.Under, Title: title, Side: side})
	}
	if p.inPrologue() { // o prólogo aponta só a etapa dele
		s := prologue[p.Intro]
		sp := wd.spotOf(s)
		switch s.Kind {
		case qTalk:
			sp = questSpot{X: hermit.X, Y: hermit.Y, R: 8}
		case qRest:
			b := wd.bonfires[len(wd.bonfires)-1]
			sp = questSpot{X: b.X, Y: b.Y, R: 8}
		}
		add(sp, s.Text, false)
		return out
	}
	if p.Quest < len(quests) && p.Quest < len(wd.mainSpots) {
		sp := wd.mainSpots[p.Quest]
		s := sp[min(p.QStep, len(sp)-1)]
		title := quests[p.Quest].Title
		if st, ok := p.curStep(); ok {
			title = st.Text
		}
		add(s, title, false)
	}
	ch := p.chapterOf()
	for i, q := range sideQuests {
		if i < len(wd.sideSpots) && q.Chapter <= ch && p.SideDone&(1<<i) == 0 {
			add(wd.sideSpots[i], q.Title, true)
		}
	}
	return out
}

// MarksOf é marksFor com a trava do mundo, para o laço do WebSocket.
func (wd *World) MarksOf(id int) []proto.Mark {
	wd.mu.RLock()
	defer wd.mu.RUnlock()
	if p, ok := wd.players[id]; ok {
		return wd.marksFor(p)
	}
	return nil
}
