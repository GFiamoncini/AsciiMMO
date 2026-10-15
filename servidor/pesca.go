package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Pesca. Com a Vara de Pesca na mochila, E na beira d'água (ou no barco)
// lança a linha: a boia cai na água mais perto e, depois de uns segundos,
// o peixe morde ("!"). Puxar (E ou Espaço) logo na mordida pega o peixe;
// cedo demais espanta, tarde demais ele escapa. O peixe depende da água
// (mar, Lago Espelhado, rio, cada lagoa), da estação e da hora. Peixe se
// vende em qualquer mercador; o registro de pesca (bits FishKinds e
// FishBodies, no save) dá duas conquistas: pescar em todas as águas e pegar
// todos os peixes.

// Águas onde se pesca, na ordem dos bits de FishBodies.
const (
	waterSea = iota
	waterLake
	waterRiver
	waterPondVila
	waterPondReeds
	waterPondIce
	waterPondOasis
	numWaters
)

var waterNames = [numWaters]string{"Mar do Sul", "Lago Espelhado", "Rio do Lago", "Lagoa da Vila", "Lagoa dos Juncos", "Lagoa Gelada", "Lagoa do Oásis"}

// Lagoas novas: centro pedido (procura-se um lugar livre em volta), raios e
// o tipo de margem.
type pond struct {
	name   string
	water  int
	x, y   float64
	rx, ry float64
	kind   int // 0 campo, 1 gelo, 2 oásis
}

var pondsBase = []pond{
	{"Lagoa da Vila", waterPondVila, villageX + 210, villageY + 170, 22, 13, 0},
	{"Lagoa dos Juncos", waterPondReeds, hubX - 220, hubY + 140, 26, 15, 0},
	{"Lagoa Gelada", waterPondIce, snowX + 70, snowY - 50, 20, 12, 1},
	{"Lagoa do Oásis", waterPondOasis, desertX - 130, desertY + 70, 18, 11, 2},
}

// Horas: bits de madrugada (0), dia (1), tarde/crepúsculo (2), noite (3).
const (
	tDawn  = 1 << 0
	tDay   = 1 << 1
	tDusk  = 1 << 2
	tNight = 1 << 3
	tAny   = 15
	sAny   = 15 // estações: primavera 1, verão 2, outono 4, inverno 8
)

type fishDef struct {
	Name    string
	Rarity  int
	Price   int
	Waters  []int
	Seasons int
	Hours   int
	Weight  int // chance relativa entre os que valem ali
	Color   rgb
}

var fishes = []fishDef{
	{"Sardinha Prateada", rarCommon, 12, []int{waterSea}, sAny, tAny, 60, 0xc8d0e0},
	{"Peixe-Lua", rarRare, 140, []int{waterSea}, sAny, tNight, 10, 0xe0e0b0},
	{"Moreia Negra", rarRare, 160, []int{waterSea}, 4 | 8, tNight | tDusk, 8, 0x3a3040},
	{"Atum-Rei", rarEpic, 320, []int{waterSea}, 2, tDay, 4, 0x4060a0},
	{"Carpa Espelhada", rarCommon, 15, []int{waterLake}, sAny, tAny, 60, 0xd0b070},
	{"Lúcio do Lago", rarUncommon, 55, []int{waterLake}, sAny, tDawn | tDusk, 20, 0x7a9a5a},
	{"Enguia-Sombra", rarRare, 150, []int{waterLake, waterRiver}, sAny, tNight, 9, 0x403850},
	{"O Velho do Lago", rarLegend, 900, []int{waterLake}, 8, tNight, 1, 0xa0a8c0},
	{"Truta Rajada", rarCommon, 18, []int{waterRiver}, 1 | 2, tAny, 55, 0xc08a70},
	{"Bagre de Lodo", rarCommon, 14, []int{waterRiver, waterPondReeds}, sAny, tNight | tDusk, 40, 0x6a5a40},
	{"Salmão Ferrugem", rarUncommon, 70, []int{waterRiver}, 4, tAny, 18, 0xd06a40},
	{"Lambari", rarCommon, 8, []int{waterPondVila, waterPondReeds}, sAny, tAny, 70, 0xb0b8a0},
	{"Peixe-Dourado", rarUncommon, 60, []int{waterPondVila}, 1 | 2, tDay, 16, 0xffc040},
	{"Peixe-Junco", rarUncommon, 45, []int{waterPondReeds}, sAny, tDawn, 18, 0x8ab060},
	{"Peixe-Gelo", rarUncommon, 65, []int{waterPondIce}, sAny, tAny, 50, 0xb8e0ff},
	{"Truta Glacial", rarRare, 170, []int{waterPondIce}, 8, tDawn | tDay, 10, 0x90c8f0},
	{"Peixe-Areia", rarUncommon, 50, []int{waterPondOasis}, sAny, tAny, 50, 0xe0c080},
	{"Escama de Sol", rarRare, 190, []int{waterPondOasis}, 2, tDay, 9, 0xffe070},
}

