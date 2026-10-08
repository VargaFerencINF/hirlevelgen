package hirlevel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A tesztadatok kitaláltak (example.com, hamis tokenek) – valós partneradat nem kerülhet a repóba.

func loadB2BFixture(t *testing.T) ([]B2BPartner, ExportReport) {
	t.Helper()
	data, err := os.ReadFile("testdata/b2b-minta.json")
	if err != nil {
		t.Fatal(err)
	}
	list, rep, err := ParseB2BExport(data)
	if err != nil {
		t.Fatal(err)
	}
	return list, rep
}

func findB2B(list []B2BPartner, email string) *B2BPartner {
	for i := range list {
		if list[i].Email == email {
			return &list[i]
		}
	}
	return nil
}

func TestParseB2BExport(t *testing.T) {
	list, rep := loadB2BFixture(t)
	// 12 rekord: 1 érvénytelen e-mail, 1 hiányzó mező → kihagyva; 1 dupla → összevonva
	if rep.Records != 12 || rep.Skipped != 2 || rep.Merged != 1 || rep.Unique != 9 || len(list) != 9 {
		t.Fatalf("riport: %+v (%d)", rep, len(list))
	}
	if rep.NoToken != 1 || rep.NoRep != 1 || rep.NoMail != 1 || strings.Join(rep.UnknownKeys, ",") != "Nyelv,Uj_mezo" {
		t.Errorf("riport: %+v", rep)
	}
	for _, w := range rep.Warnings { // a figyelmeztetésekben nincs e-mail és token
		if strings.Contains(w, "@") || secretRx.MatchString(w) {
			t.Errorf("személyes adat a figyelmeztetésben: %q", w)
		}
	}
	// S03: kisbetűs, szóköz nélküli e-mail
	nb := findB2B(list, "nagy.betu@example.com")
	if nb == nil || nb.Level != "GYEMANT" || nb.RepMono != "GAB" || nb.Shop != "Horgászbolt" {
		t.Fatalf("normalizálás: %+v", nb)
	}
	// S04: Fix = VAGY, a feliratkozás a legkorábbi
	b := findB2B(list, "belso@example.com")
	if b == nil || !b.Fix || !b.Commission || b.Subscribed != "2022-01-01 00:00:00" {
		t.Errorf("összevonás: %+v", b)
	}
	if p := findB2B(list, "torolt@example.com"); p.Token != "" || p.Prop6 != "" {
		t.Errorf("Torolt token: %+v", p)
	}
	if p := findB2B(list, "bizo@example.com"); p.Shop != "" || !p.Commission {
		t.Errorf("hiányzó Partnerbolt_statusz: %+v", p)
	}
	if p := findB2B(list, "rosszlink@example.com"); p.NoMail == "" {
		t.Error("a leiratkozó link nélküli partner nem kaphat levelet")
	}
	if p := findB2B(list, "ujmezo@example.com"); p.Extra["Uj_mezo"] != "valami" || len(p.Token) != 32 {
		t.Errorf("ismeretlen mező: %+v", p)
	}
	if p := findB2B(list, "bolt1@example.com"); !strings.HasPrefix(p.Unsubscribe, "https://energofish.hu/leiratkozas.html?&c=") || p.RepName != "Szél Zsófia" {
		t.Errorf("mezők: %+v", p)
	}
}

