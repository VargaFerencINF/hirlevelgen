// Energofish Partnerhírlevél-generátor
//
// Excel-partnerlistából és a programban megadott közös tartalomból partnerenként
// legenerálja a hírlevelet a tervezett sablonok (v1, v2, v4) valamelyikével.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	h "energofish/hirlevel/internal/hirlevel"
)

var version = "1.0.0"

const appTitle = "Energofish Partnerhírlevél-generátor"

func main() {
	var (
		port     = flag.Int("port", 0, "a helyi szerver portja (0 = automatikus)")
		browser  = flag.Bool("bongeszo", false, "saját ablak helyett a böngészőben nyílik meg")
		noOpen   = flag.Bool("nem-nyit", false, "nem nyit ablakot, csak kiírja a címet (fejlesztéshez)")
		debug    = flag.Bool("debug", false, "fejlesztői eszközök engedélyezése")
		cfgDir   = flag.String("beallitasok", "", "a beállítások mappája (alapból a felhasználói AppData)")
		batch    = flag.Bool("batch", false, "felület nélküli generálás (lásd -excel, -sablon, -kimenet)")
		excel    = flag.String("excel", "", "batch: a partnerlista (xlsx)")
		content  = flag.String("tartalom", "", "batch: tartalom JSON (üresen a mentett beállítások)")
		tplFlag  = flag.String("sablon", "", "batch: v1, v2 vagy v4 (üresen a mentett)")
		outDir   = flag.String("kimenet", "", "batch: kimeneti mappa")
		eml      = flag.Bool("eml", false, "batch: EML fájlok is")
		from     = flag.String("felado", "", "batch: feladó az EML-hez")
		selftest = flag.Bool("onteszt", false, "beépített önteszt (CI)")
		smoke    = flag.Bool("fustteszt", false, "ablak-teszt: megnyitja a felületet, majd kilép (CI)")
	)
	flag.Parse()

	configDir := *cfgDir
	if configDir == "" {
		if d, err := os.UserConfigDir(); err == nil {
			configDir = filepath.Join(d, "EnergofishHirlevel")
		}
	}
	setupLog(configDir, *batch || *selftest || *noOpen)

	if *selftest {
		if err := runSelfTest(); err != nil {
			log.Printf("ÖNTESZT HIBA: %v", err)
			os.Exit(1)
		}
		log.Printf("önteszt rendben")
		return
	}

	app, err := NewApp(configDir)
	if err != nil {
		fatal(err)
	}

	if *batch {
		if err := runBatch(app, *excel, *content, *tplFlag, *outDir, *eml, *from); err != nil {
			log.Printf("HIBA: %v", err)
			os.Exit(1)
		}
		return
	}

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(*port))
	if err != nil {
		fatal(fmt.Errorf("a helyi szerver nem indítható: %w", err))
	}
	addr := ln.Addr().(*net.TCPAddr)
	srv := &http.Server{Handler: hostGuard(addr.Port, app.routes()), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("szerver: %v", err)
		}
	}()
	url := fmt.Sprintf("http://127.0.0.1:%d/", addr.Port)
	log.Printf("%s %s – %s", appTitle, version, url)

	if *smoke {
		app.smoke = true
		time.AfterFunc(90*time.Second, func() {
			log.Printf("FÜSTTESZT HIBA: a felület nem töltődött be")
			os.Exit(2)
		})
	}
	if !*browser && !*noOpen && runWindow(app, url, *debug) {
		app.saveNow()
		if *smoke {
			app.mu.Lock()
			ok := app.inited
			app.mu.Unlock()
			if !ok {
				log.Printf("FÜSTTESZT HIBA: az ablak bezárult, a felület nem töltődött be")
				os.Exit(3)
			}
			log.Printf("füstteszt rendben (%s)", app.mode)
		}
		return
	}
	if *smoke {
		log.Printf("FÜSTTESZT HIBA: a WebView2 ablak nem nyílt meg")
		os.Exit(4)
	}

	app.mu.Lock()
	app.mode = "browser"
	app.lastBeat = time.Now()
	app.mu.Unlock()
	if *noOpen {
		fmt.Println(url)
	} else if err := openURL(url); err != nil {
		fatal(fmt.Errorf("a böngésző nem nyitható meg: %w (cím: %s)", err, url))
	}
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-app.quit:
			app.saveNow()
			return
		case <-tick.C:
			app.mu.Lock()
			idle := time.Since(app.lastBeat)
			app.mu.Unlock()
			if !*noOpen && idle > 4*time.Minute {
				log.Printf("a böngészőlap bezárult, kilépés")
				app.saveNow()
				return
			}
		}
	}
}

