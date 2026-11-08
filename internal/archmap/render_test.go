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

func TestRenderWritesEmptyListsNotNull(t *testing.T) {
	var out bytes.Buffer
	if err := Render(&out, Page{Map: Build(nil, func(string) []byte { return nil }, nil, Thresholds{MinSupport: 5, MinRatio: 0.3})}); err != nil {
		t.Fatal(err)
	}
	for _, list := range []string{`"units":[]`, `"imports":[]`, `"cochanges":[]`} {
		if !strings.Contains(out.String(), list) {
			t.Errorf("the page script needs %s, never null", list)
		}
	}
}
