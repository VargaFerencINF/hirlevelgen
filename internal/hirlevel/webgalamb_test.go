package hirlevel

import (
	"os"
	"sort"
	"strings"
	"testing"
)

func rec(email, nazon string) string {
	return `{"Email_cim":"` + email + `","Nazon":"` + nazon + `","Nev":"N","Statusz":"Aktív","Fix":"LANG_ADMIN_NO","Bizomanyos":"LANG_ADMIN_NO"}`
}

func TestDecodeWebgalambFormats(t *testing.T) {
	cases := []struct {
		name, in, format, wrapper string
		nazon                     []string
		gaps                      int
	}{
		{"tömb", "[" + rec("a@x.hu", "1") + "," + rec("b@x.hu", "2") + "]", WGFormatList, "", []string{"1", "2"}, 0},
		// lyuk (2 hiányzik) és 10 feletti kulcsok: numerikus sorrend, nem szöveges („10” nem a „2” előtt)
		{"számozott objektum", `{"10":` + rec("j@x.hu", "10") + `,"0":` + rec("a@x.hu", "0") + `,"3":` + rec("d@x.hu", "3") + `,"1":` + rec("b@x.hu", "1") +
			`,"11":` + rec("k@x.hu", "11") + `,"9":` + rec("i@x.hu", "9") + `}`, WGFormatNumbered, "", []string{"0", "1", "3", "9", "10", "11"}, 6},
		{"BOM és szóközök", "\ufeff \n\t[" + rec("a@x.hu", "1") + "]\n ", WGFormatList, "", []string{"1"}, 0},
		{"burkolt lista", `{"status":"ok","count":2,"data":[` + rec("a@x.hu", "1") + "," + rec("b@x.hu", "2") + `]}`, WGFormatList, "data", []string{"1", "2"}, 0},
		{"burkolt számozott objektum", `{"meta":{"x":1},"rows":{"0":` + rec("a@x.hu", "1") + `,"2":` + rec("b@x.hu", "2") + `}}`, WGFormatNumbered, "rows", []string{"1", "2"}, 1},
		{"kétszintű burkolás", `{"result":{"items":[` + rec("a@x.hu", "1") + `]}}`, WGFormatList, "result.items", []string{"1"}, 0},
		{"üres lista", "[]", WGFormatList, "", nil, 0},
		{"üres objektum", "{}", WGFormatNumbered, "", nil, 0},
	}
	for _, c := range cases {
		d, err := DecodeWebgalamb([]byte(c.in))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		var got []string
		for _, r := range d.Records {
			got = append(got, jsonString(r["Nazon"]))
		}
		if d.Format != c.format || d.Wrapper != c.wrapper || strings.Join(got, ",") != strings.Join(c.nazon, ",") || d.Gaps != c.gaps {
			t.Errorf("%s: %s %q %v lyuk=%d", c.name, d.Format, d.Wrapper, got, d.Gaps)
		}
	}
	// a rekord helye a figyelmeztetésekben: a számozott objektum kulcsa
	d, _ := DecodeWebgalamb([]byte(`{"0":` + rec("a@x.hu", "1") + `,"2":` + rec("b@x.hu", "2") + `}`))
	if d.Labels[1] != `2. (kulcs: "2")` || !strings.Contains(d.FormatLabel(), "1 hiányzó sorszámmal") {
		t.Errorf("címkék: %v, %s", d.Labels, d.FormatLabel())
	}
}

