package main

import (
	"math"
	"time"
)

// ---------- terreno que muda ----------
// Árvores, rochas e paredes rachadas quebram e voltam com o tempo. Cada
// mudança vira um lote de células numeradas por versão; o cliente recebe só
// as versões que ainda não tem.

const (
	flagSolid  = 1
	flagOver   = 2
	flagWater  = 4
	flagBog    = 8
	flagPoison = 16
	flagLava   = 128 // lava: queima, brilha

	maxEdits = 512 // lotes guardados; quem ficou mais atrás baixa o mapa de novo
)

type cellEdit struct {
	I  int
	Ch byte
	C  rgb
	F  byte
}

type mapEdit struct {
	Ver   int
	Cells []cellEdit
}

func (wd *World) flagsAt(i int) byte {
	var f byte
	if wd.solid[i] {
		f |= flagSolid
	}
	if wd.over[i] {
		f |= flagOver
	}
	if wd.water[i] {
		f |= flagWater
	}
	if wd.burn != nil && wd.burn[i] {
		f |= flagFire
	}
	if wd.glow != nil && wd.glow[i] {
		f |= flagGlow
	}
	switch wd.marsh[i] {
	case 1:
		f |= flagBog
	case 2:
		f |= flagBog | flagPoison
	case lavaMarsh:
		f |= flagLava | flagGlow
	}
	return f
}

func (wd *World) cellAt(i int) cellEdit {
	return cellEdit{i, wd.glyph[i], wd.color[i], wd.flagsAt(i)}
}

func (wd *World) setCell(e cellEdit) {
	i := e.I
	wd.glyph[i], wd.color[i] = e.Ch, e.C
	wd.solid[i], wd.over[i], wd.water[i] = e.F&flagSolid != 0, e.F&flagOver != 0, e.F&flagWater != 0
	wd.marsh[i] = 0
	if e.F&flagPoison != 0 {
		wd.marsh[i] = 2
	} else if e.F&flagBog != 0 {
		wd.marsh[i] = 1
	}
}

// edit aplica o lote e o registra para os clientes. Chamar com o lock.
func (wd *World) edit(cells []cellEdit) {
	for _, c := range cells {
		wd.setCell(c)
	}
	wd.mapVer++
	wd.edits = append(wd.edits, mapEdit{wd.mapVer, cells})
	if len(wd.edits) > maxEdits {
		wd.edits = wd.edits[len(wd.edits)-maxEdits:]
	}
}

// editsSince junta as células mudadas depois da versão v. reload diz que o
// cliente ficou para trás demais e precisa baixar o mapa inteiro.
func (wd *World) editsSince(v int) (cells []cellEdit, ver int, reload bool) {
	if v >= wd.mapVer {
		return nil, wd.mapVer, false
	}
	if len(wd.edits) == 0 || wd.edits[0].Ver > v+1 {
		return nil, wd.mapVer, true
	}
	for _, e := range wd.edits {
		if e.Ver > v {
			cells = append(cells, e.Cells...)
		}
	}
	return cells, wd.mapVer, false
}

// ---------- objetos quebráveis ----------

const (
	propTree = iota
	propRock
	propCrack // parede rachada que esconde a sala secreta
	propDoor  // porta trancada de cofre: abre com chave e volta a trancar
	propSeal  // porta selada: abre com a palavra ou a sequência certa
	propPrism // prisma das Cavernas de Cristal (cristais.go)
)

var propDefs = []struct {
	Name    string
	HP      int
	Pad     float64       // folga do alvo
	Drop    int           // material que larga
	Respawn time.Duration // 0 = não volta
}{
	propTree:  {"Árvore", 90, 4, itemWood, 4 * time.Minute},
	propRock:  {"Rocha", 140, 5, itemStone, 5 * time.Minute},
	propCrack: {"Parede rachada", 160, 6, itemStone, 0},
	propDoor:  {"Porta trancada", 1 << 30, 0, 0, 3 * time.Minute},
	propSeal:  {"Porta selada", 1 << 30, 0, 0, 3 * time.Minute},
	propPrism: {"Prisma", 120, 4, itemStone, 25 * time.Second},
}

