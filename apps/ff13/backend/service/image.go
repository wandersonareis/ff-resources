package service

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"ffresources/apps/ff13/backend/common"
	"ffresources/apps/ff13/backend/config"
	"ffresources/apps/ff13/backend/formats"
	"ffresources/apps/ff13/backend/parts"
	"ffresources/internal/families/white/codecs/imgb/core"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/imagery"
)

func imageSourceLabel(part string, r formats.ImageRef) dto.Metadata {
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

func imageOutputBase(r formats.ImageRef) string {
	base := imageBaseName(r.Path)
	if r.Entry == "" {
		return base
	}
	entry := strings.TrimPrefix(r.Entry, "rec:")
	entry = strings.TrimPrefix(entry, "trb:")
	entry = strings.NewReplacer(".", "_", ":", "_", "/", "_").Replace(entry)
	return base + "_" + entry
}

func imageInMods(part, rel string) bool {
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

// GetImageEntry devolve a prévia PNG + metadados da imagem/ref.
func (FF13Service) GetImageEntry(part, ref string) (dto.ImageEntry, error) {
	part = config.NormalizePart(part)
	r, err := formats.DecodeImageRef(ref)
	if err != nil {
		return dto.ImageEntry{}, err
	}

	// Arquivo direto .dds/.png.
	if r.Entry == "" && !formats.IsContainerExt(r.Path) {
		f, ferr := formats.ResolveImageFile(part, r)
		if ferr != nil {
			return dto.ImageEntry{}, ferr
		}
		raw, rerr := f.Read()
		if rerr != nil {
			return dto.ImageEntry{}, rerr
		}
		if formats.IsResidentFaceImage(r.Path) {
			dds, info, ferr := formats.ResidentFaceDDS(raw)
			if ferr != nil {
				return dto.ImageEntry{}, ferr
			}
			png, perr := imagery.DDSToPNG(dds)
			if perr != nil {
				return dto.ImageEntry{}, perr
			}
			return dto.ImageEntry{
				Metadata: imageSourceLabel(part, r),
				Source:   "imgb",
				Format:   info.Format,
				Width:    info.Width,
				Height:   info.Height,
				PNGData:  formats.ImageDataURL("image/png", png),
				Flipped:  false,
				Modded:   imageInMods(part, f.Rel),
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
				Metadata: imageSourceLabel(part, r),
				Source:   "dds",
				Format:   info.Format,
				Width:    info.Width,
				Height:   info.Height,
				PNGData:  formats.ImageDataURL("image/png", png),
				Flipped:  false,
				Modded:   imageInMods(part, f.Rel),
			}, nil
		case ".png":
			cfg, _, derr := image.DecodeConfig(bytes.NewReader(raw))
			if derr != nil {
				cfg.Width, cfg.Height = 0, 0
			}
			return dto.ImageEntry{
				Metadata: imageSourceLabel(part, r),
				Source:   "png",
				Format:   "PNG",
				Width:    uint32(cfg.Width),
				Height:   uint32(cfg.Height),
				PNGData:  formats.ImageDataURL("image/png", raw),
				Flipped:  false,
				Modded:   imageInMods(part, f.Rel),
			}, nil
		}
		return dto.ImageEntry{}, fmt.Errorf("imagem: extensão não suportada (%s)", ext)
	}

	dds, info, derr := formats.DecodeImageDDS(part, r, false)
	if derr != nil {
		return dto.ImageEntry{}, derr
	}
	png, perr := imagery.DDSToPNG(dds)
	if perr != nil {
		return dto.ImageEntry{}, perr
	}
	table, terr := formats.ResolveImageFile(part, r)
	if terr != nil {
		return dto.ImageEntry{}, terr
	}
	return dto.ImageEntry{
		Metadata:       imageSourceLabel(part, r),
		Source:         "imgb",
		Format:         info.Format,
		Width:          info.Width,
		Height:         info.Height,
		MipmapCount:    info.MipmapCount,
		MaxMipmapLevel: info.MipmapCount,
		PNGData:        formats.ImageDataURL("image/png", png),
		Flipped:        false,
		Modded:         imageInMods(part, table.Rel),
		Duplicates:     []dto.ImageDuplicate{},
	}, nil
}

// ImageExists responde se a ref ainda resolve.
func (FF13Service) ImageExists(part, ref string) (bool, error) {
	r, err := formats.DecodeImageRef(ref)
	if err != nil {
		return false, nil
	}
	if r.Entry == "" && !formats.IsContainerExt(r.Path) {
		f, ferr := formats.ResolveImageFile(config.NormalizePart(part), r)
		return ferr == nil && f.Exists(), nil
	}
	if _, _, err := formats.ImageSection(config.NormalizePart(part), r); err != nil {
		return false, nil
	}
	return true, nil
}

// ImageDiagnostics devolve formato/dimensões sem decodificar a imagem.
func (FF13Service) ImageDiagnostics(part, ref string) (imgbcore.ImageInfo, error) {
	r, err := formats.DecodeImageRef(ref)
	if err != nil {
		return imgbcore.ImageInfo{}, err
	}
	return formats.Diagnostics(config.NormalizePart(part), r)
}

// SaveImage grava .dds ou .png no caminho escolhido.
func (FF13Service) SaveImage(part, ref, format, destPath string) error {
	part = config.NormalizePart(part)
	r, err := formats.DecodeImageRef(ref)
	if err != nil {
		return err
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if r.Entry == "" && !formats.IsContainerExt(r.Path) {
		f, ferr := formats.ResolveImageFile(part, r)
		if ferr != nil {
			return ferr
		}
		raw, rerr := f.Read()
		if rerr != nil {
			return rerr
		}
		if formats.IsResidentFaceImage(r.Path) {
			dds, _, ferr := formats.ResidentFaceDDS(raw)
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
		dds, _, derr := formats.DecodeImageDDS(part, r, true)
		if derr != nil {
			return derr
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destPath, dds, 0o644)
	case "png":
		dds, _, derr := formats.DecodeImageDDS(part, r, false)
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

// ImageExportPath devolve o caminho padrão de exportação em mods/edits/images.
func (FF13Service) ImageExportPath(part, ref, format string) (string, error) {
	r, err := formats.DecodeImageRef(ref)
	if err != nil {
		return "", err
	}
	part = config.NormalizePart(part)
	root, err := config.DataRoot("mods")
	if err != nil {
		return "", err
	}
	rel := r.Path
	if r.Root != "" {
		if idx, zone, rerr := formats.ResolveFilelistIndex(parts.Get(part), r.Root); rerr == nil {
			_ = idx
			rel = formats.FilelistDataPath(zone, r.Path)
		}
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "/")
	dir := filepath.Dir(filepath.FromSlash(rel))
	if dir == "." {
		dir = ""
	}
	return filepath.Join(root, "edits", "images", part, dir, imageOutputBase(r)+"."+format), nil
}

// ExtractImageSelection grava DDS+PNG das refs em destRoot.
func (FF13Service) ExtractImageSelection(part string, refs []string, destRoot string) (dto.BatchResult, error) {
	part = config.NormalizePart(part)
	res := dto.BatchResult{Done: []string{}, Failed: []string{}, Total: len(refs)}
	for _, ref := range refs {
		r, err := formats.DecodeImageRef(ref)
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
		rel := r.Path
		if r.Root != "" {
			if _, zone, rerr := formats.ResolveFilelistIndex(parts.Get(part), r.Root); rerr == nil {
				rel = formats.FilelistDataPath(zone, r.Path)
			}
		}
		dir := filepath.Join(target, filepath.Dir(filepath.FromSlash(rel)))
		name := imageOutputBase(r)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			res.Failed = append(res.Failed, ref+": "+err.Error())
			continue
		}
		ok := true
		for _, format := range []string{"dds", "png"} {
			if serr := (FF13Service{}).SaveImage(part, ref, format, filepath.Join(dir, name+"."+format)); serr != nil {
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
func (FF13Service) ImageDeletePlans(part string, refs []string) ([]common.ImageDeletePlan, error) {
	part = config.NormalizePart(part)
	plans := make([]common.ImageDeletePlan, 0, len(refs))
	for _, ref := range refs {
		r, err := formats.DecodeImageRef(ref)
		if err != nil {
			return nil, err
		}
		if r.Source == "filelist" {
			return nil, fmt.Errorf("delete de imagem disponível apenas na árvore de data/")
		}
		plan := common.ImageDeletePlan{ID: ref, Label: r.Path}
		var tableRel, pairRel string
		if r.Entry == "" && !formats.IsContainerExt(r.Path) {
			tableRel = r.Path
		} else {
			table, terr := formats.ResolveImageFile(part, r)
			if terr != nil {
				return nil, terr
			}
			tableRel = table.Rel
			pairRel = r.Payload
			if pairRel == "" {
				pairRel = formats.ImagePairPath(tableRel)
			}
			// Conta registros para o aviso.
			if raw, rerr := table.Read(); rerr == nil {
				switch formats.ContainerKind(r.Path, raw) {
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
func (FF13Service) DeleteImages(part string, refs []string, scope string) (dto.BatchResult, error) {
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
	plans, err := (FF13Service{}).ImageDeletePlans(part, refs)
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
	formats.InvalidateImageFileIndex(part)
	return res, nil
}

// DeleteImageSelection é o alias usado pelo contrato GameApi.
func (FF13Service) DeleteImageSelection(part string, refs []string, scope string) (dto.BatchResult, error) {
	return (FF13Service{}).DeleteImages(part, refs, scope)
}

func imageExportRoot(part string) string {
	root, err := config.DataRoot("mods")
	if err != nil {
		return ""
	}
	return filepath.Join(root, "edits", "images", config.NormalizePart(part))
}

func pickImageDirectory(title, defaultDir string) string {
	app := application.Get()
	if app == nil {
		return ""
	}
	selection, err := app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                title,
		Directory:            defaultDir,
		CanChooseFiles:       false,
		CanChooseDirectories: true,
	}).PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}

func pickImageFile(title, defaultDir string, filters ...application.FileFilter) string {
	app := application.Get()
	if app == nil {
		return ""
	}
	selection, err := app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:     title,
		Directory: defaultDir,
		Filters:   filters,
	}).PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}

// SelectImageExportDir escolhe a pasta raiz do "Extrair para…".
func (FF13Service) SelectImageExportDir(part string) string {
	return pickImageDirectory("Extrair imagens para…", imageExportRoot(part))
}

func (FF13Service) SelectImageFile(part, id, version string) string {
	return pickImageFile("Importar DDS", imageExportRoot(part),
		application.FileFilter{DisplayName: "Texturas DDS (*.dds)", Pattern: "*.dds"})
}

func (FF13Service) SelectImageSavePath(part, format, suggestedName, id, version string) string {
	app := application.Get()
	if app == nil {
		return ""
	}
	selection, err := app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:     "Salvar imagem",
		Directory: imageExportRoot(part),
		Filename:  suggestedName,
		Filters:   []application.FileFilter{{DisplayName: strings.ToUpper(format), Pattern: "*." + format}},
	}).PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}
func (FF13Service) ImportImage(part, ref, ddsPath string) error {
	return fmt.Errorf("import de imagem será implementado na fase 4")
}
func (FF13Service) RefreshImageDuplicates(part string) {}

// ListImageContainer lista os filhos de uma tabela WPD/TRB. Imagens ganham
// ImageRef; os demais registros viram arquivos comuns.
func (FF13Service) ListImageContainer(part, ref string) ([]common.ImageNode, error) {
	r, err := formats.DecodeImageRef(ref)
	if err != nil {
		return nil, err
	}
	if r.Entry != "" {
		return nil, fmt.Errorf("imagem: nó não é um container")
	}
	f, err := formats.ResolveImageFile(config.NormalizePart(part), r)
	if err != nil {
		return nil, err
	}
	raw, err := f.Read()
	if err != nil {
		return nil, err
	}
	kind := formats.ContainerKind(r.Path, raw)
	if kind == "" {
		return nil, fmt.Errorf("imagem: formato de container desconhecido (%s)", r.Path)
	}
	pairRel := r.Payload
	if pairRel == "" {
		pairRel = formats.ImagePairPath(f.Rel)
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
				ID:   formats.EncodeImageRef(formats.ImageRef{Source: r.Source, Root: r.Root, Path: r.Path, Entry: entry, Payload: pairRel, Platform: formats.ImagePlatformOf(r.Path)}),
				Name: rec.Name + "." + rec.Extension,
				Path: entry,
				Size: rec.Size,
				Kind: "file",
				Pair: pairRel,
			}
			if formats.IsImageRecordExt(rec.Extension) {
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
				ID:   formats.EncodeImageRef(formats.ImageRef{Source: r.Source, Root: r.Root, Path: r.Path, Entry: entry, Payload: pairRel, Platform: formats.ImagePlatformOf(r.Path)}),
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
