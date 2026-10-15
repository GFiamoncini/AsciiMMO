package main

import (
	"asciimmo/proto"

	"context"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

const (
	maxChatLines = 50
	maxChatLen   = 120
	maxNameLen   = 12

	tickRate  = 20 // ticks por segundo
	walkSpeed = 2  // células por tick

	bossMaxHP   = 9000
	bossSpeed   = 0.35 // células por tick
	aggroRange  = 140.0
	stompRadius = 26.0

	mobAggro    = 70.0
	mobLeash    = 120.0 // distância da cova em que o esqueleto desiste
	mobReach    = 8.0
	mobRise     = 800 * time.Millisecond // tempo saindo da cova
	mobCooldown = 1200 * time.Millisecond
)

type Player struct {
	paused       bool     // pausado: parado e protegido (pausa.go)
	tame         *tameTry // doma em andamento (doma.go)
	saves        int      // quantas vezes foi gravado nesta sessão (o cliente mostra "salvo")
	ID           int
	Name         string
	X, Y         int
	Color        int
	HP, MaxHP    int
	MP, MaxMP    int
	Level, Gold  int
	FaceX, FaceY int

	Class    int
	Unlocked bool // já desceu à cripta: o terceiro golpe está liberado
	Equip    int  // arma equipada (0 = nenhuma)
	Quest    int  // índice da missão atual
	QProg    int
	QStep    int // etapa (subobjetivo) da missão principal
	inv      []invSlot

	XP, Points, SkillPts int // experiência e pontos de atributo e de árvore a gastar
	Attr                 [numAttrs]int
	Web                  []int         // graus de cada nó da teia de habilidades (skills.go)
	Bar                  [barSlots]int // golpe em cada atalho 1 a 5 (-1 vazio)
	bar0                 bool          // a barra já foi montada (save ou padrão)
	bon                  bonus         // soma das passivas aprendidas
	Skin                 int           // armadura equipada (0 = nenhuma)
	Helm, Boots, Amulet  int           // demais peças equipadas
	ST                   float64
	MaxST                int
	Kills, Chests        int
	Bosses, Seen, Ach    uint   // bits: chefes derrotados, regiões vistas, conquistas
	Slain                uint64 // bits por tipo de inimigo: chefes que já derrubou (fecham missões atrasadas)

	SideProg [numSide]int // progresso das missões secundárias
	SideDone uint         // bits das secundárias concluídas (selos quebrados)
	Bonfire  int          // última fogueira acesa: é onde renasce

	pops      []pop
	sealAt    time.Time // último aviso de selo, para não repetir
	Riding    int       // montaria em uso + 1 (0 = a pé)
	LastSteed int       // última montaria usada
	Steeds    uint      // bits das montarias domadas

	boat       bool
	Moored     bool      // deixou o barco atracado onde desceu
	Bag        int       // espaços comprados além dos de início
	Found      uint64    // pontos de viagem descobertos
	Intro      int       // etapa do prólogo (intro.go); len(prologue) = concluído
	Tech       int       // técnicas aprendidas com o Mestre Oren (bits)
	eggReady   time.Time // quando o Ovo Escarlate pode chorar de novo (divina.go)
	Gates      uint32    // Portões Rúnicos acesos (bits; portoes.go)
	Light      int       // luz acesa: 0 nada, 1 tocha, 2 lanterna (tools.go)
	LightSlot  int       // lanterna ou tocha equipada no espaço de luz (luz.go)
	House      int       // casa comprada + 1 (casa.go)
	Stash      []invSlot // baú da casa
	StashGold  int       // dinheiro no baú
	autoLit    bool      // a luz acesa foi acesa sozinha ao descer
	wasUnder   bool      // estava no subsolo no último segundo
	torchUntil time.Time // quando a tocha acesa acaba
	fog        []byte    // o que já explorou (explore.go)
	MoorX      int
	MoorY      int
	status     [numStatus]time.Time // até quando cada status negativo dura
	immune     [numStatus]time.Time // até quando a erva protege
	sprint     bool
	chestDay   map[int]int // baú → dia do jogo em que foi aberto
	token      string      // hash do segredo do cliente; vazio não salva
	tokens     []string    // segredos de todos os aparelhos que já o abriram
	password   string      // hash da senha, se houver

	moveX, moveY       int // direção segurada pelo cliente web
	hurtAt, guardUntil time.Time
	ready              [maxSkills]time.Time // quando cada golpe volta a ficar disponível
	dmgUntil           time.Time            // Grito de Guerra: +30% de dano
	critUntil          time.Time            // Olho de Águia: +25% de crítico
	venomUntil         time.Time            // Lâminas Envenenadas
	act                int                  // último golpe usado, para a animação
	actAt              time.Time
	actDir             float64
}

type Boss struct {
	X, Y      float64
	HP, MaxHP int
	Alive     bool
	RespawnIn time.Duration // preenchido no Snapshot

	armed          bool
	stompAt        time.Time
	nextStomp      time.Time
	respawnAt      time.Time
	hurtAt         time.Time
	stompX, stompY float64
}

// Mob é um esqueleto do cemitério. Morto, espera na cova até renascer.
type Mob struct {
	ID, Kind  int
	X, Y      float64
	HP, MaxHP int
	Alive     bool

	homeX, homeY, wanderX, wanderY             float64
	zone                                       int  // fogueira da área (-1 = nenhuma)
	temp                                       bool // invasor: não renasce
	bornAt, hurtAt, atkAt, respawnAt, wanderAt time.Time

	armed bool // pisão de chefe armado

	slowUntil, stunUntil time.Time // lentidão e atordoamento/congelamento
	dotUntil, dotNext    time.Time // dano contínuo (queimadura, veneno)
	dotDmg, dotOwner     int
	dotElem              int
	stompAt, nextStomp   time.Time
	stompX, stompY       float64
	nav                  *mobNav // caminho em volta das paredes (caminho.go)
}

const (
	fxSlash      = proto.FxSlash
	fxNova       = proto.FxNova
	fxWarn       = proto.FxWarn
	fxBlast      = proto.FxBlast
	fxPuff       = proto.FxPuff
	fxThrust     = proto.FxThrust
	fxSpin       = proto.FxSpin
	fxBoom       = proto.FxBoom
	fxHeal       = proto.FxHeal
	fxHoly       = proto.FxHoly
	fxMeteor     = proto.FxMeteor
	fxRain       = proto.FxRain
	fxDark       = proto.FxDark
	fxBolt       = proto.FxBolt
	fxChain      = proto.FxChain
	fxFrost      = proto.FxFrost
	fxFlame      = proto.FxFlame
	fxZap        = proto.FxZap
	fxBlink      = proto.FxBlink
	fxBuff       = proto.FxBuff
	fxSmoke      = proto.FxSmoke
	fxBone       = proto.FxBone
	fxPlague     = proto.FxPlague
	fxBlizzard   = proto.FxBlizzard
	fxCrack      = proto.FxCrack
	fxConsecrate = proto.FxConsecrate
	fxHammer     = proto.FxHammer
	fxGrave      = proto.FxGrave
	fxWhip       = proto.FxWhip
	fxStorm      = proto.FxStorm
	fxSeal       = proto.FxSeal
	fxOblit      = proto.FxOblit
	fxGrass      = proto.FxGrass
)

type Effect struct {
	Kind  int
	X, Y  float64
	R     float64
	Dir   float64
	Start time.Time
	Dur   time.Duration
	P     float64 // progresso 0..1, preenchido no Snapshot
}

// ChatLine com From vazio é mensagem do sistema.
type ChatLine struct {
	From  string
	Color int
	Text  string
}

type Snapshot struct {
	Me      Player
	Players []Player
	Boss    Boss
	Mobs    []Mob
	Projs   []Proj
	Loot    []Loot
	Steeds  []Steed
	Herbs   []Herb
	Effects []Effect
	Chat    []ChatLine
	Now     time.Time
	T       float64 // segundos desde o início do mundo, para animações
	Tod     float64 // hora do dia, 0..1 (0 = meia-noite)
	Weather int
	Invaded bool        // invasão noturna em curso
	Day     int         // dia do jogo, a partir de 1
	Minions []proto.Mob // servos dos necromantes, já prontos para o cliente
	Frozen  bool        // mundo parado: todos os jogadores pausaram
}

// World é o estado autoritativo do jogo. O terreno não muda depois de
// gerado, então pode ser lido sem lock; o resto passa pelo mutex.
type World struct {
	pathBudget int          // buscas de caminho que ainda cabem neste passo (caminho.go)
	pausedNs   atomic.Int64 // tempo em que o mundo ficou parado (pausa.go)
	w, h       int
	glyph      []byte
	color      []rgb
	solid      []bool
	over       []bool // célula desenhada por cima dos personagens (copas)
	marsh      []byte // charco: 1 lama (anda devagar), 2 venenoso
	glow       []bool // célula que emite luz (flagGlow)
	elev       []float32
	saveDir    string  // pasta dos personagens salvos; vazia desliga
	water      []bool  // mar e poças: o cliente anima
	dayLen     float64 // duração do dia, em segundos
	spawns     []spawnPt
	lights     []light
	portals    []portal
	start      time.Time

	mu            sync.RWMutex
	mapVer        int
	mainSpots     [][]questSpot // onde fazer cada etapa de cada missão (questmap.go)
	sideSpots     []questSpot
	signs, travel []travelPt // placas de direção e pontos de viagem rápida (travel.go)
	hazards       []hazard   // efeitos que duram: muralha de fogo, nevasca... (skills.go)
	minions       []*minion  // servos erguidos pelos necromantes (minions.go)
	regrows       []regrow   // capim cortado esperando crescer (grasscut.go)
	nextMinion    int
	shops         []merchant
	zones         []zone
	burn          []bool // células em chamas
	burning       map[int]burnCell
	demon         demon        // golpes de Vulkhar (vulkhar.go)
	gates         []gate       // Portões Rúnicos (portoes.go)
	reads         []proto.Read // placas, postes e pedras que se leem com E (leituras.go)
	ashes         []ashCell
	inv           invasion
	nextMob       int
	bonfires      []bonfire
	nextPop       int
	weather       int       // wxClear, wxRain ou wxStorm
	wxUntil       time.Time // quando o tempo muda de novo
	wxForced      int       // -1 deixa o tempo variar sozinho
	todShift      float64   // dias adiantados por quem esperou na fogueira (espera.go)
	edits         []mapEdit
	props         []*prop
	herbs         []*Herb
	steeds        []*Steed
	blobMu        sync.Mutex
	blob          [2][]byte // mapa empacotado: 0 superfície e cripta, 1 Profundezas
	blobVer       [2]int
	players       map[int]*Player
	boss          Boss
	mobs          []*Mob
	projs         []Proj
	blasts        []blast
	loot          []Loot
	nextLoot      int
	nextProj      int
	effects       []Effect
	chat          []ChatLine
	nextID        int
	ticks         int
	rng           *rand.Rand
}

func NewWorld(seed int64) *World {
	wd := &World{
		w:        worldW,
		h:        worldH,
		start:    time.Now(),
		dayLen:   600,
		wxForced: -1,
		players:  map[int]*Player{},
		rng:      rand.New(rand.NewSource(seed)),
		boss:     Boss{X: lairX, Y: lairY, HP: bossMaxHP, MaxHP: bossMaxHP, Alive: true},
	}
	wd.generate(uint32(seed))
	for i, sp := range wd.spawns {
		hp := mobKinds[sp.Kind].HP
		wd.mobs = append(wd.mobs, &Mob{ID: i + 1, Kind: sp.Kind, X: float64(sp.X), Y: float64(sp.Y),
			HP: hp, MaxHP: hp, Alive: true, homeX: float64(sp.X), homeY: float64(sp.Y), bornAt: wd.start})
	}
	wd.assignZones()
	wd.buildSpots()
	wd.nextMob = len(wd.mobs)
	wd.burn, wd.burning = make([]bool, wd.w*wd.h), map[int]burnCell{}
	return wd
}

func (wd *World) inBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < wd.w && y < wd.h
}

