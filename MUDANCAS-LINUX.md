# Mudanças feitas na versão Linux (14–15/10/2026)

Lista de tudo o que foi corrigido, melhorado e implementado na versão Linux
(`GFiamoncini/AsciiMMO`), para aplicar na versão Windows (`GFBmsoft/AsciiMMO`).
O código é o mesmo Go nas duas: quase tudo pode ser copiado arquivo a arquivo.
Os itens marcados **[só Linux]** não precisam ir para o Windows.

Convenções: `S/` = `servidor/`, `C/` = `cliente/`, `P/` = `proto/`.
"Novo" = arquivo que não existe na versão Windows.

---

## 0. Como portar (ordem sugerida)

1. **Protocolo primeiro** (`P/proto.go`, `P/kinds.go`): os campos novos são todos
   opcionais (`omitempty`), então servidor e cliente antigos continuam se entendendo.
2. **Servidor** (`S/`): arquivos novos + os trechos alterados abaixo.
3. **Cliente nativo** (`C/`): arquivos novos + trechos alterados.
4. `go.mod`/`go.sum`: o som usa `github.com/hajimehoshi/ebiten/v2/audio` e
   `audio/vorbis` (traz `ebitengine/oto/v3`, `jfreymuth/oggvorbis`). Rodar
   `go get github.com/hajimehoshi/ebiten/v2/audio@v2.10.5 github.com/hajimehoshi/ebiten/v2/audio/vorbis@v2.10.5`.
5. Pasta `C/sons/musica/` (6 arquivos .ogg + `CREDITOS.txt`) — vai embutida no executável.
6. Rodar `go vet ./...` e `go test ./...` (a suíte agora leva ~10 min por causa do mapa maior).

**Atenção — o mapa mudou de escala (item 9):** saves antigos continuam valendo
(nível, itens, missões), mas o mapa explorado (névoa) recomeça e a posição salva
antiga é ignorada. O cliente baixa ~37 MB de mapa ao entrar (era ~14 MB).

Sobre compilar: no Linux foi preciso `GOTOOLCHAIN=go1.27.0+auto` e
`-buildvcs=false` (disco NTFS); no Windows o `build.ps1` continua servindo.

---

## 1. Correções de bugs (aplicar mesmo que não queira as novidades)

| Bug | Causa | Correção | Onde |
|---|---|---|---|
| **Norte do mapa preto** (cemitério, castelo, montanhas, minimapa) no cliente nativo | `fetchMap` usava fatias do buffer do download; o `append` que reservava as Profundezas escrevia zeros por cima das cores das ~800 primeiras linhas | Copiar cada camada (letras, cores, flags) para fatias próprias do tamanho do mundo | `C/net.go` (`fetchMap`); teste `C/mapa_test.go` (novo) |
| **Tocha nunca apagava** | `stepLight` só era chamado nos testes | Chamar `wd.stepLight(p, now)` no laço de 1×/s dos jogadores | `S/world.go` |
| **Inimigos presos no cenário** (chefe da cripta, Apóstolo Faminto) | Andavam reto e escorregavam no eixo livre; empacavam em quina/U; bichos grandes testavam colisão de 1 célula; vagavam para pontos dentro da parede | Caminho A* quando a reta está bloqueada (`mobChase`), tamanho no chão por tipo (`mobRad`, `mobFits`, `mobLine`), vagar só para ponto alcançável (`wanderPoint`), destravar de dentro da parede (`unstick`, 1×/s) | `S/caminho.go` (novo), `S/world.go` (`stepMobs`, `mobToward`, `Mob.nav`, `World.pathBudget`); teste `S/caminho_test.go` (novo) |
| **Personagem preso nas fendas (nível 15)** | Galerias naturais das Profundezas com 1–3 células; o personagem ocupa 5 de lado (`fits`) | `widenGalleries` alarga a grade antes dos túneis | `S/depths.go`; `S/profundezas_test.go` agora busca com `fits` |
| **Chegar a Tiamath sem fazer a história** | O dragão descia do Pico atrás de quem chegasse perto | Véu de Cinzas (item 6) | `S/veu.go` (novo) |
| **Personagem novo renascia na fogueira errada** | `p.Bonfire = len(wd.bonfires)-1` ("a última"), mas a do Despertar deixou de ser a última | `introBonfire()` procura pelo nome | `S/intro.go`, `S/progress.go`, `S/questmap.go` |
| **Sombra "aplicada sobre tudo"** | A elipse de sombra de cada personagem escurecia parede/telhado/copa; sombra curta e longa somavam; a longa saía pontilhada; nuvem passava por cima de telhado | Sombra do personagem só no chão; curta e longa numa máscara só (`g.shadeF`, `applyShade`); longa contínua; nuvem só no chão e mais fraca | `C/frame.go`, `C/depth.go` (`castShadows`, `applyShade`) |
| **Luz verde estranha no subsolo** | Sem tocha, o jogador levava uma luz verde-amarelada de raio 38 | Luz fria e fraca de raio 20 (a luz de verdade vem da tocha/lanterna/fogueira) | `C/frame.go` |
| **Item 0 sumia do baú** | Filtro de id `> 0` (o 0 é a Poção de Vida) | `>= 0` | `S/progress.go` |
| **Barraca do Eremita bloqueava o baú da ilha** | Poste sólido | Nada da barraca é sólido; não pinta em cima da água | `S/lojas.go` |

