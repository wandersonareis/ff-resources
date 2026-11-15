package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	ztenc "ffresources/internal/families/white/codecs/ztr/encoding"
	"ffresources/internal/families/white/profile/config"
	loggingService "ffresources/internal/shared/logging"
)

// UnitContent e a VISAO DEFAULT (o slot "us") de uma unidade: o pristine, o
// texto atual (mods-first) e as referencias dos demais idiomas.
//
// Os textos sao indexados pelo ID do ZTR (o mesmo ID = a mesma linha): e o
// que alinha a traducao com o pristine mesmo quando a ordem das chaves muda.
type UnitContent struct {
	IDs  []string
	Orig map[string]string            // id -> pristine do idioma default
	Cur  map[string]string            // id -> texto atual (mods-first)
	Refs map[string]map[string]string // idioma -> id -> pristine
}

// ReadUnitContent decodifica a unidade: o pristine do primario, a traducao
// de mods/ e as referencias dos demais idiomas.
func ReadUnitContent(part string, unit *TextUnit) (*UnitContent, error) {
	// O codec é por ARQUIVO, não por unidade: o slot editável é o "_us"
	// (LatinJapanese), mas cada referência vem do codepage do seu sufixo
	// (_ch = Big5, _kr = EUC-KR). Um codec único para todos é o que
	// produzia o mojibake "ｾｵ ｼ� …" nas referências CJK.
	ks, err := TextCodecForLang(part, unit.PrimaryLang)
	if err != nil {
		return nil, err
	}
	// Pristine: o arquivo da unidade em data/ — ou no white bin, quando ele
	// nao foi extraido para data/.
	origRaw, err := ReadUnitFile(unit, unit.PrimaryLang)
	if err != nil {
		return nil, fmt.Errorf("lendo original %s: %w", unit.Primary, err)
	}
	orig, err := ztr.Read(origRaw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", unit.Primary, err)
	}
	origTexts, err := ks.DecodeLines(orig.Lines)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", unit.Primary, err)
	}
	content := &UnitContent{
		IDs:  orig.IDs,
		Orig: make(map[string]string, len(orig.IDs)),
		Cur:  make(map[string]string, len(orig.IDs)),
		Refs: map[string]map[string]string{},
	}
	for i, key := range orig.IDs {
		text := ""
		if i < len(origTexts) {
			text = origTexts[i]
		}
		content.Orig[key] = text
		content.Cur[key] = text
	}

	// Traducao: o arquivo de mods pode ter as mesmas chaves em outra ordem.
	modsDir, err := config.PartRoot("mods", part)
	if err != nil {
		return nil, err
	}
	if b, rerr := os.ReadFile(filepath.Join(modsDir, filepath.FromSlash(unit.Primary))); rerr == nil {
		if raw, ok := unitTextBytes(b, unit.Container); ok {
			if mf, zerr := ztr.Read(raw); zerr == nil {
				if mt, derr := ks.DecodeLines(mf.Lines); derr == nil {
					for i, k := range mf.IDs {
						if i < len(mt) {
							content.Cur[k] = mt[i]
						}
					}
				} else {
					loggingService.Warn("tradução %s ilegível (decode): %v", unit.Primary, derr)
				}
			} else {
				loggingService.Warn("tradução %s ilegível (ZTR): %v", unit.Primary, zerr)
			}
		}
	}

	// Referencias: os outros idiomas da unidade, lidos de data/ (ou do
	// white bin) e alinhados pelo ID do ZTR. ID ausente no idioma = sem
	// referencia na row.
	for _, lang := range unit.Langs {
		if lang == unit.PrimaryLang {
			continue
		}
		rel, ok := unit.Files[lang]
		if !ok {
			continue
		}
		refKS, err := TextCodecForLang(part, lang)
		if err != nil {
			loggingService.Warn("referência %s sem charset: %v", rel, err)
			continue
		}
		raw, err := ReadUnitFile(unit, lang)
		if err != nil {
			loggingService.Warn("referência %s ilegível: %v", rel, err)
			continue
		}
		zf, err := ztr.Read(raw)
		if err != nil {
			loggingService.Warn("referência %s ilegível: %v", rel, err)
			continue
		}
		texts, err := refKS.DecodeLines(zf.Lines)
		if err != nil {
			loggingService.Warn("referência %s ilegível: %v", rel, err)
			continue
		}
		m := make(map[string]string, len(zf.IDs))
		for i, k := range zf.IDs {
			if i < len(texts) {
				m[k] = texts[i]
			}
		}
		content.Refs[lang] = m
	}
	return content, nil
}

// unitTextBytes devolve os bytes do TEXTO de um arquivo da unidade: o .ztr
// solto ou o registro .ztr dentro do container WPD.
func unitTextBytes(raw []byte, container string) ([]byte, bool) {
	if container != "wpd" {
		return raw, true
	}
	_, data, ok := DetectWpdZtr(raw)
	return data, ok
}

// ZtrSaveFormat é o formato da GRAVAÇÃO de texto. O padrão continua o BE do
// jogo; o LE (par/dicionário do Pulse) fica disponível para uma escolha
// posterior (config/UI). A leitura é sempre automática.
var ZtrSaveFormat = ztr.FormatBigEndianCompressed

// writeZtrFile centraliza a gravação: todo save passa por aqui para que a
// escolha de formato seja um ponto único.
func writeZtrFile(f *ztr.File) ([]byte, error) {
	return ztr.WriteFormat(f, ZtrSaveFormat, ztr.Compressed, ztr.TieMinLast)
}

