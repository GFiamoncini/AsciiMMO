package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"asciimmo/proto"

	"github.com/gorilla/websocket"
)

// gameMap é o terreno: um caractere, uma cor e um conjunto de flags por célula.
type gameMap struct {
	w, h   int
	glyph  []byte
	color  []byte // RGB
	flags  []byte // 1 sólido, 2 por cima, 4 água, 8 lama, 16 veneno
	ver    int
	loaded bool
	topH   int  // linhas que vieram no /map (o resto são as Profundezas)
	deepOK bool // as Profundezas já chegaram (ou o servidor não as tem)
}

// fetchMap baixa o mapa empacotado ("AMAP", largura, altura e os três planos).
func fetchMap(server string) (*gameMap, error) {
	res, err := http.Get(server + "/map")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body) // o cliente HTTP já desfaz o gzip
	if err != nil {
		return nil, err
	}
	if len(b) < 8 || string(b[:4]) != "AMAP" {
		return nil, errors.New("o servidor não mandou um mapa válido")
	}
	w, h := int(binary.LittleEndian.Uint16(b[4:])), int(binary.LittleEndian.Uint16(b[6:]))
	n := w * h
	if len(b) != 8+5*n {
		return nil, fmt.Errorf("mapa com %d bytes, esperava %d", len(b), 8+5*n)
	}
	ver, _ := strconv.Atoi(res.Header.Get("X-Map-Ver"))
	// Servidor com Profundezas: o mundo é mais alto que esta parte. Reserva
	// tudo; a parte de baixo chega quando alguém desce (fetchDeep).
	full, _ := strconv.Atoi(res.Header.Get("X-Map-Full-H"))
	full = max(full, h)
	// Cada camada vai para uma fatia própria, já do tamanho do mundo todo.
	// Antes eram fatias do buffer do download, e o append que reservava as
	// Profundezas escrevia os zeros por cima das cores logo depois das
	// letras: o norte do mapa (cemitério, cripta, castelo, montanhas) ficava
	// preto.
	fn := w * full
	m := &gameMap{w: w, h: full, glyph: make([]byte, fn), color: make([]byte, 3*fn), flags: make([]byte, fn),
		ver: ver, loaded: true, topH: h, deepOK: full == h}
	copy(m.glyph, b[8:8+n])
	copy(m.color, b[8+n:8+4*n])
	copy(m.flags, b[8+4*n:])
	return m, nil
}

// deepPart é a parte de baixo do mapa (as Profundezas), já baixada.
type deepPart struct {
	glyph, color, flags []byte
}

// fetchDeep baixa as Profundezas (/map?parte=fundo).
func fetchDeep(server string, w int) (*deepPart, error) {
	res, err := http.Get(server + "/map?parte=fundo")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if len(b) < 8 || string(b[:4]) != "AMAP" || int(binary.LittleEndian.Uint16(b[4:])) != w {
		return nil, errors.New("o servidor não mandou as Profundezas")
	}
	n := w * int(binary.LittleEndian.Uint16(b[6:]))
	if len(b) != 8+5*n {
		return nil, fmt.Errorf("Profundezas com %d bytes, esperava %d", len(b), 8+5*n)
	}
	return &deepPart{b[8 : 8+n], b[8+n : 8+4*n], b[8+4*n:]}, nil
}

var migrateOnce sync.Once

// configDir é a pasta do cliente dentro da pasta de dados do jogo
// (Documentos/AsciiMMO/cliente por padrão). Na primeira vez, traz o que
// estava na pasta antiga (%AppData%/AsciiMMO), inclusive o segredo dos
// personagens.
func configDir() string {
	d := proto.ClientDir()
	migrateOnce.Do(func() {
		os.MkdirAll(d, 0o755)
		if old, err := os.UserConfigDir(); err == nil {
			proto.CopyMissing(filepath.Join(old, "AsciiMMO"), d)
		}
	})
	os.MkdirAll(d, 0o755)
	return d
}

// token é o segredo que prova ao servidor que o personagem é deste PC.
func token() string {
	p := filepath.Join(configDir(), "token")
	if b, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(b))) == 32 {
		return strings.TrimSpace(string(b))
	}
	raw := make([]byte, 16)
	rand.Read(raw)
	t := hex.EncodeToString(raw)
	os.WriteFile(p, []byte(t), 0o600)
	return t
}

// conn é a conexão com o servidor: uma goroutine lê, outra escreve.
type conn struct {
	ws     *websocket.Conn
	out    chan proto.In
	mu     sync.Mutex
	init   *proto.Init
	states []*proto.State
	err    string
	closed bool
}

