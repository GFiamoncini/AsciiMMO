package main

import (
	"math"
	"strings"

	"asciimmo/proto"
)

// O que faz barulho: tudo sai da diferença entre um estado e o anterior
// (efeitos e projéteis novos, golpes levados e dados, mortes, dinheiro,
// nível, células do mapa quebradas) e do lugar onde o jogador está (ambiente
// e música). O servidor não manda nada a mais para o som.

// Voz de cada tipo de inimigo, na ordem dos tipos do servidor (rpg.go).
var mobVoice = []string{
	"skeleton", "skeleton", "skeleton", "asura", "triton", "golem", "asura", "triton", "archon", // esqueleto, guerreiro, lorde, asura, tritão, sentinela, rainha asura, tirano das marés, arconte
	"rat", "wolf", "orc", "orc", "orc", "scarab", "mummy", "mummy", "wolf", "yeti", "giant", // rato, lobo, orcs, escaravelho, múmia, faraó, lobo do gelo, yeti, gigante
	"spider", "troll", "bat", "", "crawler", "golem", "fungus", "demon", "apostle", "apostle", // aranha, troll, morcego, boneco, rastejante, golem de magma, fungo, Ignar, apóstolo, Grael
	"scorpion", "wraith", "roots", "dwarf", "demon", "golem", // escorpião, espectro, raízes, anão, Vulkhar, golem de cristal
}

// Material de quem apanha: 1 osso, 2 pedra (o resto é carne).
func mobMat(kind int) int {
	switch kind {
	case 0, 1, 2:
		return 1
	case 5, 25, 33, 35:
		return 2
	}
	return 0
}

var steedVoice = []string{"horse", "grunt", "grunt", "growl"}

// Som de cada efeito (proto.Fx*).
var fxSound = map[int]string{
	proto.FxSlash: "swing", proto.FxNova: "missile", proto.FxWarn: "warn", proto.FxBlast: "boom",
	proto.FxPuff: "bones", proto.FxThrust: "swing_heavy", proto.FxSpin: "swing_heavy", proto.FxBoom: "boom",
	proto.FxHeal: "heal", proto.FxHoly: "holy", proto.FxMeteor: "fire", proto.FxRain: "arrow",
	proto.FxDark: "dark", proto.FxBolt: "thunder", proto.FxChain: "zap", proto.FxFrost: "ice",
	proto.FxFlame: "fire", proto.FxZap: "zap", proto.FxBlink: "blink", proto.FxBuff: "buff",
	proto.FxSmoke: "puff", proto.FxBone: "bones", proto.FxPlague: "dark", proto.FxBlizzard: "ice",
	proto.FxCrack: "stone_break", proto.FxConsecrate: "holy", proto.FxHammer: "boom", proto.FxGrave: "stone_grind",
	proto.FxStorm: "thunder", proto.FxSeal: "holy", proto.FxOblit: "boom", proto.FxGrass: "grass", proto.FxWhip: "fire",
}

var projSound = map[int]string{
	proto.ProjMissile: "missile", proto.ProjFireball: "fire", proto.ProjArrow: "arrow",
	proto.ProjShadow: "dark", proto.ProjIce: "ice", proto.ProjHoly: "holy",
}

// sfxAt toca um som que acontece em (x, y) do mundo: some com a distância,
// vai para o lado de onde vem e não atravessa andares.
func (g *Game) sfxAt(name string, x, y, vol float64) {
	if g.snd == nil || g.cur == nil || g.info == nil {
		return
	}
	me := g.cur.pmap[g.info.ID]
	if me == nil {
		return
	}
	if (y >= float64(g.info.OverH)) != (me.Y >= g.info.OverH) {
		return
	}
	dx, dy := x-float64(me.X), y-float64(me.Y)
	d := math.Hypot(dx, dy)
	if d > 90 {
		return
	}
	g.snd.play(name, vol/(1+math.Pow(d/22, 2)), dx/45, g.now())
}

// sfx toca um som sem lugar (interface, o próprio jogador).
func (g *Game) sfx(name string, vol float64) {
	if g.snd != nil {
		g.snd.play(name, vol, 0, g.now())
	}
}

