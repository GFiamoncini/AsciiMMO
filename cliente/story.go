package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------- textos flutuantes, morte e capítulo ----------

type floatText struct {
	text string
	c    col
	x, y float64
	t0   float64
}

// takePops recebe os textos novos do servidor: XP, dinheiro, avisos; o de
// morte vira a tela "Você pereceu".
func (g *Game) takePops(s *proto.State) {
	now := g.now()
	for _, p := range s.Pops {
		if g.popSeen[p.ID] {
			continue
		}
		g.popSeen[p.ID] = true
		if p.Kind == 1 {
			g.deathAt = now
			continue
		}
		g.floats = append(g.floats, floatText{p.Text, hexc(p.Color), p.X, p.Y, now})
	}
	if me := g.pmapMe(s); s.Quest != nil && me != nil {
		g.maybeOpening(me.Name, s.Quest.Chapter, s.Quest.Cur)
	}
	if s.Quest != nil && s.Quest.Chapter == 0 {
		// no prólogo o letreiro do capítulo espera: aparece quando ele acaba
	} else if s.Chapter != g.lastChapter {
		if s.Chapter >= 1 && s.Chapter <= len(g.info.Chapters) {
			g.chapterAt = now
			g.sfx("holy", 0.5)
		}
		g.lastChapter = s.Chapter
	}
}

// drawFloats desenha os textos subindo e sumindo sobre o mundo.
func (g *Game) drawFloats(dst *ebiten.Image, camL, camT float64) {
	now, cs := g.now(), float64(g.cs)
	n := 0
	for _, f := range g.floats {
		age := (now - f.t0) / 2200
		if age >= 1 {
			continue
		}
		g.floats[n] = f
		n++
		x := (f.x - camL) * cs / g.ui
		y := (f.y-camT)*cs/g.ui - 26*age
		a := uint8(255 * math.Min(1, (1-age)*1.6))
		c := f.c.rgba()
		g.txt(dst, f.text, x+1, y+1, color.RGBA{0, 0, 0, a}, 14, true, alignCenter)
		g.txt(dst, f.text, x, y, color.RGBA{uint8(int(c.R) * int(a) / 255), uint8(int(c.G) * int(a) / 255), uint8(int(c.B) * int(a) / 255), a}, 14, true, alignCenter)
	}
	g.floats = g.floats[:n]
}

// drawOverlays mostra a abertura do capítulo e a tela de morte.
func (g *Game) drawOverlays(dst *ebiten.Image, cw, chh float64) {
	now := g.now()
	if g.cur.Event == 1 { // invasão noturna: aviso fixo no alto da tela
		a := 0.75 + 0.25*math.Sin(now/300)
		g.rect(dst, cw/2-260, 52, 520, 26, color.RGBA{30, 0, 0, 200})
		g.txt(dst, "Invasão! Os mortos atacam Cinzalva — a fogueira se apagou", cw/2, 57, color.RGBA{uint8(255 * a), uint8(80 * a), uint8(60 * a), 255}, 14, true, alignCenter)
	}
	if t := (now - g.deathAt) / 3500; g.deathAt > 0 && t < 1 {
		a := math.Min(1, math.Min(t*4, (1-t)*2.5))
		g.rect(dst, 0, chh*0.38, cw, chh*0.2, color.RGBA{0, 0, 0, uint8(200 * a)})
		g.txt(dst, "Você pereceu", cw/2, chh*0.42, color.RGBA{uint8(190 * a), uint8(24 * a), uint8(24 * a), uint8(255 * a)}, 44, true, alignCenter)
		g.txt(dst, "a chama da última fogueira te chama de volta", cw/2, chh*0.42+56, color.RGBA{uint8(140 * a), uint8(120 * a), uint8(110 * a), uint8(255 * a)}, 14, false, alignCenter)
	}
	g.drawChapterCard(dst, cw, chh, now) // card animado do capítulo (historia.go)
}

// ---------- mercadores ----------

var shopStyles = []bodyStyle{
	{head: "hair", hair: pc(60, 56, 54), torso: pc(96, 70, 48), torso2: pc(60, 44, 30), belt: pc(40, 30, 22), legs: pc(70, 62, 58),
		arm: pc(196, 156, 126), skin: pc(196, 156, 126), boots: pc(50, 40, 34), weapon: "hammer", wlen: 9, seed: 95},
	{head: "hood", hair: pc(50, 80, 56), torso: pc(70, 90, 60), torso2: pc(170, 200, 120), robe: true, arm: pc(70, 90, 60),
		skin: pc(214, 186, 160), weapon: "staff", wlen: 12, orb: pc(150, 230, 140), seed: 96},
}