func (wd *World) Walkable(x, y int) bool {
	return wd.inBounds(x, y) && !wd.solid[y*wd.w+x]
}

// fits diz se um personagem cabe com os pés em (x, y).
func (wd *World) fits(x, y int) bool {
	return wd.Walkable(x-2, y) && wd.Walkable(x, y) && wd.Walkable(x+2, y)
}

func (wd *World) spawnPoint() (int, int) {
	for i := 0; i < 100; i++ {
		x, y := villageX-16+wd.rng.Intn(33), villageY-10+wd.rng.Intn(21)
		if wd.fits(x, y) {
			return x, y
		}
	}
	return villageX, villageY
}

// Run avança o mundo até o contexto ser cancelado.
func (wd *World) Run(ctx context.Context) {
	t := time.NewTicker(time.Second / tickRate)
	defer t.Stop()
	var last time.Time // começo do trecho parado
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			// Todos pausados: o mundo não anda, e o tempo parado sai do relógio.
			wd.mu.RLock()
			fz := wd.frozen()
			wd.mu.RUnlock()
			if fz {
				if !last.IsZero() {
					wd.pausedNs.Add(int64(now.Sub(last)))
				}
				last = now
				continue
			}
			last = time.Time{}
			wd.step(wd.clock())
		}
	}
}

