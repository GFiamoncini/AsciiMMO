package main

import "fmt"

// Casa própria: duas casas da vila estão à venda. E na porta compra; depois
// de comprada, E na porta abre o baú da casa, onde se guardam itens e
// dinheiro (ficam no save, a salvo de morrer). O mundo é de todos, então a
// casa é "sua" para você: outro jogador pode comprar a mesma e ter o baú dele.
// Descansar não precisa de casa (a fogueira faz isso), mas a casa grava o jogo.

type houseLot struct {
	Name  string
	I     int // índice em houses (lugares.go)
	Price int
}

var lots = []houseLot{
	{"Casa do Poço", 3, 1200},
	{"Casa da Praça", 4, 2000},
}

const (
	lotReach  = 10.0
	stashKind = 60 // tipos de item que cabem no baú
)

// lotDoor é a frente da porta da casa (onde o jogador fica para usá-la).
func lotDoor(i int) (int, int) {
	h := houses[lots[i].I]
	return villageX + h[0] + h[2]/2, villageY + h[1] + h[3] + 2
}

func (wd *World) nearLot(p *Player) int {
	for i := range lots {
		x, y := lotDoor(i)
		if dist(float64(p.X), float64(p.Y), float64(x), float64(y)) <= lotReach {
			return i
		}
	}
	return -1
}

// BuyHouse compra a casa i, se o jogador estiver na porta e tiver o dinheiro.
func (wd *World) BuyHouse(id, i int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || i < 0 || i >= len(lots) || wd.nearLot(p) != i || p.House == i+1 {
		return
	}
	now := wd.clock()
	if p.Gold < lots[i].Price {
		wd.pop(p, fmt.Sprintf("Faltam %d %s para a %s", lots[i].Price-p.Gold, currency, lots[i].Name), 0xd08060, float64(p.X), float64(p.Y)-34, now)
		return
	}
	p.Gold -= lots[i].Price
	p.House = i + 1
	wd.say(ChatLine{Text: p.Name + " comprou a " + lots[i].Name + "."})
	wd.persist(p)
}

// Stash mexe no baú da casa: op 0 guarda n do item, 1 tira n do item,
// 2 guarda n de dinheiro, 3 tira n de dinheiro.
func (wd *World) Stash(id, op, it, n int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	p, ok := wd.players[id]
	if !ok || p.House == 0 || wd.nearLot(p) != p.House-1 || n <= 0 {
		return
	}
	switch op {
	case 0:
		for k := 0; k < n && p.count(it) > 0; k++ {
			if stashCount(p.Stash, it) == 0 && len(p.Stash) >= stashKind {
				wd.pop(p, "O baú está cheio", 0xd08060, float64(p.X), float64(p.Y)-34, wd.clock())
				break
			}
			p.unequipLast(it)
			p.take(it)
			p.Stash = stashAdd(p.Stash, it, 1)
		}
	case 1:
		for k := 0; k < n && stashCount(p.Stash, it) > 0; k++ {
			if !p.give(it, 1) {
				wd.pop(p, "A mochila está cheia", 0xd08060, float64(p.X), float64(p.Y)-34, wd.clock())
				break
			}
			p.Stash = stashAdd(p.Stash, it, -1)
		}
	case 2:
		n = min(n, p.Gold)
		p.Gold -= n
		p.StashGold += n
	case 3:
		n = min(n, p.StashGold)
		p.StashGold -= n
		p.Gold += n
	}
	wd.persist(p) // o baú grava na hora
}

func stashCount(s []invSlot, it int) int {
	for _, v := range s {
		if v.Item == it {
			return v.N
		}
	}
	return 0
}

func stashAdd(s []invSlot, it, n int) []invSlot {
	for i := range s {
		if s[i].Item == it {
			s[i].N += n
			if s[i].N <= 0 {
				return append(s[:i], s[i+1:]...)
			}
			return s
		}
	}
	if n > 0 {
		s = append(s, invSlot{Item: it, N: n})
	}
	return s
}

// houseMarks põe na porta de cada casa à venda uma placa que se lê com E.
func (wd *World) houseMarks(seed uint32) {
	for i, l := range lots {
		x, y := lotDoor(i)
		wd.addRead(l.Name, fmt.Sprintf("À venda por %d %s. Com ela vem um baú para guardar itens e dinheiro. E na porta compra.", l.Price, currency), x+6, y-1)
	}
}
