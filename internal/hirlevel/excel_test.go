package hirlevel

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func loadDemo(t *testing.T) *ExcelData {
	t.Helper()
	data, err := os.ReadFile("../../demo/Energofish_partner_hirlevel_minta.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	ex, err := ReadExcel(data, "minta.xlsx", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return ex
}

func TestReadDemoExcel(t *testing.T) {
	ex := loadDemo(t)
	if ex.PartnerSheet != "Partnerek" || ex.ProductSheet != "Termékek" {
		t.Fatalf("munkalapok: %q %q", ex.PartnerSheet, ex.ProductSheet)
	}
	if len(ex.Partners) != 21 {
		t.Errorf("%d partner", len(ex.Partners))
	}
	if len(ex.Products) != 6 {
		t.Errorf("%d termék", len(ex.Products))
	}
	for _, is := range ex.Issues {
		t.Errorf("váratlan megállapítás: %+v", is)
	}
	p := ex.Partners[0]
	if p.Email != "kapitany.horgaszbolt@example.com" || p.Name != "Kovács Péter" || p.RepName != "Nagy Attila" ||
		p.RepPhoto != "{assets}/portre-helyettesito.png" || p.RepPhone != "+36 30 123 4567" ||
		p.RepEmail != "nagy.attila@energofish.hu" || p.RepRegion != "Pest és Nógrád megye" || p.Company == "" || p.Row != 2 {
		t.Errorf("első partner: %+v", p)
	}
	pr := ex.Products[0]
	if pr.Code != "T82753-204" || pr.Price != "3\u00a0090\u00a0Ft" || pr.Deal != "−15% okt. 31-ig" || !pr.On ||
		!strings.HasPrefix(pr.Image, "https://images.energofish.hu/") || !strings.HasPrefix(pr.URL, "https://b2b.") {
		t.Errorf("első termék: %+v", pr)
	}
	for _, is := range ValidatePartners(ex.Partners) {
		if is.Level != LevelInfo {
			t.Errorf("partner megállapítás: %+v", is)
		}
	}
}

// Rendhagyó fejlécek, hivatkozással megadott link, számként tárolt telefon és ár.
func TestReadExcelAliasesAndLinks(t *testing.T) {
	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "Útmutató")
	_ = f.SetCellValue("Útmutató", "A1", "Ez csak leírás, e-mail cím nélkül")
	_, _ = f.NewSheet("Címlista")
	rows := [][]any{
		{"Megjegyzés a táblázathoz"},
		{},
		{"Név", "E-mail cím", "Területi képviselője", "Területi képviselő képének linke", "Képviselő mobil", "Partnerkód", "note.body", "Képviselő e-mail"},
		{"Kiss Anna", "anna@pelda.hu", "Dr. Kovács Béla", "Kép", 36301112222, "P-001", "Egyedi szöveg Annának.", "kovacs.bela@energofish.hu"},
		{"", "", "", "", "", "", "", ""},
		{"", "rossz-cim", "Nagy Attila", "ftp://x", "06 30 999 8888", "P-002", "", ""},
	}
	for i, r := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow("Címlista", cell, &r)
	}
	_ = f.SetCellHyperLink("Címlista", "D4", "https://kepek.pelda.hu/kovacs.jpg", "External")
	_, _ = f.NewSheet("Ajánlat")
	prod := [][]any{
		{"Termékkód", "Megnevezés", "Fotó", "Link a termékhez", "Leírás", "Ár (Ft)", "Kedvezmény", "Aktív"},
		{"A-1", "Első termék", "", "", "Rövid", 12990, "", "igen"},
		{"A-2", "Második", "https://img.pelda.hu/a2.jpg", "https://shop.pelda.hu/a2", "", "990 Ft", "−10%", "nem"},
	}
	for i, r := range prod {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow("Ajánlat", cell, &r)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	ex, err := ReadExcel(buf.Bytes(), "teszt.xlsx", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if ex.PartnerSheet != "Címlista" || ex.ProductSheet != "Ajánlat" {
		t.Fatalf("munkalapok: %q / %q", ex.PartnerSheet, ex.ProductSheet)
	}
	if len(ex.Partners) != 2 {
		t.Fatalf("%d partner", len(ex.Partners))
	}
	a := ex.Partners[0]
	if a.Name != "Kiss Anna" || a.Email != "anna@pelda.hu" || a.RepName != "Dr. Kovács Béla" ||
		a.RepPhoto != "https://kepek.pelda.hu/kovacs.jpg" || a.RepPhone != "36301112222" || a.Row != 4 ||
		a.Extra["partnerkod"] != "P-001" || a.Overrides["note.body"] != "Egyedi szöveg Annának." || a.RepEmail != "kovacs.bela@energofish.hu" {
		t.Errorf("1. partner: %+v", a)
	}
	if Initials(a.RepName) != "KB" {
		t.Errorf("monogram: %q", Initials(a.RepName))
	}
	if PhoneDisplay(a.RepPhone) != "+36 30 111 2222" || PhoneURL(a.RepPhone) != "tel:+36301112222" {
		t.Errorf("telefon: %q %q", PhoneDisplay(a.RepPhone), PhoneURL(a.RepPhone))
	}
	b := ex.Partners[1]
	if PhoneURL(b.RepPhone) != "tel:+36309998888" || PhoneDisplay(b.RepPhone) != "06 30 999 8888" {
		t.Errorf("2. telefon: %q", PhoneURL(b.RepPhone))
	}
	issues := ValidatePartners(ex.Partners)
	var errs, photo int
	for _, is := range issues {
		if is.Level == LevelError && is.Index == 1 {
			errs++
		}
		if is.Key == "repPhoto" {
			photo++
		}
	}
	if errs != 1 || photo != 1 || !PartnerBlocked(&ex.Partners[1]) {
		t.Errorf("validálás: %+v", issues)
	}
	if len(ex.Products) != 2 {
		t.Fatalf("%d termék", len(ex.Products))
	}
	if p := ex.Products[0]; p.Code != "A-1" || p.Price != "12\u00a0990\u00a0Ft" || !p.On || p.Desc != "Rövid" {
		t.Errorf("1. termék: %+v", p)
	}
	if p := ex.Products[1]; p.On || p.Price != "990\u00a0Ft" || p.Deal != "−10%" || p.URL != "https://shop.pelda.hu/a2" {
		t.Errorf("2. termék: %+v", p)
	}
}

func TestMultipleEmails(t *testing.T) {
	p := Partner{Email: "bolt@pelda.hu; tulaj@pelda.hu", Name: "Bolt"}
	if PartnerBlocked(&p) || len(SplitEmails(p.Email)) != 2 {
		t.Fatal("két cím egy cellában legyen érvényes")
	}
	if FileBaseName("", &p, 1, 5) != "001_bolt@pelda.hu" {
		t.Errorf("fájlnév: %q", FileBaseName("", &p, 1, 5))
	}
	eml := string(BuildEML(nil, &p, "Tárgy", "<p>x</p>", "x"))
	if !strings.Contains(eml, "To: \"Bolt\" <bolt@pelda.hu>, <tulaj@pelda.hu>\r\n") {
		t.Errorf("To: %q", eml[:200])
	}
	if !PartnerBlocked(&Partner{Email: "bolt@pelda.hu; rossz"}) {
		t.Error("hibás második cím")
	}
}

func TestReadExcelRejectsXLS(t *testing.T) {
	if _, err := ReadExcel([]byte("x"), "regi.xls", time.Now()); err == nil || !strings.Contains(err.Error(), ".xlsx") {
		t.Errorf("hiba: %v", err)
	}
}

func testContent(t *testing.T) (Content, []Product) {
	t.Helper()
	data, err := os.ReadFile("testdata/tartalom-minta.json")
	if err != nil {
		t.Fatal(err)
	}
	c, products, err := DefaultContent(data)
	if err != nil {
		t.Fatal(err)
	}
	return c.Normalize(c), products
}

func TestBuildPerPartner(t *testing.T) {
	c, products := testContent(t)
	tpls, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	ex := loadDemo(t)
	c["offer.linkParams"] = "utm_source=hirlevel&utm_campaign={ceg}"
	for _, tpl := range tpls {
		for i := range ex.Partners {
			p := &ex.Partners[i]
			d := Build(c, ex.Products, p, tpl, "../assets")
			html, missing := tpl.Render(d)
			if len(missing) > 0 {
				t.Fatalf("%s: hiányzó kulcsok %v", tpl.ID, missing)
			}
			if p.Name != "" && !strings.Contains(html, "Kedves "+EscapeHTML(p.Name)+"!") {
				t.Errorf("%s/%s: nincs megszólítás", tpl.ID, p.Email)
			}
			if p.Name == "" && !strings.Contains(html, "Kedves Partnerünk!") {
				t.Errorf("%s: nincs tartalék megszólítás", tpl.ID)
			}
			if !strings.Contains(html, ">"+EscapeHTML(p.RepName)+"<") {
				t.Errorf("%s: nincs képviselő név", tpl.ID)
			}
			if p.RepPhoto != "" && !strings.Contains(html, `src="../assets/portre-helyettesito.png" alt="`+EscapeHTML(p.RepName)) {
				t.Errorf("%s/%s: nincs képviselő fotó", tpl.ID, p.Email)
			}
			if p.RepPhoto == "" && !strings.Contains(html, ">"+Initials(p.RepName)+"</td>") {
				t.Errorf("%s/%s: nincs monogram", tpl.ID, p.Email)
			}
			if !strings.Contains(html, "6 termék") {
				t.Errorf("%s: a {termekszam} nem cserélődött", tpl.ID)
			}
			if tpl.HasPoll && !strings.Contains(html, "partner="+strings.ReplaceAll(p.Email, "@", "%40")) {
				t.Errorf("%s: a kérdés linkjében nincs partner", tpl.ID)
			}
			if !strings.Contains(html, "utm_campaign=") || strings.Contains(html, "{ceg}") || strings.Contains(html, "{") && strings.Contains(html, "{nev}") {
				t.Errorf("%s: a változók nem cserélődtek", tpl.ID)
			}
		}
	}
	// v4 preheader kiegészítés
	d := Build(c, products, &SamplePartner, FindTemplate(tpls, "v4"), "")
	if !strings.HasSuffix(d.Values["meta.preheader"], "Plusz egy kérdés.") {
		t.Errorf("v4 preheader: %q", d.Values["meta.preheader"])
	}
	d = Build(c, products, &SamplePartner, FindTemplate(tpls, "v1"), "")
	if strings.HasSuffix(d.Values["meta.preheader"], "Plusz egy kérdés.") {
		t.Errorf("v1 preheader: %q", d.Values["meta.preheader"])
	}
	// felülírás
	p := SamplePartner
	p.Overrides = map[string]string{"note.body": "Egyedi {nev}"}
	d = Build(c, products, &p, FindTemplate(tpls, "v1"), "")
	if d.Values["note.body"] != "Egyedi Kovács Péter" {
		t.Errorf("felülírás: %q", d.Values["note.body"])
	}
}

func TestValidateDefaults(t *testing.T) {
	c, products := testContent(t)
	tpls, _ := LoadTemplates(os.DirFS("../.."), "sablonok")
	for _, tpl := range tpls {
		for _, is := range ValidateContent(c, products, tpl) {
			if is.Level == LevelError {
				t.Errorf("%s: %s", tpl.ID, is.Message)
			}
		}
	}
	for _, is := range ValidateProducts(c, products) {
		if is.Level == LevelError {
			t.Errorf("termék: %s", is.Message)
		}
	}
	c["cover.image"] = "kep.jpg"
	c["poll.answers.2.url"] = ""
	c["meta.subject"] = ""
	errs := 0
	for _, is := range ValidateContent(c, products, FindTemplate(tpls, "v4")) {
		if is.Level == LevelError {
			errs++
		}
	}
	if errs != 3 {
		t.Errorf("3 hibát vártam, %d lett", errs)
	}
}

func TestGenerate(t *testing.T) {
	c, _ := testContent(t)
	tpls, _ := LoadTemplates(os.DirFS("../.."), "sablonok")
	ex := loadDemo(t)
	partners := append([]Partner{}, ex.Partners...)
	partners = append(partners, Partner{Row: 99, Email: "hibas", Name: "Hibás"})
	dir := t.TempDir()
	res, err := Generate(os.DirFS("../../assets"), FindTemplate(tpls, "v4"), c, ex.Products, partners,
		GenerateOptions{OutputDir: dir, EML: true, From: "Energofish <hirlevel@energofish.hu>"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Generated != 21 || len(res.Skipped) != 1 {
		t.Fatalf("eredmény: %+v", res)
	}
	for _, name := range []string{"attekinto.html", "kuldesi-lista.csv", "tartalom.json", "assets/energofish-mark.png",
		"html/001_kapitany.horgaszbolt@example.com.html", "eml/021_kiskunhalas.csali@example.com.eml"} {
		if _, err := os.Stat(filepath.Join(res.Folder, name)); err != nil {
			t.Errorf("hiányzik: %s", name)
		}
	}
	eml, _ := os.ReadFile(filepath.Join(res.Folder, "eml/001_kapitany.horgaszbolt@example.com.eml"))
	s := string(eml)
	for _, want := range []string{"X-Unsent: 1\r\n", "To: =?utf-8?q?Kov=C3=A1cs_P=C3=A9ter?= <kapitany.horgaszbolt@example.com>",
		"From: \"Energofish\" <hirlevel@energofish.hu>", "Subject: =?UTF-8?q?", "multipart/alternative", "Content-Type: text/html"} {
		if !strings.Contains(s, want) {
			t.Errorf("EML-ből hiányzik: %q\n%s", want, s[:400])
		}
	}
	csvData, _ := os.ReadFile(res.CSV)
	if !strings.HasPrefix(string(csvData), "\ufeffSorszám;E-mail;") || strings.Count(string(csvData), "\r\n") != 22 {
		t.Errorf("CSV: %q", string(csvData)[:200])
	}
	// második futás új mappába kerül
	res2, err := Generate(os.DirFS("../../assets"), FindTemplate(tpls, "v1"), c, ex.Products, ex.Partners[:2],
		GenerateOptions{OutputDir: dir, FilePattern: "{ceg} - {nev}"})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Folder == res.Folder || res2.Generated != 2 {
		t.Errorf("második futás: %+v", res2)
	}
	if _, err := os.Stat(filepath.Join(res2.Folder, "html", "Kapitány Horgászbolt, Vác - Kovács Péter.html")); err != nil {
		t.Errorf("fájlnév minta: %v", err)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	c, products := testContent(t)
	j, err := ExportJSON(c, products)
	if err != nil {
		t.Fatal(err)
	}
	c2, p2, err := ImportJSON(j)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range c {
		if c2[k] != v {
			t.Errorf("%s: %q != %q", k, c2[k], v)
		}
	}
	if len(p2) != len(products) || !reflect.DeepEqual(p2[5], products[5]) {
		t.Errorf("termékek: %+v", p2)
	}
}

func TestHelpers(t *testing.T) {
	cases := map[string]string{"Nagy Attila": "NA", "dr. Kiss Péter": "KP", "Éva": "ÉV", "ifj. Szabó Ödön": "SÖ", "": ""}
	for in, want := range cases {
		if got := Initials(in); got != want {
			t.Errorf("Initials(%q) = %q", in, got)
		}
	}
	if got := FormatPrice("3090"); got != "3\u00a0090\u00a0Ft" {
		t.Errorf("ár: %q", got)
	}
	if got := FormatPrice("1234567.6"); got != "1\u00a0234\u00a0568\u00a0Ft" {
		t.Errorf("ár: %q", got)
	}
	if got := FormatPrice("kérjen ajánlatot"); got != "kérjen ajánlatot" {
		t.Errorf("ár: %q", got)
	}
	if got := AppendParams("https://a.hu/x?y=1#top", "utm_source=h"); got != "https://a.hu/x?y=1&utm_source=h#top" {
		t.Errorf("params: %q", got)
	}
	if got := AppendParams("mailto:a@b.hu", "x=1"); got != "mailto:a@b.hu" {
		t.Errorf("params: %q", got)
	}
	got, unknown, empty := Expand("https://x.hu/{Cikkszám}?e={email}&n={nev}&q={xyz}", map[string]string{"cikkszam": "A 1", "email": "a+b@c.hu", "nev": ""}, true)
	if got != "https://x.hu/A%201?e=a%2Bb%40c.hu&n=&q={xyz}" || len(unknown) != 1 || len(empty) != 1 {
		t.Errorf("Expand: %q %v %v", got, unknown, empty)
	}
	if SafeFileName(`a/b:c*?.`) != "a_b_c_" {
		t.Errorf("fájlnév: %q", SafeFileName(`a/b:c*?.`))
	}
}