type prop struct {
	Kind, X, Y int
	HP         int
	Alive      bool
	respawnAt  time.Time
	on, off    []cellEdit // células de pé e quebrado
}

// stamp desenha algo dentro do retângulo e devolve o que mudou: como ficou
// (on) e como era (off). Serve para transformar um desenho em objeto quebrável.
func (wd *World) stamp(x0, y0, x1, y1 int, draw func()) (on, off []cellEdit) {
	x0, y0, x1, y1 = max(0, x0), max(0, y0), min(wd.w-1, x1), min(wd.h-1, y1)
	before := make([]cellEdit, 0, (x1-x0+1)*(y1-y0+1))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			before = append(before, wd.cellAt(y*wd.w+x))
		}
	}
	draw()
	for _, b := range before {
		if a := wd.cellAt(b.I); a != b {
			on, off = append(on, a), append(off, b)
		}
	}
	return
}

func (wd *World) addProp(kind, x, y int, on, off []cellEdit) {
	if len(on) == 0 {
		return
	}
	wd.props = append(wd.props, &prop{Kind: kind, X: x, Y: y, HP: propDefs[kind].HP, Alive: true, on: on, off: off})
}

// settleProps descarta das peças as células que outra construção desenhou
// por cima depois: quebrar a árvore não pode apagar a casa ao lado.
func (wd *World) settleProps() {
	for _, p := range wd.props {
		keepOn, keepOff := p.on[:0], p.off[:0]
		for k, c := range p.on {
			if wd.cellAt(c.I) == c {
				keepOn, keepOff = append(keepOn, c), append(keepOff, p.off[k])
			}
		}
		p.on, p.off = keepOn, keepOff
	}
}

// hitProps fere os objetos quebráveis dentro da forma. Chamar com o lock.
func (wd *World) hitProps(owner int, now time.Time, dmg int, in shape) {
	for _, pr := range wd.props {
		if !pr.Alive || pr.Kind == propDoor || pr.Kind == propSeal || !in(float64(pr.X), float64(pr.Y), propDefs[pr.Kind].Pad) {
			continue
		}
		hit := dmg
		if pr.Kind == propTree {
			hit *= wd.players[owner].chopMult() // machado derruba árvore muito mais rápido
		}
		if pr.HP -= hit; pr.HP > 0 {
			continue
		}
		def := propDefs[pr.Kind]
		pr.Alive, pr.respawnAt = false, now.Add(def.Respawn)
		wd.edit(pr.off)
		wd.effects = append(wd.effects, Effect{Kind: fxPuff, X: float64(pr.X), Y: float64(pr.Y), R: 10, Start: now, Dur: 600 * time.Millisecond})
		n := 1 + wd.rng.Intn(3)
		if pr.Kind == propTree && wd.players[owner].chopMult() > 1 {
			n += 2
		}
		for k := 0; k < n; k++ {
			wd.drop(def.Drop, float64(pr.X), float64(pr.Y), 0, now)
		}
		if pr.Kind == propPrism {
			wd.prismsBroken(wd.players[owner], now)
		}
		if pr.Kind == propCrack {
			if p := wd.players[owner]; p != nil {
				wd.say(ChatLine{Text: p.Name + " abriu uma passagem escondida!"})
			}
		}
	}
}

// stepProps põe de volta o que já pode renascer, se ninguém estiver em cima.
func (wd *World) stepProps(now time.Time) {
	for _, pr := range wd.props {
		if pr.Alive || propDefs[pr.Kind].Respawn == 0 || now.Before(pr.respawnAt) {
			continue
		}
		blocked := false
		for _, p := range wd.players {
			if dist(float64(p.X), float64(p.Y), float64(pr.X), float64(pr.Y)) < 8 {
				blocked = true
				break
			}
		}
		if blocked {
			pr.respawnAt = now.Add(10 * time.Second)
			continue
		}
		pr.Alive, pr.HP = true, propDefs[pr.Kind].HP
		wd.edit(pr.on)
	}
}

