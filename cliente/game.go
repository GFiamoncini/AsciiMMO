package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"asciimmo/proto"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var errQuit = errors.New("sair")

// state é um estado do servidor com índices por id, para interpolar.
type state struct {
	*proto.State
	pmap map[int]*proto.Player
	mmap map[int]*proto.Mob
	jmap map[int]*proto.Proj
	smap map[int]*proto.Thing
}

func wrapState(s *proto.State) *state {
	w := &state{State: s, pmap: map[int]*proto.Player{}, mmap: map[int]*proto.Mob{}, jmap: map[int]*proto.Proj{}, smap: map[int]*proto.Thing{}}
	for i := range s.Players {
		w.pmap[s.Players[i].ID] = &s.Players[i]
	}
	for i := range s.Mobs {
		w.mmap[s.Mobs[i].ID] = &s.Mobs[i]
	}
	for i := range s.Projs {
		w.jmap[s.Projs[i].ID] = &s.Projs[i]
	}
	for i := range s.Steeds {
		w.smap[s.Steeds[i].ID] = &s.Steeds[i]
	}
	return w
}

type uiRect struct {
	x, y, w, h float64
	act        *proto.In
	fn         func()
	alt        *proto.In // botão direito (jogar fora, na mochila)
}

type Game struct {
	server string
	start  time.Time

	m        *gameMap
	mapCh    chan *gameMap
	deepCh   chan *deepPart // Profundezas baixadas (net.go)
	deepBusy bool
	mapErr   string
	loading  bool

	c         *conn
	info      *proto.Init
	cur, prev *state
	tCur      float64

	faces     map[string]int
	fxSeen    map[string]bool
	walkPhase map[string]float64
	parts     []particle
	lights    []lightT
	labels    []label
	ripples   []ripple
	rippleAt  map[string]float64
	flashAt   float64
	shake     float64

	W, H     int
	ui       float64 // pixels do dispositivo por pixel lógico
	cs       int
	csTarget int
	csAt     float64

	buf                         []byte
	cols, rows                  int
	x0, y0                      int
	g                           *gpu
	lastFrame                   float64
	fps, frames                 int
	fpsAt                       float64
	meScreen                    *[2]float64 // centro do corpo do jogador, em pixels do dispositivo
	mouseX, mouseY              float64     // pixels lógicos
	mouseAt                     float64
	lastMX, lastMY              int
	held                        [4]bool // cima, baixo, esquerda, direita
	run                         bool
	lastDir                     string
	spaceAt                     float64
	uiRects                     []uiRect
	openPanel                   string
	bigMap, mapShut             bool
	chatting                    bool
	chatBuf                     []rune
	chatSent                    []string  // o que já foi dito, para voltar com ↑/↓ no chat
	chatPos                     int       // posição em chatSent enquanto se navega (len = linha nova)
	logScroll                   int       // linhas roladas para trás no registro (L)
	pauseMenu                   bool      // menu de pausa aberto (pausa.go)
	fishBit                     bool      // o peixe estava mordendo no quadro passado
	scene                       *sceneBuf // grade da cena da abertura (abertura_cena.go)
	sceneG                      *gpu
	thunderAt                   float64
	introSpoken, introVoicePage int    // narração: palavras já faladas e de que página
	journalTab                  string // aba do diário (diario.go)
	journalScroll, journalMax   float64
	introReplay                 bool               // a abertura foi aberta pelo diário
	tameBuf                     string             // setas já tocadas na doma (doma.go)
	tameSent                    bool               // a tentativa já foi mandada
	xpAt                        float64            // quando chegou XP pela última vez (brilho na barra)
	pauseSent                   bool               // o que foi dito ao servidor sobre a pausa
	savesSeen                   int                // última contagem de gravações vista
	savedAt                     float64            // quando apareceu a última gravação
	snd                         *soundSys          // som (som.go); nulo com -sem-som e nos prints
	stepAcc                     float64            // células andadas desde o último passo tocado
	sndScanAt                   float64            // última olhada em volta para o ambiente
	sndAround                   [4]float64         // fração de água, lava, coisa alta e lama em volta
	sndNext                     map[string]float64 // quando toca o próximo som avulso de cada tipo
	sndFrame                    float64            // hora do último quadro de som
	heading                     float64            // rumo mostrado na bússola (graus, 0 = norte)
	hvx, hvy                    float64            // média do andar (rumo suave)
	compassLbl                  string             // nome do ponto à frente, que fica um pouco
	compassLblAt                float64
	headWant                    float64 // rumo para onde o personagem andou por último
	headLast                    *[2]int // posição da última leitura do rumo
	joining                     bool
	nameInput                   string
	cls                         int
	joinMsg                     string
	mini                        *ebiten.Image
	miniPix                     []byte
	miniDirty                   bool
	fog                         []byte
	fogImg                      *ebiten.Image
	fogPix                      []byte
	fogImgDirty                 bool
	fogDirty                    bool
	fogAt                       float64
	fogLast                     int
	charName                    string
	styles                      map[int]*bodyStyle
	shotPath                    string  // -print: salva a tela e fecha
	shotAfter                   float64 // ms dentro do jogo antes do print
	inGameAt                    float64
	quit                        bool
	hudImg                      *ebiten.Image

	floats      []floatText
	popSeen     map[int]bool
	deathAt     float64
	chapterAt   float64
	lastChapter int
	shopIdx     int
	shopView    string  // "menu", "talk", "buy" ou "sell" (loja.go)
	shopSel     int     // linha escolhida na lista
	shopSay     string  // fala atual do mercador
	shopSayAt   float64 // quando ela começou (surge letra a letra)
	shopTalk    int     // quantas vezes já conversou (troca a fala)
	shopWant    string  // -painel loja-compra: a vista a abrir
	opts        options
	barDone     bool
	chars       []charInfo
	charsMsg    string
	charsCh     chan charsResult
	delConfirm  string
	joinTab     string // aba da tela de entrada: "load" ou "new"
	pwInput     string
	focusPw     bool
	use3D       bool          // humanoides em 3D (modo 3D)
	mode        int           // 0 = 2D, 1 = Mescla, 2 = 3D
	depth       depthBuf      // relevo do cenário (depth.go)
	bases       bases         // linha de base das estruturas altas (life.go)
	meFeet      float64       // linha dos pés do jogador
	adj         [][]int       // vizinhos de cada nó da teia (webtree.go)
	webHover    int           // nó da teia sob o mouse (-1 nenhum)
	barHover    int           // golpe da barra sob o mouse
	webBG       *ebiten.Image // fundo da teia já desenhado
	webBGKey    string
	shadeF      []float32 // sombra projetada de cada célula da tela (1 = sol pleno; frame.go e depth.go)
	picking     bool      // janela de escolher pasta aberta
	folderMsg   string
	releaseAsk  int // montaria + 1 esperando confirmar a soltura
	releaseAt   float64
	sealMode    int // selo cujas runas estão sendo tocadas (-1 nenhum)
	sealBuf     string
	talkNPC     int // conversa aberta com este NPC (-1 nenhuma)
	talkLine    int
	introPage   int // abertura da história: página atual (-1 fechada)
	introAt     float64
	dojoSel     int // terreiro do mestre (dojo.go)
	dojoMode    int
	dojoAt      float64
	dojoBuf     string
	metMaster   bool
	techHover   int // técnica sob o mouse na teia (-1 nenhuma)
	gateSel     int // portão do painel aberto (gates.go)
	readSel     int // leitura aberta (leituras.go)
	gateBuf     string
	introDone   bool // a abertura já foi decidida nesta sessão
	introName   string
	bagPage     int                // página da mochila na ficha
	chatLog     []logLine          // registro de falas e avisos (tecla L)
	logSeen     int                // quantas linhas do registro já foram vistas
	showLog     bool               // registro aberto
	showHelp    bool               // dicas de controle abertas (tecla H)
	showcase    bool               // vitrine: as 7 classes em 3D e em 2D, lado a lado
	yaws        map[string]float64 // para onde cada modelo 3D está virado
	ops         []func()           // fila de desenho do quadro, em ordem
	jobs        []*r3job           // modelos 3D a desenhar em paralelo
	fov         fov
	seeX, seeY  float64 // centro da copa translúcida (o jogador)
	decals      []decal
	lastStep    map[string][2]int
	hitMark     map[string]bool
	bubbles     map[string]bubble
	talkers     map[string][2]float64
	shotN       int     // -print-n: quantos quadros salvar
	shotGap     float64 // intervalo entre eles, em ms
	shotDone    int
	shotLast    float64
}

