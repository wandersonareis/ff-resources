package service

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/shared/dto"
)

// TestImageImportTranslationEndToEnd cobre o caminho completo com os dados do
// repositório: exporta um registro real para o layout novo, copia (alterado)
// para translated/images/<part>/<container-rel>/<registro>.dds, importa e
// confere que SÓ o .imgb foi para mods/ e que o estado passa a "importada".
func TestImageImportTranslationEndToEnd(t *testing.T) {
	dataRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data"))
	if err != nil {
		t.Fatal(err)
	}
	const (
		part         = "part1"
		containerRel = "vfx/menu/resident/resident/veffs.us.win32.xfv"
		recordName   = "v85d9e8d2cd9c47.vtex"
	)
	if _, err := os.Stat(filepath.Join(dataRoot, part, filepath.FromSlash(containerRel))); err != nil {
		t.Skipf("dados do FF13 ausentes: %v", err)
	}
	t.Setenv("FF_RESOURCES_DATA", dataRoot)
	t.Setenv("FF_RESOURCES_GAME_ROOT", "")
	mods := t.TempDir()
	t.Setenv("FF_RESOURCES_MODS", mods)

	var svc Service
	containerRef := EncodeImageRef(ImageRef{
		Source:   "data",
		Path:     containerRel,
		Platform: ImagePlatformOf(containerRel),
	})
	nodes, err := svc.ListImageContainer(part, containerRef)
	if err != nil {
		t.Fatalf("ListImageContainer: %v", err)
	}
	var imageRef string
	for _, node := range nodes {
		if strings.EqualFold(node.Name, recordName) {
			imageRef = node.ImageRef
			break
		}
	}
	if imageRef == "" {
		t.Skipf("registro %s ausente no container", recordName)
	}

	// Export: o caminho tem que ser <part>/<container-rel>/<registro>.dds.
	ddsPath, err := svc.ImageExportPath(part, imageRef, "dds")
	if err != nil {
		t.Fatalf("ImageExportPath: %v", err)
	}
	wantSuffix := filepath.Join(part, filepath.FromSlash(containerRel), recordName+".dds")
	if !strings.HasSuffix(ddsPath, wantSuffix) {
		t.Fatalf("export fora do layout novo:\n got %s\nwant sufixo %s", ddsPath, wantSuffix)
	}
	if err := svc.SaveImage(part, imageRef, "dds", ddsPath); err != nil {
		t.Fatalf("SaveImage: %v", err)
	}
	dds, err := os.ReadFile(ddsPath)
	if err != nil {
		t.Fatalf("lendo DDS exportado: %v", err)
	}
	if len(dds) <= 200 {
		t.Fatalf("DDS exportado pequeno demais (%d bytes)", len(dds))
	}
	// Muda UM byte do payload (mesmo tamanho/formato/mips) para o import
	// produzir um payload diferente do pristine.
	dds[200] ^= 0xFF

	translated := filepath.Join(TranslatedImagesRoot(part), filepath.FromSlash(containerRel), recordName+".dds")
	if err := os.MkdirAll(filepath.Dir(translated), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(translated, dds, 0o644); err != nil {
		t.Fatal(err)
	}

	preview, err := svc.PreviewImportImages(part)
	if err != nil {
		t.Fatalf("PreviewImportImages: %v", err)
	}
	if len(preview.Errors) != 0 {
		t.Fatalf("preview com erros: %v", preview.Errors)
	}
	if len(preview.Entries) != 1 {
		t.Fatalf("preview: %d entradas, esperado 1", len(preview.Entries))
	}
	entry := preview.Entries[0]
	if entry.ValidationError != "" {
		t.Fatalf("entrada inválida: %s", entry.ValidationError)
	}
	if !entry.Differs || !entry.Selected {
		t.Fatalf("entrada deveria nascer marcada e diferente: %+v", entry)
	}

	res, err := svc.ImportImages(part, dto.ImageImportSelection{
		Entries: []dto.ImageImportSelectionEntry{{ID: entry.ID, Image: true}},
	})
	if err != nil {
		t.Fatalf("ImportImages: %v", err)
	}
	if len(res.Updated) != 1 || len(res.Failed) != 0 {
		t.Fatalf("import: %+v", res)
	}

	// Só o .imgb deve ter ido para mods/ — nunca o container.
	payloadRel := ImagePairPath(containerRel)
	modsPayload := filepath.Join(mods, part, filepath.FromSlash(payloadRel))
	if _, err := os.Stat(modsPayload); err != nil {
		t.Fatalf("payload importado ausente em mods: %v", err)
	}
	modsContainer := filepath.Join(mods, part, filepath.FromSlash(containerRel))
	if _, err := os.Stat(modsContainer); !os.IsNotExist(err) {
		t.Fatalf("container não deveria ser copiado para mods (err=%v)", err)
	}

	img, err := svc.GetImageEntry(part, imageRef)
	if err != nil {
		t.Fatalf("GetImageEntry após import: %v", err)
	}
	if !img.Modded {
		t.Fatal("imagem importada não marcada como Modded")
	}
}

// TestImportImageStandaloneEndToEnd confere o "Importar de…": o alvo é a
// imagem aberta e o nome/caminho do .dds escolhido é irrelevante.
func TestImportImageStandaloneEndToEnd(t *testing.T) {
	dataRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data"))
	if err != nil {
		t.Fatal(err)
	}
	const (
		part         = "part1"
		containerRel = "vfx/menu/resident/resident/veffs.us.win32.xfv"
		recordName   = "v85d9e8d2cd9c47.vtex"
	)
	if _, err := os.Stat(filepath.Join(dataRoot, part, filepath.FromSlash(containerRel))); err != nil {
		t.Skipf("dados do FF13 ausentes: %v", err)
	}
	t.Setenv("FF_RESOURCES_DATA", dataRoot)
	t.Setenv("FF_RESOURCES_GAME_ROOT", "")

	var svc Service
	containerRef := EncodeImageRef(ImageRef{Source: "data", Path: containerRel, Platform: ImagePlatformOf(containerRel)})
	nodes, err := svc.ListImageContainer(part, containerRef)
	if err != nil {
		t.Fatalf("ListImageContainer: %v", err)
	}
	var imageRef string
	for _, node := range nodes {
		if strings.EqualFold(node.Name, recordName) {
			imageRef = node.ImageRef
			break
		}
	}
	if imageRef == "" {
		t.Skipf("registro %s ausente", recordName)
	}

	mods := t.TempDir()
	t.Setenv("FF_RESOURCES_MODS", mods)
	exported, err := svc.ImageExportPath(part, imageRef, "dds")
	if err != nil {
		t.Fatalf("ImageExportPath: %v", err)
	}
	if err := svc.SaveImage(part, imageRef, "dds", exported); err != nil {
		t.Fatalf("SaveImage: %v", err)
	}
	dds, err := os.ReadFile(exported)
	if err != nil {
		t.Fatal(err)
	}
	if len(dds) <= 200 {
		t.Fatalf("DDS pequeno demais (%d bytes)", len(dds))
	}
	dds[200] ^= 0xFF
	// Nome e pasta aleatórios: o import avulso ignora os dois.
	picked := filepath.Join(t.TempDir(), "qualquer_nome.dds")
	if err := os.WriteFile(picked, dds, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.ImportImage(part, imageRef, picked, dto.ImageImportOptions{WriteMods: true}); err != nil {
		t.Fatalf("ImportImage: %v", err)
	}
	payloadRel := ImagePairPath(containerRel)
	if _, err := os.Stat(filepath.Join(mods, part, filepath.FromSlash(payloadRel))); err != nil {
		t.Fatalf("payload não gravado em mods: %v", err)
	}
	img, err := svc.GetImageEntry(part, imageRef)
	if err != nil {
		t.Fatalf("GetImageEntry: %v", err)
	}
	if !img.Modded {
		t.Fatal("import avulso não marcou a imagem como Modded")
	}

	// DDS incompatível: erro com as dimensões e nada gravado.
	bad := append([]byte(nil), dds...)
	binary.LittleEndian.PutUint32(bad[16:20], 123)
	badPath := filepath.Join(t.TempDir(), "incompativel.dds")
	if err := os.WriteFile(badPath, bad, 0o644); err != nil {
		t.Fatal(err)
	}
	mods2 := t.TempDir()
	t.Setenv("FF_RESOURCES_MODS", mods2)
	err = svc.ImportImage(part, imageRef, badPath, dto.ImageImportOptions{WriteMods: true})
	if err == nil || !strings.Contains(err.Error(), "dimensões") {
		t.Fatalf("esperava erro de dimensões, veio %v", err)
	}
	if _, serr := os.Stat(filepath.Join(mods2, part, filepath.FromSlash(payloadRel))); !os.IsNotExist(serr) {
		t.Fatalf("import incompatível não deveria gravar (err=%v)", serr)
	}
}

// TestPreviewImportImagesErrors cobre as mensagens do contrato: container
// fora do filelist, registro inexistente e incompatibilidade de dimensões.
func TestPreviewImportImagesErrors(t *testing.T) {
	dataRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data"))
	if err != nil {
		t.Fatal(err)
	}
	const (
		part         = "part1"
		containerRel = "vfx/menu/resident/resident/veffs.us.win32.xfv"
		recordName   = "v85d9e8d2cd9c47.vtex"
	)
	if _, err := os.Stat(filepath.Join(dataRoot, part, filepath.FromSlash(containerRel))); err != nil {
		t.Skipf("dados do FF13 ausentes: %v", err)
	}
	t.Setenv("FF_RESOURCES_DATA", dataRoot)
	t.Setenv("FF_RESOURCES_GAME_ROOT", "")

	var svc Service
	containerRef := EncodeImageRef(ImageRef{Source: "data", Path: containerRel, Platform: ImagePlatformOf(containerRel)})
	nodes, err := svc.ListImageContainer(part, containerRef)
	if err != nil {
		t.Fatalf("ListImageContainer: %v", err)
	}
	var imageRef string
	for _, node := range nodes {
		if strings.EqualFold(node.Name, recordName) {
			imageRef = node.ImageRef
			break
		}
	}
	if imageRef == "" {
		t.Skipf("registro %s ausente", recordName)
	}

	goodRoot := t.TempDir()
	t.Setenv("FF_RESOURCES_MODS", goodRoot)
	ddsPath, err := svc.ImageExportPath(part, imageRef, "dds")
	if err != nil {
		t.Fatalf("ImageExportPath: %v", err)
	}
	if err := svc.SaveImage(part, imageRef, "dds", ddsPath); err != nil {
		t.Fatalf("SaveImage: %v", err)
	}
	goodDDS, err := os.ReadFile(ddsPath)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		rel      string
		mutate   func([]byte)
		contains string
	}{
		{
			name:     "container fora do filelist",
			rel:      "nope/container/veffs.us.win32.xfv/" + recordName + ".dds",
			contains: "não encontrado no filelist",
		},
		{
			name:     "registro inexistente",
			rel:      containerRel + "/nao_existe.vtex.dds",
			contains: "não existe no container",
		},
		{
			name:     "dimensões divergentes",
			rel:      containerRel + "/" + recordName + ".dds",
			mutate:   func(b []byte) { binary.LittleEndian.PutUint32(b[16:20], 123) },
			contains: "dimensões",
		},
		{
			name: "formato divergente",
			rel:  containerRel + "/" + recordName + ".dds",
			mutate: func(b []byte) {
				fourCC := binary.LittleEndian.Uint32(b[84:88])
				other := uint32(0x31545844) // DXT1
				if fourCC == other {
					other = 0x35545844 // DXT5
				}
				binary.LittleEndian.PutUint32(b[84:88], other)
			},
			contains: "formato",
		},
		{
			name: "mipmaps divergentes",
			rel:  containerRel + "/" + recordName + ".dds",
			mutate: func(b []byte) {
				n := binary.LittleEndian.Uint32(b[28:32])
				binary.LittleEndian.PutUint32(b[28:32], n+1)
			},
			contains: "mipmaps",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mods := t.TempDir()
			t.Setenv("FF_RESOURCES_MODS", mods)
			root := TranslatedImagesRoot(part)
			target := filepath.Join(root, filepath.FromSlash(tc.rel))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			raw := append([]byte(nil), goodDDS...)
			if tc.mutate != nil {
				tc.mutate(raw)
			}
			if err := os.WriteFile(target, raw, 0o644); err != nil {
				t.Fatal(err)
			}
			preview, err := svc.PreviewImportImages(part)
			if err != nil {
				t.Fatalf("PreviewImportImages: %v", err)
			}
			if len(preview.Entries) != 1 || preview.Entries[0].ValidationError == "" {
				t.Fatalf("esperava 1 entrada inválida, veio %+v", preview.Entries)
			}
			if !strings.Contains(preview.Entries[0].ValidationError, tc.contains) {
				t.Fatalf("mensagem %q não contém %q", preview.Entries[0].ValidationError, tc.contains)
			}
		})
	}
}
