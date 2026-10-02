package hirlevel

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

func loadFeedFixture(t *testing.T) []FeedProduct {
	t.Helper()
	f, err := os.Open("testdata/feed-minta.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	list, err := ParseFeed(f)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestParseFeedSpecSamples(t *testing.T) {
	list := loadFeedFixture(t)
	if len(list) != 3 {
		t.Fatalf("%d cikk", len(list))
	}
	a := list[0]
	if a.Code != "10000-327" || a.Plain != "10000327" || a.Name != "CRAB RED 12CM" || a.Family != "WIZARD CRAB" || a.FamilyID != 24021 ||
		a.Link != "https://energofish.hu/product.php/Mucsali-WIZARD-CRAB/24021/" || a.Category != "Műcsali" || a.Subcategory != "Gumihal" ||
		a.Brand != "Arno" || a.Wholesale != 395 || a.WholesaleAkc != 320 || a.Retail != 690 || a.RetailAkc != 530 || a.Stock != 1 || !a.LowStock ||
		a.Badge != "kifutó" {
		t.Errorf("1. cikk: %+v", a)
	}
	if len(a.Images) != 7 || a.Images[0].ID != "thumb" || a.Images[1].URL != "https://images.energofish.hu/codeimage/C10000-327.JPG" || a.Images[6].ID != "gallery3" {
		t.Errorf("képek: %+v", a.Images)
	}
	// a Cikkszám paraméter (id=5) és az üres paraméterek kimaradnak
	if len(a.Params) != 3 || a.Params[0] != (FeedParam{"Méret", "12 cm"}) || a.Params[1].Value != "Red" {
		t.Errorf("paraméterek: %+v", a.Params)
	}
	b := list[1]
	// üres code és large kép kimarad, a galériaképek maradnak
	ids := []string{}
	for _, im := range b.Images {
		ids = append(ids, im.ID)
	}
	if strings.Join(ids, ",") != "thumb,small,gallery1,gallery2,gallery3,gallery4" || b.RetailAkc != 0 || b.Subcategory != "Általános teleszkópos" {
		t.Errorf("2. cikk: %v %+v", ids, b)
	}
	c := list[2]
	// CDATA-ban &, entitás a linkben, NBSP, végi szóköz, üres kategória, szóköz a kép-URL-ben
	if c.Name != "SONIK & CO BAG 10PCS/BAG" || c.Link != "https://energofish.hu/product.php/Taska-A&B/999/" || c.Subcategory != "Feeder" ||
		c.Category != "" || c.Stock != 0 || c.Badge != "készletcsökkentett" || c.Images[0].ID != "code" ||
		c.Images[0].URL != "https://images.energofish.hu/codeimage/C%20SBSLA-S_G.JPG" {
		t.Errorf("3. cikk: %+v", c)
	}
}

func TestParseFeedErrors(t *testing.T) {
	for _, src := range []string{"", "<foo><Termek></Termek></foo>", "<xml></xml>", "<xml><Termek><Cikkszam>1</Cikk"} {
		if _, err := ParseFeed(strings.NewReader(src)); err == nil {
			t.Errorf("%q: hibát vártam", src)
		}
	}
}

func TestParseFeedEncodings(t *testing.T) {
	body := `<xml><Termek><Cikkszam>1</Cikkszam><CikkszamK>1-1</CikkszamK><Termeknev><![CDATA[ŐRÜLT ÁRVÍZTŰRŐ]]></Termeknev></Termek></xml>`
	// UTF-8 BOM-mal
	list, err := ParseFeed(strings.NewReader("\ufeff<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + body))
	if err != nil || list[0].Name != "ŐRÜLT ÁRVÍZTŰRŐ" {
		t.Errorf("BOM: %v %+v", err, list)
	}
	// régi magyar kódlap
	enc, _ := charmap.ISO8859_2.NewEncoder().String(`<?xml version="1.0" encoding="ISO-8859-2"?>` + body)
	list, err = ParseFeed(strings.NewReader(enc))
	if err != nil || list[0].Name != "ŐRÜLT ÁRVÍZTŰRŐ" {
		t.Errorf("ISO-8859-2: %v %+v", err, list)
	}
	if _, err := ParseFeed(strings.NewReader(`<?xml version="1.0" encoding="KOI8-R"?>` + body)); err == nil || !strings.Contains(err.Error(), "KOI8-R") {
		t.Errorf("ismeretlen kódolás: %v", err)
	}
}

func TestSearchFeed(t *testing.T) {
	list := loadFeedFixture(t)
	check := func(q string, want ...string) {
		t.Helper()
		res := SearchFeed(list, q, 10)
		got := []string{}
		for _, p := range res {
			got = append(got, p.Code)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%q: %v, várt %v", q, got, want)
		}
	}
	check("10000-327", "10000-327")
	check("10000327", "10000-327")
	check("10729", "10729-300")
	check("crab", "10000-327")
	check("wizard red", "10000-327")
	check("altalanos teleszkopos", "10729-300")
	check("Kamasaki 3m", "10729-300")
	check("sbsla-s_g", "SBSLA-S_G")
	check("sonik & co", "SBSLA-S_G")
	check("nincs ilyen")
	// névegyezés előbb, mint a főtermék/márka egyezése
	extra := append([]FeedProduct{}, list...)
	extra = append(extra, convertTermek(&xmlTermek{CikkszamK: "00001-001", Termeknev: "EXPERT RED", Fotermek: xmlText{Value: "CARP CRAB"}, Keszleten: "1"}))
	if res := SearchFeed(extra, "crab", 10); len(res) != 2 || res[0].Code != "10000-327" || res[1].Code != "00001-001" {
		t.Errorf("rangsor: %+v", res)
	}
	check("")
	// a készleten lévők előre kerülnek
	check("1", "10000-327", "10729-300", "SBSLA-S_G")
	if p := FindFeed(list, "10729300"); p == nil || p.Code != "10729-300" {
		t.Errorf("FindFeed: %+v", p)
	}
	if DealText(690, 530) != "−23% akció" || DealText(690, 0) != "" || DealText(100, 100) != "" {
		t.Error(DealText(690, 530))
	}
}

func TestFeedStoreDownloadCacheAndConditional(t *testing.T) {
	data, _ := os.ReadFile("testdata/feed-minta.xml")
	var hits, notModified atomic.Int32
	fail := atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if fail.Load() {
			http.Error(w, "karbantartás", http.StatusServiceUnavailable)
			return
		}
		if r.Header.Get("If-None-Match") == `"v1"` {
			notModified.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Type", "application/xml")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			_, _ = gz.Write(data)
			_ = gz.Close()
			return
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	dir := t.TempDir()

	s := NewFeedStore(dir)
	if !s.Refresh(srv.URL, false) {
		t.Fatal("nem indult el")
	}
	s.Wait(10 * time.Second)
	st := s.Status()
	if !st.Ready || st.Count != 3 || st.Error != "" || st.Source != "net" || st.Bytes != int64(len(data)) {
		t.Fatalf("%+v", st)
	}
	// rögtön újra: nem kérdezi le (MinInterval)
	if s.Refresh(srv.URL, false) {
		t.Error("túl gyakori lekérdezés")
	}

	// új példány (újraindítás): a gyorsítótárból tölt, a szerver 304-et ad
	s2 := NewFeedStore(dir)
	s2.Refresh(srv.URL, false)
	s2.Wait(10 * time.Second)
	if st := s2.Status(); !st.Ready || st.Count != 3 || st.Error != "" || notModified.Load() != 1 {
		t.Fatalf("gyorsítótár: %+v, 304: %d", st, notModified.Load())
	}

	// a szerver hibázik: a meglévő cikktörzs marad, hibaüzenettel
	fail.Store(true)
	s2.Refresh(srv.URL, true)
	s2.Wait(10 * time.Second)
	if st := s2.Status(); !st.Ready || st.Count != 3 || !strings.Contains(st.Error, "503") {
		t.Fatalf("hiba esetén: %+v", st)
	}

	// hibás tartalom nem írja felül a gyorsítótárat
	fail.Store(false)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>hiba</html>"))
	}))
	defer bad.Close()
	s3 := NewFeedStore(dir)
	s3.Refresh(bad.URL, true)
	s3.Wait(10 * time.Second)
	if st := s3.Status(); st.Ready || !strings.Contains(st.Error, "gyökérelem") {
		t.Fatalf("hibás tartalom: %+v", st)
	}
	gzData, _ := os.ReadFile(dir + "/termekadatok.xml.gz")
	zr, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(zr)
	if buf.Len() != len(data) {
		t.Errorf("a gyorsítótár sérült: %d bájt", buf.Len())
	}
}

func TestFeedToProduct(t *testing.T) {
	list := loadFeedFixture(t)
	a := list[0].ToProduct(FeedOptions{})
	if a.Code != "10000-327" || a.Name != "CRAB RED 12CM" || a.Alt != a.Name || a.Desc != "12 cm · Red · Crab" ||
		a.Price != "530\u00a0Ft" || a.Deal != "−23%, kifutó" || a.URL != list[0].Link || !a.On ||
		a.Image != "https://images.energofish.hu/codeimage/C10000-327.JPG" || len(a.Images) != 7 {
		t.Errorf("alap: %+v", a)
	}
	a = list[0].ToProduct(FeedOptions{Price: "wholesale", Image: "small", Link: "pattern"})
	if a.Price != "320\u00a0Ft + áfa" || a.Deal != "−19%, kifutó" || a.URL != "" || !strings.Contains(a.Image, "/smallimage/") {
		t.Errorf("nagyker: %+v", a)
	}
	if a := list[0].ToProduct(FeedOptions{Price: "none", Image: "thumb"}); a.Price != "" || a.Deal == "" || !strings.Contains(a.Image, "/thumbimage/") {
		t.Errorf("ár nélkül: %+v", a)
	}
	// a 2. cikknek nincs code és large képe: az előnyben részesített helyett a következő jön
	b := list[1].ToProduct(FeedOptions{Image: "large"})
	if !strings.Contains(b.Image, "/smallimage/") || b.Price != "8\u00a0690\u00a0Ft" || b.Deal != "" {
		t.Errorf("2. cikk: %+v", b)
	}
	// a szám értékű paraméter a nevével kerül a leírásba, a 30 karakteren túliak kimaradnak
	if d := FeedDesc(&list[1], 30); utf8.RuneCountInString(d) > 30 || d == "" {
		t.Errorf("leírás: %q", d)
	}
	c := list[2].ToProduct(FeedOptions{})
	if FeedDesc(&FeedProduct{Subcategory: "Feeder", Brand: "Sonik", Params: []FeedParam{{"Akció", "B"}}}, 30) != "Feeder · Sonik" {
		t.Error("leírás paraméter nélkül")
	}
	if c.Deal != "Kiárusítás" || !strings.Contains(c.Image, "codeimage/C%20SBSLA") {
		t.Errorf("3. cikk: %+v", c)
	}
	webp := []FeedImage{{"thumb", "https://x/T1.WEBP"}, {"code", "https://x/C1.webp"}, {"small", "https://x/S1.JPG"}}
	if PickImage(webp, "code") != "https://x/S1.JPG" || PickImage(webp[:2], "code") != "https://x/C1.webp" {
		t.Error("a WEBP csak végső esetben")
	}
	if PickImage([]FeedImage{{"gallery2", "g2"}}, "code") != "g2" || PickImage(nil, "code") != "" {
		t.Error("PickImage")
	}
	if FeedDesc(&FeedProduct{Params: []FeedParam{{"Méret", "8"}, {"Tagok száma", "3+3"}}}, 30) != "Méret: 8 · Tagok száma: 3+3" {
		t.Error(FeedDesc(&FeedProduct{Params: []FeedParam{{"Méret", "8"}, {"Tagok száma", "3+3"}}}, 30))
	}
}

func TestFeedStoreQueuedRefreshAndCacheOnly(t *testing.T) {
	data, _ := os.ReadFile("testdata/feed-minta.xml")
	var hits atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			<-release // az első letöltés addig tart, amíg a teszt engedi
		}
		w.Header().Set("Last-Modified", "Wed, 01 Oct 2026 10:00:00 GMT")
		if r.Header.Get("If-Modified-Since") != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	dir := t.TempDir()
	s := NewFeedStore(dir)
	s.MinInterval = 0
	s.Refresh(srv.URL, false)
	time.Sleep(50 * time.Millisecond)
	if st := s.Status(); !st.Loading || st.Ready {
		t.Fatalf("%+v", st)
	}
	// futás közbeni kérés: utána lefut (feltételes lekérdezés → 304)
	if !s.Refresh(srv.URL, false) {
		t.Error("a kérés nem került sorba")
	}
	close(release)
	s.Wait(10 * time.Second)
	if st := s.Status(); !st.Ready || hits.Load() != 2 || st.HTTP != 304 {
		t.Fatalf("sorba állított frissítés: %+v, %d lekérés", st, hits.Load())
	}
	// induláskor csak a gyorsítótár töltődik be, hálózat nélkül
	s2 := NewFeedStore(dir)
	s2.LoadCached(srv.URL)
	s2.Wait(10 * time.Second)
	if st := s2.Status(); !st.Ready || st.Count != 3 || st.Source != "cache" || st.CheckedAt != "" || hits.Load() != 2 {
		t.Fatalf("gyorsítótár: %+v", st)
	}
	// más címhez nem a régi gyorsítótár töltődik
	s3 := NewFeedStore(dir)
	s3.LoadCached(srv.URL + "/mas")
	s3.Wait(10 * time.Second)
	if s3.Status().Ready {
		t.Error("más cím gyorsítótára töltődött be")
	}
}
