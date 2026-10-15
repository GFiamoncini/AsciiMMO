package main

import (
	"fmt"
	"math"
	"sort"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	hudBG    rgb = 0x000000
	hudLine  rgb = 0xcfcfcf
	hudFG    rgb = 0xe6e6e6
	hudDim   rgb = 0x8a8a96
	hpCol    rgb = 0xd03a3a
	mpCol    rgb = 0x3a6ad0
	barEmpty rgb = 0x33333d
	meCol    rgb = 0xffe066

	panelW   = 32
	chatRows = 6
)

var playerColors = []rgb{0xff6b8b, 0xffa94d, 0x8ce99a, 0x66d9e8, 0xb197fc, 0xf783ac, 0xffd43b, 0x74c0fc}

func playerColor(c int) rgb { return playerColors[c%len(playerColors)] }

var (
	dragonBody = []rune(".:-=+*%#8@")
	dragonWing = []rune("8#%8H8")
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// perf fica atrás de ponteiro porque o View recebe o model por valor.
type perf struct {
	last   time.Duration
	frames int
	fps    int
	since  time.Time
}

type model struct {
	world         *World
	id            int
	width, height int
	chatting      bool
	input         []rune
	perf          *perf
}

func newModel(w *World, id, width, height int) model {
	return model{world: w, id: id, width: width, height: height, perf: &perf{since: time.Now()}}
}

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		// O View relê o mundo a cada tick; o bubbletea só reenvia as linhas que mudaram.
		return m, tick()
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m.quit()
		}
		if m.chatting {
			return m.updateChat(msg), nil
		}
		switch msg.String() {
		case "q":
			return m.quit()
		case "enter", "t":
			m.chatting = true
		case " ":
			m.world.Use(m.id, 0, nil)
		case "f":
			m.world.Use(m.id, 1, nil)
		case "r":
			m.world.Use(m.id, 2, nil)
		case "up", "w", "k":
			m.world.Move(m.id, 0, -1)
		case "down", "s", "j":
			m.world.Move(m.id, 0, 1)
		case "left", "a", "h":
			m.world.Move(m.id, -1, 0)
		case "right", "d", "l":
			m.world.Move(m.id, 1, 0)
		}
	}
	return m, nil
}

// quit tira o jogador do mundo já na saída; a conexão SSH pode demorar a cair.
func (m model) quit() (tea.Model, tea.Cmd) {
	m.world.Leave(m.id)
	return m, tea.Quit
}

func (m model) updateChat(msg tea.KeyMsg) model {
	switch msg.Type {
	case tea.KeyEnter:
		m.world.Say(m.id, string(m.input))
		m.chatting, m.input = false, nil
	case tea.KeyEsc:
		m.chatting, m.input = false, nil
	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	case tea.KeySpace, tea.KeyRunes:
		m.input = append(m.input, msg.Runes...)
		if len(m.input) > maxChatLen {
			m.input = m.input[:maxChatLen]
		}
	}
	return m
}