type charsResult struct {
	list []charInfo
	err  error
}

// loadChars busca no servidor os personagens deste PC.
func (g *Game) loadChars() {
	go func() {
		l, err := fetchChars(g.server)
		g.charsCh <- charsResult{l, err}
	}()
}

func newGame(server string) *Game {
	g := &Game{server: server, start: time.Now(), mapCh: make(chan *gameMap, 1), deepCh: make(chan *deepPart, 1), faces: map[string]int{}, fxSeen: map[string]bool{}, sndNext: map[string]float64{},
		walkPhase: map[string]float64{}, rippleAt: map[string]float64{}, styles: map[int]*bodyStyle{}, g: newGPU(), joining: true,
		joinMsg: "Escolha um personagem ou crie um novo.", fogLast: -1, popSeen: map[int]bool{},
		charsCh: make(chan charsResult, 2), opts: loadOptions(), yaws: map[string]float64{},
		lastStep: map[string][2]int{}, hitMark: map[string]bool{}, bubbles: map[string]bubble{}, sealMode: -1, talkNPC: -1, introPage: -1, techHover: -1}
	if b, err := os.ReadFile(filepath.Join(configDir(), "ultimo")); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(b)), "\n", 2)
		g.nameInput = parts[0]
		if len(parts) > 1 {
			g.cls, _ = strconv.Atoi(parts[1])
		}
	}
	if b, err := os.ReadFile(filepath.Join(configDir(), "mapa_recolhido")); err == nil {
		g.mapShut = string(b) == "1"
	}
	g.loadMap()
	g.loadChars()
	return g
}