// soundEvents compara o estado novo com o atual e toca o que mudou.
func (g *Game) soundEvents(s *proto.State, w *state) {
	if g.snd == nil || g.cur == nil || g.info == nil {
		return
	}
	old := g.cur
	for _, e := range s.Fx {
		key := fxKey(e)
		if g.fxSeen[key] {
			continue
		}
		name, vol := fxSound[e.Kind], 0.8
		switch e.Kind {
		case proto.FxBolt, proto.FxStorm:
			vol = 3 // trovão se ouve de longe
		case proto.FxOblit, proto.FxBlast:
			vol = 1.2
		case proto.FxGrass:
			vol = 0.35
		}
		if name != "" {
			g.sfxAt(name, e.X, e.Y, vol)
		}
	}
	for _, p := range s.Projs {
		if old.jmap[p.ID] == nil {
			if n := projSound[p.Kind]; n != "" {
				g.sfxAt(n, p.X, p.Y, 0.55)
			}
		}
	}
	// Inimigos: apanhou (Hit zerou), bateu (Atk zerou), morreu (sumiu ferido por perto).
	for i := range s.Mobs {
		m := &s.Mobs[i]
		o := old.mmap[m.ID]
		if o == nil {
			continue
		}
		voice := ""
		if m.Kind >= 0 && m.Kind < len(mobVoice) {
			voice = mobVoice[m.Kind]
		}
		if m.Hit < o.Hit && m.Hit < 0.3 {
			name := []string{"hit", "hit_bone", "hit_stone"}[mobMat(m.Kind)]
			g.sfxAt(name, m.X, m.Y, 0.8)
		}
		if m.Atk < o.Atk && m.Atk < 0.3 && !m.Ally {
			g.sfxAt("swing", m.X, m.Y, 0.45)
			if voice != "" && g.snd.r.f() < 0.35 {
				v := voice
				if v == "wolf" {
					v = "growl" // lobo atacando rosna; o uivo é de longe
				}
				g.sfxAt(v, m.X, m.Y, 0.6)
			}
		}
	}
	for id, o := range old.mmap {
		if w.mmap[id] != nil || o.HP > max(1, o.MaxHP/3) || o.Ally {
			continue
		}
		if o.Kind >= 0 && o.Kind < len(mobVoice) && mobVoice[o.Kind] != "" {
			g.sfxAt(mobVoice[o.Kind], o.X, o.Y, 0.8)
		}
		if mobMat(o.Kind) == 1 {
			g.sfxAt("bones", o.X, o.Y, 0.9)
		}
	}
	// Tiamath: apanhou.
	if s.Boss.Alive && s.Boss.Hit < old.Boss.Hit && s.Boss.Hit < 0.3 {
		g.sfxAt("hit_stone", s.Boss.X, s.Boss.Y, 1)
	}
	// O próprio jogador.
	me, om := w.pmap[g.info.ID], old.pmap[g.info.ID]
	if me != nil && om != nil {
		if me.Hit < om.Hit && me.Hit < 0.3 {
			g.sfx("hurt", 0.7)
		}
		if me.HP <= 0 && om.HP > 0 {
			g.sfx("death", 0.9)
		}
		if me.Level > om.Level {
			g.sfx("levelup", 0.8)
		}
		if me.Gold > om.Gold {
			g.sfx("coin", 0.6)
		}
		if me.Ride != om.Ride && me.Ride > 0 && me.Ride-1 < len(steedVoice) {
			g.sfx(steedVoice[me.Ride-1], 0.5)
		}
		g.footsteps(me, om)
	}
	if invCount(s.Inv) > invCount(old.Inv) {
		g.sfx("pickup", 0.5)
	}
	if s.Bonfire != old.Bonfire && s.Bonfire >= 0 {
		g.sfx("bonfire", 0.7)
	}
	for _, c := range s.Chat {
		if c.From != "" && me != nil && c.From != me.Name && !chatHas(old.Chat, c) {
			g.sfx("msg", 0.35)
			break
		}
	}
}

func invCount(inv [][2]int) int {
	n := 0
	for _, it := range inv {
		n += it[1]
	}
	return n
}

func chatHas(list []proto.Chat, c proto.Chat) bool {
	for _, l := range list {
		if l.From == c.From && l.Text == c.Text {
			return true
		}
	}
	return false
}

// footsteps: um passo a cada 3 células andadas, com o som do chão.
func (g *Game) footsteps(me, om *proto.Player) {
	d := math.Hypot(float64(me.X-om.X), float64(me.Y-om.Y))
	if d == 0 || d > 8 {
		return
	}
	g.stepAcc += d
	stride := 3.0
	if me.Ride > 0 {
		stride = 4
	}
	if g.stepAcc < stride {
		return
	}
	g.stepAcc = 0
	m := g.m
	if m == nil || me.X < 0 || me.Y < 0 || me.X >= m.w || me.Y >= m.h {
		return
	}
	i := me.Y*m.w + me.X
	f := m.flags[i]
	r, gg, b := m.color[i*3], m.color[i*3+1], m.color[i*3+2]
	name := "step_stone"
	switch {
	case me.Boat:
		if g.snd.r.f() < 0.4 {
			g.sfx("splash", 0.25)
		}
		return
	case f&4 != 0:
		name = "splash"
	case f&8 != 0:
		name = "step_mud"
	case r > 180 && gg > 180 && b > 180:
		name = "step_snow"
	case gg > r && gg > b:
		name = "step_grass"
	}
	vol := 0.22
	if me.Ride > 0 {
		vol = 0.32
	}
	g.sfx(name, vol)
}

