package languages

import (
	"reflect"
	"testing"
)

// TestFF13Catalog fixa o catálogo exclusivo da trilogia: os oito idiomas do
// jogo mais as traduções comunitárias ck/uk/ru, sem duplicatas. O inglês é o
// slot editável e vem primeiro.
func TestFF13Catalog(t *testing.T) {
	want := []string{"us", "jp", "fr", "gr", "it", "sp", "ch", "kr", "ck", "uk", "ru"}
	if got := FF13Codes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("FF13Codes() = %v, queria %v", got, want)
	}
	seen := make(map[string]bool, len(want))
	for _, l := range FF13 {
		if l.Code == "" || l.Name == "" {
			t.Fatalf("idioma sem código/nome: %+v", l)
		}
		if seen[l.Code] {
			t.Fatalf("código duplicado: %q", l.Code)
		}
		seen[l.Code] = true
	}
}
