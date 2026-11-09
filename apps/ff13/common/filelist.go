// Package common concentra o CONTRATO do navegador de filelist do FF13: o
// DTO que o backend entrega e os códigos (grupo, kind, idioma, plataforma)
// que as camadas de dados e de árvore compartilham.
//
// O DTO carrega só FATOS: nome cru do arquivo, caminho dentro do índice,
// código de idioma/plataforma. Nenhum rótulo pronto, nenhuma minúscula
// forçada — a formatação de texto é do frontend (a arquitetura fica pronta
// para i18n).
package common

import (
	"regexp"
	"strings"
)

// Grupos e kinds do DTO de filelist.
const (
	// GroupZone separa os índices de ZONA dos índices globais.
	GroupZone = "zone"

	// KindDir: diretório dentro do índice.
	KindDir = "dir"
	// KindFile: arquivo dentro do índice.
	KindFile = "file"
	// KindIndex: o próprio arquivo é um filelist (índice embutido).
	KindIndex = "index"
	// KindDisk: índice em disco.
	KindDisk = "disk"
	// KindEmbedded: índice embutido num white bin.
	KindEmbedded = "embedded"
)

// Idiomas dos índices (o áudio do part1 é o que muda entre eles).
const (
	// LangU: índice em inglês.
	LangU = "u"
	// LangC: índice em japonês.
	LangC = "c"
)

// Plataformas possíveis de um índice (derivadas do nome do arquivo).
const (
	// SourceWin32: índice de PC (Windows).
	SourceWin32 = "win32"
	// SourceX360: índice de Xbox 360.
	SourceX360 = "x360"
	// SourcePS3: índice de PlayStation 3.
	SourcePS3 = "ps3"
)

// FilelistRoot é um índice (filelist) descoberto na part.
type FilelistRoot struct {
	// Name é o nome do arquivo do índice, com o casing ORIGINAL.
	Name string `json:"name"`
	// Path é o caminho do índice: absoluto quando ele está em disco;
	// "pai#virtual" quando é EMBUTIDO. É o valor que a listagem recebe.
	Path string `json:"path"`
	// Part é a part (part1/part2/part3) onde a raiz foi descoberta.
	Part string `json:"part"`
	// SourceDir é a pasta onde a cópia vencedora mora: a pasta de dados da
	// part (white_data/alba_data/weiss_data) ou a cópia extraída do app
	// (data/partN).
	SourceDir string `json:"sourceDir"`
	// Entries é o número de entradas do índice.
	Entries int `json:"entries"`
	// WhiteBin é o white bin PAR (mesma chave de par), quando existe.
	WhiteBin string `json:"whiteBin,omitempty"`
	// Group separa os índices globais ("") dos de ZONA ("zone").
	Group string `json:"group,omitempty"`
	// Zone é o id da zona ("z000") dos índices de zona.
	Zone string `json:"zone,omitempty"`
	// Language é o idioma do índice ("u"/"c"), quando o nome diz.
	Language string `json:"language,omitempty"`
	// Source é a plataforma do índice ("win32"/"x360"/"ps3"), quando o
	// nome diz. A plataforma é do ÍNDICE INTEIRO: as entradas herdam.
	Source string `json:"source,omitempty"`
	// Kind é "disk" ou "embedded".
	Kind string `json:"kind,omitempty"`
}

// FilelistNode é um nó da árvore dentro de um índice.
type FilelistNode struct {
	// Name é o nome cru do nó (primeiro segmento do caminho).
	Name string `json:"name"`
	// Path é o caminho real dentro do índice: é o que a listagem recebe
	// para devolver os filhos (no caso de um diretório, o diretório final
	// da cadeia fundida).
	Path string `json:"path"`
	// MergeChain é a cadeia de diretórios com filho único fundidos no nó
	// ("empty/empty"): o rótulo do nó é a cadeia unida por "/".
	MergeChain []string `json:"mergeChain,omitempty"`
	// IsDir marca os diretórios.
	IsDir bool `json:"isDir"`
	// Size é o tamanho DESCOMPRIMIDO, quando o fileInfo é parseável.
	Size uint32 `json:"size,omitempty"`
	// Kind é "dir"/"file"/"index".
	Kind string `json:"kind,omitempty"`
	// Language é o idioma do índice embutido ("u"/"c"), quando o nome diz.
	Language string `json:"language,omitempty"`
	// UnitID é o id da UNIDADE DE TEXTO do arquivo ("<dir>/<base>"), quando
	// o arquivo é texto (.ztr solto ou .bin container WPD). É o mesmo id da
	// árvore /data: a folha do filelist abre a mesma unidade.
	UnitID string `json:"unitId,omitempty"`
	// UnitLang é o IDIOMA do arquivo de texto ("us"/"jp"/…): é o idioma que
	// a folha clicada representa. Vazio quando o nome não diz (ex.:
	// "txtres_ck.ztr" — "ck" está fora do catálogo).
	UnitLang string `json:"unitLang,omitempty"`
	// Text marca o arquivo como texto (clique abre a tabela).
	Text bool `json:"text,omitempty"`
	// Container é o tipo do arquivo editável: "ztr" | "wpd".
	Container string `json:"container,omitempty"`
}