func (wd *World) step(now time.Time) {
	wd.mu.Lock()
	defer wd.mu.Unlock()

	live := wd.effects[:0]
	for _, e := range wd.effects {
		if now.Sub(e.Start) < e.Dur {
			live = append(live, e)
		}
	}
	wd.effects = live

	wd.ticks++
	wd.pathBudget = pathPerTick
	if wd.ticks%tickRate == 0 {
		wd.stepWeather(now)
		wd.stepInvasion(now)
		wd.burnDamage(now)
	}
	if wd.ticks%4 == 0 {
		wd.stepFire(now)
	}
	for _, p := range wd.players {
		if p.paused { // pausado (com outros jogando): parado, sem regenerar, só grava
			if wd.ticks%(30*tickRate) == 0 {
				wd.persist(p)
			}
			continue
		}
		if wd.ticks%tickRate == 0 {
			p.HP, p.MP = min(p.MaxHP, p.HP+2), min(p.MaxMP, p.MP+2+p.bon.MPRegen)
		}
		moving := p.moveX != 0 || p.moveY != 0
		running := moving && p.sprint && p.ST >= sprintCost
		if running {
			p.ST -= sprintCost
		} else {
			regen := stRegen * (1 + float64(p.bon.Stam)/100)
			if now.Before(p.status[stHeat]) {
				regen /= 2 // o calor do deserto cansa
			}
			p.ST = math.Min(float64(p.MaxST), p.ST+regen)
		}
		if moving {
			// Na diagonal anda menos por eixo, para a velocidade não crescer.
			n := walkSpeed + p.speedBonus()
			if running {
				n++
			}
			if p.boat {
				n = boatSpeed
			} else if wd.marsh[p.Y*wd.w+p.X] != 0 {
				n = max(1, n-2) // a lama prende o passo
			}
			if p.moveX != 0 && p.moveY != 0 && wd.ticks%2 == 0 {
				n--
			}
			if p.boat {
				wd.sail(p, p.moveX, p.moveY, n)
			} else {
				wd.walk(p, p.moveX, p.moveY, n)
				wd.enter(p)
			}
		}
		if wd.ticks%tickRate == 0 {
			if p.Y >= overH {
				p.Seen |= 1 << seenCrypt
			} else if r := regionAt(p.X, p.Y); r >= 0 {
				p.Seen |= 1 << r
			}
			wd.questEvent(p, qVisit, now)
			wd.openChests(p, now)
			wd.achieve(p, now)
			wd.environ(p, now)
			wd.tendFire(p, now)
			wd.stepLight(p, now) // tocha queima e a chuva apaga (antes nunca era chamado)
			wd.autoLight(p, now)
			wd.tameExpire(p, now)
		}
		if wd.ticks%(30*tickRate) == 0 {
			wd.persist(p)
		}
	}
	wd.stepProjs(now)
	wd.stepMinions(now)
	if wd.ticks%tickRate == 0 {
		wd.stepRegrow(now)
	}
	wd.stepLoot(now)
	wd.stepMobs(now)
	wd.stepBoss(now)
	wd.stepProps(now)
	wd.stepHerbs(now)
	wd.stepSteeds(now)
}

