package archmap

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderEmbedsTheMapWithoutBreakingTheScript(t *testing.T) {
	var out bytes.Buffer
	p := Page{Name: "acme", Map: Map{Units: []Unit{{ID: "api", Title: "API", Job: "</script><b>"}}}}
	if err := Render(&out, p); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Contains(html, "/*__DATA__*/") || !strings.Contains(html, `"name":"acme"`) {
		t.Fatal("data was not embedded")
	}
	if strings.Count(html, "</script>") != 1 {
		t.Error("a README closed the page's script")
	}
}
