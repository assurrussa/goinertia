package core

import (
	"bytes"
	"fmt"
	"html/template"
)

// ExecuteTemplate renders a library-owned template before response commitment.
func ExecuteTemplate(t *template.Template, data any) ([]byte, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}
	return b.Bytes(), nil
}