func TestParseB2BExportErrors(t *testing.T) {
	cases := map[string]string{
		"":                                      "JSON",
		"[]":                                    "üres",
		"<html>Hiba</html>":                     "ismeretlen szerkezetű",
		`{"hiba":"token"}`:                      "ismeretlen szerkezetű",
		`[{"Nev":"a"},{"Nev":"b"},{"Nev":"c"}]`: "hiányzik az e-mail cím",
	}
	for in, want := range cases {
		if _, _, err := ParseB2BExport([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", in, err)
		}
	}
	// BOM, szám és null érték, igen/nem
	data := "\ufeff" + `[{"Email_cim":"a@example.com","Nazon":12345,"Nev":"A","Telefonszam":null,"Feliratkozas_datum":"rossz",
	"Leiratkozas_link":"https://energofish.hu/leiratkozas.html?&c=0123456789abcdef0123","Teruleti_kepviselo_nev":"",
	"Teruleti_kepviselo_telefonszam":"","Teruleti_kepviselo_email_cim":"","Teruleti_kepviselo_monogram":"","Fix":"nem",
	"Bizomanyos":"igen","Besor":"basic","Megye":"Vas","Statusz":"Aktív","Tulajdonsag_6":"","Token":"x"}]`
	list, rep, err := ParseB2BExport([]byte(data))
	if err != nil || len(list) != 1 || list[0].Nazon != "12345" || list[0].Subscribed != "" || list[0].Level != "BASIC" || !list[0].Commission || len(rep.Warnings) == 0 {
		t.Errorf("%v %+v %+v", err, list, rep)
	}
}

// a spec minta_szinkron lapjának esetei
func TestB2BSyncRules(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)
	mk := func(email, nazon, besor string) B2BPartner {
		return B2BPartner{Email: email, Nazon: nazon, Name: nazon, Level: besor, County: "Pest", WGStatus: "Aktív",
			Unsubscribe: "https://energofish.hu/leiratkozas.html?&c=0123456789abcdef0123"}
	}
	db := &B2BDB{Group: "B2B_HU"}
	if _, err := db.Apply([]B2BPartner{mk("marad@example.com", "90002", "ARANY"), mk("szintlepes@example.com", "90003", "EZUST"),
		mk("leiratkozott@example.com", "90004", "BASIC"), mk("visszajott@example.com", "90005", "EZUST"),
		mk("nagy.betu@example.com", "90007", "BASIC")}, ExportReport{}, t0, 0.8, false); err != nil {
		t.Fatal(err)
	}
	// a visszajött partner közben inaktív lett
	db2 := db.Clone()
	for _, p := range db2.Partners {
		if p.Email == "visszajott@example.com" {
			p.Active = false
		}
	}
	export := []B2BPartner{mk("uj.bolt@example.com", "90001", "BASIC"), mk("marad@example.com", "90002", "ARANY"),
		mk("szintlepes@example.com", "90003", "GYEMANT"), mk("visszajott@example.com", "90005", "EZUST"),
		mk("nagy.betu@example.com", "90007", "BASIC")}
	l, err := db2.Apply(export, ExportReport{Records: 6}, t1, 0.8, false)
	if err != nil {
		t.Fatal(err)
	}
	if l.New != 1 || l.Updated != 3 || l.Changed != 1 || l.Reactivated != 1 || l.Inactivated != 1 || l.Active != 5 || !l.OK || l.ID != 2 {
		t.Errorf("napló: %+v", l)
	}
	get := func(e string) *B2BPartner {
		for _, p := range db2.Partners {
			if p.Email == e {
				return p
			}
		}
		return nil
	}
	if p := get("leiratkozott@example.com"); p == nil || p.Active || p.Inactivated == nil || !strings.Contains(p.InactiveWhy, "leiratkozott") {
		t.Errorf("leiratkozott (nem törölhető, csak inaktív): %+v", p)
	}
	if p := get("visszajott@example.com"); !p.Active || p.Inactivated != nil {
		t.Errorf("újraaktiválás: %+v", p)
	}
	if p := get("szintlepes@example.com"); p.Level != "GYEMANT" || !p.FirstImport.Equal(t0) || !p.LastImport.Equal(t1) {
		t.Errorf("frissítés: %+v", p)
	}
	if p := get("uj.bolt@example.com"); !p.Active || !p.FirstImport.Equal(t1) {
		t.Errorf("új: %+v", p)
	}
	// S01b: gyanúsan kevés partner → semmi nem változik
	db3 := db2.Clone()
	_, err = db3.Apply(export[:2], ExportReport{}, t1, 0.8, false)
	if !errors.Is(err, ErrSuspiciousExport) || db3.ActiveCount() != 5 {
		t.Errorf("józansági ellenőrzés: %v, %d aktív", err, db3.ActiveCount())
	}
	// kényszerítve lefut
	if l, err := db3.Apply(export[:2], ExportReport{}, t1, 0.8, true); err != nil || l.Inactivated != 3 {
		t.Errorf("kényszerítve: %v %+v", err, l)
	}
	if _, err := db3.Apply(nil, ExportReport{}, t1, 0.8, true); err == nil {
		t.Error("üres export")
	}
}

