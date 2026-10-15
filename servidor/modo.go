package main

import (
	"container/heap"
	"math"
	"time"
)

// Modo de combate e caminhar até um ponto do mapa.
//
// Modo (Player.Mode, tecla B no cliente, ícone ao lado do card):
//   - Pacifista: só golpeia quando você manda (o jeito de sempre);
//   - Defensivo: revida sozinho com o golpe básico em quem te feriu há pouco;
//   - Agressivo: golpeia sozinho o inimigo mais perto ao alcance, inclusive
//     andando ("o que achar no caminho").
//
// Caminhar até um ponto (clique no mapa grande): o servidor acha o caminho
// numa grade grossa (de 4 em 4 células, com o tamanho do personagem) e vai
// conduzindo o jogador ponto a ponto. Qualquer tecla de andar cancela; parar
// preso refaz o caminho uma vez e depois desiste.

const (
	modePacifist = iota
	modeDefensive
	modeAggressive
	numModes
)

var modeNames = [numModes]string{"pacifista", "defensivo", "agressivo"}

// SetMode troca o modo de combate.
func (wd *World) SetMode(id, m int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok && m >= 0 && m < numModes {
		p.Mode = m
		wd.pop(p, "Modo "+modeNames[m], 0xd0c090, float64(p.X), float64(p.Y)-34, wd.clock())
	}
}

// basicReach é o alcance do golpe básico da classe (perto ou de longe).
func basicReach(class int) float64 {
	switch class {
	case 1, 4, 6: // mago, arqueiro, necromante
		return 34
	}
	return 11
}

// autoFight solta o golpe básico sozinho, conforme o modo. A cada 0,4 s.
func (wd *World) autoFight(p *Player, now time.Time) {
	if p.Mode == modePacifist || p.paused || p.boat || p.tame != nil || p.fish != nil {
		return
	}
	if p.Mode == modeDefensive && now.Sub(p.hurtAt) > 4*time.Second {
		return
	}
	reach := basicReach(p.Class)
	var best *Mob
	bd := reach
	for _, m := range wd.mobs {
		if !m.Alive || now.Sub(m.bornAt) < mobRise || m.Kind == mobDummy && p.Mode != modeAggressive {
			continue
		}
		if d := dist(m.X, m.Y, float64(p.X), float64(p.Y)); d < bd && wd.clearLine(float64(p.X), float64(p.Y)-1, m.X, m.Y-1) {
			best, bd = m, d
		}
	}
	if best == nil {
		return
	}
	a := math.Atan2(best.Y-float64(p.Y), best.X-float64(p.X))
	wd.use(p, 0, &a, now)
}

// ---------- caminhar até um ponto ----------

type goal struct {
	x, y       int
	path       [][2]int
	at         int
	lastX      int
	lastY      int
	stillSince time.Time
	retried    bool
}

// SetGoal manda o jogador caminhar até (x, y) do mesmo andar.
func (wd *World) SetGoal(id, x, y int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.boat {
		return
	}
	now := wd.clock()
	if (y >= overH) != (p.Y >= overH) {
		wd.pop(p, "Esse ponto é em outro andar: procure a escada", 0xc0a070, float64(p.X), float64(p.Y)-34, now)
		return
	}
	path := wd.coarsePath(p.X, p.Y, x, y)
	if path == nil {
		wd.pop(p, "Não achei caminho até lá", 0xc0a070, float64(p.X), float64(p.Y)-34, now)
		p.goal = nil
		return
	}
	p.goal = &goal{x: x, y: y, path: path, lastX: p.X, lastY: p.Y, stillSince: now}
}

// CancelGoal para de caminhar.
func (wd *World) CancelGoal(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok {
		p.goal = nil
	}
}

