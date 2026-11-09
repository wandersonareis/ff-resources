// Package launcher concentra o contrato do manifesto que guia a tela inicial
// (apps/launcher). Nao importa nenhuma familia: o launcher so sabe abrir apps.
package launcher

import (
	"encoding/json"
	"fmt"
	"os"
)

// AppEntry descreve um app no manifesto do launcher.
type AppEntry struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Exe      string       `json:"exe"`
	Cover    string       `json:"cover,omitempty"`
	Profiles []ProfileRef `json:"profiles,omitempty"`
}

// ProfileRef e um profile/versoes que o app abre com --profile <id>.
type ProfileRef struct {
	ID     string `json:"id"`
	Display string `json:"display"`
	Cover  string `json:"cover,omitempty"`
}

// Manifest e o conteudo de manifest.json.
type Manifest struct {
	Apps []AppEntry `json:"apps"`
}

// LoadManifest le e parseia o manifest.json ao lado do executavel.
func LoadManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("manifest invalido %q: %w", path, err)
	}
	return &m, nil
}