const rarUncommon = rarRare // o jogo tem quatro raridades; o "incomum" do peixe usa a cor do raro

// Itens novos (fora do bloco const de rpg.go, acrescentados no init).
var (
	itemFishRod int
	itemFish0   int // primeiro peixe; os outros seguem na ordem de fishes
)

func init() {
	itemFishRod = len(items)
	items = append(items, item{"Vara de Pesca", kindTool, rarCommon, 0, 0, 0, 0, 0xb08850, "E na beira d'água lança a linha. Puxe na mordida."})
	itemFish0 = len(items)
	for _, f := range fishes {
		items = append(items, item{f.Name, kindFish, f.Rarity, 0, 0, 0, 0, f.Color, fishDesc(f)})
	}
	for i := range merchantsBase {
		switch merchantsBase[i].Name {
		case "Ysolde, a herbalista", "A Velha do Charco", "O Eremita da Ilha":
			merchantsBase[i].Stock = append(merchantsBase[i].Stock, [2]int{itemFishRod, 120})
		}
	}
	achievements = append(achievements,
		achievement{"Todas as Águas", "Pesque no mar, no lago, no rio e nas quatro lagoas", 500, func(p *Player) bool { return p.FishBodies == 1<<numWaters-1 }},
		achievement{"Mestre Pescador", "Pegue todos os peixes de Midrath", 1500, func(p *Player) bool { return p.FishKinds == 1<<len(fishes)-1 }},
	)
}

// fishDesc diz onde e quando o peixe aparece (vai na ficha do item).
func fishDesc(f fishDef) string {
	var ws []string
	for _, w := range f.Waters {
		ws = append(ws, waterNames[w])
	}
	s := strings.Join(ws, ", ")
	if f.Seasons != sAny {
		var ss []string
		for i, n := range []string{"primavera", "verão", "outono", "inverno"} {
			if f.Seasons&(1<<i) != 0 {
				ss = append(ss, n)
			}
		}
		s += " · " + strings.Join(ss, ", ")
	}
	if f.Hours != tAny {
		var hs []string
		for i, n := range []string{"madrugada", "dia", "fim de tarde", "noite"} {
			if f.Hours&(1<<i) != 0 {
				hs = append(hs, n)
			}
		}
		s += " · " + strings.Join(hs, ", ")
	}
	return s
}

func fishPrice(it int) int {
	if k := it - itemFish0; k >= 0 && k < len(fishes) {
		return fishes[k].Price
	}
	return 0
}

