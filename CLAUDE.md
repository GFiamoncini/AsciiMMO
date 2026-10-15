# AsciiMMO — memória do projeto

MMORPG em Go onde o mundo inteiro é desenhado com caracteres coloridos (cada célula é um "pixel" de pixel art).
Começou em 13/10/2026 como protótipo por SSH em `D:\Projetos\AsciiMMO`; em 14/10/2026 o usuário moveu o projeto
para `D:\Projetos\Private\AsciiMMO`. Repositório git no GitHub: https://github.com/GFBmsoft/AsciiMMO (só a branch
`main`).

## Estado da sessão (14/10/2026, noite, no Linux)

- **Agora ele roda no Linux** (Fedora, projeto em `/mnt/3C0CEC910CEC478A/Projetos/GO/AsciiMMO`, disco NTFS). Para
  compilar: `GOTOOLCHAIN=go1.27.0+auto` (o Go do sistema é 1.26.9 travado em `local`) e `-buildvcs=false` (o git
  recusa o repositório por "propriedade duvidosa" no NTFS; nos comandos git usar `git -c safe.directory='*'`).
  Binários `asciimmo` e `asciimmo-cliente` na raiz (ainda fora do `.gitignore`, que só pega `*.exe`).
- **Prints de teste no monitor 1** (`-monitor 1` = HDMI-4); ele joga no 2 (eDP-1). A janela de teste rouba o foco:
  tecla que ele digita cai nela (já abriu mapa/ficha/opções nos prints) — repetir o print se sair com painel aberto.
  `pkill -f` com o nome do binário no padrão mata o próprio shell: matar por `pgrep -x nome`.
- **Segunda lista da noite** (inimigos presos, lojas, mapa preto, pausa, salvar, fenda estreita): feita, ver
  "Correções, pausa, salvar e lojas".
- **Rodada da noite (lista colada por ele, tudo feito)**: bússola, ↑/↓ no chat e no registro, moldura no card, sombras
  numa máscara só, sistema de som, bonecos mais legíveis, cards da história animados. Detalhes em "Som, bússola e
  história animada" mais abaixo. Commits locais; ainda **não enviados** ao GitHub (esperar o "pode commitar").
- **Próxima sessão começa pela faixa dos Planos** (ver "Fila (próximas rodadas)"): estimativa de ~50 min com testes
  (mapa em três partes no servidor e no cliente, Ilha das Fadas, Terra do Eclipse com evento de eclipse). Depois Costa
  Vulcânica e Cidade Afundada.
- **Jeito de trabalhar combinado**:
  - "pode commitar" depois de um commit já feito quer dizer **enviar ao GitHub** (`git push`).
  - Quando ele dá um horário limite ("tempo limite 17:55"), parar antes e deixar tudo commitado. Calcular o tempo e
    dizer com franqueza se não cabe, em vez de começar algo que vai ficar pela metade.
  - Prints das telas novas vão para `previas/` (que está no repositório).
  - Para conferir a barra de título ou qualquer coisa da janela, capturar **só a janela do jogo** (PrintWindow pelo
    identificador), nunca a tela inteira.
- **Cliente web congelado**: o cliente principal hoje é o nativo (`cliente/`); o do navegador não recebe as telas novas.
- **Regra dele: tudo que precisa ser lido abre com E.** `servidor/leituras.go` + `cliente/leituras.go`: `signText` e
  `signpost` registram sozinhos uma leitura (`wd.addRead`), assim como as pedras rúnicas ("Pedra rúnica III": seta e
  direção por extenso) e a inscrição da porta da palavra; vão no Init como `reads`. A dica "[E] ler: …" tem a menor
  prioridade entre as ações de E (descansar, domar, embarcar etc. vêm antes). Texto novo no mapa: usar `signText`,
  `signpost` ou `addRead`, nunca só células soltas (uma célula é pequena demais para ler). `-painel read`. TestLeituras.

## Como rodar

```powershell
.\build.ps1                    # compila servidor e cliente com o ícone embutido
.\asciimmo.exe                 # SSH em :2222, cliente web em :8080, saves em Documentos\AsciiMMO\save
.\asciimmo-cliente.exe         # cliente nativo
```

Flags: `-addr` (SSH), `-http` (web, vazio desliga), `-key` (chave do host), `-seed`, `-day` (duração do dia),
`-saves` (pasta; vazio = pasta de dados, `-` desliga). Testes: `go test ./...` (sobe servidor SSH e web de verdade).

**Estrutura**: `servidor/` (pacote main do servidor, com `web/index.html` embutido), `cliente/` (cliente nativo),
`proto/kinds.go` (tipos de item, efeito e projétil compartilhados: nunca usar números soltos no cliente),
`proto/` (mensagens e caminhos compartilhados), `previas/`, `EVs/` (referências do usuário). Repositório git desde
14/10/2026 (um commit por rodada; `.gitattributes` fixa LF). Os nomes de arquivo citados abaixo sem pasta são do
`servidor/`.

## Arquivos

- `world.go` — estado autoritativo, tick (20/s), jogadores, chefe dragão, inimigos, snapshot.
- `rpg.go` — tipos de inimigo, itens, saque, missões, hora do dia.
- `combat.go` — classes, golpes, projéteis, formas de acerto, portais.
- `progress.go` — atributos, árvore, conquistas, baús, saves em JSON (segredo do navegador como prova de posse).
- `terrain.go` — geração do mapa (continente + subsolo abaixo de `overH`), biomas, construções.
- `web.go` + `web/index.html` — cliente principal: WebGL desenhando a grade de caracteres, HUD em canvas 2D, WebSocket.
- `ui.go` + `render.go` — modo SSH (bubbletea/wish), simples; não recebe as telas novas.
- `proto/` — mensagens do protocolo (init, estado, entrada), compartilhadas por servidor e cliente nativo; `web.go` usa
  apelidos (`wsState = proto.State`...). Mudou o protocolo? Mexer aqui e nos dois clientes.
- `cliente/` — **cliente nativo** (Go + Ebitengine), começado em 14/10/2026 a pedido dele. Porta 1:1 do `web/index.html`:
  `net.go` (mapa, WebSocket, segredo em `%AppData%\AsciiMMO\token`), `gfx.go` (atlas de caracteres + shader Kage em modo
  pixels desenhando a grade na GPU), `frame.go` (composição do quadro), `body.go` (corpos e bichos), `fx.go` (efeitos,
  clima, água), `hud.go` (HUD, minimapa com névoa, ficha, tela de entrada). Compilar: `go build -o asciimmo-cliente.exe ./cliente`.
  Rodar: `asciimmo-cliente.exe -servidor http://IP:8080` (opções `-nome`, `-classe`, `-zoom`). Para conferir o visual sem
  ele: `-nome X -print arquivo.png -print-apos 6s` salva a tela e fecha (abre uma janela rápida na tela dele).
  Diferenças do web: sem comando de voz; fonte da grade é a Go Mono (embutida). O cliente web continua funcionando;
  enquanto os dois existirem, mudança visual tem de ir nos dois.
- Outras extensões ficam documentadas no fim deste arquivo, em "Sistemas".

## Direção decidida pelo usuário

