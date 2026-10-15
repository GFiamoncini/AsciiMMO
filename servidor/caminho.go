package main

import (
	"container/heap"
	"math"
	"time"
)

// Caminho dos inimigos. Andar reto até o alvo escorregando pelas paredes
// (mobToward) prende o inimigo atrás de pilar, quina e parede em U: o chefe
// da cripta ficava empacado sem chegar ao jogador. Quando a reta está
// bloqueada, o inimigo pede um caminho (A* na grade, 8 vizinhos, numa caixa
// em volta dele) e segue os pontos. O caminho é refeito quando o alvo se
// afasta do fim dele, quando acaba, ou a cada pouco; e há um limite de
// buscas por passo do mundo, para muitos inimigos juntos não pesarem.

const (
	pathBox      = 72   // raio da caixa da busca
	pathMaxNodes = 9000 // células abertas no máximo por busca
	pathPerTick  = 10   // buscas por passo do mundo
)

type mobNav struct {
	path       [][2]int
	at         int
	goalX      int
	goalY      int
	redo       time.Time
	lastX      float64
	lastY      float64
	stillSince time.Time
}

// clearLine diz se dá para andar em linha reta de a até b (célula a célula).
func (wd *World) clearLine(ax, ay, bx, by float64) bool {
	d := math.Hypot(bx-ax, by-ay)
	n := int(d) + 1
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		if !wd.Walkable(int(ax+(bx-ax)*t), int(ay+(by-ay)*t)) {
			return false
		}
	}
	return true
}

// mobLine é clearLine com o tamanho do inimigo.
func (wd *World) mobLine(m *Mob, ax, ay, bx, by float64) bool {
	d := math.Hypot(bx-ax, by-ay)
	n := int(d*2) + 1
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		if !wd.mobFits(m, ax+(bx-ax)*t, ay+(by-ay)*t) {
			return false
		}
	}
	return true
}

// mobChase leva o inimigo até (x, y): em linha reta se der, senão pelo caminho.
func (wd *World) mobChase(m *Mob, x, y, speed float64, now time.Time) {
	if wd.mobLine(m, m.X, m.Y, x, y) {
		m.nav = nil
		wd.mobToward(m, x, y, speed)
		return
	}
	n := m.nav
	gx, gy := int(x), int(y)
	stale := n == nil || n.at >= len(n.path) || now.After(n.redo) ||
		math.Hypot(float64(gx-n.goalX), float64(gy-n.goalY)) > 6
	if stale && wd.pathBudget > 0 {
		wd.pathBudget--
		path := wd.findPathR(int(m.X), int(m.Y), gx, gy, mobRad(m.Kind))
		if n == nil {
			n = &mobNav{lastX: m.X, lastY: m.Y, stillSince: now}
			m.nav = n
		}
		n.path, n.at, n.goalX, n.goalY = path, 0, gx, gy
		n.redo = now.Add(time.Duration(1200+wd.rng.Intn(600)) * time.Millisecond)
	}
	if n == nil || n.at >= len(n.path) {
		wd.mobToward(m, x, y, speed) // sem caminho (alvo fora da caixa): tenta reto
		return
	}
	// Pula os pontos que já dá para ver direto (o caminho fica mais solto).
	for n.at+1 < len(n.path) && wd.mobLine(m, m.X, m.Y, float64(n.path[n.at+1][0])+0.5, float64(n.path[n.at+1][1])+0.5) && n.at < 6 {
		n.at++
	}
	wp := n.path[n.at]
	wx, wy := float64(wp[0])+0.5, float64(wp[1])+0.5
	if math.Hypot(wx-m.X, wy-m.Y) < 1.2 {
		n.at++
	}
	wd.mobToward(m, wx, wy, speed)
	// Parado no mesmo lugar há um tempo: o caminho não serve, refaz.
	if math.Hypot(m.X-n.lastX, m.Y-n.lastY) > 0.5 {
		n.lastX, n.lastY, n.stillSince = m.X, m.Y, now
	} else if now.Sub(n.stillSince) > 800*time.Millisecond {
		n.redo, n.stillSince = now, now
	}
}

type pathNode struct {
	i    int
	f    float64
	heap int
}

type pathHeap []*pathNode

func (h pathHeap) Len() int           { return len(h) }
func (h pathHeap) Less(a, b int) bool { return h[a].f < h[b].f }
func (h pathHeap) Swap(a, b int)      { h[a], h[b] = h[b], h[a]; h[a].heap = a; h[b].heap = b }
func (h *pathHeap) Push(x any)        { n := x.(*pathNode); n.heap = len(*h); *h = append(*h, n) }
func (h *pathHeap) Pop() any          { old := *h; n := old[len(old)-1]; *h = old[:len(old)-1]; return n }

// findPath acha um caminho de (sx, sy) até (gx, gy) dentro da caixa em volta
// do começo; se o alvo não for alcançável, vai até a célula mais perto dele.
// Devolve os pontos do caminho, sem o começo.
func (wd *World) findPath(sx, sy, gx, gy int) [][2]int {
	return wd.findPathR(sx, sy, gx, gy, 0)
}

