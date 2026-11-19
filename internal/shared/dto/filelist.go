package dto

// FilelistRestoreResult é o retrato do "reverter inserções": quantos white
// bins foram truncados, quantos filelists voltaram do backup e o total de
// bytes devolvidos ao tamanho original.
type FilelistRestoreResult struct {
	Applied   int      `json:"applied"`
	Bins      int      `json:"bins"`
	Filelists int      `json:"filelists"`
	Bytes     int64    `json:"bytes"`
	Warnings  []string `json:"warnings"`
}
