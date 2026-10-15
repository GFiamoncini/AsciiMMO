package main

import (
	"bytes"
	"context"
	"github.com/gorilla/websocket"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type client struct {
	in  io.Writer
	out *syncBuf
}

func dial(t *testing.T, addr, user string) *client {
	t.Helper()
	conn, err := gossh.Dial("tcp", addr, &gossh.ClientConfig{
		User:            user,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	sess, err := conn.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	c := &client{out: &syncBuf{}}
	sess.Stdout = c.out
	if c.in, err = sess.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	if err := sess.RequestPty("xterm-256color", 40, 120, gossh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	if err := sess.Shell(); err != nil {
		t.Fatal(err)
	}
	return c
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("tempo esgotado esperando: %s", what)
}

func TestDoisJogadores(t *testing.T) {
	world := NewWorld(1)
	srv, err := newServer("127.0.0.1:0", filepath.Join(t.TempDir(), "key"), world)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	ana := dial(t, ln.Addr().String(), "ana")
	waitFor(t, "tela da ana", func() bool { return strings.Contains(ana.out.String(), "ana") })

	bob := dial(t, ln.Addr().String(), "bob")
	waitFor(t, "ana ver o bob", func() bool { return strings.Contains(ana.out.String(), "bob") })

	// Chat do bob chega na tela da ana.
	for _, k := range []string{"\r", "oi pessoal", "\r"} {
		io.WriteString(bob.in, k)
		time.Sleep(100 * time.Millisecond)
	}
	waitFor(t, "chat do bob na tela da ana", func() bool { return strings.Contains(ana.out.String(), "oi pessoal") })

	// Alguma das quatro direções precisa mover o bob.
	pos := func() (int, int) {
		for _, p := range world.Snapshot(0).Players {
			if p.Name == "bob" {
				return p.X, p.Y
			}
		}
		t.Fatal("bob não está no mundo")
		return 0, 0
	}
	x0, y0 := pos()
	moved := false
	for _, k := range []string{"d", "a", "w", "s"} {
		x, y := pos()
		io.WriteString(bob.in, k)
		time.Sleep(200 * time.Millisecond)
		if nx, ny := pos(); nx != x || ny != y {
			moved = true
			break
		}
	}
	if !moved {
		t.Fatalf("bob não saiu de (%d,%d)", x0, y0)
	}

	// Sair remove o jogador do mundo.
	io.WriteString(bob.in, "q")
	waitFor(t, "bob sair do mundo", func() bool { return len(world.Snapshot(0).Players) == 1 })
}

func TestSanitize(t *testing.T) {
	if got := sanitizeName("\x1b[31mzé!!"); got != "31mzé" {
		t.Errorf("sanitizeName = %q", got)
	}
	if got := sanitizeName("\x1b\x07"); got != "anon" {
		t.Errorf("sanitizeName vazio = %q", got)
	}
	if got := sanitizeText("oi\x1b[2J\x07 mundo"); strings.ContainsAny(got, "\x1b\x07") {
		t.Errorf("sanitizeText deixou controle passar: %q", got)
	}
}

func TestWeb(t *testing.T) {
	world := NewWorld(1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go world.Run(ctx)
	srv := httptest.NewServer(newWebServer("", world).Handler)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/map")
	if err != nil {
		t.Fatal(err)
	}
	blob, _ := io.ReadAll(res.Body) // o cliente HTTP já desfaz o gzip
	res.Body.Close()
	if want := 8 + 5*worldW*deepY0; len(blob) != want || string(blob[:4]) != "AMAP" {
		t.Fatalf("mapa com %d bytes, esperava %d", len(blob), want)
	}
	if h := res.Header.Get("X-Map-Full-H"); h != strconv.Itoa(worldH) {
		t.Fatalf("altura do mundo todo no cabeçalho: %q", h)
	}
	res, err = http.Get(srv.URL + "/map?parte=fundo") // as Profundezas, à parte
	if err != nil {
		t.Fatal(err)
	}
	deep, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if want := 8 + 5*worldW*deepH; len(deep) != want {
		t.Fatalf("Profundezas com %d bytes, esperava %d", len(deep), want)
	}

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?name=bia", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	var init wsInit
	if err := conn.ReadJSON(&init); err != nil || !init.Init || init.W != worldW {
		t.Fatalf("init = %+v, err = %v", init, err)
	}
	x := func() int {
		var st wsState
		if err := conn.ReadJSON(&st); err != nil {
			t.Fatal(err)
		}
		for _, p := range st.Players {
			if p.ID == init.ID {
				return p.X
			}
		}
		t.Fatal("jogador não veio no estado")
		return 0
	}

	// Segurar uma direção faz o jogador andar sozinho a cada tick.
	x0 := x()
	moved := false
	for _, dx := range []int{1, -1} {
		conn.WriteJSON(wsIn{T: "in", DX: dx})
		for i := 0; i < 10 && !moved; i++ {
			moved = x() != x0
		}
	}
	if !moved {
		t.Fatalf("jogador não saiu de x=%d", x0)
	}

	conn.Close()
	waitFor(t, "bia sair do mundo", func() bool { return len(world.Snapshot(0).Players) == 0 })
}
