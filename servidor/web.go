package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"asciimmo/proto"

	"github.com/gorilla/websocket"
)

//go:embed web/index.html
var indexHTML []byte

// webChatRows é quantas linhas de chat vão em cada estado.
const webChatRows = 8

// webMobRange é até onde cada cliente recebe os inimigos comuns.
const webMobRange = 280.0

// As mensagens ficam no pacote proto, compartilhado com o cliente nativo.
type (
	wsSkill    = proto.Skill
	wsPortal   = proto.Portal
	wsInit     = proto.Init
	wsZone     = proto.Zone
	wsSteedK   = proto.SteedK
	wsThing    = proto.Thing
	wsAffinity = proto.Affinity
	wsError    = proto.Error
	wsNode     = proto.Node
	wsItem     = proto.Item
	wsRegion   = proto.Region
	wsLoot     = proto.Loot
	wsQuest    = proto.Quest
	wsPlayer   = proto.Player
	wsBoss     = proto.Boss
	wsMob      = proto.Mob
	wsFx       = proto.Fx
	wsProj     = proto.Proj
	wsChat     = proto.Chat
	wsState    = proto.State
	wsIn       = proto.In
)

// Os tamanhos do protocolo têm de bater com os do jogo.
var (
	_ [numAttrs]int  = [proto.NumAttrs]int{}
	_ [numStatus]int = [proto.NumStatus]int{}
)

