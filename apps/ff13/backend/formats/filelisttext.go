package formats

import (
	"path/filepath"
	"strings"
)

// MAPEAMENTO entre o que o FILELIST indexa e o layout de dados do app
// (data/partN).
//
// O índice GLOBAL declara o caminho interno como ele é no layout de dados
// ("db/ai/npc/pack/auto_c01_us.bin"). O índice de ZONA declara o caminho
// RELATIVO à zona ("txtres/resident/book/txtres_us.ztr"), que no layout de
// dados vive em "zone/<z>/...". É a mesma regra do importador
// (scripts/ff13import/main.go:452-463) — e é o que faz a folha do filelist
// abrir a MESMA unidade da árvore /data.

// FilelistDataPath converte o caminho indexado de uma entrada no caminho
// relativo a data/partN.
//
//	zona ""   : "db/ai/npc/pack/auto_c01_us.bin" -> "db/ai/npc/pack/auto_c01_us.bin"
//	zona z000 : "txtres/x_us.ztr"                -> "zone/z000/txtres/x_us.ztr"
//	            "zone/z000/txtres/x_us.ztr"      -> "zone/z000/txtres/x_us.ztr"
//
// O prefixo "zone/<z>/" sai e volta: o índice de zona usa o caminho
// relativo, os índices globais usam o prefixo. Tudo em minúsculas — o layout
// de dados do app é minúsculo e o do importador casa sem distinção de caixa.
func FilelistDataPath(zone, indexPath string) string {
	p := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(indexPath), "\\", "/"))
	p = strings.TrimPrefix(p, "/")
	if zone == "" {
		return p
	}
	prefix := "zone/" + zone + "/"
	p = strings.TrimPrefix(p, prefix)
	return prefix + p
}

// IsTextExt diz se a EXTENSÃO do arquivo é de um formato que pode conter
// texto: ".ztr" solto ou ".bin" (container WPD). A verificação real do
// container (o registro .ztr dentro) acontece na leitura.
func IsTextExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ztr", ".bin":
		return true
	}
	return false
}

// ContainerOfExt devolve o tipo de container do arquivo pela extensão:
// "ztr" (texto solto) ou "wpd" (container .bin). O "" é o resultado para
// qualquer outra extensão.
func ContainerOfExt(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ztr":
		return "ztr"
	case ".bin":
		return "wpd"
	}
	return ""
}

// FilelistUnitID devolve o id da unidade ("<dir>/<base>") e o idioma de uma
// entrada indexada. ok=false quando o arquivo não é de um formato de texto.
//
// O idioma pode vir vazio (um sufixo fora do catálogo da trilogia): nesse
// caso a unidade existe mas não tem primário "_us" — ela não é editável,
// como na árvore /data.
func FilelistUnitID(dataPath string) (id, lang string, ok bool) {
	dir, file := SplitRel(dataPath)
	if !IsTextExt(file) {
		return "", "", false
	}
	base, l := ParseUnitFileName(file)
	return UnitID(dir, base), l, true
}
