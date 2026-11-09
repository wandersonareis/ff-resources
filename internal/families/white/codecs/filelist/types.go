// Package filelist implementa leitura, escrita e conversão dos arquivos
// "filelist" da trilogia Final Fantasy XIII (e do KEL.DAT do Dirge of
// Cerberus), portando a lógica de Surihix/WhiteBinTools para Go.
//
// O ponto central é a Collection: uma representação neutra (independente do
// layout binário) do conteúdo de um filelist. Ela pode ser convertida para
// JSON, texto, uma árvore de arquivos em UI, ou qualquer outro formato, sem
// tocar no parser/serializador binário.
//
// # Compatibilidade entre plataformas
//
// O formato do filelist é o MESMO em PC (win32), PS3 e Xbox 360:
//   - cabeçalho de criptografia idêntico (semente de 16 bytes + tamanho em
//     big-endian + tag 0x1DE03478 + 8 bytes zerados);
//   - mesmas entradas de 8 bytes (V2 para ff13-2/LR, V1 para ff13-1/dirge);
//   - mesmos chunks de caminhos comprimidos com zlib.
//
// O que muda entre as plataformas é apenas o CONTEÚDO: os caminhos usam tokens
// "win32." / "ps3." / "x360." e os dados no white bin diferem. Por isso este
// parser/serializador atende às três plataformas sem nenhum modo especial.
// Validado contra filelists reais de PC, PS3 (disco) e Xbox 360 (RGH), com
// decriptação byte-idêntica ao ffxiiicrypt e extração byte-idêntica ao
// WhiteBinTools.
package filelist

import (
	"fmt"
	"strconv"
	"strings"
)

// GameCode identifica o jogo; determina o layout das entradas do filelist e
// se há (ou não) criptografia.
type GameCode int

const (
	// GameDirge é o KEL.DAT do Dirge of Cerberus (layout V1, além do gerador
	// de paths).
	GameDirge GameCode = iota
	// GameFF131 é o Final Fantasy XIII (Steam). Layout V1, sem criptografia.
	GameFF131
	// GameFF132 é o Final Fantasy XIII-2 e o Lightning Returns. Layout V2,
	// com criptografia/checksum no filelist.
	GameFF132
)

func (g GameCode) String() string {
	switch g {
	case GameDirge:
		return "dirge"
	case GameFF131:
		return "ff131"
	case GameFF132:
		return "ff132"
	default:
		return "unknown"
	}
}

// UsesV2Entries informa se o layout usa entradas de 8 bytes no formato V2
// (fileCode + fileInfoPos/chunk + fileTypeID).
func (g GameCode) UsesV2Entries() bool { return g == GameFF132 }

// ParseGameCode aceita "dirge", "ff131", "ff132" (com ou sem '-' inicial).
func ParseGameCode(s string) (GameCode, error) {
	switch strings.ToLower(strings.TrimPrefix(s, "-")) {
	case "dirge":
		return GameDirge, nil
	case "ff131":
		return GameFF131, nil
	case "ff132":
		return GameFF132, nil
	default:
		return 0, fmt.Errorf("game code inválido: %q (esperado: dirge, ff131, ff132)", s)
	}
}

// EncryptionTagConstant é o valor encontrado no offset 20 de um filelist
// criptografado (FilelistCryptHeader.EncryptionTagConstant).
const EncryptionTagConstant uint32 = 0x1DE03478

// Entry é uma entrada do filelist já resolvida: o endereço e o caminho foram
// separados, de forma independente da posição bruta no arquivo.
type Entry struct {
	FileCode uint32
	// FileTypeID só é usado no layout V2 (ff132/LR).
	FileTypeID byte
	// FileInfo é a string "posH:uncmpH:cmpH:caminho" (sem o '\0' final).
	FileInfo string
}

// Chunk agrupa as entradas cujos dados de caminho ficam no mesmo bloco
// comprimido. A ordem das entradas dentro do chunk é preservada.
type Chunk struct {
	ID      uint16
	Entries []Entry
}

// CryptInfo descreve o cabeçalho de criptografia (apenas ff132).
type CryptInfo struct {
	Encrypted     bool
	SeedA         uint64
	SeedB         uint64
	EncryptionTag uint32
}