func (wd *World) stepBoss(now time.Time) {
	b := &wd.boss
	if !b.Alive {
		if now.After(b.respawnAt) {
			*b = Boss{X: lairX, Y: lairY, HP: bossMaxHP, MaxHP: bossMaxHP, Alive: true}
			wd.say(ChatLine{Text: "Tiamath voltou ao ninho."})
		}
		return
	}

	// Pisão armado: o chefe fica parado até a onda sair, para dar tempo de fugir.
	if b.armed {
		if now.After(b.stompAt) {
			b.armed, b.nextStomp = false, now.Add(3500*time.Millisecond)
			wd.effects = append(wd.effects, Effect{Kind: fxBlast, X: b.stompX, Y: b.stompY, R: stompRadius, Start: now, Dur: 500 * time.Millisecond})
			for _, p := range wd.players {
				if dist(float64(p.X), float64(p.Y), b.stompX, b.stompY) <= stompRadius {
					wd.hurt(p, int(float64(18+wd.rng.Intn(13))*lvScale(levelDragon, p.Level)), "por Tiamath", now)
				}
			}
		}
		return
	}

	target, best := wd.nearestFor(b.X, b.Y, now)
	if target == nil || best > aggroRange {
		wd.bossToward(lairX, lairY, 2)
		b.HP = min(b.MaxHP, b.HP+8)
		return
	}
	wd.bossToward(float64(target.X), float64(target.Y), 14)
	if best < 44 && now.After(b.nextStomp) {
		b.armed, b.stompAt, b.stompX, b.stompY = true, now.Add(time.Second), b.X, b.Y
		wd.effects = append(wd.effects, Effect{Kind: fxWarn, X: b.X, Y: b.Y, R: stompRadius, Start: now, Dur: time.Second})
	}
}

