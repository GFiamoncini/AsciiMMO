package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"asciimmo/proto"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

func newServer(addr, keyPath string, world *World) (*ssh.Server, error) {
	handler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		id := world.Join(s.User(), 0) // no terminal todo mundo é guerreiro
		go func() {
			<-s.Context().Done()
			world.Leave(id)
		}()
		pty, _, _ := s.Pty()
		m := newModel(world, id, pty.Window.Width, pty.Window.Height)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
	return wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(keyPath),
		wish.WithMiddleware(
			bubbletea.Middleware(handler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
}

func main() {
	addr := flag.String("addr", ":2222", "endereço de escuta do servidor SSH")
	keyPath := flag.String("key", ".ssh/id_ed25519", "chave do host (criada se não existir)")
	httpAddr := flag.String("http", ":8080", "endereço do cliente web (vazio desliga)")
	seed := flag.Int64("seed", 1, "semente do mapa")
	day := flag.Duration("day", 10*time.Minute, "duração de um dia no jogo")
	saves := flag.String("saves", "", "pasta dos personagens salvos (padrão: <pasta de dados>/save, escolhida no cliente; \"-\" desliga)")
	weather := flag.String("weather", "auto", "clima fixo: auto, clear, rain ou storm")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // Ctrl+C ou "servidor.sh parar": grava todos e sai
	defer stop()

	world := NewWorld(*seed)
	world.dayLen = max(10, day.Seconds())
	world.saveDir = *saves
	switch *saves {
	case "":
		world.saveDir = proto.SaveDir()
		os.MkdirAll(world.saveDir, 0o755)
		if st, err := os.Stat("saves"); err == nil && st.IsDir() { // saves de antes, na pasta do jogo
			proto.CopyMissing("saves", world.saveDir)
		}
		log.Printf("Personagens salvos em %s", world.saveDir)
	case "-":
		world.saveDir = ""
	}
	if v, ok := map[string]int{"clear": wxClear, "rain": wxRain, "storm": wxStorm, "fog": wxFog}[*weather]; ok {
		world.wxForced = v
	}
	defer world.SaveAll()
	go world.Run(ctx)

	srv, err := newServer(*addr, *keyPath, world)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Printf("AsciiMMO em %s — entre com: ssh -p <porta> <nome>@<host>", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	var web *http.Server
	if *httpAddr != "" {
		web = newWebServer(*httpAddr, world)
		go func() {
			log.Printf("Cliente web em %s — abra http://localhost:<porta> no navegador", *httpAddr)
			if err := web.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatal(err)
			}
		}()
	}

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if web != nil {
		// Close, e não Shutdown: as conexões WebSocket não fecham sozinhas.
		_ = web.Close()
	}
	_ = srv.Shutdown(shutdown)
}