// findPathR é findPath para quem ocupa r células para cada lado.
func (wd *World) findPathR(sx, sy, gx, gy, r int) [][2]int {
	free := func(x, y int) bool {
		for dx := -r; dx <= r; dx++ {
			if !wd.Walkable(x+dx, y) {
				return false
			}
		}
		return true
	}
	if !wd.inBounds(sx, sy) {
		return nil
	}
	x0, y0 := max(0, sx-pathBox), max(0, sy-pathBox)
	x1, y1 := min(wd.w-1, sx+pathBox), min(wd.h-1, sy+pathBox)
	bw := x1 - x0 + 1
	gx, gy = clampInt(gx, x0, x1), clampInt(gy, y0, y1)
	idx := func(x, y int) int { return (y-y0)*bw + (x - x0) }
	g := make(map[int]float64, 1024)
	from := make(map[int]int, 1024)
	closed := make(map[int]bool, 1024)
	h := func(x, y int) float64 {
		dx, dy := math.Abs(float64(x-gx)), math.Abs(float64(y-gy))
		return dx + dy - 0.59*math.Min(dx, dy) // octil
	}
	start := idx(sx, sy)
	g[start] = 0
	open := &pathHeap{}
	heap.Push(open, &pathNode{i: start, f: h(sx, sy)})
	best, bestH := start, h(sx, sy)
	opened := 0
	for open.Len() > 0 && opened < pathMaxNodes {
		cur := heap.Pop(open).(*pathNode)
		if closed[cur.i] {
			continue
		}
		closed[cur.i] = true
		cx, cy := cur.i%bw+x0, cur.i/bw+y0
		if hh := h(cx, cy); hh < bestH {
			best, bestH = cur.i, hh
		}
		if cx == gx && cy == gy {
			break
		}
		for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nx, ny := cx+d[0], cy+d[1]
			if nx < x0 || ny < y0 || nx > x1 || ny > y1 || !free(nx, ny) {
				continue
			}
			cost := 1.0
			if d[0] != 0 && d[1] != 0 {
				if !free(cx+d[0], cy) || !free(cx, cy+d[1]) {
					continue // não corta quina
				}
				cost = 1.41
			}
			j := idx(nx, ny)
			if closed[j] {
				continue
			}
			ng := g[cur.i] + cost
			if old, ok := g[j]; ok && old <= ng {
				continue
			}
			g[j], from[j] = ng, cur.i
			heap.Push(open, &pathNode{i: j, f: ng + h(nx, ny)})
			opened++
		}
	}
	if best == start {
		return nil
	}
	var rev [][2]int
	for i := best; i != start; i = from[i] {
		rev = append(rev, [2]int{i%bw + x0, i/bw + y0})
	}
	for a, b := 0, len(rev)-1; a < b; a, b = a+1, b-1 {
		rev[a], rev[b] = rev[b], rev[a]
	}
	return rev
}

// Tamanho de cada inimigo no chão (meia largura, em células): os grandes não
// encostam o corpo na parede nem entram em fresta onde não cabem.
func mobRad(kind int) int {
	switch kind {
	case mobApostle, mobGrael, mobMagmaGolem, mobCrystalGolem, mobTroll, mobYeti, mobFrostGiant, mobIgnar, mobVulkhar, mobOrcChief, mobDwarfGuard, mobCroc, mobWerewolf, mobCentaur:
		return 2
	case mobRat, mobBat, mobScarab, mobDummy:
		return 0
	}
	return 1
}

// mobFits diz se o inimigo cabe com os pés em (x, y).
func (wd *World) mobFits(m *Mob, x, y float64) bool {
	ix, iy := int(x), int(y)
	r := mobRad(m.Kind)
	for dx := -r; dx <= r; dx++ {
		if !wd.Walkable(ix+dx, iy) {
			return false
		}
	}
	return true
}

// unstick tira de dentro da parede o inimigo que não cabe onde está (nasceu
// ou foi empurrado para lá): vai para a célula livre mais perto.
func (wd *World) unstick(m *Mob) {
	if wd.mobFits(m, m.X, m.Y) {
		return
	}
	for r := 1; r <= 8; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if max(abs(dx), abs(dy)) != r {
					continue
				}
				if x, y := m.X+float64(dx), m.Y+float64(dy); wd.mobFits(m, x, y) {
					m.X, m.Y = x, y
					return
				}
			}
		}
	}
}

// wanderPoint sorteia para onde vagar em volta da cova: só onde o inimigo cabe
// e dá para chegar em linha reta (senão ele ficava se esfregando na parede).
func (wd *World) wanderPoint(m *Mob) (float64, float64) {
	for i := 0; i < 6; i++ {
		x, y := m.homeX+float64(wd.rng.Intn(37)-18), m.homeY+float64(wd.rng.Intn(25)-12)
		if wd.mobFits(m, x, y) && wd.mobLine(m, m.X, m.Y, x, y) {
			return x, y
		}
	}
	return m.X, m.Y // fica parado olhando em volta
}