func (wd *World) nearest(x, y float64) (*Player, float64) {
	var target *Player
	best := math.MaxFloat64
	for _, p := range wd.players {
		if d := dist(float64(p.X), float64(p.Y), x, y); d < best {
			target, best = p, d
		}
	}
	return target, best
}

func (wd *World) bossToward(x, y, stop float64) {
	b := &wd.boss
	dx, dy := x-b.X, y-b.Y
	d := math.Hypot(dx, dy)
	if d <= stop {
		return
	}
	step := math.Min(bossSpeed, d-stop)
	b.X += dx / d * step
	b.Y += dy / d * step
}

func (wd *World) stepMobs(now time.Time) {
	for _, m := range wd.mobs {
		if !m.Alive {
			if !m.temp && now.After(m.respawnAt) && !wd.purged(m, now) {
				m.X, m.Y, m.HP, m.Alive, m.bornAt, m.nextStomp = m.homeX, m.homeY, m.MaxHP, true, now, now
			}
			continue
		}
		if now.Sub(m.bornAt) < mobRise {
			continue
		}
		// Efeitos das habilidades: queimadura e veneno batem a cada segundo;
		// atordoado (ou congelado) não anda nem ataca; lento anda menos.
		wd.tickDot(m, now)
		if m.Kind == mobDummy {
			continue // boneco de treino: fica parado apanhando
		}
		if !m.Alive || now.Before(m.stunUntil) {
			continue
		}
		if wd.ticks%tickRate == int(m.ID)%tickRate {
			wd.unstick(m) // uma vez por segundo: ninguém fica dentro da parede
		}
		k := mobKinds[m.Kind]
		if now.Before(m.slowUntil) {
			k.Speed *= 0.45
		}
		target, d := wd.nearestFor(m.X, m.Y, now)
		if target == nil || d > mobAggro || dist(m.X, m.Y, m.homeX, m.homeY) > mobLeash {
			// Sem alvo, vaga devagar em volta da cova.
			if now.After(m.wanderAt) {
				m.wanderX, m.wanderY = wd.wanderPoint(m)
				m.wanderAt = now.Add(time.Duration(2000+wd.rng.Intn(3000)) * time.Millisecond)
			}
			if dist(m.X, m.Y, m.homeX, m.homeY) > 24 {
				wd.mobChase(m, m.homeX, m.homeY, k.Speed/2, now) // longe da cova: volta contornando
			} else {
				wd.mobToward(m, m.wanderX, m.wanderY, k.Speed/2)
			}
			continue
		}
		if m.Kind == mobVulkhar && wd.demonStep(m, k, target, d, now) {
			continue
		}
		if k.Boss && wd.bossStomp(m, k, d, now) {
			continue
		}
		// Um servo de necromante mais perto que o jogador vira o alvo.
		if s := wd.minionNear(m.X, m.Y, math.Min(d, 40)); s != nil {
			if dist(m.X, m.Y, s.X, s.Y) > mobReach {
				wd.mobChase(m, s.X, s.Y, k.Speed, now)
			} else if now.Sub(m.atkAt) > mobCooldown {
				m.atkAt = now
				s.HP -= k.Dmg + wd.rng.Intn(k.DmgSp)
				s.hurtAt = now
			}
			continue
		}
		if d > mobReach {
			wd.mobChase(m, float64(target.X), float64(target.Y), k.Speed, now)
		} else if now.Sub(m.atkAt) > mobCooldown {
			m.atkAt = now
			wd.hurt(target, max(1, int(float64(k.Dmg+wd.rng.Intn(k.DmgSp))*lvScale(mobLv(m.Kind), target.Level))), "por "+k.Name, now)
			if s, ok := mobInflict[m.Kind]; ok && wd.rng.Intn(100) < 35 {
				wd.afflict(target, s, 6*time.Second, now)
			}
		}
	}
}

