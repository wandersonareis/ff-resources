// Package metadata monta as chaves canônicas dos artefatos do FFX a partir do
// tipo de jogo. Fica na família FFX porque o formato da chave (versão,
// pack root, event/objects/help/macro) é exclusivo do jogo; o DTO neutro só
// carrega Key/ID.
package metadata

import (
	"path/filepath"
	"strings"

	"ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/shared/dto"
)

// NewEventMetadata gera a metadata de um evento a partir do id.
// Chave canônica: <version>/event/obj_ps3/<short>/<id>/<id>.bin,
// onde short = id[:2]. Sem dependência dos geradores legados.
func NewEventMetadata(eventID string, version common.GameVersion) dto.Metadata {
	id := strings.TrimSpace(eventID)
	if len(id) < 2 {
		return dto.Metadata{ID: id}
	}
	short := id[:2]
	key := common.VersionPathName(version) + "/event/obj_ps3/" + short + "/" + id + "/" + id + ".bin"
	return dto.Metadata{Key: key, ID: id}
}

// NewObjectMetadata gera a metadata de um objectsfile.
// id = basename sem extensão (important, command, btl_txt, ...).
func NewObjectMetadata(version common.GameVersion, dirPattern, fileName, key string) dto.Metadata {
	_ = version
	_ = dirPattern
	id := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	if key == "" {
		key = strings.Trim(strings.TrimSpace(dirPattern)+"/"+strings.TrimSpace(fileName), "/")
	}
	return dto.Metadata{Key: filepath.ToSlash(key), ID: id}
}

// NewHelpMetadata gera a metadata de um painel de ajuda (kind help).
// dirPattern = "help/<dir>" (ex.: help/now_help, help/mon_boku);
// id = stem do arquivo (now_help, now_help_page, mon_boku, ...).
func NewHelpMetadata(name, dirPattern string, version common.GameVersion) dto.Metadata {
	id := strings.TrimSpace(name)
	key := strings.Trim(strings.TrimSpace(dirPattern)+"/"+strings.TrimSpace(id)+".sps2", "/")
	return dto.Metadata{Key: filepath.ToSlash(common.VersionPathName(version) + "/" + key), ID: id}
}

// NewMacroMetadata gera a metadata de um chunk de macrodic.
// A key é compartilhada por todos os chunks; a identidade é o id (chunk_NN).
func NewMacroMetadata(version common.GameVersion, chunkIndex int) dto.Metadata {
	return dto.Metadata{
		Key: common.VersionPathName(version) + "/menu/macrodic.dcp",
		ID:  dto.MacroChunkID(chunkIndex),
	}
}

// NewImageMetadata monta a metadata canônica de uma textura: a key segue o
// padrão dos demais kinds (<versão>/<relPath>), o id é o caminho relativo
// da árvore sem o sufixo .dds.phyre.
func NewImageMetadata(id string, version common.GameVersion) dto.Metadata {
	return dto.Metadata{
		Key: common.VersionPathName(version) + "/" + id + ".dds.phyre",
		ID:  id,
	}
}
