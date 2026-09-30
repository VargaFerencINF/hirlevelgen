package hirlevel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestLibrary(t *testing.T, dir string) *Library {
	t.Helper()
	b, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	l, errs := OpenLibrary(dir, b, os.DirFS("../../assets"))
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	return l
}

func TestLibraryImportDesignerZip(t *testing.T) {
	dir := t.TempDir()
	l := newTestLibrary(t, dir)
	data, _ := os.ReadFile("testdata/tervezo-csomag-1.1.zip")
	res, err := l.ImportFile("Energofish_Partner_Newsletter_Design.zip", data, nil)
	if err != nil {
		t.Fatal(err)
	}
	// a 3 sablon (az elonezet/ kitöltött fájljai kimaradnak), mind a beépítetteket frissíti
	if len(res) != 3 {
		t.Fatalf("%+v", res)
	}
	for _, r := range res {
		if !r.OK || !r.Override || len(r.Notes) > 0 {
			t.Errorf("%+v", r)
		}
	}
	all := l.All()
	if len(all) != 3 || !all[0].Custom || !all[0].Overrides || all[0].Short != "v1" || all[0].Name != "Sötét lemez" {
		t.Errorf("%+v", all[0])
	}
	// a képek azonosak a beépítettekkel, ezért nem másolódnak
	if all[0].AssetsDir != "" {
		t.Errorf("assets: %s", all[0].AssetsDir)
	}
	// törlés után az eredeti beépített tér vissza
	if err := l.Delete("v1-sotet-lemez"); err != nil {
		t.Fatal(err)
	}
	if a := l.All(); !a[0].Builtin || a[0].Custom {
		t.Errorf("visszaállítás: %+v", a[0])
	}
	if err := l.Delete("v2-nincs"); err == nil {
		t.Error("nem létező törlés")
	}
}

func TestLibraryCustomTemplateLifecycle(t *testing.T) {
	dir := t.TempDir()
	l := newTestLibrary(t, dir)
	raw, _ := os.ReadFile("../../sablonok/v2-waterside.html")
	src := strings.Replace(string(raw), "{{assets.base}}/wave-band-white.png", "{{assets.base}}/tavaszi-hullam.png", 1)
	img := []byte("\x89PNG kép")
	find := func(name string) ([]byte, bool) {
		if name == "tavaszi-hullam.png" {
			return img, true
		}
		return nil, false
	}
	res, err := l.ImportFile(`C:\Tervek\v5-tavaszi-akció.html`, []byte(src), find)
	if err != nil || len(res) != 1 || !res[0].OK {
		t.Fatalf("%v %+v", err, res)
	}
	r := res[0]
	if r.ID != "v5-tavaszi-akcio" || r.Name != "Tavaszi akció" || r.Override || len(r.Notes) > 0 {
		t.Errorf("%+v", r)
	}
	all := l.All()
	c := all[len(all)-1]
	if len(all) != 4 || c.Short != "v5" || !c.Custom || c.AssetsDir == "" {
		t.Fatalf("%+v", c)
	}
	if b, ok := l.AssetFile("tavaszi-hullam.png"); !ok || string(b) != string(img) {
		t.Error("saját kép")
	}
	if _, ok := l.AssetFile("energofish-mark.png"); !ok {
		t.Error("beépített kép")
	}
	if err := l.Update(c.ID, "Tavaszi akció 2027", "új leírás"); err != nil {
		t.Fatal(err)
	}
	// újranyitás lemezről
	l2 := newTestLibrary(t, dir)
	all = l2.All()
	if len(all) != 4 || all[3].Name != "Tavaszi akció 2027" || all[3].Desc != "új leírás" || all[3].AssetsDir == "" {
		t.Fatalf("%+v", all[3])
	}
	// ugyanaz a fájl újra: frissítés, a név marad
	res, _ = l2.ImportFile("v5-tavaszi-akció.html", []byte(src), find)
	if !res[0].Replaced || l2.All()[3].Name != "Tavaszi akció 2027" {
		t.Errorf("%+v", res[0])
	}
	// hiányzó kép jelzése
	res, _ = l2.ImportFile("masik.html", []byte(src), nil)
	if !res[0].OK || len(res[0].Notes) != 1 || !strings.Contains(res[0].Notes[0], "tavaszi-hullam.png") || l2.All()[4].Short != "S1" {
		t.Errorf("%+v %+v", res[0], l2.All()[4])
	}
	// generálás a saját sablonnal: a saját kép is a kimenetbe kerül
	c5 := FindTemplate(l2.All(), "v5-tavaszi-akcio")
	content, products := testContent(t)
	gen, err := Generate(os.DirFS("../../assets"), c5, content, products, []Partner{SamplePartner}, GenerateOptions{OutputDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(gen.Folder, "assets", "tavaszi-hullam.png")); err != nil {
		t.Error("a saját kép nem került a kimenetbe")
	}
	// hibás fájlok
	if res, _ := l2.ImportFile("rossz.html", []byte("<p>{{ismeretlen.mezo}}</p>"), nil); res[0].OK || res[0].Error == "" {
		t.Errorf("%+v", res[0])
	}
	if _, err := l2.ImportFile("kep.png", []byte("x"), nil); err == nil {
		t.Error("png")
	}
	if err := l2.Delete("v1-sotet-lemez"); err == nil {
		t.Error("beépített törlése")
	}
}

func TestPrettyNameAndSlug(t *testing.T) {
	for in, want := range map[string][2]string{
		"v5-tavaszi-akció.html":   {"Tavaszi akció", "v5"},
		"Karácsonyi_ajánlat.html": {"Karácsonyi ajánlat", ""},
		"sablonok/V12 nyári.html": {"Nyári", "v12"},
		"v7.html":                 {"V7", "v7"},
	} {
		n, s := prettyName(in)
		if n != want[0] || s != want[1] {
			t.Errorf("%s: %q %q", in, n, s)
		}
	}
	if Slug("V5 Tavaszi Akció!") != "v5-tavaszi-akcio" || Slug("???") != "sablon" {
		t.Error(Slug("V5 Tavaszi Akció!"))
	}
}
