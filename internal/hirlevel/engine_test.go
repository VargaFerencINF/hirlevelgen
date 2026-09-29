package hirlevel

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// sampleRenderData a tervező mintatartalmából közvetlenül (Build nélkül) készít kitöltési adatot.
func sampleRenderData(t *testing.T) *RenderData {
	t.Helper()
	data, err := os.ReadFile("testdata/tartalom-minta.json")
	if err != nil {
		t.Fatal(err)
	}
	c, products, err := ImportJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	// minden kulcs, a partnerenkéntiekkel együtt
	values := map[string]string{}
	var raw map[string]any
	_ = jsonUnmarshal(data, &raw)
	Flatten(raw, "", values)
	for k, v := range c {
		values[k] = v
	}
	values["rep.photo"] = ""
	items := []map[string]string{}
	for _, p := range products {
		items = append(items, map[string]string{"url": p.URL, "image": p.Image, "imageAlt": p.Alt, "code": p.Code,
			"name": p.Name, "desc": p.Desc, "price": p.Price, "deal": p.Deal, "cta": p.CTA})
	}
	return &RenderData{Values: values, Items: items}
}

func TestOriginalTemplatesMatchDesignerPreview(t *testing.T) {
	d := sampleRenderData(t)
	for _, info := range TemplateInfos {
		src, _ := os.ReadFile("testdata/eredeti/" + info.ID + ".html")
		nodes, _, err := ParseTemplate(lf(src))
		if err != nil {
			t.Fatal(err)
		}
		tpl := &Template{nodes: nodes}
		got, missing := tpl.Render(d)
		want, _ := os.ReadFile("testdata/elonezet/" + info.ID + ".html")
		if len(missing) > 0 {
			t.Errorf("%s: hiányzó kulcsok: %v", info.ID, missing)
		}
		if got != lf(want) {
			t.Errorf("%s: az eredeti sablon kimenete eltér a tervezői előnézettől", info.ID)
		}
	}
}

// A saját (átalakított) sablonok alapesetben bájtra azonos kimenetet adnak.
func TestConvertedTemplatesMatchDesignerPreview(t *testing.T) {
	d := sampleRenderData(t)
	tpls, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	for _, tpl := range tpls {
		got, missing := tpl.Render(d)
		wantB, _ := os.ReadFile("testdata/elonezet/" + tpl.ID + ".html")
		want := lf(wantB)
		if len(missing) > 0 {
			t.Errorf("%s: hiányzó kulcsok: %v", tpl.ID, missing)
		}
		if got != want {
			i := 0
			for i < len(got) && i < len(want) && got[i] == want[i] {
				i++
			}
			lo := i - 120
			if lo < 0 {
				lo = 0
			}
			hiG, hiW := min(i+120, len(got)), min(i+120, len(want))
			t.Errorf("%s: eltérés a(z) %d. bájtnál\n kapott: %q\n várt:   %q", tpl.ID, i, got[lo:hiG], want[lo:hiW])
		}
	}
}

func TestGridVariableCountAndConditionals(t *testing.T) {
	tpls, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	base := sampleRenderData(t)
	for _, tpl := range tpls {
		for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 9, 12} {
			d := &RenderData{Values: map[string]string{}, Items: nil}
			for k, v := range base.Values {
				d.Values[k] = v
			}
			for i := 0; i < n; i++ {
				it := map[string]string{}
				for k, v := range base.Items[i%6] {
					it[k] = v
				}
				it["code"] = "KOD-" + strconv.Itoa(i+1)
				d.Items = append(d.Items, it)
			}
			html, missing := tpl.Render(d)
			if len(missing) > 0 {
				t.Fatalf("%s/%d: hiányzó kulcsok %v", tpl.ID, n, missing)
			}
			if c := strings.Count(html, `<div class="tile"`); c != n {
				t.Errorf("%s/%d: %d tégla", tpl.ID, n, c)
			}
			rows := strings.Count(html, "</td></tr><tr><td width=\"177\"")
			if want := (n - 1) / 3; rows != want {
				t.Errorf("%s/%d: %d MSO sortörés, várt %d", tpl.ID, n, rows, want)
			}
			for i := 1; i <= n; i++ {
				if !strings.Contains(html, ">KOD-"+strconv.Itoa(i)+"<") {
					t.Errorf("%s/%d: hiányzik a KOD-%d", tpl.ID, n, i)
				}
			}
		}
		// fotó és elhagyható részek
		d := &RenderData{Values: map[string]string{}, Items: base.Items}
		for k, v := range base.Values {
			d.Values[k] = v
		}
		d.Values["rep.photo"] = "https://pelda.hu/foto.jpg"
		d.Values["rep.phone.url"] = ""
		d.Values["poll.question"] = ""
		d.Values["social.tiktok.url"] = ""
		html, _ := tpl.Render(d)
		if !strings.Contains(html, `src="https://pelda.hu/foto.jpg"`) || strings.Contains(html, ">NA</td>") {
			t.Errorf("%s: a fotó helyett monogram jelent meg", tpl.ID)
		}
		if strings.Contains(html, "tel:") {
			t.Errorf("%s: üres telefonnál is van tel: link", tpl.ID)
		}
		if strings.Contains(html, "tiktok") {
			t.Errorf("%s: üres TikTok linknél is van ikon", tpl.ID)
		}
		if tpl.HasPoll && (strings.Contains(html, "Egy kérdés") || strings.Contains(html, "1 kérdés")) {
			t.Errorf("%s: üres kérdésnél is megjelent a kérdés-blokk", tpl.ID)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{"{{#if a}}x", "{{/if}}", "{{#grid a}}{{/if}}", "{{else}}", "{{#if a}}{{else}}{{else}}{{/if}}"} {
		if _, _, err := ParseTemplate(src); err == nil {
			t.Errorf("%q: hibát vártam", src)
		}
	}
	nodes, _, err := ParseTemplate(`a{{#if x}}[{{#if y}}Y{{else}}n{{/if}}]{{else}}-{{/if}}b {{v}}`)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &Template{nodes: nodes}
	got, _ := tpl.Render(&RenderData{Values: map[string]string{"x": "1", "y": "", "v": `<&">` + "\nx"}})
	if got != "a[n]b &lt;&amp;&quot;&gt;<br>x" {
		t.Errorf("kapott: %q", got)
	}
}
