package formats

// DataEntry é uma entrada da coleção de data/partN: uma unidade de texto
// (Text=true) ou um arquivo que o app não decodifica (Text=false). Também
// descreve imagens e containers de imagem (Image/ImageContainer).
type DataEntry struct {
	// ID é o id da unidade ("<dir>/<base>") no caso do texto; no caso de um
	// arquivo fora do escopo, o caminho relativo em data/partN.
	ID string `json:"id"`
	// Label é o rótulo: o nome-base da unidade (sem o sufixo de idioma) ou o
	// nome do arquivo.
	Label string `json:"label"`
	// Type é a categoria de conteúdo (system, mission, event, ...).
	Type string `json:"type"`
	// Kind é a FAMÍLIA do texto, pelo diretório (ac/resident/zone/event/
	// scenario). É o escopo de export/import.
	Kind string `json:"kind,omitempty"`
	// Text marca as entradas editáveis (unidades de texto).
	Text bool `json:"text"`
	// Langs são os idiomas presentes na unidade.
	Langs []string `json:"langs,omitempty"`
	// Primary é o arquivo editável da unidade (o "_us").
	Primary string `json:"primary,omitempty"`
	// Files mapeia idioma -> arquivo da unidade.
	Files map[string]string `json:"files,omitempty"`
	// Container é o tipo do arquivo editável ("ztr" | "wpd") ou o tipo do
	// container de imagem ("wpd" | "trb").
	Container string `json:"container,omitempty"`
	// Image marca a entrada como imagem/container de imagem.
	Image bool `json:"image,omitempty"`
	// ImageRef é a referência opaca da imagem/container.
	ImageRef string `json:"imageRef,omitempty"`
	// ImageContainer marca um nó expansível de imagem.
	ImageContainer bool `json:"imageContainer,omitempty"`
	// Pair é o binário .imgb pareado.
	Pair string `json:"pair,omitempty"`
	// Modded marca a unidade que já tem tradução em mods/partN. A árvore
	// usa isso para escolher a cópia certa quando o mesmo caminho visual
	// existe no data/ global e no de zona (as duas são idênticas no
	// original, mas só uma pode estar traduzida).
	Modded bool `json:"modded,omitempty"`
}