---

## 2. Interface

- **Tela reorganizada** — `C/hud_layout.go` (novo), `C/hud.go`:
  minimapa à esquerda sob o card (`drawMiniLeft`); outros jogadores em fichas
  compactas (`drawPartyChips`); coluna da direita só com missão e menu.
- **Bússola** — `C/compass.go` (novo): faixa no topo central que gira com o rumo
  (média do andar, sem tremer), pontos de viagem descobertos, missão (com a seta
  ligada) e destino; no subsolo só treme no Abismo/Ventre de Fogo. Embaixo dela,
  **ícones de estação, clima e hora com dica** + hora e lugar (`drawWorldInfo`).
  Opção "Bússola no alto da tela" no O.
- **Card do personagem** — `C/itemicons.go` (`drawStatusCard`): moldura dourada
  inteira com cantos `+`; **barra de XP** com "N / M · faltam X" e brilho ao ganhar.
- **Ícone do modo de combate** ao lado do card (item 7).
- **Seta da missão** só com a seta (sem o texto por cima) — `C/minimapa.go`.
- **Chat**: ↑/↓ voltam pelo que já foi dito; registro (L) rola com ↑/↓, roda,
  PgUp/PgDn, End volta ao fim — `C/game.go`, `C/hud.go`, `C/life.go`.
- **Menu de pausa** (Esc com nada aberto, ou P) — `C/pausa.go` (novo): Continuar,
  Salvar agora (F5), Opções, Controles, Voltar à tela de entrada, **Sair do jogo**.
- **Tela de entrada no padrão das lojas** — `C/entrada.go`: cena do vale animada
  ao fundo, moldura com cantos, abas "Seus personagens" (1–9 carregam) e "Novo
  personagem" (classes com retrato + painel de detalhes `classLore`), botão
  **Sair do jogo**.
- **Mapa grande** — `C/mapa_zoom.go` (novo): zoom (roda, + e −, até 8x no
  cursor), arrastar com o botão direito, 0 volta; ao abrir fecha os painéis e
  escurece o fundo (não fica mais por cima de outros menus); clicar = caminhar até
  lá (item 7).
- **Diário (N)** — `C/diario.go` (novo): abas Missões (principais e secundárias),
  História (rever a abertura e os cards dos capítulos), Bestiário e Pesca.
- **Ajuda (H)**: teclas novas (Z luz, G pescar, B modo, N diário, P/Esc pausa, F5 salvar).
- **Opções (O)** — `C/story.go`: volumes de música/efeitos/ambiente, calar sem
  foco, pausar ao abrir menus, bússola, luz suave (GPU). O painel cresce conforme
  o número de linhas.

## 3. Som (tudo novo)

- `C/som.go`, `C/som_sintese.go`, `C/som_eventos.go`, `C/voz.go` (novos);
  `C/sons/musica/*.ogg` (dia, vila, noite, cripta, fundo, chefe — CC0 do
  OpenGameArt, créditos em `CREDITOS.txt`).
- Efeitos e ambiente **gerados por código** na abertura (golpes, magias, quebras de
  madeira/pedra/cristal, árvore caindo, passos por chão, interface; vento, grilos,
  chuva, mar, caverna, lava, fogueira; pássaros, coruja, sapos, pingos).
- **Voz própria de cada inimigo** (`mobVoice`, na ordem dos tipos do servidor) com
  vozes de formante (`formantVoice`/`genVoices`): orc, troll, gigante, yeti,
  esqueleto, aranha, escorpião, escaravelho, asura, tritão, golem, apóstolo, anão,
  múmia, espectro, raízes, arconte, corvo, harpia, bruxa…
