// Artifact Writer/Reader: a camada de ARQUIVO dos formatos, padronizada
// sobre a interface Formatter (byte-level). O construtor recebe os
// parâmetros (formatter, collection, caminho, opções) e devolve uma
// instância PRONTA; o chamador decide QUANDO gravar/lêr e como lidar com
// os erros — Write/Read não recebem nada.
//
// A gravação/leitura é a mesma para TODOS os formatos (json, strings,
// csv, xlsx e variantes no-dedup): o byte-level é do Formatter recebido,
// tipicamente vindo do registry (Get). Naming continua decisão do
// chamador — os helpers de caminho vivem na família (ffx/artifactpaths) e
// SwapExt troca a extensão para o formato pedido.
package formatters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffresources/internal/shared/dto"
	logging "ffresources/internal/shared/logging"
)

// Writer é o artefato pronto para gravar: serializa a Collection no
// formato do Formatter recebido no construtor e escreve no caminho.
type Writer interface {
	// Write serializa e grava o arquivo, devolvendo o caminho escrito.
	Write() (string, error)
}

// Reader é o artefato pronto para ler: lê o arquivo no formato do
// Formatter recebido no construtor e devolve a Collection (DTO).
type Reader interface {
	// Read lê e parseia o arquivo de volta para a Collection (DTO).
	Read() (dto.Collection, error)
}

// OptionsMarshaler é implementado pelos formatters que entendem as opções
// neutras (idiomas, identidade, dedup).
type OptionsMarshaler interface {
	MarshalOptions(c dto.Collection, opts Options) ([]byte, error)
}

// OptionsUnmarshaler é o equivalente na leitura.
type OptionsUnmarshaler interface {
	UnmarshalOptions(data []byte, opts Options) (dto.Collection, error)
}

// NewWriter constrói o gravador do artefato. O 4º argumento aceita Options
// (perfil completo) ou []string (filtro de idiomas, contrato legado).
func NewWriter(f Formatter, c dto.Collection, path string, optsOrLangs any) Writer {
	return &artifactWriter{f: f, c: c, path: path, opts: optionsOf(optsOrLangs)}
}

// NewWriterLangs mantém o contrato antigo (só filtro de idiomas).
func NewWriterLangs(f Formatter, c dto.Collection, path string, langs []string) Writer {
	opts := DefaultOptions()
	opts.Languages = langs
	return NewWriter(f, c, path, opts)
}

// NewReader constrói o leitor do artefato. O 3º argumento é opcional e aceita
// Options (perfil completo; default neutro).
func NewReader(f Formatter, path string, optsAny ...any) Reader {
	opts := DefaultOptions()
	if len(optsAny) > 0 {
		opts = optionsOf(optsAny[0])
	}
	return &artifactReader{f: f, path: path, opts: opts}
}

// NewReaderDefault mantém o contrato antigo (sem opções).
func NewReaderDefault(f Formatter, path string) Reader {
	return NewReader(f, path, DefaultOptions())
}

// optionsOf normaliza o argumento legado/perfil.
func optionsOf(v any) Options {
	switch t := v.(type) {
	case Options:
		return t
	case []string:
		opts := DefaultOptions()
		opts.Languages = t
		return opts
	default:
		return DefaultOptions()
	}
}

// SwapExt troca a extensão do caminho pela do formatter — o naming
// canônico é o mesmo basename para todo formato, só a extensão muda
// (no-op quando o caminho já tem a extensão do formatter).
func SwapExt(f Formatter, path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + f.Extension()
}

// artifactWriter é a implementação única de Writer para todos os formatos.
type artifactWriter struct {
	f    Formatter
	c    dto.Collection
	path string
	opts Options
}

// Write serializa (respeitando as opções) e grava no disco.
func (w *artifactWriter) Write() (string, error) {
	if len(w.c) == 0 {
		return "", fmt.Errorf("no text data to export")
	}
	var raw []byte
	var err error
	if om, ok := w.f.(OptionsMarshaler); ok {
		raw, err = om.MarshalOptions(w.c, w.opts)
	} else {
		raw, err = w.f.MarshalLangs(w.c, w.opts.Languages)
	}
	if err != nil {
		return "", err
	}
	if w.opts.CheckWrite != nil {
		if err := w.opts.CheckWrite(w.path); err != nil {
			return "", fmt.Errorf("error writing %s file %s: %w", w.f.Extension(), w.path, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return "", fmt.Errorf("error creating directory for %s: %w", w.path, err)
	}
	if err := os.WriteFile(w.path, raw, 0o644); err != nil {
		return "", fmt.Errorf("error writing %s file %s: %w", w.f.Extension(), w.path, err)
	}
	logging.Verbose("Exported %s artifact: %s", w.f.Extension(), w.path)
	return w.path, nil
}

// artifactReader é a implementação única de Reader para todos os formatos.
type artifactReader struct {
	f    Formatter
	path string
	opts Options
}

// Read lê o arquivo e parseia de volta para a Collection.
func (r *artifactReader) Read() (dto.Collection, error) {
	raw, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s file %s: %w", r.f.Extension(), r.path, err)
	}
	var c dto.Collection
	if ou, ok := r.f.(OptionsUnmarshaler); ok {
		c, err = ou.UnmarshalOptions(raw, r.opts)
	} else {
		c, err = r.f.Unmarshal(raw)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load %s file %s: %w", r.f.Extension(), r.path, err)
	}
	return c, nil
}