// buildPonds cava as lagoas: procura em volta do centro pedido um lugar
// longe de trilha, praça, cais, boca de caverna e marco; a margem é de areia
// e junco (de gelo na Lagoa Gelada, de areia clara e palmeira no oásis).
func (wd *World) buildPonds(seed uint32) {
	for _, base := range pondsBase {
		pd := base
		ok := false
	search:
		for r := 0.0; r <= 160; r += 20 {
			for a := 0.0; a < 2*math.Pi; a += math.Pi / 4 {
				pd.x, pd.y = base.x+math.Cos(a)*r, base.y+math.Sin(a)*r*0.7
				if wd.pondFree(pd) {
					ok = true
					break search
				}
				if r == 0 {
					break
				}
			}
		}
		if !ok {
			pd = base // sem lugar ideal: fica onde foi pedida
		}
		wd.ponds = append(wd.ponds, pd)
		for y := int(pd.y - pd.ry - 5); y <= int(pd.y+pd.ry+5); y++ {
			for x := int(pd.x - pd.rx - 7); x <= int(pd.x+pd.rx+7); x++ {
				d := math.Hypot((float64(x)-pd.x)/pd.rx, (float64(y)-pd.y)/pd.ry) + (fbm(float64(x)/8, float64(y)/8, seed+1500)-0.5)*0.35
				switch {
				case d < 1:
					if pd.kind == 1 {
						i := y*wd.w + x
						wd.sea(x, y, seed)
						wd.glyph[i], wd.color[i] = pick("~-=", hash2(x, y, seed+1501)), mix(0x7aa0c8, 0xc8e0f0, rnd01(x, y, seed+1502))
					} else {
						wd.sea(x, y, seed)
						if d > 0.86 && rnd01(x, y, seed+1503) < 0.12 && pd.kind == 0 {
							i := y*wd.w + x
							wd.glyph[i], wd.color[i] = 'o', 0x5a8a3a // vitória-régia
						}
					}
				case d < 1.22:
					c := []rgb{0x8a7a50, 0xd0e0f0, 0xd8c088}[pd.kind]
					ch := pick(".,:", hash2(x, y, seed+1504))
					if pd.kind == 0 && rnd01(x, y, seed+1505) < 0.28 {
						ch, c = pick("|!/", hash2(x, y, seed+1506)), dim(0x6a9a4a, 0.7+0.3*rnd01(x, y, seed+1507)) // junco
					}
					wd.put(x, y, ch, dim(c, 0.75+0.25*rnd01(x, y, seed+1508)), false, false)
				}
			}
		}
		if pd.kind == 2 { // palmeiras no oásis
			for k := 0; k < 3; k++ {
				a := float64(k)*2.1 + 0.4
				px, py := int(pd.x+math.Cos(a)*(pd.rx+5)), int(pd.y+math.Sin(a)*(pd.ry+3))
				for dy := 0; dy < 8; dy++ {
					wd.put(px, py-dy, '|', 0x8a6a40, dy == 0, dy > 1)
				}
				for dx := -4; dx <= 4; dx++ {
					wd.put(px+dx, py-8+abs(dx)/2, pick("/\\~", hash2(dx, k, seed+1509)), 0x5aa04a, false, true)
				}
			}
		}
	}
}

func (wd *World) pondFree(pd pond) bool {
	if pathDist(pd.x, pd.y) < pd.rx+10 || plazaDist(pd.x, pd.y) < 1.6 || nearFeature(pd.x, pd.y, pd.rx+30) ||
		lakeDist(pd.x, pd.y, 0) < 1.6 || riverDist(pd.x, pd.y, 0) < pd.rx+12 || castleDist(pd.x, pd.y) < 1.5 {
		return false
	}
	for _, b := range wd.bonfires {
		if dist(pd.x, pd.y, float64(b.X), float64(b.Y)) < pd.rx+30 {
			return false
		}
	}
	for y := int(pd.y - pd.ry - 3); y <= int(pd.y+pd.ry+3); y += 2 {
		for x := int(pd.x - pd.rx - 4); x <= int(pd.x+pd.rx+4); x += 2 {
			if !wd.inBounds(x, y) || wd.water[y*wd.w+x] {
				return false
			}
		}
	}
	return true
}

// waterAt diz que água é a célula (para a pesca), ou -1.
func (wd *World) waterAt(x, y int) int {
	if !wd.inBounds(x, y) || !wd.water[y*wd.w+x] || y >= overH {
		return -1
	}
	px, py := float64(x), float64(y)
	for _, pd := range wd.ponds {
		if math.Hypot((px-pd.x)/pd.rx, (py-pd.y)/pd.ry) < 1.4 {
			return pd.water
		}
	}
	if lakeDist(px, py, wd.seed) < 1.3 {
		return waterLake
	}
	if riverDist(px, py, wd.seed) < riverHalf+3 {
		return waterRiver
	}
	return waterSea
}

type fishTry struct {
	x, y   int
	water  int
	biteAt time.Time
	until  time.Time
	px, py int // onde o jogador estava (andar cancela)
}

