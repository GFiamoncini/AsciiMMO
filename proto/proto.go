// Package proto define as mensagens trocadas entre o servidor e os clientes
// (navegador e nativo): o init, o estado de cada tick e o que o jogador manda.
// Tudo vai em JSON por WebSocket.
package proto

// Tamanhos fixos que o servidor também usa.
const (
	NumAttrs  = 4 // vida, mana, força, stamina
	BarSlots  = 9 // atalhos de golpes na barra (teclas 1 a 9; abrem com o nível)
	NumStatus = 3 // veneno, frio, calor
	MiniScale = 4 // o minimapa reduz o mapa 4 vezes (e o explorado é guardado nessa escala)
)

type Skill struct {
	Icon string  `json:"icon,omitempty"`
	Desc string  `json:"desc,omitempty"`
	Elem int     `json:"elem,omitempty"`
	Name string  `json:"name"`
	Cost int     `json:"cost"`
	CD   float64 `json:"cd"`
}

type Portal struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Label string `json:"label"`
	Up    bool   `json:"up,omitempty"` // sobe para a luz (saída da cripta, da caverna, das Profundezas)
}

type Init struct {
	Init      bool        `json:"init"`
	ID        int         `json:"id"`
	W         int         `json:"w"`
	H         int         `json:"h"`
	Class     string      `json:"class"`
	Skills    []Skill     `json:"skills"`
	Lights    [][4]int    `json:"lights"` // x, y, raio, tipo; a primeira é a fogueira
	Portals   []Portal    `json:"portals"`
	OverH     int         `json:"overH"` // abaixo disso é a cripta
	Items     []Item      `json:"items"`
	Regions   []Region    `json:"regions"`
	Attrs     [][2]string `json:"attrs"`               // nome e efeito de cada atributo
	Tree      []Node      `json:"tree"`                // nós da árvore de habilidades
	Achs      [][2]string `json:"achs"`                // nome e descrição de cada conquista
	Chests    [][2]int    `json:"chests"`              // posição dos baús
	Docks     [][2]int    `json:"docks"`               // ponta de cada cais
	Zones     []Zone      `json:"zones"`               // áreas do subsolo, para o minimapa
	SteedK    []SteedK    `json:"steedk"`              // tipos de montaria
	HerbK     []int       `json:"herbk"`               // item que cada tipo de erva dá
	Status    []string    `json:"status"`              // nome de cada status negativo
	Raft      int         `json:"raft"`                // madeira para uma jangada
	Currency  string      `json:"cur"`                 // nome do dinheiro do jogo
	Shops     []Shop      `json:"shops"`               // mercadores
	Bonfires  []Bonfire   `json:"bonfires"`            // fogueiras onde se renasce
	Chapters  [][2]string `json:"chapters"`            // título e abertura de cada capítulo
	Sides     []SideDef   `json:"sides"`               // missões secundárias
	Quests    []SideDef   `json:"quests,omitempty"`    // missões principais em ordem (diário)
	Houses    []HouseLot  `json:"houses,omitempty"`    // casas à venda (porta e preço)
	Beasts    []Beast     `json:"beasts,omitempty"`    // bestiário: um por tipo de inimigo, na ordem do servidor
	SteedLore []string    `json:"steedlore,omitempty"` // bestiário: montarias
	Waters    []string    `json:"waters,omitempty"`    // águas de pesca, na ordem dos bits
	Fishes    []string    `json:"fishes,omitempty"`    // onde e quando cada peixe aparece
	Fish0     int         `json:"fish0,omitempty"`     // id do primeiro peixe (os outros seguem)
	Modes     []string    `json:"modes,omitempty"`     // nomes dos modos de combate
	Fog       string      `json:"fog,omitempty"`       // explorado, em bits na escala do minimapa, gzip + base64
	Doors     [][2]int    `json:"doors"`               // portas de cofre (centro da base)
	Travel    []TravelPt  `json:"travel"`              // pontos de viagem rápida
	Seals     []Seal      `json:"seals"`               // portas seladas com pista
	NPCs      []NPC       `json:"npcs"`                // gente com quem conversar
	Techs     []Tech      `json:"techs"`               // técnicas do Mestre Oren
	Gates     []Gate      `json:"gates"`               // Portões Rúnicos
	Reads     []Read      `json:"reads"`               // o que se lê com E
	Book      []Skill     `json:"book"`                // todos os golpes da classe (Skills tem só os 3 antigos, para o navegador)
	Web       []WebNode   `json:"web"`                 // teia de habilidades
	Branches  []Branch    `json:"branches"`            // ramos da teia
	MainAttr  int         `json:"mattr"`               // atributo da classe (a teia pede pontos nele)
	Elems     []string    `json:"elems"`               // nome de cada elemento
	Affinity
}