// editSounds toca a quebra das células que vão mudar (antes de aplicar):
// árvore cai, rocha esfarela, cristal se parte.
func (g *Game) editSounds(ed [][4]int) {
	if g.snd == nil || g.m == nil {
		return
	}
	m := g.m
	var wood, stone, crystal int
	var sx, sy float64
	for _, e := range ed {
		i := e[0]
		if i < 0 || i >= m.w*m.h {
			continue
		}
		of, nf := m.flags[i], byte(e[3])
		if of&3 == 0 || nf&3 != 0 { // só o que era alto ou sólido e deixou de ser
			continue
		}
		r, gg, b := int(m.color[i*3]), int(m.color[i*3+1]), int(m.color[i*3+2])
		switch {
		case of&64 != 0:
			crystal++
		case abs(r-gg) < 28 && abs(gg-b) < 28:
			stone++
		default:
			wood++
		}
		sx += float64(i % m.w)
		sy += float64(i / m.w)
	}
	n := wood + stone + crystal
	if n == 0 {
		return
	}
	x, y := sx/float64(n), sy/float64(n)
	switch {
	case crystal >= stone && crystal >= wood:
		g.sfxAt("crystal_break", x, y, 1)
	case stone >= wood:
		g.sfxAt("stone_break", x, y, 1)
	case wood > 40:
		g.sfxAt("tree_fall", x, y, 1.1)
	default:
		g.sfxAt("wood_break", x, y, 1)
	}
}