func TestDecodeWebgalambErrors(t *testing.T) {
	cases := map[string]string{
		"":                          "üres",
		"   \n ":                    "üres",
		"<html><body>Hiba</body>":   "HTML oldal",
		`"szöveg"`:                  "gyökéreleme ismeretlen szerkezetű (várt: lista vagy számozott objektum)",
		`42`:                        "gyökéreleme ismeretlen szerkezetű",
		`{"a":1,"b":"x"}`:           "gyökéreleme ismeretlen szerkezetű",
		`{"error":"invalid token"}`: "a szerver hibát adott: invalid token",
		"[\n" + rec("a@x.hu", "1") + ",\n{\"Email_cim\": }\n]": "nem érvényes JSON (szintaktikai hiba) (3. sor, 15. oszlop)",
		`[` + rec("a@x.hu", "1"):                               "csonka",
		`[` + rec("a@x.hu", "1") + `] xyz`:                     "további, értelmezhetetlen adat",
	}
	for in, want := range cases {
		_, err := DecodeWebgalamb([]byte(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v (várt: %s)", in, err, want)
		}
		// nyers Go-hibaüzenet nem jut a felhasználóhoz
		if err != nil && (strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") || strings.Contains(err.Error(), "unexpected EOF")) {
			t.Errorf("%q: nyers hibaüzenet: %v", in, err)
		}
	}
	// a ParseB2BExport ugyanezeket a hibákat adja; az üres export biztonsági okból hiba
	for in, want := range map[string]string{"[]": "üres (0 partner)", "{}": "üres (0 partner)", `{"x":true}`: "ismeretlen szerkezetű"} {
		if _, _, err := ParseB2BExport([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ParseB2BExport(%s): %v", in, err)
		}
	}
}

func TestParseB2BExportTolerantFields(t *testing.T) {
	in := `{"0":{"Email_cim":"A@Example.com","Nazon":"04838","Nev":"Első","Teruleti_kepviselo_nev":null,"Uj_mezo":"x","Fix":"LANG_ADMIN_NO"},` +
		`"1":{"Email_cim":"b@example.com","Nazon":4838,"Nev":"Második","Besor":"ARANY","Megye":"Pest"},` +
		`"2":{"Nev":"Cím nélkül","Nazon":"00001"},` +
		`"3":{"Email_cim":"  ","Nazon":"00002"},` +
		`"5":{"Email_cim":"c@example.com","Nazon":"10003","Statusz":"Aktív","Bizomanyos":"LANG_ADMIN_YES"}}`
	list, rep, err := ParseB2BExport([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || rep.Records != 5 || rep.Skipped != 2 || !strings.HasPrefix(rep.Format, WGFormatNumbered) {
		t.Fatalf("%d partner, %+v", len(list), rep)
	}
	a, b, c := findB2B(list, "a@example.com"), findB2B(list, "b@example.com"), findB2B(list, "c@example.com")
	// a vezető nullák megmaradnak (szövegként jött); a szám szövegként tárolódik
	if a.Nazon != "04838" || b.Nazon != "4838" || a.RepName != "" || a.Extra["Uj_mezo"] != "x" || a.NoMail == "" {
		t.Errorf("mezők: %+v / %+v", a, b)
	}
	// hiányzó Statusz/Fix/Bizomanyos nem hiba; a megadott érték számít
	if b.NoMail != "nincs érvényes leiratkozó link" || b.Fix || b.Commission || !c.Commission {
		t.Errorf("hiányzó mezők: %+v / %+v", b, c)
	}
	warn := strings.Join(rep.Warnings, "\n")
	if !strings.Contains(warn, `3. (kulcs: "2") rekord (Nazon: 00001): hiányzó vagy üres e-mail cím (Email_cim) – kihagyva`) ||
		!strings.Contains(warn, `4. (kulcs: "3") rekord`) {
		t.Errorf("figyelmeztetések:\n%s", warn)
	}
}

func TestParseB2BExportDuplicates(t *testing.T) {
	in := `[{"Email_cim":"g@energofish.hu","Nazon":"TK1","Feliratkozas_datum":"2021-01-01 10:00:00","Fix":"LANG_ADMIN_YES"},` +
		`{"Email_cim":"x@example.com","Nazon":"10001"},` +
		`{"Email_cim":"G@energofish.hu","Nazon":"TK2","Feliratkozas_datum":"2024-01-01 10:00:00","Fix":"LANG_ADMIN_NO"}]`
	list, rep, err := ParseB2BExport([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	// címenként egy partner (egy levél); a figyelmeztetésből látszik, melyik rekord maradt ki
	if len(list) != 2 || rep.Merged != 1 || len(rep.Duplicates) != 1 {
		t.Fatalf("%d partner, %+v", len(list), rep)
	}
	d := rep.Duplicates[0]
	if !strings.Contains(d, "g…@energofish.hu: kétszer szerepel") || !strings.Contains(d, "a(z) 3. rekord (Nazon: TK2) adatai maradtak") ||
		!strings.Contains(d, "a(z) 1. rekord (Nazon: TK1) kimaradt") {
		t.Errorf("duplikátum: %s", d)
	}
	if p := findB2B(list, "g@energofish.hu"); p.Nazon != "TK2" || !p.Fix || p.Subscribed != "2021-01-01 10:00:00" {
		t.Errorf("összevonás: %+v", p)
	}
}

// A két valós export (anonimizálva): a régi tömbös és az új, számozott objektumos – ugyanazok a
// mezők, a rekordszám a tartalmi különbség miatt eltér (567 / 568, az újban a „228” kulcs hiányzik).
func TestWebgalambRealSamples(t *testing.T) {
	type res struct {
		list []B2BPartner
		rep  ExportReport
		keys string
	}
	load := func(name string) res {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		d, err := DecodeWebgalamb(data)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, r := range d.Records {
			for k := range r {
				seen[k] = true
			}
		}
		var keys []string
		for k := range seen {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		list, rep, err := ParseB2BExport(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return res{list, rep, strings.Join(keys, ",")}
	}
	old, cur := load("webgalamb-lista-anon.json"), load("webgalamb-objektum-anon.json")
	if old.rep.Records != 567 || old.rep.Format != WGFormatList {
		t.Errorf("régi: %d rekord, %s", old.rep.Records, old.rep.Format)
	}
	if cur.rep.Records != 568 || cur.rep.Format != WGFormatNumbered+" (1 hiányzó sorszámmal)" {
		t.Errorf("új: %d rekord, %s", cur.rep.Records, cur.rep.Format)
	}
	if old.keys != cur.keys || !strings.Contains(cur.keys, "Email_cim") || !strings.Contains(cur.keys, "Token") {
		t.Errorf("mezők eltérnek:\n%s\n%s", old.keys, cur.keys)
	}
	for _, r := range []res{old, cur} {
		if r.rep.Merged != 2 || len(r.rep.Duplicates) != 2 || r.rep.Skipped != 0 || len(r.list) != r.rep.Records-2 {
			t.Errorf("%s: %d partner, %+v", r.rep.Format, len(r.list), r.rep)
		}
		for _, p := range r.list {
			if p.Email == "" || p.Nazon == "" || p.Unsubscribe == "" {
				t.Fatalf("hiányos partner: %+v", p)
			}
		}
	}
	// az új exportban egy partnernél null a képviselő: ez nem hiba
	if cur.rep.NoRep == 0 {
		t.Error("a null képviselőjű partner nincs megszámolva")
	}
}
