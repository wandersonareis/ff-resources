package dto

// TextImportOptions escolhe os DESTINOS da importação de texto. Os dois são
// independentes:
//
//	WriteMods      → reconstrói o binário (.ztr/container WPD) e grava em
//	                 mods/partN (o destino que a UI chama de "Comprimir");
//	InsertFilelist → insere o binário no filelist (append no white bin +
//	                 reaponta os ponteiros). Não toca em data/.
//
// Sem nenhum dos dois a importação não tem destino: o backend assume
// WriteMods (compatibilidade).
type TextImportOptions struct {
	WriteMods      bool `json:"write_mods"`
	InsertFilelist bool `json:"insert_filelist"`
}