const shopReach = 14.0

// nearShop devolve o mercador ao alcance, ou -1.
func (g *Game) nearShop(me *proto.Player) int {
	for i, s := range g.info.Shops {
		if math.Hypot(float64(s.X-me.X), float64(s.Y-me.Y)) <= shopReach {
			return i
		}
	}
	return -1
}

// drawShops desenha os mercadores parados, olhando para o jogador.
func (g *Game) drawShops(me *proto.Player, T float64) {
	for i, s := range g.info.Shops {
		face := 1
		if me.X < s.X {
			face = -1
		}
		st := &shopStyles[clampi(s.Kind, 0, len(shopStyles)-1)]
		if i == 2 { // a velha do charco: manto roxo e luz doentia
			v := *st
			v.torso, v.arm, v.orb, v.hair, v.seed = pc(70, 50, 80), pc(70, 50, 80), pc(190, 255, 90), pc(140, 140, 140), 97
			st = &v
		}
		g.drawBody(bodyDraw{style: st, x: s.X, y: s.Y, T: T, hf: 1, face: face, tint: white})
		g.light(float64(s.X), float64(s.Y)-8, 30, 0.6, warm, col{60, 40, 10})
		g.labels = append(g.labels, label{x: float64(s.X), y: float64(s.Y) - 31, text: s.Name, color: col{255, 210, 140}})
	}
}

// sellPrice é a mesma conta do servidor (rpg.go): metade do preço, se o
// mercador vende o item; senão, pelo tipo e pela raridade.
func (g *Game) sellPrice(s proto.Shop, id int) int {
	for _, st := range s.Stock {
		if st[0] == id {
			return max(1, st[1]/2)
		}
	}
	it := g.info.Items[id]
	switch it.Kind {
	case proto.KindPotion:
		return 15
	case proto.KindMaterial:
		return 3
	case proto.KindHerb:
		return 10
	case proto.KindScroll:
		return 100
	case proto.KindMoney:
		return 0
	case proto.KindFish:
		return it.Price
	}
	return []int{30, 90, 200, 400}[clampi(it.Rarity, 0, 3)]
}

// ---------- selos, fogueiras e missões secundárias ----------

// sealedBy devolve a secundária que ainda sela a região, ou -1.
func (g *Game) sealedBy(region int) int {
	for i, s := range g.info.Sides {
		for _, r := range s.Unlock {
			if r == region && g.cur.SideDone&(1<<i) == 0 {
				return i
			}
		}
	}
	return -1
}

func (g *Game) bonfireLabels(me *proto.Player) {
	for i, b := range g.info.Bonfires {
		if math.Hypot(float64(b.X-me.X), float64(b.Y-me.Y)) > 90 {
			continue
		}
		t, c := b.Name, col{255, 170, 90}
		if i == g.cur.Bonfire {
			t += " (acesa)"
			c = col{255, 210, 120}
		}
		g.labels = append(g.labels, label{x: float64(b.X), y: float64(b.Y) - 16, text: t, color: c})
	}
}

// ---------- opções ----------

type options struct {
	Fullscreen     bool `json:"tela_cheia"`
	ShowFPS        bool `json:"mostrar_fps"`
	Mode           int  `json:"modo_personagens"` // 0 = 2D, 1 = Mescla (2D com luz 3D), 2 = 3D
	Bloom          bool `json:"brilho"`
	FOVClassic     bool `json:"campo_visao_classico"` // esconde o que está fora da vista, como nos roguelikes
	NoDepth        bool `json:"sem_profundidade"`     // desliga o volume do cenário (depth.go)
	QuestArrow     bool `json:"seta_missao"`          // seta na borda da tela apontando a missão (padrão desligada: explorar)
	HideMenu       bool `json:"esconder_menu"`        // esconde o menu de teclas (lista da direita e linha H/L)
	NoCompass      bool `json:"sem_bussola"`          // esconde a bússola no alto da tela
	VolMusic       int  `json:"volume_musica"`        // 0..100
	VolSfx         int  `json:"volume_efeitos"`       // 0..100
	VolAmb         int  `json:"volume_ambiente"`      // 0..100
	MuteUnfocused  bool `json:"mudo_sem_foco"`        // cala o jogo quando a janela perde o foco
	NoPauseInMenus bool `json:"nao_pausar_menus"`     // janelas abertas não pausam (pausa.go)
	NoSmoothLight  bool `json:"sem_luz_suave"`        // desliga a luz suave da GPU (gfx.go)
}

