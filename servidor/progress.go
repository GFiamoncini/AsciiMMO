package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Atributos: cada nível dá pontos para distribuir entre eles.
const (
	attrVit = iota
	attrMana
	attrStr
	attrSta
	numAttrs
)

const (
	pointsPerLevel = 3
	vitHP          = 10 // vida por ponto
	manaMP         = 6  // mana por ponto
	strDmg         = 3  // % de dano por ponto
	staST          = 10 // stamina por ponto
	baseST         = 100
	sprintCost     = 1.0 // stamina por tick correndo
	stRegen        = 0.6 // stamina por tick parado ou andando
	maxDef         = 60  // teto da proteção, em %
)

var attrNames = [numAttrs]string{"Vida", "Mana", "Força", "Stamina"}
var attrDescs = [numAttrs]string{"+10 de vida", "+6 de mana", "+3% de dano", "+10 de stamina (Shift corre)"}

// numOldTree é o tamanho da árvore antiga (6 nós), só para ler saves velhos.
const numOldTree = 6

// xpNeed é quanto falta de experiência para sair do nível dado.
func xpNeed(level int) int { return 60 + 40*level*level }

// recalc refaz os bônus da teia e os máximos a partir da classe, do nível,
// dos atributos e das passivas.
func (p *Player) recalc() {
	p.fixWeb()
	p.recalcBonus()
	c := classes[p.Class]
	p.MaxHP = c.HP + 8*(p.Level-1) + vitHP*p.Attr[attrVit] + p.bon.HP
	p.MaxMP = c.MP + 4*(p.Level-1) + manaMP*p.Attr[attrMana] + p.bon.MP
	p.MaxST = baseST + staST*p.Attr[attrSta]
	p.HP, p.MP, p.ST = min(p.HP, p.MaxHP), min(p.MP, p.MaxMP), min(p.ST, float64(p.MaxST))
}

// dmgPct é o multiplicador de dano, em %, para o golpe i do livro: arma,
// Força, passivas da teia e, nas classes mágicas, o bônus das peças.
func (p *Player) dmgPct(i int) int {
	pct := 100 + p.weaponBonus() + strDmg*p.Attr[attrStr]
	if i == 0 {
		pct += p.bon.Basic
	} else {
		pct += p.bon.Skill
	}
	// Amuleto soma dano para todos; o bônus mágico das peças, só nas classes mágicas.
	for _, it := range p.worn() {
		pct += items[it].Bonus
		if classes[p.Class].Magic {
			pct += items[it].Magic
		}
	}
	return pct
}

// defPct é quanto do dano recebido as peças e a teia absorvem, em %.
func (p *Player) defPct() int {
	def := p.bon.Def
	if p.Riding != 0 {
		def += steedKinds[p.Riding-1].Def
	}
	for _, it := range p.worn() {
		def += items[it].Def
	}
	return min(maxDef, def)
}

// slot devolve onde fica equipada uma peça do tipo dado; nil se o tipo não
// se equipa (poção).
func (p *Player) slot(kind int) *int {
	switch kind {
	case kindWeapon:
		return &p.Equip
	case kindSkin:
		return &p.Skin
	case kindHelm:
		return &p.Helm
	case kindBoots:
		return &p.Boots
	case kindAmulet:
		return &p.Amulet
	}
	return nil
}

// worn lista as peças vestidas, fora a arma (0 onde o espaço está vazio).
func (p *Player) worn() [4]int { return [4]int{p.Skin, p.Helm, p.Boots, p.Amulet} }

func (wd *World) levelUp(p *Player) {
	p.Level++
	p.Points += pointsPerLevel
	p.SkillPts++
	p.recalc()
	p.HP, p.MP, p.ST = p.MaxHP, p.MaxMP, float64(p.MaxST)
	wd.say(ChatLine{Text: p.Name + " chegou ao nível " + strconv.Itoa(p.Level) + "."})
}

func (wd *World) gainXP(p *Player, n int) {
	for p.XP += n; p.XP >= xpNeed(p.Level); {
		p.XP -= xpNeed(p.Level)
		wd.levelUp(p)
	}
}

// SpendAttr gasta um ponto de nível no atributo i.
func (wd *World) SpendAttr(id, i int) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if p, ok := wd.players[id]; ok && i >= 0 && i < numAttrs && p.Points > 0 {
		p.Points--
		p.Attr[i]++
		p.recalc()
	}
}

