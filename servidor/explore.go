package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"

	"asciimmo/proto"
)

// O que cada personagem já explorou fica no save dele, na escala do
// minimapa (um bit por bloco de MiniScale×MiniScale células). Assim o mapa
// não esquece nada ao reiniciar o jogo, ao trocar de PC ou se o arquivo local
// do cliente se perder; o cliente junta isto com o que guarda em disco.

const exploreR = 28 // raio revelado em volta do jogador, em blocos do minimapa

func (wd *World) fogDims() (int, int) { return wd.w / proto.MiniScale, wd.h / proto.MiniScale }

// explore marca como visto o círculo em volta do jogador.
func (wd *World) explore(p *Player) {
	w, h := wd.fogDims()
	if len(p.fog) != (w*h+7)/8 {
		p.fog = make([]byte, (w*h+7)/8)
	}
	mx, my := p.X/proto.MiniScale, p.Y/proto.MiniScale
	for dy := -exploreR; dy <= exploreR; dy++ {
		for dx := -exploreR; dx <= exploreR; dx++ {
			x, y := mx+dx, my+dy
			if dx*dx+dy*dy > exploreR*exploreR || x < 0 || y < 0 || x >= w || y >= h {
				continue
			}
			i := y*w + x
			p.fog[i>>3] |= 1 << (i & 7)
		}
	}
}

// packFog comprime os bits para o save e para o init.
func packFog(bits []byte) string {
	if len(bits) == 0 {
		return ""
	}
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	zw.Write(bits)
	zw.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

// unpackFog desfaz packFog; devolve nil se o tamanho não bater com o mapa.
func (wd *World) unpackFog(s string) []byte {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(raw) == 0 {
		return nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	bits, err := io.ReadAll(zr)
	w, h := wd.fogDims()
	if err != nil || len(bits) > (w*h+7)/8 {
		return nil
	}
	// Mapa que cresceu para baixo (Profundezas): a névoa antiga é o começo da
	// nova, porque as linhas novas ficam no fim; completa com o não visto.
	return append(bits, make([]byte, (w*h+7)/8-len(bits))...)
}

// FogOf é o explorado do jogador, empacotado, para mandar no init.
func (wd *World) FogOf(id int) string {
	wd.mu.RLock()
	defer wd.mu.RUnlock()
	if p, ok := wd.players[id]; ok {
		return packFog(p.fog)
	}
	return ""
}
