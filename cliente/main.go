// Cliente nativo do AsciiMMO: desenha o mundo direto na placa de vídeo, sem
// navegador. Conversa com o servidor pelo mesmo protocolo do cliente web
// (mapa em /map, estado por WebSocket em /ws), então o servidor não muda.
package main

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	server := flag.String("servidor", "http://localhost:8080", "endereço do servidor (o mesmo do cliente web)")
	name := flag.String("nome", "", "entra direto com este nome, sem a tela inicial")
	cls := flag.Int("classe", -1, "classe ao entrar direto: 0 Guerreiro ... 6 Necromante, 7 Depravado")
	zoom := flag.Int("zoom", 0, "lado da célula em pixels (0 escolhe sozinho)")
	shot := flag.String("print", "", "salva um print da tela neste arquivo PNG e fecha (para testes)")
	shotAfter := flag.Duration("print-apos", 5*time.Second, "quanto esperar dentro do jogo antes do print")
	use3d := flag.Bool("3d", false, "personagens em 3D")
	mix := flag.Bool("mescla", false, "personagens no modo Mescla (2D com luz 3D)")
	flat := flag.Bool("2d", false, "personagens em 2D chapado")
	noDepth := flag.Bool("sem-profundidade", false, "desliga o volume do cenário (para comparar)")
	panel := flag.String("painel", "", "abre um painel ao entrar: tree, sheet, ach, opts (para prints)")
	vitrine := flag.Bool("vitrine", false, "mostra as 7 classes em 2D, Mescla e 3D, lado a lado (teste)")
	shotN := flag.Int("print-n", 1, "quantos quadros salvar com -print (para animação)")
	shotGap := flag.Duration("print-intervalo", 80*time.Millisecond, "intervalo entre os quadros do -print-n")
	prof := flag.String("perfil", "", "grava um perfil de CPU neste arquivo (para achar lentidão)")
	noSound := flag.Bool("sem-som", false, "desliga todo o som (música, ambiente e efeitos)")
	monitor := flag.Int("monitor", 0, "abre a janela neste monitor (1 = o principal; 0 deixa o sistema escolher)")
	iconOut := flag.String("gerar-icone", "", "grava o ícone (256×256) neste PNG e fecha (para o build)")
	flag.Parse()
	if *iconOut != "" {
		f, err := os.Create(*iconOut)
		if err != nil {
			panic(err)
		}
		png.Encode(f, windowIcon(256))
		f.Close()
		return
	}
	if *prof != "" {
		if f, err := os.Create(*prof); err == nil {
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
	}

	g := newGame(strings.TrimRight(*server, "/"))
	g.shotPath, g.shotAfter = *shot, shotAfter.Seconds()*1000
	if !*noSound && *shot == "" && *iconOut == "" { // prints de teste ficam mudos
		g.snd = newSoundSys()
	}
	g.showcase = *vitrine
	g.openPanel = *panel
	if *panel == "read" { // a leitura mais perto (depois de entrar)
		g.readSel = -2
	}
	if *panel == "gate" { // painel do portão mais perto (depois de entrar)
		g.gateSel = -2
	}
	if *panel == "talk" { // conversa com o Eremita aberta, numa fala do meio
		g.openPanel, g.talkNPC, g.talkLine = "", 0, 3
	}
	if strings.HasPrefix(*panel, "loja") { // a loja mais perto (depois de entrar): "loja" ou "loja-compra"
		g.openPanel, g.shopIdx, g.shopWant = "", -2, strings.TrimPrefix(strings.TrimPrefix(*panel, "loja"), "-")
	}
	if strings.HasPrefix(*panel, "abertura") { // abertura0..2: uma tela da abertura, já com texto
		g.openPanel, g.introPage, g.introAt, g.introDone = "", int((*panel)[len(*panel)-1]-'0'), 4, true
	}
	if strings.HasPrefix(*panel, "diario-") { // diario-quests, -story, -beasts, -fish
		g.openPanel, g.journalTab = "diario", strings.TrimPrefix(*panel, "diario-")
	}
	if *panel == "pausa" {
		g.openPanel, g.pauseMenu = "", true
	}
	if *panel == "help" {
		g.openPanel, g.showHelp = "", true
	}
	g.mode = clampi(g.opts.Mode, 0, 2)
	switch {
	case *use3d:
		g.mode = mode3D
	case *mix:
		g.mode = modeMix
	case *flat:
		g.mode = mode2D
	}
	g.use3D = g.mode == mode3D
	g.opts.NoDepth = g.opts.NoDepth || *noDepth
	g.shotN, g.shotGap = *shotN, shotGap.Seconds()*1000
	if *zoom > 0 {
		g.csTarget = clampi(*zoom, 3, 28)
		g.cs = g.csTarget
	}
	if *cls >= 0 {
		g.cls = clampi(*cls, 0, len(classInfo)-1)
	}
	if *name != "" {
		g.nameInput = *name
		g.join()
	}

	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowIcon([]image.Image{windowIcon(16), windowIcon(32), windowIcon(48), windowIcon(64)})
	ebiten.SetFullscreen(g.opts.Fullscreen)
	ebiten.SetWindowSize(1600, 900)
	if *monitor > 0 {
		ms := ebiten.AppendMonitors(nil)
		for i, m := range ms {
			log.Printf("monitor %d: %s", i+1, m.Name())
		}
		if *monitor <= len(ms) {
			ebiten.SetMonitor(ms[*monitor-1])
		}
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(true)
	ebiten.SetScreenClearedEveryFrame(false)
	if err := ebiten.RunGame(g); err != nil && err != errQuit {
		log.Print(err)
	}
}

// windowTitle é o título da janela; a barra escura (darkTitleBar) acha a
// janela por ele.
const windowTitle = "AsciiMMO — Midrath"