// Conquistas: cada uma é checada uma vez por segundo e paga ouro ao sair.
type achievement struct {
	Name, Desc string
	Gold       int
	ok         func(p *Player) bool
}

// Bits de Player.Bosses, na ordem em que os chefes aparecem nas missões.
var bossBit = map[int]uint{mobLord: 0, mobQueen: 1, mobTyrant: 2, mobArchon: 3, killDragon: 4}

const (
	allBosses = 1<<5 - 1
	seenCrypt = 30 // bit de Player.Seen para a cripta; os outros são as regiões
)

var achievements = []achievement{
	{"Primeiro Sangue", "Derrote um inimigo", 20, func(p *Player) bool { return p.Kills >= 1 }},
	{"Caçador", "Derrote 50 inimigos", 150, func(p *Player) bool { return p.Kills >= 50 }},
	{"Exterminador", "Derrote 250 inimigos", 600, func(p *Player) bool { return p.Kills >= 250 }},
	{"Explorador", "Visite todas as regiões e a cripta", 300, func(p *Player) bool { return p.Seen == 1<<len(regions)-1|1<<seenCrypt }},
	{"Caça-Tesouros", "Abra 5 baús", 200, func(p *Player) bool { return p.Chests >= 5 }},
	{"Lendário", "Obtenha um item lendário", 400, func(p *Player) bool {
		for _, s := range p.inv {
			if items[s.Item].Rarity == rarLegend {
				return true
			}
		}
		return false
	}},
	{"Matador de Chefes", "Derrote os cinco chefes", 800, func(p *Player) bool { return p.Bosses == allBosses }},
	{"Veterano", "Chegue ao nível 5", 200, func(p *Player) bool { return p.Level >= 5 }},
	{"Mestre", "Chegue ao nível 10", 600, func(p *Player) bool { return p.Level >= 10 }},
	{"Herói", "Conclua todas as missões", 1000, func(p *Player) bool { return p.Quest >= len(quests) }},
}

func (wd *World) achieve(p *Player, now time.Time) {
	for i, a := range achievements {
		if p.Ach&(1<<i) != 0 || !a.ok(p) {
			continue
		}
		p.Ach |= 1 << i
		p.Gold += a.Gold
		wd.say(ChatLine{Text: p.Name + " desbloqueou a conquista \"" + a.Name + "\"."})
		wd.effects = append(wd.effects, Effect{Kind: fxHoly, X: float64(p.X), Y: float64(p.Y), R: 14, Start: now, Dur: 600 * time.Millisecond})
	}
}

// chest é um baú: cada jogador abre uma vez por dia do jogo. Além de ouro e
// poção, pode sair o item único do baú — um para o dia, outro para a noite.
type chest struct{ X, Y, Day, Night int }

var chests = []chest{
	{340, 732 + cryptDY, itemCryptPlate, itemMidnightDagger},
	{492, 738 + cryptDY, itemTravelCloak, itemTravelCloak},
	{citadelX + 30, citadelY + 22, itemMarapaBow, itemAsuraGarb},
	{tideX - 14, tideY + 20, itemTideScales, 0},
	{radiantX + 26, radiantY + 22, itemSolarRelic, itemRadiantAegis},
	{islandX, islandY + 2, itemLakeHelm, itemLakeHelm},                                            // ilha do lago: só de barco
	{1326, overH + 190, itemDeepAmulet, itemDeepAmulet},                                           // sala atrás da parede rachada
	{grottoX + 4, grotY - 2, itemExplorerBoots, itemExplorerBoots},                                // gruta do buraco
	{vaults[0][0], vaults[0][1] - 9, itemVaelorHelm, itemVaelorHelm},                              // cofre das Ruínas de Vaelor
	{vaults[1][0], vaults[1][1] - 9, itemGuardBoots, itemGuardBoots},                              // depósito da guarda
	{graveX - 170, graveY - 40 - 9, itemTideScales, itemTideScales},                               // Portão do Amigo (selo)
	{villageX - 250, villageY + 70 - 9, itemCryptPlate, itemAsuraGarb},                            // Cripta dos Ecos (selo)
	{hubX - 40 + 16, hubY - 210 + 6, itemLantern, itemLantern},                                    // torre em ruínas da Encruzilhada
	{treeX + treeRiftOffX + 8, deepY0 + treeY + treeRiftOffY - 6, itemAbyssArmor, itemAbyssArmor}, // sob a Árvore Divina
	// Câmaras secretas das Profundezas e o cofre de Dorrim (dorrim.go).
	{desertX + secretDX, deepY0 + desertY + secretDY, itemGoldScarab, itemGoldScarab},
	{snowX + secretDX, deepY0 + snowY + secretDY, itemIceCrown, itemIceCrown},
	{swampX + secretDX, deepY0 + swampY + secretDY, itemMyceliumBoots, itemMyceliumBoots},
	{jungleX + secretDX, deepY0 + jungleY + secretDY, itemRootBow, itemRootBow},
	{dorrimVaultX, dorrimVaultY, itemDorrimHelm, itemDorrimHammer},
	{crystalX, crystalY - 4 - 9, itemCrystalHeart, itemPrismBlade}, // cofre de cristal (cristais.go)
}

