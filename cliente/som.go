package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"log"
	"math"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// Som do jogo: três canais com volume próprio nas opções (O): efeitos
// (golpes, magias, quebras, bichos, passos, interface), ambiente (laços de
// vento, grilos, chuva, mar, caverna, lava e fogo, mais pássaros, corujas,
// sapos e pingos soltos) e música (ogg CC0 em sons/musica, trocando com
// fade conforme o lugar e a hora). Efeitos e ambiente são gerados por código
// (som_sintese.go) numa goroutine na abertura; até ficarem prontos, nada toca.
// Os eventos que disparam cada som estão em som_eventos.go.

//go:embed sons/musica/*.ogg
var musicFS embed.FS

// musicBase é o volume de cada faixa, para nenhuma gritar mais que as outras.
var musicBase = map[string]float64{"dia": 0.55, "vila": 0.5, "noite": 0.6, "cripta": 0.75, "fundo": 0.8, "chefe": 0.45}

type ambLoop struct {
	p         *audio.Player
	cur, want float64
}

type musicTrack struct {
	name string
	p    *audio.Player
	v    float64 // volume atual do fade (0..1)
}

type soundSys struct {
	ctx    *audio.Context
	ready  atomic.Bool
	sfx    map[string][]wave
	loops  map[string]wave
	active []*audio.Player
	last   map[string]float64 // quando cada som tocou por último (ms)
	amb    map[string]*ambLoop
	music  *musicTrack
	old    *musicTrack // faixa saindo no fade
	want   string      // faixa pedida pelo lugar
	wantAt float64     // desde quando a pedida é a mesma
	r      rnd
	muted  bool // janela sem foco ou som desligado nas opções

	// Volumes de cada canal (0..1), das opções.
	vMusic, vSfx, vAmb float64
}

func newSoundSys() *soundSys {
	s := &soundSys{ctx: audio.NewContext(sampleRate), last: map[string]float64{}, amb: map[string]*ambLoop{}, r: rnd{0xabcdef1}}
	go func() {
		s.sfx, s.loops = genSounds(), genLoops()
		s.ready.Store(true)
	}()
	return s
}

// pcm converte um som mono em 16 bits estéreo, com ganho em cada lado.
func pcm(w wave, gl, gr float64) []byte {
	b := make([]byte, len(w)*4)
	for i, v := range w {
		l := int16(clamp(float64(v)*gl, -1, 1) * 32000)
		r := int16(clamp(float64(v)*gr, -1, 1) * 32000)
		binary.LittleEndian.PutUint16(b[i*4:], uint16(l))
		binary.LittleEndian.PutUint16(b[i*4+2:], uint16(r))
	}
	return b
}

// play toca um efeito com volume (0..1) e lado (-1 esquerda, 1 direita).
// Um mesmo som não repete em menos de 45 ms, e há no máximo 28 ao mesmo tempo.
func (s *soundSys) play(name string, vol, pan, now float64) {
	if s != nil {
		s.playCh(name, vol*s.vSfx, pan, now)
	}
}

// playAmb toca um som avulso do ambiente (pássaro, pingo), no volume do ambiente.
func (s *soundSys) playAmb(name string, vol, pan, now float64) {
	if s != nil {
		s.playCh(name, vol*s.vAmb, pan, now)
	}
}

func (s *soundSys) playCh(name string, vol, pan, now float64) {
	if !s.ready.Load() || s.muted {
		return
	}
	if vol < 0.01 {
		return
	}
	vs := s.sfx[name]
	if len(vs) == 0 || now-s.last[name] < 45 {
		return
	}
	s.last[name] = now
	live := s.active[:0]
	for _, p := range s.active {
		if p.IsPlaying() {
			live = append(live, p)
		} else {
			p.Close()
		}
	}
	s.active = live
	if len(s.active) >= 28 {
		return
	}
	w := vs[int(s.r.next())%len(vs)]
	pan = clamp(pan, -1, 1)
	gl, gr := math.Sqrt((1-pan)/2)*1.41, math.Sqrt((1+pan)/2)*1.41
	p := s.ctx.NewPlayerFromBytes(pcm(w, gl*vol, gr*vol))
	p.Play()
	s.active = append(s.active, p)
}

// playWave toca um som pronto (a voz do narrador), no volume dos efeitos.
func (s *soundSys) playWave(w wave, vol float64) {
	if s == nil || !s.ready.Load() || s.muted || len(w) == 0 {
		return
	}
	vol *= s.vSfx
	p := s.ctx.NewPlayerFromBytes(pcm(w, vol*1.2, vol*1.2))
	p.Play()
	s.active = append(s.active, p)
}

// setAmb pede um volume para um laço de ambiente; update faz o fade.
func (s *soundSys) setAmb(name string, v float64) {
	if s == nil || !s.ready.Load() {
		return
	}
	a := s.amb[name]
	if a == nil {
		if v <= 0.005 {
			return
		}
		w := s.loops[name]
		if w == nil {
			return
		}
		b := pcm(w, 1, 1)
		p, err := s.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b))))
		if err != nil {
			return
		}
		p.SetVolume(0)
		p.Play()
		a = &ambLoop{p: p}
		s.amb[name] = a
	}
	a.want = v
}

// setMusic pede uma faixa; ela só entra se continuar pedida por 2,5 s
// (passar rápido pela porta da vila não troca a música).
func (s *soundSys) setMusic(name string, now float64) {
	if s == nil {
		return
	}
	if name != s.want {
		s.want, s.wantAt = name, now
	}
}

func (s *soundSys) startTrack(name string) *musicTrack {
	data, err := musicFS.ReadFile("sons/musica/" + name + ".ogg")
	if err != nil {
		return nil
	}
	st, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		log.Printf("música %s: %v", name, err)
		return nil
	}
	p, err := s.ctx.NewPlayer(audio.NewInfiniteLoop(st, st.Length()))
	if err != nil {
		return nil
	}
	p.SetVolume(0)
	p.Play()
	return &musicTrack{name: name, p: p}
}

// update avança os fades (chamado a cada quadro).
func (s *soundSys) update(dt, now float64) {
	if s == nil {
		return
	}
	k := math.Min(1, dt*1.5)
	for _, a := range s.amb {
		want := a.want * s.vAmb
		if s.muted {
			want = 0
		}
		a.cur += (want - a.cur) * k
		a.p.SetVolume(a.cur)
	}
	// Música: troca depois de 2,5 s pedindo a mesma; fade de ~3 s.
	if s.want != "" && (s.music == nil || s.music.name != s.want) && now-s.wantAt > 2500 && s.old == nil {
		s.old = s.music
		s.music = s.startTrack(s.want)
	}
	fade := dt / 3
	if s.old != nil {
		s.old.v -= fade
		if s.old.v <= 0 {
			s.old.p.Close()
			s.old = nil
		} else {
			s.old.p.SetVolume(s.old.v * musicBase[s.old.name] * s.vMusic)
		}
	}
	if s.music != nil {
		target := 1.0
		if s.muted {
			target = 0
		}
		if s.music.v < target {
			s.music.v = math.Min(target, s.music.v+fade)
		} else {
			s.music.v = math.Max(target, s.music.v-fade)
		}
		s.music.p.SetVolume(s.music.v * musicBase[s.music.name] * s.vMusic)
	}
}