// ---------- ervas ----------

const (
	herbAntidote = iota // pântano: cura veneno
	herbEmber           // gelo: aquece
	herbAloe            // deserto: refresca
)

var herbItem = []int{itemAntidote, itemEmberRoot, itemAloe}

const (
	herbReach   = 7.0
	herbRespawn = 2 * time.Minute
)

type Herb struct {
	ID, Kind, X, Y int
	Alive          bool
	respawnAt      time.Time
}

// plant espalha n ervas do tipo em chão livre (e fora do charco) em volta do ponto.
func (wd *World) plant(kind, cx, cy, r, n int, seed uint32) {
	for i, placed := 0, 0; i < n*60 && placed < n; i++ {
		h := hash2(i, kind, seed)
		x, y := cx+int(h%uint32(2*r))-r, cy+int(h>>12%uint32(2*r))-r
		if dist(float64(x), float64(y), float64(cx), float64(cy)) > float64(r) || !wd.fits(x, y) || wd.marsh[y*wd.w+x] != 0 {
			continue
		}
		wd.herbs = append(wd.herbs, &Herb{ID: len(wd.herbs) + 1, Kind: kind, X: x, Y: y, Alive: true})
		placed++
	}
}

// ---------- status negativos ----------

const (
	stPoison = iota
	stCold
	stHeat
	numStatus
)

var statusNames = [numStatus]string{"Envenenado", "Congelando", "Insolação"}

// dano por segundo de cada status
var statusDmg = [numStatus]int{3, 2, 2}

// mobInflict é o status que o golpe de cada inimigo pode deixar (35% das vezes).
var mobInflict = map[int]int{mobSpider: stPoison, mobTroll: stPoison, mobScarab: stPoison,
	mobIceWolf: stCold, mobYeti: stCold, mobFrostGiant: stCold, mobMummy: stHeat, mobPharaoh: stHeat,
	mobMagmaGolem: stHeat, mobIgnar: stHeat, mobFungusHusk: stPoison,
	mobSandScorpion: stPoison, mobIceWraith: stCold, mobRootHorror: stPoison, mobVulkhar: stHeat}

func (wd *World) afflict(p *Player, s int, d time.Duration, now time.Time) {
	if now.Before(p.immune[s]) {
		return
	}
	if until := now.Add(d); until.After(p.status[s]) {
		p.status[s] = until
	}
}

// nearFire diz se o jogador está aquecido por fogueira ou tocha.
func (wd *World) nearFire(p *Player) bool {
	for _, l := range wd.lights {
		if l.Kind < 2 && dist(float64(p.X), float64(p.Y), float64(l.X), float64(l.Y)) < float64(l.R)*0.7 {
			return true
		}
	}
	return false
}

// environ aplica, uma vez por segundo, o que o bioma faz com o jogador e o
// dano dos status ativos.
func (wd *World) environ(p *Player, now time.Time) {
	wd.lavaStep(p, now)
	wd.abyssStep(p, now)
	wd.gateFree(p, now)
	px, py := float64(p.X), float64(p.Y)
	if p.Y < overH {
		tod := wd.timeOfDay(now.Sub(wd.start).Seconds())
		if wd.marsh[p.Y*wd.w+p.X] == 2 {
			wd.afflict(p, stPoison, 6*time.Second, now)
		}
		if snowDist(px, py) < 0.95 && !wd.nearFire(p) {
			wd.afflict(p, stCold, 3*time.Second, now)
		}
		if desertDist(px, py) < 0.95 && tod > 0.3 && tod < 0.7 && !p.boat && wd.weather == wxClear {
			wd.afflict(p, stHeat, 3*time.Second, now)
		}
	}
	for s := 0; s < numStatus; s++ {
		if now.Before(p.status[s]) {
			wd.hurtRaw(p, statusDmg[s], "por "+[]string{"veneno", "frio", "calor"}[s], now)
		}
	}
}

