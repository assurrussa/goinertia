package goinertia

// PageDTO type.
type PageDTO struct {
	Component      string         `json:"component"`
	Props          map[string]any `json:"props"`
	URL            string         `json:"url"`
	Version        string         `json:"version"`
	EncryptHistory bool           `json:"encryptHistory,omitempty"`
	ClearHistory   bool           `json:"clearHistory,omitempty"`
}

// SsrDTO type.
type SsrDTO struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}
