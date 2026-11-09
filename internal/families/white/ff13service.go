package white

// TextEntrySummary e o indice leve de um arquivo de TEXTO da familia White:
// ID e o caminho interno relativo a data/partN; Key e a chave canonica que a
// UI e o exporter usam (partN/caminho); Type e a categoria de CONTEUDO
// (system, mission, event, zone, book, review, text) — o frontend decide o
// rotulo/ícone, o backend nao traduz.
type TextEntrySummary struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Type string `json:"type,omitempty"`
	// Text marca os arquivos que o app sabe decodificar (.ztr). Os demais
	// (imagens, scripts, indices) aparecem na arvore como "fora do escopo":
	// a arvore de /data mostra TUDO que o importador extraiu, e so o texto
	// e editavel.
	Text bool `json:"text"`
	// Size do arquivo em bytes (0 quando desconhecido).
	Size int64 `json:"size,omitempty"`
}

// TextUnitSummary e o indice leve de uma UNIDADE de TEXTO da familia White.
//
// A unidade e o conjunto de arquivos do mesmo diretorio que compartilham o
// nome-base (txtres_us.ztr + txtres_fr.ztr -> a unidade "txtres"): o id nao
// depende do idioma, entao a arvore da UI mostra uma folha por unidade (e nao
// uma por arquivo) e os outros idiomas entram como REFERENCIAS somente-leitura.
type TextUnitSummary struct {
	// ID da unidade = "<dir>/<base>" (ex.: txtres/resident/book/txtres).
	ID string `json:"id"`
	// Key e a chave canonica (partN/<dir>/<base>).
	Key string `json:"key"`
	// Label e o nome-base da unidade, SEM o sufixo de idioma.
	Label string `json:"label"`
	// Type e a categoria de CONTEUDO (system, mission, event, zone, dlc, text).
	Type string `json:"type,omitempty"`
	// Kind e a FAMILIA do texto, pelo diretorio (ac/resident/zone/event/
	// scenario). E o escopo de export/import: o FF13 tem um formato de
	// texto so (o .ztr), entao a pasta e o que separa os tipos.
	Kind string `json:"kind,omitempty"`
	// Langs sao os idiomas presentes na unidade (o _us e o editavel).
	Langs []string `json:"langs"`
	// Primary e o arquivo editavel (relativo a data/partN).
	Primary string `json:"primary"`
	// Files mapeia idioma -> arquivo (relativo a data/partN).
	Files map[string]string `json:"files"`
	// Container e o tipo do primario: "ztr" (texto solto) ou "wpd" (container).
	Container string `json:"container"`
}

// PartInfo e o resumo de um profile da trilogia FF13 que a UI consome:
// ID estavel (part1/part2/part3), nome de exibicao, game code do ZTR e o
// perfil de idiomas.
type PartInfo struct {
	ID        string   `json:"id"`
	Display   string   `json:"display"`
	GameCode  string   `json:"gameCode"`
	Languages []string `json:"languages"`
	Reference string   `json:"reference"`
	Target    string   `json:"target"`
}

// FF13Service exposto ao app Wails (apps/ff13). Inicialmente so lista os
// perfis da trilogia; as operacoes de texto/imagem entram nos milestones
// seguintes.
type FF13Service struct{}

func (FF13Service) ListParts() []PartInfo {
	out := []PartInfo{}
	for _, p := range New().Profiles() {
		lp := p.Languages()
		out = append(out, PartInfo{
			ID:        p.ID(),
			Display:   p.DisplayName(),
			GameCode:  p.GameCode(),
			Languages: lp.Codes,
			Reference: lp.ReferenceLanguage,
			Target:    lp.TargetLanguage,
		})
	}
	return out
}

func (FF13Service) Profile(id string) (PartInfo, bool) {
	for _, p := range New().Profiles() {
		if p.ID() == id {
			lp := p.Languages()
			return PartInfo{
				ID:        p.ID(),
				Display:   p.DisplayName(),
				GameCode:  p.GameCode(),
				Languages: lp.Codes,
				Reference: lp.ReferenceLanguage,
				Target:    lp.TargetLanguage,
			}, true
		}
	}
	return PartInfo{}, false
}
