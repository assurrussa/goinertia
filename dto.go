package goinertia

// PageDTO type.
type PageDTO struct {
	Component      string                      `json:"component"`
	Props          map[string]any              `json:"props"`
	URL            string                      `json:"url"`
	Version        string                      `json:"version"`
	EncryptHistory bool                        `json:"encryptHistory,omitempty"`
	ClearHistory   bool                        `json:"clearHistory,omitempty"`
	DeferredProps  map[string][]string         `json:"deferredProps,omitempty"`
	MergeProps     []string                    `json:"mergeProps,omitempty"`
	PrependProps   []string                    `json:"prependProps,omitempty"`
	DeepMergeProps []string                    `json:"deepMergeProps,omitempty"`
	MatchPropsOn   []string                    `json:"matchPropsOn,omitempty"`
	ScrollProps    map[string]ScrollPropConfig `json:"scrollProps,omitempty"`
	OnceProps      map[string]OncePropConfig   `json:"onceProps,omitempty"`
}

// SsrDTO type.
type SsrDTO struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}

// ScrollPropConfig defines pagination metadata for infinite scroll props.
type ScrollPropConfig struct {
	PageName     string `json:"pageName,omitempty"`
	PreviousPage any    `json:"previousPage,omitempty"`
	NextPage     any    `json:"nextPage,omitempty"`
	CurrentPage  any    `json:"currentPage,omitempty"`
}

// OncePropConfig defines a once prop configuration.
// ExpiresAt is a unix timestamp in milliseconds. Nil encodes as null.
type OncePropConfig struct {
	Prop      string `json:"prop"`
	ExpiresAt *int64 `json:"expiresAt"`
}