func (g *Game) now() float64 { return float64(time.Since(g.start).Microseconds()) / 1000 }

func (g *Game) loadMap() {
	if g.loading {
		return
	}
	g.loading = true
	go func() {
		m, err := fetchMap(g.server)
		if err != nil {
			g.mapErr = "Não consegui baixar o mapa de " + g.server + ": " + err.Error()
			m = nil
		}
		g.mapCh <- m
	}()
}

func (g *Game) join() {
	name := strings.TrimSpace(g.nameInput)
	if name == "" {
		name = "anon"
	}
	g.charName = name
	os.WriteFile(filepath.Join(configDir(), "ultimo"), []byte(name+"\n"+strconv.Itoa(g.cls)), 0o644)
	if g.m == nil {
		g.joinMsg = "Baixando o mapa…"
		return // entra quando o mapa chegar
	}
	g.joinMsg = "Conectando…"
	c, err := dial(g.server, name, g.pwInput, g.cls, g.m.ver)
	if err != nil {
		g.joinMsg = "Não consegui conectar: " + err.Error()
		return
	}
	g.c = c
}

func (g *Game) send(m proto.In) {
	if g.c != nil {
		g.c.send(m)
	}
}

// Update roda 60 vezes por segundo: rede, teclado e mouse.
func (g *Game) Update() error {
	if g.quit {
		return errQuit
	}
	if !g.barDone {
		g.barDone = darkTitleBar(windowTitle) // acha a janela pelo título: tem de ser o mesmo
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		g.setFullscreen(!g.opts.Fullscreen)
	}
	select {
	case r := <-g.charsCh:
		g.chars, g.charsMsg = r.list, ""
		if r.err != nil {
			g.charsMsg = "Não consegui listar os personagens: " + r.err.Error()
		}
	default:
	}
	g.wantDeep()
	if now := g.now(); g.snd != nil {
		g.snd.muted = !ebiten.IsFocused() && g.opts.MuteUnfocused
		g.updateSound(math.Min(0.1, (now-g.sndFrame)/1000))
		g.sndFrame = now
	}
	select {
	case m := <-g.mapCh:
		g.loading = false
		if m != nil {
			g.m = m
			g.buildMinimap()
			if g.joining && g.charName != "" && g.c == nil {
				g.join()
			}
		} else if g.joining {
			g.joinMsg = g.mapErr
		}
	default:
	}
	if g.c != nil {
		in, sts, err, closed := g.c.take()
		if in != nil {
			g.saveFog()
			g.info, g.fog, g.joining, g.adj = in, nil, false, nil
		}
		for _, s := range sts {
			g.onState(s)
		}
		if closed {
			g.c.close()
			g.c, g.info, g.cur, g.prev, g.joining = nil, nil, nil, nil, true
			g.pauseMenu, g.pauseSent, g.savesSeen, g.savedAt = false, false, 0, 0
			g.lastChapter, g.chapterAt, g.deathAt, g.floats = 0, 0, 0, nil
			g.loadChars()
			if err != "" {
				g.joinMsg = err
			}
		}
	}
	if g.joining {
		g.updateJoin()
		return nil
	}
	g.syncPause()
	g.updateInput()
	return nil
}

