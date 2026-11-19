package dto

// ExportOptions escolhe o que o Exportar produz para o escopo selecionado:
//
//	Decode  → "Descompactar": os formatos úteis (artefato de texto ou DDS/PNG).
//	Extract → "Extrair": o binário REAL do filelist/white bin, descomprimido
//	          quando necessário, gravado em data/partN (só vale para filelist).
//
// Os dois são independentes; ambos ligados = cada um faz a sua parte. Nenhum
// ligado = Decode (compatibilidade com chamadas antigas).
type ExportOptions struct {
	Decode  bool `json:"decode"`
	Extract bool `json:"extract"`
}
