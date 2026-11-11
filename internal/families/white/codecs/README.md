# Codecs White (filelist, ZTR, WPD, IMGB)

Portes **Go** trazidos de `F:\ff13crypt` para o monorepo `ffresources`.
Nenhum código C# foi copiado; os nomes de pacote e o comportamento dos
originais foram preservados, mudando apenas os caminhos de import.

## Origem → destino

| Origem (`F:\ff13crypt`) | Destino (`ffresources/internal/families/white/codecs`) |
|---|---|
| `filelistgo/backend/core/filelist` | `filelist` |
| `filelistgo/backend/fs/whitebin` | `whitebin` |
| `filelistgo/testdata` | `filelist/testdata` |
| `ztrgo/backend/fs/ztr` | `ztr` |
| `ztrgo/backend/core/encoding` | `ztr/encoding` |
| `ztrgo/backend/fs/ztr/testdata` | `ztr/testdata` |
| `wpdgo/backend/fs/wpd` | `wpd` |
| `imgbgo/backend/core/imgb` | `imgb/core` |
| `imgbgo/backend/fs/imgb` | `imgb/fs` |

Os CLIs (`cmd/*`), scripts, binários e o código C# **não** foram trazidos.

## Pacotes

- **`filelist`** — leitura/decrypt, escrita, JSON, texto e rebuild do filelist
  (FF13 + KEL.DAT do Dirge). O app usa **somente leitura**; o writer existe
  para ferramentas/testes.
- **`whitebin`** — extração de arquivos do white bin referenciado pelo
  filelist (cópia ou descompressão zlib).
- **`ztr`** — container ZTR padrão: header, chunks, line info, modos
  comprimido/não comprimido e as regras de desempate (`TieFirst`/`TieMinLast`).
- **`ztr/encoding`** — camada de texto do ZTR: codepage física, dicionários de
  tags/chaves por jogo/codepage e conversão texto ↔ bytes.
- **`wpd`** — container WPD (header, tabela de registros de 32 bytes e dados).
- **`imgb/core`** — GTEX/DDS, mipmaps, formatos veffs/c001/TRB, PS3 unswizzle.
- **`imgb/fs`** — unpack/repack de IMGB usando um header GTEX.

## Testes

```powershell
go test ./internal/families/white/codecs/... -count=1
```

Cobrem:

- filelist: golden JSON do WhiteBinTools, round-trip binário e criptografia;
- whitebin: extração de um arquivo real (quando o jogo está instalado);
- ZTR: oráculos byte a byte da implementação de referência + round-trip
  comprimido/não comprimido;
- encoding: os 9 pares jogo × codepage e amostras dos dicionários;
- IMGB: amostras PS3/GTEX.

## Notas

- A **compressão ZTR** existe no codec, mas é **experimental** no app: a
  política default é `none`; `auto_yus_us` exige compressão e qualquer grupo
  acima de 256 KB no WPD deve ser comprimido ou recusado.
- O **layout reindexado** `mission`/`system` ainda não está neste pacote; ele
  será um codec separado (`ztrlayout`) para preservar o bloco de chaves do
  `system`.
- O `RepackIMGB` é a parte mais frágil do `imgbgo`; validar com `veffs` e
  `c001` antes de liberar a edição de imagens.
- O filelist é lido do jogo/`/data`; o app não escreve nele. O writer do codec
  não deve ser exposto na UI sem uma decisão explícita.
