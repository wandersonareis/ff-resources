package ztr

import (
	"bytes"
	"fmt"
)

// txtSeparator divide ID e texto em cada linha do .txt (formato de referência).
const txtSeparator = " |:| "

// WriteTxt monta o arquivo de texto "id |:| texto\r\n" a partir dos IDs e
// dos textos JÁ decodificados (core/encoding).
func WriteTxt(ids, texts []string) []byte {
	var out bytes.Buffer
	for i := range ids {
		out.WriteString(ids[i])
		out.WriteString(txtSeparator)
		out.WriteString(texts[i])
		out.WriteString("\r\n")
	}
	return out.Bytes()
}

// ReadTxt interpreta um .txt de volta em IDs e textos. Aceita \r\n e \n.
// A divisão é pela PRIMEIRA ocorrência do separador — melhoria em relação
// ao C#, que truncaria um texto contendo " |:| " (Split()[1] perde o resto).
func ReadTxt(data []byte) ([]string, []string, error) {
	nn := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := bytes.Split(nn, []byte("\n"))
	if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
		lines = lines[:n-1] // trailing newline não é linha
	}
	ids := make([]string, 0, len(lines))
	texts := make([]string, 0, len(lines))
	for i, line := range lines {
		idx := bytes.Index(line, []byte(txtSeparator))
		if idx < 0 {
			return nil, nil, fmt.Errorf("ztr: linha %d sem separador %q", i+1, txtSeparator)
		}
		ids = append(ids, string(line[:idx]))
		texts = append(texts, string(line[idx+len(txtSeparator):]))
	}
	return ids, texts, nil
}