type Zone struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

type SteedK struct {
	Name  string `json:"name"`
	Level int    `json:"lv"`
	Speed int    `json:"speed"`
	Def   int    `json:"def"`
}

// Thing é algo parado ou lento no chão: montaria selvagem ou erva.
type Thing struct {
	ID   int     `json:"id"`
	Kind int     `json:"k"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// Affinity diz ao cliente com que armas a classe do jogador se dá bem.
type Affinity struct {
	WTypes []string `json:"wtypes"` // nome de cada tipo de arma
	Aff    []int    `json:"aff"`    // tipos com afinidade
	AffPct [2]int   `json:"affPct"` // % do bônus com e sem afinidade
}

// Error vai no lugar do init quando a entrada é recusada.
type Error struct {
	Error string `json:"error"`
}

type Node struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	Max  int    `json:"max"`
}

type Item struct {
	Name   string `json:"name"`
	Kind   int    `json:"kind"` // 0 poção, 1 arma, 2 skin
	Rarity int    `json:"rar"`
	Bonus  int    `json:"bonus"`
	Def    int    `json:"def"`
	Magic  int    `json:"magic"`
	WType  int    `json:"wt"` // tipo da arma; -1 se não for arma
	Color  int    `json:"color"`
	Desc   string `json:"desc"`
	Price  int    `json:"price,omitempty"` // peixe: quanto o mercador paga
}

type Region struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	R    int    `json:"r"`
}

type Loot struct {
	ID   int `json:"id"`
	Item int `json:"item"`
	X    int `json:"x"`
	Y    int `json:"y"`
	N    int `json:"n,omitempty"` // dinheiro: quanto há no montinho
}

// Quest é a missão atual do jogador; some quando a corrente acaba.
type Quest struct {
	N       int     `json:"n"` // número da missão, a partir de 1
	Of      int     `json:"of"`
	Title   string  `json:"title"`
	Text    string  `json:"text"`
	Prog    int     `json:"prog"`
	Need    int     `json:"need"`
	Chapter int     `json:"ch"`
	Steps   []QStep `json:"steps,omitempty"` // subobjetivos, em ordem
	Cur     int     `json:"cur"`             // etapa atual
}

type Player struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	X     int     `json:"x"`
	Y     int     `json:"y"`
	Color int     `json:"c"`
	HP    int     `json:"hp"`
	MaxHP int     `json:"mhp"`
	MP    int     `json:"mp"`
	MaxMP int     `json:"mmp"`
	Level int     `json:"lv"`
	Gold  int     `json:"g"`
	Hit   float64 `json:"hit"` // segundos desde o último golpe levado
	Class int     `json:"cls"`
	Act   int     `json:"act"` // último golpe usado, há quanto tempo e para onde
	ActAt float64 `json:"aa"`
	Dir   float64 `json:"ad"`
	Guard bool    `json:"guard"`
	Equip int     `json:"w"`  // arma equipada, para todos verem
	Skin  int     `json:"sk"` // skin equipada
	Ride  int     `json:"mt"` // montaria em uso + 1
	Boat  bool    `json:"bt"`
	Light int     `json:"lt,omitempty"` // luz acesa: 1 tocha, 2 lanterna
	Pz    bool    `json:"pz,omitempty"` // pausado (os outros veem "pausado")
}

type Boss struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	HP      int     `json:"hp"`
	MaxHP   int     `json:"mhp"`
	Alive   bool    `json:"alive"`
	Respawn int     `json:"resp"`
	Hit     float64 `json:"hit"`
}

type Mob struct {
	ID    int     `json:"id"`
	Kind  int     `json:"k"`
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	HP    int     `json:"hp"`
	MaxHP int     `json:"mhp"`
	Age   float64 `json:"age"` // segundos desde que saiu da cova
	Hit   float64 `json:"hit"`
	Atk   float64 `json:"atk"` // segundos desde o último golpe dado
	Boss  bool    `json:"boss"`
	St    int     `json:"st,omitempty"`   // efeitos: 1 lento, 2 atordoado, 4 queimando, 8 envenenado
	Lv    int     `json:"lv"`             // nível do inimigo
	Ally  bool    `json:"ally,omitempty"` // servo de um jogador (necromante)
}

type Fx struct {
	Kind int     `json:"k"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	R    float64 `json:"r"`
	Dir  float64 `json:"dir"`
	P    float64 `json:"p"`
}

