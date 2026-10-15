package proto

// Tipos compartilhados entre o servidor e o cliente nativo. Os dois lados
// usam estes nomes; nunca números soltos (um número trocado de um lado só já
// mostrou o ícone errado nas ferramentas).

// Tipos de item (Item.Kind).
const (
	KindPotion = iota
	KindWeapon
	KindSkin // armadura: é a peça que muda a cor da roupa
	KindHelm
	KindBoots
	KindAmulet
	KindMaterial // madeira e pedra de árvores e rochas quebradas
	KindHerb     // erva que cura um status negativo
	KindMoney    // dinheiro caído
	KindScroll   // tônico ou tomo: muda o próprio personagem
	KindKey      // chave de porta trancada
	KindBag      // ampliação da mochila: Bonus é quantos espaços soma
	KindTool     // ferramenta de ambiente: lanterna, tocha, machado
	KindRing     // anel (ainda não existe no jogo; o cliente já tem o ícone)
)

// Efeitos (Fx.Kind).
const (
	FxSlash      = iota // arco do ataque do jogador
	FxNova              // anel que se expande a partir do jogador
	FxWarn              // aviso do pisão do chefe
	FxBlast             // onda do pisão
	FxPuff              // ossos voando quando um esqueleto cai
	FxThrust            // rastro reto: estocada, investida, carga (R é o comprimento)
	FxSpin              // redemoinho do guerreiro
	FxBoom              // explosão de fogo
	FxHeal              // anel de cura
	FxHoly              // julgamento do paladino
	FxMeteor            // aviso do meteoro do mago, que cai no fim
	FxRain              // aviso da chuva de flechas
	FxDark              // nova sombria do necromante
	FxBolt              // raio da tempestade (clima: clarão na tela)
	FxChain             // raio que salta entre alvos: de (X,Y), comprimento R, ângulo Dir
	FxFrost             // nova congelante
	FxFlame             // muralha de chamas: centro, meio comprimento R, ângulo Dir
	FxZap               // raio de magia caindo num ponto (sem clarão)
	FxBlink             // lampejo de quem some ou aparece
	FxBuff              // aura de reforço; Dir é o tipo (0 fúria, 1 proteção, 2 veneno, 3 arcano, 4 mira, 5 sangue)
	FxSmoke             // nuvem de fumaça
	FxBone              // lança de osso em linha
	FxPlague            // nuvem de praga
	FxBlizzard          // nevasca (dura)
	FxCrack             // fenda no chão em linha
	FxConsecrate        // chão consagrado (dura)
	FxHammer            // martelo de luz caindo
	FxGrave             // mãos saindo da cova
	FxStorm             // nuvem da tempestade (dura)
	FxSeal              // Obliterar: os cinco selos se juntando em volta do mago
	FxOblit             // Obliterar: o feixe (de X,Y, comprimento R, ângulo Dir)
	FxGrass             // capim cortado voando (R = quantos tufos)
	FxWhip              // aviso do chicote de Vulkhar: linha em brasa de (X,Y), comprimento R, ângulo Dir
)

// Projéteis (Proj.Kind).
const (
	ProjMissile = iota
	ProjFireball
	ProjArrow
	ProjShadow
	ProjIce
	ProjHoly
)