func (m model) View() string {
	if m.width < 60 || m.height < 20 {
		return "Terminal pequeno demais: maximize a janela e diminua a fonte (Ctrl e -)."
	}
	start := time.Now()
	s := m.world.Snapshot(m.id)
	wd := m.world
	f := newFrame(m.width, m.height)

	// O terminal é o modo simples: cada linha cobre duas células do mundo, já
	// que um caractere tem o dobro da altura. (ox, oy) é o canto da tela, em
	// colunas e linhas; a câmera segue o jogador e para nas bordas do mapa.
	vw, vh := min(f.w, wd.w), min(f.h, wd.h/2)
	ox := clamp(s.Me.X-vw/2, 0, wd.w-vw) - (f.w-vw)/2
	oy := clamp(s.Me.Y/2-vh/2, 0, wd.h/2-vh) - (f.h-vh)/2
	scr := func(x, y float64) (int, int) {
		return int(math.Round(x)) - ox, int(math.Round(y/2)) - oy
	}

	flick := 0.8 + 0.2*math.Sin(s.T*9)
	for sy := 0; sy < f.h; sy++ {
		for sx := 0; sx < f.w; sx++ {
			if wx, wy := sx+ox, 2*(sy+oy); wd.inBounds(wx, wy) {
				i := wy*wd.w + wx
				c := cell{rune(wd.glyph[i]), wd.color[i], noBG}
				if dx, dy := wx-fireX, (wy-fireY)/2; dx > -28 && dx < 28 && dy > -14 && dy < 14 {
					if d := math.Hypot(float64(dx)/2, float64(dy)); d < 14 {
						c.fg = mix(c.fg, 0xffa040, (1-d/14)*flick)
					}
				}
				if h := hash2(wx, wy, 7); h%45 == 0 && !wd.solid[i] && (int(s.T*3)+int(h>>8))%8 < 6 {
					c.ch, c.fg = '.', 0xfff4c0
					if h>>16&3 == 0 {
						c.ch = '*'
					}
				}
				f.cells[sy*f.w+sx] = c
			}
		}
	}

	fx, fy := scr(fireX, fireY)
	drawFire(f, fx, fy, s.T)

	bx, by := scr(s.Boss.X, s.Boss.Y)
	if s.Boss.Alive {
		m.drawDragon(f, bx+18, by+9, s.T, true)
	}
	for _, e := range s.Effects {
		if e.Kind == fxWarn {
			ex, ey := scr(e.X, e.Y)
			f.ring(ex, ey, e.R/2, 0, 2*math.Pi, 0xff5a40)
			f.ring(ex, ey, e.R/2*e.P, 0, 2*math.Pi, 0x992020)
		}
	}

	for _, mb := range s.Mobs {
		x, y := scr(mb.X, mb.Y)
		drawMob(f, mb, x, y)
	}
	players := append([]Player(nil), s.Players...)
	sort.Slice(players, func(i, j int) bool { return players[i].Y < players[j].Y })
	for _, p := range players {
		if p.ID != s.Me.ID {
			x, y := scr(float64(p.X), float64(p.Y))
			drawPlayer(f, p, x, y, false)
		}
	}
	if s.Boss.Alive {
		m.drawDragon(f, bx, by, s.T, false)
		label(f, bx, by, "Dragão", s.Boss.HP, s.Boss.MaxHP)
	}
	mx, my := scr(float64(s.Me.X), float64(s.Me.Y))
	drawPlayer(f, s.Me, mx, my, true)

	for _, e := range s.Effects {
		ex, ey := scr(e.X, e.Y)
		switch e.Kind {
		case fxSlash:
			f.ring(ex, ey-1, e.R/2, e.Dir-1.1, e.Dir+1.1, 0xffffff)
		case fxNova:
			f.ring(ex, ey-1, e.R/2*e.P, 0, 2*math.Pi, 0x9ad0ff)
		case fxBlast:
			f.ring(ex, ey, e.R/2*e.P, 0, 2*math.Pi, 0xff7060)
			f.ring(ex, ey, math.Max(0, e.R/2*e.P-1), 0, 2*math.Pi, 0xc03020)
		case fxPuff:
			f.ring(ex, ey-1, e.R/2*e.P, 0, 2*math.Pi, 0xd8c8a8)
		case fxSpin, fxBoom, fxHeal, fxHoly, fxMeteor, fxRain, fxDark:
			f.ring(ex, ey-1, e.R/2*e.P, 0, 2*math.Pi, 0xffe0a0)
		}
	}

	m.drawHUD(f, s)
	m.drawChat(f, s)

	out := f.String()
	pf := m.perf
	pf.last = time.Since(start)
	pf.frames++
	if el := time.Since(pf.since); el >= time.Second {
		pf.fps, pf.frames, pf.since = int(float64(pf.frames)/el.Seconds()+0.5), 0, time.Now()
	}
	return out
}

