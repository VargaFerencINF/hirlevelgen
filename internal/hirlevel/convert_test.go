package hirlevel

import (
	"os"
	"strings"
	"testing"
)

func TestConvertBuiltins(t *testing.T) {
	for _, info := range TemplateInfos {
		raw, err := os.ReadFile("../../sablonok/" + info.ID + ".html")
		if err != nil {
			t.Fatal(err)
		}
		conv, ci := Convert(string(raw))
		if !ci.Grid || ci.GridCols != 3 || !ci.Photo || ci.FixedSlots != 0 || len(ci.Notes) > 0 {
			t.Errorf("%s: %+v", info.ID, ci)
		}
		for _, want := range []string{"{{#grid offer.items:3}}", "{{#if rep.photo}}", "{{#if rep.phone.url}}", "{{#if rep.email.url}}",
			"{{#if utility.browserLink.url}}", "{{#if offer.more.url}}", "{{#if social.tiktok.url}}"} {
			if !strings.Contains(conv, want) {
				t.Errorf("%s: hiányzik %s", info.ID, want)
			}
		}
		if strings.Contains(string(raw), "{{poll.question}}") && strings.Count(conv, "{{#if poll.") != 5 {
			t.Errorf("%s: kérdés-blokk: %d feltétel", info.ID, strings.Count(conv, "{{#if poll."))
		}
		again, _ := Convert(conv)
		if again != conv {
			t.Errorf("%s: az átalakítás nem idempotens", info.ID)
		}
	}
}

// Kézzel írt, egyszerű sablon: nem szabványos rács → fix termékhelyek.
func TestCustomTemplateFixedSlots(t *testing.T) {
	src := `<html><head><title>{{meta.subject}}</title></head><body>
<p>{{note.greeting}}</p><img src="{{assets.base}}/sajat-fejlec.png">
<table><tr><td>{{offer.items.1.name}} {{offer.items.1.price}}</td><td>{{offer.items.2.name}}</td></tr></table>
<p>{{rep.name}} {{rep.initials}}</p></body></html>`
	tpl, err := NewTemplate("sajat", src)
	if err != nil {
		t.Fatal(err)
	}
	if tpl.FixedSlots != 2 || tpl.PhotoOK || len(tpl.Report) != 2 || tpl.HasPoll {
		t.Errorf("%+v", tpl)
	}
	if len(tpl.AssetRefs) != 1 || tpl.AssetRefs[0] != "sajat-fejlec.png" {
		t.Errorf("képek: %v", tpl.AssetRefs)
	}
	html, missing := tpl.Render(&RenderData{Values: map[string]string{"meta.subject": "T", "note.greeting": "Szia", "assets.base": "a", "rep.name": "N", "rep.initials": "NN"},
		Items: []map[string]string{{"name": "Egy", "price": "1 Ft"}}})
	if len(missing) > 0 || !strings.Contains(html, "Egy 1 Ft") {
		t.Errorf("render: %v %s", missing, html)
	}
}

func TestCustomTemplateRejectsUnknownKeys(t *testing.T) {
	if _, err := NewTemplate("x", `<p>{{partner.kedvezmeny}} {{meta.subject}}</p>`); err == nil || !strings.Contains(err.Error(), "partner.kedvezmeny") {
		t.Errorf("hiba: %v", err)
	}
	if _, err := NewTemplate("x", `<p>nincs helyőrző</p>`); err == nil {
		t.Error("helyőrző nélküli fájl")
	}
	if _, err := NewTemplate("x", `{{#if a}}`); err == nil {
		t.Error("hibás blokk")
	}
}

// Két oszlopos rács: a sortörés a sablonból jön.
func TestConvertTwoColumnGrid(t *testing.T) {
	tile := func(n string) string {
		return `<div class="tile"><a href="{{offer.items.` + n + `.url}}">{{offer.items.` + n + `.name}}</a></div>`
	}
	src := `<div>` + tile("1") + `<!--[if mso]></td><td width="250" valign="top"><![endif]-->` + "\n" + tile("2") +
		`<!--[if mso]></td></tr><tr><td width="250" valign="top"><![endif]-->` + "\n" + tile("3") +
		`<!--[if mso]></td><td width="250" valign="top"><![endif]-->` + "\n" + tile("4") + `</div>`
	tpl, err := NewTemplate("ket", src)
	if err != nil {
		t.Fatal(err)
	}
	conv, ci := Convert(src)
	if !ci.Grid || ci.GridCols != 2 || !strings.Contains(conv, "{{#grid offer.items:2}}") || tpl.FixedSlots != 0 {
		t.Fatalf("%+v\n%s", ci, conv)
	}
	items := []map[string]string{}
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		items = append(items, map[string]string{"name": n, "url": "u"})
	}
	html, _ := tpl.Render(&RenderData{Values: map[string]string{}, Items: items})
	if strings.Count(html, "</tr><tr>") != 2 || strings.Count(html, `class="tile"`) != 5 {
		t.Errorf("%s", html)
	}
}
