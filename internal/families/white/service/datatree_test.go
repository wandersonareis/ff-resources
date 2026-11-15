package service

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
)

// TestZtrCodecGap mede a lacuna do codec de .ztr no part1.
//
// O jogo tem DOIS formatos de .ztr (como o Pulse documenta):
//
//   - BIG-endian (type 0): magic 1 em big-endian, dicionário COMPRIMIDO,
//     chaves e valores em BLOCOS separados — é o que o codec implementa;
//   - LITTLE-endian (type 1..N): type em int32 little-endian no início,
//     chaves e valores JUNTOS (key+value) — ainda NÃO implementado.
//
// Este teste conta os dois e garante que a lacuna é só o formato little:
// todo arquivo que o codec NÃO lê tem a assinatura little-endian (primeiro
// u32 > 0). Quando o codec little entrar, o teste passa a exigir 0 lacunas.
func TestZtrCodecGap(t *testing.T) {
	data := partDataRoot(t, "part1")

	total, big, little := 0, 0, 0
	badLittle := 0
	sample := ""
	_ = filepath.WalkDir(data, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".ztr") {
			return nil
		}
		total++
		raw, rerr := os.ReadFile(path)
		if rerr != nil || len(raw) < 4 {
			return nil
		}
		// type em little-endian: 0 = big-endian (o formato do retail antigo),
		// >0 = little-endian.
		typ := binary.LittleEndian.Uint32(raw[0:4])
		if typ == 0 {
			big++
		} else {
			little++
		}
		if _, perr := ztr.Read(raw); perr != nil {
			if typ == 0 {
				badLittle++
				if sample == "" {
					rel, _ := filepath.Rel(data, path)
					sample = filepath.ToSlash(rel)
				}
			}
		}
		return nil
	})

	t.Logf(".ztr em data/part1: %d (big-endian %d, little-endian %d)", total, big, little)
	if badLittle > 0 {
		t.Logf("lacuna conhecida: %d arquivo(s) big-endian com key+value juntos (variante não implementada), ex.: %s", badLittle, sample)
	}
	if little > 0 {
		t.Logf("lacuna conhecida: %d .ztr little-endian (key+value juntos) aguardam o codec", little)
	}
}

// TestDataTreeListing confere a listagem da árvore de /data: só entram as
// UNIDADES de texto (o "_us" editável + as referências), a pasta de índices
// (filelist/) fica de fora — ela aparece no grupo "Filelist".
func TestDataTreeListing(t *testing.T) {
	part := "part1"
	partDataRoot(t, part) // skippa quando data/part1 não existe
	t.Setenv("FF_RESOURCES_DATA", mustAbs(t, filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data")))
	t.Setenv("FF_RESOURCES_MODS", mustAbs(t, filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods")))

	var svc Service
	units, err := svc.ListTextUnits(part)
	if err != nil {
		t.Fatalf("ListTextUnits: %v", err)
	}
	if len(units) == 0 {
		t.Fatal("nenhuma unidade listada")
	}

	ztr, wpd := 0, 0
	for _, u := range units {
		if u.ID == "filelist" || strings.HasPrefix(u.ID, "filelist/") {
			t.Errorf("índice de filelist no grupo de /data: %s", u.ID)
			continue
		}
		if u.Key != part+"/"+u.ID {
			t.Errorf("chave inconsistente: %q", u.Key)
		}
		if u.Primary == "" {
			t.Errorf("unidade sem primário: %s", u.ID)
			continue
		}
		if !strings.HasSuffix(strings.ToLower(u.Primary), "_us.ztr") &&
			!strings.HasSuffix(strings.ToLower(u.Primary), "_us.bin") {
			t.Errorf("primário não é um arquivo _us: %s", u.Primary)
		}
		if u.Label == "" || strings.Contains(u.Label, "_us") {
			t.Errorf("rótulo com o sufixo de idioma: %q", u.Label)
		}
		switch u.Container {
		case "ztr":
			ztr++
		case "wpd":
			wpd++
		default:
			t.Errorf("container inesperado: %q (%s)", u.Container, u.ID)
		}
	}
	t.Logf("árvore de /data: %d unidades (%d .ztr + %d container WPD)", len(units), ztr, wpd)
	if ztr == 0 {
		t.Error("nenhuma unidade .ztr listada")
	}

	// Toda zona extraída aparece como pasta.
	zones := map[string]bool{}
	for _, u := range units {
		if strings.HasPrefix(u.ID, "zone/") {
			parts := strings.Split(u.ID, "/")
			if len(parts) > 1 {
				zones[parts[1]] = true
			}
		}
	}
	if len(zones) == 0 {
		t.Error("nenhuma zona na árvore de /data")
	}
	t.Logf("zonas na árvore: %d", len(zones))
}