// updateSound decide ambiente e música pelo lugar, solta os sons avulsos
// (pássaros, coruja, sapos, pingos, chamados dos bichos) e avança os fades.
func (g *Game) updateSound(dt float64) {
	s := g.snd
	if s == nil {
		return
	}
	s.vMusic, s.vSfx, s.vAmb = float64(g.opts.VolMusic)/100, float64(g.opts.VolSfx)/100, float64(g.opts.VolAmb)/100
	now := g.now()
	s.update(dt, now)
	if g.cur == nil || g.info == nil || g.m == nil {
		s.setMusic("", now)
		for _, n := range []string{"wind", "crickets", "rain", "sea", "cave", "lava", "fire"} {
			s.setAmb(n, 0)
		}
		return
	}
	me := g.cur.pmap[g.info.ID]
	if me == nil {
		return
	}
	cur := g.cur
	if g.introPage >= 0 { // abertura: música da noite e o ambiente de cada cena
		amb := [][7]float64{{0.3, 0, 0, 0, 0, 0, 0.7}, {0.7, 0, 0.25, 0, 0, 0, 0}, {0.2, 0.55, 0, 0, 0, 0, 0.45}}[clampi(g.introPage, 0, 2)]
		for i, n := range []string{"wind", "crickets", "rain", "sea", "cave", "lava", "fire"} {
			s.setAmb(n, amb[i])
		}
		s.setMusic("noite", now-3000)
		return
	}
	under := me.Y >= g.info.OverH
	deep := me.Y >= g.info.OverH+240 // Profundezas: abaixo da cripta (depths.go, cryptH)
	sun := 0.0
	if !under {
		sun = daylight(cur.Tod)
	}
	region := g.regionName(me.X, me.Y)

	// O que há em volta (amostra esparsa num raio de 24).
	if now-g.sndScanAt > 250 {
		g.sndScanAt = now
		var water, lava, tall, marsh, n int
		m := g.m
		for dy := -24; dy <= 24; dy += 3 {
			for dx := -24; dx <= 24; dx += 3 {
				x, y := me.X+dx, me.Y+dy
				if x < 0 || y < 0 || x >= m.w || y >= m.h || dx*dx+dy*dy > 576 {
					continue
				}
				f := m.flags[y*m.w+x]
				n++
				switch {
				case f&128 != 0:
					lava++
				case f&4 != 0:
					water++
				case f&8 != 0:
					marsh++
				case f&2 != 0:
					tall++
				}
			}
		}
		k := 1 / float64(max(1, n))
		g.sndAround = [4]float64{float64(water) * k, float64(lava) * k, float64(tall) * k, float64(marsh) * k}
	}
	water, lava, tall, marsh := g.sndAround[0], g.sndAround[1], g.sndAround[2], g.sndAround[3]
	snowy := cur.Season == 3 || strings.Contains(region, "Gelo") || strings.Contains(region, "Pico")
	dry := strings.Contains(region, "Deserto")

	wind, crick, rain, cave, fire := 0.0, 0.0, 0.0, 0.0, 0.0
	if !under {
		wind = 0.3
		if snowy {
			wind = 0.55
		}
		if cur.Wx == 2 {
			wind += 0.35
		}
		if sun < 0.3 && !snowy && cur.Wx == 0 {
			crick = 0.5 * (1 - sun/0.3)
		}
		if cur.Wx >= 1 && !snowy && !dry {
			rain = []float64{0, 0.55, 0.9, 0}[clampi(cur.Wx, 0, 3)]
		}
	} else {
		cave = 0.55
		if deep {
			cave = 0.7
		}
	}
	// Fogueira por perto: estala.
	for _, l := range g.info.Lights {
		if l[3] >= 2 {
			continue
		}
		if d := math.Hypot(float64(l[0]-me.X), float64(l[1]-me.Y)); d < 20 {
			fire = math.Max(fire, 0.7*(1-d/20))
		}
	}
	s.setAmb("wind", wind)
	s.setAmb("crickets", crick)
	s.setAmb("rain", rain)
	s.setAmb("sea", math.Min(0.7, water*1.4))
	s.setAmb("lava", math.Min(0.8, lava*2))
	s.setAmb("cave", cave)
	s.setAmb("fire", fire)

	// Sons avulsos do ambiente, cada um no seu relógio.
	rr := &s.r
	shot := func(key string, every [2]float64, name string, vol float64) {
		if now < g.sndNext[key] {
			return
		}
		if g.sndNext[key] > 0 {
			s.playAmb(name, vol, rr.n()*0.8, now)
		}
		g.sndNext[key] = now + rr.rng(every[0], every[1])*1000
	}
	if !under && sun > 0.35 && cur.Wx == 0 && tall > 0.08 {
		shot("bird", [2]float64{2.5, 7}, "bird", 0.35)
	}
	if !under && sun < 0.2 && tall > 0.05 {
		shot("owl", [2]float64{18, 40}, "owl", 0.3)
	}
	if marsh > 0.05 {
		shot("frog", [2]float64{3, 9}, "frog", 0.35)
	}
	if under {
		shot("drip", [2]float64{1.2, 4.5}, "drip", 0.4)
	}
	if !under && sun < 0.25 && strings.Contains(region, "Picos") {
		shot("howl", [2]float64{20, 45}, "wolf", 0.25)
	}
	// Chamados dos bichos por perto, de vez em quando (o mais perto de cada vez).
	if now > g.sndNext["mob"] {
		g.sndNext["mob"] = now + rr.rng(4, 9)*1000
		best, bd := -1, 40.0
		for i, m := range cur.Mobs {
			if m.Ally || m.Kind < 0 || m.Kind >= len(mobVoice) || mobVoice[m.Kind] == "" {
				continue
			}
			if d := math.Hypot(m.X-float64(me.X), m.Y-float64(me.Y)); d < bd && rr.f() < 0.7 {
				best, bd = i, d
			}
		}
		if best >= 0 {
			m := cur.Mobs[best]
			g.sfxAt(mobVoice[m.Kind], m.X, m.Y, 0.5)
		}
		for _, t := range cur.Steeds {
			if t.Kind >= 0 && t.Kind < len(steedVoice) && rr.f() < 0.25 {
				g.sfxAt(steedVoice[t.Kind], t.X, t.Y, 0.4)
				break
			}
		}
	}
	if cur.Boss.Alive && math.Hypot(cur.Boss.X-float64(me.X), cur.Boss.Y-float64(me.Y)) < 120 {
		if now > g.sndNext["roar"] {
			if g.sndNext["roar"] > 0 {
				g.sfxAt("roar", cur.Boss.X, cur.Boss.Y, 2)
			}
			g.sndNext["roar"] = now + rr.rng(10, 22)*1000
		}
	}

	// Música pelo lugar e pela hora.
	track := "dia"
	bossNear := cur.Boss.Alive && math.Hypot(cur.Boss.X-float64(me.X), cur.Boss.Y-float64(me.Y)) < 110
	for _, m := range cur.Mobs {
		if m.Boss && !m.Ally && math.Hypot(m.X-float64(me.X), m.Y-float64(me.Y)) < 50 {
			bossNear = true
		}
	}
	switch {
	case bossNear || cur.Event == 1:
		track = "chefe"
	case deep:
		track = "fundo"
	case under:
		track = "cripta"
	case region == "Vila" || region == "Vale do Despertar" || region == "Terreiro das Sequências":
		track = "vila"
	case sun < 0.25:
		track = "noite"
	}
	s.setMusic(track, now)
}