// Modos de desenho dos personagens.
const (
	mode2D  = 0 // desenho 2D chapado
	modeMix = 1 // Mescla: silhueta 2D com luz de volume; bichos, montarias e Tiamath em 3D
	mode3D  = 2 // tudo em 3D
)

var modeNames = [3]string{"2D", "Mescla", "3D"}

// setMode troca o modo de desenho e guarda nas opções.
func (g *Game) setMode(m int) {
	g.mode, g.opts.Mode = m, m
	g.use3D = m == mode3D
	g.saveOptions()
}

func optPath() string { return filepath.Join(configDir(), "opcoes.json") }

func loadOptions() options {
	o := options{ShowFPS: true, Mode: modeMix, Bloom: true, VolMusic: 50, VolSfx: 80, VolAmb: 70}
	if b, err := os.ReadFile(optPath()); err == nil {
		json.Unmarshal(b, &o)
	}
	return o
}

func (g *Game) saveOptions() {
	b, _ := json.MarshalIndent(g.opts, "", "  ")
	os.WriteFile(optPath(), b, 0o644)
}

func (g *Game) setFullscreen(on bool) {
	g.opts.Fullscreen = on
	ebiten.SetFullscreen(on)
	g.saveOptions()
}

// nextVol é o próximo degrau do volume: 0, 20, 40, 60, 80, 100 e volta.
func nextVol(v int) int {
	if v >= 100 {
		return 0
	}
	return min(100, (v/20+1)*20)
}