func TestB2BFilterAndFacets(t *testing.T) {
	list, _ := loadB2BFixture(t)
	db := &B2BDB{Group: "B2B_HU"}
	if _, err := db.Apply(list, ExportReport{}, time.Now(), 0.8, false); err != nil {
		t.Fatal(err)
	}
	emails := func(ps []*B2BPartner) string {
		var s []string
		for _, p := range ps {
			s = append(s, strings.Split(p.Email, "@")[0])
		}
		return strings.Join(s, ",")
	}
	check := func(f PartnerFilter, want string) {
		t.Helper()
		if got := emails(SelectB2B(db, f)); got != want {
			t.Errorf("%+v: %s, várt %s", f, got, want)
		}
	}
	// a leiratkozó link nélküli partner soha nincs a halmazban; név szerint rendezve
	check(PartnerFilter{}, "belso,bizo,bolt1,bolt2,nagy.betu,nincstk,torolt,ujmezo")
	check(PartnerFilter{Reps: []string{"ZSO"}}, "bolt1,bolt2,torolt")
	check(PartnerFilter{Reps: []string{"ZSO", ""}}, "bolt1,bolt2,nincstk,torolt")
	check(PartnerFilter{Reps: []string{"ZSO"}, RepsNot: true}, "belso,bizo,nagy.betu,nincstk,ujmezo")
	check(PartnerFilter{Counties: []string{"Budapest"}, Internal: "exclude"}, "bolt2")
	check(PartnerFilter{Internal: "only"}, "belso")
	check(PartnerFilter{Commission: "exclude", Levels: []string{"ARANY", "GYEMANT", "PLATINA"}}, "bolt1,nagy.betu,ujmezo")
	check(PartnerFilter{Shops: []string{""}}, "bizo")
	check(PartnerFilter{Props: []string{"DIREKT"}, PropsNot: true}, "nincstk,torolt")
	check(PartnerFilter{SubFrom: "2025-01-01"}, "nagy.betu,ujmezo")
	check(PartnerFilter{SubFrom: "2023-04-25", SubTo: "2023-04-25"}, "bolt1")
	check(PartnerFilter{Query: "horgász"}, "bolt1")
	check(PartnerFilter{Query: "10002"}, "bolt2")
	check(PartnerFilter{Reps: []string{"ZSO"}, Exclude: []string{"BOLT2@example.com"}}, "bolt1,torolt")

	fc := B2BFacets(db, PartnerFilter{Reps: []string{"ZSO"}, Counties: []string{"Pest"}})
	var zso, gab, none FacetValue
	for _, v := range fc.Reps {
		switch v.Value {
		case "ZSO":
			zso = v
		case "GAB":
			gab = v
		case "":
			none = v
		}
	}
	// a képviselő-szempont saját kijelölése nem szűkít (Count: a többi feltétellel, itt Pest megye)
	if zso.Total != 3 || zso.Count != 1 || gab.Total != 3 || gab.Count != 0 || gab.Label != "Gábor Bence" || none.Label != "nincs képviselő" || fc.Reps[len(fc.Reps)-1].Value != "" {
		t.Errorf("képviselők: %+v", fc.Reps)
	}
	if fc.Counties[0].Value != "Budapest" || fc.Levels[0].Value != "BASIC" || fc.Levels[len(fc.Levels)-1].Value != "BIZOMANYOS" {
		t.Errorf("sorrend: %+v %+v", fc.Counties, fc.Levels)
	}
	if fc.Commission["no"] != 1 || fc.Internal["no"] != 1 {
		t.Errorf("igen/nem: %+v %+v", fc.Commission, fc.Internal)
	}
}