// MD5HashBytes reconstrói os 16 bytes usados como semente (o "MD5Hash" do
// WhiteBinTools): SeedA e SeedB em little-endian.
func (c CryptInfo) MD5HashBytes() [16]byte {
	var out [16]byte
	putLE64(out[0:8], c.SeedA)
	putLE64(out[8:16], c.SeedB)
	return out
}

// Collection é a representação neutra do conteúdo de um filelist. É o ponto
// de extensão pedido: converta-a para JSON, texto, árvore de UI, outro
// formato, etc. A Collection não guarda offsets binários — eles são
// recalculados ao (re)construir o arquivo.
type Collection struct {
	GameCode GameCode
	Crypt    CryptInfo
	Chunks   []Chunk
}

// FileCount é o número total de entradas.
func (c *Collection) FileCount() int {
	n := 0
	for i := range c.Chunks {
		n += len(c.Chunks[i].Entries)
	}
	return n
}

// ChunkCount é o número de chunks.
func (c *Collection) ChunkCount() int { return len(c.Chunks) }

// EntryRef referencia uma entrada dentro da Collection.
type EntryRef struct {
	Chunk int
	Entry int
}

// EachEntry percorre todas as entradas na ordem global (por chunk, depois
// pela ordem interna do chunk), que é a ordem usada na tabela binária.
func (c *Collection) EachEntry(fn func(ref EntryRef, e *Entry) error) error {
	for ci := range c.Chunks {
		for ei := range c.Chunks[ci].Entries {
			if err := fn(EntryRef{ci, ei}, &c.Chunks[ci].Entries[ei]); err != nil {
				return err
			}
		}
	}
	return nil
}

// FileLocation é a parte de endereço de um FileInfo, já decomposta.
type FileLocation struct {
	// PosSectors é a posição em setores de 2048 bytes (o número em hex que
	// aparece antes do primeiro ':').
	PosSectors uint64
	UncmpSize  uint32
	CmpSize    uint32
	Path       string
	// RawPath preserva a string de caminho exatamente como lida; útil porque
	// arquivos sem path vêm como " " (um espaço).
	RawPath string
}

// IsCompressed informa se os tamanhos indicam compressão zlib.
func (l FileLocation) IsCompressed() bool { return l.UncmpSize != l.CmpSize }

// ParseFileInfo decompõe "posH:uncmpH:cmpH:caminho". Os três primeiros campos
// são hexadecimais; o caminho é o restante (não contém ':').
func ParseFileInfo(s string) (FileLocation, error) {
	parts := strings.SplitN(s, ":", 4)
	if len(parts) < 4 {
		return FileLocation{}, fmt.Errorf("fileInfo com dados insuficientes: %q", s)
	}
	pos, err := parseHexU64(parts[0])
	if err != nil {
		return FileLocation{}, fmt.Errorf("posição inválida em %q: %w", s, err)
	}
	uncmp, err := parseHexU32(parts[1])
	if err != nil {
		return FileLocation{}, fmt.Errorf("tamanho (uncmp) inválido em %q: %w", s, err)
	}
	cmp, err := parseHexU32(parts[2])
	if err != nil {
		return FileLocation{}, fmt.Errorf("tamanho (cmp) inválido em %q: %w", s, err)
	}
	return FileLocation{PosSectors: pos, UncmpSize: uncmp, CmpSize: cmp, Path: parts[3], RawPath: parts[3]}, nil
}

// FormatFileInfo monta a string "posH:uncmpH:cmpH:caminho" (hex minúsculo,
// sem zeros à esquerda), do mesmo jeito que o WhiteBinTools.
func FormatFileInfo(l FileLocation) string {
	return strconv.FormatUint(l.PosSectors, 16) + ":" +
		strconv.FormatUint(uint64(l.UncmpSize), 16) + ":" +
		strconv.FormatUint(uint64(l.CmpSize), 16) + ":" + l.Path
}

func parseHexU64(s string) (uint64, error) { return strconv.ParseUint(strings.TrimSpace(s), 16, 64) }
func parseHexU32(s string) (uint32, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 16, 32)
	return uint32(v), err
}