// drawDragon pinta o chefe centrado em (cx, cy). Como sombra, só escurece o
// que já está na tela, deslocado, para dar a impressão de voo.
func (m model) drawDragon(f *frame, cx, cy int, t float64, shadow bool) {
	const bg rgb = 0x08080a
	for dy := -16; dy <= 21; dy++ {
		for dx := -48; dx <= 48; dx++ {
			x, y := cx+dx, cy+dy
			if !f.in(x, y) {
				continue
			}
			px, py := float64(dx)/2, float64(dy)
			class := dragonClass(px, py, t)
			if class == dgNone {
				continue
			}
			if shadow {
				c := &f.cells[y*f.w+x]
				c.fg = dim(c.fg, 0.12)
				continue
			}
			// Luz vindo da esquerda e de cima: o lado direito some no escuro.
			h := hash2(dx, dy, 99)
			l := 0.72 - px*0.022 - py*0.012 + (float64(h&0xff)/255-0.5)*0.3
			l = math.Max(0.28, math.Min(1, l))
			ch := dragonBody[int(l*float64(len(dragonBody)-1))]
			switch class {
			case dgEye:
				f.set(x, y, 'O', 0xffe060, 0x200808)
			case dgSpine:
				f.set(x, y, '#', dim(0xf0b890, l), bg)
			case dgHead:
				f.set(x, y, ch, dim(0xd87848, l), bg)
			case dgBody:
				f.set(x, y, ch, dim(0xc8643a, l), bg)
			case dgBone:
				f.set(x, y, '#', dim(0xe8a070, math.Min(1, l+0.2)), bg)
			case dgWing:
				f.set(x, y, dragonWing[h%uint32(len(dragonWing))], dim(0xb5522e, l), bg)
			}
		}
	}
}

// label escreve um nome com uma barrinha de vida embaixo, sobre o que já está na tela.
func label(f *frame, x, y int, name string, cur, max int) {
	n := utf8.RuneCountInString(name)
	f.text(x-n/2, y, name, 0xffffff, 0)
	f.bar(x-n/2, y+1, n, cur, max, 0xe8e8e8, 0x303030)
}

// drawFire desenha a fogueira com a base em (x, y); as chamas mudam a cada décimo de segundo.
func drawFire(f *frame, x, y int, t float64) {
	step := uint32(t * 10)
	for dy := 0; dy <= 2; dy++ {
		for dx := -2 + dy; dx <= 2-dy; dx++ {
			h := hash2(dx, dy, step)
			if dy > 0 && h%4 == 0 {
				continue
			}
			f.glyph(x+dx, y-dy, []rune("^*)(W")[h>>4%5], mix(0xff5a1a, 0xffe070, float64(h>>8&0xff)/255))
		}
	}
	f.text(x-2, y+1, "=====", 0x6b4a2a, noBG)
}

// drawPlayer desenha o boneco com os pés em (x, y) e o nome em cima.
func drawPlayer(f *frame, p Player, x, y int, me bool) {
	col, nameCol := playerColor(p.Color), rgb(0xffffff)
	if me {
		nameCol = meCol
	}
	f.set(x, y-1, 'o', 0xffffff, noBG)
	f.set(x-1, y, '/', col, noBG)
	f.set(x, y, '#', col, noBG)
	f.set(x+1, y, '\\', col, noBG)
	nx := x - utf8.RuneCountInString(p.Name)/2
	for _, r := range p.Name {
		f.glyph(nx, y-2, r, nameCol)
		nx++
	}
}

// drawMob desenha um esqueleto com os pés em (x, y), nome e vida em cima.
func drawMob(f *frame, mb Mob, x, y int) {
	const bone rgb = 0xd8c8a8
	f.set(x, y-1, 'o', bone, noBG)
	f.set(x-1, y, '/', bone, noBG)
	f.set(x, y, '|', bone, noBG)
	f.set(x+1, y, '\\', bone, noBG)
	label(f, x, y-3, mobKinds[mb.Kind].Name, mb.HP, mb.MaxHP)
}

func stat(f *frame, x, y int, label string, cur, max int) {
	f.text(x, y, label, hudFG, hudBG)
	f.ticks(x+3, y, 14, cur, max, hudFG, 0x3a3a3a)
	f.text(x+18, y, fmt.Sprintf("%d/%d", cur, max), hudFG, hudBG)
}

