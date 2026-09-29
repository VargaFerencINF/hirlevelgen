package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

// OutputSettings a generálás mentett beállításai.
type OutputSettings struct {
	Dir         string `json:"dir"`
	EML         bool   `json:"eml"`
	From        string `json:"from"`
	FilePattern string `json:"filePattern"`
}

// ClientState a felület által szerkesztett állapot.
type ClientState struct {
	Content  h.Content      `json:"content"`
	Products []h.Product    `json:"products"`
	Template string         `json:"template"`
	Output   OutputSettings `json:"output"`
}

// Saved a lemezre mentett beállítások.
type Saved struct {
	Version   int         `json:"version"`
	State     ClientState `json:"state"`
	ExcelPath string      `json:"excelPath"`
	ExcelMod  time.Time   `json:"excelMod"`
}

const savedVersion = 1

// App a program állapota.
type App struct {
	mu         sync.Mutex
	token      string
	templates  []*h.Template
	defaults   h.Content
	defProds   []h.Product
	state      ClientState
	excel      *h.ExcelData
	excelIssue []h.Issue // Excel + partner megállapítások (Excel betöltéskor számolva)
	configPath string
	saveTimer  *time.Timer
	assets     fs.FS
	web        fs.FS
	demoXLSX   []byte
	mode       string // "webview" vagy "browser"
	hwnd       uintptr
	lastBeat   time.Time
	smoke      bool // füstteszt: az első betöltés után kilép
	inited     bool
	quit       chan struct{}
	quitOnce   sync.Once
}

// NewApp betölti a sablonokat, az alapértékeket és a mentett beállításokat.
func NewApp(configDir string) (*App, error) {
	tpls, err := h.LoadTemplates(templateFS, "sablonok")
	if err != nil {
		return nil, err
	}
	sample, err := fs.ReadFile(demoFS, "demo/tartalom-minta.json")
	if err != nil {
		return nil, err
	}
	defs, defProds, err := h.DefaultContent(sample)
	if err != nil {
		return nil, err
	}
	defs = defs.Normalize(defs)
	demo, _ := fs.ReadFile(demoFS, "demo/Energofish_partner_hirlevel_minta.xlsx")
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	a := &App{
		token: hex.EncodeToString(b), templates: tpls, defaults: defs, defProds: defProds,
		assets: mustSub(assetsFS, "assets"), web: mustSub(webFS, "web"), demoXLSX: demo,
		quit: make(chan struct{}), lastBeat: time.Now(),
	}
	a.state = ClientState{
		Content: cloneContent(defs), Products: append([]h.Product{}, defProds...), Template: "v4-partnerjelentes",
		Output: OutputSettings{Dir: defaultOutputDir(), FilePattern: h.DefaultFilePattern},
	}
	if configDir != "" {
		a.configPath = filepath.Join(configDir, "beallitasok.json")
		a.loadSaved()
	}
	return a, nil
}

func cloneContent(c h.Content) h.Content {
	out := make(h.Content, len(c))
	for k, v := range c {
		out[k] = v
	}
	return out
}

func (a *App) loadSaved() {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return
	}
	var s Saved
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("hibás beállításfájl: %v", err)
		return
	}
	if s.State.Content != nil {
		a.state.Content = s.State.Content.Normalize(a.defaults)
	}
	if s.State.Products != nil {
		a.state.Products = s.State.Products
	}
	if h.FindTemplate(a.templates, s.State.Template) != nil {
		a.state.Template = s.State.Template
	}
	if s.State.Output.Dir != "" {
		a.state.Output = s.State.Output
	}
	if a.state.Output.FilePattern == "" {
		a.state.Output.FilePattern = h.DefaultFilePattern
	}
	// az utoljára használt Excel újraolvasása; ha közben módosult, a termékeket is onnan vesszük
	if s.ExcelPath != "" {
		if st, err := os.Stat(s.ExcelPath); err == nil {
			changed := !st.ModTime().Equal(s.ExcelMod)
			if err := a.loadExcelPath(s.ExcelPath, changed || len(a.state.Products) == 0); err != nil {
				log.Printf("az Excel nem tölthető be újra: %v", err)
			}
		}
	}
}