// hostGuard csak a 127.0.0.1/localhost címre érkező kéréseket engedi (DNS-rebinding ellen).
func hostGuard(port int, next http.Handler) http.Handler {
	ok := map[string]bool{
		"127.0.0.1:" + strconv.Itoa(port): true,
		"localhost:" + strconv.Itoa(port): true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ok[r.Host] {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setupLog(dir string, console bool) {
	log.SetFlags(log.LstdFlags)
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "naplo.txt")
	if st, err := os.Stat(path); err == nil && st.Size() > 1<<20 {
		_ = os.Rename(path, path+".regi")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	if console {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	} else {
		log.SetOutput(f)
	}
}

func fatal(err error) {
	log.Printf("végzetes hiba: %v", err)
	showError(appTitle, err.Error())
	os.Exit(1)
}

// runBatch felület nélküli generálás (pl. ütemezett feladatból).
func runBatch(app *App, excelPath, contentPath, tpl, out string, eml bool, from string) error {
	if excelPath == "" {
		return errors.New("add meg az -excel kapcsolót")
	}
	if err := app.loadExcelPath(excelPath, true); err != nil {
		return err
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	if contentPath != "" {
		data, err := os.ReadFile(contentPath)
		if err != nil {
			return err
		}
		c, products, err := h.ImportJSON(data)
		if err != nil {
			return err
		}
		for k, v := range c {
			app.state.Content[k] = v
		}
		if len(products) > 0 {
			app.state.Products = products
		}
	}
	if tpl != "" {
		t := h.FindTemplate(app.templates, tpl)
		if t == nil {
			return fmt.Errorf("ismeretlen sablon: %s (v1, v2 vagy v4)", tpl)
		}
		app.state.Template = t.ID
	}
	if out == "" {
		out = app.state.Output.Dir
	}
	for _, is := range app.excelIssue {
		if is.Level == h.LevelError && is.Scope == h.ScopeExcel {
			return errors.New(is.Message)
		}
	}
	res, err := h.Generate(app.assets, app.template(), app.state.Content, app.state.Products, app.excel.Partners,
		h.GenerateOptions{OutputDir: out, EML: eml, From: from, FilePattern: app.state.Output.FilePattern})
	if err != nil {
		return err
	}
	log.Printf("kész: %d hírlevél, %d kihagyva → %s", res.Generated, len(res.Skipped), res.Folder)
	for _, s := range res.Skipped {
		log.Printf("  kihagyva: %d. sor %s (%s)", s.Row, s.Email, s.Reason)
	}
	return nil
}

// runSelfTest a beágyazott demóadatokkal mindhárom sablont legenerálja.
func runSelfTest() error {
	dir, err := os.MkdirTemp("", "energofish-onteszt")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	app, err := NewApp("")
	if err != nil {
		return err
	}
	data, err := fs.ReadFile(demoFS, "demo/Energofish_partner_hirlevel_minta.xlsx")
	if err != nil {
		return err
	}
	if err := app.loadExcelData(data, filepath.Join(dir, "minta.xlsx"), time.Now(), true); err != nil {
		return err
	}
	if n := len(app.excel.Partners); n != 21 {
		return fmt.Errorf("%d partner (21 várt)", n)
	}
	for _, t := range app.templates {
		res, err := h.Generate(app.assets, t, app.state.Content, app.state.Products, app.excel.Partners,
			h.GenerateOptions{OutputDir: dir, EML: true})
		if err != nil {
			return fmt.Errorf("%s: %w", t.ID, err)
		}
		if res.Generated != 21 {
			return fmt.Errorf("%s: %d hírlevél", t.ID, res.Generated)
		}
		b, err := os.ReadFile(res.FirstHTML)
		if err != nil || !strings.Contains(string(b), "Kedves Kovács Péter!") {
			return fmt.Errorf("%s: hibás kimenet", t.ID)
		}
		log.Printf("%s: %d hírlevél, %d ms", t.ID, res.Generated, res.Millis)
	}
	return nil
}