// ---------- montarias ----------

const (
	steedHorse = iota
	steedBoar
	steedOstrich
	steedRaptor
)

var steedKinds = []struct {
	Name  string
	Level int // nível para domar
	Speed int // células a mais por tick
	Def   int // % de proteção montado
}{
	steedHorse:   {"Cavalo", 1, 2, 0},
	steedBoar:    {"Javali", 3, 1, 10},
	steedOstrich: {"Avestruz", 5, 2, 0},
	steedRaptor:  {"Velociraptor", 8, 3, 5},
}

const (
	steedReach   = 9.0
	steedRespawn = 2 * time.Minute
)

// Steed é um bicho selvagem que dá para domar. Domado, some do mapa por um
// tempo e passa a ser do jogador (Player.Steeds).
type Steed struct {
	ID, Kind  int
	X, Y      float64
	Alive     bool
	homeX     float64
	homeY     float64
	tx, ty    float64
	moveAt    time.Time
	respawnAt time.Time
}

func (wd *World) herd(kind, cx, cy, r, n int, seed uint32) {
	for i, placed := 0, 0; i < n*60 && placed < n; i++ {
		h := hash2(i, kind+40, seed)
		x, y := cx+int(h%uint32(2*r))-r, cy+int(h>>12%uint32(2*r))-r
		if dist(float64(x), float64(y), float64(cx), float64(cy)) > float64(r) || !wd.fits(x, y) {
			continue
		}
		wd.steeds = append(wd.steeds, &Steed{ID: len(wd.steeds) + 1, Kind: kind, X: float64(x), Y: float64(y), Alive: true,
			homeX: float64(x), homeY: float64(y), tx: float64(x), ty: float64(y)})
		placed++
	}
}

func (wd *World) stepSteeds(now time.Time) {
	for _, s := range wd.steeds {
		if !s.Alive {
			if now.After(s.respawnAt) {
				s.X, s.Y, s.Alive = s.homeX, s.homeY, true
			}
			continue
		}
		if now.After(s.moveAt) {
			s.tx, s.ty = s.homeX+float64(wd.rng.Intn(41)-20), s.homeY+float64(wd.rng.Intn(25)-12)
			s.moveAt = now.Add(time.Duration(3000+wd.rng.Intn(4000)) * time.Millisecond)
		}
		dx, dy := s.tx-s.X, s.ty-s.Y
		if d := math.Hypot(dx, dy); d > 1 {
			nx, ny := s.X+dx/d*0.35, s.Y+dy/d*0.35
			if wd.fits(int(nx), int(ny)) {
				s.X, s.Y = nx, ny
			}
		}
	}
}

// speedBonus é o que a montaria soma ao passo do jogador.
func (p *Player) speedBonus() int {
	if p.Riding == 0 || p.boat {
		return 0
	}
	return steedKinds[p.Riding-1].Speed
}

// Ride monta ou desmonta. Montar usa o último bicho montado (ou o primeiro domado).
func (wd *World) Ride(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.boat {
		return
	}
	if p.Riding != 0 {
		p.Riding = 0
		return
	}
	k := p.LastSteed
	if p.Steeds&(1<<k) == 0 {
		k = -1
		for i := range steedKinds {
			if p.Steeds&(1<<i) != 0 {
				k = i
			}
		}
	}
	if k >= 0 {
		p.Riding, p.LastSteed = k+1, k
	}
}

// ---------- barcos ----------

const (
	boatSpeed = 3
	raftWood  = 6 // madeira para montar uma jangada em qualquer margem
	dockReach = 10.0
	moorReach = 16.0 // distância do barco atracado para embarcar de novo
	doorReach = 9.0  // distância da porta de cofre para destrancar
)