// startFish lança a linha. Chamar com o lock; true se lançou.
func (wd *World) startFish(p *Player, now time.Time) bool {
	if p.count(itemFishRod) == 0 {
		return false
	}
	// A boia cai a umas 10 células (perto demais ela some atrás do corpo).
	bx, by, bw, bd := 0, 0, -1, 1e9
	for dy := -26; dy <= 10; dy++ {
		for dx := -18; dx <= 18; dx++ {
			if w := wd.waterAt(p.X+dx, p.Y+dy); w >= 0 {
				d := math.Hypot(float64(dx), float64(dy)*1.5)
				if abs(dx) <= 5 && dy >= -20 && dy <= 1 {
					continue // atrás do corpo do personagem a boia não aparece
				}
				if score := math.Abs(d - 10); d > 2 && score < bd {
					bx, by, bw, bd = p.X+dx, p.Y+dy, w, score
				}
			}
		}
	}
	if bw < 0 {
		return false
	}
	bite := now.Add(time.Duration(2000+wd.rng.Intn(5000)) * time.Millisecond)
	p.fish = &fishTry{x: bx, y: by, water: bw, biteAt: bite, until: bite.Add(900 * time.Millisecond), px: p.X, py: p.Y}
	return true
}

// Reel puxa a linha.
func (wd *World) Reel(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.fish == nil {
		return
	}
	now, f := wd.clock(), p.fish
	p.fish = nil
	switch {
	case now.Before(f.biteAt):
		wd.pop(p, "Puxou cedo demais: o peixe se espantou", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
		return
	case now.After(f.until):
		wd.pop(p, "O peixe escapou", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
		return
	}
	k := wd.pickFish(f.water, now)
	if k < 0 {
		wd.pop(p, "Nada mordeu de verdade…", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
		return
	}
	if !p.give(itemFish0+k, 1) {
		wd.pop(p, "A mochila está cheia", 0xd08060, float64(p.X), float64(p.Y)-34, now)
		return
	}
	first := p.FishKinds&(1<<k) == 0
	p.FishKinds |= 1 << k
	p.FishBodies |= 1 << f.water
	msg := "Pescou: " + fishes[k].Name
	if first {
		msg += " (novo!)"
	}
	wd.pop(p, msg, int(items[itemFish0+k].Color), float64(p.X), float64(p.Y)-40, now)
	if fishes[k].Rarity >= rarEpic {
		wd.say(ChatLine{Text: fmt.Sprintf("%s pescou um %s!", p.Name, fishes[k].Name)})
	}
	wd.achieve(p, now)
}

// pickFish sorteia o peixe da água pela estação e hora.
func (wd *World) pickFish(water int, now time.Time) int {
	t := now.Sub(wd.start).Seconds()
	tod := wd.timeOfDay(t)
	hour := tNight
	switch {
	case tod >= 0.2 && tod < 0.32:
		hour = tDawn
	case tod >= 0.32 && tod < 0.7:
		hour = tDay
	case tod >= 0.7 && tod < 0.82:
		hour = tDusk
	}
	season := 1 << seasonOf(wd.dayOf(t))
	total := 0
	var ok []int
	for i, f := range fishes {
		if f.Hours&hour == 0 || f.Seasons&season == 0 {
			continue
		}
		for _, w := range f.Waters {
			if w == water {
				ok = append(ok, i)
				total += f.Weight
			}
		}
	}
	if total == 0 {
		return -1
	}
	r := wd.rng.Intn(total)
	for _, i := range ok {
		if r -= fishes[i].Weight; r < 0 {
			return i
		}
	}
	return ok[0]
}

// stepFish cancela a pesca de quem andou ou deixou passar muito tempo.
func (wd *World) stepFish(p *Player, now time.Time) {
	if f := p.fish; f != nil && (abs(p.X-f.px) > 2 || abs(p.Y-f.py) > 2 || now.After(f.until.Add(3*time.Second))) {
		p.fish = nil
	}
}

// FishCast é a tecla de pesca: lança a linha, ou puxa se ela já está na água.
func (wd *World) FishCast(id int) {
	wd.mu.Lock()
	p, ok := wd.players[id]
	if !ok {
		wd.mu.Unlock()
		return
	}
	if p.fish != nil {
		wd.mu.Unlock()
		wd.Reel(id)
		return
	}
	defer wd.mu.Unlock()
	now := wd.clock()
	switch {
	case p.count(itemFishRod) == 0:
		wd.pop(p, "Precisa de uma Vara de Pesca (Ysolde vende)", 0xc0a070, float64(p.X), float64(p.Y)-34, now)
	case !wd.startFish(p, now):
		wd.pop(p, "Sem água por perto para lançar a linha", 0x9ab0d0, float64(p.X), float64(p.Y)-34, now)
	}
}
