package hirlevel

import (
	"os"
	"strings"
	"testing"
)

// A beépített sablonok maguk „csak világos”-ak: az alkalmazás a kitöltött levélhez sem nyúl.
func TestLightOnlyKeepsBuiltins(t *testing.T) {
	tpls, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	for _, tpl := range tpls {
		if !declaresLightOnly(tpl.Raw) {
			t.Errorf("%s: a sablon nem jelöli magát csak világosnak", tpl.ID)
		}
		got, _ := tpl.Render(sampleRenderDataFor(t, tpl.ID))
		if LightOnly(got) != got {
			t.Errorf("%s: LightOnly módosította a szabály szerinti levelet", tpl.ID)
		}
	}
}

// A régi („light dark”) sablonok kész levele csak világos jelölést kap.
func TestLightOnlyLegacy(t *testing.T) {
	b, err := os.ReadFile("testdata/kerdes-sablon.html")
	if err != nil {
		t.Fatal(err)
	}
	out := LightOnly(string(b))
	if strings.Contains(out, "light dark") {
		t.Error("maradt „light dark” jelölés")
	}
	if !strings.Contains(out, `<meta name="color-scheme" content="light only">`) ||
		!strings.Contains(out, `<meta name="supported-color-schemes" content="light only">`) ||
		strings.Count(out, lightOnlyCSS) != 1 || !declaresLightOnly(out) {
		t.Error("hiányzik a csak világos jelölés")
	}
	if LightOnly(out) != out {
		t.Error("másodszori futtatásra változott")
	}
	// jelölés és <style> nélküli sablon
	out = LightOnly("<html><head><title>x</title></head><body>y</body></html>")
	if !strings.Contains(out, `<meta name="color-scheme" content="light only">`) ||
		!strings.Contains(out, `<meta name="supported-color-schemes" content="light only">`) ||
		!strings.Contains(out, "<style>"+lightOnlyCSS+"</style>\n</head>") {
		t.Errorf("%s", out)
	}
}
