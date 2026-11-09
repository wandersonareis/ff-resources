package filelist

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WriteTextDir grava a Collection no formato "unpacked filelist" do
// WhiteBinTools (ufl): um #info.txt e um Chunk_i.txt por chunk.
func (c *Collection) WriteTextDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var info strings.Builder
	if c.GameCode == GameFF132 {
		info.WriteString("encrypted: " + strconv.FormatBool(c.Crypt.Encrypted) + "\r\n")
		if c.Crypt.Encrypted {
			info.WriteString("seedA: " + strconv.FormatUint(c.Crypt.SeedA, 10) + "\r\n")
			info.WriteString("seedB: " + strconv.FormatUint(c.Crypt.SeedB, 10) + "\r\n")
			info.WriteString("encryptionTag(DO_NOT_CHANGE): " + strconv.FormatUint(uint64(c.Crypt.EncryptionTag), 10) + "\r\n")
		}
	}
	info.WriteString("fileCount: " + strconv.Itoa(c.FileCount()) + "\r\n")
	info.WriteString("chunkCount: " + strconv.Itoa(c.ChunkCount()) + "\r\n")

	if err := os.WriteFile(filepath.Join(dir, "#info.txt"), []byte(info.String()), 0o644); err != nil {
		return err
	}

	for i := range c.Chunks {
		var sb strings.Builder
		for _, e := range c.Chunks[i].Entries {
			sb.WriteString(strconv.FormatUint(uint64(e.FileCode), 10))
			if c.GameCode.UsesV2Entries() {
				sb.WriteByte('|')
				sb.WriteString(strconv.Itoa(int(e.FileTypeID)))
			}
			sb.WriteByte('|')
			sb.WriteString(e.FileInfo)
			sb.WriteString("\r\n")
		}
		name := filepath.Join(dir, "Chunk_"+strconv.Itoa(i)+".txt")
		if err := os.WriteFile(name, []byte(sb.String()), 0o644); err != nil {
			return err
		}
	}

	return nil
}

// ReadTextDir lê um diretório no formato "unpacked filelist" (rfl) e devolve
// uma Collection.
func ReadTextDir(dir string, g GameCode) (*Collection, error) {
	coll := &Collection{GameCode: g}

	infoLines, err := readAllLines(filepath.Join(dir, "#info.txt"))
	if err != nil {
		return nil, fmt.Errorf("lendo #info.txt: %w", err)
	}
	info := &infoCursor{lines: infoLines}

	var chunkCount int
	if g == GameFF132 {
		encStr, err := info.take("encrypted: ")
		if err != nil {
			return nil, err
		}
		enc, err := strconv.ParseBool(encStr)
		if err != nil {
			return nil, fmt.Errorf("#info.txt: 'encrypted' inválido")
		}
		coll.Crypt.Encrypted = enc
		if enc {
			coll.Crypt.SeedA, err = info.takeUint64("seedA: ")
			if err != nil {
				return nil, err
			}
			coll.Crypt.SeedB, err = info.takeUint64("seedB: ")
			if err != nil {
				return nil, err
			}
			tag, err := info.takeUint32("encryptionTag(DO_NOT_CHANGE): ")
			if err != nil {
				return nil, err
			}
			coll.Crypt.EncryptionTag = tag
		}
		if _, err := info.takeUint32("fileCount: "); err != nil {
			return nil, err
		}
		cc, err := info.take("chunkCount: ")
		if err != nil {
			return nil, err
		}
		chunkCount, err = strconv.Atoi(cc)
		if err != nil {
			return nil, fmt.Errorf("#info.txt: chunkCount inválido")
		}
	} else {
		if _, err := info.takeUint32("fileCount: "); err != nil {
			return nil, err
		}
		cc, err := info.take("chunkCount: ")
		if err != nil {
			return nil, err
		}
		chunkCount, err = strconv.Atoi(cc)
		if err != nil {
			return nil, fmt.Errorf("#info.txt: chunkCount inválido")
		}
	}

	if chunkCount < 0 {
		chunkCount = 0
	}
	coll.Chunks = make([]Chunk, chunkCount)
	for i := range coll.Chunks {
		coll.Chunks[i].ID = uint16(i)
	}

	for i := 0; i < chunkCount; i++ {
		name := filepath.Join(dir, "Chunk_"+strconv.Itoa(i)+".txt")
		lines, err := readAllLines(name)
		if err != nil {
			return nil, fmt.Errorf("lendo Chunk_%d.txt: %w", i, err)
		}
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			entry, err := parseTextEntry(line, g)
			if err != nil {
				return nil, fmt.Errorf("Chunk_%d.txt: %w", i, err)
			}
			coll.Chunks[i].Entries = append(coll.Chunks[i].Entries, entry)
		}
	}

	return coll, nil
}

func parseTextEntry(line string, g GameCode) (Entry, error) {
	var e Entry
	if g.UsesV2Entries() {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			return e, fmt.Errorf("linha inválida (esperado code|type|info): %q", line)
		}
		code, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			return e, fmt.Errorf("fileCode inválido: %q", parts[0])
		}
		ft, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 8)
		if err != nil {
			return e, fmt.Errorf("fileTypeID inválido: %q", parts[1])
		}
		e.FileCode = uint32(code)
		e.FileTypeID = byte(ft)
		e.FileInfo = parts[2]
		return e, nil
	}

	parts := strings.SplitN(line, "|", 2)
	if len(parts) < 2 {
		return e, fmt.Errorf("linha inválida (esperado code|info): %q", line)
	}
	code, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil {
		return e, fmt.Errorf("fileCode inválido: %q", parts[0])
	}
	e.FileCode = uint32(code)
	e.FileInfo = parts[1]
	return e, nil
}

// infoCursor percorre as linhas do #info.txt de forma sequencial e estrita,
// como o WhiteBinTools espera.
type infoCursor struct {
	lines []string
	idx   int
}

func (c *infoCursor) take(prefix string) (string, error) {
	if c.idx >= len(c.lines) || !strings.HasPrefix(c.lines[c.idx], prefix) {
		return "", fmt.Errorf("#info.txt: esperado %q na linha %d", prefix, c.idx)
	}
	v := strings.TrimSpace(strings.TrimPrefix(c.lines[c.idx], prefix))
	c.idx++
	return v, nil
}

func (c *infoCursor) takeUint64(prefix string) (uint64, error) {
	s, err := c.take(prefix)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("#info.txt: valor inválido para %q", prefix)
	}
	return v, nil
}

func (c *infoCursor) takeUint32(prefix string) (uint32, error) {
	s, err := c.take(prefix)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("#info.txt: valor inválido para %q", prefix)
	}
	return uint32(v), nil
}

func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		lines = append(lines, strings.TrimSuffix(sc.Text(), "\r"))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