// driveGoal põe a direção do próximo ponto no jogador (a cada tick).
func (wd *World) driveGoal(p *Player, now time.Time) {
	g := p.goal
	if g == nil {
		return
	}
	for g.at < len(g.path) && abs(g.path[g.at][0]-p.X) <= 2 && abs(g.path[g.at][1]-p.Y) <= 2 {
		g.at++
	}
	if g.at >= len(g.path) {
		p.goal, p.moveX, p.moveY = nil, 0, 0
		wd.pop(p, "Chegou", 0x9ae070, float64(p.X), float64(p.Y)-34, now)
		return
	}
	wp := g.path[g.at]
	dx, dy := wp[0]-p.X, wp[1]-p.Y
	p.moveX, p.moveY = sgn(dx, 1), sgn(dy, 1)
	if abs(p.X-g.lastX)+abs(p.Y-g.lastY) > 0 {
		g.lastX, g.lastY, g.stillSince = p.X, p.Y, now
	} else if now.Sub(g.stillSince) > 1500*time.Millisecond {
		if g.retried {
			p.goal, p.moveX, p.moveY = nil, 0, 0
			wd.pop(p, "O caminho está bloqueado", 0xc0a070, float64(p.X), float64(p.Y)-34, now)
			return
		}
		g.retried, g.stillSince = true, now
		if path := wd.coarsePath(p.X, p.Y, g.x, g.y); path != nil {
			g.path, g.at = path, 0
		}
	}
	if dist(float64(p.X), float64(p.Y), float64(wp[0]), float64(wp[1])) > 60 {
		p.goal, p.moveX, p.moveY = nil, 0, 0 // passou por um portal ou foi levado longe
	}
}

func sgn(v, dead int) int {
	switch {
	case v > dead:
		return 1
	case v < -dead:
		return -1
	}
	return 0
}

// coarsePath acha um caminho numa grade de 4 em 4 células (A*, 8 vizinhos,
// até 150 mil nós); cada nó tem de caber o personagem e o trecho entre dois
// nós vizinhos tem de ser andável.
func (wd *World) coarsePath(sx, sy, gx, gy int) [][2]int {
	const step = 4
	free := func(x, y int) bool { return wd.fits(x, y) }
	snap := func(x, y int) (int, int, bool) { // nó livre mais perto
		cx, cy := x/step*step+step/2, y/step*step+step/2
		for r := 0; r <= 3; r++ {
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					if nx, ny := cx+dx*step, cy+dy*step; free(nx, ny) {
						return nx, ny, true
					}
				}
			}
		}
		return 0, 0, false
	}
	ax, ay, ok1 := snap(sx, sy)
	bx, by, ok2 := snap(gx, gy)
	if !ok1 || !ok2 {
		return nil
	}
	key := func(x, y int) int { return y/step*(wd.w/step+1) + x/step }
	h := func(x, y int) float64 {
		dx, dy := math.Abs(float64(x-bx)), math.Abs(float64(y-by))
		return dx + dy - 0.59*math.Min(dx, dy)
	}
	g := map[int]float64{key(ax, ay): 0}
	from := map[int][2]int{}
	closed := map[int]bool{}
	open := &pathHeap{}
	type xy struct{ x, y int }
	pos := map[int]xy{key(ax, ay): {ax, ay}}
	heap.Push(open, &pathNode{i: key(ax, ay), f: h(ax, ay)})
	found := false
	for n := 0; open.Len() > 0 && n < 150000; n++ {
		cur := heap.Pop(open).(*pathNode)
		if closed[cur.i] {
			continue
		}
		closed[cur.i] = true
		c := pos[cur.i]
		if c.x == bx && c.y == by {
			found = true
			break
		}
		for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			nx, ny := c.x+d[0]*step, c.y+d[1]*step
			if !free(nx, ny) || !free(c.x+d[0]*step/2, c.y+d[1]*step/2) || (ny >= overH) != (c.y >= overH) {
				continue
			}
			if d[0] != 0 && d[1] != 0 && (!free(c.x+d[0]*step, c.y) || !free(c.x, c.y+d[1]*step)) {
				continue
			}
			k := key(nx, ny)
			if closed[k] {
				continue
			}
			cost := 1.0
			if d[0] != 0 && d[1] != 0 {
				cost = 1.41
			}
			ng := g[cur.i] + cost
			if old, ok := g[k]; ok && old <= ng {
				continue
			}
			g[k], from[k], pos[k] = ng, [2]int{c.x, c.y}, xy{nx, ny}
			heap.Push(open, &pathNode{i: k, f: ng + h(nx, ny)})
		}
	}
	if !found {
		return nil
	}
	var rev [][2]int
	for x, y := bx, by; !(x == ax && y == ay); {
		rev = append(rev, [2]int{x, y})
		f := from[key(x, y)]
		x, y = f[0], f[1]
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	if wd.fits(gx, gy) {
		rev = append(rev, [2]int{gx, gy})
	}
	return rev
}
