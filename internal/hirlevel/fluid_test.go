package hirlevel

import (
	"os"
	"strings"
	"testing"
)

func TestFluidTiles(t *testing.T) {
	in := `<div style="font-size:0;line-height:0;text-align:left;"><!--[if mso]><table><tr><td width="177"><![endif]-->` +
		`<div class="tile" style="display:inline-block;width:33.333%;max-width:177px;vertical-align:top;">A</div>` +
		`<div class="tile wide" style="display:inline-block;width:50%;vertical-align:top;">B</div></div>`
	out := FluidTiles(in)
	want := `<div class="tile" style="display:inline-block;vertical-align:top;width:33.333%;min-width:33.333%;max-width:100%;width:calc(230400px - 48000%);">`
	if !strings.Contains(out, want) {
		t.Errorf("a tégla nem lett rugalmas:\n%s", out)
	}
	if !strings.Contains(out, `width:50%;vertical-align:top;">B`) {
		t.Errorf("max-width nélküli téglához nem szabad nyúlni:\n%s", out)
	}
	if !strings.Contains(out, `<div style="font-size:0;line-height:0;text-align:left;">`) {
		t.Errorf("a rács burkolója megváltozott:\n%s", out)
	}
	if FluidTiles(out) != out {
		t.Error("másodszori futtatásra változott")
	}
	if s := `<p>nincs tégla</p>`; FluidTiles(s) != s {
		t.Error("tégla nélküli levél megváltozott")
	}
}

// A beépített sablonok kész levelében minden termék rugalmas: a <style> blokkot eldobó vagy a
// média-lekérdezést nem alkalmazó levelezőkben ettől kerülnek egymás alá a termékek.
func TestFluidTilesBuiltin(t *testing.T) {
	for _, id := range []string{"v1-sotet-lemez", "v2-waterside"} {
		b, err := os.ReadFile("../../sablonok/" + id + ".html")
		if err != nil {
			t.Fatal(err)
		}
		n := strings.Count(string(b), `class="tile"`)
		if n == 0 {
			t.Fatalf("%s: nincs tégla a sablonban", id)
		}
		if got := strings.Count(FluidTiles(string(b)), "max-width:100%;width:calc("); got != n {
			t.Errorf("%s: %d/%d tégla rugalmas", id, got, n)
		}
	}
}