func TestB2BToPartner(t *testing.T) {
	list, _ := loadB2BFixture(t)
	opts := B2BMapOptions{RepPhotos: map[string]string{"GAB": "https://kep.example.com/gab.jpg"}}
	// a név a {nev} és a {ceg} változóba is bekerül (csupa nagybetűsből olvasható alakban),
	// a képviselő területe a partner megyéje
	p := B2BToPartner(findB2B(list, "nagy.betu@example.com"), 3, "B2B_HU", opts)
	if p.Name != "Nagy Betű Bt." || p.Company != "Nagy Betű Bt." || p.FallbackGreeting || p.RepName != "Gábor Bence" ||
		p.RepPhoto != "https://kep.example.com/gab.jpg" || p.RepRegion != "Bács-Kiskun megye" ||
		p.Extra["besorolas"] != "Gyémánt" || p.Extra["nazon"] != "10003" || p.Extra["feliratkozas"] != "2025.06.01." ||
		p.Extra["partnernev"] != "NAGY BETŰ BT." || p.Extra["megye"] != "Bács-Kiskun" || p.Source != "B2B_HU" || p.Row != 3 {
		t.Errorf("cég: %+v", p)
	}
	if p.Unsubscribe == "" || len(p.Token) != 32 {
		t.Error("a leiratkozó link és a token átjön")
	}
	// a felület felé a titkos mezők nem mennek ki
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), p.Token) || strings.Contains(string(b), "leiratkozas") {
		t.Errorf("titkos adat a JSON-ban: %s", b)
	}
	q := B2BToPartner(findB2B(list, "bolt2@example.com"), 1, "B2B_HU", B2BMapOptions{KeepRepSuffix: true, Greeting: "auto"})
	if q.Name != "Kiss Péter" || q.Company != "Kiss Péter" || q.FallbackGreeting || q.RepRegion != "Budapest" {
		t.Errorf("személynév: %+v", q)
	}
	if r := B2BToPartner(findB2B(list, "nagy.betu@example.com"), 1, "B2B_HU", B2BMapOptions{KeepRepSuffix: true, KeepCaps: true, Greeting: "auto"}); r.RepName != "Gábor Bence - Energofish Kft." ||
		r.Name != "NAGY BETŰ BT." || !r.FallbackGreeting {
		t.Errorf("utótag, nagybetű, automatikus megszólítás: %+v", r)
	}
	if r := B2BToPartner(findB2B(list, "bolt2@example.com"), 1, "B2B_HU", B2BMapOptions{Greeting: "fallback"}); r.Name != "Kiss Péter" || !r.FallbackGreeting {
		t.Errorf("tartalék: %+v", r)
	}
	if r := B2BToPartner(findB2B(list, "ujmezo@example.com"), 1, "B2B_HU", opts); r.Extra["ujmezo"] != "valami" || r.Extra["nyelv"] != "hu" {
		t.Errorf("új mező változóként: %+v", r.Extra)
	}

	// a levélben: a partner saját leiratkozó linkje változatlanul, és minden változó kitöltve
	c, _ := testContent(t)
	c["note.body"] = "{nev} | {ceg} | {kepviselo} | {terulet} | {megye} | {nazon} | {besorolas} | {partnernev}"
	tpls, err := LoadTemplates(os.DirFS("../.."), "sablonok")
	if err != nil {
		t.Fatal(err)
	}
	rd := Build(c, nil, &p, FindTemplate(tpls, "v4"), "")
	if rd.Values["footer.unsubscribe.url"] != p.Unsubscribe || rd.Values["note.greeting"] != "Kedves Nagy Betű Bt.!" ||
		rd.Values["note.body"] != "Nagy Betű Bt. | Nagy Betű Bt. | Gábor Bence | Bács-Kiskun megye | Bács-Kiskun | 10003 | Gyémánt | NAGY BETŰ BT." ||
		rd.Values["rep.region"] != "Bács-Kiskun megye" || rd.Values["rep.name"] != "Gábor Bence" {
		t.Errorf("levél: %q %q %q %q", rd.Values["footer.unsubscribe.url"], rd.Values["note.greeting"], rd.Values["note.body"], rd.Values["rep.region"])
	}
	// „cégeknek tartalék” beállítással a cég a tartalék megszólítást kapja, a változók maradnak
	pa := B2BToPartner(findB2B(list, "nagy.betu@example.com"), 3, "B2B_HU", B2BMapOptions{Greeting: "auto"})
	rd = Build(c, nil, &pa, FindTemplate(tpls, "v4"), "")
	if rd.Values["note.greeting"] != strings.TrimSpace(c["note.greetingFallback"]) || !strings.HasPrefix(rd.Values["note.body"], "Nagy Betű Bt. |") {
		t.Errorf("tartalék megszólítás: %q %q", rd.Values["note.greeting"], rd.Values["note.body"])
	}
	if iss := ValidatePartners([]Partner{p}); len(iss) != 0 {
		t.Errorf("a cégnév miatt nem kell megjegyzés: %+v", iss)
	}
	// ismert B2B változók nem „ismeretlenek” a tartalom ellenőrzésében
	known := map[string]bool{}
	for k := range p.Extra {
		known[k] = true
	}
	for _, is := range ValidateContentFor(c, nil, FindTemplate(tpls, "v4"), known) {
		if strings.Contains(is.Message, "ismeretlen változó") {
			t.Errorf("ismert változó ismeretlennek jelölve: %s", is.Message)
		}
	}
	unk := 0
	for _, is := range ValidateContent(c, nil, FindTemplate(tpls, "v4")) {
		if strings.Contains(is.Message, "ismeretlen változó") {
			unk++
		}
	}
	if unk == 0 {
		t.Error("partnertörzs nélkül a {megye} ismeretlen változó")
	}
}

