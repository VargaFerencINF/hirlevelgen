package hirlevel

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestFluidTiles(t *testing.T) {
	in := `<div style="font-size:0;line-height:0;text-align:left;"><!--[if mso]><table><tr><td width="177"><![endif]-->` +
		`<div class="tile" style="display:inline-block;width:33.333%;max-width:177px;vertical-align:top;">A</div>` +
		`<div class="tile wide" style="display:inline-block;width:50%;vertical-align:top;">B</div></div>`
	out := FluidTiles(in)
	if !strings.Contains(out, `<div class="tile" style="display:inline-block;width:100%;max-width:177px;vertical-align:top;text-align:left;">`) {
		t.Errorf("a tégla nem lett rugalmas:\n%s", out)
	}
	if !strings.Contains(out, `width:50%;vertical-align:top;">B`) {
		t.Errorf("max-width nélküli téglához nem szabad nyúlni:\n%s", out)
	}
	if !strings.Contains(out, `<div style="font-size:0;line-height:0;text-align:center;">`) {
		t.Errorf("a rács nem lett középre igazítva:\n%s", out)
	}
	if FluidTiles(out) != out {
		t.Error("másodszori futtatásra változott")
	}
	if s := `<p>nincs tégla</p>`; FluidTiles(s) != s {
		t.Error("tégla nélküli levél megváltozott")
	}
}

// A beépített sablonok kész levelében egy termék sem maradhat százalékos szélességű:
// a <style> blokkot eldobó telefonos levelezőkben ettől csúsztak egymásra a termékek.
func TestFluidTilesBuiltin(t *testing.T) {
	pct := regexp.MustCompile(`class="tile"[^>]*;width:[1-9]?\d(\.\d+)?%`) // 100% alatti
	for _, id := range []string{"v1-sotet-lemez", "v2-waterside"} {
		b, err := os.ReadFile("../../sablonok/" + id + ".html")
		if err != nil {
			t.Fatal(err)
		}
		if !pct.Match(b) {
			t.Fatalf("%s: a teszt feltételezi a 33,333%%-os téglákat", id)
		}
		out := FluidTiles(string(b))
		if pct.MatchString(out) {
			t.Errorf("%s: maradt százalékos tégla", id)
		}
		if !strings.Contains(out, "max-width:177px") || !strings.Contains(out, "text-align:center") {
			t.Errorf("%s: hiányzik a max-width vagy a középre igazítás", id)
		}
	}
}