func (wd *World) wet(x, y int) bool {
	return wd.inBounds(x, y) && wd.water[y*wd.w+x]
}

// floats diz se o barco cabe com o casco em (x, y).
func (wd *World) floats(x, y int) bool {
	return wd.wet(x-2, y) && wd.wet(x, y) && wd.wet(x+2, y)
}

// sail anda de barco; ao encostar em terra, o jogador desce.
func (wd *World) sail(p *Player, dx, dy, n int) {
	p.FaceX, p.FaceY = dx, dy
	for ; n > 0 && p.boat; n-- {
		for _, d := range [][2]int{{dx, 0}, {0, dy}} {
			if d[0] == 0 && d[1] == 0 {
				continue
			}
			nx, ny := p.X+d[0], p.Y+d[1]
			if !wd.canEnter(p, nx, ny, wd.clock()) {
				continue
			}
			if wd.floats(nx, ny) {
				p.X, p.Y = nx, ny
			} else if x, y, ok := wd.landing(nx, ny, d[0], d[1]); ok {
				// O barco fica atracado onde estava, para voltar com ele
				// (sem isso, quem descia na ilha ficava preso).
				p.Moored, p.MoorX, p.MoorY = true, p.X, p.Y
				p.X, p.Y, p.boat = x, y, false
				break
			}
		}
	}
}

// landing acha onde descer quando o casco bate na margem: a beira raramente
// tem chão livre bem na frente, então procura alguns passos adiante e de lado.
func (wd *World) landing(x, y, dx, dy int) (int, int, bool) {
	for k := 0; k < 5; k++ {
		for _, off := range []int{0, -1, 1, -2, 2, -3, 3} {
			cx, cy := x+dx*k, y+dy*k
			if dx != 0 {
				cy += off
			} else {
				cx += off
			}
			if wd.fits(cx, cy) {
				return cx, cy, true
			}
		}
	}
	return 0, 0, false
}

// launch põe o jogador no barco, na água mais próxima de (x, y) até r células.
func (wd *World) launch(p *Player, x, y int, r int) bool {
	for d := 0; d <= r; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if max(abs(dx), abs(dy)) == d && wd.floats(x+dx, y+dy) {
					p.X, p.Y, p.boat, p.Riding, p.Moored = x+dx, y+dy, true, 0, false
					return true
				}
			}
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ---------- interação (tecla E) ----------

// Interact faz a ação mais útil por perto: domar montaria, colher erva,
// embarcar no cais ou montar uma jangada na margem.
func (wd *World) Interact(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	now := wd.clock()
	p, ok := wd.players[id]
	if !ok || p.boat {
		return
	}
	px, py := float64(p.X), float64(p.Y)
	for i, b := range wd.bonfires {
		if dist(px, py, float64(b.X), float64(b.Y)) <= bonfireReach {
			wd.rest(p, i, now)
			return
		}
	}
	for i, s := range wd.steeds {
		if !s.Alive || dist(px, py, s.X, s.Y) > steedReach {
			continue
		}
		wd.startTame(p, i, now) // nível, fôlego e a sequência de setas (doma.go)
		return
	}
	for _, h := range wd.herbs {
		if !h.Alive || dist(px, py, float64(h.X), float64(h.Y)) > herbReach {
			continue
		}
		if p.give(herbItem[h.Kind], 1) {
			h.Alive, h.respawnAt = false, now.Add(herbRespawn)
		}
		return
	}
	if p.Moored && dist(px, py, float64(p.MoorX), float64(p.MoorY)) <= moorReach && wd.launch(p, p.MoorX, p.MoorY, 6) {
		return
	}
	// Porta de cofre: abre com a Chave de Ferro (que fica na fechadura).
	for _, pr := range wd.props {
		if pr.Kind != propDoor || !pr.Alive || dist(px, py, float64(pr.X), float64(pr.Y)) > doorReach {
			continue
		}
		if p.count(itemIronKey) == 0 {
			wd.pop(p, "Trancada. O ferreiro da vila vende a Chave de Ferro.", 0xd0a060, px, py-34, now)
			return
		}
		p.take(itemIronKey)
		pr.Alive, pr.respawnAt = false, now.Add(propDefs[propDoor].Respawn)
		wd.edit(pr.off)
		wd.say(ChatLine{Text: p.Name + " destrancou um cofre."})
		return
	}
	for _, d := range docks {
		if dist(px, py, float64(d[0]), float64(d[1])) <= dockReach && wd.launch(p, d[0], d[1], 14) {
			return
		}
	}
	if p.count(itemWood) >= raftWood && wd.launch(p, p.X, p.Y, 6) {
		for k := 0; k < raftWood; k++ {
			p.take(itemWood)
		}
		wd.say(ChatLine{Text: p.Name + " montou uma jangada."})
	}
}