func TestReadableName(t *testing.T) {
	for in, want := range map[string]string{
		"JDB HUNGARY ZRT.":          "JDB Hungary Zrt.",
		"HORGÁSZ CENTRUM KFT.":      "Horgász Centrum Kft.",
		"KISS PÉTER E.V.":           "Kiss Péter e.v.",
		"KOVÁCS ÉS TÁRSA BT":        "Kovács és Társa Bt.",
		"ZÖLD ÁG HORGÁSZBOLT":       "Zöld Ág Horgászbolt",
		"BALATON-PART 2000 KFT.":    "Balaton-Part 2000 Kft.",
		"  HALAS   TÓ  ":            "Halas Tó",
		"Kapitány Horgászbolt, Vác": "Kapitány Horgászbolt, Vác",
		"Kiss Péter":                "Kiss Péter",
		"RYBÁRSTVO SRO":             "Rybárstvo s.r.o.",
	} {
		if got := ReadableName(in); got != want {
			t.Errorf("ReadableName(%q) = %q, várt: %q", in, got, want)
		}
	}
	for in, want := range map[string]bool{"HORGÁSZ BT.": true, "Horgász Kft.": true, "Kiss Péter e.v.": true,
		"Kiss Péter": false, "Zöld Ág Horgászbolt": false, "Kft Horgászbolt": false} {
		if LooksLikeCompany(in) != want {
			t.Errorf("LooksLikeCompany(%q) != %v", in, want)
		}
	}
	for _, c := range [][3]string{{"B2B_HU", "Pest", "Pest megye"}, {"B2B_HU", "Budapest", "Budapest"}, {"B2B_HU", "Pest megye", "Pest megye"},
		{"B2B_HU", "", ""}, {"B2B_SK", "Bratislavský kraj", "Bratislavský kraj"}} {
		if got := B2BRegion(c[0], c[1]); got != c[2] {
			t.Errorf("B2BRegion(%q, %q) = %q", c[0], c[1], got)
		}
	}
}

