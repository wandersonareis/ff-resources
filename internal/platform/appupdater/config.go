package appupdater

// Config configura o updater de UM app. O pacote é neutro: cada app diz de
// qual repositório/releases ele vem, qual prefixo de asset é dele e como
// obter o tema da janela.
type Config struct {
	// Repository é "owner/repo" no GitHub. Obrigatório.
	Repository string
	// AssetPrefix restringe os assets ao app (ex.: "spira"/"pulse"): releases
	// do mesmo repo contêm zips de mais de um app.
	AssetPrefix string
	// ChecksumAsset é o sidecar de hashes (default "SHA256SUMS").
	ChecksumAsset string
	// WindowTitle é o título da janela nativa de update.
	WindowTitle string
	// Theme devolve o tema atual do app ("dark"/"light"). Nil = dark.
	Theme func() string
}