// onState recebe um estado do servidor: aplica as mudanças do terreno,
// descobre para que lado cada um olha e solta os efeitos novos.
func (g *Game) onState(s *proto.State) {
	if s.Reload {
		g.loadMap()
	} else if len(s.Ed) > 0 && g.m != nil {
		g.editSounds(s.Ed)
		g.applyEdits(s.Ed)
	}
	w := wrapState(s)
	g.soundEvents(s, w)
	if g.cur != nil {
		for _, p := range s.Players {
			if o := g.cur.pmap[p.ID]; o != nil && o.X != p.X {
				g.faces[fmt.Sprint("p", p.ID)] = sign(float64(p.X - o.X))
			}
		}
		for _, m := range s.Mobs {
			if o := g.cur.mmap[m.ID]; o != nil && math.Abs(o.X-m.X) > 0.05 {
				g.faces[fmt.Sprint("m", m.ID)] = sign(m.X - o.X)
			}
		}
		for _, t := range s.Steeds {
			if o := g.cur.smap[t.ID]; o != nil && math.Abs(o.X-t.X) > 0.05 {
				g.faces[fmt.Sprint("s", t.ID)] = sign(t.X - o.X)
			}
		}
	}
	seen := map[string]bool{}
	for _, e := range s.Fx {
		key := fxKey(e)
		seen[key] = true
		if !g.fxSeen[key] {
			g.fxStart(e)
		}
	}
	g.fxSeen = seen
	if g.info != nil {
		g.takePops(s)
		g.takeChat(s)
	}
	if g.cur != nil && (s.XP > g.cur.XP || s.XPNeed != g.cur.XPNeed) {
		g.xpAt = g.now()
	}
	g.prev, g.cur, g.tCur = g.cur, w, g.now()
}

func fxKey(e proto.Fx) string { return fmt.Sprintf("%d:%v:%v:%v", e.Kind, e.X, e.Y, e.R) }