// zoneIDPattern casa o id da zona no nome do índice ("filelist_z000u" → "z000").
var zoneIDPattern = regexp.MustCompile(`z\d+`)

// SourceOf devolve a PLATAFORMA de um índice pelo NOME do arquivo: o segmento
// ".win32"/".x360"/".ps3" antes da extensão. Vazio quando o nome não diz
// (a fonte nunca é inventada).
//
// A plataforma é do índice INTEIRO (D6): nenhum arquivo muda a plataforma do
// índice que o contém.
func SourceOf(name string) string {
	n := baseName(name)
	n = strings.TrimSuffix(n, ".bin2")
	n = strings.TrimSuffix(n, ".bin")
	switch {
	case strings.HasSuffix(n, "."+SourceWin32):
		return SourceWin32
	case strings.HasSuffix(n, "."+SourceX360):
		return SourceX360
	case strings.HasSuffix(n, "."+SourcePS3):
		return SourcePS3
	}
	return ""
}

// LanguageOf devolve o IDIOMA de um índice pelo NOME do arquivo: "u" quando o
// nome (sem extensão) TERMINA em "u" (inglês) e "c" quando termina em "c"
// (japonês). Regra mecânica (D3): o nome que termina em "c" é japonês, mesmo
// que seja por coincidência ("filelist_patch_c").
func LanguageOf(name string) string {
	stem := stemOf(name)
	if stem == "" {
		return ""
	}
	switch stem[len(stem)-1] {
	case byte(LangU[0]):
		return LangU
	case byte(LangC[0]):
		return LangC
	}
	return ""
}

// ZoneIDOf devolve o ID da ZONA de um índice: "filelist_z000u.win32.bin" →
// "z000". Vazio quando o nome não traz id de zona.
func ZoneIDOf(name string) string {
	return zoneIDPattern.FindString(baseName(name))
}

// stemOf devolve o nome do arquivo sem a extensão final (".bin"/".bin2") e
// sem o segmento de plataforma ("filelistu.win32.bin" → "filelistu").
func stemOf(name string) string {
	n := baseName(name)
	n = strings.TrimSuffix(n, ".bin2")
	n = strings.TrimSuffix(n, ".bin")
	for _, src := range []string{SourceWin32, SourceX360, SourcePS3} {
		n = strings.TrimSuffix(n, "."+src)
	}
	return n
}

// baseName devolve o último segmento de um caminho (aceita "/" e "\" como
// separadores), em minúsculas.
func baseName(name string) string {
	n := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if i := strings.LastIndex(n, "/"); i >= 0 {
		return n[i+1:]
	}
	return n
}

// DataDirName devolve o nome da pasta de dados de uma part: white_data,
// alba_data ou weiss_data. É o "sourceDir" do DTO e o critério do desempate
// entre as cópias do mesmo índice (D4).
func DataDirName(part string) string {
	switch strings.ToLower(strings.TrimSpace(part)) {
	case "part2":
		return "alba_data"
	case "part3":
		return "weiss_data"
	default:
		return "white_data"
	}
}

// AppDataDir devolve o nome da pasta do APP onde a cópia extraída do índice
// mora ("data/part1"): a alternativa ao sourceDir quando o jogo não está
// instalado.
func AppDataDir(part string) string {
	switch strings.ToLower(strings.TrimSpace(part)) {
	case "part2":
		return "data/part2"
	case "part3":
		return "data/part3"
	default:
		return "data/part1"
	}
}
