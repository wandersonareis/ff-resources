package filelist

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// WriteJSON grava a Collection no mesmo formato JSON produzido pelo
// WhiteBinTools (cfj), para que a coleção possa ser reutilizada por outras
// saídas sem depender deste formato.
//
// A conversão é separada da coleção de propósito: a UI pode criar uma árvore
// de arquivos a partir de Collection e ainda usar WriteJSON para o dump bruto.
func (c *Collection) WriteJSON(w io.Writer) error {
	bw := bufio.NewWriter(w)
	const nl = "\r\n"
	line := func(s string) {
		bw.WriteString(s)
		bw.WriteString(nl)
	}

	line("{")
	if c.GameCode == GameFF132 {
		line("  \"encrypted\": " + strconv.FormatBool(c.Crypt.Encrypted) + ",")
		if c.Crypt.Encrypted {
			line("  \"seedA\": " + strconv.FormatUint(c.Crypt.SeedA, 10) + ",")
			line("  \"seedB\": " + strconv.FormatUint(c.Crypt.SeedB, 10) + ",")
			line("  \"encryptionTag(DO_NOT_CHANGE)\": " + strconv.FormatUint(uint64(c.Crypt.EncryptionTag), 10) + ",")
		}
	}
	line("  \"fileCount\": " + strconv.Itoa(c.FileCount()) + ",")
	line("  \"chunkCount\": " + strconv.Itoa(c.ChunkCount()) + ",")
	line("  \"data\": {")

	for i := range c.Chunks {
		line("    \"Chunk_" + strconv.Itoa(i) + "\": [")
		entries := c.Chunks[i].Entries
		for j := range entries {
			e := entries[j]
			line("      {")
			line("        \"fileCode\": " + strconv.FormatUint(uint64(e.FileCode), 10) + ",")
			if c.GameCode == GameFF132 {
				line("        \"fileTypeID\": " + strconv.Itoa(int(e.FileTypeID)) + ",")
			}
			line("        \"fileInfo\": \"" + e.FileInfo + "\"")
			if j == len(entries)-1 {
				line("      }")
			} else {
				line("      },")
			}
		}
		if i == len(c.Chunks)-1 {
			line("    ]")
		} else {
			line("    ],")
		}
	}

	line("  }")
	bw.WriteString("}")
	return bw.Flush()
}

// ReadJSON lê o formato JSON do WhiteBinTools e devolve uma Collection.
// Aceita tanto a chave "fileInfo" (v2.0.0) quanto "filePath" (versões antigas).
func ReadJSON(data []byte, g GameCode) (*Collection, error) {
	lines := strings.Split(string(data), "\n")
	idx := 0
	next := func() (string, bool) {
		for idx < len(lines) {
			l := strings.TrimSpace(strings.TrimSuffix(lines[idx], "\r"))
			idx++
			if l != "" {
				return l, true
			}
		}
		return "", false
	}
	coll := &Collection{GameCode: g}
	chunkCount := -1

	// Cabeçalho.
	for {
		l, ok := next()
		if !ok {
			return nil, fmt.Errorf("json: fim inesperado no cabeçalho")
		}
		if strings.HasPrefix(l, "\"data\"") && strings.HasSuffix(l, "{") {
			break
		}
		key, val, isProp := splitProp(l)
		if !isProp {
			continue
		}
		switch key {
		case "encrypted":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("json: 'encrypted' inválido: %w", err)
			}
			coll.Crypt.Encrypted = b
		case "seedA":
			v, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("json: 'seedA' inválido: %w", err)
			}
			coll.Crypt.SeedA = v
		case "seedB":
			v, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("json: 'seedB' inválido: %w", err)
			}
			coll.Crypt.SeedB = v
		case "encryptionTag(DO_NOT_CHANGE)":
			v, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("json: 'encryptionTag' inválido: %w", err)
			}
			coll.Crypt.EncryptionTag = uint32(v)
		case "chunkCount":
			v, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("json: 'chunkCount' inválido: %w", err)
			}
			chunkCount = v
		}
	}

	if chunkCount < 0 {
		return nil, fmt.Errorf("json: propriedade 'chunkCount' ausente")
	}

	coll.Chunks = make([]Chunk, chunkCount)
	for i := range coll.Chunks {
		coll.Chunks[i].ID = uint16(i)
	}

	// Cada bloco "Chunk_i": [ ... ].
	for i := 0; i < chunkCount; i++ {
		// Avança até a linha de abertura do chunk (tolera lixo/comentários).
		for {
			l, ok := next()
			if !ok {
				return nil, fmt.Errorf("json: chunk %d não encontrado", i)
			}
			if strings.HasPrefix(l, "\"Chunk_"+strconv.Itoa(i)+"\"") {
				break
			}
		}

		for {
			l, ok := next()
			if !ok {
				return nil, fmt.Errorf("json: fim inesperado no chunk %d", i)
			}
			if strings.HasPrefix(l, "]") {
				break
			}
			if !strings.HasPrefix(l, "{") {
				continue
			}

			entry, err := readJSONEntry(next)
			if err != nil {
				return nil, fmt.Errorf("json: chunk %d: %w", i, err)
			}
			coll.Chunks[i].Entries = append(coll.Chunks[i].Entries, entry)
		}
	}

	return coll, nil
}

func readJSONEntry(next func() (string, bool)) (Entry, error) {
	var e Entry
	for {
		l, ok := next()
		if !ok {
			return e, fmt.Errorf("fim inesperado dentro da entrada")
		}
		if l == "}" || l == "}," {
			return e, nil
		}
		key, val, isProp := splitProp(l)
		if !isProp {
			continue
		}
		switch key {
		case "fileCode":
			v, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return e, fmt.Errorf("'fileCode' inválido: %w", err)
			}
			e.FileCode = uint32(v)
		case "fileTypeID":
			v, err := strconv.ParseUint(val, 10, 8)
			if err != nil {
				return e, fmt.Errorf("'fileTypeID' inválido: %w", err)
			}
			e.FileTypeID = byte(v)
		case "fileInfo", "filePath":
			e.FileInfo = val
		}
	}
}

// splitProp interpreta uma linha `"chave": valor` (com vírgula opcional).
func splitProp(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "\"") {
		return "", "", false
	}
	end := strings.IndexByte(line[1:], '"')
	if end < 0 {
		return "", "", false
	}
	key = line[1 : 1+end]
	rest := strings.TrimSpace(line[2+end:])
	if !strings.HasPrefix(rest, ":") {
		return "", "", false
	}
	rest = strings.TrimSpace(rest[1:])
	if strings.HasPrefix(rest, "\"") {
		endv := strings.IndexByte(rest[1:], '"')
		if endv < 0 {
			return "", "", false
		}
		value = rest[1 : 1+endv]
	} else {
		value = strings.TrimSuffix(strings.TrimSpace(rest), ",")
	}
	return key, value, true
}