// applyEdits grava as células mudadas e marca o minimapa para refazer.
func (g *Game) applyEdits(ed [][4]int) {
	m := g.m
	for _, e := range ed {
		i := e[0]
		if i < 0 || i >= m.w*m.h {
			continue
		}
		m.glyph[i] = byte(e[1])
		m.color[i*3], m.color[i*3+1], m.color[i*3+2] = byte(e[2]>>16), byte(e[2]>>8), byte(e[2])
		m.flags[i] = byte(e[3])
		mx, my := i%m.w/mm, i/m.w/mm
		g.miniPixel(mx, my, g.miniPix[(my*(m.w/mm)+mx)*4:])
		g.miniDirty = true
	}
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	g.ui = ebiten.Monitor().DeviceScaleFactor()
	return int(float64(outsideW) * g.ui), int(float64(outsideH) * g.ui)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.W, g.H = screen.Bounds().Dx(), screen.Bounds().Dy()
	if g.csTarget == 0 {
		// Começa de perto e abre o plano, como a câmera do vídeo de referência.
		// Câmera de RPG isométrico (Diablo 2, Age of Empires): bem aberta, o
		// personagem ocupa por volta de um décimo da altura da tela.
		g.csTarget = clampi(g.H/210, 3, 14) // personagens menores: aproxima um pouco
		g.cs = min(26, g.csTarget*3)
	}
	g.uiRects = g.uiRects[:0]
	if g.joining || g.info == nil || g.cur == nil || g.m == nil {
		screen.Fill(rgbaHex(0x09090f))
		g.drawJoin(screen)
		// -print sem -nome fotografa a tela de entrada.
		if g.shotPath != "" && g.charName == "" && g.now() > g.shotAfter && !g.quit {
			savePNG(screen, g.shotPath)
			g.quit = true
		}
		return
	}
	g.frame(screen)
	if g.shotPath != "" {
		if g.inGameAt == 0 {
			g.inGameAt = g.now()
		} else if g.now()-g.inGameAt > g.shotAfter && !g.quit && g.now()-g.shotLast >= g.shotGap {
			path := g.shotPath
			if g.shotN > 1 {
				path = strings.TrimSuffix(path, ".png") + fmt.Sprintf("_%03d.png", g.shotDone)
			}
			savePNG(screen, path)
			g.shotDone++
			g.shotLast = g.now()
			g.quit = g.shotDone >= max(1, g.shotN)
		}
	}
}

// ---------- entrada ----------

var dirKeys = [4][]ebiten.Key{
	{ebiten.KeyArrowUp, ebiten.KeyW}, {ebiten.KeyArrowDown, ebiten.KeyS},
	{ebiten.KeyArrowLeft, ebiten.KeyA}, {ebiten.KeyArrowRight, ebiten.KeyD},
}