- **Música por lugar e hora** com fade; trovão, rugido de Tiamath, chamados dos bichos.
- **Voz dos NPCs e mercadores**: fala surgindo letra a letra e cada palavra dita
  na "língua velha" no tom de quem fala (`voicePitch`).
- Opções de volume; `-sem-som`; prints de teste ficam mudos. Teste `C/som_test.go` (novo).
- O servidor não manda nada a mais para o som (tudo sai da diferença entre estados).

## 4. Visual

- **Bonecos mais legíveis** — `C/body.go`: rampa curta de letras densas
  (`bodyRamp`), membros mais grossos, ombros largos, cabeça/mãos/botas maiores.
- **Raio do mago dourado** (milagre); tempestade segue branca — `C/fx.go`
  (`drawBoltC`), `C/fx_habilidades.go`.
- **Luz suave (GPU)**: o shader interpola o brilho entre células — `C/gfx.go`
  (`Smooth`, `drawS`), `C/frame.go`.
- **Saídas para a superfície desenhadas** (arco, degraus, raios de luz de dia ou
  luar) — `C/fx.go` (`drawExit`), `P/proto.go` (`Portal.Up`), `S/web.go`.
- **História animada** — `C/historia.go` (novo): arte ASCII animada letra a letra
  (`drawArt`); **card de cada capítulo** com arte própria (`chapterArt`), título e
  narração surgindo.
- **Abertura no estilo do jogo** — `C/abertura_cena.go` (novo): cenas
  procedurais na grade do jogo (Cinzalva ardendo, Tiamath com relâmpagos, o vale)
  desenhadas por uma segunda instância do shader com atlas próprio; narração na
  língua velha sincronizada com a legenda; música e ambiente. `C/opening.go`.
- **Retratos no bestiário** — `C/retrato_bicho.go` (novo).
- **Véu de Cinzas e regiões corrompidas** — `C/veu.go` (novo).

## 5. Lojas, casa, luz, pesca

- **Lojas como conversa** — `C/loja.go` (novo; o `drawShop` antigo saiu de
  `C/story.go`): fala do mercador, 1 Comprar / 2 Vender / 3 Conversar / 4 Até
  logo, lista com detalhes, comparação com a arma equipada, quantidades.
  **Barracas/cabana** para os mercadores de fora da vila — `S/lojas.go` (novo).
- **Casa com baú** — `S/casa.go`, `C/casa.go` (novos): Casa do Poço (1200) e Casa
  da Praça (2000) na vila; baú de itens e dinheiro no save. Teste `S/casa_test.go`.
- **Espaço de luz** — `S/luz.go` (novo): lanterna/tocha equipada, tecla **Z**,
  acende sozinha ao descer; espaço "luz (Z)" na ficha (`C/ficha.go`).
- **Pesca** — `S/pesca.go`, `C/pesca.go` (novos): Vara de Pesca, tecla **G**, 4
  lagoas novas, 7 águas, 18 peixes por água/estação/hora, venda, conquistas
  "Todas as Águas" e "Mestre Pescador". `P/kinds.go` ganhou `KindFish`;
  `P/proto.go` ganhou `Item.Price`. Teste `S/pesca_test.go`.

## 6. Mundo e história

- **Véu de Cinzas** — `S/veu.go` (novo): o Pico de Tiamath só abre com os cinco
  guardiões caídos (Lorde, Rainha Asura, Tirano, Faraó, Gigante de Gelo); Tiamath
  não sai do Pico, não persegue e não leva dano de quem não derrubou o Véu; cada
  guardião purifica a região (estado `pure`). Teste `S/veu_test.go`; os testes que
  batem em Tiamath usam `veilDown`.
- **Fogueira com mais opções** — `S/espera.go` (novo), `C/fogueira.go` (novo):
  Descansar aqui, Viagem rápida, Descansar e sair; **esperar** por ícones até
  amanhecer/meio-dia/anoitecer/meia-noite ou o clima virar (vale para todos).
- **Doma com fôlego e sequência** — `S/doma.go`, `C/doma.go` (novos).
- **Bestiário** — `S/bestiario.go` (novo): verbetes, fraquezas e resistências de
  cada criatura (fraqueza = 150% do dano, resistência = 60%), registro do que viu,
  matou e domou. Teste `S/pesca_test.go` (`TestBestiario`).
- **Vozes, inimigos novos e biomas novos**: itens 3 e 9.

## 7. Jogo

- **Pausa** — `S/pausa.go`, `C/pausa.go` (novos): sozinho, o mundo para (relógio
  do mundo `wd.clock()` substituiu `time.Now()` nas funções do servidor); com
  outros jogando, você fica parado e protegido. Teste `S/pausa_test.go`.
