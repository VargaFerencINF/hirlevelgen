package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	h "energofish/hirlevel/internal/hirlevel"
)

// A cikktörzs (termékfeed) kezelése: frissítés, keresés, átvétel a termékek közé.

// feedStore az ország cikktörzse (első kéréskor jön létre, saját gyorsítótár-mappával).
func (a *App) feedStore(code string) *h.FeedStore {
	code = h.FindFeedCountry(code).Code
	a.feedMu.Lock()
	defer a.feedMu.Unlock()
	if st := a.feeds[code]; st != nil {
		return st
	}
	dir := ""
	if a.feedDir != "" {
		dir = filepath.Join(a.feedDir, code)
	}
	st := h.NewFeedStore(dir)
	st.Country = code
	a.feeds[code] = st
	return st
}

// autoFeedCountryLocked a betöltött partnerhalmaz célcsoportjának cikktörzse (Excelnél magyar).
func (a *App) autoFeedCountryLocked() string {
	if a.excel != nil && a.excel.B2B != nil {
		return h.FeedCountryForGroup(a.excel.B2B.Group)
	}
	return "hu"
}

// activeFeedLocked az aktív cikktörzs és a címe (a hívó tartja a.mu-t; "" = az ország
// alapértelmezett címe). Saját cím országonként adható meg (Beállítások › Cikktörzs).
func (a *App) activeFeedLocked() (*h.FeedStore, string) {
	code := strings.TrimSpace(a.state.Feed.Country)
	if code == "" {
		code = a.autoFeedCountryLocked()
	}
	st := a.feedStore(code)
	return st, a.state.Feed.URLFor(st.Country)
}

func (a *App) activeFeed() (*h.FeedStore, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.activeFeedLocked()
}

// feedStatusLocked az aktív cikktörzs állapota (a hívó tartja a.mu-t), kiegészítve az ország
// adataival: pénznem-jel, nettó-felirat, az automatikus (partnerhalmaz szerinti) ország.
func (a *App) feedStatusLocked() map[string]any {
	st, _ := a.activeFeedLocked()
	c := h.FindFeedCountry(st.Country)
	cur := c.Currency
	if v := strings.TrimSpace(a.state.Feed.Currency[c.Code]); v != "" {
		cur = v
	}
	auto := h.FindFeedCountry(a.autoFeedCountryLocked())
	out := map[string]any{}
	b, _ := json.Marshal(st.Status())
	_ = json.Unmarshal(b, &out)
	out["currency"], out["netSuffix"], out["auto"], out["autoLabel"] = cur, c.NetSuffix, auto.Code, auto.Label
	out["manual"] = strings.TrimSpace(a.state.Feed.Country) != ""
	return out
}

// feedStatusView ugyanez zárolással.
func (a *App) feedStatusView() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.feedStatusLocked()
}

func (a *App) apiFeedStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	return a.feedStatusView(), nil
}

func (a *App) apiFeedRefresh(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Force bool   `json:"force"`
		URL   string `json:"url"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	st, url := a.activeFeed()
	if u := strings.TrimSpace(req.URL); u != "" {
		url = u // kézzel megadott cím (az aktív ország cikktörzséhez)
	}
	if url != "" && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, errors.New("a cikktörzs címe http:// vagy https:// kezdetű legyen")
	}
	st.Refresh(url, req.Force)
	return a.feedStatusView(), nil
}

// feedResult a keresési találat a felületnek (a cikkadatok + hogy szerepel-e már).
type feedResult struct {
	h.FeedProduct
	Added bool `json:"added"`
}

func (a *App) apiFeedSearch(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Q     string `json:"q"`
		Limit int    `json:"limit"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > 200 {
		req.Limit = 60
	}
	st, _ := a.activeFeed()
	list := st.Products()
	found := h.SearchFeed(list, req.Q, req.Limit+1)
	more := len(found) > req.Limit
	if more {
		found = found[:req.Limit]
	}
	a.mu.Lock()
	have := map[string]bool{}
	for _, p := range a.state.Products {
		have[codeKey(p.Code)] = true
		if p.Feed != nil { // főtermékre váltott cikk: a saját cikkszámával is szerepel
			have[codeKey(p.Feed.Code)] = true
		}
	}
	a.mu.Unlock()
	res := make([]feedResult, len(found))
	for i, p := range found {
		res[i] = feedResult{FeedProduct: p, Added: have[codeKey(p.Code)]}
	}
	return map[string]any{"status": a.feedStatusView(), "results": res, "more": more}, nil
}

func codeKey(code string) string { return strings.ReplaceAll(h.Norm(code), "-", "") }

// apiFeedProducts a kiválasztott cikkeket hírlevél-termékké alakítja (a felület adja hozzá a listához).
func (a *App) apiFeedProducts(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Codes   []string      `json:"codes"`
		Options h.FeedOptions `json:"options"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	st, _ := a.activeFeed()
	list := st.Products()
	if len(list) == 0 {
		return nil, errors.New("a cikktörzs még nincs betöltve")
	}
	out := []h.Product{}
	missing := []string{}
	for _, c := range req.Codes {
		if p := h.FindFeed(list, c); p != nil {
			out = append(out, p.ToProduct(req.Options))
		} else {
			missing = append(missing, c)
		}
	}
	return map[string]any{"products": out, "missing": missing}, nil
}

// apiFeedImages a megadott cikkszámok képváltozatai (az Excelből jött termékek képváltójához).
func (a *App) apiFeedImages(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Codes []string `json:"codes"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	st, _ := a.activeFeed()
	list := st.Products()
	out := map[string][]h.FeedImage{}
	refs := map[string]*h.FeedRef{}
	for _, c := range req.Codes {
		if p := h.FindFeed(list, c); p != nil {
			refs[c] = p.Ref()
			if len(p.Images) > 0 {
				out[c] = p.Images
			}
		}
	}
	return map[string]any{"ready": len(list) > 0, "images": out, "refs": refs}, nil
}