func pressed(keys []ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func (g *Game) sendDir() {
	dx, dy := 0, 0
	if !g.chatting && g.openPanel != "shop" {
		up, down := dirKeys[0], dirKeys[1]
		if g.showLog { // as setas ↑/↓ estão rolando o registro; W e S seguem andando
			up, down = up[1:], down[1:]
		}
		if pressed(up) {
			dy--
		}
		if pressed(down) {
			dy++
		}
		if pressed(dirKeys[2]) {
			dx--
		}
		if pressed(dirKeys[3]) {
			dx++
		}
	}
	run := !g.chatting && ebiten.IsKeyPressed(ebiten.KeyShift)
	if k := fmt.Sprint(dx, dy, run); k != g.lastDir {
		g.lastDir = k
		g.send(proto.In{T: "in", DX: dx, DY: dy, Run: run})
	}
}

// aim é o ângulo da mira, enquanto o mouse estiver em uso; parado há um tempo,
// os golpes do teclado saem na direção em que o personagem anda.
func (g *Game) aim(force bool) *float64 {
	if g.meScreen == nil || !force && g.now()-g.mouseAt > 2500 {
		return nil
	}
	a := math.Atan2(g.mouseY*g.ui-g.meScreen[1], g.mouseX*g.ui-g.meScreen[0])
	return &a
}

func (g *Game) use(slot int, force bool) {
	g.send(proto.In{T: "use", Slot: slot, Dir: g.aim(force)})
}

func (g *Game) useItem(it int) { g.send(proto.In{T: "item", Item: it}) }

func (g *Game) togglePanel(p string) {
	if g.openPanel == p {
		g.openPanel = ""
	} else {
		g.openPanel = p
	}
}

func (g *Game) updateInput() {
	mx, my := ebiten.CursorPosition()
	if mx != g.lastMX || my != g.lastMY {
		g.lastMX, g.lastMY = mx, my
		g.mouseX, g.mouseY, g.mouseAt = float64(mx)/g.ui, float64(my)/g.ui, g.now()
	}
	if _, wy := ebiten.Wheel(); g.openPanel == "diario" {
		g.journalKeys(wy) // o diário rola com a roda e as setas
	} else if wy != 0 && g.showLog && !g.chatting {
		g.logScroll += 3 * sign(wy) // com o registro aberto, a roda rola as mensagens
	} else if wy != 0 && g.shotPath == "" { // nos prints de teste a roda não mexe no zoom
		g.csTarget = clampi(g.csTarget+sign(wy), 3, 28)
	}
	if g.showLog && !g.chatting { // registro aberto: ↑/↓ e PageUp/PageDown voltam pelas mensagens
		if repeatKey(ebiten.KeyArrowUp) {
			g.logScroll++
		}
		if repeatKey(ebiten.KeyArrowDown) {
			g.logScroll--
		}
		if repeatKey(ebiten.KeyPageUp) {
			g.logScroll += 12
		}
		if repeatKey(ebiten.KeyPageDown) {
			g.logScroll -= 12
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
			g.logScroll = 0
		}
	}

	if g.chatting {
		g.chatBuf = ebiten.AppendInputChars(g.chatBuf)
		if len(g.chatBuf) > 120 {
			g.chatBuf = g.chatBuf[:120]
		}
		if repeatKey(ebiten.KeyBackspace) && len(g.chatBuf) > 0 {
			g.chatBuf = g.chatBuf[:len(g.chatBuf)-1]
		}
		// ↑/↓ voltam pelo que já foi dito, como num terminal.
		if repeatKey(ebiten.KeyArrowUp) && g.chatPos > 0 {
			g.chatPos--
			g.chatBuf = []rune(g.chatSent[g.chatPos])
		}
		if repeatKey(ebiten.KeyArrowDown) && g.chatPos < len(g.chatSent) {
			g.chatPos++
			g.chatBuf = nil
			if g.chatPos < len(g.chatSent) {
				g.chatBuf = []rune(g.chatSent[g.chatPos])
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
			if s := strings.TrimSpace(string(g.chatBuf)); s != "" {
				g.send(proto.In{T: "say", Text: s})
				if n := len(g.chatSent); n == 0 || g.chatSent[n-1] != s {
					g.chatSent = append(g.chatSent, s)
				}
				if len(g.chatSent) > 50 {
					g.chatSent = g.chatSent[len(g.chatSent)-50:]
				}
			}
			g.chatting, g.chatBuf = false, nil
		} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.chatting, g.chatBuf = false, nil
		}
		g.sendDir()
		return
	}

	if g.updatePauseMenu() {
		return
	}
	if g.cur != nil && g.updateTame() { // doma: as setas são dela
		g.send(proto.In{T: "in"})
		g.lastDir = ""
		return
	}
	if g.updateIntro() || g.updateTalk() || g.updateDojo() || g.updateGate() || g.updateRead() { // abertura da história ou conversa: tomam as teclas
		g.send(proto.In{T: "in"})
		g.lastDir = ""
		return
	}
	if g.updateSeal() { // tocando as runas de uma porta selada
		g.send(proto.In{T: "in"})
		g.lastDir = ""
		return
	}
	g.sendDir()
	// Pesca: com a linha na água, G, E ou Espaço puxam.
	if g.cur != nil && g.cur.Fish != nil && (justKey(ebiten.KeyG) || justKey(ebiten.KeyE) || justKey(ebiten.KeySpace)) {
		g.send(proto.In{T: "fish"})
		return
	}
	// Com a loja aberta, as teclas são dela (loja.go); o mouse segue abaixo.
	if !g.updateShopKeys() && !g.updateBonfireKeys() {
		if ebiten.IsKeyPressed(ebiten.KeySpace) && g.now()-g.spaceAt > 100 {
			g.spaceAt = g.now()
			g.use(0, false)
		}
		just := inpututil.IsKeyJustPressed
		switch {
		case just(ebiten.KeyF):
			g.useBar(0, false)
		case just(ebiten.KeyR):
			g.useBar(1, false)
		case just(ebiten.KeyI), just(ebiten.KeyC):
			g.togglePanel("sheet")
		case just(ebiten.KeyK):
			g.togglePanel("tree")
		case just(ebiten.KeyJ):
			g.togglePanel("ach")
		case just(ebiten.KeyV):
			if me := g.cur.pmap[g.info.ID]; me != nil && g.nearTravel(me) >= 0 {
				g.togglePanel("travel")
			}
		case just(ebiten.KeyL):
			g.showLog, g.logScroll = !g.showLog, 0
		case just(ebiten.KeyH):
			g.showHelp = !g.showHelp
		case just(ebiten.KeyM):
			g.bigMap = !g.bigMap
		case just(ebiten.KeyEscape):
			if g.openPanel == "" && !g.bigMap && !g.showHelp && !g.showLog {
				g.pauseMenu = true // nada aberto: Esc pausa
			}
			g.openPanel, g.bigMap, g.showHelp = "", false, false
		case just(ebiten.KeyP):
			g.pauseMenu = true
		case just(ebiten.KeyF5):
			g.saveNow()
		case just(ebiten.KeyB):
			g.cycleMode()
		case just(ebiten.KeyG):
			g.send(proto.In{T: "fish"}) // lançar a linha (pesca.go)
		case just(ebiten.KeyN):
			g.togglePanel("diario")
		case just(ebiten.KeyZ):
			g.send(proto.In{T: "light"}) // acende/apaga a lanterna ou tocha (espaço de luz)
		case just(ebiten.KeyO):
			g.togglePanel("opts")
		case just(ebiten.KeyQ):
			g.useItem(0)
		case just(ebiten.KeyE) && g.cur != nil && g.cur.pmap[g.info.ID] != nil && g.nearGate(g.cur.pmap[g.info.ID]) >= 0:
			g.openGate(g.nearGate(g.cur.pmap[g.info.ID]))
		case just(ebiten.KeyE) && g.cur != nil && g.cur.pmap[g.info.ID] != nil && g.nearNPC(g.cur.pmap[g.info.ID]) >= 0:
			g.openTalk(g.nearNPC(g.cur.pmap[g.info.ID]))
		case just(ebiten.KeyE) && g.cur != nil && g.cur.pmap[g.info.ID] != nil && strings.HasPrefix(g.interactHint(g.cur.pmap[g.info.ID]), "[E] ler"):
			g.openRead(g.nearRead(g.cur.pmap[g.info.ID]))
		case just(ebiten.KeyE) && g.cur != nil && g.sealHere() >= 0:
			g.sealMode, g.sealBuf = g.sealHere(), ""
		case just(ebiten.KeyE):
			if me := g.cur.pmap[g.info.ID]; me != nil {
				if l := g.nearLot(me); l >= 0 && g.openPanel != "casa" {
					g.openPanel = "casa" // casa à venda ou o baú da sua casa (casa.go)
					break
				} else if g.openPanel == "casa" {
					g.openPanel = ""
					break
				}
				if s := g.nearShop(me); s >= 0 && g.openPanel != "shop" {
					g.openShop(s) // conversa com o mercador (loja.go)
					break
				} else if g.openPanel == "shop" {
					g.openPanel = ""
					break
				}
			}
			g.send(proto.In{T: "act"})
			if me := g.cur.pmap[g.info.ID]; me != nil && g.nearBonfire(me) && g.openPanel == "" {
				g.openPanel = "fogo" // descansou: painel para esperar a hora ou o clima
			}
		case just(ebiten.KeyX):
			g.send(proto.In{T: "ride"})
		case just(ebiten.KeyEnter), just(ebiten.KeyT):
			g.chatting, g.chatBuf, g.chatPos = true, nil, len(g.chatSent)
			ebiten.AppendInputChars(nil) // descarta o T que abriu o chat
		}
		// Atalhos 1 a 9 (os abertos): usam o golpe da barra. Com a teia aberta e o mouse sobre
		// um golpe aprendido, o número põe esse golpe no atalho.
		for d := 0; d < g.barN(); d++ {
			if !just(ebiten.KeyDigit1+ebiten.Key(d)) || g.cur == nil {
				continue
			}
			if g.openPanel == "tree" && g.techHover >= 0 {
				g.send(proto.In{T: "bar", I: d, Slot: g.techHover})
				continue
			}
			if g.openPanel == "tree" && g.webHover > 0 && g.webHover < len(g.info.Web) {
				if n := g.info.Web[g.webHover]; n.Skill >= 0 && g.rank(g.webHover) > 0 {
					g.send(proto.In{T: "bar", I: d, Slot: n.Skill})
				}
				continue
			}
			g.useBar(d, false)
		}
	}

	// Botão esquerdo: golpe básico; direito e do meio: atalhos 1 e 2 (-1 e -2).
	for b, slot := range map[ebiten.MouseButton]int{ebiten.MouseButtonLeft: 0, ebiten.MouseButtonRight: -1, ebiten.MouseButtonMiddle: -2} {
		if !inpututil.IsMouseButtonJustPressed(b) {
			continue
		}
		g.mouseAt = g.now()
		// Sobre uma janela, o clique é dela (nunca sai golpe por trás). O
		// botão direito só faz algo onde há ação alternativa (jogar fora).
		hit := false
		for _, r := range g.uiRects {
			if g.mouseX >= r.x && g.mouseX < r.x+r.w && g.mouseY >= r.y && g.mouseY < r.y+r.h {
				switch {
				case b == ebiten.MouseButtonRight:
					if r.alt != nil {
						g.send(*r.alt)
					}
				case b != ebiten.MouseButtonLeft:
				case r.fn != nil:
					g.sfx("click", 0.4)
					r.fn()
				case r.act != nil:
					g.send(*r.act)
				}
				hit = true
				break
			}
		}
		if !hit {
			if slot < 0 {
				g.useBar(-slot-1, true)
			} else {
				g.use(slot, true)
			}
		}
	}
}

// repeatKey vale no aperto e, segurando, de tempos em tempos.
func repeatKey(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || d > 25 && d%3 == 0
}

func (g *Game) updateJoin() {
	// Aba dos personagens salvos: 1 a 9 carregam; o teclado não escreve nos
	// campos (eles estão na outra aba).
	if g.joinTab == "load" {
		for d := 0; d < min(9, len(g.chars)); d++ {
			if inpututil.IsKeyJustPressed(ebiten.KeyDigit1+ebiten.Key(d)) && g.c == nil {
				ch := g.chars[d]
				g.nameInput, g.pwInput, g.cls = ch.Name, "", ch.Class
				g.join()
			}
		}
		ebiten.AppendInputChars(nil)
		mx, my := ebiten.CursorPosition()
		g.mouseX, g.mouseY = float64(mx)/g.ui, float64(my)/g.ui
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			for _, r := range g.uiRects {
				if g.mouseX >= r.x && g.mouseX < r.x+r.w && g.mouseY >= r.y && g.mouseY < r.y+r.h && r.fn != nil {
					r.fn()
					break
				}
			}
		}
		return
	}
	// O texto digitado vai para o campo em foco (nome ou senha); Tab alterna.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.focusPw = !g.focusPw
	}
	field, limit := &g.nameInput, 12
	if g.focusPw {
		field, limit = &g.pwInput, 32
	}
	r := ebiten.AppendInputChars([]rune(*field))
	if len(r) > limit {
		r = r[:limit]
	}
	if repeatKey(ebiten.KeyBackspace) && len(r) > 0 {
		r = r[:len(r)-1]
	}
	*field = string(r)
	for d := 0; d < len(classInfo); d++ {
		if inpututil.IsKeyJustPressed(ebiten.KeyF1 + ebiten.Key(d)) {
			g.cls = d
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.cls = (g.cls + 7) % 8
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.cls = (g.cls + 1) % 8
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.cls = (g.cls + 6) % 8
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.cls = (g.cls + 2) % 8
	}
	mx, my := ebiten.CursorPosition()
	g.mouseX, g.mouseY = float64(mx)/g.ui, float64(my)/g.ui
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for _, r := range g.uiRects {
			if g.mouseX >= r.x && g.mouseX < r.x+r.w && g.mouseY >= r.y && g.mouseY < r.y+r.h && r.fn != nil {
				r.fn()
			}
		}
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)) && g.c == nil {
		g.join()
	}
}

// useBar usa o golpe do atalho i da barra, se houver um.
func (g *Game) useBar(i int, force bool) {
	if g.cur != nil && i >= 0 && i < len(g.cur.Bar) && g.cur.Bar[i] >= 0 {
		g.use(g.cur.Bar[i], force)
	}
}