// scheduleSave késleltetve menti a beállításokat (a hívó tartja a zárat).
func (a *App) scheduleSave() {
	if a.configPath == "" {
		return
	}
	if a.saveTimer != nil {
		a.saveTimer.Stop()
	}
	a.saveTimer = time.AfterFunc(600*time.Millisecond, a.saveNow)
}

func (a *App) saveNow() {
	a.mu.Lock()
	s := Saved{Version: savedVersion, State: a.state}
	if a.excel != nil {
		s.ExcelPath, s.ExcelMod = a.excel.Path, a.excel.ModTime
	}
	data, err := json.MarshalIndent(s, "", "  ")
	path := a.configPath
	a.mu.Unlock()
	if err != nil || path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err == nil {
		_ = os.Rename(tmp, path)
	}
}

func (a *App) template() *h.Template {
	if t := h.FindTemplate(a.templates, a.state.Template); t != nil {
		return t
	}
	return a.templates[len(a.templates)-1]
}

func (a *App) partners() []h.Partner {
	if a.excel == nil {
		return nil
	}
	return a.excel.Partners
}

// loadExcelPath beolvassa az Excelt lemezről (a hívó NEM tartja a zárat).
func (a *App) loadExcelPath(path string, takeProducts bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("a fájl nem olvasható (zárolt?): %s", filepath.Base(path))
		}
		return fmt.Errorf("a fájl nem nyitható meg: %w", err)
	}
	st, _ := os.Stat(path)
	var mod time.Time
	if st != nil {
		mod = st.ModTime()
	}
	return a.loadExcelData(data, path, mod, takeProducts)
}

func (a *App) loadExcelData(data []byte, path string, mod time.Time, takeProducts bool) error {
	ex, err := h.ReadExcel(data, path, mod)
	if err != nil {
		return err
	}
	issues := append([]h.Issue{}, ex.Issues...)
	issues = append(issues, h.ValidateColumns(ex, a.templates)...)
	issues = append(issues, h.ValidatePartners(ex.Partners)...)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.excel = ex
	a.excelIssue = issues
	if takeProducts && len(ex.Products) > 0 {
		a.state.Products = ex.Products
	}
	a.scheduleSave()
	return nil
}

// ---------------------------------------------------------------------------
// HTTP

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.Handle("/static/", http.StripPrefix("/static/", noCache(http.FileServer(http.FS(a.web)))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(a.assets))))
	mux.HandleFunc("/elonezet", a.handleBrowserPreview)

	api := map[string]func(w http.ResponseWriter, r *http.Request) (any, error){
		"/api/init":             a.apiInit,
		"/api/state":            a.apiState,
		"/api/issues":           a.apiIssues,
		"/api/excel/browse":     a.apiExcelBrowse,
		"/api/excel/load":       a.apiExcelLoad,
		"/api/excel/upload":     a.apiExcelUpload,
		"/api/excel/reload":     a.apiExcelReload,
		"/api/excel/products":   a.apiExcelProducts,
		"/api/excel/open":       a.apiExcelOpenExternal,
		"/api/excel/close":      a.apiExcelClose,
		"/api/demo/save":        a.apiDemoSave,
		"/api/images/check":     a.apiImagesCheck,
		"/api/output/browse":    a.apiOutputBrowse,
		"/api/generate":         a.apiGenerate,
		"/api/open":             a.apiOpen,
		"/api/openurl":          a.apiOpenURL,
		"/api/content/export":   a.apiContentExport,
		"/api/content/import":   a.apiContentImport,
		"/api/content/defaults": a.apiContentDefaults,
		"/api/heartbeat":        a.apiHeartbeat,
		"/api/quit":             a.apiQuit,
	}
	for path, fn := range api {
		fn := fn
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Token") != a.token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			res, err := fn(w, r)
			if res == nil && err == nil {
				return // a kezelő maga válaszolt
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(res)
		})
	}
	mux.HandleFunc("/api/preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Token") != a.token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		a.writePreview(w, r)
	})
	return mux
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(a.web, "index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	html := strings.Replace(string(data), "{{TOKEN}}", a.token, 1)
	html = strings.Replace(html, "{{VERSION}}", version, -1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, html)
}

