package types

// GeneratedFile represents the structure expected from the LLM for each file.
type GeneratedFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"` // e.g., "tsx", "css", "json"
	Content  string `json:"content"`
}
