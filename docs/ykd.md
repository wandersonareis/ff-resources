# YKD — telas de UI do FF13 (preview e edição de textos)

Documento do app FF13 (`apps/ff13`). O `ykd` não é uma imagem nem um script:
é a **definição 2D de uma tela de menu/HUD** — um grafo de blocos com
transformação, animações nomeadas e uma tabela de sprites.

O caso de uso principal no editor é **aumentar a largura das caixas de texto**
para a tradução caber. A interface do jogo não é refeita: só o campo de largura
da caixa é alterado.

## Como usar no app

1. Abra o app (`apps/ff13/bin/pulse.exe`).
2. **Imagens** → `resident` → `system.win32.xgr` (ou `title0`, `review/…`).
3. Selecione um `.ykd` (ex.: `gr_btlpanel01.ykd`, `gr_mainmenu.ykd`).
4. O painel **Telas (YKD)** abre com o seletor de telas do mesmo container no
   topo (as 38 do `system`), o canvas 1280×720 e a lista de blocos.
5. As **caixas de texto** aparecem como retângulos tracejados com
   `largura×altura · fonte`. Clique em uma.
6. No card **Caixa de texto**, ajuste **Largura** (input ou slider) e clique
   **Aplicar**.
7. O container é gravado em `bin/data/mods/partN/<caminho do container>`. O
   `data/` original não é alterado. O preview recarrega já com a largura nova.

O checkbox **wireframe** desenha os sprites só como contornos — útil para ver
o layout sem as texturas. O checkbox **pivô** alterna o uso do pivô no
posicionamento dos sprites (as caixas de texto sempre usam o pivô como
alinhamento).

## Formato (o que o parser lê)

Tudo no `ykd` é **little endian** (o WPD que o contém é big endian).

```
header (16)
bloco de fundo (0x10)
tabela de blocos -> blocos
tabela de recursos -> recursos
```

### Header

| Offset | Campo |
|---|---|
| 0x00 | magic `"YKD_"` (`0x5F444B59`) |
| 0x04 | `Unknown1` (3 na maioria; 1 na UI de batalha) |
| 0x08 | 4 bytes desconhecidos (constantes por arquivo) |
| 0x0C | dummy (0) |

### Bloco

`Type`, `Index`, `AssociatedIndex` (= nó pai; `0xFFFFFFFF` = raiz),
`Unknown` (0), e **5 × Vector4** float32:

| Vetor | Uso |
|---|---|
| Scale | multiplica o tamanho do recurso |
| Rotation | radianos (X, Y, Z); π nos eixos X/Y vira flip |
| Position | posição em pixels (pai-relativa) |
| Pivot | ponto de rotação/escala/alinhamento |
| Extra | desconhecido (às vezes igual ao pivô) |

Tipos: `0` sprite (1 recurso), `1` grupo, `4` N sprites, `5/6` texto,
`2000/2001` tela/raiz. A cauda do tipo `0`/`4` liga o bloco aos recursos
(`word0` = `Resource.Index`).

### Texto (types 5/6)

A cauda de 48 bytes é:

```
(0,0,0,0)
(largura, altura, fonte, flags)
(cor1, cor2, cor3, modo)
```

- **largura** é o limite de caracteres da caixa — é o campo editável;
- **altura** 0/negativa significa "calculada pela fonte";
- **fonte** é o tamanho (type 5) ou um id/estilo (type 6);
- **flags** ∈ {0,1,4,5}: modo de exibição/alinhamento (não decodificado bit a
  bit; uma eventual rolagem automática estaria aqui);
- **posição** vem do bloco (`Position` + `Pivot`), não da cauda.

### Recurso (sprite)

`Type`, `Index`, `Dummy2`, `Dummy3`, `Name[16]`, viewport:

- `Fragment` (0): `srcX, srcY, srcW, srcH, viewW, viewH, flags, unk5, 4 cores`;
- `Empty` (1): placeholder;
- `Full` (3): 4 floats, view, 4 cores, 2 ints;
- `Extra` (5): Fragment + 16 bytes.

O tamanho final é `view × Scale`. `Name` casa com uma textura `txbh/vtex` do
mesmo container; quando não existe, o backend procura no índice global de
`gui/**/*.win32.xgr` da part.

## Limite de largura (segurança)

O teto é calculado a partir da largura **original em `data/`** (não do mod já
salvo):

```
máx = min( max(original × 2, original + 64), 1280 − posição_esquerda_na_tela )
```

- `original × 2` = até dobrar; `original + 64` cobre caixas pequenas;
- `1280 − esquerda` impede a caixa de passar da borda direita;
- salvar de novo **não aumenta** o teto (a base é sempre o pristine);
- a UI mostra `original`, `atual` e `máx.`; o backend recusa valores fora do
  intervalo.

Isso limita o crescimento, mas **não conhece a moldura do painel**: se a caixa
crescer além do painel do menu, o texto pode desenhar fora dele. O preview
mostra o retângulo e o painel, então dá para conferir antes de injetar.

## Gravação e integridade

- Só o campo de **largura** é alterado (4 bytes, tamanho fixo).
- Offsets e tamanhos do WPD **não mudam**; o container inteiro é regravado em
  `mods/partN/<caminho>` e o registro é reparseado antes da gravação.
- O `data/` original permanece intacto. O `.imgb` pareado continua vindo de
  `data/`; só o `.xgr` é sobrescrito em `mods/`.
- Reverter: apague o `.xgr` do container em `bin/data/mods/...` ou use
  **restaurar** + **Aplicar** para voltar à largura original.

## Limitações atuais

- O **conteúdo** dos textos e a **rolagem automática** não estão no `.ykd`; o
  texto em si vem do sistema de mensagens do jogo. O painel desenha
  placeholders.
- Só a **largura** é editável; altura, fonte, flags, posição e pivô ficam como
  no arquivo.
- As animações `in/loop/out` são lidas pelo parser, mas o preview é estático
  (transform base).
- O fundo 3D/cena externo ao YKD não é renderizado; só a camada 2D.
- O preview foca no conjunto `.win32.xgr` + texturas `txbh/vtex`. `.grs`/DDS
  ainda não entram no pipeline de imagem.
- A primeira abertura de um YKD por part monta o índice de texturas de GUI
  (alguns segundos); depois fica em cache.

## Onde está no código

| Camada | Arquivo |
|---|---|
| Parser | `internal/families/white/codecs/ykd/ykd.go` |
| DTO | `internal/shared/dto/ykd.go` |
| Serviço (preview, texturas, edição, teto) | `internal/families/white/service/ykd.go` |
| Marcação do registro na árvore | `internal/families/white/service/image.go` |
| Painel | `apps/ff13/frontend/src/components/entry-view/ykd-panel/index.tsx` |
| Store/kind/tree | `apps/ff13/frontend/src/components/entry-view/entry-view-store.ts` |
| Contrato compartilhado | `frontend/shared/src/dto.ts`, `api/GameApi.ts`, `app/tree-data.ts` |

Testes: `go test ./internal/families/white/...` e
`apps/ff13/frontend` (`npx vitest run`).
