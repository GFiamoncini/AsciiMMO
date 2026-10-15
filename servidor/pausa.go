package main

import (
	"time"
)

// Pausa e salvar. O mundo é compartilhado, então pausar tem dois jeitos:
//   - todos os jogadores conectados pausados (o caso de quem joga sozinho):
//     o mundo inteiro para. O relógio do mundo (clock) deixa de andar, e tudo
//     o que conta tempo (renascer, recarga, efeito, dia e noite) fica onde
//     estava;
//   - só alguns pausados: o mundo segue para os outros, e quem pausou fica
//     parado e protegido (inimigo não o vê, nada o fere, não regenera).
// O cliente pede pausa no menu de pausa (Esc) e, se a opção estiver ligada,
// enquanto houver janela aberta (mochila, loja, teia...).
//
// Salvar: o personagem é gravado a cada 30 s, ao sair, ao descansar e ao
// pedir (F5 ou o menu de pausa); agora com o lugar onde ele estava. Cada
// gravação sobe Player.saves, que vai ao cliente para mostrar "salvo".

// clock é a hora do mundo: a hora real menos o tempo em que ele ficou parado.
func (wd *World) clock() time.Time {
	return time.Now().Add(-time.Duration(wd.pausedNs.Load()))
}

// frozen diz se o mundo está parado: há gente conectada e todos pausaram.
// Chamar com wd.mu travado.
func (wd *World) frozen() bool {
	if len(wd.players) == 0 {
		return false
	}
	for _, p := range wd.players {
		if !p.paused {
			return false
		}
	}
	return true
}

// SetPaused liga ou desliga a pausa de um jogador.
func (wd *World) SetPaused(id int, on bool) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok && p.paused != on {
		p.paused = on
		p.moveX, p.moveY, p.sprint = 0, 0, false
	}
}

// SaveNow grava o personagem na hora (pedido do jogador).
func (wd *World) SaveNow(id int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok {
		wd.persist(p)
		if wd.saveDir == "" || p.token == "" {
			wd.pop(p, "Este servidor não guarda personagens", 0xc08080, float64(p.X), float64(p.Y)-34, wd.clock())
			return
		}
		wd.pop(p, "Jogo salvo", 0x90e090, float64(p.X), float64(p.Y)-34, wd.clock())
	}
}

// savedPos diz onde o personagem salvo deve reaparecer: no lugar gravado, se
// ainda couber lá; senão onde spawn o pôs.
func (wd *World) savedPos(p *Player, sv saveData) {
	if !sv.HasPos || p.inPrologue() || !wd.fits(sv.PosX, sv.PosY) {
		return
	}
	if inDeep(sv.PosY) && sv.PosY >= deepY0+deepH {
		return
	}
	p.X, p.Y = sv.PosX, sv.PosY
}