// since devolve há quantos segundos t aconteceu, com teto para o zero de time.Time.
func since(now, t time.Time) float64 {
	return math.Round(math.Min(9, now.Sub(t).Seconds())*100) / 100
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func stateOf(s Snapshot) wsState {
	st := wsState{
		T: math.Round(s.T*1000) / 1000,
		Boss: wsBoss{X: round1(s.Boss.X), Y: round1(s.Boss.Y), HP: s.Boss.HP, MaxHP: s.Boss.MaxHP, Alive: s.Boss.Alive,
			Respawn: int(s.Boss.RespawnIn.Seconds()) + 1, Hit: since(s.Now, s.Boss.hurtAt)},
		Players: make([]wsPlayer, 0, len(s.Players)),
		Mobs:    make([]wsMob, 0, len(s.Mobs)),
		Fx:      make([]wsFx, 0, len(s.Effects)),
		Projs:   make([]wsProj, 0, len(s.Projs)),
		Unl:     s.Me.Unlocked,
		Tod:     math.Round(s.Tod*1000) / 1000,
		Season:  seasonOf(s.Day),
		Wx:      s.Weather,
		Inv:     make([][2]int, 0, len(s.Me.inv)),
		Loot:    make([]wsLoot, 0, len(s.Loot)),
		XP:      s.Me.XP, XPNeed: xpNeed(s.Me.Level), Points: s.Me.Points, SPts: s.Me.SkillPts,
		Attr: s.Me.Attr, Tree: append([]int(nil), s.Me.Web...), ST: int(s.Me.ST), MaxST: s.Me.MaxST, Bar: s.Me.Bar, BarN: s.Me.barCap(),
		Dmg: [2]int{s.Me.dmgPct(0), s.Me.dmgPct(1)}, Def: s.Me.defPct(), Ach: s.Me.Ach,
		Chests: make([]bool, len(chests)),
		Gear:   [5]int{s.Me.Equip, s.Me.Helm, s.Me.Skin, s.Me.Boots, s.Me.Amulet},
		Owned:  s.Me.Steeds,

		Chapter:  s.Me.chapterOf(),
		Event:    map[bool]int{true: 1}[s.Invaded],
		SideDone: s.Me.SideDone,
		SideProg: s.Me.SideProg[:],
		Bonfire:  s.Me.Bonfire,
	}
	st.BagCap, st.Found, st.Tech, st.Gates = s.Me.bagCap(), s.Me.Found, s.Me.Tech, s.Me.Gates
	st.Paused, st.Frozen, st.Saves = s.Me.paused, s.Frozen, s.Me.saves
	st.LightSlot = s.Me.LightSlot
	st.House, st.StashGold = s.Me.House, s.Me.StashGold
	st.Mode, st.FishKinds, st.FishBodies = s.Me.Mode, s.Me.FishKinds, s.Me.FishBodies
	st.Pure = s.Me.purified()
	st.BestSeen, st.SteedSeen, st.BestKills = s.Me.BestSeen, s.Me.SteedSeen, s.Me.BestKills
	if g := s.Me.goal; g != nil {
		st.Goal = &[2]int{g.x, g.y}
	}
	if f := s.Me.fish; f != nil {
		st.Fish = &proto.FishLine{X: f.x, Y: f.y, Bite: !s.Now.Before(f.biteAt) && !s.Now.After(f.until), Water: f.water}
	}
	for _, v := range s.Me.Stash {
		st.Stash = append(st.Stash, [2]int{v.Item, v.N})
	}
	if t := s.Me.tame; t != nil {
		st.Tame = &proto.Tame{Name: tameName(t), Seq: t.seq, Left: math.Max(0, t.until.Sub(s.Now).Seconds())}
	}
	if s.Me.Moored {
		st.Moor = &[2]int{s.Me.MoorX, s.Me.MoorY}
	}
	for _, p := range s.Me.pops {
		if s.Now.Sub(p.at) < 3*time.Second {
			st.Pops = append(st.Pops, proto.Pop{ID: p.ID, Text: p.Text, Color: p.Color, X: p.X, Y: p.Y, Kind: p.Kind})
		}
	}
	for i := range st.Sts {
		st.Sts[i] = math.Max(0, math.Round(s.Me.status[i].Sub(s.Now).Seconds()*10)/10)
		st.Imm[i] = math.Max(0, math.Round(s.Me.immune[i].Sub(s.Now).Seconds()))
	}
	mx, my := float64(s.Me.X), float64(s.Me.Y)
	for _, sd := range s.Steeds {
		if dist(sd.X, sd.Y, mx, my) <= webMobRange {
			st.Steeds = append(st.Steeds, wsThing{ID: sd.ID, Kind: sd.Kind, X: round1(sd.X), Y: round1(sd.Y)})
		}
	}
	for _, h := range s.Herbs {
		if dist(float64(h.X), float64(h.Y), mx, my) <= webMobRange {
			st.Herbs = append(st.Herbs, wsThing{ID: h.ID, Kind: h.Kind, X: float64(h.X), Y: float64(h.Y)})
		}
	}
	for i := range chests {
		st.Chests[i] = s.Me.chestDay[i] == s.Day
	}
	for _, sl := range s.Me.inv {
		st.Inv = append(st.Inv, [2]int{sl.Item, sl.N})
	}
	for _, l := range s.Loot {
		st.Loot = append(st.Loot, wsLoot{ID: l.ID, Item: l.Item, X: l.X, Y: l.Y, N: l.N})
	}
	if s.Me.inPrologue() {
		st.Quest = prologueQuest(&s.Me)
	} else if s.Me.Quest < len(quests) {
		q := quests[s.Me.Quest]
		steps := stepsOf(q)
		cur := min(s.Me.QStep, len(steps)-1)
		st.Quest = &wsQuest{N: s.Me.Quest + 1, Of: len(quests), Title: q.Title, Text: q.Text, Prog: s.Me.QProg, Need: steps[cur].Need, Chapter: q.Chapter, Cur: cur}
		for i, sp := range steps {
			qs := proto.QStep{Text: sp.Text, Need: sp.Need, Done: i < cur}
			if i == cur {
				qs.Prog = s.Me.QProg
			}
			st.Quest.Steps = append(st.Quest.Steps, qs)
		}
	}
	book := classes[s.Me.Class].Skills
	st.CD2 = make([]float64, len(book))
	for i := range book {
		st.CD2[i] = math.Max(0, math.Round(s.Me.ready[i].Sub(s.Now).Seconds()*10)/10)
		if i < len(st.CDs) {
			st.CDs[i] = st.CD2[i]
		}
	}
	for i, t := range []time.Time{s.Me.dmgUntil, s.Me.critUntil, s.Me.venomUntil} {
		st.Buf[i] = math.Max(0, math.Round(t.Sub(s.Now).Seconds()*10)/10)
	}
	for _, pr := range s.Projs {
		st.Projs = append(st.Projs, wsProj{ID: pr.ID, Kind: pr.Kind, X: round1(pr.X), Y: round1(pr.Y)})
	}
	for _, p := range s.Players {
		st.Players = append(st.Players, wsPlayer{ID: p.ID, Name: p.Name, X: p.X, Y: p.Y, Color: p.Color, HP: p.HP, MaxHP: p.MaxHP, MP: p.MP, MaxMP: p.MaxMP,
			Level: p.Level, Gold: p.Gold, Hit: since(s.Now, p.hurtAt), Class: p.Class, Act: classes[p.Class].Skills[clampInt(p.act, 0, len(classes[p.Class].Skills)-1)].Anim, ActAt: since(s.Now, p.actAt),
			Dir: math.Round(p.actDir*100) / 100, Guard: s.Now.Before(p.guardUntil), Equip: p.Equip, Skin: p.Skin, Ride: p.Riding, Boat: p.boat, Light: p.Light, Pz: p.paused})
	}
	// Só vão os inimigos por perto; os chefes vão sempre, para o minimapa.
	for _, m := range s.Mobs {
		k := mobKinds[m.Kind]
		if !k.Boss && dist(m.X, m.Y, float64(s.Me.X), float64(s.Me.Y)) > webMobRange {
			continue
		}
		st.Mobs = append(st.Mobs, wsMob{ID: m.ID, Kind: m.Kind, Name: k.Name, X: round1(m.X), Y: round1(m.Y), HP: m.HP, MaxHP: m.MaxHP,
			Age: since(s.Now, m.bornAt), Hit: since(s.Now, m.hurtAt), Atk: since(s.Now, m.atkAt), Boss: k.Boss, St: mobStatus(&m, s.Now), Lv: mobLv(m.Kind)})
	}
	st.Mobs = append(st.Mobs, s.Minions...)
	for _, e := range s.Effects {
		st.Fx = append(st.Fx, wsFx{Kind: e.Kind, X: e.X, Y: e.Y, R: e.R, Dir: e.Dir, P: math.Round(e.P*100) / 100})
	}
	chat := s.Chat
	if len(chat) > webChatRows {
		chat = chat[len(chat)-webChatRows:]
	}
	st.Chat = make([]wsChat, 0, len(chat))
	for _, l := range chat {
		st.Chat = append(st.Chat, wsChat{From: l.From, Color: l.Color, Text: l.Text})
	}
	return st
}

func initOf(wd *World, id int) wsInit {
	class := wd.Snapshot(id).Me.Class
	c := classes[class]
	in := wsInit{Init: true, ID: id, W: wd.w, H: wd.h, Class: c.Name, OverH: overH, Fog: wd.FogOf(id),
		Affinity: wsAffinity{WTypes: weaponNames, Aff: affinity[class], AffPct: [2]int{affinePct, clumsyPct}}}
	for i, it := range items {
		wt := -1
		if it.Kind == kindWeapon {
			wt = weaponType[i]
		}
		in.Items = append(in.Items, wsItem{Name: it.Name, Kind: it.Kind, Rarity: it.Rarity, Bonus: it.Bonus, Def: it.Def, Magic: it.Magic, WType: wt, Color: int(it.Color), Desc: it.Desc, Price: fishPrice(i)})
	}
	for i := range attrNames {
		in.Attrs = append(in.Attrs, [2]string{attrNames[i], attrDescs[i]})
	}
	// A teia vai inteira para o cliente nativo; o navegador lê os mesmos nós
	// como lista (nome, descrição, grau máximo).
	for _, n := range c.Web {
		in.Tree = append(in.Tree, wsNode{Name: n.Name, Desc: n.Desc, Max: n.Max})
		in.Web = append(in.Web, proto.WebNode{Name: n.Name, Desc: n.Desc, Icon: n.Icon, Elem: n.Elem, Ring: n.Ring, Ang: n.Ang,
			Links: n.Links, Max: n.Max, Lv: n.Lv, Attr: n.Attr, Skill: n.Skill, Crypt: n.Crypt})
	}
	for _, b := range c.Branches {
		in.Branches = append(in.Branches, proto.Branch{Name: b.Name, Ang: b.Ang, Elem: b.Elem})
	}
	in.MainAttr, in.Elems = c.Attr, elemNames[:]
	if c.anyAttr {
		in.MainAttr = -1 // qualquer atributo: vale o mais alto
	}
	for _, a := range achievements {
		in.Achs = append(in.Achs, [2]string{a.Name, a.Desc})
	}
	for _, c := range chests {
		in.Chests = append(in.Chests, [2]int{c.X, c.Y})
	}
	for _, r := range regions {
		in.Regions = append(in.Regions, wsRegion{Name: r.Name, X: r.X, Y: r.Y, R: r.R})
	}
	for i, sk := range c.Skills {
		s := wsSkill{Name: sk.Name, Cost: sk.Cost, CD: sk.CD.Seconds(), Icon: sk.Icon, Desc: sk.Desc, Elem: sk.Elem}
		if i < 3 {
			in.Skills = append(in.Skills, s) // os três de sempre, para o navegador
		}
		in.Book = append(in.Book, s)
	}
	for _, l := range wd.lights {
		in.Lights = append(in.Lights, [4]int{l.X, l.Y, l.R, l.Kind})
	}
	for _, p := range wd.portals {
		if p.Label != "" { // o buraco não aparece como portal
			up := p.Y >= overH && p.TY < p.Y && (p.TY < overH || inDeep(p.Y) && !inDeep(p.TY))
			in.Portals = append(in.Portals, wsPortal{X: p.X, Y: p.Y, Label: p.Label, Up: up})
		}
	}
	in.Docks = docks
	in.Travel = wd.travelInfo()
	in.Seals = wd.sealInfo()
	in.NPCs, in.Techs, in.Gates, in.Reads = wd.npcInfo(), techInfo(&c), wd.gateInfo(), wd.reads
	for _, v := range vaults {
		in.Doors = append(in.Doors, [2]int{v[0], v[1]})
	}
	for _, z := range zones {
		in.Zones = append(in.Zones, wsZone{Name: z.Name, X: z.X, Y: z.Y, W: z.W, H: z.H})
	}
	for _, k := range steedKinds {
		in.SteedK = append(in.SteedK, wsSteedK{Name: k.Name, Level: k.Level, Speed: k.Speed, Def: k.Def})
	}
	in.HerbK = herbItem
	in.Status = statusNames[:]
	in.Raft = raftWood
	in.Currency = currency
	for _, m := range wd.shops {
		in.Shops = append(in.Shops, proto.Shop{Name: m.Name, X: m.X, Y: m.Y, Kind: m.Kind, Stock: m.Stock})
	}
	for _, b := range wd.bonfires {
		in.Bonfires = append(in.Bonfires, proto.Bonfire{Name: b.Name, X: b.X, Y: b.Y})
	}
	for _, c := range chapters {
		in.Chapters = append(in.Chapters, [2]string{c.Title, c.Intro})
	}
	for k := range mobKinds {
		b := bestiary[k]
		in.Beasts = append(in.Beasts, proto.Beast{Name: mobKinds[k].Name, Lv: mobLv(k), HP: mobKinds[k].HP, Dmg: mobKinds[k].Dmg,
			DmgSp: mobKinds[k].DmgSp, Boss: mobKinds[k].Boss, Lore: b.Lore, Weak: elemList(b.Weak), Resist: elemList(b.Resist)})
	}
	for k, sk := range steedKinds {
		in.SteedLore = append(in.SteedLore, steedLore[k%len(steedLore)])
		_ = sk
	}
	in.Waters = waterNames[:]
	for _, f := range fishes {
		in.Fishes = append(in.Fishes, fishDesc(f))
	}
	in.Fish0, in.Modes = itemFish0, modeNames[:]
	for i, l := range lots {
		x, y := lotDoor(i)
		in.Houses = append(in.Houses, proto.HouseLot{Name: l.Name, X: x, Y: y, Price: l.Price})
	}
	for _, q := range quests { // missões principais, para o diário
		in.Quests = append(in.Quests, proto.SideDef{Title: q.Title, Text: q.Text, Chapter: q.Chapter, Need: q.Need})
	}
	for _, q := range sideQuests {
		in.Sides = append(in.Sides, proto.SideDef{Title: q.Title, Text: q.Text, Chapter: q.Chapter, Need: q.Need, Unlock: q.Unlock})
	}
	return in
}

// mapBlob empacota o terreno para o navegador: "AMAP", largura e altura
// (uint16), depois um plano de caracteres, um de cores RGB e um de flags
// (1 = sólido, 2 = por cima dos personagens, 4 = água, 8 = lama, 16 = veneno), tudo comprimido com gzip.
//
// O mapa vem em duas partes: a 0 (padrão) é a superfície com a cripta e as
// cavernas, linhas [0, deepY0); a 1 (/map?parte=fundo) são as Profundezas,
// linhas [deepY0, h). Cada parte diz a própria altura no cabeçalho, então o
// navegador, que só pede a 0, continua igual; o cliente nativo baixa a 1
// quando alguém desce.
func mapBlob(wd *World, part int) []byte {
	y0, y1 := 0, deepY0
	if part == 1 {
		y0, y1 = deepY0, wd.h
	}
	a, b := y0*wd.w, y1*wd.w
	var raw bytes.Buffer
	raw.WriteString("AMAP")
	binary.Write(&raw, binary.LittleEndian, [2]uint16{uint16(wd.w), uint16(y1 - y0)})
	raw.Write(wd.glyph[a:b])
	for _, c := range wd.color[a:b] {
		raw.Write([]byte{byte(c >> 16), byte(c >> 8), byte(c)})
	}
	for i := a; i < b; i++ {
		raw.WriteByte(wd.flagsAt(i))
	}
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	zw.Write(raw.Bytes())
	zw.Close()
	return out.Bytes()
}

// MapBlob devolve a parte do mapa empacotada na versão atual, refeita só
// quando o terreno mudou desde a última vez.
func (wd *World) MapBlob(part int) ([]byte, int) {
	wd.blobMu.Lock()
	defer wd.blobMu.Unlock()
	wd.mu.RLock()
	defer wd.mu.RUnlock()
	if wd.blob[part] == nil || wd.blobVer[part] != wd.mapVer {
		wd.blob[part], wd.blobVer[part] = mapBlob(wd, part), wd.mapVer
	}
	return wd.blob[part], wd.blobVer[part]
}

// EditsSince é editsSince com o lock.
func (wd *World) EditsSince(v int) ([]cellEdit, int, bool) {
	wd.mu.RLock()
	defer wd.mu.RUnlock()
	return wd.editsSince(v)
}

func newWebServer(addr string, world *World) *http.Server {
	// Origem não é conferida: o jogo não tem login nem cookie para proteger.
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/map", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Cache-Control", "no-cache")
		part := 0
		if r.URL.Query().Get("parte") == "fundo" {
			part = 1
		}
		blob, ver := world.MapBlob(part)
		w.Header().Set("X-Map-Full-H", strconv.Itoa(world.h)) // altura do mundo todo (o cliente reserva)
		w.Header().Set("X-Map-Ver", strconv.Itoa(ver))
		w.Write(blob)
	})
	// Personagens deste aparelho: listar e excluir (a tela de entrada usa).
	mux.HandleFunc("/chars", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		list := world.Chars(r.URL.Query().Get("token"))
		if list == nil {
			list = []CharInfo{}
		}
		json.NewEncoder(w).Encode(list)
	})
	mux.HandleFunc("/chars/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		res := map[string]string{}
		if err := world.DeleteChar(r.FormValue("token"), r.FormValue("name")); err != nil {
			res["error"] = err.Error()
		}
		json.NewEncoder(w).Encode(res)
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(1024)

		q := r.URL.Query()
		class, _ := strconv.Atoi(q.Get("cls"))
		id, err := world.JoinSaved(q.Get("name"), class, q.Get("token"), q.Get("pw"))
		if err != nil {
			conn.WriteJSON(wsError{Error: err.Error()})
			return
		}
		defer world.Leave(id)
		mapVer, _ := strconv.Atoi(q.Get("mv")) // versão do mapa que o cliente baixou

		// Só esta goroutine escreve na conexão; a de leitura avisa quando cai.
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				var in wsIn
				if err := conn.ReadJSON(&in); err != nil {
					// JSON ruim não derruba a sessão; conexão fechada, sim.
					if _, bad := err.(*json.SyntaxError); bad {
						continue
					}
					return
				}
				switch in.T {
				case "in":
					world.SetInput(id, in.DX, in.DY, in.Run)
				case "attr":
					world.SpendAttr(id, in.I)
				case "tree":
					world.SpendTree(id, in.I)
				case "bar":
					world.SetBar(id, in.I, in.Slot)
				case "use":
					world.Use(id, in.Slot, in.Dir)
				case "item":
					world.UseItem(id, in.Item)
				case "say":
					world.Say(id, in.Text)
				case "act":
					world.Interact(id)
				case "ride":
					world.Ride(id)
				case "steed":
					world.PickSteed(id, in.I)
				case "release":
					world.ReleaseSteed(id, in.I)
				case "talk":
					world.Talk(id, in.I)
				case "learn":
					world.Learn(id, in.I, in.Text)
				case "gaterune":
					world.GateRune(id, in.I, in.Text)
				case "gate":
					world.GateGo(id, in.I)
				case "seal":
					world.Unseal(id, in.Text)
				case "travel":
					world.Travel(id, in.I)
				case "drop":
					world.Drop(id, in.Item)
				case "sell":
					world.Sell(id, in.I, in.Item)
				case "buy":
					world.Buy(id, in.I, in.Item)
				case "pause":
					world.SetPaused(id, in.I == 1)
				case "save":
					world.SaveNow(id)
				case "light":
					world.ToggleLight(id)
				case "tame":
					world.Tame(id, in.Text)
				case "mode":
					world.SetMode(id, in.I)
				case "goto":
					world.SetGoal(id, in.I, in.Slot)
				case "nogoto":
					world.CancelGoal(id)
				case "fish":
					world.FishCast(id)
				case "house":
					world.BuyHouse(id, in.I)
				case "stash":
					world.Stash(id, in.I, in.Item, in.Slot)
				case "wait":
					world.Wait(id, in.I)
				case "waitwx":
					world.WaitWeather(id)
				}
			}
		}()

		if conn.WriteJSON(initOf(world, id)) != nil {
			return
		}
		t := time.NewTicker(time.Second / tickRate)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-r.Context().Done():
				return
			case <-t.C:
				st := stateOf(world.Snapshot(id))
				cells, ver, reload := world.EditsSince(mapVer)
				for _, c := range cells {
					st.Ed = append(st.Ed, [4]int{c.I, int(c.Ch), int(c.C), int(c.F)})
				}
				st.MV, st.Reload, mapVer = ver, reload, ver
				st.Purged = world.PurgedFires()
				st.Marks = world.MarksOf(id)
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if conn.WriteJSON(st) != nil {
					return
				}
			}
		}
	})
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
}
