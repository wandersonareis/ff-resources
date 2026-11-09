package encoding

// TABELAS COMPARTILHADAS: uma lista de NOMES (comum a todas as codepages) e,
// por codepage, os SPANS de bytes que a preenchem.
//
// É a resposta a "as cores repetidas mudam só os bytes": o nome vive uma vez
// só e o que varia é o span. Ex. {COLOR:White} em três codepages:
//
//	xiii2Colors = SpanSet{
//	    Names: []string{"White", "IceBlue", ...},
//	    Spans: map[Codepage][]Span{
//	        LatinJapanese: {{Lead: 0xF9, Trail: 0x40, Count: 31, At: 0}},
//	        Chinese:       {{Lead: 0xFE, Trail: 0x40, Count: 31, At: 0}},
//	        Korean:        {{Lead: 0xAC, Trail: 0xA1, Count: 31, At: 0}},
//	    },
//	}
//
// O span também absorve os BURACOS da tabela (ff131 tem trails ausentes):
// cada faixa contígua é um span, e `At` diz em que índice de Names ela
// começa — a ordem de consulta (e portanto o reverse por nome, que guarda a
// primeira ocorrência) é preservada.

// Span é uma faixa contígua de pares: (Lead, Trail), (Lead, Trail+1), ...
// Count itens, iniciando no índice At de SpanSet.Names.
type Span struct {
	Lead  byte
	Trail byte
	Count int
	At    int
}

// SpanSet é uma tabela compartilhada nomes + spans por codepage.
type SpanSet struct {
	Names []string
	Spans map[Codepage][]Span
}

// expand devolve os pares da codepage, na MESMA ordem da lista original —
// é o que o decode/encode consomem (o KeySet continua com []Tag).
func (s SpanSet) expand(class TagClass, cp Codepage) []Tag {
	spans := s.Spans[cp]
	total := 0
	for _, sp := range spans {
		total += sp.Count
	}
	out := make([]Tag, 0, total)
	for _, sp := range spans {
		for i := 0; i < sp.Count; i++ {
			out = append(out, Tag{
				Class: class,
				B1:    sp.Lead,
				B2:    sp.Trail + byte(i),
				Name:  s.Names[sp.At+i],
			})
		}
	}
	return out
}