func TestB2BStoreSync(t *testing.T) {
	data, _ := os.ReadFile("testdata/b2b-minta.json")
	const tok = "0123456789abcdef0123456789abcdef" // kitalált
	var mode atomic.Value
	mode.Store("ok")
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Query().Get("token") != tok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch mode.Load().(string) {
		case "500once":
			mode.Store("ok")
			http.Error(w, "hiba", http.StatusInternalServerError)
		case "html":
			_, _ = w.Write([]byte("<html>bejelentkezés</html>"))
		case "few":
			_, _ = w.Write([]byte(`[` + strings.SplitN(strings.TrimPrefix(string(data), "["), "},", 2)[0] + `}]`))
		default:
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(data)
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	xor := func(b []byte) ([]byte, error) {
		out := make([]byte, len(b))
		for i := range b {
			out[i] = b[i] ^ 0x5a
		}
		return out, nil
	}
	newStore := func() *B2BStore {
		s := NewB2BStore(dir)
		s.Protect, s.Unprotect, s.RetryDelay = xor, xor, time.Millisecond
		return s
	}
	s := newStore()
	src := srv.URL + "/export?action=export&token=" + tok
	mode.Store("500once")
	res, err := s.Sync(context.Background(), "B2B_HU", src, false)
	if err != nil || res.Log.New != 9 || res.Log.Active != 9 || res.Log.HTTP != 200 || hits.Load() != 2 {
		t.Fatalf("első szinkron (újrapróbálással): %v %+v, %d kérés", err, res.Log, hits.Load())
	}
	// a fájl titkosítva, csak a felhasználónak olvasható
	raw, _ := os.ReadFile(filepath.Join(dir, "partnertorzs-b2b_hu.dat"))
	if !strings.HasPrefix(string(raw), protectedMagic) || strings.Contains(string(raw), "example.com") {
		t.Error("a partnertörzs nincs titkosítva")
	}
	if st, _ := os.Stat(filepath.Join(dir, "partnertorzs-b2b_hu.dat")); st.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("jogosultság: %v", st.Mode())
	}
	// hibás token: a hibaüzenetben nincs benne a token
	_, err = s.Sync(context.Background(), "B2B_HU", srv.URL+"/export?token=ffffffffffffffffffffffffffffffff", false)
	if err == nil || strings.Contains(err.Error(), "ffffffff") || !strings.Contains(err.Error(), "403") {
		t.Errorf("403: %v", err)
	}
	// nem JSON → semmi nem változik
	mode.Store("html")
	if _, err := s.Sync(context.Background(), "B2B_HU", src, false); err == nil || !strings.Contains(err.Error(), "nem JSON") {
		t.Errorf("html: %v", err)
	}
	// gyanúsan kevés partner → megáll, force-szal lefut
	mode.Store("few")
	_, err = s.Sync(context.Background(), "B2B_HU", src, false)
	if !errors.Is(err, ErrSuspiciousExport) {
		t.Errorf("kevés: %v", err)
	}
	// újraindítás: a mentett (titkosított) adatbázis visszatölthető, a napló a hibákat is tartalmazza
	s2 := newStore()
	db, err := s2.DB("B2B_HU")
	if err != nil || db.ActiveCount() != 9 || len(db.Log) != 4 || db.Log[0].OK != true || db.Log[1].OK || db.SyncedAt.IsZero() {
		t.Fatalf("visszatöltés: %v %d aktív, napló: %+v", err, db.ActiveCount(), db.Log)
	}
	for _, l := range db.Log {
		b, _ := json.Marshal(l)
		// teljes e-mail cím nem lehet benne (a duplikátumoknál csak kitakart: „bel…@example.com”)
		if strings.Contains(string(b), tok) || regexp.MustCompile(`[A-Za-z0-9._%+-]@example\.com`).MatchString(string(b)) {
			t.Errorf("a naplóban titkos/személyes adat: %s", b)
		}
	}
	if res, err := s2.Sync(context.Background(), "B2B_HU", src, true); err != nil || res.Log.Inactivated != 8 || res.Log.Active != 1 {
		t.Errorf("kényszerítve: %v %+v", err, res.Log)
	}
	// titkosítás nélküli tár nem olvassa a titkosított fájlt
	plain := NewB2BStore(dir)
	if _, err := plain.DB("B2B_HU"); err == nil {
		t.Error("titkosított fájl visszafejtés nélkül")
	}
	if _, err := s.Sync(context.Background(), "B2B_SK", "", false); err == nil {
		t.Error("forrás nélkül")
	}
}

func TestB2BSecrets(t *testing.T) {
	text := "B2B HU: https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=0123456789abcdef0123456789abcdef\n\n" +
		"B2B SK: https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=fedcba9876543210fedcba9876543210\n" +
		"B2B COM: https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=00000000000000000000000000000001\n" +
		"B2B XX: https://energofish.hu/x?token=11111111111111111111111111111111"
	m := ParseB2BSources(text)
	if len(m) != 3 || m["B2B_HU"] != B2BExportURL+"0123456789abcdef0123456789abcdef" || m["B2B_COM"] == "" {
		t.Errorf("%v", m)
	}
	if NormalizeB2BSource("0123456789abcdef0123456789abcdef") != B2BExportURL+"0123456789abcdef0123456789abcdef" ||
		NormalizeB2BSource("https://x.hu/nincs-token") != "" || NormalizeB2BSource("valami") != "" {
		t.Error("NormalizeB2BSource")
	}
	if ms := MaskSource(m["B2B_SK"]); ms != "energofish.hu · token fedc…3210" {
		t.Error(ms)
	}
	if TokenFromURL("0123456789ABCDEF0123456789abcdef") != "0123456789abcdef0123456789abcdef" || TokenFromURL("https://x.hu/?token=abc") != "" {
		t.Error("TokenFromURL")
	}
	if MaskToken("0123456789abcdef0123456789abcdef") != "0123…cdef" {
		t.Error(MaskToken("0123456789abcdef0123456789abcdef"))
	}
	msg := MaskSecrets(`Get "https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=0123456789abcdef0123456789abcdef": dial tcp`)
	if strings.Contains(msg, "0123456789") || !strings.Contains(msg, "token=***") {
		t.Error(msg)
	}
	if g := FindB2BGroup("hu"); g == nil || g.ID != "B2B_HU" || g.Env != "WEBGALAMB_TOKEN_B2B_HU" {
		t.Errorf("%+v", g)
	}
	if !IsCompanyName("HORGÁSZ BT.") || IsCompanyName("Kiss Péter") || IsCompanyName("A") {
		t.Error("IsCompanyName")
	}
}