const chestReach = 7.0

// isNight vale do pôr ao nascer do sol.
func isNight(tod float64) bool { return tod < 0.25 || tod >= 0.75 }

// openChests abre os baús em que o jogador encostou e ainda não abriu hoje.
func (wd *World) openChests(p *Player, now time.Time) {
	t := now.Sub(wd.start).Seconds()
	day, tod := wd.dayOf(t), wd.timeOfDay(t)
	for i, c := range chests {
		if p.chestDay[i] == day || dist(float64(p.X), float64(p.Y), float64(c.X), float64(c.Y)) > chestReach {
			continue
		}
		if p.chestDay == nil {
			p.chestDay = map[int]int{}
		}
		p.chestDay[i] = day
		p.Chests++
		gold := 30 + wd.rng.Intn(51)
		p.Gold += gold
		p.give(itemPotion, 1)
		text := p.Name + " abriu um baú: " + strconv.Itoa(gold) + " de ouro e uma poção"
		// O item único sai mais fácil quanto menos raro for.
		unique := c.Day
		if isNight(tod) {
			unique = c.Night
		}
		if unique != 0 && p.count(unique) == 0 && wd.rng.Intn(100) < 55-10*items[unique].Rarity && p.give(unique, 1) {
			text += " — e " + items[unique].Name + "!"
		}
		wd.say(ChatLine{Text: text})
		wd.effects = append(wd.effects, Effect{Kind: fxHoly, X: float64(c.X), Y: float64(c.Y) + 10, R: 10, Start: now, Dur: 500 * time.Millisecond})
	}
}

