# Não lançado

## FINAL FANTASY XIII — telas de UI (YKD)

- Novo leitor de `.ykd` (header, blocos, transform, animações, recursos e
  caudas) em `internal/families/white/codecs/ykd`.
- Preview de tela 1280×720 no app FF13: sprites com recorte, posição, escala,
  rotação, pivô e flip; caixas de texto; wireframe; seletor das telas do mesmo
  container; destaque e ocultação de camadas.
- Texturas resolvidas no mesmo WPD e, como fallback, no índice global de
  `gui/**/*.win32.xgr` da part.
- Edição da **largura das caixas de texto** para ajustar a tradução, com teto
  de 2× a largura original (ou +64 px) limitado pela borda direita da tela.
  A gravação altera só o campo de largura e grava o container em `mods/partN`;
  `data/` e os offsets do WPD não mudam.
- Documentação do formato e dos limites em [`docs/ykd.md`](ykd.md).

# 1.0.0 — primeiro release

Primeiro release público do editor de recursos de FINAL FANTASY X | X-2 HD
Remaster e FINAL FANTASY XIII Trilogy.

## Formatos de texto compatíveis

**FINAL FANTASY X | X-2 HD Remaster**

- Eventos
- Objetos e sistema
- Dicionário de macros
- Loc Kit
- Painéis de ajuda
- Textos de batalha
- Cloud
- Tutorial
- Menu principal

**FINAL FANTASY XIII Trilogy (part1, part2 e part3)**

- Arquivos de texto ZTR
- Containers WPD que carregam texto ZTR
- Escopos por família de conteúdo: `ac`, `resident`, `zone`, `event` e
  `scenario`

**Comum aos dois jogos**

- Edição por idioma, com os demais idiomas da unidade como referência
  somente-leitura
- Gravação sempre em `mods/`; o original em `data/` não é alterado
- Deduplicação opcional de textos repetidos

## Formatos de imagem compatíveis

**FINAL FANTASY X | X-2 HD Remaster**

- Texturas Phyre (`.dds.phyre`)
- Exportação em DDS e PNG
- Importação de DDS
- Navegação de containers VBF

**FINAL FANTASY XIII Trilogy**

- Texturas IMGB, incluindo as faces residentes do GUI
- Containers de imagem WPD e TRB e as extensões de container suportadas
- Arquivos diretos DDS e PNG
- Pré-visualização e exportação em DDS e PNG
- Importação de DDS em lote de `mods/translated/images/<part>` (pasta =
  caminho do container, arquivo = nome do registro) ou avulsa para a imagem
  aberta, validando dimensões, formato DX e mipmaps

**Comum aos dois jogos**

- Pré-visualização de texturas
- Exportação de imagens sem alterar o original em `data/`

## Tipos de exportação

- JSON
- Strings
- CSV
- XLSX
- PO

A exportação de texto é feita por família de conteúdo ou por unidade
selecionada. Imagens são exportadas em DDS e PNG.