type Proj struct {
	ID   int     `json:"id"`
	Kind int     `json:"k"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type Chat struct {
	From  string `json:"from"`
	Color int    `json:"c"`
	Text  string `json:"text"`
}

type State struct {
	T       float64    `json:"t"`
	Players []Player   `json:"players"`
	Boss    Boss       `json:"boss"`
	Mobs    []Mob      `json:"mobs"`
	Fx      []Fx       `json:"fx"`
	Projs   []Proj     `json:"projs"`
	Chat    []Chat     `json:"chat"`
	CDs     [3]float64 `json:"cds"` // segundos até cada golpe voltar
	Unl     bool       `json:"unl"` // golpe da cripta liberado
	Tod     float64    `json:"tod"` // hora do dia, 0..1
	Inv     [][2]int   `json:"inv"` // item, quantidade
	Loot    []Loot     `json:"loot"`
	Quest   *Quest     `json:"quest"`
	Season  int        `json:"season"` // 0 primavera, 1 verão, 2 outono, 3 inverno
	Wx      int        `json:"wx"`     // 0 tempo bom, 1 chuva, 2 tempestade

	// Progresso de quem recebe o estado.
	XP     int           `json:"xp"`
	XPNeed int           `json:"xpn"`
	Points int           `json:"pts"`  // pontos de atributo a gastar
	SPts   int           `json:"spts"` // pontos de árvore a gastar
	Attr   [NumAttrs]int `json:"attr"`
	Tree   []int         `json:"tree"` // graus de cada nó da teia (o navegador mostra como lista)
	CD2    []float64     `json:"cd2"`  // segundos até cada golpe do livro voltar
	Bar    [BarSlots]int `json:"bar"`  // golpe em cada atalho (-1 vazio)
	BarN   int           `json:"barn"` // atalhos já abertos (5 a 9, pelo nível)
	Buf    [3]float64    `json:"buf"`  // segundos de reforço: dano, crítico, veneno nas lâminas
	ST     int           `json:"st"`
	MaxST  int           `json:"mst"`
	Dmg    [2]int        `json:"dmg"`    // % de dano no golpe básico e nos de classe
	Def    int           `json:"def"`    // % de proteção
	Ach    uint          `json:"ach"`    // bits das conquistas
	Chests []bool        `json:"chests"` // baús já abertos hoje
	Gear   [5]int        `json:"gear"`   // equipado: arma, elmo, armadura, botas, amuleto

	Steeds []Thing            `json:"steeds"`       // montarias selvagens por perto
	Herbs  []Thing            `json:"herbs"`        // ervas por perto
	Sts    [NumStatus]float64 `json:"sts"`          // segundos que faltam de cada status
	Imm    [NumStatus]float64 `json:"imm"`          // segundos de proteção da erva
	Owned  uint               `json:"owned"`        // bits das montarias domadas
	Ed     [][4]int           `json:"ed,omitempty"` // células mudadas: índice, caractere, cor, flags
	MV     int                `json:"mv"`           // versão do mapa depois dessas mudanças
	Reload bool               `json:"reload,omitempty"`

	Chapter    int       `json:"chap"`             // capítulo atual (1..6; 7 depois do fim)
	SideDone   uint      `json:"sdone"`            // bits das secundárias concluídas
	SideProg   []int     `json:"sprog"`            // progresso de cada secundária
	Bonfire    int       `json:"bf"`               // última fogueira acesa
	Purged     []int     `json:"purged,omitempty"` // fogueiras com a área purificada agora
	Event      int       `json:"ev,omitempty"`     // 1 = invasão noturna em curso
	Pops       []Pop     `json:"pops"`
	Moor       *[2]int   `json:"moor,omitempty"`  // onde ficou o seu barco atracado
	Marks      []Mark    `json:"marks,omitempty"` // onde fazer as missões em aberto
	BagCap     int       `json:"bag"`             // espaços da mochila
	Found      uint64    `json:"found"`           // pontos de viagem descobertos
	Tech       int       `json:"tech,omitempty"`  // técnicas do Mestre aprendidas (bits)
	Gates      uint32    `json:"gates,omitempty"` // Portões Rúnicos acesos (bits)
	Paused     bool      `json:"pz,omitempty"`    // você está pausado
	Frozen     bool      `json:"frz,omitempty"`   // o mundo parou (todos pausados)
	Saves      int       `json:"sv,omitempty"`    // vezes que o personagem foi gravado (muda = "salvo")
	LightSlot  int       `json:"lslot,omitempty"` // lanterna ou tocha no espaço de luz
	Tame       *Tame     `json:"tame,omitempty"`  // doma em andamento: a sequência a tocar
	House      int       `json:"house,omitempty"` // casa comprada + 1
	Stash      [][2]int  `json:"stash,omitempty"` // baú da casa: item, quantidade
	StashGold  int       `json:"stashg,omitempty"`
	Mode       int       `json:"mode,omitempty"`   // modo de combate: 0 pacifista, 1 defensivo, 2 agressivo
	Goal       *[2]int   `json:"goal,omitempty"`   // caminhando até este ponto
	Fish       *FishLine `json:"fish,omitempty"`   // linha de pesca na água
	FishKinds  uint64    `json:"fishk,omitempty"`  // peixes já pegos (bits)
	FishBodies uint32    `json:"fishb,omitempty"`  // águas onde já pescou (bits)
	BestSeen   uint64    `json:"bseen,omitempty"`  // bestiário: inimigos vistos (bits)
	SteedSeen  uint32    `json:"sseen,omitempty"`  // montarias vistas (bits)
	BestKills  []int     `json:"bkills,omitempty"` // mortes por tipo de inimigo
}

// Beast é uma criatura do bestiário.
type Beast struct {
	Name   string   `json:"name"`
	Lv     int      `json:"lv"`
	HP     int      `json:"hp"`
	Dmg    int      `json:"dmg"`
	DmgSp  int      `json:"dsp"`
	Boss   bool     `json:"boss,omitempty"`
	Lore   string   `json:"lore"`
	Weak   []string `json:"weak,omitempty"`
	Resist []string `json:"resist,omitempty"`
}

// FishLine é a linha de pesca: onde está a boia e se o peixe está mordendo.
type FishLine struct {
	X     int  `json:"x"`
	Y     int  `json:"y"`
	Bite  bool `json:"bite,omitempty"`
	Water int  `json:"w"`
}

// HouseLot é uma casa à venda: nome, frente da porta e preço.
type HouseLot struct {
	Name  string `json:"name"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Price int    `json:"price"`
}

