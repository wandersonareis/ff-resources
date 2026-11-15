package service

import (
	"ffresources/apps/ff13/backend/formats"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/apps/ff13/backend/parts"
)

// TestImagePreviewRealData usa a árvore extraída do repositório quando ela
// existe. Não falha em CI sem dados: vira skip.
func TestImagePreviewRealData(t *testing.T) {
	root, err := filepath.Abs("bin/data")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "part1", "gui", "resident", "title0.win32.xgr")); err != nil {
		t.Skip("dados do FF13 ausentes")
	}
	t.Setenv("FF_RESOURCES_DATA", root)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	t.Setenv("FF_RESOURCES_GAME_ROOT", "")

	// part3: o filelist que só tem a key do DLC não aparece na árvore.
	if _, err := os.Stat(filepath.Join(root, "part3", "filelist", "filelist_p0000001img_a.win32.bin")); err == nil {
		roots, rerr := (FF13Service{}).ListFilelistRoots("part3")
		if rerr != nil {
			t.Fatalf("ListFilelistRoots(part3): %v", rerr)
		}
		for _, r := range roots {
			if strings.Contains(strings.ToLower(r.Name), "p0000000img") {
				t.Fatalf("filelist só-key não deveria aparecer: %s", r.Name)
			}
		}
	}

	p := parts.Get("part1")
	_ = p
	entries, err := (FF13Service{}).ListDataEntries("part1")
	if err != nil {
		t.Fatalf("ListDataEntries: %v", err)
	}
	var imageEntries int
	for _, e := range entries {
		if e.Image {
			imageEntries++
			if e.ID == "gui/resident/title0.win32.xgr" && !e.ImageContainer {
				t.Fatalf("title0 deveria ser container de imagem")
			}
		}
	}
	if imageEntries == 0 {
		t.Fatal("nenhuma entrada de imagem no inventário de data/")
	}

	ref := formats.EncodeImageRef(formats.ImageRef{
		Source:   "data",
		Path:     "gui/resident/title0.win32.xgr",
		Platform: "win32",
	})
	nodes, err := (FF13Service{}).ListImageContainer("part1", ref)
	if err != nil {
		t.Fatalf("ListImageContainer: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("container sem filhos")
	}
	var imageNode string
	for _, n := range nodes {
		if n.Kind == "image" {
			imageNode = n.ImageRef
			if !strings.Contains(n.Name, "parts") && !strings.Contains(n.Name, "thm_que") {
				t.Logf("imagem: %s", n.Name)
			}
			break
		}
	}
	if imageNode == "" {
		t.Fatalf("nenhuma imagem no container: %+v", nodes)
	}
	entry, err := (FF13Service{}).GetImageEntry("part1", imageNode)
	if err != nil {
		t.Fatalf("GetImageEntry: %v", err)
	}
	if entry.Width == 0 || entry.Height == 0 || entry.PNGData == "" {
		t.Fatalf("prévia vazia: %+v", entry)
	}
	if !strings.HasPrefix(entry.PNGData, "data:image/png;base64,") {
		t.Fatalf("PNGData inválido: %.40s", entry.PNGData)
	}

	// Delete do container + .imgb pareado em uma cópia temporária.
	tmp := t.TempDir()
	copyTree := func(rel string) {
		src := filepath.Join(root, "part1", filepath.FromSlash(rel))
		dst := filepath.Join(tmp, "part1", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, rerr := os.ReadFile(src)
		if rerr != nil {
			t.Fatal(rerr)
		}
		if werr := os.WriteFile(dst, raw, 0o644); werr != nil {
			t.Fatal(werr)
		}
	}
	copyTree("gui/resident/title0.win32.xgr")
	copyTree("gui/resident/title0.win32.imgb")
	t.Setenv("FF_RESOURCES_DATA", tmp)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(tmp, "mods"))
	// Cópia em mods/ para validar o escopo data (default both já é testado
	// abaixo; aqui garantimos que "data" deixa mods intacto).
	for _, rel := range []string{"gui/resident/title0.win32.xgr", "gui/resident/title0.win32.imgb"} {
		src := filepath.Join(tmp, "part1", filepath.FromSlash(rel))
		dst := filepath.Join(tmp, "mods", "part1", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, rerr := os.ReadFile(src)
		if rerr != nil {
			t.Fatal(rerr)
		}
		if werr := os.WriteFile(dst, raw, 0o644); werr != nil {
			t.Fatal(werr)
		}
	}

	ref = formats.EncodeImageRef(formats.ImageRef{Source: "data", Path: "gui/resident/title0.win32.xgr", Platform: "win32"})
	nodes, err = (FF13Service{}).ListImageContainer("part1", ref)
	if err != nil {
		t.Fatalf("ListImageContainer(temp): %v", err)
	}
	imageNode = ""
	for _, n := range nodes {
		if n.Kind == "image" {
			imageNode = n.ImageRef
			break
		}
	}
	if imageNode == "" {
		t.Fatal("sem imagem no container temporário")
	}
	res, err := (FF13Service{}).DeleteImages("part1", []string{imageNode}, "data")
	if err != nil {
		t.Fatalf("DeleteImages(data): %v", err)
	}
	if len(res.Done) != 1 {
		t.Fatalf("delete data: %+v", res)
	}
	for _, rel := range []string{"gui/resident/title0.win32.xgr", "gui/resident/title0.win32.imgb"} {
		if _, err := os.Stat(filepath.Join(tmp, "part1", filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("%s deveria ter sido apagado de data/", rel)
		}
		if _, err := os.Stat(filepath.Join(tmp, "mods", "part1", filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s deveria permanecer em mods/: %v", rel, err)
		}
	}
	res, err = (FF13Service{}).DeleteImages("part1", []string{imageNode}, "both")
	if err != nil {
		t.Fatalf("DeleteImages(both): %v", err)
	}
	if len(res.Done) != 1 {
		t.Fatalf("delete both: %+v", res)
	}
	for _, rel := range []string{"gui/resident/title0.win32.xgr", "gui/resident/title0.win32.imgb"} {
		if _, err := os.Stat(filepath.Join(tmp, "mods", "part1", filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("%s deveria ter sido apagado de mods/", rel)
		}
	}

	// VTEX em .xfv.
	t.Setenv("FF_RESOURCES_DATA", root)
	t.Setenv("FF_RESOURCES_MODS", filepath.Join(root, "mods"))
	xfv := "vfx/ac/ac_eark_011/veffs.us.win32.xfv"
	ref = formats.EncodeImageRef(formats.ImageRef{Source: "data", Path: xfv, Platform: "win32"})
	nodes, err = (FF13Service{}).ListImageContainer("part1", ref)
	if err != nil {
		t.Fatalf("ListImageContainer(xfv): %v", err)
	}
	for _, n := range nodes {
		if n.Kind != "image" {
			continue
		}
		entry, err = (FF13Service{}).GetImageEntry("part1", n.ImageRef)
		if err != nil {
			t.Fatalf("GetImageEntry(xfv/%s): %v", n.Name, err)
		}
		if entry.PNGData == "" {
			t.Fatalf("prévia xfv vazia (%s)", n.Name)
		}
		return
	}
	t.Fatalf("nenhuma imagem no xfv: %+v", nodes)

	// TRB real (c001), quando o sample de referência está disponível.
	trbSrc := `F:\ff13crypt\c001.win32.trb`
	imgbSrc := `F:\ff13crypt\c001.win32.imgb`
	if _, err := os.Stat(trbSrc); err == nil {
		tmpTrb := t.TempDir()
		relTrb := "chr/pc/c001/bin/c001.win32.trb"
		relImgb := "chr/pc/c001/bin/c001.win32.imgb"
		for src, rel := range map[string]string{trbSrc: relTrb, imgbSrc: relImgb} {
			dst := filepath.Join(tmpTrb, "part1", filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatal(err)
			}
			raw, rerr := os.ReadFile(src)
			if rerr != nil {
				t.Fatal(rerr)
			}
			if werr := os.WriteFile(dst, raw, 0o644); werr != nil {
				t.Fatal(werr)
			}
		}
		t.Setenv("FF_RESOURCES_DATA", tmpTrb)
		t.Setenv("FF_RESOURCES_MODS", filepath.Join(tmpTrb, "mods"))
		ref = formats.EncodeImageRef(formats.ImageRef{Source: "data", Path: relTrb, Platform: "win32"})
		nodes, err = (FF13Service{}).ListImageContainer("part1", ref)
		if err != nil {
			t.Fatalf("ListImageContainer(trb): %v", err)
		}
		for _, n := range nodes {
			if n.Kind != "image" {
				continue
			}
			entry, err = (FF13Service{}).GetImageEntry("part1", n.ImageRef)
			if err != nil {
				t.Fatalf("GetImageEntry(trb/%s): %v", n.Name, err)
			}
			if entry.PNGData == "" {
				t.Fatalf("prévia trb vazia (%s)", n.Name)
			}
			return
		}
		t.Fatalf("nenhuma imagem no trb (%d nós)", len(nodes))
	}
}
