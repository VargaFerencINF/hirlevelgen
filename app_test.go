package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	h "energofish/hirlevel/internal/hirlevel"

	"github.com/xuri/excelize/v2"
)

func newTestServer(t *testing.T) (*App, *httptest.Server) {
	t.Helper()
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app.state.Output.Dir = filepath.Join(t.TempDir(), "kimenet")
	srv := httptest.NewServer(app.routes())
	t.Cleanup(srv.Close)
	return app, srv
}

func call(t *testing.T, srv *httptest.Server, token, path string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
	req.Header.Set("X-Token", token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestAPIFlow(t *testing.T) {
	app, srv := newTestServer(t)

	// token nélkül tilos
	if code, _ := call(t, srv, "rossz", "/api/init", nil); code != http.StatusForbidden {
		t.Fatalf("token nélkül: %d", code)
	}
	code, init := call(t, srv, app.token, "/api/init", nil)
	if code != 200 || init["fields"] == nil || init["state"] == nil {
		t.Fatalf("init: %d %v", code, init)
	}

	// index: a token bekerül az oldalba
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(page), app.token) || strings.Contains(string(page), "{{TOKEN}}") {
		t.Error("az index nem tartalmazza a tokent")
	}

	// Excel betöltése
	xlsx, _ := filepath.Abs("demo/Energofish_partner_hirlevel_minta.xlsx")
	code, ex := call(t, srv, app.token, "/api/excel/load", map[string]string{"path": xlsx})
	if code != 200 {
		t.Fatalf("excel: %d %v", code, ex)
	}
	partners := ex["excel"].(map[string]any)["partners"].([]any)
	if len(partners) != 21 {
		t.Fatalf("%d partner", len(partners))
	}

	// állapot módosítása + előnézet
	app.mu.Lock()
	st := app.state
	app.mu.Unlock()
	st.Content["cover.headline"] = "Teszt {nev}"
	st.Template = "v1"
	if code, _ := call(t, srv, app.token, "/api/state", st); code != 200 {
		t.Fatalf("state: %d", code)
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/preview?p=1&tpl=v1-sotet-lemez", nil)
	req.Header.Set("X-Token", app.token)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(html), "Teszt Tóth Erika") || !strings.Contains(string(html), `src="/assets/energofish-mark-light.png"`) {
		t.Errorf("előnézet: %.300s", html)
	}

	// generálás két partnerre
	code, gen := call(t, srv, app.token, "/api/generate", map[string]any{"only": []int{0, 5}})
	if code != 200 {
		t.Fatalf("generálás: %d %v", code, gen)
	}
	if gen["generated"].(float64) != 2 {
		t.Errorf("generált: %v", gen["generated"])
	}
	if _, err := os.Stat(gen["index"].(string)); err != nil {
		t.Errorf("áttekintő: %v", err)
	}

	// feltöltés (drag & drop útvonal) ékezetes névvel: csak a beállított nevű Excel olvasható be
	data, _ := os.ReadFile(xlsx)
	upload := func() int {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/excel/upload", bytes.NewReader(data))
		req.Header.Set("X-Token", app.token)
		req.Header.Set("X-Filename", "Partnerlista%20okt%C3%B3ber.xlsx")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	if code := upload(); code != 400 {
		t.Fatalf("más nevű Excel: %d", code)
	}
	if code, st := call(t, srv, app.token, "/api/settings/save", map[string]any{"excelName": "Partnerlista október"}); code != 200 ||
		st["import"].(map[string]any)["excelName"] != "Partnerlista október.xlsx" {
		t.Fatalf("név beállítása: %d %v", code, st)
	}
	if code := upload(); code != 200 {
		t.Fatalf("feltöltés: %d", code)
	}
	app.mu.Lock()
	name := app.excel.FileName
	app.mu.Unlock()
	if name != "Partnerlista október.xlsx" {
		t.Errorf("fájlnév: %q", name)
	}

	// openurl csak biztonságos sémákkal
	if code, _ := call(t, srv, app.token, "/api/openurl", map[string]string{"url": "file:///c:/windows"}); code != 400 {
		t.Errorf("file: séma: %d", code)
	}
}

func TestSettingsPersist(t *testing.T) {
	dir := t.TempDir()
	app, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	xlsx, _ := filepath.Abs("demo/Energofish_partner_hirlevel_minta.xlsx")
	if err := app.loadExcelPath(xlsx, true); err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	app.state.Content["meta.subject"] = "Mentett tárgy"
	app.state.Products[0].Name = "Módosított név"
	app.state.Template = "v2-waterside"
	app.state.Feed = h.FeedOptions{URL: "https://példa.hu/feed.xml", Price: "wholesale", Image: "large"}
	app.state.Products[0].Images = []h.FeedImage{{ID: "code", URL: "https://kep/c.jpg"}}
	app.mu.Unlock()
	app.saveNow()

	app2, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	if app2.state.Content["meta.subject"] != "Mentett tárgy" || app2.state.Template != "v2-waterside" {
		t.Errorf("nem töltődött vissza: %q %q", app2.state.Content["meta.subject"], app2.state.Template)
	}
	if app2.excel == nil || len(app2.excel.Partners) != 21 {
		t.Fatal("az Excel nem töltődött újra")
	}
	if app2.state.Feed.URL != "https://példa.hu/feed.xml" || app2.state.Feed.Image != "large" || len(app2.state.Products[0].Images) != 1 {
		t.Errorf("a cikktörzs-beállítás nem töltődött vissza: %+v %+v", app2.state.Feed, app2.state.Products[0].Images)
	}
	if app2.state.Products[0].Name != "Módosított név" {
		t.Errorf("a változatlan Excel felülírta a termék-módosítást: %q", app2.state.Products[0].Name)
	}

	// ha az Excel módosult, a termékek onnan jönnek
	tmp := filepath.Join(dir, "lista.xlsx")
	data, _ := os.ReadFile(xlsx)
	_ = os.WriteFile(tmp, data, 0o644)
	if err := app2.loadExcelPath(tmp, false); err != nil {
		t.Fatal(err)
	}
	app2.saveNow()
	future := time.Now().Add(time.Hour)
	_ = os.Chtimes(tmp, future, future)
	app3, _ := NewApp(dir)
	if app3.state.Products[0].Name != "Wizard Vertix Vibrato" {
		t.Errorf("módosult Excel után: %q", app3.state.Products[0].Name)
	}
}

func TestHostGuard(t *testing.T) {
	h := hostGuard(1234, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for host, want := range map[string]int{"127.0.0.1:1234": 204, "localhost:1234": 204, "evil.example:1234": 403, "127.0.0.1:9999": 403} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = host
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("%s: %d", host, w.Code)
		}
	}
}

func TestAPITemplates(t *testing.T) {
	app, srv := newTestServer(t)
	zipData, _ := os.ReadFile("internal/hirlevel/testdata/tervezo-csomag-1.1.zip")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/templates/upload", bytes.NewReader(zipData))
	req.Header.Set("X-Token", app.token)
	req.Header.Set("X-Filename", "csomag.zip")
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("upload: %v %v", err, res.StatusCode)
	}
	var out struct {
		Templates []struct {
			ID        string `json:"id"`
			Overrides bool   `json:"overrides"`
		} `json:"templates"`
		Results []struct{ OK bool } `json:"results"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if len(out.Results) != 3 || len(out.Templates) != 3 || !out.Templates[2].Overrides {
		t.Fatalf("%+v", out)
	}
	// saját sablon + előnézet + saját kép kiszolgálása
	raw, _ := os.ReadFile("sablonok/v1-sotet-lemez.html")
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/templates/upload", bytes.NewReader(raw))
	req.Header.Set("X-Token", app.token)
	req.Header.Set("X-Filename", "v9-proba.html")
	res, _ = http.DefaultClient.Do(req)
	res.Body.Close()
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/preview?p=-1&tpl=v9-proba", nil)
	req.Header.Set("X-Token", app.token)
	res, _ = http.DefaultClient.Do(req)
	html, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(html), "Kedves Kovács Péter!") {
		t.Fatalf("előnézet: %d", res.StatusCode)
	}
	if r, _ := http.Get(srv.URL + "/assets/energofish-mark.png"); r.StatusCode != 200 {
		t.Errorf("assets: %d", r.StatusCode)
	}
	if code, _ := call(t, srv, app.token, "/api/templates/delete", map[string]string{"id": "v9-proba"}); code != 200 {
		t.Errorf("törlés: %d", code)
	}
	if code, _ := call(t, srv, app.token, "/api/templates/delete", map[string]string{"id": "v2-waterside"}); code != 200 {
		t.Errorf("frissítés törlése: %d", code)
	}
	if code, _ := call(t, srv, app.token, "/api/templates/delete", map[string]string{"id": "v2-waterside"}); code != 400 {
		t.Errorf("beépített törlése: %d", code)
	}
}

func TestAPIFeed(t *testing.T) {
	app, srv := newTestServer(t)
	data, _ := os.ReadFile("internal/hirlevel/testdata/feed-minta.xml")
	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rossz" {
			http.Error(w, "nincs", http.StatusNotFound)
			return
		}
		_, _ = w.Write(data)
	}))
	defer feed.Close()

	// hibás cím: a hibaüzenet nem „error” kulcson jön (azt a felület API-hibának venné)
	if code, st := call(t, srv, app.token, "/api/feed/refresh", map[string]any{"url": "ftp://x"}); code != 400 || st["error"] == nil {
		t.Errorf("ftp cím: %d %v", code, st)
	}
	call(t, srv, app.token, "/api/feed/refresh", map[string]any{"url": feed.URL + "/rossz"})
	app.feed.Wait(10 * time.Second)
	code, st := call(t, srv, app.token, "/api/feed/status", nil)
	if code != 200 || st["error"] != nil || !strings.Contains(st["lastError"].(string), "404") || st["ready"] != false {
		t.Fatalf("hibás feed: %d %v", code, st)
	}

	app.mu.Lock()
	app.state.Feed.URL = feed.URL
	app.state.Products = []h.Product{{On: true, Code: "10000327", Name: "Excelből"}, {On: true, Code: "NINCS-1"}}
	app.mu.Unlock()
	call(t, srv, app.token, "/api/feed/refresh", map[string]any{"force": true})
	app.feed.Wait(10 * time.Second)
	_, st = call(t, srv, app.token, "/api/feed/status", nil)
	if st["ready"] != true || st["count"].(float64) != 3 {
		t.Fatalf("feed: %v", st)
	}
	_, res := call(t, srv, app.token, "/api/feed/search", map[string]any{"q": "kamasaki", "limit": 5})
	list := res["results"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["code"] != "10729-300" || list[0].(map[string]any)["added"] != false {
		t.Errorf("keresés: %v", res)
	}
	_, res = call(t, srv, app.token, "/api/feed/search", map[string]any{"q": "10000-327"})
	if r0 := res["results"].([]any)[0].(map[string]any); r0["added"] != true || len(r0["images"].([]any)) != 7 {
		t.Errorf("a már felvett cikk jelölése: %v", r0)
	}
	_, res = call(t, srv, app.token, "/api/feed/products", map[string]any{"codes": []string{"10729-300", "nincs"}, "options": map[string]string{"price": "wholesale"}})
	prods := res["products"].([]any)
	if len(prods) != 1 || prods[0].(map[string]any)["price"] != "4\u00a0970\u00a0Ft + áfa" || res["missing"].([]any)[0] != "nincs" {
		t.Errorf("átvétel: %v", res)
	}
	_, res = call(t, srv, app.token, "/api/feed/images", map[string]any{"codes": []string{"10000327", "NINCS-1"}})
	imgs := res["images"].(map[string]any)
	if res["ready"] != true || len(imgs) != 1 || len(imgs["10000327"].([]any)) != 7 {
		t.Errorf("képek: %v", res)
	}
}

func TestAPIB2B(t *testing.T) {
	const tok = "0123456789abcdef0123456789abcdef" // kitalált
	data, _ := os.ReadFile("internal/hirlevel/testdata/b2b-minta.json")
	var list []map[string]any
	_ = json.Unmarshal(data, &list)
	export := list
	fail := false
	srv0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != tok {
			http.Error(w, "forbidden", 403)
			return
		}
		if fail {
			http.Error(w, "hiba", 500)
			return
		}
		if strings.Contains(r.URL.Path, "leiratkozas") {
			t.Error("a leiratkozó linket meghívták!")
		}
		_ = json.NewEncoder(w).Encode(export)
	}))
	defer srv0.Close()
	dir := t.TempDir()
	app, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	app.state.Output.Dir = filepath.Join(t.TempDir(), "kimenet")
	app.b2b.RetryDelay = time.Millisecond
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	noSecret := func(where string, v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), tok) || strings.Contains(string(b), "leiratkozas.html") {
			t.Errorf("%s: titkos adat a válaszban", where)
		}
	}

	// forrás beillesztése (több soros szövegből)
	_, st := call(t, srv, app.token, "/api/b2b/sources", map[string]any{"text": "B2B HU: " + srv0.URL + "/export?action=export&token=" + tok + "\nB2B XX: valami"})
	noSecret("sources", st)
	g0 := st["groups"].([]any)[0].(map[string]any)
	if st["saved"].([]any)[0] != "B2B HU" || g0["configured"] != true || !strings.Contains(g0["masked"].(string), "0123…cdef") {
		t.Fatalf("források: %v", st)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "partnerforrasok.dat")); secretsProtected && strings.Contains(string(raw), tok) {
		t.Error("a forrás nincs titkosítva")
	}

	// szinkron
	_, st = call(t, srv, app.token, "/api/b2b/sync", map[string]any{"group": "B2B_HU"})
	noSecret("sync", st)
	if st["failed"] != nil || st["result"].(map[string]any)["log"].(map[string]any)["new"].(float64) != 9 {
		t.Fatalf("szinkron: %v", st)
	}

	// halmaz lekérdezése
	_, q := call(t, srv, app.token, "/api/b2b/query", map[string]any{"group": "B2B_HU", "filter": map[string]any{"reps": []string{"ZSO"}}})
	noSecret("query", q)
	if q["count"].(float64) != 3 || q["mailable"].(float64) != 8 || q["noMail"].(float64) != 1 || len(q["rows"].([]any)) != 3 || !strings.Contains(q["summary"].(string), "Szél Zsófia") {
		t.Fatalf("lekérdezés: %v", q)
	}

	// betöltés partnerlistaként
	_, ld := call(t, srv, app.token, "/api/b2b/load", map[string]any{"group": "B2B_HU", "filter": map[string]any{"internal": "exclude"}, "name": "Teszt halmaz",
		"options": map[string]any{"repPhotos": map[string]string{"ZSO": "https://kep.example.com/zso.jpg"}}})
	noSecret("load", ld)
	ex := ld["excel"].(map[string]any)
	parts := ex["partners"].([]any)
	if ex["source"] != "b2b" || len(parts) != 7 || ex["b2b"].(map[string]any)["selected"].(float64) != 7 || !strings.HasPrefix(ex["fileName"].(string), "Teszt halmaz") {
		t.Fatalf("betöltés: %v", ex)
	}

	// az előnézetben a valódi leiratkozó link nem szerepel
	for i := range parts {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/preview?p=%d", srv.URL, i), nil)
		req.Header.Set("X-Token", app.token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if strings.Contains(string(body), "leiratkozas.html") || !strings.Contains(string(body), previewUnsubscribe) {
			t.Fatalf("előnézet %d: a valódi leiratkozó link látszik", i)
		}
	}
	if code, r := call(t, srv, app.token, "/api/openurl", map[string]any{"url": "https://energofish.hu/leiratkozas.html?&c=0123456789abcdef0123"}); code != 400 || !strings.Contains(r["error"].(string), "leiratkoz") {
		t.Errorf("leiratkozó link megnyitása: %d %v", code, r)
	}

	// küldés előtt kötelező frissítés: egy partner leiratkozott, egy új jött
	var next []map[string]any
	for _, p := range list {
		if p["Email_cim"] != "bolt2@example.com" {
			next = append(next, p)
		}
	}
	nw := map[string]any{}
	for k, v := range list[0] {
		nw[k] = v
	}
	nw["Email_cim"], nw["Nazon"], nw["Token"] = "uj@example.com", "10099", "fedcba9876543210fedcba9876543210"
	nw["Leiratkozas_link"] = "https://energofish.hu/leiratkozas.html?&c=fedcba9876543210fedc"
	export = append(next, nw)
	all := []int{}
	for i := range parts {
		all = append(all, i)
	}
	_, gen := call(t, srv, app.token, "/api/generate", map[string]any{"only": all})
	noSecret("generate", map[string]any{"sync": gen["sync"], "excel": gen["excel"]})
	sync := gen["sync"].(map[string]any)
	res := gen["result"].(map[string]any)
	if sync["dropped"].(float64) != 1 || sync["added"].(float64) != 1 || res["generated"].(float64) != 7 {
		t.Fatalf("generálás frissítéssel: %v", gen)
	}
	// a kész levélben a partner saját leiratkozó linkje van
	files, _ := filepath.Glob(filepath.Join(res["folder"].(string), "html", "*uj@example.com*.html"))
	if len(files) != 1 {
		t.Fatalf("az új partner levele: %v", files)
	}
	html, _ := os.ReadFile(files[0])
	if !strings.Contains(string(html), "leiratkozas.html?&amp;c=fedcba9876543210fedc") {
		t.Error("a levélben nincs a partner saját leiratkozó linkje")
	}
	if idx, _ := os.ReadFile(filepath.Join(res["folder"].(string), "attekinto.html")); !strings.Contains(string(idx), "valódi, egyedi leiratkozó linkjei") {
		t.Error("az áttekintőben nincs figyelmeztetés a leiratkozó linkekről")
	}
	if fs, _ := filepath.Glob(filepath.Join(res["folder"].(string), "html", "*bolt2@example.com*")); len(fs) != 0 {
		t.Error("a leiratkozott partner levelet kapott")
	}

	// sikertelen frissítés: a felület dönthet (nem „error”), skipSync-kel generál
	fail = true
	code, gen := call(t, srv, app.token, "/api/generate", map[string]any{"only": []int{0}})
	if code != 200 || gen["syncFailed"] == nil || gen["error"] != nil {
		t.Fatalf("sikertelen frissítés: %d %v", code, gen)
	}
	if _, gen = call(t, srv, app.token, "/api/generate", map[string]any{"only": []int{0}, "skipSync": true}); gen["generated"].(float64) != 1 {
		t.Errorf("skipSync: %v", gen)
	}
	fail = false

	// mentett halmaz
	_, pr := call(t, srv, app.token, "/api/b2b/presets", map[string]any{"action": "save", "name": "ZSO boltjai", "group": "B2B_HU", "filter": map[string]any{"reps": []string{"ZSO"}}})
	if len(pr["presets"].([]any)) != 1 {
		t.Errorf("mentés: %v", pr)
	}
	app.saveNow()

	// újraindítás: a halmaz a helyi adatbázisból visszaépül
	app2, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	if app2.excel == nil || app2.excel.Source != "b2b" || len(app2.excel.Partners) != 7 || len(app2.b2bSet.Presets) != 1 || app2.excel.Partners[0].Unsubscribe == "" {
		t.Fatalf("visszaállítás: %+v", app2.excel)
	}
	// Excel betöltése után a partnerek onnan jönnek
	xlsx, _ := filepath.Abs("demo/Energofish_partner_hirlevel_minta.xlsx")
	if err := app2.loadExcelPath(xlsx, false); err != nil || app2.b2bSet.Loaded != nil {
		t.Errorf("Excel: %v", err)
	}
}

func TestB2BImportFileAndSourcesFile(t *testing.T) {
	dir := t.TempDir()
	// a program mellé / a beállítások közé tett forráslista egyszer beolvasódik, majd törlődik
	txt := "Partnerforrások\nB2B HU: https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=0123456789abcdef0123456789abcdef\n" +
		"B2B SK: https://energofish.hu/admintool/webgalamb_mod.php?action=export&token=fedcba9876543210fedcba9876543210\n"
	if err := os.WriteFile(filepath.Join(dir, sourcesImportFile), []byte(txt), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	app.importSourcesFiles()
	if _, err := os.Stat(filepath.Join(dir, sourcesImportFile)); !os.IsNotExist(err) {
		t.Error("a nyílt szöveges forrásfájl nem törlődött")
	}
	if src, origin := app.sourceFor("B2B_SK"); origin != "saved" || !strings.HasSuffix(src, "fedcba9876543210fedcba9876543210") {
		t.Errorf("forrás: %q %q", src, origin)
	}
	if len(app.notices) != 1 || app.notices[0]["kind"] != "ok" || strings.Contains(app.notices[0]["text"], "0123456789abcdef") {
		t.Errorf("üzenet: %v", app.notices)
	}
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	_, init := call(t, srv, app.token, "/api/init", nil)
	if n := init["notices"].([]any); len(n) != 1 {
		t.Errorf("init üzenetek: %v", init["notices"])
	}

	// kézzel letöltött export (JSON-fájl) betöltése
	data, _ := os.ReadFile("internal/hirlevel/testdata/b2b-minta.json")
	upload := func(group, force string, body []byte) map[string]any {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/b2b/import", bytes.NewReader(body))
		req.Header.Set("X-Token", app.token)
		req.Header.Set("X-Group", group)
		req.Header.Set("X-Filename", "webgalamb_37_B2B-teljes-celcsoport.json")
		req.Header.Set("X-Force", force)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return out
	}
	r := upload("B2B_HU", "0", data)
	l := r["result"].(map[string]any)["log"].(map[string]any)
	if r["failed"] != nil || l["new"].(float64) != 9 || l["source"] != "fájl: webgalamb_37_B2B-teljes-celcsoport.json" {
		t.Fatalf("fájlbetöltés: %v", r)
	}
	// hibás fájl: nem változik semmi
	r = upload("B2B_HU", "0", []byte("<html>"))
	if r["failed"] == nil || r["error"] != nil {
		t.Errorf("hibás fájl: %v", r)
	}
	db, _ := app.b2b.DB("B2B_HU")
	if db.ActiveCount() != 9 {
		t.Errorf("aktív: %d", db.ActiveCount())
	}
	// ismeretlen célcsoport
	if r := upload("B2B_XX", "0", data); r["error"] == nil {
		t.Error("ismeretlen célcsoport")
	}
}

func TestImportExcelSettings(t *testing.T) {
	dir := t.TempDir()
	xdir := filepath.Join(dir, "hírlevél import")
	_ = os.MkdirAll(xdir, 0o755)
	demo, _ := os.ReadFile("demo/Energofish_partner_hirlevel_minta.xlsx")
	app, err := NewApp(filepath.Join(dir, "cfg"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	// hibás beállítások
	for _, bad := range []map[string]any{{"excelName": "a/b.xlsx"}, {"excelName": "lista.csv"}, {"excelDir": "relativ"}, {"excelDir": filepath.Join(dir, "nincs")}} {
		if code, _ := call(t, srv, app.token, "/api/settings/save", bad); code != 400 {
			t.Errorf("%v: %d", bad, code)
		}
	}
	_, st := call(t, srv, app.token, "/api/settings/save", map[string]any{"excelDir": xdir})
	in := st["import"].(map[string]any)
	if in["excelName"] != DefaultImportExcel || in["exists"] != false || in["path"] != filepath.Join(xdir, DefaultImportExcel) {
		t.Fatalf("beállítás: %v", in)
	}
	// nincs még fájl → érthető hiba
	if code, r := call(t, srv, app.token, "/api/excel/import", nil); code != 400 || !strings.Contains(r["error"].(string), "nem található") {
		t.Errorf("hiányzó fájl: %d %v", code, r)
	}
	_ = os.WriteFile(filepath.Join(xdir, DefaultImportExcel), demo, 0o644)
	_ = os.WriteFile(filepath.Join(xdir, "masik.xlsx"), demo, 0o644)
	if code, r := call(t, srv, app.token, "/api/excel/import", nil); code != 200 || len(r["excel"].(map[string]any)["partners"].([]any)) != 21 || r["import"].(map[string]any)["exists"] != true {
		t.Fatalf("betöltés: %d", code)
	}
	if code, _ := call(t, srv, app.token, "/api/excel/load", map[string]any{"path": filepath.Join(xdir, "masik.xlsx")}); code != 400 {
		t.Error("más nevű Excel betöltése")
	}
	app.saveNow()
	// újraindításkor a beállított helyről töltődik
	app2, _ := NewApp(filepath.Join(dir, "cfg"))
	if app2.excel == nil || app2.excel.Path != filepath.Join(xdir, DefaultImportExcel) {
		t.Fatalf("újraindítás: %+v", app2.excel)
	}
	// a név változik → a régi fájl nem töltődik be
	app2.imp.ExcelName = "Uj_lista.xlsx"
	app2.saveNow()
	app3, _ := NewApp(filepath.Join(dir, "cfg"))
	if app3.excel != nil {
		t.Error("más nevű Excel töltődött be")
	}
	if n, err := normalizeExcelName("Partnerek"); err != nil || n != "Partnerek.xlsx" {
		t.Error(n, err)
	}
}

func TestBadExcelAndFactoryReset(t *testing.T) {
	dir := t.TempDir()
	app, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	xlsx, _ := filepath.Abs("demo/Energofish_partner_hirlevel_minta.xlsx")
	if err := app.loadExcelPath(xlsx, true); err != nil {
		t.Fatal(err)
	}
	// partner nélküli Excel: elutasítva, a korábbi lista marad
	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", "Cikkszám")
	_ = f.SetCellValue("Sheet1", "B1", "Ár")
	var buf bytes.Buffer
	_ = f.Write(&buf)
	if err := app.loadExcelData(buf.Bytes(), "Energofish_partner_hirlevel_minta.xlsx", time.Now(), true); err == nil || !strings.Contains(err.Error(), "nem találtam egyetlen partnert") {
		t.Errorf("rossz Excel: %v", err)
	}
	if app.excel == nil || len(app.excel.Partners) != 21 {
		t.Fatal("a korábbi lista elveszett")
	}
	// üres listák null helyett
	app.excel.Products = nil
	app.mu.Lock()
	v := app.excelViewLocked()
	app.mu.Unlock()
	if b, _ := json.Marshal(v); strings.Contains(string(b), `"products":null`) {
		t.Error("null lista a felület felé")
	}

	// teljes visszaállítás
	app.mu.Lock()
	app.state.Content["meta.subject"] = "Átírt tárgy"
	app.state.Products = nil
	app.imp.ExcelName = "Mas.xlsx"
	app.b2bSet.Presets = []B2BPreset{{Name: "x", Group: "B2B_HU"}}
	app.mu.Unlock()
	app.saveNow()
	_ = app.saveSources(map[string]string{"B2B_HU": h.B2BExportURL + "0123456789abcdef0123456789abcdef"})
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	_, r := call(t, srv, app.token, "/api/settings/reset", map[string]any{"sources": false, "b2b": false})
	if r["ok"] != true || r["backup"] == "" {
		t.Fatalf("visszaállítás: %v", r)
	}
	old, _ := os.ReadFile(r["backup"].(string))
	if !strings.Contains(string(old), "Átírt tárgy") {
		t.Error("a mentés nem a régi beállításokat tartalmazza")
	}
	app2, _ := NewApp(dir)
	if app2.state.Content["meta.subject"] == "Átírt tárgy" || len(app2.state.Products) != 6 || app2.excel != nil || app2.imp.ExcelName != "" || len(app2.b2bSet.Presets) != 1 {
		t.Errorf("alapállapot: %q %d %v %q %d", app2.state.Content["meta.subject"], len(app2.state.Products), app2.excel != nil, app2.imp.ExcelName, len(app2.b2bSet.Presets))
	}
	if src, _ := app2.sourceFor("B2B_HU"); src == "" {
		t.Error("a forrásoknak meg kellett maradniuk")
	}
	call(t, srv, app.token, "/api/settings/reset", map[string]any{"sources": true, "b2b": true})
	app3, _ := NewApp(dir)
	if src, _ := app3.sourceFor("B2B_HU"); src != "" || len(app3.b2bSet.Presets) != 0 {
		t.Error("a források / halmazok nem törlődtek")
	}
	// -alaphelyzet kapcsoló
	if b, err := resetSettingsFile(dir); err != nil || b == "" {
		t.Errorf("kapcsoló: %v %q", err, b)
	}
	if _, err := os.Stat(filepath.Join(dir, "beallitasok.json")); !os.IsNotExist(err) {
		t.Error("a beállításfájl megmaradt")
	}
}

// A B2B partnertörzsből betöltött partnerek adatai a levél változóiba kerülnek.
func TestB2BVariablesInPreview(t *testing.T) {
	data, _ := os.ReadFile("internal/hirlevel/testdata/b2b-minta.json")
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app.state.Output.Dir = filepath.Join(t.TempDir(), "kimenet")
	if _, err := app.b2b.ImportData("B2B_HU", data, "minta.json", false); err != nil {
		t.Fatal(err)
	}
	app.state.Content["note.body"] = "Partner: {nev} · cég: {ceg} · megye: {megye} · azonosító: {nazon} · képviselő: {kepviselo}, {terulet}"
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	_, ld := call(t, srv, app.token, "/api/b2b/load", map[string]any{"group": "B2B_HU", "filter": map[string]any{"reps": []string{"GAB"}}})
	ex := ld["excel"].(map[string]any)
	idx := -1
	for i, p := range ex["partners"].([]any) {
		if p.(map[string]any)["email"] == "nagy.betu@example.com" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("a partner nincs a halmazban: %v", ex["partners"])
	}
	toks, _ := json.Marshal(ex["tokens"])
	if !strings.Contains(string(toks), "{megye}") || !strings.Contains(string(toks), "{nazon}") || strings.Contains(string(toks), "{nev}") {
		t.Errorf("a változó-lista: %s", toks)
	}
	// a tartalom ellenőrzése nem jelöli ismeretlennek a B2B változókat
	b, _ := json.Marshal(ld["issues"])
	if strings.Contains(string(b), "ismeretlen változó") {
		t.Errorf("ismeretlen változó: %s", b)
	}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/preview?p=%d", srv.URL, idx), nil)
	req.Header.Set("X-Token", app.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{"Kedves Nagy Betű Bt.!", "Partner: Nagy Betű Bt. · cég: Nagy Betű Bt. · megye: Bács-Kiskun · azonosító: 10003 · képviselő: Gábor Bence, Bács-Kiskun megye",
		"Bács-Kiskun megye", "gabor.bence@example.com"} {
		if !strings.Contains(html, want) {
			t.Errorf("az előnézetből hiányzik: %q", want)
		}
	}
	if strings.Contains(html, "{nev}") || strings.Contains(html, "{megye}") || strings.Contains(html, "leiratkozas.html?&amp;c=3130") {
		t.Error("kitöltetlen változó vagy valódi leiratkozó link az előnézetben")
	}
}

// A partnerválasztó térképe: a partnertörzs megyéi a térkép régióira esnek.
func TestAPIB2BMap(t *testing.T) {
	data, _ := os.ReadFile("internal/hirlevel/testdata/b2b-minta.json")
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.b2b.ImportData("B2B_HU", data, "minta.json", false); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	_, r := call(t, srv, app.token, "/api/b2b/map", map[string]any{"group": "B2B_HU"})
	m := r["map"].(map[string]any)
	regions := m["regions"].([]any)
	if len(regions) != 20 || !strings.HasPrefix(m["viewBox"].(string), "0 0 ") {
		t.Fatalf("térkép: %d régió, %v", len(regions), m["viewBox"])
	}
	if _, ok := regions[0].(map[string]any)["keys"]; ok {
		t.Error("a névváltozatok fölöslegesen mennek a felületre")
	}
	match := r["match"].(map[string]any)
	for v, id := range map[string]string{"Pest": "HU-PE", "Budapest": "HU-BU", "Bács-Kiskun": "HU-BK", "Fejér": "HU-FE"} {
		ids, _ := match[v].([]any)
		if len(ids) != 1 || ids[0] != id {
			t.Errorf("%s → %v", v, match[v])
		}
	}
	if u := r["unmatched"].([]any); len(u) != 0 {
		t.Errorf("nem párosított megyék: %v", u)
	}
	// célcsoport szinkron nélkül: a térkép megvan, párosítandó érték nincs
	_, r = call(t, srv, app.token, "/api/b2b/map", map[string]any{"group": "B2B_DE"})
	if r["map"].(map[string]any)["title"] != "Németország" || len(r["match"].(map[string]any)) != 0 {
		t.Errorf("DE: %v", r["map"].(map[string]any)["title"])
	}
	// a nemzetközi célcsoport földgömbje: országok földrajzi koordinátákkal
	_, r = call(t, srv, app.token, "/api/b2b/map", map[string]any{"group": "B2B_COM"})
	gm := r["map"].(map[string]any)
	if gm["kind"] != "globe" || len(gm["regions"].([]any)) < 200 {
		t.Fatalf("COM: %v, %d ország", gm["kind"], len(gm["regions"].([]any)))
	}
	if g, ok := gm["regions"].([]any)[0].(map[string]any)["g"].([]any); !ok || len(g) == 0 {
		t.Error("COM: nincs alakzat")
	}
}

// Postmark-küldés a felület API-ján át, hamis Postmark szerverrel.
func TestAPIPostmarkSend(t *testing.T) {
	const tok = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" // kitalált
	var mu sync.Mutex
	var sent []map[string]any
	pm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Postmark-Server-Token") != tok && r.Header.Get("X-Postmark-Server-Token") != "POSTMARK_API_TEST" {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"ErrorCode":10,"Message":"Bad token"}`)
			return
		}
		switch r.URL.Path {
		case "/message-streams/broadcast":
			fmt.Fprint(w, `{"ID":"broadcast","Name":"Broadcasts","MessageStreamType":"Broadcasts","SubscriptionManagementConfiguration":{"UnsubscribeHandlingType":"Custom"}}`)
		case "/message-streams/broadcast/suppressions/dump":
			fmt.Fprint(w, `{"Suppressions":[{"EmailAddress":"bolt2@example.com","SuppressionReason":"HardBounce","Origin":"Recipient","CreatedAt":"2026-10-01T10:00:00Z"}]}`)
		case "/email/batch":
			var msgs []map[string]any
			_ = json.NewDecoder(r.Body).Decode(&msgs)
			mu.Lock()
			sent = append(sent, msgs...)
			mu.Unlock()
			out := make([]map[string]any, len(msgs))
			for i := range msgs {
				out[i] = map[string]any{"ErrorCode": 0, "Message": "OK", "MessageID": fmt.Sprintf("m-%d", i), "To": msgs[i]["To"]}
			}
			_ = json.NewEncoder(w).Encode(out)
		default:
			w.WriteHeader(404)
		}
	}))
	defer pm.Close()

	data, _ := os.ReadFile("internal/hirlevel/testdata/b2b-minta.json")
	dir := t.TempDir()
	app, err := NewApp(dir)
	if err != nil {
		t.Fatal(err)
	}
	app.postmarkBase = pm.URL
	app.postmarkSleep = func(context.Context, time.Duration) error { return nil }
	app.imageCheck = func(_ context.Context, ts []h.ImageTarget) []h.ImageCheck {
		out := make([]h.ImageCheck, len(ts))
		for i, x := range ts {
			out[i] = h.ImageCheck{ImageTarget: x, Status: "ok"}
		}
		return out
	}
	out := filepath.Join(t.TempDir(), "kimenet")
	app.state.Output.Dir = out
	app.state.Content["assets.base"] = "https://kepek.example.com/hirlevel"
	if _, err := app.b2b.ImportData("B2B_HU", data, "minta.json", false); err != nil {
		t.Fatal(err)
	}
	if err := app.loadB2BSet("B2B_HU", h.PartnerFilter{Internal: "exclude"}, "", nil); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.routes())
	defer srv.Close()
	noTok := func(where string, v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), tok) {
			t.Errorf("%s: token a válaszban", where)
		}
	}

	// beállítások: a token maszkolva látszik
	_, v := call(t, srv, app.token, "/api/postmark/settings/save", map[string]any{"liveToken": tok,
		"settings": map[string]any{"stream_id": "broadcast", "from": "Energofish Partner Brief <hirlevel@energofish.hu>", "internal_test_addresses": []string{"teszt@energofish.hu"},
			"test_count": 2, "track_opens": true, "track_links": "None", "utm": h.DefaultUTM, "one_click": true, "reply_to_rep": true}})
	noTok("settings", v)
	if v["live"].(map[string]any)["set"] != true || !strings.Contains(v["live"].(map[string]any)["masked"].(string), "aaaa…") {
		t.Fatalf("beállítások: %v", v)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "postmark-tokenek.dat")); secretsProtected && strings.Contains(string(raw), tok) {
		t.Error("a token nincs titkosítva")
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "postmark.json")); strings.Contains(string(raw), tok) {
		t.Error("token a postmark.json-ban")
	}

	n := len(app.excel.Partners)
	only := make([]int, n)
	for i := range only {
		only[i] = i
	}
	req := map[string]any{"mode": "eles", "campaign": "teszt-kampany", "only": only, "skipSync": true}
	_, c := call(t, srv, app.token, "/api/postmark/check", req)
	noTok("check", c)
	plan := c["plan"].(map[string]any)
	if _, ok := plan["errors"].([]any); !ok {
		t.Error("a hibák listája null")
	}
	if c["ok"] != true || c["stream"].(map[string]any)["handling"] != "Custom" || c["suppressed"].(float64) != 1 {
		t.Fatalf("ellenőrzés: %v", c)
	}
	recipients := int(plan["recipients"].(float64))
	ex, _ := json.Marshal(plan["excluded"])
	if recipients == 0 || !strings.Contains(string(ex), "bolt2@example.com") || !strings.Contains(string(ex), "HardBounce") {
		t.Fatalf("terv: %d címzett, kizárva: %s", recipients, ex)
	}

	// élesen megerősítés nélkül nem indul
	_, er := call(t, srv, app.token, "/api/postmark/send", req)
	if resp := fmt.Sprint(er["error"]); !strings.Contains(resp, "címzettek számát") {
		t.Errorf("megerősítés nélkül: %s", resp)
	}
	req["confirm"] = recipients
	_, st := call(t, srv, app.token, "/api/postmark/send", req)
	if st["started"] != true {
		t.Fatalf("küldés: %v", st)
	}
	var job map[string]any
	for i := 0; i < 200; i++ {
		_, s := call(t, srv, app.token, "/api/postmark/status", nil)
		job = s["job"].(map[string]any)
		if job["running"] != true {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	p := job["progress"].(map[string]any)
	if int(p["ok"].(float64)) != recipients || job["lastError"] != nil {
		t.Fatalf("eredmény: %v", job)
	}
	mu.Lock()
	first := sent[0]
	mu.Unlock()
	hdr, _ := json.Marshal(first["Headers"])
	if first["MessageStream"] != "broadcast" || first["Tag"] != "teszt-kampany" || !strings.Contains(string(hdr), "List-Unsubscribe") ||
		!strings.Contains(string(hdr), "leiratkozas.html") {
		t.Errorf("levél: %v %s", first["To"], hdr)
	}
	logData, _ := os.ReadFile(job["logPath"].(string))
	if strings.Contains(string(logData), tok) || strings.Count(string(logData), ";0;") != recipients {
		t.Errorf("napló:\n%s", logData)
	}

	// újra: már mindenki megkapta, nem küld duplán
	_, c = call(t, srv, app.token, "/api/postmark/check", req)
	plan = c["plan"].(map[string]any)
	if c["ok"] == true || int(plan["alreadySent"].(float64)) != recipients {
		t.Errorf("ismételt ellenőrzés: %v", plan)
	}
	// az éles napló nem törölhető
	if _, er := call(t, srv, app.token, "/api/postmark/log/clear", req); !strings.Contains(fmt.Sprint(er["error"]), "nem törölhető") {
		resp := fmt.Sprint(er["error"])
		t.Errorf("éles napló törlése: %s", resp)
	}

	// belső teszt: csak a tesztcímre, [TESZT] tárggyal, valódi leiratkozó link nélkül
	mu.Lock()
	sent = nil
	mu.Unlock()
	req = map[string]any{"mode": "belsoteszt", "campaign": "teszt-kampany", "only": only, "skipSync": true}
	_, st = call(t, srv, app.token, "/api/postmark/send", req)
	for i := 0; i < 200; i++ {
		_, s := call(t, srv, app.token, "/api/postmark/status", nil)
		if s["job"].(map[string]any)["running"] != true {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	if len(sent) != 2 {
		t.Errorf("belső teszt: %d levél", len(sent))
	}
	for _, m := range sent {
		b, _ := json.Marshal(m)
		if m["To"] != "teszt@energofish.hu" || !strings.HasPrefix(m["Subject"].(string), "[TESZT] ") || strings.Contains(string(b), "leiratkozas.html") {
			t.Errorf("tesztlevél: %v %v", m["To"], m["Subject"])
		}
	}
	mu.Unlock()

	// visszajelzések
	_, fb := call(t, srv, app.token, "/api/postmark/feedback", nil)
	if fb["hardBounce"].(float64) != 1 || fb["matched"].(float64) != 1 {
		t.Errorf("visszajelzések: %v", fb)
	}
	if b, _ := os.ReadFile(fb["path"].(string)); !strings.Contains(string(b), "bolt2@example.com") {
		t.Error("visszajelzés CSV")
	}
}
