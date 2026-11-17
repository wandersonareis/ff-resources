package event

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/families/ffx/profile/models"
)

// ReadAllEventFiles carrega todos os eventos da árvore original direto no
// store (caminho síncrono/legado: appcore e CLI).
func ReadAllEventFiles(eventsFolder common.FileAccessor, version common.GameVersion) error {
	if !eventsFolder.Exists {
		return fmt.Errorf("events directory not found: %s", eventsFolder.ResolvedPath)
	}

	discovered, err := discoverEventFiles(eventsFolder, version, false)
	if err != nil {
		return fmt.Errorf("failed to discover event files: %w", err)
	}

	for _, info := range discovered {
		if _, err := ReadCompleteEventFile(info); err != nil {
			common.LogVerbose("failed to read event file %s: %v", info.EventID, err)
			continue
		}
	}

	common.LogInfo("events loaded: %d", len(discovered))
	return nil
}

func SortedSupportedLocalizations() []string {
	keys := make([]string, 0, len(common.SupportedLanguages))
	for loc := range common.SupportedLanguages {
		if loc == common.DefaultLocalization {
			continue
		}
		keys = append(keys, loc)
	}
	sort.Strings(keys)
	return append([]string{common.DefaultLocalization}, keys...)
}

// discoverEventFiles enumera as pastas de evento da árvore informada.
//
// A descoberta é DATA-DRIVEN e de UMA localização (a padrão): as pastas são
// globais (event/obj_ps3/<xx>/<id>), não por idioma - varrer as demais
// localizações relia a MESMA pasta ~10× e nunca encontrava nada novo.
//
// includeSkipped=true devolve também os ids da lista estática de skip: usado
// pelo inventário de mods, que compara ARQUIVOS EXISTENTES em disco (a lista
// de skip esconde da árvore, não do disco).
func discoverEventFiles(eventsFolder common.FileAccessor, version common.GameVersion, includeSkipped bool) ([]*models.EventFileInfo, error) {
	entries, err := os.ReadDir(eventsFolder.ResolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read events directory: %w", err)
	}

	var eventInfos []*models.EventFileInfo
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		subInfos, err := discoverSubdirectoryEvents(eventsFolder.ResolvedPath, entry.Name(), version, includeSkipped)
		if err != nil {
			common.LogVerbose("failed to read subdirectory %s: %v", entry.Name(), err)
			continue
		}
		eventInfos = append(eventInfos, subInfos...)
	}

	sort.Slice(eventInfos, func(i, j int) bool {
		return eventInfos[i].EventID < eventInfos[j].EventID
	})
	return eventInfos, nil
}

// discoverSubdirectoryEvents aceita como evento toda subpasta que tenha o
// binário homônimo (<id>/<id>.bin) - o MESMO critério de eventtable.IDs.
// Pasta sem binário não é evento: fica fora da descoberta (e do universo
// conhecido) em vez de virar, mais tarde, um "evento pulado (sem conteúdo)".
func discoverSubdirectoryEvents(eventsRoot, subdirName string, version common.GameVersion, includeSkipped bool) ([]*models.EventFileInfo, error) {
	subPath := filepath.Join(eventsRoot, subdirName)
	subEntries, err := os.ReadDir(subPath)
	if err != nil {
		return nil, err
	}

	var eventInfos []*models.EventFileInfo
	for _, subEntry := range subEntries {
		if !subEntry.IsDir() || strings.HasPrefix(subEntry.Name(), ".") {
			continue
		}
		bin := filepath.Join(subPath, subEntry.Name(), subEntry.Name()+".bin")
		if _, statErr := os.Stat(bin); statErr != nil {
			common.LogVerbose("pasta de evento sem binário em data/: %s/%s", subdirName, subEntry.Name())
			continue
		}
		info := models.NewEventFileInfo(subEntry.Name(), version)
		if info == nil {
			continue
		}
		if !includeSkipped && common.IsSkippedFilePath(info.LocalizationPattern, info.Version) {
			continue // silencioso: lista estática
		}
		eventInfos = append(eventInfos, info)
	}

	return eventInfos, nil
}

// DiscoverEventIDsFromData devolve os ids de evento que existem em data/ (o
// binário homônimo), INCLUINDO os da lista estática de skip. É a fonte da
// verdade do inventário de mods: comparação por arquivo em disco, não pelo
// store (que só tem o que a árvore decidiu carregar).
func DiscoverEventIDsFromData(version common.GameVersion) ([]string, error) {
	folder, err := NewEventsBinaryFile(version).eventsFolder(common.DefaultLocalization)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve events directory: %w", err)
	}
	if !folder.Exists {
		return nil, nil
	}
	infos, err := discoverEventFiles(folder, version, true)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		ids = append(ids, info.EventID)
	}
	return ids, nil
}

// readEventFile lê UM evento do disco (preferindo o binário de mods) SEM tocar
// no store. É a metade "leitura" de ReadCompleteEventFile - a publicação fica
// com quem orquestra a carga (permite ler em paralelo e publicar em ordem).
func readEventFile(info *models.EventFileInfo) (*EventFile, error) {
	if info == nil || len(info.EventID) < 2 {
		return nil, fmt.Errorf("invalid event ID")
	}

	if common.IsSkippedFilePath(info.LocalizationPattern, info.Version) {
		return nil, nil // silencioso: lista estática
	}

	localizedStrings := ReadLocalizedStringFiles(info.LocalizationPattern, info.Version)
	if len(localizedStrings) == 0 {
		common.LogVerbose("Skipping event due to missing or empty localization: %s", info.EventID)
		return nil, nil
	}

	return &EventFile{
		ID:      info.EventID,
		Version: info.Version,
		Strings: localizedStrings,
	}, nil
}

// ReadCompleteEventFile lê o evento e o publica no store (caminho de UM
// arquivo - frescura/recarga). A carga bulk usa readEventFile e publica em
// ordem, depois de ler tudo em paralelo.
func ReadCompleteEventFile(info *models.EventFileInfo) (*EventFile, error) {
	eventFile, err := readEventFile(info)
	if err != nil || eventFile == nil {
		return eventFile, err
	}
	SetEvent(info.Version, info.EventID, eventFile)
	return eventFile, nil
}
