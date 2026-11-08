package archmap

import (
	_ "embed"
	"encoding/json"
	"io"
	"strings"
)

//go:embed page.html
var page string

// Page is a Map plus what the HTML needs to name and link it.
type Page struct {
	Map
	Name       string  `json:"name"`
	Repository string  `json:"repository,omitempty"` // e.g. https://github.com/owner/repo
	Head       string  `json:"head"`
	Generated  string  `json:"generated"`
	MinSupport int     `json:"min_support"`
	MinRatio   float64 `json:"min_ratio"`
	MaxFiles   int     `json:"max_files"`
}

// Render writes the self-contained page with p embedded as JSON.
func Render(w io.Writer, p Page) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	// A "</script>" inside a README must not close the page's script.
	safe := strings.ReplaceAll(string(data), "<", `<`)
	_, err = io.WriteString(w, strings.Replace(page, "/*__DATA__*/ null", safe, 1))
	return err
}
