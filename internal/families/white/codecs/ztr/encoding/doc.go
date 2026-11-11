// Package encoding reproduz a camada de texto do ZTR da trilogia FFXIII:
// codepage física, pares de 2 bytes que o jogo renderiza como caracteres
// latinos e tags estruturais (cor, ícone, botão, espaço, variáveis...).
//
// Diferenças deliberadas em relação ao decoder/encoder legado:
//   - os pares de glifos são CARACTERES da codepage, não tags: no txt saem
//     sem chaves (0x85 0x57 -> "—"); chaves apenas nas tags estruturais;
//   - o encoder trabalha sobre runas, não sobre bytes da codepage, o que
//     elimina a colisão entre o byte 0x7B ('{') e trails DBCS.
//
// As tabelas de dados ficam no subpacote tags. A identidade de cada tag é o
// par de bytes; nomes e swatches são display. O formato textual é
//
//	{CLASSE:B1:B2[:NOME]}
//
// com o hex sempre autoritativo. Cada arquivo carrega a tabela do SEU idioma
// (codepage), inclusive a paleta de cores, cujos bytes mudam entre LJ/Ch/Kr.
package encoding
