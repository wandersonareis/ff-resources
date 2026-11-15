//go:build e2e

package service

// E2E do catálogo de tags: sobe o app em MODO SERVER (`wails3 run:server`),
// chama o binding GetTagCatalog pelo transporte HTTP do Wails e valida o
// payload que o autocomplete do editor consome.
//
// Uso:
//
//	go build -tags server -o bin/ff13-server.exe ./apps/ff13
//	go test -tags e2e ./apps/ff13/backend/service -run TestTagCatalogE2E -v
//
// O teste é.opcional (build tag `e2e`): ele precisa do binário de server, que
// não faz parte do gate de unidade. O que ele garante além dos testes de
// unidade: o binding existe de ponta a ponta e os valores que o servidor
// oferece são exatamente os que o codec emite (o editor não pode sugerir uma
// tag que o encode não aceite).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runtimeEnvelope é o formato que @wailsio/runtime envia para /wails/runtime.
type runtimeEnvelope struct {
	Object int               `json:"object"` // 0 = Call
	Method int               `json:"method"` // 0 = CallBinding
	Args   map[string]any    `json:"args"`   // AsMap: call-id + CallOptions
	Extra  map[string]string `json:"-"`
}

type tagSuggestionJSON struct {
	Tag   string `json:"tag"`
	Label string `json:"label"`
	Key   string `json:"key"`
}

type tagCatalogEntryJSON struct {
	Tag    string              `json:"tag"`
	Prefix string              `json:"prefix"`
	Hint   string              `json:"hint"`
	Values []tagSuggestionJSON `json:"values"`
}

type tagCatalogJSON struct {
	Version string                `json:"version"`
	Lang    string                `json:"lang"`
	Tags    []tagCatalogEntryJSON `json:"tags"`
}

const (
	serverBinary = "../../bin/ff13-server.exe"
	runtimePath  = "/wails/runtime"
	healthPath   = "/health"
	// Método do binding no formato <pacote>.<Tipo>.<Método> (o FQN é o
	// PackagePath do tipo, não o nome do pacote Go).
	bindingMethod = "ffresources/apps/ff13/backend/service.FF13Service.GetTagCatalog"
)

// startServer sobe o binário de server e espera o /health.
func startServer(t *testing.T) (baseURL string, stop func()) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(wd, serverBinary)
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("binário de server ausente (%s): rode `go build -tags server -o %s .`", bin, serverBinary)
	}

	cmd := exec.Command(bin)
	appRoot := filepath.Join(wd, "..", "..")
	cmd.Dir = appRoot
	cmd.Env = append(os.Environ(), "FF_RESOURCES_DATA="+filepath.Join(appRoot, "bin", "data"))
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start do server: %v", err)
	}

	base := "http://localhost:8080"
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(base + healthPath); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base, func() {
					_ = cmd.Process.Kill()
					_, _ = cmd.Process.Wait()
					if t.Failed() {
						t.Logf("log do server:\n%s", out.String())
					}
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	t.Fatalf("server não respondeu em /health:\n%s", out.String())
	return "", nil
}

// callBinding invoca um método de serviço pelo transporte HTTP do Wails.
func callBinding(t *testing.T, base, method string, args ...any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"object": 0,
		"method": 0,
		"args": map[string]any{
			"call-id":    "e2e-test",
			"methodName": method,
			"args":       args,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, base+runtimePath, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-wails-client-id", "e2e-test")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("chamada de %s: %v", method, err)
	}
	defer resp.Body.Close()
	data := new(bytes.Buffer)
	if _, err := data.ReadFrom(resp.Body); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: HTTP %d: %s", method, resp.StatusCode, data.String())
	}
	if strings.HasPrefix(data.String(), "Binding call failed") {
		t.Fatalf("%s: %s", method, data.String())
	}
	return data.Bytes()
}

// TestTagCatalogE2E valida o catálogo servido pelo app em modo server: as
// classes do FF13, os valores por part e — o mais importante — que cada valor
// sugerido é uma tag que o ENCODE aceita (senão o editor sugeriria algo que
// viraria texto literal no binário).
func TestTagCatalogE2E(t *testing.T) {
	base, stop := startServer(t)
	defer stop()

	// Contagens por part (composição do codec — scripts/tagtables_check).
	wantCounts := map[string]map[string]int{
		"part1": {"CLR": 43, "ICON": 54, "BTN": 33, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "SPACE": 1},
		"part2": {"CLR": 31, "ICON": 88, "BTN": 33, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "SPACE": 1},
		"part3": {"CLR": 31, "ICON": 90, "BTN": 35, "VAR": 4, "CTR": 3, "KEY": 1, "PAGE": 1, "NL": 1, "SPACE": 1},
	}

	for _, part := range []string{"part1", "part2", "part3"} {
		t.Run(part, func(t *testing.T) {
			raw := callBinding(t, base, bindingMethod, part, "us")
			var cat tagCatalogJSON
			if err := json.Unmarshal(raw, &cat); err != nil {
				t.Fatalf("payload do catálogo: %v (%s)", err, truncate(raw, 300))
			}
			if cat.Version != part || cat.Lang != "us" {
				t.Errorf("version/lang = %q/%q, queria %q/us", cat.Version, cat.Lang, part)
			}

			got := map[string][]tagSuggestionJSON{}
			for _, e := range cat.Tags {
				got[e.Tag] = e.Values
			}
			for _, cls := range []string{"CLR", "ICON", "BTN", "VAR", "CTR", "KEY", "PAGE", "NL", "SPACE"} {
				values, ok := got[cls]
				if !ok {
					t.Fatalf("classe %s ausente do catálogo", cls)
				}
				if want := wantCounts[part][cls]; len(values) != want {
					t.Errorf("%s: %d valores, queria %d", cls, len(values), want)
				}
			}

			// Todo valor sugerido tem que codificar: o editor não pode
			// oferecer tag que o encode não aceite.
			for _, e := range cat.Tags {
				for _, v := range e.Values {
					inner := strings.TrimSuffix(strings.TrimPrefix(v.Tag, "{"), "}")
					if b, ok := encodeTagForTest(part, inner); !ok {
						t.Errorf("%s: valor sugerido não codifica: %s", e.Tag, v.Tag)
					} else if len(b) == 0 {
						t.Errorf("%s: codificou para zero bytes: %s", e.Tag, v.Tag)
					}
				}
			}

			// Amostras fixas (marcam o formato que o tradutor vê).
			assertHas(t, got["CLR"], "{CLR:F9:40:White}", "White")
			assertHas(t, got["VAR"], "{VAR:F4:40}", "F4:40")
			assertHas(t, got["NL"], "{NL:40:72}", "40:72")
			assertHas(t, got["SPACE"], "{SPACE:85:60}", "NBSP")
			if part == "part1" {
				assertHas(t, got["ICON"], "{ICON:F0:40:Clock}", "Clock")
			}
			if part == "part3" {
				// O mesmo nome em bytes diferentes por jogo: o par do ff133.
				assertHas(t, got["ICON"], "{ICON:F2:5C:Shield}", "Shield")
			}
		})
	}
}

func assertHas(t *testing.T, values []tagSuggestionJSON, tag, label string) {
	t.Helper()
	for _, v := range values {
		if v.Tag == tag {
			if v.Label != label {
				t.Errorf("%s: label = %q, queria %q", tag, v.Label, label)
			}
			return
		}
	}
	t.Errorf("%s não está entre os %d valores sugeridos", tag, len(values))
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
