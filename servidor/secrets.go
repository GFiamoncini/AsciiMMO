package main

import (
	"math"
	"strings"
	"time"

	"asciimmo/proto"
)

// Passagens secretas com pista: santuários selados cuja porta só abre com a
// palavra certa dita em voz alta (chat) ou com uma sequência de setas lida
// em pedras rúnicas espalhadas por perto. As runas da porta e das pedras
// brilham (flag 64) e o cliente solta faíscas nelas: a pista está no mapa.

type sealDef struct {
	Title, Hint string
	X, BY       int      // centro da base da porta
	Word        string   // palavra a dizer (chat); vazia se for por setas
	Seq         string   // sequência de setas: U D L R
	Clues       [][2]int // pedras rúnicas (só nas de setas), na ordem I, II, III...
	Day, Night  int      // o baú lá dentro
}

var seals = []sealDef{
	{Title: "Portão do Amigo", Hint: "Na pedra, em letras antigas: FALE, AMIGO, E ENTRE.",
		X: graveX - 170, BY: graveY - 40, Word: "amigo", Day: itemTideScales, Night: itemTideScales},
	{Title: "Cripta dos Ecos", Hint: "Os ecos guardam a ordem: leia as pedras de I a V e toque as runas (E).",
		X: villageX - 250, BY: villageY + 70, Seq: "ULDRU", Day: itemCryptPlate, Night: itemAsuraGarb},
}

var arrowOf = map[byte]byte{'U': '^', 'D': 'v', 'L': '<', 'R': '>'}

// buildSeals ergue os santuários e as pedras rúnicas.
func (wd *World) buildSeals(seed uint32) {
	for i := range seals {
		s := &seals[i]
		wd.vaultKind(s.X, s.BY, propSeal, seed+uint32(i)*31)
		// Runas na verga da porta, acesas.
		for dx := -4; dx <= 4; dx++ {
			x, y := s.X+dx, s.BY-7
			wd.put(x, y, "*+x+*+x+*"[dx+4], 0x9ad8ff, true, false)
			wd.lit(x, y)
		}
		if s.Seq == "" { // porta da palavra: a inscrição se lê com E
			wd.addRead(s.Title, s.Hint, s.X, s.BY+2)
			continue
		}
		// Pedras rúnicas em volta, cada uma com o numeral e a seta da sua vez.
		s.Clues = nil
		numerals := []string{"I", "II", "III", "IV", "V", "VI"}
		for k := 0; k < len(s.Seq); k++ {
			a := float64(k)/float64(len(s.Seq))*6.283 + 0.7
			cx, cy := s.X+int(48*cos(a)), s.BY+20+int(26*sin(a))
			if x, y, ok := wd.freeNear(cx, cy, 14); ok {
				cx, cy = x, y
			}
			wd.clearArea(cx-4, cy-7, cx+4, cy+1, seed)
			for dy := -5; dy <= 0; dy++ {
				for dx := -2; dx <= 2; dx++ {
					wd.put(cx+dx, cy+dy, pick("#H8&", hash2(cx+dx, cy+dy, seed+960)), dim(0x6a6a7a, 1.05-0.12*float64(dx+2)), dy >= -1, dy < -1)
				}
			}
			n := numerals[k]
			for j := 0; j < len(n); j++ {
				wd.put(cx-len(n)/2+j, cy-4, n[j], 0x9ad8ff, false, true)
				wd.lit(cx-len(n)/2+j, cy-4)
			}
			wd.put(cx, cy-2, arrowOf[s.Seq[k]], 0xc8f0ff, false, true)
			word := map[byte]string{'U': "cima", 'D': "baixo", 'L': "esquerda", 'R': "direita"}[s.Seq[k]]
			wd.addRead("Pedra rúnica "+n, runeArrows(s.Seq[k:k+1])+"  "+word+"\n\nUma das pedras da "+s.Title+
				". Leia todas, de I a "+numerals[len(s.Seq)-1]+", e toque as runas na porta na mesma ordem (E diante dela).", cx, cy)
			wd.lit(cx, cy-2)
			s.Clues = append(s.Clues, [2]int{cx, cy})
		}
	}
}

func (wd *World) sealProp(s *sealDef) *prop {
	for _, pr := range wd.props {
		if pr.Kind == propSeal && pr.X == s.X && pr.Y == s.BY {
			return pr
		}
	}
	return nil
}

// tryUnseal abre o selo perto do jogador se a resposta bater. Chamar com o lock.
func (wd *World) tryUnseal(p *Player, word, seq string, now time.Time) bool {
	for i := range seals {
		s := &seals[i]
		if dist(float64(p.X), float64(p.Y), float64(s.X), float64(s.BY)) > 18 {
			continue
		}
		pr := wd.sealProp(s)
		if pr == nil || !pr.Alive {
			return false
		}
		ok := s.Word != "" && strings.EqualFold(strings.TrimSpace(word), s.Word) ||
			s.Seq != "" && seq == s.Seq
		if !ok {
			if seq != "" {
				wd.pop(p, "As runas se apagam. A ordem não é essa.", 0x8090c0, float64(p.X), float64(p.Y)-34, now)
			}
			return false
		}
		pr.Alive, pr.respawnAt = false, now.Add(propDefs[propSeal].Respawn)
		wd.edit(pr.off)
		wd.effects = append(wd.effects, Effect{Kind: fxHoly, X: float64(s.X), Y: float64(s.BY), R: 18, Start: now, Dur: 800 * time.Millisecond})
		wd.say(ChatLine{Text: p.Name + " abriu o " + s.Title + "."})
		return true
	}
	return false
}

// Unseal recebe a sequência de setas que o jogador tocou nas runas.
func (wd *World) Unseal(id int, seq string) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok && len(seq) <= 12 {
		wd.tryUnseal(p, "", strings.ToUpper(seq), wd.clock())
	}
}

func (wd *World) sealInfo() []proto.Seal {
	var out []proto.Seal
	for _, s := range seals {
		k := 0
		if s.Seq != "" {
			k = 1
		}
		out = append(out, proto.Seal{Title: s.Title, Hint: s.Hint, X: s.X, Y: s.BY, Kind: k, Len: len(s.Seq), Clues: s.Clues})
	}
	return out
}

func cos(a float64) float64 { return math.Cos(a) }
func sin(a float64) float64 { return math.Sin(a) }