// WriteZtrText reescreve o .ztr solto da unidade (pristine da unidade + os
// textos editados, por ID) e devolve o blob pronto para mods/.
//
// O pristine vem de data/ ou do white bin (quando o arquivo não foi
// extraído para data/).
func WriteZtrText(unit *TextUnit, ks *ztenc.TextCodec, byKey map[string]string) ([]byte, error) {
	raw, err := ReadUnitFile(unit, unit.PrimaryLang)
	if err != nil {
		return nil, err
	}
	zf, err := ztr.Read(raw)
	if err != nil {
		return nil, err
	}
	lines, err := ApplyTexts(ks, zf, byKey)
	if err != nil {
		return nil, err
	}
	return writeZtrFile(&ztr.File{IDs: zf.IDs, Lines: lines})
}

// PackWpdText reempacota o container WPD: o registro .ztr e reescrito com os
// textos editados e os demais registros vao byte a byte (o .wdb e o resto).
//
// O container base e o de mods/ quando existe (o que o usuario ja salvou),
// senao o pristine da unidade (data/ ou white bin).
func PackWpdText(modsDir string, unit *TextUnit, ks *ztenc.TextCodec, byKey map[string]string) ([]byte, error) {
	var raw []byte
	if b, err := os.ReadFile(filepath.Join(modsDir, filepath.FromSlash(unit.Primary))); err == nil {
		raw = b
	} else {
		var rerr error
		raw, rerr = ReadSource(unit.Sources[unit.PrimaryLang].Source)
		if rerr != nil {
			return nil, rerr
		}
	}
	f, err := wpd.Read(raw)
	if err != nil {
		return nil, err
	}
	ztrIndex := -1
	for i, r := range f.Header.Records {
		if strings.EqualFold(r.Extension, "ztr") {
			ztrIndex = i
			break
		}
	}
	if ztrIndex < 0 {
		return nil, fmt.Errorf("container sem registro .ztr")
	}
	rec := f.Header.Records[ztrIndex]
	zraw, ok := f.ExtractRecord(rec.Name, rec.Extension)
	if !ok {
		return nil, fmt.Errorf("registro %s.%s não encontrado", rec.Name, rec.Extension)
	}
	zf, err := ztr.Read(zraw)
	if err != nil {
		return nil, err
	}
	lines, err := ApplyTexts(ks, zf, byKey)
	if err != nil {
		return nil, err
	}
	blob, err := writeZtrFile(&ztr.File{IDs: zf.IDs, Lines: lines})
	if err != nil {
		return nil, err
	}
	parts := make([]wpd.Part, 0, len(f.Header.Records))
	for i, r := range f.Header.Records {
		data, ok := f.ExtractRecord(r.Name, r.Extension)
		if !ok {
			return nil, fmt.Errorf("registro %s.%s não encontrado", r.Name, r.Extension)
		}
		if i == ztrIndex {
			data = blob
		}
		parts = append(parts, wpd.Part{Name: r.Name, Extension: r.Extension, Data: data})
	}
	return wpd.WriteFromParts(parts)
}

// ApplyTexts reescreve as linhas do ZTR com os textos editados (por ID). As
// linhas sem edicao mantem o texto pristine.
func ApplyTexts(ks *ztenc.TextCodec, zf *ztr.File, byKey map[string]string) ([][]byte, error) {
	texts := make([]string, len(zf.IDs))
	for i, key := range zf.IDs {
		if t, ok := byKey[key]; ok {
			texts[i] = t
			continue
		}
		if i >= len(zf.Lines) {
			return nil, fmt.Errorf("linha %d ausente no arquivo", i)
		}
		origText, derr := ks.DecodeLine(zf.Lines[i])
		if derr != nil {
			return nil, fmt.Errorf("linha %d: %w", i, derr)
		}
		texts[i] = origText
	}
	return ks.EncodeLines(texts)
}

// TextCodec devolve o codec do slot EDITÁVEL da unidade (o "_us", sempre
// LatinJapanese). Quem lê OUTRO idioma precisa de TextCodecForLang: cada
// codepage tem seus próprios pares/glifos e o charset é o do arquivo.
func TextCodec(part string) (*ztenc.TextCodec, error) {
	return TextCodecFor(part, ztenc.LatinJapanese)
}

// TextCodecFor devolve o codec do (game, codepage), preparando sob demanda
// (idempotente: rechamar sobrescreve o registro).
func TextCodecFor(part string, cp ztenc.Codepage) (*ztenc.TextCodec, error) {
	var game ztenc.GameCode
	switch config.NormalizePart(part) {
	case "part2":
		game = ztenc.FF132
	case "part3":
		game = ztenc.FF133
	default:
		game = ztenc.FF131
	}
	if err := ztenc.PrepareTextCodec(game, cp); err != nil {
		return nil, fmt.Errorf("charset %s: %w", game, err)
	}
	return ztenc.TextCodecFor(game, cp)
}

// CodepageOf devolve o codepage FÍSICO do .ztr de um idioma, pelo sufixo
// do arquivo: _kr é EUC-KR, _ch é Big5 e _jp/_us são Shift-JIS. Os demais
// códigos seguem no LatinJapanese — o texto desses arquivos é ASCII/glifo e
// decodifica igual nos três codecs.
//
// ERRAR aqui é o bug do mojibake: ler EUC-KR/Big5 como Shift-JIS transforma
// cada byte 0xA1–0xDF em kana de meia largura e os pares em ideographas
// erradas. Também vale para o encode: gravar o idioma com o codec do outro
// produziria bytes que o jogo não lê.
func CodepageOf(lang string) ztenc.Codepage {
	if cp, ok := ztenc.CodepageForLanguage(lang); ok {
		return cp
	}
	return ztenc.LatinJapanese
}

// TextCodecForLang é o CodepageOf resolvido no registro.
func TextCodecForLang(part, lang string) (*ztenc.TextCodec, error) {
	return TextCodecFor(part, CodepageOf(lang))
}
