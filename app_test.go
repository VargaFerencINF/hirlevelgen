package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
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

	// feltöltés (drag & drop útvonal) ékezetes névvel
	data, _ := os.ReadFile(xlsx)
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/excel/upload", bytes.NewReader(data))
	req.Header.Set("X-Token", app.token)
	req.Header.Set("X-Filename", "Partnerlista%20okt%C3%B3ber.xlsx")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("feltöltés: %v %v", err, res.StatusCode)
	}
	res.Body.Close()
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
