package tags

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

// expand devolve os pares da codepage, na MESMA ordem da lista original.
func (s SpanSet) expand(class Class, cp Codepage) []Tag {
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
