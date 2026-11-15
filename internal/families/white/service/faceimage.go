package service

import (
	"fmt"
	"regexp"
	"strings"

	imgbcore "ffresources/internal/families/white/codecs/imgb/core"
)

// residentFacePattern casa as texturas de face do GUI que NÃO têm tabela
// externa: o FF13-2 grava gui/resident/face/faceNNN.win32.imgb como DXT1
// 128x256, um atlas fixo. O arquivo de 8 bytes é placeholder.
var residentFacePattern = regexp.MustCompile(`(?i)^gui/resident/face/face\d+\.win32\.imgb$`)

const (
	ResidentFaceWidth   = 128
	ResidentFaceHeight  = 256
	ResidentFaceRawSize = ResidentFaceWidth * ResidentFaceHeight / 2 // DXT1 = 0,5 byte/pixel
)

func IsResidentFaceImage(path string) bool {
	return residentFacePattern.MatchString(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"))
}

// ResidentFaceDDS monta o DDS implícito do atlas de face.
func ResidentFaceDDS(raw []byte) ([]byte, imgbcore.ImageInfo, error) {
	var info imgbcore.ImageInfo
	if len(raw) != ResidentFaceRawSize {
		return nil, info, fmt.Errorf("face: tamanho %d (esperado %d para DXT1 %dx%d)",
			len(raw), ResidentFaceRawSize, ResidentFaceWidth, ResidentFaceHeight)
	}
	g := &imgbcore.GTEX{
		IsValid:  true,
		Width:    ResidentFaceWidth,
		Height:   ResidentFaceHeight,
		MipCount: 1,
		Format:   imgbcore.GTEXFormatDXT1,
		Type:     imgbcore.GTEXTypeClassic,
	}
	header, err := imgbcore.GetDDSHeaderMips(g, 1, 0)
	if err != nil {
		return nil, info, err
	}
	dds := make([]byte, 0, len(header)+len(raw))
	dds = append(dds, header...)
	dds = append(dds, raw...)
	info = imgbcore.ImageInfo{
		Format:      "DXT1",
		Width:       ResidentFaceWidth,
		Height:      ResidentFaceHeight,
		MipmapCount: 1,
		Layers:      1,
	}
	return dds, info, nil
}