- **Referência visual:** canal do Senje, "Making the ASCII MMORPG of my dreams" (https://www.youtube.com/@Ansenje/shorts;
  vídeo base https://www.youtube.com/watch?v=GbBCprld6HI; luta do dragão aparece no react https://www.youtube.com/watch?v=7pzCOdRNiqs).
  Estilo: grade quadrada e densa, paleta escura, capim em lâminas que balançam, copas densas de letras (Q0O8&@),
  dragão ferrugem com sombra preta, arcos de golpe, barra do chefe atravessando o topo, nomes com barrinha de vida.
  Dos shorts (vistos em 14/10): masmorra/caverna no escuro total com luz só em volta do jogador (verde-amarelada);
  água escura com ondulação em arcos e reflexo; teias/"fantasmas" de linhas brancas; chuva e fogo.
  O jogo dele é cena gráfica com filtro de caracteres: curvas lisas e luz suave só se aproximam numa grade.
  Imitar o estilo, nunca copiar a arte dele.
- **Cliente principal:** o do navegador (WebGL em `web/index.html`). O SSH ficou como modo simples.
- **Para depois (pedido em 14/10):** sair do navegador — um cliente nativo que rode o jogo e o servidor só conectando
  os jogadores. Ainda não decidido stack; propor quando ele pedir.
- **Mapa, biomas e inimigos:** inspiração no Tibia (temas e nomes), desenho do mapa próprio.
- **Menus:** janelas "como qualquer jogo 2D" — a de personagem tem os espaços de equipamento em volta do boneco.
  Lista de texto na lateral foi rejeitada ("não ficou como pedi").
- **Jeito de pedir:** ele cola listas de funcionalidades, às vezes sem texto próprio; espera tudo implementado na mesma
  rodada. Não perguntar de novo o que já está decidido aqui.

## Como conferir o visual (sem o usuário)

Chrome em `C:\Program Files\Google\Chrome\Application\chrome.exe`, sem janela. Há Node 24 para `node --check` no JS
extraído do `web/index.html`. Não há yt-dlp/ffmpeg instalados: quando precisei, baixei versões portáteis para a pasta
temporária da sessão (yt-dlp.exe do GitHub, ffmpeg-release-essentials do gyan.dev).

- Subir o servidor em portas alternativas: `asciimmo.exe -saves "" -addr 127.0.0.1:9xxx -http 127.0.0.1:8xxx -key <tmp>/testkey`.
- `chrome.exe --headless=new --no-first-run --disable-extensions --user-data-dir=<pasta NOVA a cada execução> --enable-unsafe-swiftshader --window-size=1280,720 --virtual-time-budget=4000 --screenshot=<png> "http://127.0.0.1:8xxx/?name=X&cls=N&zoom=5"`.
- Reusar o mesmo `--user-data-dir` faz o Chrome sair sem print; às vezes falha mesmo assim — repetir até o arquivo existir.
- Parâmetros de URL do cliente: `zoom=N`, `cls=0..6`, `panel=sheet|tree|ach`, `map`.
- FPS aparece 0 nesses prints (tempo virtual): não serve para medir fluidez.
- Para fotografar um lugar/estado, compilar binário temporário FORA do projeto com um `zz_debug.go` (teleporta, dá itens,
  fixa a hora) e apagar o arquivo depois. Matar só o próprio processo (por porta ou `dbg.exe`), nunca todo `chrome.exe`/`asciimmo.exe`.
- Depois de mudar visual ou interface, tirar ao menos um print e olhar antes de dizer que ficou pronto.

## História, economia e personagens (14/10/2026, terceira rodada)

- **História** (`story.go`): 6 capítulos (`chapters`), 14 missões principais (`quests`, com `Chapter`) e 5 secundárias
  (`sideQuests`) que quebram os selos das regiões: Selva+Cidadela, Talássia, Deserto, Picos de Gelo e Pico de Tiamath.
  `canEnter` barra quem está fora de região selada (andar, barco e investida). Saves antigos convertem por `oldQuest`.
- **Tiamath** (o dragão) mudou para as montanhas a noroeste (`lairX/lairY` = 560,430, `mountDist`, elevação extra, chão
  de cinza e veios de brasa). O antigo covil virou a **Encruzilhada Queimada** (`hubX/hubY`), onde os caminhos se cruzam.
  HP 9000.
- **XP e dinheiro**: tabela `mobReward` por inimigo; XP vai para quem está perto (texto flutuante "+12 XP (8%)"); o
  dinheiro cai no chão em montinhos (`itemCoins`, `Loot.N`). O nome do dinheiro é a constante `currency` = "bosta"
  (pedido literal do usuário); os clientes recebem o nome no init.
- **Mercadores** (`merchantsBase`): Morvan, o ferreiro (armas novas para todas as classes) e Ysolde, a herbalista, na vila;
  a Velha do Charco no pântano. E abre a janela de compra; `wsIn{T:"buy", I: loja, Item}`.
- **Tom sombrio** (inspiração Dark Souls/Bloodborne/Lovecraft, nada copiado): fogueiras (`bonfiresBase`) acendem ao encostar
  e são onde se renasce; ao morrer, o dinheiro fica caído só para o dono por 10 min ("Você pereceu"); textos e nomes novos
  (Cinzalva); no cliente, cores lavadas, névoa (`fogAt`/`mistAt`), noite mais funda e crepúsculo vermelho.
- **Personagens**: o save guarda `Tokens` (todos os aparelhos que já abriram) e `Password` (opcional). Entrar com nome +
  senha abre o personagem em qualquer aparelho e libera aquele aparelho. `/chars?token=` lista, `POST /chars/delete` manda
  para `saves/lixeira/` (não apaga). As duas telas de entrada mostram a lista com Carregar/Excluir (com confirmação).
- **Cliente nativo**: barra de título escura (`winbar_windows.go`, DWM), ícone procedural (olho de Tiamath, `windowIcon`),
  F11 e painel de opções (tecla O) com tela cheia e mostrar FPS, salvos em `%AppData%\AsciiMMO\opcoes.json`.
  `-perfil arquivo` grava perfil de CPU; com `-print` o FPS sai no stderr a cada segundo. Um FPS baixo em teste com a
  janela atrás das outras é o Windows segurando o desenho, não o jogo (medido: 60 estável).

## Quarta rodada (14/10/2026)

- **3D aplicado a todos** no cliente nativo (aprovado por ele): gente (inclusive esqueletos: ossos finos, costelas,
  caveira com elmo/coroa), bichos e montarias (`pose3dBeast`) e Tiamath (`drawDragon3D`, asas com dedos e membrana).
  Fila do quadro: tudo entra em `g.op`/`g.enqueue3d` na ordem de desenho; `flush3d` desenha os modelos em paralelo
  (`renderModel` só lê o estado) e depois roda a fila. (Substituído na quinta rodada pelo modo 2D/Mescla/3D.)
- **Brilho (bloom)** no shader Kage: células claras e saturadas (fogo, magia, lava) soltam halo; folhagem não. Opção no O.
- **Renascimento**: comum 3 min, chefe 10 min (`purify.go`). Cada inimigo pertence à fogueira mais próxima do mesmo andar;
  E na fogueira = descansar (cura tudo); com a área limpa, purifica por 30 min (`purifyFor`; não é permanente porque o
  mundo é compartilhado — ele pode pedir para mudar). Fogueiras novas na cripta e nas cavernas.
- **Servidor** (`events.go`): críticos (12%, ladino +10%) e números de dano para quem bate; dano levado sobe em vermelho;
  bola de fogo/meteoro incendeiam o capim (flag 32), que vira cinza, espalha e volta em 4 min (chuva apaga); invasão
  noturna em metade das noites (8 mortos em volta da vila, fogueira apagada, `Event` = 1); ladino no escuro é notado
  ao dobro da distância (`nearestFor`); neblina como clima (`wxFog` = 3).
- **Cliente nativo** (`life.go`): campo de visão por shadowcasting (raio maior de dia, menor à noite/neblina/subsolo;
  o mascaramento do mapa foi revisto na quinta rodada); copa/telhado translúcido perto do jogador (`canopyHides`);
  pegadas na neve/lama e sangue (ou pó de osso) e faíscas no golpe (`decals`); inimigo ferido amarela/avermelha, parado e
  sozinho mostra "z"; envenenado pulsa verde; mago/necromante emitem luz, paladino aura dourada, ladino escurece no
  escuro; balões de fala; itens no chão por símbolo (! ) [ " * % $); aviso dos golpes de área piscando "!" vermelho;
  faixa da invasão. Câmera centralizada no subsolo (sem prender na borda) e zoom inicial aberto estilo Diablo 2/AoE
  (`H/280`).
- **Navegador congelado** para recursos visuais novos (3D, bloom, campo de visão etc. só no nativo); ele recebe só o que
  vem do servidor e correções para não quebrar (ex.: rótulo da neblina).

## Quinta rodada (14/10/2026): Mescla e mapa inteiro

- Ele gostou da iluminação, mas **não quer o mapa escondido** pelo campo de visão, e **não gostou do mago 3D** (preferia
  o 2D). Pediu uma mescla dos dois.
- **Mapa inteiro de novo**: a linha de visão só (a) escurece a sombra atrás das paredes (`shadeOut`: nada de dia,
  até 0,55 à noite, 0,5 no subsolo) e (b) esconde inimigos/rótulos atrás das paredes **só no escuro** (noite `sun<0.35`
  ou subsolo). O modo antigo (memória cinza, nunca visto = preto) virou a opção "Campo de visão clássico" (padrão desligado).
- **Modo de personagens** (`options.Mode`, `g.mode`; `mode2D`/`modeMix`/`mode3D` em story.go), escolhido no painel O
  (clique alterna) e pelas flags `-2d`, `-mescla` e `-3d`. **Padrão: Mescla**, ou seja, humanoides no desenho 2D com luz
  de volume (`body.lit` → `pxLit` em body.go: normal estimada pela cápsula, luz difusa, contorno e especular, e o glifo
  pelo brilho), enquanto bichos, montarias e Tiamath ficam em 3D. `g.use3D` agora quer dizer "humanoides em 3D" (só no modo 3D).
- **Mago 3D refeito**: aba larga como disco fino (raios e anel em cápsulas, já que cone arredondado grande vira bola),
  copa alta que entorta para trás, faixa escura, barba e bigode grisalhos, manto em sino até o chão com barra escura e
  mangas que abrem até o pulso (`sleeve`).
- **Vitrine** com 3 fileiras: 3D, Mescla (destacada) e 2D. Prévias: `previas\vitrine-2d-mescla-3d.png`,
  `mago-3d-novo.png`, `mapa-inteiro-mescla.png`, `caverna-mescla.png`.

## Sexta rodada (14/10/2026): profundidade, castelo e kit de construção

- Referências do usuário em `EVS\` (capim denso, teias, sombra de bicho grande, castelo em ruínas refletido na água) e
  uma lista de dicas dele: silhueta, argamassa escura, caractere pela face, vãos escuros, quente x frio, desgaste,
  reflexo, escala, neblina na altura, recorte com o jogador atrás e interiores diferentes. A profundidade que ele quer
  é a do castelo em ruínas.
- **Pisca do personagem** (pior no mago): a respiração movia o corpo em frações de célula e, no Mescla, cada fração
  trocava o caractere de tudo. A respiração e o passo agora andam em células inteiras; `capLuma` segura o brilho do
  corpo abaixo do limiar do bloom (só o que já é claro de nascença, como o orbe, brilha).
- **Anel no chão** (estilo Diablo/AoE, `groundRing` em `cliente/depth.go`): dourado no seu personagem (maior montado) e
  vermelho no inimigo sob o mouse. Usa `( ) / \ - =`, porque `_` sai cortado e `-` some em célula pequena.
- **Cenário 2.5D** (`cliente/depth.go`, opção "Profundidade do cenário" no O, flag `-sem-profundidade`): o que é alto
  vira um campo de altura (máscara borrada fina e larga, com caixa deslizante), o que dá volume a copa, telhado, rocha
  e muralha; o chão escurece encostado nelas; sombras de nuvem passam de dia; e o alto das estruturas some na neblina.
- **Quem fica na frente de quem** (`life.go`: `computeBases`, `baseAt`, `hides`, `cutoutAt`): cada célula "por cima"
  sabe a linha de base da sua estrutura. Pés acima da base → atrás (some); abaixo → na frente. Com o jogador atrás, a
  parede em volta dele fica vazada e pontilhada na borda. Vale para 2D, Mescla e 3D. Luzes têm `base`: luz de quem está
  atrás de uma fachada não ilumina a face da frente, e a pedra alta pega luz pela metade.
- **Flag 64 = brilho** (`flagGlow`, `wd.lit`): janela acesa, cristal, lava e as janelas frias do castelo não escurecem à
  noite nem levam sombra ou neblina.
- **Bug antigo corrigido**: `dim` não limitava o canal; clarear acima de 1 estourava o vermelho para o bit de cima e o
  amarelo virava verde (eram as janelas "verdes"). Agora cada canal para em 255.
- **Kit de construção** (`kit.go`): `masonry` (argamassa funda, tijolo, musgo, bloco caído), `coneRoof`, `gableRoof`,
  `chimney`, `banner`, `brazier(x, y, high)`, `clearArea`, `watchtower` (inteira ou em ruína), `archRow`, `stoneCircle`.
  A casa da vila foi refeita com ele: empena de duas águas, chaminé, enxaimel, porta de madeira, janelas acesas e
  escuro sob o beiral.
- **Ruínas de Vaelor** (`castle.go`, região nova no FIM de `regions`, para não mexer nos índices das missões): castelo
  na margem norte do Lago Espelhado, com a fachada na beira do fosso para o reflexo. Muralha de 50 de altura com ameias,
  portão em arco, janelas em dois andares, brecha à direita; torres dos cantos (74 e 40, esta desabada), do portão
  (64 com telhado e 60) e do fundo; muralhas laterais vistas de cima com a brecha oeste (por onde chega a trilha nova);
  pátio de lajes com capim, poço seco e entulho; cais com braseiros; braseiros no alto e duas janelas de luz fria.
  4 esqueletos guerreiros e 4 esqueletos no pátio.
- **Marcos** (`landmarks` em `kit.go`, longe de árvores, rochas e falésias): torre de vigia inteira na vila, três
  torres em ruína nas trilhas, claustro de arcos perto do cemitério e círculo de menires numa clareira a oeste da vila.
- **Desempenho**: terreno, sombra, profundidade, borrão e passe final rodam por linha em paralelo (`parallelRows` do
  cliente): 60 FPS no zoom 3 (antes ficava em ~32 com a profundidade).
- Teste novo `TestRuinasEKit`. Prévias: `castelo-vaelor-dia.png`, `castelo-vaelor-anoitecer.png`,
  `castelo-vaelor-portao.png`, `vila-casas-novas.png`, `marcos-vigia-ruina-arcos-menires.png`,
  `profundidade-sem-x-com.png`, `anel-de-selecao.png`.
- Das referências, ainda não feito: capim denso de lâminas finas, teias de aranha em linha e a sombra grande de
  Tiamath no chão; interior do castelo com paleta própria; bandeiras balançando no vento.

## Sétima rodada (14/10/2026): teia de habilidades, economia e correções

- **Teia de habilidades** (`skills.go`, `classes.go`; cliente `webtree.go`, `icons.go`, `fx_habilidades.go`): cada classe tem um
  livro de golpes (`classes[c].Skills`, o 0 é o básico, o 1 já vem aprendido) e uma teia: núcleo + ramos (3 por classe,
  4 no mago: FOGO, GELO, RAIO, ARCANO) com 4 anéis e fios no 2º anel ligando ramos vizinhos. Nó pede nível (2/4/7/11),
  pontos no atributo da classe (`class.Attr`: Força, Mana, Vida, Stamina; 1/3/6/10), 1 ponto de habilidade e ligação a
  um nó aprendido; o golpe da cripta pede `Unlocked`. Ativos ensinam golpes; passivos somam `bonus` por grau
  (`p.bon`, recalculado em `recalc`). Descrições longas com números em cada golpe/passiva. `SpendTree(i)` = aprender
  nó i (nome antigo, o navegador ainda usa); `SetBar(slot, skill)` = atalho. Save: `Web`, `Bar`, `SkillVer=1`; save
  antigo devolve os pontos da árvore velha e dá o golpe da cripta de graça a quem já tinha descido.
- **Elementos e efeitos** (`hitOpt`): lentidão, atordoamento/congelamento (chefe não), queimadura e veneno por segundo
  (`tickDot`), reforços (Grito +30% dano, Olho de Águia +25% crítico, Lâminas Envenenadas), zonas que duram
  (`hazard`: muralha de chamas, nevasca, tempestade, consagrar), corrente de raio (`chain`), piscar/rolamento
  (`blink`). Fx novos 14–28 (`fxChain`… `fxStorm`), projéteis `projIce`=4 e `projHoly`=5.
- **Barra de ações**: Espaço/clique = básico; 1–5 = atalhos (F, R, botão direito e do meio = atalhos 1 e 2). Com a
  teia aberta (K), mouse sobre golpe aprendido + número põe no atalho. Ícones em pixel art 12×12 por figura e elemento.
- **Nível dos inimigos** (`levels.go`, estilo The Witcher 3): nível por tipo; `lvScale` pesa o dano nos dois sentidos;
  6+ níveis acima = caveira vermelha no nome e golpe que pode matar de uma vez.
- **Barco atracado**: ao descer, o barco fica onde estava (`Moored`, salvo); E perto dele embarca (ilha não prende mais).
- **Missão de chefe já cumprida** (`catchUp`, `Player.Slain`): quem matou a Rainha (etc.) antes da missão dela chegar
  tem a missão fechada sozinha; vale para secundárias e usa os bits antigos de `Bosses`.
- **Explorado salvo no servidor** (`explore.go`, save `Fog`, init `fog`); cliente junta com o arquivo local.
- **Marcas de missão** (`questmap.go`): `!` dourado (principal) e `?` lilás (secundárias) no minimapa e mapa grande,
  com a entrada quando o alvo é no subsolo. Seta na tela é opção (O), desligada: o jogo é para explorar.
- **Economia**: Tônico do Esquecimento (atributos) e Tomo do Recomeço (teia) nas herbalistas; vender itens em qualquer
  mercador (coluna "Vender"); jogar fora com botão direito na mochila; Chave de Ferro no ferreiro abre os cofres
  (`vaults`, porta `propDoor` que volta a trancar em 3 min) com baú dentro: pátio de Vaelor e depósito da guarda.
- **Interface**: avisos do sistema vão para o registro (L, conta os novos); falas aparecem 12 s; controles no H.
- **Mapa**: mata fechada só no miolo (f > 0,74), borda com vegetação rasteira andável e trilhas de bicho cortando;
  trilhas entre regiões bem sutis (chão gasto que some em trechos).
- **Tiamath estilo King Ghidorah** (`drawDragon3D`, `dragonClass`): cinco cabeças em pescoços longos em leque (sombra,
  raio, bronze, veneno, fogo), duas caudas, asas de quatro dedos, escala 1,7. Modelo grande (`rad*scale > 50`) faz um
  raio a cada bloco 2×2.
- **Obliterar** (mago, golpe 11, Exódia): nó final da teia num 5º anel (`capstone`, nível 18, Mana 14, liga aos 4º
  anéis). 1,2 s de selos em pentagrama (`fxSeal`=29, protegido) e feixe de 130×28 (`fxOblit`=30), 500–800 + 3 s de
  atordoamento; 80 MP + zera a mana; recarga 90 s.
- **Shader e capim**: fundo da célula tingido pela cor (`Tint` 0,13); atlas 16×16 com glifos próprios a partir do 128
  (`glyphs.go`: lâminas de capim em 12 inclinações curtas/longas e 4 flores); capim usa lâminas por vento e desvio,
  tom por lâmina e flores raras; pasto mais denso no gerador. Desempenho: luzes por linha (`rowLit`), névoa e nuvens
  numa grade grossa (`fogGrid`).
- **Ícones de itens e conquistas** (`cliente/itemicons.go`): figura por peça do set (elmo, peitoral, botas, amuleto,
  anel), por tipo de arma, poção, erva, madeira, pedra, moeda, tomo, chave; cor do próprio item (`tintedIcon`).
  Conquistas viraram grade de cartões com ícone aceso/apagado (`achIcons`, na ordem do servidor). A arte "ring" e o
  tipo 11 já existem no cliente, mas o jogo ainda não tem anéis (nem espaço de anel no equipamento).
- **Card de status** (`drawStatusCard`): retrato com o ícone da classe, selo do nível, "+" piscando com pontos a gastar,
  barras cheias com números, XP fina e dinheiro com moeda. A fonte não tem ✓/✗: usar `checkMark`.
- **Teia em todas as classes** (`classes_ramos.go`): cada classe além do mago ganhou um 4º ramo (Guerreiro SANGUE,
  Cavaleiro HONRA, Paladino JUSTIÇA, Arqueiro CAÇA, Ladino ASSASSINO, Necromante MORTOS) com 2 golpes (7, 8) e um
  golpe final (9) no 5º anel (`capstoneOf`). Ramos em X (-135, -45, 45, 135).
- **Servos do necromante** (`minions.go`): Erguer Mortos (2 esqueletos, 25 s), Colheita de Almas (dreno em área, cura
  por alvo), Legião dos Mortos (6 guerreiros, 30 s). Servos seguem o dono, atacam o inimigo mais perto, são alvo dos
  inimigos, máx. 8; vão ao cliente como `Mob` com `Ally` (ID negativo, rótulo verde). Passiva Ossos Firmes (`bon.Minion`).
- **Bug da EV01 corrigido**: o 5º anel cabia por cima do título; nós se encostavam e números ficavam embaixo dos
  vizinhos. Agora o passo da teia considera o 5º anel, o tamanho dos nós segue o passo, e graus/atalhos vão numa
  segunda passada, dentro dos nós.
- **Mochila**: 50 espaços de início (`baseBag`), teto 140 (`maxBag`), `Player.Bag` salvo. Ampliações (`kindBag`=12,
  Bonus = espaços) valem na compra, sem ocupar espaço, vendidas por mercadores escondidos: Eremita da Ilha (+10),
  Mascate das Sombras na sala secreta das cavernas (+20/+10), Velho Guardião no pátio de Vaelor (+30/+20). Ficha
  mostra a mochila em páginas de 10×5.
- **Escala**: todo personagem passa por `charScale` = 0,62 em `sc()` (gente, bichos, montarias); Tiamath tem escala
  própria. Zoom inicial passou a `H/210`. Rótulos, luzes e projéteis ajustados.
- **Vila** (`kit_vila.go`): cabanas de mercador com frente aberta, balcão, telhado de palha e placa com o nome (ferreiro e
  herbalista; o mercador fica na boca da cabana), tapetes, caixotes, barris, potes, lenha, carroça e cercas. Placas de
  direção ao lado das trilhas (8 de 12 couberam), texto só em ASCII (`asciiUpper`).
- **Sombras longas** (`castShadows`): cada célula alta projeta sombra do tamanho da altura acima da base, para o lado
  oposto ao sol; `computeBases` com margem de 70 para estruturas fora da tela.
- Referências EV2/EV3 (Warlords Battlecry, Diablo 2) são isométricas; converter o jogo para isométrico seria refazer
  terreno, colisão e desenho. Ficou em aberto, para decidir com o usuário.
- **Missões em etapas** (`queststeps.go`): objetivo = título/texto; subobjetivos em `questSteps` (por título): chegar
  a região, descer à cripta, derrotar N. `Player.QStep` (salvo; save antigo cai na etapa do mesmo tipo). `catchStep`
  fecha na hora etapa já cumprida (já na região, chefe já derrubado). Marcas do mapa e painel seguem a etapa atual
  (`Quest.Steps`, `Quest.Cur`); painel mostra visto nas feitas, ">" na atual.
- **Viagem rápida** (`travel.go`, cliente `travelui.go`): fogueiras + placas de direção (`wd.signs`) são pontos;
  descobertos a 24 células (`Player.Found`, salvo); tecla V num ponto descoberto abre o mapa/lista; `Travel(i)`.
- **Pasta de dados** (`proto/paths.go`): raiz `Documentos\AsciiMMO` (ou a escolhida no O, guardada em
  `%AppData%\AsciiMMO\pasta.txt`); servidor salva em `<raiz>\save` (flag `-saves` sobrepõe, `-saves -` desliga; copia
  `saves\` antigo na primeira vez); cliente usa `<raiz>\cliente` (copia o `%AppData%\AsciiMMO` antigo, inclusive o
  token). Botão "Escolher pasta…" usa a janela do Windows via PowerShell (`folder_windows.go`).
- **Montarias**: lista na ficha (I) com Montar/Descer e Soltar (pede confirmação); servidor `PickSteed`,
  `ReleaseSteed`. Cavaleiro sentado (`bodyDraw.seated`, 2D e 3D): quadril na sela, pernas pelos flancos, contorno só
  da sela para cima (`edgeMaxY`); montarias com escala 1,6.
- **Portas seladas** (`secrets.go`, cliente `seals.go`): santuários com baú; Portão do Amigo (dizer "amigo" no chat
  diante da porta) e Cripta dos Ecos (E abre o modo de runas; sequência de setas lida em 5 pedras rúnicas com numeral
  e seta). Runas e pedras brilham e soltam faíscas (a pista no mapa); porta volta a selar em 3 min.
- Barra: "Espaço" virou "ESP". Ícone próprio por montaria (`steedIcons`).
- **Depravado** (`depraved.go`, classe 7): teia com o 1º ramo de cada classe (golpes copiados para o livro dele) +
  golpe final Caos Depravado; segue no máximo 3 escolas (`schools`, `maxSchools`); teia pede o atributo mais alto
  (`anyAttr`; init manda `mattr = -1`); começa com o Pedaço de Pau (`itemStick`, arma "club"); afinidade com tudo.
  `maxSkills` = 20. `webNode.Branch` marca o ramo de cada nó.
- **Card** (`drawStatusCard`): enxuto em ASCII: retrato 40px, "Nome · Classe  nv N", barras em texto (= cheio, · vazio) com número, xp e dinheiro numa linha; fundo translúcido e traço dourado. Retratos 16×16 por classe (`cliente/portraits.go`).
- **Menus em lista**: telha de tecla (`keyTile`) + nome; listas com marcador (`bullets`). Controles (H) agrupados (Movimento, Combate, Mundo, Janelas); menu da direita clicável; linha de baixo H/L; dicas da ficha e da teia em lista. `-painel help` abre a ajuda para prints. Opção "Menu de teclas na tela" (`HideMenu`) esconde a lista da direita e a linha H/L.
- **Cortar vegetação** (`grasscut.go`): golpes que não são gelo/sombra/sagrado/veneno cortam capim e moitas na forma
  do golpe (vira restolho "."), volta em ~3 min (`regrows`); às vezes larga moedas, erva, poção, tocha ou machado;
  moitas rendem madeira. Fx `fxGrass`=31 (partícula "cut").
- **Ferramentas** (`tools.go`, `kindTool`=12): Lanterna (luz, funciona na chuva; herbalista ou baú da torre em ruínas
  da Encruzilhada), Tocha (luz forte 5 min, chuva apaga; grátis ao descansar na fogueira ou herbalista), Machado
  (na mochila: árvore 4x mais rápido e +2 madeira; ferreiro ou achado cortando moitas). Clicar acende/apaga
  (`Player.Light`, vai ao cliente como `lt`). Tempestade: arma de metal equipada atrai raio (35%), e raio pode cair
  numa árvore, que cai e incendeia o capim (`boltTarget`, `strikeTree`).
- **Tipos de item no cliente**: 9 pergaminho, 10 chave, 11 mochila, 12 ferramenta, 13 anel (anel ainda não existe).
- **Ícone no .exe**: `build.ps1` gera `cliente/winres/icone.png` (`-gerar-icone`, do `windowIcon`) e os `.syso` com
  o go-winres (`~/go/bin/go-winres.exe`), depois compila os dois. Sem isso o .exe não tem ícone (a dock lê do .exe).
  Compilar sempre com `.\build.ps1` (ou `go build` comum, que reaproveita os `.syso` já gerados).
- Ainda na fila desta conversa: mapa grande dividido por regiões (estilo mapa de
  campanha); shader: fundo da célula tingido, rampa ordenada por cobertura, glifos de traço para o capim denso com flores.

## Prólogo: Vale do Despertar (14/10/2026)

- `servidor/intro.go`: vale em elipse (70×46, mureta de pedra aberta a leste) em `introX,introY` = vila −330/+300.
  Personagem **novo** (`JoinSaved`, ramo sem save) nasce com `Intro=0` perto do Eremita, renascendo na "Fogueira do
  Despertar" (última de `bonfiresBase`, fora da viagem rápida). `spawn` põe `Intro=len(prologue)`, então Join/SSH/testes
  e saves antigos (sem `Prologue`) pulam o prólogo. Quem volta no meio do prólogo reaparece no vale (o save não guarda
  posição).
- Etapas próprias (`prologue`, fora da corrente de missões): falar com o Eremita (`qTalk`, msg `talk`), 3 bonecos de
  treino (`mobDummy`: não anda, não ataca, sem saque), descansar (`qRest`, de `rest()`), chegar à vila (`qVisit` 1).
  `finishPrologue`: +30 bosta, 2 poções, fogueira da vila. Painel mostra "Prólogo"; o letreiro "Capítulo I" espera o fim.
- NPCs falantes: `proto.NPC` no Init (`npcs`); cliente `npcs.go` desenha (estilo `hermitStyle`), E a ≤20 abre a conversa
  (uma fala por vez, E/Espaço/Enter/clique avança, Esc fecha). `-painel talk` abre numa fala do meio, para prints.
- Abertura (`cliente/opening.go`): 3 telas em ASCII (Cinzalva ardendo, Tiamath, o vale), texto letra a letra, na primeira
  entrada no prólogo etapa 0; marca `abertura-<nome>.visto` na pasta do cliente.

## Técnicas, barra de 9, Profundezas e Midrath (14/10/2026)

- **Mestre Oren** (`servidor/techniques.go`, cliente `dojo.go`): Terreiro das Sequências em `hubX-120, hubY+70`
  (região própria, trilha a partir da Encruzilhada). 5 técnicas para qualquer classe (Rolamento DRR nv3, Brado DUU
  nv5, Redemoinho LDRU nv8, Punho do Dragão DRDRU nv12, Golpe de Misericórdia RLRDUL nv16), cada uma com nível e
  preço. Ficam no fim do livro de cada classe (`class.techBase`; `maxSkills` = 32), fora da teia; `Player.Tech` (bits,
  no save e no estado como `tech`). O mestre mostra a sequência seta a seta e a esconde; o jogador repete (setas ou
  WASD) no tempo; abaixo do nível ele nem mostra. Msg `learn` (I = técnica, Text = sequência); o servidor confere tudo.
  `proto.NPC.Teach` faz a conversa terminar no terreiro (e da segunda vez E vai direto). Técnicas aprendidas aparecem
  numa faixa na teia (K) e no terreiro: mouse/seleção + 1–9 põe na barra. `-painel dojo` para prints.
- **Barra de 5 a 9 atalhos** (`barSlots`=9, `barCap()`, `barLevels` = 6, 10, 14, 18; os dois lados têm a lista).
  Estado leva `barn`; atalho trancado mostra "nv X". Save com barra de 5 volta com o resto vazio (`emptyBar`).
- **Profundezas** (`servidor/depths.go`): faixa do mapa abaixo da cripta (`cryptH`=240, `deepY0`=overH+240,
  `deepH`=overH; mundo 3072×5040), alinhada 1:1 com a superfície (x, y) ↔ (x, deepY0+y); só há galeria sob terra
  firme. Galerias = curvas de nível de dois ruídos + salões; túneis garantidos em anel entre as fendas, de cada região
  de baixo à fenda mais perto, e da última fenda ao coração. Biomas pelo que está em cima (`deepBiome`): fogo/lava sob
  o Pico, arenito, gelo, fungos (poças venenosas), pedra (cristais). 9 fendas (`rifts`, portal "Fenda · nível 15+",
  escada "Subida"). Inimigos: Rastejante Cego (16), Casca de Fungo (17), Golem de Magma (19), **Ignar, o Coração de
  Magma** (22, chefe no anel de lava sob o Pico; larga Malho de Magma e Botas de Obsidiana). Fogueiras do Fundo, das
  Raízes e do Coração; **Durgo, o anão do fundo** vende Manto de Cinzas (700) e Botas de Obsidiana (1100).
- **Lava**: `wd.marsh` = 3 (`lavaMarsh`), flag 128 (`flagLava`) + glow; `lavaStep` (no `environ`, 1×/s) tira 45 e
  dá insolação; Botas de Obsidiana anulam, Manto de Cinzas divide por 4. Cliente anima a lava (crosta, veios, bolhas).
- **Mapa em duas partes**: `/map` manda só [0, deepY0) (o navegador segue igual) com `X-Map-Full-H`; `/map?parte=fundo`
  manda as Profundezas. O cliente reserva o mundo todo e baixa a parte de baixo quando chega a 40 linhas dela
  (`wantDeep`), com a tela "Descendo às Profundezas…" se precisar esperar. Zona "Profundezas" no minimapa; regiões de
  baixo dão o nome do lugar. Névoa salva antiga é prefixo da nova (completa com zeros, no servidor e no cliente).
- **Midrath** (`servidor/divina.go`, tema inspirado em Berserk com nomes nossos): o continente se chama Midrath (tela
  de entrada, título da janela, abertura, Eremita). **Árvore Divina** em (1240, 960): clareira, raízes, tronco torcido,
  copa em espiral que brilha; fogueira e fenda próprias; região "Raízes da Árvore Divina" embaixo. **Apóstolo Faminto**
  (18) ronda a árvore e as Ruínas de Vaelor; **Grael, o Apóstolo Rei** (23) guarda a clareira e larga a Lâmina de Ferro
  Bruto (+42%) e o **Ovo Escarlate** (amuleto: abaixo de 1/4 da vida devolve 30%, a cada 2 min). **Armadura do Abismo**
  (baú sob a árvore, nas Profundezas): +40% de dano com menos de metade da vida, cobra 2 de vida por segundo.
- Testes novos: TestMestreDasSequencias, TestBarraPorNivel, TestProfundezas (fendas, ligação por busca em largura,
  lava), TestArvoreDivina. dbg7 tem lugares de print: `mestre`, `fundo`, `coracao`, `arvore`.

## Profundezas, rodada 2 (14/10/2026)

- `servidor/dorrim.go`. **Salões de Dorrim** (ruínas anãs) em `hubX-90, deepY0+hubY+50` (88×52): muralha dupla,
  piso de lajes, colunas, forja apagada com fio de lava, estátuas nas portas leste e oeste; cada porta tem túnel até
  uma fenda diferente (`carveDeep2`, na grade, antes de pintar). **Cofre do rei** no canto nordeste, fechado por parede
  rachada; baú com Elmo de Dorrim (dia) ou Martelo de Dorrim (noite). 6 **Guardiões de Dorrim** (18).
- **Câmaras secretas** sob Arenito, Gelo Antigo, Jardim de Fungos e Raízes de Marapá: câmara a (+38, −10) do centro da
  região, corredor estreito, parede rachada no meio (`deepSecrets`); baús com Escaravelho Dourado, Coroa do Gelo
  Antigo, Botas de Micélio e Arco de Raiz Viva. São 5 paredes rachadas nas Profundezas no total.
- Inimigos por bioma (`scatterDeep2`, pelas regiões de baixo): Escorpião de Arenito (16, veneno), Horror das Raízes
  (17, veneno; também sob a Árvore), Espectro Gélido (18, frio), Casca de Fungo no Jardim.
- Secundárias novas (`numSide` = 8; não quebram selo): "O que rasteja embaixo" (cap. 4, 6 Rastejantes), "A forja
  apagada" (cap. 5, 5 Guardiões), "O coração que arde" (cap. 6, Ignar, sobe um nível). `oldQuest` continua marcando só
  as 5 antigas. TestProfundezas2 cobre ligação, cofre fechado, baús alcançáveis depois das rachaduras e o crédito de
  Ignar (o crédito vai para quem está perto do inimigo). Print em dbg7: `dorrim` (nível 60, os guardiões matam).

## Vulkhar, o Flagelo das Profundezas (14/10/2026)

- Pedido: um demônio das profundezas como o Balrog — feito com nome e golpes nossos (`servidor/vulkhar.go`). Mora no
  **Abismo de Dorrim**, ao sul dos salões (porta sul nova + corredor, `carveAbyss` na grade): salão de raio 46 cortado
  por um precipício (rocha que não se pisa, com brasas que brilham no fundo) e uma ponte de 5 de largura. A região do
  Abismo fica fora do laço de túneis das regiões de baixo: só se chega pela ponte.
- Nível 27, 6200 de vida, chefe (tem o pisão). Golpes próprios em `demonStep` (estado em `wd.demon`, um só no mundo):
  **chicote** (marca a linha com `FxWhip` por 0,7 s e então arde com `fxFlame`, 52 de comprimento; fere quem está a
  ≤ 5 da linha e dá insolação; a cada 5 s), **aura** (quem está a ≤ 16 se queima 1×/s; o Manto de Cinzas divide por 4),
  **fúria** abaixo de metade (chicote a cada 3 s, aura 24, aviso na tela). Larga o Chicote de Vulkhar (+46%) e a Coroa
  de Chifres. Secundária "O Flagelo das Profundezas" (cap. 6, `numSide` = 9).
- Cliente: `FxWhip` (linha em brasa que pulsa mais rápido perto de arder); `bodyStyle.embers` (Vulkhar, Ignar e
  Golens soltam brasas e iluminam em volta). TestVulkhar cobre precipício/ponte, chegada, chicote só na linha, aura e
  fúria. Prints em `previas/` (profundezas-*, midrath-arvore-divina, terreiro-das-sequencias); dbg7: `ponte`,
  `vulkhar` (vida de nível 60 e dano de 24, para a luta durar no print).

## Portões Rúnicos e Cavernas de Cristal (14/10/2026)

- `servidor/portoes.go`, cliente `gates.go`. 9 arcos de pedra (`gatesBase`; a ordem é a corrente e o índice dos bits
  em `Player.Gates`, que vai no save e no estado como `gates`): Cinzalva (sem runa; acende ao chegar perto, por
  `gateFree` no `environ`, e dá 3 Pedras de Eco), Marapá (junto à Fogueira da Selva), Areias, Marés, Picos, Árvore
  Divina (Grael perto: por isso no fim da superfície), e embaixo Dorrim, Cristais, Coração. Acender o i grava no chat
  a runa do i+1; o cliente mostra essa runa no painel do portão aceso e na entrada do apagado. E diante do portão:
  apagado → setas tocam a runa (msg `gaterune`); aceso → lista dos acesos, clicar atravessa (msg `gate`) gastando
  uma **Pedra de Eco** (`itemEchoStone`: 12% de cair de inimigos das Profundezas; Ysolde 60, Durgo 50). O painel
  fecha ao se afastar. Os portões de baixo têm patamar e túnel até a fenda mais perto (`carveGates`). `-painel gate`.
- `servidor/cristais.go`: **Cavernas de Cristal** sob o Lago Espelhado (`crystalDist`, bioma `dbCrystal`): galerias
  mais abertas, 9% das paredes com cristal que brilha, região própria, **Golem de Cristal** (20).
- **Cofre de cristal**: `vaultKind` com porta `propSeal` no salão central (base em `crystalY-4`) e 3 **prismas**
  (`propPrism`, 120 de vida, voltam em 25 s). Quebrar os três a tempo (`prismsBroken`, chamado em `hitProps`) abre a
  porta, que fecha sozinha em 3 min como as seladas. Baú: Coração de Cristal (dia) ou Lâmina Prismática (noite).
  TestCofreDeCristal. No cliente, lanterna e tocha alcançam 60% mais longe nas Cavernas de Cristal (`regionName`).
- TestPortoes cobre o primeiro portão, runa errada/certa, travessia gastando pedra, save e os portões de baixo
  dentro das galerias. Prints: `previas/portao-runico*.png`, `previas/profundezas-cavernas-de-cristal.png`.

## Som, bússola e história animada (14/10/2026, noite)

- **Som** (`cliente/som.go`, `som_sintese.go`, `som_eventos.go`): efeitos e ambiente **gerados por código** na abertura
  (goroutine; `genSounds`/`genLoops`), música em **ogg CC0** embutida (`cliente/sons/musica/`: dia, vila, noite,
  cripta, fundo, chefe; créditos em `CREDITOS.txt`). Escolha feita com ele: "som misto". Tudo sai da diferença entre
  estados (`soundEvents`: Fx e projéteis novos, `Hit`/`Atk` zerando, inimigo ferido que some, ouro, nível, inventário,
  fogueira, chat) e das células quebradas (`editSounds`, antes de `applyEdits`: madeira, pedra, cristal, árvore
  caindo). Ambiente por lugar (`updateSound`: vento, grilos de noite, chuva, mar/lago, caverna, lava, fogueira) e
  avulsos (pássaro, coruja, sapo, pingo, uivo, chamado dos bichos, rugido de Tiamath). Voz por tipo de inimigo em
  `mobVoice`, **na ordem dos tipos de `servidor/rpg.go`** (inimigo novo = acrescentar lá). Passos pelo chão.
  Volumes no O (música 50, efeitos 80, ambiente 70; degraus de 20) e "Calar o jogo sem foco". `-sem-som`; prints
  ficam mudos. TestSintese. O servidor não manda nada a mais para o som.
- **Bússola** (`cliente/compass.go`): faixa no topo central, gira com o rumo do andar; marca até 6 pontos de viagem
  descobertos do mesmo andar e a missão só com a seta ligada; treme no subsolo, enlouquece no Abismo/Ventre de Fogo.
  Opção no O.
- **Chat**: ↑/↓ voltam pelo que ele já disse (`chatSent`); com o registro (L) aberto, ↑/↓, roda, PgUp/PgDn rolam
  (`logScroll`), End volta ao fim; com o registro aberto as setas ↑/↓ não andam (W/S andam).
- **Card do personagem**: moldura dourada inteira com cantos `+`.
- **Sombras**: a curta e a longa vão para uma máscara só (`g.shadeF`, `applyShade`, fica a mais escura) em vez de
  somar; a longa é faixa contínua do pé à ponta (antes só a ponta, pontilhada); nuvem só no chão; a sombra elíptica
  dos personagens não escurece mais parede/telhado/copa (era o "aplica sobre tudo").
- **Bonecos** (`body.go`): letras do corpo numa rampa curta e densa (`bodyRamp` "o0O8&%#@", sem sorteio por célula),
  membros mais grossos (limb 1,75), ombros largos, cabeça 3,7, mãos e botas maiores. Escala (`charScale`) mantida.
- **História** (`cliente/historia.go`): `drawArt` desenha ASCII animado letra a letra (água corre, fogo treme, estrela
  pisca, olhos pulsam, linhas balançam) com partículas (brasa, neve, pó, esporos, chuva, estrelas). Abertura usa isso;
  o letreiro do capítulo virou **card de 9 s** com arte própria de cada capítulo (`chapterArt`), número romano, título
  e subtítulo surgindo letra a letra e uma narração. Prévias: `card-capitulo-vi.png`, `card-capitulo-iii.png`,
  `abertura-animada.png`, `bussola-card-moldura.png`, `bonecos-antes-depois.png`.
- Ainda não feito disto: equipamento (elmo/botas/amuleto) aparecendo no corpo; cena ao fechar cada capítulo.

## Correções, pausa, salvar e lojas (14/10/2026, noite)

- **Mapa preto ao norte (bug antigo do cliente nativo)**: `fetchMap` (`cliente/net.go`) usava fatias do buffer do
  download, e o `append` que reservava as Profundezas escrevia zeros por cima das cores das ~800 primeiras linhas
  (cemitério, castelo, montanhas). Agora cada camada é copiada para uma fatia própria. TestMapaCores.
- **Inimigos presos** (`servidor/caminho.go`): com a reta bloqueada, o inimigo pede um caminho A* (8 vizinhos, caixa de
  raio 72, até 9000 nós, no máximo 10 buscas por passo, `wd.pathBudget`) e segue os pontos; refaz quando o alvo se
  afasta, quando para no lugar ou a cada ~1,5 s; volta para a cova contornando. `Mob.nav`. TestCaminho (parede em U).
- **Fendas estreitas** (`widenGalleries` em `depths.go`): as galerias naturais (curvas de nível, 1 a 3 células) são
  alargadas 3 para cada lado e 1 para cima/baixo antes dos túneis; o personagem ocupa 5 de lado (`fits`).
  TestProfundezas agora busca com `fits`.
- **Pausa** (`servidor/pausa.go`, `cliente/pausa.go`): msg `pause` (I=1/0). Todos os conectados pausados → o mundo
  para: `wd.clock()` (hora real menos `pausedNs`) substituiu `time.Now()` no servidor (menos SSH, rede e lixeira), e
  `Run` não chama `step`. Com outros jogando: pausado fica parado, sem regenerar, invisível para inimigos
  (`nearestFor`), sem dano (`hurt`/`hurtRaw`), sem andar/golpear. Cliente: Esc sem nada aberto ou P abre o menu
  (Continuar, Salvar, Opções, Controles, Voltar à entrada); opção "Pausar ao abrir menus" (padrão ligada; mochila,
  loja, teia, mapa grande, conversa, abertura). Estado: `pz`, `frz`, `sv`; outros veem "· pausado". `-painel pausa`.
  TestPausa.
- **Salvar**: o save guarda a posição (`HasPos`, `PosX`, `PosY`; volta lá se ainda couber, fora do prólogo); grava a
  cada 30 s, ao sair, ao descansar e com F5 / menu (msg `save`, `SaveNow`, pop "Jogo salvo"). Cada gravação sobe
  `Player.saves` → cliente mostra "[=] salvo". TestSalvarPosicao.
- **Lojas** (`cliente/loja.go`, `servidor/lojas.go`): E no balcão abre conversa (fala do mercador letra a letra,
  falas próprias em `shopLines`); 1 Comprar, 2 Vender, 3 Conversar, 4 Até logo. Lista com ícone e preço + painel de
  detalhes (descrição, dano/proteção, comparação com a arma equipada, quantos tem) e botões Comprar/Vender 1, 5 ou
  todos; ↑/↓, Enter, Tab, Esc. A Velha do Charco ganhou cabana (`shopHut`); Eremita, Guardião, Mascate e Durgo, barraca
  (`stall`: balcão, postes, toldo listrado, placa; nada sólido e nada em cima da água). `-painel loja` /
  `-painel loja-compra`. Prévias: `loja-conversa.png`, `loja-comprar.png`, `menu-pausa.png`, `castelo-mapa-corrigido.png`.
- Prints de teste: rodar o cliente com `HOME=<scratch>/home` (a pasta real dele é `~/AsciiMMO/cliente`, personagem
  `gandalf`).

## Terceira lista da noite (14/10/2026): interface, doma, casa, abertura

- **Repositório**: esta é a versão Linux, `origin` = https://github.com/GFiamoncini/AsciiMMO (conta própria dele). A do
  Windows fica em GFBmsoft/AsciiMMO e recebe as mudanças à mão: **quando ele disser que terminou, entregar a lista de
  correções e melhorias desde `1a4ae09`** (os commits desta fase são a base).
- **Tela** (`cliente/hud_layout.go`): minimapa à esquerda sob o card (`drawMiniLeft`), outros jogadores em fichas
  (`drawPartyChips`), estação/clima/hora em ícones com dica sob a bússola (`drawWorldInfo`, `tooltip`), coluna da
  direita só missão e menu. Bússola: rumo pela média do andar (`hvx/hvy`), sem tremer fora do Abismo/Ventre, nome à
  frente fica 1,2 s. Card com barra de XP ("faltam N"). Seta da missão sem texto.
- **Tela de entrada** (`entrada.go`): mesmo desenho das lojas, cena do vale ao fundo, abas Seus personagens (1–9
  carregam) / Novo personagem (classes com retrato + painel de detalhes `classLore`).
- **Luz**: no subsolo o jogador sem luz enxerga raio 20, luz fria fraca (era verde, raio 38). Espaço de luz
  (`servidor/luz.go`, `LightSlot` no save e no estado `lslot`): clicar lanterna/tocha equipa e acende, Z liga/desliga
  (msg `light`), acende sozinha ao descer. `stepLight` agora roda (antes a tocha nunca apagava). **Luz suave (GPU)**:
  o shader interpola o brilho entre células (`Smooth`, opção no O, padrão ligada).
- **Raio do mago dourado** (`drawBoltC(…, true)`, corrente e faíscas `volt`); tempestade segue branca.
- **Doma** (`servidor/doma.go`, cliente `doma.go`): pede stamina (`tameNeed`: 30/45/55/75) e uma sequência de 3 a 6
  setas no tempo; errar gasta metade e o bicho foge. Estado `tame`, msg `tame`.
- **Fogueira** (`servidor/espera.go`, cliente `fogueira.go`): ao descansar abre painel; esperar até amanhecer, meio-dia,
  anoitecer, meia-noite (`wd.todShift`, `dayOf`) ou o clima virar (`WaitWeather`; respeita `-weather`). Vale para todos.
- **Inimigos**: `mobRad` (tamanho no chão; grandes = 2), `mobFits`, `mobLine`, A* com o tamanho, vagar só para ponto
  alcançável (`wanderPoint`), `unstick` 1×/s. TestCaminho com esqueleto e Apóstolo.
- **Vozes por inimigo** (`genVoices`, `formantVoice` com formantes de vogal) e `mobVoice` revisto (orc, troll, gigante,
  yeti, esqueleto, aranha, escorpião, escaravelho, asura, tritão, golem, apóstolo, anão, múmia, espectro…).
- **Saídas** (`drawExit`, `Portal.Up`): arco, degraus, raios e luz de dia/luar nas saídas para a superfície.
- **Diário** (N, `cliente/diario.go`): missões principais (Init `quests`) e secundárias; história com rever a abertura
  e os cards dos capítulos.
- **Abertura** (`cliente/abertura_cena.go`): cenas procedurais na grade (segundo `gpu` com atlas próprio, porque o
  atlas é do tamanho da célula), narração na "língua velha" (`oldWord`, nomes próprios ficam) com voz sintetizada
  palavra a palavra no ritmo da legenda (`introCPS` 22), música da noite e ambiente por cena. `-painel abertura0..2`.
- **Casa** (`servidor/casa.go`, cliente `casa.go`): Casa do Poço (1200) e Casa da Praça (2000) na vila; E na porta
  compra; depois abre o baú (itens e dinheiro, no save: `House`, `Stash`, `StashGold`). Cada jogador tem a sua.
  TestCasa. Atenção: o item 0 é a Poção de Vida (filtros de id usam `>= 0`).

## Fila (próximas rodadas)

- **Faixa dos Planos** (pedido "aumentar o mapa"): terceira faixa abaixo das Profundezas (parte 2 do mapa, baixada à
  parte) para as terras só por portão — **Ilha das Fadas** (luz eterna, loja única) e **Terra do Eclipse** (abre num
  evento raro de eclipse; Apóstolos em bando e o chefe mais difícil). Portões "sem par" levam a elas.
- **Costa Vulcânica** (litoral norte, livre por volta de (2240, 340): areia preta, gêiseres, caranguejos de basalto)
  e **Cidade Afundada** (Profundezas sob Talássia: ruas alagadas, Amuleto de Fôlego, tritões antigos, chefe das marés).
- Profundezas, rodada 3: eventos (desabamento, rio de lava que sobe), missão principal que passe por baixo, poços de
  luz ligando a superfície; talvez um rei anão de pedra em Dorrim.
- Midrath: mais coisas do tema (eclipse como evento raro? marca amaldiçoada que atrai mortos à noite?) — propor antes.

## Fila "para depois"

- **Pedido em 14/10 (noite), para depois**: marcar um ponto no mapa e o personagem andar até lá; modo do
  personagem (agressivo: ataca o que achar no caminho / pacifista), com ícones; lagos e lagoas novos e **pesca**
  (peixes por estação e horário, vender peixe, conquista por pescar em todos os lagos/rios/lagoas e por pegar todos
  os peixes); **bestiário** (cada bicho morto, visto ou domado entra, com histórico curto, fraquezas e dano).

- **Ambiente com cara mais 3D / misturar pixel art** (pedido em 14/10): analisado na resposta — recomendação foi não
  misturar pixel art (quebra a identidade de "tudo é caractere") e sim passar o ambiente pelo mesmo caminho 3D→ASCII
  (árvores, casas, rochas e relevo como modelos/alturas iluminados), começando por um teste na vitrine. Aguardando ele.
- Da lista longa de 14/10 ainda não feito: housing (desenhar a casa com glifos num lote), brasão de guilda 5×3,
  profissões visuais (forja com faíscas, pesca com boia), fonte bitmap estilo CP437/Dwarf Fortress, cache do mapa de
  luz, fogo de tocha que o jogador carrega na invasão.

- **Personagem em 3D** (aprovado e aplicado; hoje é o modo "3D", e o padrão é a Mescla). Como funciona, `cliente/body3d.go`:
  modelo próprio montado com formas arredondadas (cápsula, cone arredondado, caixa) presas a um esqueleto posado a cada
  quadro (`pose3d`: passo, golpes, respiração), desenhado por ray marching em SDF, uma amostra por célula, com luz chave,
  ambiente, especular e luz de contorno; o brilho escolhe o caractere (sempre cheio dentro do corpo). Otimização: cada
  forma projeta um retângulo na tela e o raio só testa as formas que passam por ele (60 FPS com 8 modelos).
  Flags: `-3d` (o seu personagem e os outros jogadores em 3D) e `-vitrine` (as 7 classes em 3D em cima e em 2D embaixo,
  num palco escuro). Prévias em `previas\` (png, gif e mp4). Se ele aprovar: aplicar a inimigos humanoides (mesmo
  `bodyStyle`), depois bichos, montarias e Tiamath (precisam de pose própria) e desenhar os modelos em paralelo; tirar o
  modo 2D de cena só depois disso. Só existe no cliente nativo (no navegador ficaria pesado).

## Pendências conhecidas

- Aranhas pouco visíveis no pântano; elmo/botas/amuleto não aparecem no corpo.
- Equilíbrio nunca foi testado jogando.
- Salvar depende do segredo no navegador: trocar de navegador perde o personagem.
- O modo SSH lê o terreno sem lock (o terreno agora muda: árvores e rochas quebram); corrida benigna, mas existe.

## Sistemas

Acrescentados em 14/10/2026 (lista colada pelo usuário):

- **Terreno que muda** (`nature.go`): árvores do campo, rochas e a parede rachada são `prop`s. `stamp` grava as células
  de pé (`on`) e quebradas (`off`); `settleProps` descarta células que outra construção cobriu. Quebrar chama `edit`,
  que sobe `mapVer` e guarda o lote (até 512). O cliente baixa `/map` com o cabeçalho `X-Map-Ver`, conecta com `mv=`
  e recebe `ed` (índice, caractere, cor, flags) a cada estado; `reload` faz baixar o mapa de novo. O blob do mapa é
  refeito sob demanda (`MapBlob`). Flags: 1 sólido, 2 por cima, 4 água, 8 lama, 16 veneno.
- **Materiais**: árvore larga Madeira, rocha larga Pedra. 6 madeiras viram jangada em qualquer margem (tecla E).
- **Água** (`biomes.go`): rio do Lago Espelhado (a leste do covil) até o mar do sul; pontes onde os caminhos cruzam;
  dois cais (`docks`). Barco: `Player.boat`, anda só em água (`floats`), desce sozinho ao encostar (`landing`). A ilha do
  lago tem baú (Elmo do Lago), só de barco.
- **Relevo**: altura `elev` calculada uma vez; encostas para noroeste clareiam, terra alta vira rocha; onde a altura
  desce um degrau para o sul nasce falésia (sólida), longe de regiões, caminhos e água.
- **Pântano**: os charcos viraram lama andável e lenta (`marsh`=1); o Charco Venenoso (sul do pântano) envenena (`marsh`=2).
- **Status** (veneno, frio, calor): `environ` roda 1x/s — veneno no charco, frio nos Picos longe de fogo, calor no deserto
  de dia (também corta a stamina pela metade). Alguns inimigos aplicam status ao bater (`mobInflict`). Ervas colhidas
  com E curam e protegem: Erva-antídoto (pântano), Raiz-brasa (gelo), Babosa (deserto).
- **Montarias**: Cavalo (nv 1, +2), Javali (nv 3, +1, +10% proteção), Avestruz (nv 5, +2), Velociraptor (nv 8, +3), cada
  uma no seu bioma. E doma (fica salvo em `Steeds`), X monta/desmonta. Portal desmonta.
- **Subsolo**: além da cripta, cavernas (`caveRooms`/`caveTunnels`) com ratos e morcegos, cogumelos que iluminam (luz tipo 3),
  duas bocas no continente, sala secreta atrás da parede rachada (Amuleto das Profundezas) e a gruta do buraco no campo a
  sudeste da vila (queda tira 12 de vida; a corda devolve; Botas do Explorador). No cliente o subsolo é escuro: só se vê
  o que a luz do jogador, as tochas e os cogumelos alcançam.
- **Inimigos**: contagem reduzida (~40%); ratos saíram da vila e moram nas cavernas. `TestBiomas` exige ao menos 4 de cada
  inimigo comum.
- Gerar o mundo leva ~2 s (os testes criam vários mundos; a suíte leva ~30 s).
- **Mapa maior** (14/10, segunda rodada): ele achou que andava pouco e já achava os biomas. `mapScale = 2` multiplica
  as posições das regiões (continente 3072x2400); construções mantêm o tamanho; biomas ficaram ~1,4x maiores. A geração
  das células roda em paralelo (`parallelRows`); o mundo leva ~2,3 s e o mapa empacotado tem ~14 MB.
- **Clima** (`stepWeather` em `nature.go`): tempo bom, chuva e tempestade, trocando a cada 3–6 min; na tempestade caem
  raios (`fxBolt`, 22 de dano em 7 células) perto de quem está ao ar livre. Chuva corta a insolação. Não chove no deserto;
  nos Picos e no inverno vira neve. Flag `-weather clear|rain|storm` fixa o clima (bom para testar).
- **Ambientação 2.5D no cliente** (passes em `frame()`): sombras projetadas de copas/paredes/falésias pelo lado oposto ao
  sol (direção muda com a hora), sombra de cada personagem, capim em lâminas que o vento inclina e que dobra quando alguém
  passa, água quase preta com cristas finas, reflexo espelhado na linha da margem e anéis de ondulação (barco e chuva),
  vinheta, céu escurecendo na chuva, clarão e tremor no raio.
- Conferência visual de 14/10: montarias ainda pequenas e pouco legíveis no zoom padrão (desenho de bicho por cápsulas);
  candidatas a melhorar quando ele reclamar do visual.