func decode(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("üres kérés")
	}
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(v)
}

// ---------------------------------------------------------------------------
// Nézetek

type excelView struct {
	*h.ExcelData
	Reps int `json:"reps"`
}

func (a *App) excelViewLocked() *excelView {
	if a.excel == nil {
		return nil
	}
	reps := map[string]bool{}
	for _, p := range a.excel.Partners {
		if p.RepName != "" {
			reps[p.RepName] = true
		}
	}
	return &excelView{ExcelData: a.excel, Reps: len(reps)}
}

func (a *App) contentIssuesLocked() []h.Issue {
	tpl := a.template()
	out := h.ValidateContent(a.state.Content, a.state.Products, tpl)
	return append(out, h.ValidateProducts(a.state.Content, a.state.Products)...)
}

type issuesView struct {
	Content  []h.Issue      `json:"content"`
	Partners []h.Issue      `json:"partners"`
	Counts   map[string]int `json:"counts"`
	Blocked  int            `json:"blocked"`
}

func (a *App) issuesLocked() issuesView {
	content := a.contentIssuesLocked()
	all := append(append([]h.Issue{}, content...), a.excelIssue...)
	blocked := 0
	for i := range a.partners() {
		if h.PartnerBlocked(&a.excel.Partners[i]) {
			blocked++
		}
	}
	if content == nil {
		content = []h.Issue{}
	}
	partners := a.excelIssue
	if partners == nil {
		partners = []h.Issue{}
	}
	return issuesView{Content: content, Partners: partners, Counts: h.CountIssues(all), Blocked: blocked}
}

// ---------------------------------------------------------------------------
// API

func (a *App) apiInit(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.inited = true
	if a.smoke {
		log.Printf("füstteszt: a felület betöltődött (%s)", r.UserAgent())
		time.AfterFunc(3*time.Second, a.Quit)
	}
	tpls := make([]h.Template, 0, len(a.templates))
	for _, t := range a.templates {
		tpls = append(tpls, *t)
	}
	return map[string]any{
		"version":   version,
		"mode":      a.mode,
		"fields":    h.Fields,
		"groups":    h.Groups,
		"templates": tpls,
		"tokens":    h.Tokens,
		"state":     a.state,
		"defaults":  a.defaults,
		"excel":     a.excelViewLocked(),
		"issues":    a.issuesLocked(),
		"sample":    h.SamplePartner,
		"config":    a.configPath,
	}, nil
}

func (a *App) apiState(w http.ResponseWriter, r *http.Request) (any, error) {
	var s ClientState
	if err := decode(r, &s); err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.Content != nil {
		a.state.Content = s.Content.Normalize(a.defaults)
	}
	if s.Products != nil {
		a.state.Products = s.Products
	}
	if h.FindTemplate(a.templates, s.Template) != nil {
		a.state.Template = h.FindTemplate(a.templates, s.Template).ID
	}
	a.state.Output = s.Output
	a.scheduleSave()
	return map[string]any{"issues": a.issuesLocked()}, nil
}

func (a *App) apiIssues(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.issuesLocked(), nil
}

func (a *App) excelResponse() (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]any{"excel": a.excelViewLocked(), "products": a.state.Products, "issues": a.issuesLocked()}, nil
}

func (a *App) apiExcelBrowse(w http.ResponseWriter, r *http.Request) (any, error) {
	dir := ""
	a.mu.Lock()
	if a.excel != nil && a.excel.Path != "" {
		dir = filepath.Dir(a.excel.Path)
	}
	a.mu.Unlock()
	path, err := a.pickFile("Partnerlista (Excel) kiválasztása", dir, []fileFilter{{"Excel munkafüzet", []string{"*.xlsx", "*.xlsm"}}})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return map[string]any{"cancelled": true}, nil
	}
	if err := a.loadExcelPath(path, true); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

