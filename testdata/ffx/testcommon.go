// Package testcommon concentra os HELPERS dos testes Go herdados do projeto
// FFX: a raiz dos fixtures, o ambiente do binário de teste e o notificador
// falso.
//
// Os FIXTURES de jogo (testdata/ffx/FFX/..., FFX-2/...) são trazidos
// manualmente e não entram no git (.gitignore); este código, sim, porque as
// importações dos testes apontam para ele.
package testcommon

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetTestDataRootDirectory devolve a pasta dos fixtures (testdata/ffx): é a
// raiz onde os testes montam <raiz>/FFX/binary e <raiz>/FFX-2/binary.
func GetTestDataRootDirectory() string {
	return filepath.Dir(currentFilePath())
}

// SetBuildBinPath aponta APP_BASE_PATH para o diretório de binários do app
// FFX. O monorepo não consome a variável (os caminhos de dados vêm de
// common.GameFilesRoot), mas o contrato dos testes herdados espera sucesso.
func SetBuildBinPath() error {
	root := findRepoRoot()
	candidates := []string{
		filepath.Join(root, "apps", "ffx", "build", "bin"),
		filepath.Join(root, "apps", "ffx", "bin"),
	}
	for _, bin := range candidates {
		if info, err := os.Stat(bin); err == nil && info.IsDir() {
			return os.Setenv("APP_BASE_PATH", bin)
		}
	}
	return nil
}

// findRepoRoot sobe a partir deste arquivo até achar o go.mod do monorepo.
func findRepoRoot() string {
	current := filepath.Dir(currentFilePath())
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			panic("go.mod não encontrado na hierarquia de diretórios")
		}
		current = parent
	}
}

// currentFilePath devolve o caminho deste arquivo em tempo de execução.
func currentFilePath() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("não foi possível resolver o caminho do helper de teste")
	}
	return filename
}