// stepHerbs faz as ervas colhidas brotarem de novo.
func (wd *World) stepHerbs(now time.Time) {
	for _, h := range wd.herbs {
		if !h.Alive && now.After(h.respawnAt) {
			h.Alive = true
		}
	}
}

// ---------- clima ----------

const (
	wxClear = iota
	wxRain
	wxStorm
	wxFog // neblina: a visão encurta
)

const (
	boltDmg    = 22
	boltRadius = 7.0
)

// stepWeather roda uma vez por segundo: troca o tempo de vez em quando e,
// na tempestade, solta raios perto de quem está ao ar livre.
func (wd *World) stepWeather(now time.Time) {
	if wd.wxForced >= 0 {
		wd.weather = wd.wxForced
	} else if now.After(wd.wxUntil) {
		switch roll := wd.rng.Intn(100); {
		case roll < 50:
			wd.weather = wxClear
		case roll < 72:
			wd.weather = wxRain
		case roll < 86:
			wd.weather = wxStorm
		default:
			wd.weather = wxFog
		}
		wd.wxUntil = now.Add(time.Duration(180+wd.rng.Intn(180)) * time.Second)
	}
	if wd.weather != wxStorm || wd.rng.Intn(100) >= 30 {
		return
	}
	// Escolhe alguém ao ar livre e fora do deserto (lá não chove).
	var out []*Player
	for _, p := range wd.players {
		if p.Y < overH && desertDist(float64(p.X), float64(p.Y)) >= 1 {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return
	}
	p := out[wd.rng.Intn(len(out))]
	x, y, tree := wd.boltTarget(p, now)
	if tree {
		return // caiu numa árvore: o fogo já foi aceso
	}
	wd.effects = append(wd.effects, Effect{Kind: fxBolt, X: x, Y: y, R: boltRadius, Start: now, Dur: 700 * time.Millisecond})
	for _, q := range wd.players {
		if dist(float64(q.X), float64(q.Y), x, y) <= boltRadius {
			wd.hurtRaw(q, boltDmg, "por um raio", now)
		}
	}
}

// PickSteed escolhe a montaria k entre as domadas e já monta nela.
func (wd *World) PickSteed(id, k int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.boat || k < 0 || k >= len(steedKinds) || p.Steeds&(1<<k) == 0 {
		return
	}
	p.LastSteed, p.Riding = k, k+1
}

// ReleaseSteed solta a montaria k: ela deixa de ser sua e volta para a natureza
// ali mesmo (se o jogador estiver nela, desce antes).
func (wd *World) ReleaseSteed(id, k int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || k < 0 || k >= len(steedKinds) || p.Steeds&(1<<k) == 0 {
		return
	}
	if p.Riding == k+1 {
		p.Riding = 0
	}
	p.Steeds &^= 1 << k
	now := wd.clock()
	wd.say(ChatLine{Text: p.Name + " soltou o " + steedKinds[k].Name + "."})
	wd.effects = append(wd.effects, Effect{Kind: fxPuff, X: float64(p.X) + 8, Y: float64(p.Y), R: 8, Start: now, Dur: 500 * time.Millisecond})
}