// mobToward anda em direção ao ponto, escorregando pelos obstáculos.
func (wd *World) mobToward(m *Mob, x, y, speed float64) {
	dx, dy := x-m.X, y-m.Y
	d := math.Hypot(dx, dy)
	if d < 1 {
		return
	}
	nx, ny := m.X+dx/d*speed, m.Y+dy/d*speed
	switch {
	case wd.mobFits(m, nx, ny):
		m.X, m.Y = nx, ny
	case wd.mobFits(m, nx, m.Y):
		m.X = nx
	case wd.mobFits(m, m.X, ny):
		m.Y = ny
	}
}

func (wd *World) hurt(p *Player, dmg int, by string, now time.Time) {
	if p.paused {
		return // pausado: nada fere
	}
	if now.Before(p.guardUntil) {
		dmg = dmg * 3 / 10
	}
	dmg = max(1, dmg*(100-p.defPct())/100)
	wd.pop(p, fmt.Sprintf("-%d", dmg), 0xff5050, float64(p.X), float64(p.Y)-34, now)
	wd.hurtRaw(p, dmg, by, now)
}

// hurtRaw tira vida sem passar pela proteção (veneno, frio, queda).
func (wd *World) hurtRaw(p *Player, dmg int, by string, now time.Time) {
	if p.paused {
		return
	}
	p.HP -= dmg
	p.hurtAt = now
	if p.HP > 0 {
		wd.eggCheck(p, now) // Ovo Escarlate: abaixo de 1/4 da vida
		return
	}
	wd.say(ChatLine{Text: p.Name + " foi derrotado " + by + "."})
	// Como nas almas perdidas: o dinheiro fica no chão, só para o dono, por um tempo.
	if p.Gold > 0 {
		wd.dropCoins(p.Gold, float64(p.X), float64(p.Y), p.ID, 10*time.Minute, now)
		p.Gold = 0
	}
	p.X, p.Y = wd.respawnAt(p)
	wd.popKind(p, "Você pereceu", 0xc02020, float64(p.X), float64(p.Y), 1, now)
	p.HP, p.MP, p.ST = p.MaxHP, p.MaxMP, float64(p.MaxST)
	p.boat, p.status = false, [numStatus]time.Time{}
}

