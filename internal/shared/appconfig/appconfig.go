// Package appconfig lê o arquivo de configuração LOCAL do app: onde estão
// instalados os jogos/ binários que o editor precisa.
//
// O arquivo NUNCA entra no git (é gitignored) e é criado com placeholders
// quando não existe. Os testes usam o MESMO arquivo — nada de caminho
// hardcoded no código.
package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvPath sobrescreve o caminho do arquivo de config (útil em CI).
const EnvPath = "FF_RESOURCES_CONFIG"

// FileName é o nome do arquivo procurado a partir da pasta do app/teste.
const FileName = "ff13.config.json"

// Config é o conteúdo do arquivo local.
type Config struct {
	// Roots: pasta de instalação de cada jogo, por part ("part1", ...).
	Roots map[string]string `json:"roots"`
	// Binaries: caminho de qualquer binário necessário (nome livre ->
	// caminho absoluto). Ex.: {"ffx": "D:\\Games\\FFX\\ffx.exe"}.
	Binaries map[string]string `json:"binaries"`
	// EnableMods: ligado = a tabela mostra a TRADUÇÃO (mods/); desligado =
	// mostra o original. nil = não configurado (default: ligado, mods-first).
	//
	// O ponteiro distingue "não configurado" de "desligado" — sem ele, o
	// default (ligado) e o valor explícito false seriam o mesmo campo.
	EnableMods *bool `json:"enableMods,omitempty"`
}

// Placeholder devolve a config inicial (sem caminhos reais): serve para o
// arquivo ser criado na primeira execução.
func Placeholder() *Config {
	return &Config{
		Roots: map[string]string{
			"part1": `C:\Games\FINAL FANTASY XIII`,
			"part2": `C:\Games\FINAL FANTASY XIII-2`,
			"part3": `C:\Games\LIGHTNING RETURNS FINAL FANTASY XIII`,
		},
		Binaries: map[string]string{},
	}
}

// ResolvePath devolve o caminho do arquivo de config: env > pasta atual >
// pasta do executável. Devolve "" quando não encontra nenhum (aí o Load
// cria o arquivo na pasta atual).
func ResolvePath() string {
	if p := strings.TrimSpace(os.Getenv(EnvPath)); p != "" {
		return p
	}
	if p := findUp(FileName, "."); p != "" {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		if p := findUp(FileName, filepath.Dir(exe)); p != "" {
			return p
		}
	}
	return ""
}

// findUp procura o arquivo subindo a partir de start (no máximo 8 níveis).
func findUp(name, start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, name)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// Load lê a config. Quando o arquivo não existe, cria com placeholders.
// Quando existe mas está inválido, devolve o erro (sem sobrescrever).
func Load() (*Config, string, error) {
	path := ResolvePath()
	if path == "" {
		path = filepath.Join(".", FileName)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, path, err
		}
		cfg := Placeholder()
		if err := Save(path, cfg); err != nil {
			return nil, path, err
		}
		return cfg, path, nil
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, path, fmt.Errorf("config invalida (%s): %w", path, err)
	}
	if cfg.Roots == nil {
		cfg.Roots = map[string]string{}
	}
	if cfg.Binaries == nil {
		cfg.Binaries = map[string]string{}
	}
	return cfg, path, nil
}

// Save grava a config no caminho dado (cria a pasta se preciso).
func Save(path string, cfg *Config) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// Root devolve a raiz configurada de uma part ("" se não houver).
func (c *Config) Root(part string) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Roots[part])
}

// Binary devolve o caminho configurado de um binário ("" se não houver).
func (c *Config) Binary(name string) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Binaries[name])
}

// EnableModsEnabled devolve a preferência de mods: LIGADA por padrão
// (mods-first). nil (não configurado) = ligada.
func (c *Config) EnableModsEnabled() bool {
	if c == nil || c.EnableMods == nil {
		return true
	}
	return *c.EnableMods
}

// SetEnableMods grava a preferência de mods (o ponteiro é sempre gravado:
// nil = não configurado ≠ desligado).
func (c *Config) SetEnableMods(enabled bool) {
	if c == nil {
		return
	}
	v := enabled
	c.EnableMods = &v
}