// Tame é a doma em andamento: o bicho, a sequência (U D L R) e o tempo que falta.
type Tame struct {
	Name string  `json:"name"`
	Seq  string  `json:"seq"`
	Left float64 `json:"left"`
}

// In é o que o navegador manda: direção segurada, golpes e chat.
type In struct {
	T    string   `json:"t"`
	DX   int      `json:"dx"`
	DY   int      `json:"dy"`
	Slot int      `json:"s"`
	Item int      `json:"item"`
	I    int      `json:"i"`   // índice do atributo ou do nó da árvore
	Run  bool     `json:"sp"`  // Shift segurado: corre gastando stamina
	Dir  *float64 `json:"dir"` // mira do mouse; nulo usa a direção do andar
	Text string   `json:"text"`
}

// Shop é um mercador: onde está e o que vende (item, preço).
type Shop struct {
	Name  string   `json:"name"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Kind  int      `json:"k"` // 0 armas, 1 ervas
	Stock [][2]int `json:"stock"`
}

type Bonfire struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// SideDef é uma missão secundária: a partir de que capítulo vale e que regiões abre.
type SideDef struct {
	Title   string  `json:"title"`
	Text    string  `json:"text"`
	Chapter int     `json:"ch"`
	Steps   []QStep `json:"steps,omitempty"` // subobjetivos, em ordem
	Cur     int     `json:"cur"`             // etapa atual
	Need    int     `json:"need"`
	Unlock  []int   `json:"unlock"`
}

// Pop é um texto que sobe sobre a cabeça: XP, dinheiro, avisos. Kind 1 é a morte.
type Pop struct {
	ID    int     `json:"id"`
	Text  string  `json:"text"`
	Color int     `json:"c"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Kind  int     `json:"k"`
}

