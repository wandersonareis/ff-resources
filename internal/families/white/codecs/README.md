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
  (FF13 + KEL.DAT do Dirge). O writer sustenta a **inserção de mods** do app
  (append no white bin + reapontamento), além das ferramentas/testes.
- **`whitebin`** — extração e inserção de arquivos no white bin referenciado
  pelo filelist (cópia, descompressão zlib e append alinhado a 2048).
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
- O app escreve no filelist **somente** pelo fluxo explícito de inserção de
  mods (import com "Inserir"): o arquivo vai para o fim do white bin e todas as
  ocorrências do caminho são reapontadas. A leitura do navegador de filelist é
  source-aware — mostra o que está no filelist (com overlay de `mods/` quando
  ligado), nunca o snapshot de `data/`.
- A inserção mantém um manifesto em `mods/partN/filelist-inserts.json` (tamanho
  original dos bins + backup dos filelists) para o "Reverter inserções". Um
  crash entre o append e a gravação do filelist deixa dados órfãos no fim do
  bin, mas os ponteiros antigos continuam válidos.
