package dto

// YkdScreen é a definição de uma tela de UI (arquivo .ykd) pronta para o
// preview: o grafo de nós, os sprites com as texturas já decodificadas
// (data URLs) e uma lista PLANA de itens de desenho na ordem dos blocos.
type YkdScreen struct {
	// CanvasWidth/CanvasHeight são a área de desenho do jogo (1280x720).
	CanvasWidth  int `json:"canvasWidth"`
	CanvasHeight int `json:"canvasHeight"`

	Header    YkdHeader     `json:"header"`
	Blocks    []YkdBlock    `json:"blocks"`
	Resources []YkdResource `json:"resources"`
	Items     []YkdItem     `json:"items"`

	// Textures é indexado pelo NOME do recurso (o mesmo de YkdItem.Texture).
	Textures map[string]YkdTexture `json:"textures"`

	// MissingTextures são nomes referenciados que não resolveram em nenhum
	// container do part (o preview mostra placeholder).
	MissingTextures []string `json:"missingTextures"`

	// Siblings são as outras telas .ykd do MESMO container (seletor rápido).
	Siblings []YkdScreenRef `json:"siblings"`

	Stats YkdStats `json:"stats"`
}

// YkdScreenRef identifica outra tela .ykd do mesmo container.
type YkdScreenRef struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

// YkdHeader reproduz os campos desconhecidos do header do arquivo.
type YkdHeader struct {
	Unknown1 int32 `json:"unknown1"`
	U2       int   `json:"u2"`
	U3       int   `json:"u3"`
	U4       int   `json:"u4"`
	U5       int   `json:"u5"`
}

// YkdBlock é um nó da tela (transform local + animações + refs).
type YkdBlock struct {
	Type        uint32 `json:"type"`
	Index       uint32 `json:"index"`
	ParentIndex uint32 `json:"parentIndex"` // 0xFFFFFFFF = raiz
	Unknown     uint32 `json:"unknown"`

	Scale    [4]float32 `json:"scale"`
	Rotation [4]float32 `json:"rotation"`
	Position [4]float32 `json:"position"`
	Pivot    [4]float32 `json:"pivot"`
	Extra    [4]float32 `json:"extra"`

	ResourceIndexes []int32  `json:"resourceIndexes"`
	Text            *YkdText `json:"text,omitempty"`
	Entries         []string `json:"entries"`
}

// YkdResource é um sprite (recorte de textura + tamanho + cores).
type YkdResource struct {
	Index    int32  `json:"index"`
	Type     int32  `json:"type"`
	TypeName string `json:"typeName"`
	Name     string `json:"name"`

	SourceX      int32 `json:"sourceX"`
	SourceY      int32 `json:"sourceY"`
	SourceWidth  int32 `json:"sourceWidth"`
	SourceHeight int32 `json:"sourceHeight"`
	ViewWidth    int32 `json:"viewWidth"`
	ViewHeight   int32 `json:"viewHeight"`
	Flags        int32 `json:"flags"`

	// Colors são BGRA (top-left, bottom-left, top-right, bottom-right).
	Colors [4]uint32 `json:"colors"`

	// Texture é a chave em YkdScreen.Textures ("" = não resolvida).
	Texture string `json:"texture,omitempty"`
	Missing bool   `json:"missing"`
}

// YkdItem é uma instrução de desenho já com o transform de mundo aproximado.
type YkdItem struct {
	Kind       string `json:"kind"` // "sprite" | "text"
	BlockIndex int    `json:"blockIndex"`
	BlockType  uint32 `json:"blockType"`
	NodeIndex  uint32 `json:"nodeIndex"`

	// Canto superior esquerdo em pixels da tela.
	X float32 `json:"x"`
	Y float32 `json:"y"`

	ScaleX   float32 `json:"scaleX"`
	ScaleY   float32 `json:"scaleY"`
	Rotation float32 `json:"rotation"` // radianos
	PivotX   float32 `json:"pivotX"`
	PivotY   float32 `json:"pivotY"`
	FlipX    bool    `json:"flipX"`
	FlipY    bool    `json:"flipY"`

	ResourceIndex int32 `json:"resourceIndex"`
	// Texture é a chave em YkdScreen.Textures; vazio/ausente = placeholder.
	Texture string `json:"texture,omitempty"`
	Missing bool   `json:"missing"`

	SourceX      int32 `json:"sourceX"`
	SourceY      int32 `json:"sourceY"`
	SourceWidth  int32 `json:"sourceWidth"`
	SourceHeight int32 `json:"sourceHeight"`
	ViewWidth    int32 `json:"viewWidth"`
	ViewHeight   int32 `json:"viewHeight"`
	Flags        int32 `json:"flags"`

	Text *YkdText `json:"text,omitempty"`
}

// YkdText são os parâmetros da CAIXA de texto. O CONTEÚDO do texto não está
// no .ykd: o jogo o fornece pelo sistema de mensagens. Width/Height limitam
// quantos caracteres cabem; FontSize é o tamanho da fonte; Flags é o modo de
// exibição (alinhamento/quebra/rolagem).
type YkdText struct {
	Width    int32 `json:"width"`
	Height   int32 `json:"height"`
	FontSize int32 `json:"fontSize"`
	Flags    int32 `json:"flags"`
	Color1   int32 `json:"color1"`
	Color2   int32 `json:"color2"`
	Color3   int32 `json:"color3"`
	Mode     int32 `json:"mode"`
	// OriginalWidth é a largura PRISTINE (data/), base do teto e do botão
	// "restaurar". Pode diferir de Width quando já existe mod salvo.
	OriginalWidth int32 `json:"originalWidth"`
	// MaxWidth é o teto seguro para esta caixa: 2× a largura ORIGINAL (ou
	// original+64 em caixas pequenas), nunca além da borda direita da tela.
	MaxWidth int32 `json:"maxWidth"`
}

// YkdTextPatch é a edição da LARGURA de uma caixa de texto. Só a largura é
// alterada: altura/fonte/flags/posição ficam como no arquivo original.
// BlockIndex é o ORDINAL do bloco no arquivo (o índice da lista de blocos,
// não o `Index` do nó).
type YkdTextPatch struct {
	BlockIndex int   `json:"blockIndex"`
	Width      int32 `json:"width"`
}

// YkdTexture é uma textura decodificada para o preview.
type YkdTexture struct {
	PNGData string `json:"pngData"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	// Source é o registro WPD de origem ("nome.ext").
	Source string `json:"source"`
}

// YkdStats resume a tela para o cabeçalho do painel.
type YkdStats struct {
	Blocks          int `json:"blocks"`
	Resources       int `json:"resources"`
	Items           int `json:"items"`
	Textures        int `json:"textures"`
	MissingTextures int `json:"missingTextures"`
	TextNodes       int `json:"textNodes"`
	TexturedItems   int `json:"texturedItems"`
}