func (m model) drawHUD(f *frame, s Snapshot) {
	f.box(0, 0, panelW, 6, hudLine)
	f.text(2, 1, s.Me.Name, hudFG, hudBG)
	stat(f, 2, 2, "HP", s.Me.HP, s.Me.MaxHP)
	stat(f, 2, 3, "MP", s.Me.MP, s.Me.MaxMP)
	f.text(2, 4, fmt.Sprintf("LV %d · %dg", s.Me.Level, s.Me.Gold), hudFG, hudBG)

	y, shown := 6, 0
	for _, p := range s.Players {
		if p.ID == s.Me.ID || shown == 4 || y+5 > f.h-chatRows-3 {
			continue
		}
		f.box(0, y, panelW, 5, hudLine)
		f.text(2, y+1, fmt.Sprintf("%s · %d", p.Name, p.Level), hudFG, hudBG)
		stat(f, 2, y+2, "HP", p.HP, p.MaxHP)
		stat(f, 2, y+3, "MP", p.MP, p.MaxMP)
		y, shown = y+5, shown+1
	}
	if shown > 0 {
		f.box(0, y, 9, 3, hudLine)
		f.text(2, y+1, "party", hudFG, hudBG)
	}

	// Chefe: caixa com o nome no topo e a barra vermelha atravessando a tela.
	if bw := f.w - panelW - 2; bw >= 30 {
		label := fmt.Sprintf("Dragão %d/%d", s.Boss.HP, s.Boss.MaxHP)
		if !s.Boss.Alive {
			label = fmt.Sprintf("Dragão derrotado · renasce em %ds", int(s.Boss.RespawnIn.Seconds())+1)
		}
		nw := min(48, bw-28)
		f.box(panelW+1, 0, nw, 3, hudLine)
		f.text(panelW+1+(nw-utf8.RuneCountInString(label))/2, 1, label, hudFG, hudBG)
		f.bar(panelW+1, 3, bw, s.Boss.HP, s.Boss.MaxHP, 0xe0402a, 0x3a0c08)
		f.glyph(panelW+1+bw/3, 3, '│', 0x1a0504)
		f.glyph(panelW+1+2*bw/3, 3, '│', 0x1a0504)
	}

	perf := fmt.Sprintf("FPS %d · frame %.1f ms", m.perf.fps, float64(m.perf.last.Microseconds())/1000)
	x := f.w - 1 - utf8.RuneCountInString(perf)
	for _, r := range perf {
		f.glyph(x, 1, r, 0x7dff7d)
		x++
	}
}

func (m model) drawChat(f *frame, s Snapshot) {
	chat := s.Chat
	if len(chat) > chatRows {
		chat = chat[len(chat)-chatRows:]
	}
	y := f.h - 1 - len(chat)
	for _, l := range chat {
		if l.From == "" {
			f.text(1, y, truncate(" * "+l.Text+" ", f.w-2), hudDim, hudBG)
		} else {
			name := " " + l.From
			f.text(1, y, name, playerColor(l.Color), hudBG)
			f.text(1+utf8.RuneCountInString(name), y, truncate(": "+l.Text+" ", f.w-2-utf8.RuneCountInString(name)), hudFG, hudBG)
		}
		y++
	}

	f.fill(0, f.h-1, f.w, 1, hudBG)
	if !m.chatting {
		f.text(1, f.h-1, truncate("setas/WASD: mover · Espaço, F, R: golpes · Enter: falar · Q: sair", f.w-2), hudDim, hudBG)
		return
	}
	in := m.input
	if room := f.w - 5; len(in) > room {
		in = in[len(in)-room:]
	}
	f.text(1, f.h-1, "> "+string(in)+"_", hudFG, hudBG)
}

func truncate(s string, w int) string {
	r := []rune(s)
	if w < 0 {
		w = 0
	}
	if len(r) > w {
		return string(r[:w])
	}
	return s
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}
