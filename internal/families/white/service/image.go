package service

import (
	"bytes"
	imgbcore "ffresources/internal/families/white/codecs/imgb/core"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/imagery"
	"ffresources/internal/families/white/profile/common"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/dto"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

func imageSourceLabel(part string, r ImageRef) dto.Metadata {
	return dto.Metadata{
		Key: part + "/" + strings.TrimPrefix(r.Path, "/"),
		ID:  strings.TrimPrefix(r.Path, "/"),
	}
}

func imageBaseName(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		p = p[i+1:]
	}
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[:i]
	}
	return p
}

// imageOutputBase é o nome do ARQUIVO DIRETO exportado (sem extensão): o
// nome original, sem a extensão de origem. Imagens de container usam a pasta
// do container + o nome do registro (ver imageExportRelative).
func imageOutputBase(r ImageRef) string {
	return imageBaseName(r.Path)
}

// imageFileInMods diz se o arquivo interno existe em mods/partN.
func imageFileInMods(part, rel string) bool {
	if rel == "" {
		return false
	}
	mods, err := config.PartRoot("mods", part)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(mods, filepath.FromSlash(rel)))
	return err == nil
}

// imageRecordInMods diz se o REGISTRO da imagem já tem bytes diferentes em
// mods/. O container compartilha um .imgb entre vários registros, então a
// simples existência do arquivo em mods não basta: comparamos só os mips
// deste registro com o pristine de data/ (ou o white bin).
func imageRecordInMods(part string, r ImageRef) bool {
	sec, table, err := ImageSection(part, r)
	if err != nil {
		return false
	}
	payloadRel := r.Payload
	if payloadRel == "" {
		payloadRel = ImagePairPath(table.Rel)
	}
	payloadRel = normalizeImageRel(payloadRel)
	if payloadRel == "" {
		return false
	}
	mods, err := config.PartRoot("mods", part)
	if err != nil {
		return false
	}
	modded, err := os.ReadFile(filepath.Join(mods, filepath.FromSlash(payloadRel)))
	if err != nil {
		return false
	}
	pristineFile, ok := resolveRelFilePristine(part, payloadRel)
	if !ok {
		// Só existe em mods/: é uma importação por definição.
		return true
	}
	pristine, err := pristineFile.Read()
	if err != nil {
		return true
	}
	gtex, err := imgbcore.SectionGTEX(sec)
	if err != nil {
		return true
	}
	mips, err := imgbcore.ParseMipInfoTable(sec, gtex)
	if err != nil {
		return true
	}
	for _, m := range mips {
		start := int(m.Offset)
		end := start + int(m.Size)
		if start < 0 || end > len(modded) || end > len(pristine) {
			return true
		}
		if !bytes.Equal(modded[start:end], pristine[start:end]) {
			return true
		}
	}
	return false
}

// canonicalImageRel devolve o caminho interno (relativo a data/partN) da
// tabela/arquivo da ref, resolvendo o índice do filelist quando a ref veio
// da árvore de filelist.
func canonicalImageRel(part string, r ImageRef) string {
	rel := r.Path
	if r.Root != "" {
		if _, zone, err := ResolveFilelistIndex(parts.Get(part), r.Root); err == nil {
			rel = FilelistDataPath(zone, r.Path)
		}
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	return strings.TrimPrefix(rel, "/")
}

// sanitizeImageName remove só os caracteres inválidos em nomes de arquivo do
// Windows; pontos (nome.ext do registro) são preservados.
func sanitizeImageName(name string) string {
	return strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	).Replace(name)
}

// imageRecordFileName devolve o nome do registro como a árvore exibe:
// "nome.ext" no WPD e "N.tipo" no TRB (fallback N).
func imageRecordFileName(part string, r ImageRef) string {
	entry := strings.TrimSpace(r.Entry)
	switch {
	case strings.HasPrefix(entry, "rec:"):
		name := strings.TrimPrefix(entry, "rec:")
		if name != "" {
			return sanitizeImageName(name)
		}
	case strings.HasPrefix(entry, "trb:"):
		n := strings.TrimPrefix(entry, "trb:")
		if sec, _, err := ImageSection(part, r); err == nil {
			if sh, herr := imgbcore.ParseSectionHeader(sec); herr == nil {
				return sanitizeImageName(n + "." + sh.TypeName())
			}
		}
		return sanitizeImageName(n)
	}
	return sanitizeImageName(entry)
}

