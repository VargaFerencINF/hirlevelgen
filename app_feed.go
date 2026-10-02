package main

import (
	"errors"
	"net/http"
	"strings"

	h "energofish/hirlevel/internal/hirlevel"
)

// A cikktörzs (termékfeed) kezelése: frissítés, keresés, átvétel a termékek közé.

func (a *App) feedURL() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return strings.TrimSpace(a.state.Feed.URL)
}

func (a *App) apiFeedStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	return a.feed.Status(), nil
}

func (a *App) apiFeedRefresh(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Force bool   `json:"force"`
		URL   string `json:"url"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		url = a.feedURL()
	}
	if url != "" && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, errors.New("a cikktörzs címe http:// vagy https:// kezdetű legyen")
	}
	a.feed.Refresh(url, req.Force)
	return a.feed.Status(), nil
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
	list := a.feed.Products()
	found := h.SearchFeed(list, req.Q, req.Limit+1)
	more := len(found) > req.Limit
	if more {
		found = found[:req.Limit]
	}
	a.mu.Lock()
	have := map[string]bool{}
	for _, p := range a.state.Products {
		have[codeKey(p.Code)] = true
	}
	a.mu.Unlock()
	res := make([]feedResult, len(found))
	for i, p := range found {
		res[i] = feedResult{FeedProduct: p, Added: have[codeKey(p.Code)]}
	}
	return map[string]any{"status": a.feed.Status(), "results": res, "more": more}, nil
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
	list := a.feed.Products()
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
	list := a.feed.Products()
	out := map[string][]h.FeedImage{}
	for _, c := range req.Codes {
		if p := h.FindFeed(list, c); p != nil && len(p.Images) > 0 {
			out[c] = p.Images
		}
	}
	return map[string]any{"ready": len(list) > 0, "images": out}, nil
}
