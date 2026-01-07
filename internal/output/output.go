// Package output handles formatting output in different formats.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Format represents an output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Writer handles output in the specified format.
type Writer struct {
	format Format
	w      io.Writer
}

// NewWriter creates a new output writer.
func NewWriter(w io.Writer, format Format) *Writer {
	return &Writer{format: format, w: w}
}

// Write outputs the given value in the configured format.
func (w *Writer) Write(v interface{}) error {
	switch w.format {
	case FormatJSON:
		enc := json.NewEncoder(w.w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case FormatYAML:
		enc := yaml.NewEncoder(w.w)
		enc.SetIndent(2)
		return enc.Encode(v)
	default:
		// Text format - assume v implements fmt.Stringer or use default
		if s, ok := v.(fmt.Stringer); ok {
			_, err := fmt.Fprintln(w.w, s.String())
			return err
		}
		_, err := fmt.Fprintf(w.w, "%+v\n", v)
		return err
	}
}

// ParseFormat parses a format string into a Format.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown format: %s", s)
	}
}