func (wd *World) hitBoss(dmg int, now time.Time) {
	b := &wd.boss
	if !b.Alive {
		return
	}
	b.HP -= dmg
	b.hurtAt = now
	if b.HP > 0 {
		return
	}
	b.HP, b.Alive, b.armed, b.respawnAt = 0, false, false, now.Add(15*time.Second)
	wd.say(ChatLine{Text: "Tiamath foi derrotada!"})
	wd.dropCoins(800, b.X, b.Y, 0, lootLife, now)
	for _, p := range wd.near(b.X, b.Y) {
		p.Gold += 500
		p.Kills++
		p.Bosses |= 1 << bossBit[killDragon]
		p.Slain |= slainDragon
		wd.levelUp(p)
		// Além da garra, o dragão larga a pele para quem ainda não a tem.
		for _, it := range []int{itemDragonHide, itemDragonEye} {
			if p.count(it) == 0 {
				wd.drop(it, b.X, b.Y, p.ID, now)
			}
		}
	}
	wd.bossDrop(itemDragonClaw, b.X, b.Y, now)
	wd.credit(killDragon, b.X, b.Y, now)
}

func (wd *World) Join(name string, class int) int {
	wd.mu.Lock()
	defer wd.mu.Unlock()

	name = sanitizeName(name)
	base := name
	for n := 2; wd.nameTaken(name); n++ {
		name = base + strconv.Itoa(n)
	}

	return wd.spawn(name, class).ID
}

// spawn cria o jogador na vila, no nível 1 da classe.
func (wd *World) spawn(name string, class int) *Player {
	wd.nextID++
	x, y := wd.spawnPoint()
	if class < 0 || class >= len(classes) {
		class = 0
	}
	p := &Player{ID: wd.nextID, Name: name, X: x, Y: y, Color: wd.nextID, Class: class, Level: 1, FaceX: 1, Bonfire: -1, Intro: len(prologue)}
	if class == classDepraved {
		p.give(itemStick, 1) // entra na mão sozinho (primeira arma)
	}
	p.recalc()
	p.HP, p.MP, p.ST = p.MaxHP, p.MaxMP, float64(p.MaxST)
	wd.players[p.ID] = p
	wd.say(ChatLine{Text: name + " entrou no mundo."})
	return p
}

