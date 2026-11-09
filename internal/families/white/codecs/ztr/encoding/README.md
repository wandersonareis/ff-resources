# ztr/encoding — texto do ZTR (trilogia FFXIII)

Camada de TEXTO do `.ztr`: converte as linhas cruas (bytes da codepage +
tags) em UTF-8 e de volta. É a única parte do codec que o tradutor vê.

## De onde veio

Porte do `ztrgo/backend/core/encoding`, que por sua vez espelhava os
`KeysDecoder/KeysEncoder` do **ZTRtool C#**. As tabelas eram geradas dos
`KeyDicts*.cs` e os nomes das tags saíam de lá: `{Color White}`,
`{Icon Clock}`, `{Entity 1}`, `{Unk81 40}`.

## O que mudou (refatoração)

O nome não era identidade: `{Icon Shield}` é `F2:63` no ff132 e `F2:5C` no
ff133. A sintaxe nova segue o conversor do FFX (`converter/ParseCommand`):

| antes (ZTRtool) | agora | por quê |
|---|---|---|
| `{Color White}` | `{COLOR:White}` | nome resolve o byte (paleta compartilhada) |
| `{Icon Clock}` | `{ICON:F0:40:Clock}` | hex autoritativo + nome de display |
| `{Entity 1}` | `{VAR:F4:40}` | hex puro, **sem nome** |
| `{Counter Type 1}` | `{CTR:F7:40}` | idem |
| `{Key Entity}` | `{KEY:F6:40}` | idem |
| `{TEXT_NEW_PAGE}` | `{PAGE:40:70}` | idem |
| `{TEXT_NEW_LINE}` | `{NL:40:72}` | idem |
| `{TEXT_TAB}` | `{TAB:85:60}` | idem |
| `{Unk81 40}` / `{ExChara85 80}` | `{RAW:81:40}` | bytes crus preservados |
| `{FF}` | `{B:FF}` | byte isolado |
| `{End}` `{Escape}` `{StraightLine}` `{Article}` `{ArticleMany}` | `{END}` `{ESC}` `{LINE}` `{ART}` `{ART2}` | controles de byte único |

Regra: **onde há hex, o hex manda** — o nome é display e é ignorado no
encode. `{COLOR:...}` é a exceção: a paleta é fechada (31–43 nomes) e a
mesma cor muda de byte entre codepages, então o nome resolve.

### Formato

```
{COLOR:White}          nome → byte (por jogo/codepage)
{ICON:F0:40:Clock}     par + nome (só ICON/BTN têm nome)
{BTN:F1:40:A}
{VAR:F4:40} {CTR:F7:40} {KEY:F6:40}
{PAGE:40:70} {NL:40:72} {TAB:85:60} {RAW:81:40} {B:FF}
{END} {ESC} {TEXT_ITALIC} {TEXT_NORMAL} {LINE} {ART} {ART2}
```

## Arquivos

| arquivo | responsabilidade |
|---|---|
| `tag.go` | modelo `Tag` (classe, bytes, nome), `Text()`/`AppendText()`, classes |
| `decode.go` | bytes → texto (a máquina de estados do C#) |
| `encode.go` | texto → bytes (por runa, sem a colisão do `{`) |
| `tag_codec.go` | parse da tag: gramática, `closeBrace`, helpers de hex |
| `key_tables.go` | tabelas geradas (`emit_key_tables`) |
| `spans.go` | `SpanSet`: a forma **compartilhada** das tabelas |
| `indexes.go` | índices (par→Tag, nome→Tag, chara) e a ordem de consulta |
| `build.go` | compõe o `KeySet` de cada (jogo, codepage) |
| `userpairs.go` | área PUA do Shift-JIS (0xF040–0xF9FC → U+E000–U+E757) |
| `charset.go` | Shift-JIS / Big5 / EUC-KR (texto comum) |

## Ordem de consulta

`color, icon, btn, chara, special, raw, raw2` — **uma só ordem** para o
decode e o encode. O C# tinha duas (exchara e special trocados); a
unificação é segura porque nenhum par de bytes existe em duas categorias,
e `scripts/keytables_check` falha se isso quebrar.

O `chara` fica no meio (entre Btn e Special), como no C#. Os pares chara
viram **runas**, não tags: `0x85 0x57` → `—`.

## Tabelas compartilhadas

Antes cada tag era repetida por jogo/codepage (1.210 linhas). Agora:

```go
var xiii2Colors = SpanSet{
    Names: []string{"White", "IceBlue", ...},
    Spans: map[Codepage][]Span{
        LatinJapanese: {{Lead: 0xF9, Trail: 0x40, Count: 31, At: 0}},
        Chinese:       {{Lead: 0xFE, Trail: 0x40, Count: 31, At: 0}},
        Korean:        {{Lead: 0xAC, Trail: 0xA1, Count: 31, At: 0}},
    },
}
```

Um nome, três pares. O `Span` (lead, primeiro trail, contagem, índice em
`Names`) também absorve os **buracos** da tabela — o ff131 tem trails
ausentes, e cada faixa contígua é um span.

Composição por (jogo, codepage): `scripts/keytables_check` imprime a
tabela e valida as invariantes.

```
$ go run ./scripts/keytables_check
ff131  LatinJapanese  color=43  icon=54  btn=33  special=11  raw=60  raw2=10  chara=118 single=6
ff132  LatinJapanese  color=31  icon=88  btn=33  special=11  raw=60  raw2=10  chara=118 single=6
ff133  LatinJapanese  color=31  icon=90  btn=35  special=11  raw=60  raw2=10  chara=118 single=6
key tables ok
```

Chinês não consulta `unk2` (`Raw2 = nil`), como no C#.

## Garantias

- **round-trip byte-a-byta**: `Encode(Decode(raw)) == raw` para todas as
  linhas dos oráculos em `ztr/testdata` e dos `.ztr` de `apps/ff13/bin/data`;
- **re-empacotamento** do `.ztr` idêntico ao original;
- tag desconhecida **não casa**: o `{` vira literal e o conteúdo volta a ser
  texto — nada é corrompido em silêncio;
- par fora das tabelas em `ICON/BTN/VAR/...` é aceito (hex é autoritativo)
  mas **logado**: o decode seguinte emite `{RAW:..}` e a degradação fica
  visível.

Os oráculos `.txt` em `ztr/testdata` foram regenerados com a sintaxe nova
(`UPDATE_ORACLES=1 go test ./internal/families/white/codecs/ztr`); os bytes
de referência (`orig.ztr`, `raw.txt`, `ids.bin`) são os mesmos.