// saveData é o que vai para o disco de cada personagem.
type saveData struct {
	Token      string // hash do segredo que o cliente guarda
	Class      int
	Level      int
	XP         int
	Gold       int
	Points     int
	SkillPts   int
	Attr       [numAttrs]int
	Tree       [numOldTree]int // árvore antiga: só lida, para devolver os pontos
	Inv        []invSlot
	Equip      int
	Skin       int
	Quest      int
	QProg      int
	QStep      int
	Unlocked   bool
	Kills      int
	Chests     int
	Bosses     uint
	Seen       uint
	Ach        uint
	Helm       int
	Boots      int
	Amulet     int
	Steeds     uint
	LastSteed  int
	QuestVer   int // 1 = corrente em capítulos; 0 = save antigo
	SideProg   [numSide]int
	SideDone   uint
	Bonfire    int
	Tokens     []string // segredos de todos os aparelhos que já abriram este personagem
	Password   string   // hash da senha opcional; abre o personagem em qualquer aparelho
	Moored     bool     // barco atracado: onde ficou
	MoorX      int
	MoorY      int
	Fog        string // explorado (explore.go)
	Web        []int  // graus de cada nó da teia de habilidades
	Bar        []int  // golpes nos atalhos 1 a 5 (-1 vazio)
	Slain      uint64 // chefes já derrubados, por tipo
	Bag        int    // espaços comprados na mochila
	Found      uint64 // pontos de viagem descobertos
	Tech       int    // técnicas do Mestre Oren (bits)
	Gates      uint32 // Portões Rúnicos acesos (bits)
	Prologue   *int   // etapa do prólogo; ausente = save de antes do prólogo (concluído)
	SkillVer   int    // 1 = teia de habilidades; 0 = save da árvore antiga
	HasPos     bool   // o lugar onde ele estava (saves antigos não têm)
	PosX       int
	PosY       int
	LightSlot  int       // lanterna ou tocha no espaço de luz
	House      int       // casa comprada + 1
	Stash      []invSlot // baú da casa
	StashGold  int
	FishKinds  uint64 // pesca: peixes pegos
	FishBodies uint32 // pesca: águas
	BestSeen   uint64 // bestiário
	SteedSeen  uint32
	BestKills  []int
	Mode       int // modo de combate
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (wd *World) savePath(name string) string {
	return filepath.Join(wd.saveDir, strings.ToLower(name)+".json")
}

// persist grava o personagem; quem entrou sem segredo (terminal) não é salvo.
func (wd *World) persist(p *Player) {
	if wd.saveDir == "" || p.token == "" {
		return
	}
	b, err := json.MarshalIndent(saveData{p.token, p.Class, p.Level, p.XP, p.Gold, p.Points, p.SkillPts, p.Attr, [numOldTree]int{},
		p.inv, p.Equip, p.Skin, p.Quest, p.QProg, p.QStep, p.Unlocked, p.Kills, p.Chests, p.Bosses, p.Seen, p.Ach, p.Helm, p.Boots, p.Amulet,
		p.Steeds, p.LastSteed, 1, p.SideProg, p.SideDone, p.Bonfire, p.tokens, p.password, p.Moored, p.MoorX, p.MoorY, packFog(p.fog), p.Web, p.Bar[:], p.Slain, p.Bag, p.Found, p.Tech, p.Gates, &p.Intro, 1, true, p.X, p.Y, p.LightSlot, p.House, p.Stash, p.StashGold, p.FishKinds, p.FishBodies, p.BestSeen, p.SteedSeen, p.BestKills, p.Mode}, "", "\t")
	if err != nil || os.MkdirAll(wd.saveDir, 0o755) != nil {
		return
	}
	// Grava ao lado e troca, para uma queda no meio não corromper o save.
	tmp := wd.savePath(p.Name) + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil && os.Rename(tmp, wd.savePath(p.Name)) == nil {
		p.saves++
	}
}

// SaveAll grava todo mundo que está conectado.
func (wd *World) SaveAll() {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	for _, p := range wd.players {
		wd.persist(p)
	}
}

// JoinSaved entra com um personagem salvo. O nome é a identidade; a posse se
// prova pelo segredo de um aparelho que já abriu o personagem ou pela senha,
// quando ele tem uma. Entrar com a senha libera também este aparelho. Num
// personagem que já existe, vale a classe salva, não a pedida.
func (wd *World) JoinSaved(name string, class int, token, password string) (int, error) {
	if wd.saveDir == "" || token == "" {
		return wd.Join(name, class), nil
	}
	wd.mu.Lock()
	defer wd.mu.Unlock()
	name = sanitizeName(name)
	if wd.nameTaken(name) {
		return 0, errors.New("esse personagem já está conectado")
	}
	var sv saveData
	b, err := os.ReadFile(wd.savePath(name))
	found := err == nil
	tok := hashToken(token)
	if found {
		if json.Unmarshal(b, &sv) != nil {
			return 0, errors.New("o save desse personagem está corrompido")
		}
		pwOK := sv.Password != "" && password != "" && sv.Password == hashPw(name, password)
		if !sv.owns(tok) && !pwOK {
			if sv.Password != "" {
				return 0, errors.New("senha errada para esse personagem")
			}
			return 0, errors.New("esse nome já pertence a outro jogador (abra-o no aparelho dele e defina uma senha)")
		}
		class = sv.Class
	}
	p := wd.spawn(name, class)
	p.token = tok
	if found {
		p.apply(sv)
		p.tokens, p.password = sv.allTokens(), sv.Password
		p.fog = wd.unpackFog(sv.Fog)
		if p.inPrologue() { // ainda no prólogo: volta ao vale
			p.X, p.Y = wd.introSpawn()
		} else {
			wd.savedPos(p, sv) // volta onde estava, se ainda couber lá
		}
	} else {
		// Personagem novo: acorda no Vale do Despertar, no começo do prólogo.
		p.Intro = 0
		p.X, p.Y = wd.introSpawn()
		p.Bonfire = len(wd.bonfires) - 1
	}
	if !has1(p.tokens, tok) {
		p.tokens = append(p.tokens, tok)
	}
	// Quem é dono e ainda não tem senha pode criá-la agora.
	if p.password == "" && password != "" {
		p.password = hashPw(name, password)
	}
	wd.persist(p)
	return p.ID, nil
}

func hashPw(name, pw string) string { return hashToken("asciimmo:" + strings.ToLower(name) + ":" + pw) }

// allTokens junta o segredo antigo (um só) com a lista nova.
func (sv saveData) allTokens() []string {
	out := append([]string(nil), sv.Tokens...)
	if sv.Token != "" && !has1(out, sv.Token) {
		out = append(out, sv.Token)
	}
	return out
}

func (sv saveData) owns(tok string) bool { return has1(sv.allTokens(), tok) }

func has1(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// CharInfo é um personagem na lista da tela de entrada.
type CharInfo struct {
	Name     string `json:"name"`
	Class    int    `json:"cls"`
	Level    int    `json:"lv"`
	Chapter  int    `json:"ch"`
	Password bool   `json:"pw"`
	Online   bool   `json:"on"`
}

// Chars lista os personagens que este aparelho já abriu.
func (wd *World) Chars(token string) []CharInfo {
	if wd.saveDir == "" || token == "" {
		return nil
	}
	tok := hashToken(token)
	files, _ := filepath.Glob(filepath.Join(wd.saveDir, "*.json"))
	var out []CharInfo
	for _, f := range files {
		b, err := os.ReadFile(f)
		var sv saveData
		if err != nil || json.Unmarshal(b, &sv) != nil || !sv.owns(tok) {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		ch := 1
		if q := min(max(0, sv.Quest), len(quests)); sv.QuestVer == 1 && q < len(quests) {
			ch = quests[q].Chapter
		} else if q >= len(quests) {
			ch = len(chapters) + 1
		}
		wd.mu.RLock()
		online := wd.nameTaken(name)
		wd.mu.RUnlock()
		out = append(out, CharInfo{Name: name, Class: sv.Class, Level: max(1, sv.Level), Chapter: ch, Password: sv.Password != "", Online: online})
	}
	return out
}

// DeleteChar apaga um personagem deste aparelho. O arquivo não some: vai para
// a lixeira dentro da pasta de saves, com a data, para dar para recuperar.
func (wd *World) DeleteChar(token, name string) error {
	if wd.saveDir == "" || token == "" {
		return errors.New("este servidor não salva personagens")
	}
	name = sanitizeName(name)
	wd.mu.Lock()
	defer wd.mu.Unlock()
	if wd.nameTaken(name) {
		return errors.New("esse personagem está conectado agora")
	}
	b, err := os.ReadFile(wd.savePath(name))
	var sv saveData
	if err != nil || json.Unmarshal(b, &sv) != nil {
		return errors.New("personagem não encontrado")
	}
	if !sv.owns(hashToken(token)) {
		return errors.New("esse personagem não é deste aparelho")
	}
	trash := filepath.Join(wd.saveDir, "lixeira")
	if err := os.MkdirAll(trash, 0o755); err != nil {
		return err
	}
	return os.Rename(wd.savePath(name), filepath.Join(trash, strings.ToLower(name)+"-"+time.Now().Format("20060102-150405")+".json"))
}

// apply carrega um save no jogador, descartando o que não fizer sentido
// (arquivo editado à mão ou de versão antiga).
func (p *Player) apply(sv saveData) {
	if sv.House > 0 && sv.House <= len(lots) {
		p.House = sv.House
	}
	for _, v := range sv.Stash {
		if v.Item >= 0 && v.Item < len(items) && v.N > 0 {
			p.Stash = append(p.Stash, v)
		}
	}
	p.StashGold = max(0, sv.StashGold)
	p.FishKinds, p.FishBodies, p.BestSeen, p.SteedSeen = sv.FishKinds, sv.FishBodies, sv.BestSeen, sv.SteedSeen
	p.BestKills = append([]int(nil), sv.BestKills...)
	p.Mode = clampInt(sv.Mode, 0, numModes-1)
	if sv.LightSlot == itemLantern || sv.LightSlot == itemTorch {
		p.LightSlot = sv.LightSlot
	}
	p.Level, p.XP, p.Gold = max(1, sv.Level), max(0, sv.XP), max(0, sv.Gold)
	p.Points, p.SkillPts = max(0, sv.Points), max(0, sv.SkillPts)
	p.Quest, p.QProg, p.Unlocked = min(max(0, sv.Quest), len(quests)), max(0, sv.QProg), sv.Unlocked
	if p.Quest < len(quests) {
		p.QStep = min(max(0, sv.QStep), len(stepsOf(quests[p.Quest]))-1)
		if sv.QStep == 0 && sv.QProg > 0 { // save de antes das etapas: o progresso era da etapa do mesmo tipo
			for i, s := range stepsOf(quests[p.Quest]) {
				if s.Kind == quests[p.Quest].Kind {
					p.QStep = i
				}
			}
		}
	}
	p.SideProg, p.SideDone, p.Bonfire = sv.SideProg, sv.SideDone&(1<<numSide-1), sv.Bonfire
	if sv.QuestVer == 0 && sv.Quest > 0 {
		// Save de antes dos capítulos: traz a missão para a corrente nova.
		p.Quest, p.SideDone = oldQuest(sv.Quest)
		p.QProg = 0
	}
	if sv.QuestVer == 0 {
		p.Bonfire = -1
	}
	p.Kills, p.Chests, p.Bosses, p.Seen, p.Ach, p.Slain = sv.Kills, sv.Chests, sv.Bosses, sv.Seen, sv.Ach, sv.Slain
	p.Moored, p.MoorX, p.MoorY = sv.Moored, sv.MoorX, sv.MoorY
	p.Bag = max(0, min(sv.Bag, maxBag-baseBag))
	p.Found = sv.Found
	p.Tech = sv.Tech & (1<<len(techs) - 1)
	p.Gates = sv.Gates
	p.Intro = len(prologue) // save de antes do prólogo: já passou dele
	if sv.Prologue != nil {
		p.Intro = min(max(0, *sv.Prologue), len(prologue))
	}
	p.Steeds, p.LastSteed = sv.Steeds&(1<<len(steedKinds)-1), min(max(0, sv.LastSteed), len(steedKinds)-1)
	for i := range p.Attr {
		p.Attr[i] = max(0, sv.Attr[i])
	}
	if sv.SkillVer == 0 {
		// Save da árvore antiga: os pontos gastos voltam para a teia nova, e
		// quem já tinha descido à cripta ganha o golpe dela de graça.
		for _, r := range sv.Tree {
			p.SkillPts += max(0, r)
		}
		p.Web = make([]int, len(p.web()))
		if p.Unlocked {
			for i, n := range p.web() {
				if n.Crypt {
					p.Web[i] = 1
				}
			}
			p.Bar, p.bar0 = emptyBar(1, 2), true
		}
	} else {
		p.Web = append([]int(nil), sv.Web...)
		p.bar0 = len(sv.Bar) > 0 // barra de 5 de antes também vale
		for i := range p.Bar {
			p.Bar[i] = -1
			if i < len(sv.Bar) {
				p.Bar[i] = sv.Bar[i]
			}
		}
	}
	if !p.bar0 {
		p.Bar = [barSlots]int{}
	}
	p.inv = nil
	for _, s := range sv.Inv {
		if s.Item >= 0 && s.Item < len(items) && s.N > 0 && len(p.inv) < baseBag+max(0, min(sv.Bag, maxBag-baseBag)) {
			p.inv = append(p.inv, s)
		}
	}
	// Só volta equipado o que está na mochila e é do tipo do espaço.
	p.Equip, p.Skin, p.Helm, p.Boots, p.Amulet = 0, 0, 0, 0, 0
	for _, it := range []int{sv.Equip, sv.Skin, sv.Helm, sv.Boots, sv.Amulet} {
		if it > 0 && it < len(items) && p.count(it) > 0 {
			if s := p.slot(items[it].Kind); s != nil {
				*s = it
			}
		}
	}
	p.recalc()
	p.HP, p.MP, p.ST = p.MaxHP, p.MaxMP, float64(p.MaxST)
}
