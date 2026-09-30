package main

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	h "energofish/hirlevel/internal/hirlevel"
)

// Egyedi sablonok kezelése: hozzáadás (.html vagy a tervező .zip csomagja), átnevezés, törlés.

func (a *App) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := path.Base(strings.TrimPrefix(r.URL.Path, "/assets/"))
	a.mu.Lock()
	data, ok := a.lib.AssetFile(name)
	a.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	if ct := mime.TypeByExtension(strings.ToLower(path.Ext(name))); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (a *App) templatesLocked() []h.Template {
	list := a.tpls()
	out := make([]h.Template, 0, len(list))
	for _, t := range list {
		out = append(out, *t)
	}
	return out
}

func (a *App) templatesResponse(results []h.ImportResult) map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	if h.FindTemplate(a.tpls(), a.state.Template) == nil {
		a.state.Template = a.template().ID
		a.scheduleSave()
	}
	if results == nil {
		results = []h.ImportResult{}
	}
	return map[string]any{"templates": a.templatesLocked(), "results": results, "template": a.state.Template, "issues": a.issuesLocked()}
}

// htmlSiblingAssets a HTML-fájl melletti képeket keresi (ugyanott, assets/ vagy ../assets/ mappában).
func htmlSiblingAssets(htmlPath string) func(string) ([]byte, bool) {
	dir := filepath.Dir(htmlPath)
	return func(name string) ([]byte, bool) {
		for _, p := range []string{filepath.Join(dir, "assets", name), filepath.Join(dir, name), filepath.Join(filepath.Dir(dir), "assets", name)} {
			if b, err := os.ReadFile(p); err == nil {
				return b, true
			}
		}
		return nil, false
	}
}

func (a *App) importPaths(paths []string) ([]h.ImportResult, error) {
	var all []h.ImportResult
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			all = append(all, h.ImportResult{File: filepath.Base(p), Error: err.Error()})
			continue
		}
		a.mu.Lock()
		res, err := a.lib.ImportFile(p, data, htmlSiblingAssets(p))
		a.mu.Unlock()
		if err != nil {
			all = append(all, h.ImportResult{File: filepath.Base(p), Error: err.Error()})
			continue
		}
		all = append(all, res...)
	}
	return all, nil
}

func (a *App) apiTemplatesImport(w http.ResponseWriter, r *http.Request) (any, error) {
	paths, err := a.pickFiles("Sablon hozzáadása (.html vagy a tervező .zip csomagja)", documentsDir(),
		[]fileFilter{{"Hírlevélsablon", []string{"*.html", "*.htm", "*.zip"}}})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return map[string]any{"cancelled": true}, nil
	}
	res, err := a.importPaths(paths)
	if err != nil {
		return nil, err
	}
	return a.templatesResponse(res), nil
}

func (a *App) apiTemplatesUpload(w http.ResponseWriter, r *http.Request) (any, error) {
	name := r.Header.Get("X-Filename")
	if n, err := url.QueryUnescape(name); err == nil {
		name = n
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	res, err := a.lib.ImportFile(filepath.Base(name), data, nil)
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return a.templatesResponse(res), nil
}

func (a *App) apiTemplatesUpdate(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct{ ID, Name, Desc string }
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	a.mu.Lock()
	err := a.lib.Update(req.ID, req.Name, req.Desc)
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return a.templatesResponse(nil), nil
}

func (a *App) apiTemplatesDelete(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct{ ID string }
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	a.mu.Lock()
	err := a.lib.Delete(req.ID)
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return a.templatesResponse(nil), nil
}

func (a *App) apiTemplatesFolder(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.lib.Dir == "" {
		return nil, errors.New("nincs beállítási mappa")
	}
	_ = os.MkdirAll(a.lib.Dir, 0o755)
	return map[string]any{"ok": true}, openPath(a.lib.Dir)
}
