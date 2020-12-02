package types

type File struct {
	Name     string `json:"name,omitempty"`
	Contents string `json:"contents,omitempty"`
}