// Mark é o lugar de uma missão no mapa. Se fica no subsolo (Under), EX/EY é a
// entrada para quem está em cima.
type Mark struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	R     int    `json:"r"`
	EX    int    `json:"ex,omitempty"`
	EY    int    `json:"ey,omitempty"`
	Under bool   `json:"under,omitempty"`
	Title string `json:"title"`
	Side  bool   `json:"side,omitempty"`
}

// WebNode é um nó da teia de habilidades.
type WebNode struct {
	Name  string  `json:"name"`
	Desc  string  `json:"desc"`
	Icon  string  `json:"icon"`
	Elem  int     `json:"elem"`
	Ring  int     `json:"ring"`
	Ang   float64 `json:"ang"`
	Links []int   `json:"links"`
	Max   int     `json:"max"`
	Lv    int     `json:"lv"`
	Attr  int     `json:"attr"`
	Skill int     `json:"skill"` // golpe que ensina; -1 passivo
	Crypt bool    `json:"crypt,omitempty"`
}

// Branch é um ramo da teia: nome, ângulo e elemento.
type Branch struct {
	Name string  `json:"name"`
	Ang  float64 `json:"ang"`
	Elem int     `json:"elem"`
}

// QStep é um subobjetivo da missão: o que fazer e quanto falta.
type QStep struct {
	Text string `json:"text"`
	Prog int    `json:"prog"`
	Need int    `json:"need"`
	Done bool   `json:"done,omitempty"`
}

// TravelPt é um ponto de viagem rápida (fogueira ou placa).
type TravelPt struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Kind int    `json:"k"`
}

// Seal é uma porta selada: a pista, onde fica e como abre (0 palavra no chat,
// 1 sequência de setas de tamanho Len, lida nas pedras Clues).
type Seal struct {
	Title string   `json:"title"`
	Hint  string   `json:"hint"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Kind  int      `json:"k"`
	Len   int      `json:"len"`
	Clues [][2]int `json:"clues,omitempty"`
}

// NPC é alguém com quem se conversa (E): o cliente mostra as falas.
type NPC struct {
	Name  string   `json:"name"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Lines []string `json:"lines"`
	Teach bool     `json:"teach,omitempty"` // ensina técnicas depois da conversa
}

// Tech é uma técnica do Mestre: o golpe no livro, a sequência de setas (U D L
// R), o nível e o preço.
type Tech struct {
	Skill int    `json:"skill"`
	Seq   string `json:"seq"`
	Lv    int    `json:"lv"`
	Price int    `json:"price"`
}

// Gate é um Portão Rúnico: nome, lugar e a runa (setas U D L R; vazia = acende
// sem runa). O cliente só mostra a runa de um portão quando o anterior da
// corrente está aceso.
type Gate struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Seq  string `json:"seq,omitempty"`
}

// Read é algo que se lê com E: placa, poste de direção, pedra rúnica,
// inscrição. Título e texto já prontos para a janela de leitura.
type Read struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}