func (a *App) apiExcelLoad(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if err := a.loadExcelPath(req.Path, true); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

func (a *App) apiExcelUpload(w http.ResponseWriter, r *http.Request) (any, error) {
	name := r.Header.Get("X-Filename")
	if n, err := url.QueryUnescape(name); err == nil {
		name = n
	}
	name = filepath.Base(name)
	if name == "" {
		name = "feltoltott.xlsx"
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	// ha a behúzott fájl mellett tudjuk az útvonalát (böngészőben nem), csak a nevét tároljuk
	if err := a.loadExcelData(data, name, time.Now(), true); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

func (a *App) apiExcelReload(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	path := ""
	if a.excel != nil {
		path = a.excel.Path
	}
	a.mu.Unlock()
	if path == "" || !filepath.IsAbs(path) {
		return nil, errors.New("nincs újratölthető Excel: a behúzott fájlt húzd be újra, vagy válaszd ki a Tallózás gombbal")
	}
	if err := a.loadExcelPath(path, true); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

func (a *App) apiExcelProducts(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.excel == nil || len(a.excel.Products) == 0 {
		return nil, errors.New("a betöltött Excelben nincs terméklista")
	}
	a.state.Products = append([]h.Product{}, a.excel.Products...)
	a.scheduleSave()
	return map[string]any{"products": a.state.Products, "issues": a.issuesLocked()}, nil
}

func (a *App) apiExcelOpenExternal(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	path := ""
	if a.excel != nil {
		path = a.excel.Path
	}
	a.mu.Unlock()
	if path == "" || !filepath.IsAbs(path) {
		return nil, errors.New("a fájl helye nem ismert")
	}
	return map[string]any{"ok": true}, openPath(path)
}

func (a *App) apiExcelClose(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.excel, a.excelIssue = nil, nil
	a.scheduleSave()
	return map[string]any{"issues": a.issuesLocked()}, nil
}

func (a *App) apiDemoSave(w http.ResponseWriter, r *http.Request) (any, error) {
	path, err := a.pickSave("Minta Excel mentése", documentsDir(), "Energofish_partner_hirlevel_minta.xlsx",
		[]fileFilter{{"Excel munkafüzet", []string{"*.xlsx"}}})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return map[string]any{"cancelled": true}, nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		path += ".xlsx"
	}
	if err := os.WriteFile(path, a.demoXLSX, 0o644); err != nil {
		return nil, fmt.Errorf("a mentés nem sikerült: %w", err)
	}
	if err := a.loadExcelPath(path, true); err != nil {
		return nil, err
	}
	return a.excelResponse()
}

func (a *App) apiImagesCheck(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	targets := h.CollectImageTargets(a.state.Content, a.state.Products, a.partners(), a.template())
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	res := h.CheckImages(ctx, targets)
	sort.SliceStable(res, func(i, j int) bool { return rank(res[i].Status) < rank(res[j].Status) })
	return map[string]any{"results": res}, nil
}

func rank(s string) int {
	switch s {
	case "error":
		return 0
	case "warn":
		return 1
	case "ok":
		return 2
	}
	return 3
}

func (a *App) apiOutputBrowse(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	dir := a.state.Output.Dir
	a.mu.Unlock()
	path, err := a.pickFolder("Kimeneti mappa kiválasztása", dir)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return map[string]any{"cancelled": true}, nil
	}
	a.mu.Lock()
	a.state.Output.Dir = path
	a.scheduleSave()
	a.mu.Unlock()
	return map[string]any{"dir": path}, nil
}

func (a *App) apiGenerate(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Only []int `json:"only"`
	}
	_ = decode(r, &req)
	a.mu.Lock()
	if a.excel == nil || len(a.excel.Partners) == 0 {
		a.mu.Unlock()
		return nil, errors.New("nincs betöltött partnerlista (Excel)")
	}
	tpl := a.template()
	content := cloneContent(a.state.Content)
	products := append([]h.Product{}, a.state.Products...)
	partners := append([]h.Partner{}, a.excel.Partners...)
	out := a.state.Output
	a.mu.Unlock()
	if len(req.Only) == 0 {
		return nil, errors.New("nincs kiválasztott partner")
	}
	return h.Generate(a.assets, tpl, content, products, partners, h.GenerateOptions{
		OutputDir: out.Dir, EML: out.EML, From: out.From, FilePattern: out.FilePattern, Only: req.Only,
	})
}

func (a *App) apiOpen(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Path   string `json:"path"`
		Reveal bool   `json:"reveal"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if req.Path == "" {
		return nil, errors.New("nincs megadva útvonal")
	}
	if _, err := os.Stat(req.Path); err != nil {
		if req.Reveal {
			_ = os.MkdirAll(req.Path, 0o755)
		} else {
			return nil, fmt.Errorf("nem található: %s", req.Path)
		}
	}
	return map[string]any{"ok": true}, openPath(req.Path)
}

func (a *App) apiOpenURL(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		URL string `json:"url"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	u, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "mailto" && u.Scheme != "tel") {
		return nil, errors.New("csak http(s), mailto: vagy tel: link nyitható meg")
	}
	return map[string]any{"ok": true}, openURL(u.String())
}

