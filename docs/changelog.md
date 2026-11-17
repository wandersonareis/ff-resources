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
