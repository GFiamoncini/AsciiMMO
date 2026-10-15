package main

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMapaCores: o mapa de cima chega com as Profundezas reservadas embaixo,
// e as cores, letras e flags da parte de cima ficam intactas. Antes o append
// da reserva escrevia zeros por cima das cores, e o norte do mapa ficava preto.
func TestMapaCores(t *testing.T) {
	const w, h, full = 40, 30, 70
	n := w * h
	b := make([]byte, 8+5*n)
	copy(b, "AMAP")
	binary.LittleEndian.PutUint16(b[4:], w)
	binary.LittleEndian.PutUint16(b[6:], h)
	for i := 0; i < n; i++ {
		b[8+i] = 'A' + byte(i%26)
		b[8+n+3*i], b[8+n+3*i+1], b[8+n+3*i+2] = byte(i), byte(i>>8)|1, 200
		b[8+4*n+i] = byte(i%7) + 1
	}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("X-Map-Full-H", "70")
		rw.Write(b)
	}))
	defer srv.Close()
	m, err := fetchMap(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if m.h != full || m.deepOK || len(m.glyph) != w*full || len(m.color) != 3*w*full || len(m.flags) != w*full {
		t.Fatalf("tamanhos: h=%d deepOK=%v glyph=%d color=%d flags=%d", m.h, m.deepOK, len(m.glyph), len(m.color), len(m.flags))
	}
	for i := 0; i < n; i++ {
		if m.glyph[i] != 'A'+byte(i%26) || m.color[3*i] != byte(i) || m.color[3*i+1] != byte(i>>8)|1 || m.color[3*i+2] != 200 || m.flags[i] != byte(i%7)+1 {
			t.Fatalf("célula %d estragada: %c %d,%d,%d flags %d", i, m.glyph[i], m.color[3*i], m.color[3*i+1], m.color[3*i+2], m.flags[i])
		}
	}
}