// imageExportRelative devolve o caminho de exportação SEM a extensão do
// formato:
//
//	arquivo direto → <pasta>/<nome original>
//	container      → <container-rel>/<nome do registro>
//
// O import de translated/images espelha exatamente esse caminho.
func imageExportRelative(part string, r ImageRef) (string, error) {
	rel := canonicalImageRel(part, r)
	if rel == "" || strings.Contains(rel, "..") || filepath.IsAbs(filepath.FromSlash(rel)) {
		return "", fmt.Errorf("caminho da imagem inválido: %s", rel)
	}
	if r.Entry == "" {
		dir := filepath.Dir(filepath.FromSlash(rel))
		if dir == "." {
			dir = ""
		}
		return filepath.Join(dir, imageOutputBase(r)), nil
	}
	record := imageRecordFileName(part, r)
	if record == "" || record == "." || record == ".." || strings.ContainsAny(record, `/\`) {
		return "", fmt.Errorf("nome do registro inválido: %q", record)
	}
	return filepath.Join(filepath.FromSlash(rel), record), nil
}

// GetImageEntry devolve a prévia PNG + metadados da imagem/ref.
func (Service) GetImageEntry(part, ref string) (dto.ImageEntry, error) {
	part = config.NormalizePart(part)
	r, err := DecodeImageRef(ref)
	if err != nil {
		return dto.ImageEntry{}, err
	}

	// Arquivo direto .dds/.png.
	if r.Entry == "" && !IsContainerExt(r.Path) {
		f, ferr := ResolveImageFile(part, r)
		if ferr != nil {
			return dto.ImageEntry{}, ferr
		}
		raw, rerr := f.Read()
		if rerr != nil {
			return dto.ImageEntry{}, rerr
		}
		if IsResidentFaceImage(r.Path) {
			dds, info, ferr := ResidentFaceDDS(raw)
			if ferr != nil {
				return dto.ImageEntry{}, ferr
			}
			png, perr := imagery.DDSToPNG(dds)
			if perr != nil {
				return dto.ImageEntry{}, perr
			}
			return dto.ImageEntry{
				Metadata:  imageSourceLabel(part, r),
				AssetPath: r.Path,
				Source:    "imgb",
				Format:    info.Format,
				Width:     info.Width,
				Height:    info.Height,
				PNGData:   ImageDataURL("image/png", png),
				Flipped:   false,
				Modded:    imageFileInMods(part, f.Rel),
			}, nil
		}
		ext := strings.ToLower(filepath.Ext(r.Path))
		switch ext {
		case ".dds":
			png, perr := imagery.DDSToPNG(raw)
			if perr != nil {
				return dto.ImageEntry{}, perr
			}
			info, _ := imagery.InfoDDS(raw)
			return dto.ImageEntry{
				Metadata:  imageSourceLabel(part, r),
				AssetPath: r.Path,
				Source:    "dds",
				Format:    info.Format,
				Width:     info.Width,
				Height:    info.Height,
				PNGData:   ImageDataURL("image/png", png),
				Flipped:   false,
				Modded:    imageFileInMods(part, f.Rel),
			}, nil
		case ".png":
			cfg, _, derr := image.DecodeConfig(bytes.NewReader(raw))
			if derr != nil {
				cfg.Width, cfg.Height = 0, 0
			}
			return dto.ImageEntry{
				Metadata:  imageSourceLabel(part, r),
				AssetPath: r.Path,
				Source:    "png",
				Format:    "PNG",
				Width:     uint32(cfg.Width),
				Height:    uint32(cfg.Height),
				PNGData:   ImageDataURL("image/png", raw),
				Flipped:   false,
				Modded:    imageFileInMods(part, f.Rel),
			}, nil
		}
		return dto.ImageEntry{}, fmt.Errorf("imagem: extensão não suportada (%s)", ext)
	}

	dds, info, derr := DecodeImageDDS(part, r, false)
	if derr != nil {
		return dto.ImageEntry{}, derr
	}
	png, perr := imagery.DDSToPNG(dds)
	if perr != nil {
		return dto.ImageEntry{}, perr
	}
	return dto.ImageEntry{
		Metadata:       imageSourceLabel(part, r),
		AssetPath:      r.Path,
		Source:         "imgb",
		Format:         info.Format,
		Width:          info.Width,
		Height:         info.Height,
		MipmapCount:    info.MipmapCount,
		MaxMipmapLevel: info.MipmapCount,
		PNGData:        ImageDataURL("image/png", png),
		Flipped:        false,
		Modded:         imageRecordInMods(part, r),
		Duplicates:     nil,
	}, nil
}

// ImageExists responde se a ref ainda resolve.
func (Service) ImageExists(part, ref string) (bool, error) {
	r, err := DecodeImageRef(ref)
	if err != nil {
		return false, nil
	}
	if r.Entry == "" && !IsContainerExt(r.Path) {
		f, ferr := ResolveImageFile(config.NormalizePart(part), r)
		return ferr == nil && f.Exists(), nil
	}
	if _, _, err := ImageSection(config.NormalizePart(part), r); err != nil {
		return false, nil
	}
	return true, nil
}

// ImageDiagnostics devolve formato/dimensões sem decodificar a imagem.
func (Service) ImageDiagnostics(part, ref string) (imgbcore.ImageInfo, error) {
	r, err := DecodeImageRef(ref)
	if err != nil {
		return imgbcore.ImageInfo{}, err
	}
	return Diagnostics(config.NormalizePart(part), r)
}

// SaveImage grava .dds ou .png no caminho escolhido.
func (Service) SaveImage(part, ref, format, destPath string) error {
	part = config.NormalizePart(part)
	r, err := DecodeImageRef(ref)
	if err != nil {
		return err
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if r.Entry == "" && !IsContainerExt(r.Path) {
		f, ferr := ResolveImageFile(part, r)
		if ferr != nil {
			return ferr
		}
		raw, rerr := f.Read()
		if rerr != nil {
			return rerr
		}
		if IsResidentFaceImage(r.Path) {
			dds, _, ferr := ResidentFaceDDS(raw)
			if ferr != nil {
				return ferr
			}
			out := dds
			if format == "png" {
				png, perr := imagery.DDSToPNG(dds)
				if perr != nil {
					return perr
				}
				out = png
			} else if format != "dds" {
				return fmt.Errorf("formato %q inválido (esperado dds ou png)", format)
			}
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(destPath, out, 0o644)
		}
		ext := strings.ToLower(filepath.Ext(r.Path))
		var out []byte
		switch format {
		case "dds":
			if ext != ".dds" {
				return fmt.Errorf("imagem %s não tem DDS de origem", r.Path)
			}
			out = raw
		case "png":
			if ext == ".png" {
				out = raw
			} else if ext == ".dds" {
				png, perr := imagery.DDSToPNG(raw)
				if perr != nil {
					return perr
				}
				out = png
			} else {
				return fmt.Errorf("formato de origem não suportado: %s", ext)
			}
		default:
			return fmt.Errorf("formato %q inválido (esperado dds ou png)", format)
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destPath, out, 0o644)
	}
	switch format {
	case "dds":
		dds, _, derr := DecodeImageDDS(part, r, true)
		if derr != nil {
			return derr
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destPath, dds, 0o644)
	case "png":
		dds, _, derr := DecodeImageDDS(part, r, false)
		if derr != nil {
			return derr
		}
		png, perr := imagery.DDSToPNG(dds)
		if perr != nil {
			return perr
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destPath, png, 0o644)
	}
	return fmt.Errorf("formato %q inválido (esperado dds ou png)", format)
}

// ImageExportPath devolve o caminho padrão de exportação em mods/edits/images:
// <part>/<container-rel>/<registro>.<fmt> para imagens de container (o mesmo
// caminho que o import de translated/images espera) e o layout antigo com o
// nome original para arquivos diretos.
func (Service) ImageExportPath(part, ref, format string) (string, error) {
	r, err := DecodeImageRef(ref)
	if err != nil {
		return "", err
	}
	part = config.NormalizePart(part)
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "dds" && format != "png" {
		return "", fmt.Errorf("formato %q inválido (esperado dds ou png)", format)
	}
	root, err := config.DataRoot("mods")
	if err != nil {
		return "", err
	}
	rel, err := imageExportRelative(part, r)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "edits", "images", part, rel+"."+format), nil
}

// ExtractImageSelection grava DDS+PNG das refs em destRoot.
func (Service) ExtractImageSelection(part string, refs []string, destRoot string) (dto.BatchResult, error) {
	part = config.NormalizePart(part)
	res := dto.BatchResult{Done: []string{}, Failed: []string{}, Total: len(refs)}
	for _, ref := range refs {
		r, err := DecodeImageRef(ref)
		if err != nil {
			res.Failed = append(res.Failed, ref+": "+err.Error())
			continue
		}
		target := destRoot
		if strings.TrimSpace(target) == "" {
			root, rerr := config.DataRoot("mods")
			if rerr != nil {
				res.Failed = append(res.Failed, ref+": "+rerr.Error())
				continue
			}
			target = filepath.Join(root, "edits", "images", part)
		}
		rel, rerr := imageExportRelative(part, r)
		if rerr != nil {
			res.Failed = append(res.Failed, ref+": "+rerr.Error())
			continue
		}
		dir := filepath.Join(target, filepath.Dir(filepath.FromSlash(rel)))
		name := filepath.Base(rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			res.Failed = append(res.Failed, ref+": "+err.Error())
			continue
		}
		ok := true
		for _, format := range []string{"dds", "png"} {
			if serr := (Service{}).SaveImage(part, ref, format, filepath.Join(dir, name+"."+format)); serr != nil {
				res.Failed = append(res.Failed, ref+": "+serr.Error())
				ok = false
				break
			}
		}
		if ok {
			res.Done = append(res.Done, ref)
		}
	}
	return res, nil
}

// DeletePlanFor monta a lista de arquivos que o delete apagaria.
func (Service) ImageDeletePlans(part string, refs []string) ([]common.ImageDeletePlan, error) {
	part = config.NormalizePart(part)
	plans := make([]common.ImageDeletePlan, 0, len(refs))
	for _, ref := range refs {
		r, err := DecodeImageRef(ref)
		if err != nil {
			return nil, err
		}
		if r.Source == "filelist" {
			return nil, fmt.Errorf("delete de imagem disponível apenas na árvore de data/")
		}
		plan := common.ImageDeletePlan{ID: ref, Label: r.Path}
		var tableRel, pairRel string
		if r.Entry == "" && !IsContainerExt(r.Path) {
			tableRel = r.Path
		} else {
			table, terr := ResolveImageFile(part, r)
			if terr != nil {
				return nil, terr
			}
			tableRel = table.Rel
			pairRel = r.Payload
			if pairRel == "" {
				pairRel = ImagePairPath(tableRel)
			}
			// Conta registros para o aviso.
			if raw, rerr := table.Read(); rerr == nil {
				switch ContainerKind(r.Path, raw) {
				case "wpd":
					if wf, werr := wpd.Read(raw); werr == nil {
						plan.RecordCount = len(wf.Header.Records)
					}
				case "trb":
					if h, _, terr := imgbcore.ParseTRB(raw); terr == nil {
						plan.RecordCount = len(h.Entries)
					}
				}
			}
		}
		add := func(rel, kind string) {
			if rel == "" {
				return
			}
			for _, scope := range []string{"data", "mods"} {
				root, rerr := config.PartRoot(scope, part)
				if rerr != nil {
					continue
				}
				disk := filepath.Join(root, filepath.FromSlash(rel))
				if st, serr := os.Stat(disk); serr == nil && !st.IsDir() {
					plan.Files = append(plan.Files, common.ImageDeleteFile{
						Path:  disk,
						Scope: scope,
						Kind:  kind,
						Size:  st.Size(),
					})
				}
			}
		}
		add(tableRel, "container")
		add(pairRel, "payload")
		if plan.RecordCount > 1 {
			plan.Warning = fmt.Sprintf("o container tem %d registros; apagar esta imagem apaga todos", plan.RecordCount)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// DeleteImages apaga as tabelas/pares/diretos no escopo escolhido
// (data, mods ou both). O default da UI é both.
func (Service) DeleteImages(part string, refs []string, scope string) (dto.BatchResult, error) {
	part = config.NormalizePart(part)
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = "both"
	}
	switch scope {
	case "data", "mods", "both":
	default:
		return dto.BatchResult{}, fmt.Errorf("escopo %q inválido (data, mods ou both)", scope)
	}
	plans, err := (Service{}).ImageDeletePlans(part, refs)
	if err != nil {
		return dto.BatchResult{}, err
	}
	res := dto.BatchResult{Done: []string{}, Failed: []string{}, Total: len(refs)}
	for _, plan := range plans {
		ok := true
		for _, f := range plan.Files {
			if scope != "both" && f.Scope != scope {
				continue
			}
			if rerr := os.Remove(f.Path); rerr != nil && !os.IsNotExist(rerr) {
				res.Failed = append(res.Failed, f.Path+": "+rerr.Error())
				ok = false
			}
		}
		if ok {
			res.Done = append(res.Done, plan.ID)
		}
	}
	invalidateCollection(part)
	InvalidateImageFileIndex(part)
	return res, nil
}

// DeleteImageSelection é o alias usado pelo contrato GameApi.
func (Service) DeleteImageSelection(part string, refs []string, scope string) (dto.BatchResult, error) {
	return (Service{}).DeleteImages(part, refs, scope)
}

func (Service) RefreshImageDuplicates(part string) {}

// ListImageContainer lista os filhos de uma tabela WPD/TRB. Imagens ganham
// ImageRef; os demais registros viram arquivos comuns.
func (Service) ListImageContainer(part, ref string) ([]common.ImageNode, error) {
	r, err := DecodeImageRef(ref)
	if err != nil {
		return nil, err
	}
	if r.Entry != "" {
		return nil, fmt.Errorf("imagem: nó não é um container")
	}
	f, err := ResolveImageFile(config.NormalizePart(part), r)
	if err != nil {
		return nil, err
	}
	raw, err := f.Read()
	if err != nil {
		return nil, err
	}
	kind := ContainerKind(r.Path, raw)
	if kind == "" {
		return nil, fmt.Errorf("imagem: formato de container desconhecido (%s)", r.Path)
	}
	pairRel := r.Payload
	if pairRel == "" {
		pairRel = ImagePairPath(f.Rel)
	}

	nodes := make([]common.ImageNode, 0, 32)
	switch kind {
	case "wpd":
		wf, err := wpd.Read(raw)
		if err != nil {
			return nil, err
		}
		for _, rec := range wf.Header.Records {
			entry := fmt.Sprintf("rec:%s.%s", rec.Name, rec.Extension)
			node := common.ImageNode{
				ID:   EncodeImageRef(ImageRef{Source: r.Source, Root: r.Root, Path: r.Path, Entry: entry, Payload: pairRel, Platform: ImagePlatformOf(r.Path)}),
				Name: rec.Name + "." + rec.Extension,
				Path: entry,
				Size: rec.Size,
				Kind: "file",
				Pair: pairRel,
			}
			if IsImageRecordExt(rec.Extension) {
				if sec, ok := wf.ExtractRecord(rec.Name, rec.Extension); ok {
					if _, gerr := imgbcore.SectionGTEX(sec); gerr == nil {
						node.Kind = "image"
						node.ImageRef = node.ID
						node.Container = "wpd"
					}
				}
			}
			nodes = append(nodes, node)
		}
	case "trb":
		h, entries, err := imgbcore.ParseTRB(raw)
		if err != nil {
			return nil, err
		}
		for i, e := range entries {
			sec, serr := h.SectionData(raw, e)
			name := fmt.Sprintf("%d", i)
			if serr == nil {
				if sh, herr := imgbcore.ParseSectionHeader(sec); herr == nil {
					name = fmt.Sprintf("%d.%s", i, sh.TypeName())
				}
			}
			entry := fmt.Sprintf("trb:%d", i)
			node := common.ImageNode{
				ID:   EncodeImageRef(ImageRef{Source: r.Source, Root: r.Root, Path: r.Path, Entry: entry, Payload: pairRel, Platform: ImagePlatformOf(r.Path)}),
				Name: name,
				Path: entry,
				Size: e.Length,
				Kind: "file",
				Pair: pairRel,
			}
			if serr == nil {
				if _, gerr := imgbcore.SectionGTEX(sec); gerr == nil {
					node.Kind = "image"
					node.ImageRef = node.ID
					node.Container = "trb"
				}
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}
