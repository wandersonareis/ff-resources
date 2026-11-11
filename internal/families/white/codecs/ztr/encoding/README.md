# ztr/encoding — texto do ZTR (trilogia FFXIII)

Camada de TEXTO do `.ztr`: converte as linhas cruas (bytes da codepage +
tags) em UTF-8 e de volta. É a única parte do codec que o tradutor vê.

## Modelo

A identidade de uma tag de par é **(classe, B1, B2)**. O nome e o swatch são
apenas display: o mesmo nome pode existir em bytes diferentes entre jogos e
idiomas. O formato é comum a todas as classes:

```
{CLASSE:B1:B2:NOME}   com nome
{CLASSE:B1:B2}        sem nome
```

| caso | texto |
|---|---|
| cor | `{CLR:F9:40:White}` |
| ícone | `{ICON:F0:40:Clock}` |
| botão | `{BTN:F1:40:A}` |
| espaço sem quebra | `{SPACE:85:60}` |
| variável/contador/chave | `{VAR:F4:40}` / `{CTR:F7:40}` / `{KEY:F6:40}` |
| estruturais | `{PAGE:40:70}` / `{NL:40:72}` |
| bytes crus | `{RAW:81:40}` / `{B:FF}` |
| controles | `{END}` `{ESC}` `{TEXT_ITALIC}` `{TEXT_NORMAL}` `{LINE}` `{ART}` `{ART2}` |

Regra: **o hex manda**. O nome é ignorado no encode. `{COLOR:Nome}` e
`{TAB:...}` não são aceitos pelo codec do FF13: a cor exige os dois bytes do
idioma e o antigo “text tab” é um espaço sem quebra.

`{SPACE:85:60}` não é tabulação: é o espaço sem quebra (U+00A0; no francês,
U+202F). O catálogo/editor mostra NBSP ou NNBSP conforme o idioma.

As cores `Ex00..Ex28` são semitransparentes e majoritariamente sem uso no
jogo. O codec as mantém (decode/encode), mas o editor as coloca num grupo
secundário/recolhido; a paleta principal mostra as cores nomeadas.

## Idiomas

Cada ARQUIVO carrega a tabela do seu idioma (codepage). Isso vale para os
ícones, os pares estruturais e principalmente para a **paleta de cores**, cujos
bytes mudam entre LJ/Ch/Kr:

| idioma | codepage | White |
|---|---|---|
| us/jp/fr/de/it/sp | Shift-JIS | `F9:40` |
| ch | Big5 | `FE:40` |
| kr | EUC-KR | `AC:A1` |

- vazio → `us`;
- idioma fora do catálogo → `paleta de cores indisponível para o idioma XX`;
- leitura de arquivo com sufixo fora do catálogo cai em Shift-JIS para o
  charset, mas o catálogo do editor é estrito.

## Pacote `tags`

As tabelas ficam em `encoding/tags`, um arquivo por grupo:

| arquivo | responsabilidade |
|---|---|
| `tag.go` | `Class`, `Tag`, `ByteTag`, `GlyphPair`, `Text()`/`AppendText()` |
| `color.go` | paletas explícitas por jogo × idioma |
| `icon.go` | `SpanSet` dos ícones |
| `button.go` | `SpanSet` dos botões |
| `special.go` | PAGE/NL/SPACE + VAR/CTR/KEY |
| `raw.go` | raw e raw extra |
| `control.go` | controles 0x00..0x05 |
| `glyph.go` | pares que o jogo renderiza como caracteres (runas visíveis) |
| `span.go` | `Span`/`SpanSet` |
| `table.go` | `Tables` + `For(jogo, codepage)` |

Os pares de glifos viram **runas**, não tags: `0x85 0x57` → `—`.

## Pacote `encoding`

| arquivo | responsabilidade |
|---|---|
| `doc.go` | doc do pacote |
| `errors.go` | sentinelas (inclui `ErrPaletteUnavailable`) |
| `identity.go` | aliases de `Codepage`/`GameCode` |
| `language.go` | idioma → codepage, `DefaultLanguage = "us"` |
| `tag.go` | aliases de `Tag`/`Class`, parser da classe |
| `tag_codec.go` | gramática da tag, `closeBrace`, helpers de hex |
| `textcodec.go` | `TextCodec` e a ordem de consulta |
| `indexes.go` | índices (par→Tag, nome→Tag, glifo) |
| `build.go` | composição via `tags.For` |
| `registry.go` | registro de `TextCodec` por (jogo, codepage) |
| `decode.go` | bytes → texto |
| `encode.go` | texto → bytes (por runa, sem a colisão do `{`) |
| `userpairs.go` | área PUA do Shift-JIS (0xF040–0xF9FC → U+E000–U+E757) |
| `charset.go` | Shift-JIS / Big5 / EUC-KR (texto comum) |

## Ordem de consulta

`colors, icons, buttons, glyphs, PAGE, NL, SPACE, VAR, CTR, KEY, RAW` — uma
só ordem para decode e encode. Nenhum par pode existir em duas categorias;
`scripts/tagtables_check` falha se isso quebrar.

## Verificação

```
$ go run ./scripts/tagtables_check
ff131  LatinJapanese  colors=43  icons=54  buttons=33  specials=11  raw=60  raw_extra=10  glyphs=118 controls=6
...
tag tables ok
```

## Garantias

- **round-trip byte-a-byta**: `Encode(Decode(raw)) == raw` para todas as
  linhas dos oráculos em `ztr/testdata` e dos `.ztr` de `apps/ff13/bin/data`;
- **re-empacotamento** do `.ztr` idêntico ao original;
- tag desconhecida **não casa**: o `{` vira literal e o conteúdo volta a ser
  texto — nada é corrompido em silêncio;
- par fora das tabelas em `ICON/BTN/VAR/...` é aceito (hex é autoritativo)
  mas **logado**: o decode seguinte emite `{RAW:..}` e a degradação fica
  visível.

Os oráculos `.txt` em `ztr/testdata` são regenerados com a sintaxe nova
(`UPDATE_ORACLES=1 go test ./internal/families/white/codecs/ztr`); os bytes
de referência (`orig.ztr`, `raw.txt`, `ids.bin`) são os mesmos.