func (wd *World) nameTaken(name string) bool {
	for _, p := range wd.players {
		if strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}

func (wd *World) Leave(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok {
		wd.persist(p)
		delete(wd.players, id)
		wd.say(ChatLine{Text: p.Name + " saiu."})
	}
}

// walk anda n células na direção dada, um eixo por vez, para o jogador
// escorregar ao longo das paredes.
func (wd *World) walk(p *Player, dx, dy, n int) {
	p.FaceX, p.FaceY = dx, dy
	now := wd.clock()
	for ; n > 0; n-- {
		if dx != 0 && wd.fits(p.X+dx, p.Y) && wd.canEnter(p, p.X+dx, p.Y, now) {
			p.X += dx
		}
		if dy != 0 && wd.fits(p.X, p.Y+dy) && wd.canEnter(p, p.X, p.Y+dy, now) {
			p.Y += dy
		}
	}
}

// Move é o passo de uma tecla no terminal.
func (wd *World) Move(id, dx, dy int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok {
		wd.walk(p, dx, dy, 2)
		wd.enter(p)
	}
}

// SetInput guarda a direção que o cliente web está segurando; o jogador anda
// a cada tick enquanto ela não for zero.
func (wd *World) SetInput(id, dx, dy int, sprint bool) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok && !p.paused {
		p.moveX, p.moveY, p.sprint = max(-1, min(1, dx)), max(-1, min(1, dy)), sprint
	}
}

func (wd *World) Say(id int, text string) {
	text = sanitizeText(text)
	if text == "" {
		return
	}
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok {
		wd.say(ChatLine{From: p.Name, Color: p.Color, Text: text})
		wd.tryUnseal(p, text, "", wd.clock()) // dizer a palavra diante da porta selada
	}
}

func (wd *World) say(l ChatLine) {
	wd.chat = append(wd.chat, l)
	if len(wd.chat) > maxChatLines {
		wd.chat = wd.chat[len(wd.chat)-maxChatLines:]
	}
}

func (wd *World) Snapshot(id int) Snapshot {
	wd.mu.RLock()
	defer wd.mu.RUnlock()

	now := wd.clock()
	s := Snapshot{
		Players: make([]Player, 0, len(wd.players)),
		Boss:    wd.boss,
		Effects: append([]Effect(nil), wd.effects...),
		Minions: wd.minionsOf(now),
		Projs:   append([]Proj(nil), wd.projs...),
		Chat:    append([]ChatLine(nil), wd.chat...),
		Now:     now,
		T:       now.Sub(wd.start).Seconds(),
		Weather: wd.weather,
		Invaded: wd.inv.active,
		Frozen:  wd.frozen(),
	}
	s.Tod, s.Day = wd.timeOfDay(s.T), wd.dayOf(s.T)
	for _, l := range wd.loot {
		if l.Owner == 0 || l.Owner == id {
			s.Loot = append(s.Loot, l)
		}
	}
	if !s.Boss.Alive {
		s.Boss.RespawnIn = wd.boss.respawnAt.Sub(now)
	}
	for i := range s.Effects {
		e := &s.Effects[i]
		e.P = math.Min(1, math.Max(0, float64(now.Sub(e.Start))/float64(e.Dur)))
	}
	for _, m := range wd.mobs {
		if m.Alive {
			s.Mobs = append(s.Mobs, *m)
		}
	}
	for _, st := range wd.steeds {
		if st.Alive {
			s.Steeds = append(s.Steeds, *st)
		}
	}
	for _, h := range wd.herbs {
		if h.Alive {
			s.Herbs = append(s.Herbs, *h)
		}
	}
	for _, p := range wd.players {
		s.Players = append(s.Players, *p)
		if p.ID == id {
			s.Me = *p
			s.Me.inv = append([]invSlot(nil), p.inv...)
			s.Me.Stash = append([]invSlot(nil), p.Stash...)
			s.Me.chestDay = maps.Clone(p.chestDay)
			s.Me.pops = append([]pop(nil), p.pops...)
		}
	}
	sort.Slice(s.Players, func(i, j int) bool { return s.Players[i].ID < s.Players[j].ID })
	return s
}

// O nome e o chat vão parar na tela dos outros jogadores, então só
// passam caracteres imprimíveis (nada de sequências de escape).
func sanitizeName(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n == maxNameLen {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
			n++
		}
	}
	if n == 0 {
		return "anon"
	}
	return b.String()
}

func sanitizeText(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n == maxChatLen {
			break
		}
		if unicode.IsPrint(r) {
			b.WriteRune(r)
			n++
		}
	}
	return strings.TrimSpace(b.String())
}