// drawOptions é o painel de opções (tecla O).
func (g *Game) drawOptions(dst *ebiten.Image, cw, chh, lh float64) {
	rows := []struct {
		label string
		on    bool
		fn    func()
	}{
		{"Tela cheia  (F11)", g.opts.Fullscreen, func() { g.setFullscreen(!g.opts.Fullscreen) }},
		{"Mostrar FPS", g.opts.ShowFPS, func() { g.opts.ShowFPS = !g.opts.ShowFPS; g.saveOptions() }},
		{"Personagens: " + modeNames[g.mode] + "  (clique troca)", g.mode != mode2D, func() { g.setMode((g.mode + 1) % 3) }},
		{"Brilho (bloom)", g.opts.Bloom, func() { g.opts.Bloom = !g.opts.Bloom; g.saveOptions() }},
		{"Campo de visão clássico", g.opts.FOVClassic, func() { g.opts.FOVClassic = !g.opts.FOVClassic; g.saveOptions() }},
		{"Profundidade do cenário", !g.opts.NoDepth, func() { g.opts.NoDepth = !g.opts.NoDepth; g.saveOptions() }},
		{"Seta da missão na tela", g.opts.QuestArrow, func() { g.opts.QuestArrow = !g.opts.QuestArrow; g.saveOptions() }},
		{"Menu de teclas na tela", !g.opts.HideMenu, func() { g.opts.HideMenu = !g.opts.HideMenu; g.saveOptions() }},
		{"Bússola no alto da tela", !g.opts.NoCompass, func() { g.opts.NoCompass = !g.opts.NoCompass; g.saveOptions() }},
		{fmt.Sprintf("Música: %d%%  (clique troca)", g.opts.VolMusic), g.opts.VolMusic > 0, func() { g.opts.VolMusic = nextVol(g.opts.VolMusic); g.saveOptions() }},
		{fmt.Sprintf("Efeitos: %d%%  (clique troca)", g.opts.VolSfx), g.opts.VolSfx > 0, func() { g.opts.VolSfx = nextVol(g.opts.VolSfx); g.saveOptions(); g.sfx("hit", 0.8) }},
		{fmt.Sprintf("Ambiente: %d%%  (clique troca)", g.opts.VolAmb), g.opts.VolAmb > 0, func() { g.opts.VolAmb = nextVol(g.opts.VolAmb); g.saveOptions() }},
		{"Luz suave (GPU)", !g.opts.NoSmoothLight, func() { g.opts.NoSmoothLight = !g.opts.NoSmoothLight; g.saveOptions() }},
		{"Pausar ao abrir menus", !g.opts.NoPauseInMenus, func() { g.opts.NoPauseInMenus = !g.opts.NoPauseInMenus; g.saveOptions() }},
		{"Calar o jogo sem foco", g.opts.MuteUnfocused, func() { g.opts.MuteUnfocused = !g.opts.MuteUnfocused; g.saveOptions() }},
	}
	const rowH = 30.0
	w, h := 440.0, 44+float64(len(rows))*rowH+100
	x, y := (cw-rightW-w)/2, math.Max(4, (chh-h)/2)
	g.panel(dst, x, y, w, h, cGold)
	g.txt(dst, "Opções (O fecha)", x+14, y+10, cGold, 14, true, alignLeft)
	for i, r := range rows {
		ry := y + 44 + float64(i)*rowH
		box := "[ ]"
		if r.on {
			box = "[x]"
		}
		c := color.Color(cText)
		if g.hover(x+10, ry-4, w-20, rowH-2) {
			c = cGold
		}
		g.txt(dst, box+" "+r.label, x+20, ry, c, 14, false, alignLeft)
		g.uiRects = append(g.uiRects, uiRect{x: x + 10, y: ry - 4, w: w - 20, h: rowH - 2, fn: r.fn})
	}
	// Pasta dos dados: onde ficam os personagens (save) e as coisas do cliente.
	py := y + 44 + float64(len(rows))*rowH + 6
	g.txt(dst, "Pasta dos dados do jogo:", x+20, py, cDim, 12, false, alignLeft)
	root := proto.DataRoot()
	if len([]rune(root)) > 46 {
		root = "…" + string([]rune(root)[len([]rune(root))-45:])
	}
	g.txt(dst, root, x+20, py+16, cText, 12, false, alignLeft)
	bc := color.Color(cGold)
	if g.hover(x+20, py+36, 150, 22) {
		bc = cGreen
	}
	g.stroke(dst, x+20, py+36, 150, 22, bc, 1)
	g.txt(dst, "Escolher pasta…", x+95, py+39, bc, 12, true, alignCenter)
	g.uiRects = append(g.uiRects, uiRect{x: x + 20, y: py + 36, w: 150, h: 22, fn: g.pickDataFolder})
	if g.folderMsg != "" {
		g.txt(dst, g.folderMsg, x+180, py+40, cGreen, 11, false, alignLeft)
	}
	g.uiRects = append(g.uiRects, uiRect{x: x, y: y, w: w, h: h, fn: func() {}})
}

// ---------- ícone da janela ----------

// windowIcon desenha o ícone: o olho de Tiamath, uma íris em brasa com a
// pupila em fenda, sobre fundo escuro.
func windowIcon(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	s := float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := (float64(x)+0.5)/s*2-1, (float64(y)+0.5)/s*2-1
			c := color.RGBA{14, 10, 14, 255}
			if math.Hypot(fx, fy) > 0.98 {
				c = color.RGBA{0, 0, 0, 0}
			}
			// Olho em amêndoa: duas curvas que se encontram nas pontas.
			if math.Abs(fy) < 0.62*(1-fx*fx) {
				r := math.Hypot(fx, fy*1.3)
				k := clamp(1-r, 0, 1)
				c = color.RGBA{uint8(150 + 105*k), uint8(40 + 150*k*k), uint8(10 + 40*k*k*k), 255}
				if math.Abs(fx) < 0.09*(1-math.Abs(fy)*1.2) { // pupila em fenda
					c = color.RGBA{8, 4, 6, 255}
				}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// pickDataFolder abre a janela do sistema para escolher a pasta dos dados;
// a escolha vale para o cliente e, ao reiniciar, para o servidor desta máquina.
func (g *Game) pickDataFolder() {
	if g.picking {
		return
	}
	g.picking = true
	go func() {
		defer func() { g.picking = false }()
		dir, err := chooseFolder(proto.DataRoot())
		if err != nil || dir == "" {
			return
		}
		g.saveFog()
		if err := proto.SetDataRoot(dir); err != nil {
			g.folderMsg = "Não consegui usar essa pasta"
			return
		}
		g.folderMsg = "Pasta trocada (reinicie o servidor)"
	}()
}
