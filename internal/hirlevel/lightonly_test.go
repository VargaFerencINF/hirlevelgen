package hirlevel

import (
	"os"
	"strings"
	"testing"
)

func TestLightOnly(t *testing.T) {
	for _, id := range []string{"v1-sotet-lemez", "v2-waterside", "v6-osz-avar"} {
		b, err := os.ReadFile("../../sablonok/" + id + ".html")
		if err != nil {
			t.Fatal(err)
		}
		out := LightOnly(string(b))
		if strings.Contains(out, "light dark") {
			t.Errorf("%s: maradt „light dark” jelölés", id)
		}
		if !strings.Contains(out, `<meta name="color-scheme" content="light only">`) ||
			!strings.Contains(out, `<meta name="supported-color-schemes" content="light only">`) ||
			strings.Count(out, lightOnlyCSS) != 1 {
			t.Errorf("%s: hiányzik a csak világos jelölés", id)
		}
		if LightOnly(out) != out {
			t.Errorf("%s: másodszori futtatásra változott", id)
		}
	}
	// jelölés és <style> nélküli sablon
	out := LightOnly("<html><head><title>x</title></head><body>y</body></html>")
	if !strings.Contains(out, `<meta name="color-scheme" content="light only">`) ||
		!strings.Contains(out, `<meta name="supported-color-schemes" content="light only">`) ||
		!strings.Contains(out, "<style>"+lightOnlyCSS+"</style>\n</head>") {
		t.Errorf("%s", out)
	}
}