- **Salvar**: posição no save, F5/menu salvam na hora, descansar grava, aviso
  "[=] salvo". Teste `TestSalvarPosicao`.
- **Modo de combate (B)** — `S/modo.go` (novo): pacifista / defensivo / agressivo.
- **Caminhar até um ponto do mapa** (clique no mapa grande) — `S/modo.go`
  (`coarsePath`, `driveGoal`). Teste `TestModoECaminho`.

## 8. Ferramentas

- **`servidor.sh`** **[só Linux]**: iniciar, parar (SIGTERM, grava todos),
  reiniciar, estado, log, compilar. No Windows, o equivalente seria um `.ps1` (o
  servidor já trata `SIGTERM` e Ctrl+C — `S/main.go`).
- **Flags do cliente**: `-monitor N` (abrir num monitor), `-sem-som`, `-painel`
  novos (`loja`, `loja-compra`, `pausa`, `abertura0..2`, `diario-quests|story|beasts|fish`).

## 9. Mapa em escala 3 e biomas novos

- `S/lugares.go`: `mapScale = 3` (era 2) e `bsF = mapScale/2`, que faz os biomas
  crescerem junto (deserto, gelo, pântano, selva, Talássia, montanhas, lago,
  Cavernas de Cristal, Charco Venenoso) com as regiões deles. Construções mantêm o
  tamanho. Posições que eram absolutas passaram a escalar (ninho de Tiamath, um
  trecho de trilha, dois marcos em `S/kit.go`); a Cidadela fica na mesma distância
  da Selva; Grael nasce no chão livre mais perto (`S/divina.go`).
- **Biomas novos** — `S/biomas_novos.go` (novo): Brejo das Garças, Mangue de Sal,
  Floresta Cinzenta, Planalto dos Ventos, Charneca Roxa; cada um com chão, mata,
  água, fogueira, trilha e dois inimigos. Ganchos em `S/terrain.go` (`newBiomeWater`,
  `newBiomeGround`, `forestMod`, `newBiomeLeaf`, `buildNewBiomes`).
- **10 inimigos novos** no fim dos tipos (`S/rpg.go`): Sapo-Boi, Sanguessuga
  Gigante, Caranguejo de Lodo, Crocodilo do Mangue, Corvo Carniceiro, Lobisomem,
  Harpia, Centauro Errante, Mariposa Gigante, Bruxa da Charneca — com nível
  (`S/levels.go`), recompensa (`S/story.go`), veneno (`S/nature.go`), bestiário,
  aparência (`C/body.go`, `mobStyles`) e voz (`C/som_eventos.go`). Teste
  `S/biomas_test.go`.
- **Névoa por largura do mapa**: o save guarda `FogW` (outra largura = descarta);
  o arquivo local passou a ser `nevoa_<nome>_<largura>.bin`.

## 10. Protocolo e save (resumo para conferir)

**`P/proto.go` — State** (todos opcionais): `pz`, `frz`, `sv`, `lslot`, `tame`,
`house`, `stash`, `stashg`, `mode`, `pure`, `goal`, `fish`, `fishk`, `fishb`,
`bseen`, `sseen`, `bkills`. **Player**: `pz`. **Init**: `quests`, `houses`,
`beasts`, `steedlore`, `waters`, `fishes`, `fish0`, `modes`. **Item**: `price`.
**Portal**: `up`. Tipos novos: `Tame`, `HouseLot`, `Beast`, `FishLine`.
`P/kinds.go`: `KindFish`.

**Mensagens novas (cliente → servidor)**: `pause`, `save`, `light`, `tame`,
`wait`, `waitwx`, `house`, `stash`, `mode`, `goto`, `nogoto`, `fish`.

**Save (`saveData`, sempre no fim)**: `HasPos`, `PosX`, `PosY`, `LightSlot`,
`House`, `Stash`, `StashGold`, `FishKinds`, `FishBodies`, `BestSeen`, `SteedSeen`,
`BestKills`, `Mode`, `FogW`.

## 11. Testes novos

`C/mapa_test.go`, `C/som_test.go`, `S/caminho_test.go`, `S/pausa_test.go`,
`S/casa_test.go`, `S/pesca_test.go` (pesca, bestiário, modo e caminho),
`S/veu_test.go`, `S/biomas_test.go`; `S/mundo_test.go` (doma) e
`S/profundezas_test.go` (busca com `fits`) mudaram; `S/combate_test.go` usa `veilDown`.