func dial(server, name, pw string, cls, mapVer int) (*conn, error) {
	u, err := url.Parse(server)
	if err != nil {
		return nil, err
	}
	u.Scheme = map[string]string{"https": "wss"}[u.Scheme]
	if u.Scheme == "" {
		u.Scheme = "ws"
	}
	u.Path = "/ws"
	u.RawQuery = url.Values{"name": {name}, "cls": {strconv.Itoa(cls)}, "token": {token()}, "mv": {strconv.Itoa(mapVer)}, "pw": {pw}}.Encode()
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	c := &conn{ws: ws, out: make(chan proto.In, 64)}
	go c.read()
	go c.write()
	return c, nil
}

func (c *conn) read() {
	defer func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.out)
	}()
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if c.err == "" {
				c.err = "Conexão perdida. Entre de novo."
			}
			c.mu.Unlock()
			return
		}
		var probe struct {
			Init  bool   `json:"init"`
			Error string `json:"error"`
		}
		json.Unmarshal(data, &probe)
		c.mu.Lock()
		switch {
		case probe.Error != "":
			c.err = probe.Error
		case probe.Init:
			var in proto.Init
			if json.Unmarshal(data, &in) == nil {
				c.init = &in
			}
		default:
			var st proto.State
			if json.Unmarshal(data, &st) == nil {
				// Se o jogo travar, não acumula estado velho: só os últimos importam.
				if len(c.states) > 8 {
					c.states = c.states[len(c.states)-8:]
				}
				c.states = append(c.states, &st)
			}
		}
		c.mu.Unlock()
	}
}

func (c *conn) write() {
	for m := range c.out {
		if c.ws.WriteJSON(m) != nil {
			return
		}
	}
}

// send manda sem travar o jogo; se a fila estiver cheia, a mensagem se perde.
func (c *conn) send(m proto.In) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return
	}
	defer func() { recover() }() // canal fechado no meio do caminho
	select {
	case c.out <- m:
	default:
	}
}

// take devolve o init (uma vez) e os estados que chegaram desde a última chamada.
func (c *conn) take() (*proto.Init, []*proto.State, string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	in, st, err := c.init, c.states, c.err
	c.init, c.states = nil, nil
	return in, st, err, c.closed
}

func (c *conn) close() { c.ws.Close() }

// charInfo é um personagem deste PC, como o servidor lista.
type charInfo struct {
	Name     string `json:"name"`
	Class    int    `json:"cls"`
	Level    int    `json:"lv"`
	Chapter  int    `json:"ch"`
	Password bool   `json:"pw"`
	Online   bool   `json:"on"`
}

func fetchChars(server string) ([]charInfo, error) {
	res, err := http.Get(server + "/chars?token=" + url.QueryEscape(token()))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var list []charInfo
	err = json.NewDecoder(res.Body).Decode(&list)
	return list, err
}

func deleteChar(server, name string) error {
	res, err := http.PostForm(server+"/chars/delete", url.Values{"token": {token()}, "name": {name}})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var out struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&out)
	if out.Error != "" {
		return errors.New(out.Error)
	}
	return nil
}

// wantDeep baixa as Profundezas na primeira vez em que o jogador está nelas
// (ou chega perto da borda) e, quando chegam, copia para o mapa.
func (g *Game) wantDeep() {
	m := g.m
	if m == nil || m.deepOK {
		return
	}
	select {
	case d := <-g.deepCh:
		g.deepBusy = false
		if d == nil {
			return // tenta de novo no próximo quadro em que precisar
		}
		a := m.topH * m.w
		copy(m.glyph[a:], d.glyph)
		copy(m.color[3*a:], d.color)
		copy(m.flags[a:], d.flags)
		m.deepOK = true
		g.buildMinimap()
		return
	default:
	}
	var me *proto.Player
	if g.cur != nil && g.info != nil {
		me = g.cur.pmap[g.info.ID]
	}
	if g.deepBusy || me == nil || me.Y < m.topH-40 {
		return
	}
	g.deepBusy = true
	srv, w := g.server, m.w
	go func() {
		d, err := fetchDeep(srv, w)
		if err != nil {
			d = nil
		}
		g.deepCh <- d
	}()
}

// deepWaiting diz se o jogador está nas Profundezas sem elas terem chegado.
func (g *Game) deepWaiting(y int) bool { return g.m != nil && !g.m.deepOK && y >= g.m.topH }