func (a *App) apiContentExport(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	data, err := h.ExportJSON(a.state.Content, a.state.Products)
	name := "hirlevel-tartalom_" + time.Now().Format("2006-01-02") + ".json"
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	path, err := a.pickSave("Tartalom mentése", documentsDir(), name, []fileFilter{{"Tartalom (JSON)", []string{"*.json"}}})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return map[string]any{"cancelled": true}, nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": path}, nil
}

func (a *App) apiContentImport(w http.ResponseWriter, r *http.Request) (any, error) {
	path, err := a.pickFile("Tartalom betöltése", documentsDir(), []fileFilter{{"Tartalom (JSON)", []string{"*.json"}}})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return map[string]any{"cancelled": true}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c, products, err := h.ImportJSON(data)
	if err != nil {
		return nil, err
	}
	if len(c) == 0 {
		return nil, errors.New("a fájlban nincs felismerhető hírlevél-tartalom")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, v := range c {
		a.state.Content[k] = v
	}
	if len(products) > 0 {
		a.state.Products = products
	}
	a.scheduleSave()
	return map[string]any{"state": a.state, "issues": a.issuesLocked(), "path": path}, nil
}

func (a *App) apiContentDefaults(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Content = cloneContent(a.defaults)
	a.scheduleSave()
	return map[string]any{"state": a.state, "issues": a.issuesLocked()}, nil
}

func (a *App) apiHeartbeat(w http.ResponseWriter, r *http.Request) (any, error) {
	a.mu.Lock()
	a.lastBeat = time.Now()
	a.mu.Unlock()
	return map[string]any{"ok": true}, nil
}

func (a *App) apiQuit(w http.ResponseWriter, r *http.Request) (any, error) {
	go func() {
		time.Sleep(200 * time.Millisecond)
		a.Quit()
	}()
	return map[string]any{"ok": true}, nil
}

// Quit leállítja a programot.
func (a *App) Quit() { a.quitOnce.Do(func() { close(a.quit) }) }

// ---------------------------------------------------------------------------
// Előnézet

func (a *App) renderPreview(pIdx int, tplID string, assetsBase string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	tpl := h.FindTemplate(a.templates, tplID)
	if tpl == nil {
		tpl = a.template()
	}
	p := h.SamplePartner
	if parts := a.partners(); pIdx >= 0 && pIdx < len(parts) {
		p = parts[pIdx]
	}
	d := h.Build(a.state.Content, a.state.Products, &p, tpl, assetsBase)
	html, missing := tpl.Render(d)
	if len(missing) > 0 {
		return "", fmt.Errorf("ismeretlen mezők: %s", strings.Join(missing, ", "))
	}
	return html, nil
}

func (a *App) writePreview(w http.ResponseWriter, r *http.Request) {
	p, err := strconv.Atoi(r.URL.Query().Get("p"))
	if err != nil {
		p = -1
	}
	html, err := a.renderPreview(p, r.URL.Query().Get("tpl"), "/assets")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, html)
}

// handleBrowserPreview a rendszer-böngészőben megnyitható előnézet (a token a címben).
func (a *App) handleBrowserPreview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("t") != a.token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	a.writePreview(w, r)
}
