package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"

	h "energofish/hirlevel/internal/hirlevel"
)

// Képtár feltöltése FTP-re. A beállítások a beállítások mappájában: ftp.json (titok nélkül) és
// ftp-jelszo.dat (a jelszó, Windowson DPAPI-val titkosítva). A jelszó soha nem megy ki a
// felületre, naplóba vagy hibaüzenetbe.

// FTPSettings az FTP-képtár beállításai.
type FTPSettings struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	TLS        string `json:"tls"`        // auto (FTPS, ha a szerver tudja), explicit (csak FTPS), none (titkosítás nélkül)
	Verify     bool   `json:"verify"`     // a TLS-tanúsítvány ellenőrzése
	RemoteDir  string `json:"remoteDir"`  // a képtár mappája a szerveren, pl. /public_html/hirlevel/assets
	PublicURL  string `json:"publicUrl"`  // ugyanez a weben, pl. https://energofish.hu/hirlevel/assets
	LocalDir   string `json:"localDir"`   // saját képek mappája (opcionális)
	Overwrite  bool   `json:"overwrite"`  // a fent lévő, de eltérő méretű fájl felülírása
	SetBase    bool   `json:"setBase"`    // feltöltés után a képtár webcíme = PublicURL
	AutoUpload bool   `json:"autoUpload"` // a küldés előtti ellenőrzésnél automatikus feltöltés
}

func defaultFTPSettings() FTPSettings {
	return FTPSettings{Port: 21, TLS: "auto", Verify: true, SetBase: true, AutoUpload: true}
}

const ftpMagic = "EFFTP1\n"

func (a *App) loadFTPSettings() FTPSettings {
	s := defaultFTPSettings()
	if p := a.postmarkFile("ftp.json"); p != "" {
		if data, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(data, &s); err != nil {
				log.Printf("ftp.json hibás, alapértékek: %v", err)
				s = defaultFTPSettings()
			}
		}
	}
	if s.Port <= 0 || s.Port > 65535 {
		s.Port = 21
	}
	if s.TLS == "" {
		s.TLS = "auto"
	}
	return s
}

func (a *App) saveFTPSettings(s FTPSettings) error {
	p := a.postmarkFile("ftp.json")
	if p == "" {
		return nil
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (a *App) loadFTPPassword() string {
	p := a.postmarkFile("ftp-jelszo.dat")
	if p == "" {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	if bytes.HasPrefix(data, []byte(ftpMagic)) {
		if unprotectSecret == nil {
			return ""
		}
		if data, err = unprotectSecret(data[len(ftpMagic):]); err != nil {
			log.Printf("az FTP jelszó nem fejthető vissza ezen a gépen/felhasználóval")
			return ""
		}
	}
	return string(data)
}

func (a *App) saveFTPPassword(pw string) error {
	p := a.postmarkFile("ftp-jelszo.dat")
	if p == "" {
		return nil
	}
	if pw == "" {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data := []byte(pw)
	if protectSecret != nil {
		enc, err := protectSecret(data)
		if err != nil {
			return fmt.Errorf("a jelszó titkosítása nem sikerült: %v", err)
		}
		data = append([]byte(ftpMagic), enc...)
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s FTPSettings) configured() bool {
	return strings.TrimSpace(s.Host) != "" && strings.TrimSpace(s.User) != "" && strings.TrimSpace(s.RemoteDir) != ""
}

func (a *App) ftpView() map[string]any {
	s := a.loadFTPSettings()
	return map[string]any{"settings": s, "password": a.loadFTPPassword() != "", "configured": s.configured(), "protected": secretsProtected}
}

func (a *App) apiFTPSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	return a.ftpView(), nil
}

func normalizeFTP(s FTPSettings) (FTPSettings, error) {
	s.Host = strings.TrimSpace(s.Host)
	// „ftp://host:port/mappa” alak is megadható
	if u := strings.TrimPrefix(strings.TrimPrefix(s.Host, "ftp://"), "ftps://"); u != s.Host {
		host, rest, _ := strings.Cut(u, "/")
		s.Host = host
		if rest != "" && strings.TrimSpace(s.RemoteDir) == "" {
			s.RemoteDir = "/" + rest
		}
	}
	if hp, port, err := net.SplitHostPort(s.Host); err == nil {
		s.Host = hp
		if n, err := strconv.Atoi(port); err == nil {
			s.Port = n
		}
	}
	if s.Port <= 0 || s.Port > 65535 {
		s.Port = 21
	}
	s.User = strings.TrimSpace(s.User)
	s.RemoteDir = strings.TrimSpace(strings.ReplaceAll(s.RemoteDir, "\\", "/"))
	if s.RemoteDir != "" {
		s.RemoteDir = "/" + strings.Trim(path.Clean("/"+s.RemoteDir), "/")
	}
	s.PublicURL = strings.TrimRight(strings.TrimSpace(s.PublicURL), "/")
	if s.PublicURL != "" && !strings.HasPrefix(s.PublicURL, "https://") && !strings.HasPrefix(s.PublicURL, "http://") {
		return s, errors.New("a képtár webcíme https:// (vagy http://) kezdetű legyen, pl. https://energofish.hu/hirlevel/assets")
	}
	s.LocalDir = strings.TrimSpace(s.LocalDir)
	switch s.TLS {
	case "auto", "explicit", "none":
	default:
		s.TLS = "auto"
	}
	return s, nil
}

func (a *App) apiFTPSettingsSave(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Settings      FTPSettings `json:"settings"`
		Password      string      `json:"password"`
		ClearPassword bool        `json:"clearPassword"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	s, err := normalizeFTP(req.Settings)
	if err != nil {
		return nil, err
	}
	if err := a.saveFTPSettings(s); err != nil {
		return nil, err
	}
	if req.ClearPassword {
		if err := a.saveFTPPassword(""); err != nil {
			return nil, err
		}
	} else if req.Password != "" {
		if err := a.saveFTPPassword(req.Password); err != nil {
			return nil, err
		}
	}
	return a.ftpView(), nil
}

// apiFTPFolder a saját képek mappájának kiválasztása.
func (a *App) apiFTPFolder(w http.ResponseWriter, r *http.Request) (any, error) {
	s := a.loadFTPSettings()
	dir := s.LocalDir
	if dir == "" {
		dir = documentsDir()
	}
	p, err := a.pickFolder("A saját képek mappája", dir)
	if err != nil {
		return nil, err
	}
	if p == "" {
		return map[string]any{"cancelled": true}, nil
	}
	s.LocalDir = p
	if err := a.saveFTPSettings(s); err != nil {
		return nil, err
	}
	return a.ftpView(), nil
}

// ftpRemote az FTP-kapcsolat a feltöltéshez (h.RemoteFS).
type ftpRemote struct{ c *ftp.ServerConn }

func (f ftpRemote) List(dir string) (map[string]int64, error) {
	entries, err := f.c.List(dir)
	if err != nil {
		var te *textproto.Error
		// nem létező mappa: a szerverek eltérő kódot adnak (550, 501, 450); a létrehozás úgyis
		// pontos hibát ad, ha valójában jogosultsági gond van
		if errors.As(err, &te) && (te.Code == 550 || te.Code == 501 || te.Code == 450 || te.Code == 553) {
			return map[string]int64{}, nil
		}
		return nil, ftpError(err)
	}
	out := map[string]int64{}
	for _, e := range entries {
		switch e.Type {
		case ftp.EntryTypeFolder:
			if e.Name != "." && e.Name != ".." {
				out[path.Base(e.Name)+"/"] = -1
			}
		default:
			out[path.Base(e.Name)] = int64(e.Size)
		}
	}
	return out, nil
}

func (f ftpRemote) MakeDir(dir string) error { return ftpError(f.c.MakeDir(dir)) }

func (f ftpRemote) Stor(file string, r io.Reader) error { return ftpError(f.c.Stor(file, r)) }

// ftpError az FTP-hiba magyarul.
func ftpError(err error) error {
	if err == nil {
		return nil
	}
	var te *textproto.Error
	if errors.As(err, &te) {
		hu := map[int]string{
			421: "a szerver bontotta a kapcsolatot (túl sok kapcsolat vagy időtúllépés)",
			425: "az adatkapcsolat nem jött létre (tűzfal / passzív mód)",
			426: "az átvitel megszakadt",
			430: "hibás felhasználónév vagy jelszó",
			450: "a fájl most nem érhető el",
			451: "a szerver helyi hibája",
			452: "nincs elég hely a szerveren",
			500: "a szerver nem ismeri a parancsot",
			502: "a szerver nem támogatja a parancsot",
			530: "hibás felhasználónév vagy jelszó (nincs bejelentkezve)",
			532: "a feltöltéshez külön jogosultság kell",
			550: "nincs jogosultság, vagy nem létező fájl / mappa",
			552: "betelt a tárhely",
			553: "a fájlnév nem engedélyezett",
		}[te.Code]
		if hu == "" {
			hu = "FTP-hiba"
		}
		return fmt.Errorf("%s (%d: %s)", hu, te.Code, strings.TrimSpace(te.Msg))
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return errors.New("időtúllépés – a szerver nem válaszol (rossz cím/port, vagy tűzfal)")
	}
	var tlsErr *tls.CertificateVerificationError
	if errors.As(err, &tlsErr) || strings.Contains(err.Error(), "x509:") {
		return errors.New("a szerver TLS-tanúsítványa nem ellenőrizhető (pl. IP-címmel csatlakozva, vagy saját aláírású tanúsítvány) – kapcsold ki a „Tanúsítvány ellenőrzése” beállítást, ha megbízol a szerverben")
	}
	return errors.New(h.SafeNetErr(err))
}

// ftpConnect csatlakozik és bejelentkezik. Automatikus módban előbb titkosítva (FTPS) próbál,
// és ha a szerver nem támogatja, titkosítás nélkül. Visszaadja, hogy titkosított-e a kapcsolat.
func (a *App) ftpConnect(ctx context.Context, s FTPSettings, pw string) (*ftp.ServerConn, bool, error) {
	if !s.configured() {
		return nil, false, errors.New("az FTP nincs beállítva (szerver, felhasználó, mappa)")
	}
	if pw == "" {
		return nil, false, errors.New("nincs megadva az FTP jelszó")
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	dial := func(useTLS bool) (*ftp.ServerConn, error) {
		opts := []ftp.DialOption{ftp.DialWithContext(ctx), ftp.DialWithTimeout(30 * time.Second)}
		if useTLS {
			opts = append(opts, ftp.DialWithExplicitTLS(&tls.Config{ServerName: s.Host, InsecureSkipVerify: !s.Verify,
				ClientSessionCache: tls.NewLRUClientSessionCache(8), MinVersion: tls.VersionTLS12}))
		}
		c, err := ftp.Dial(addr, opts...)
		if err != nil {
			return nil, err
		}
		if err := c.Login(s.User, pw); err != nil {
			_ = c.Quit()
			return nil, err
		}
		return c, nil
	}
	switch s.TLS {
	case "none":
		c, err := dial(false)
		return c, false, ftpError(err)
	case "explicit":
		c, err := dial(true)
		return c, true, ftpError(err)
	}
	c, err := dial(true)
	if err == nil {
		return c, true, nil
	}
	// a szerver nem tud titkosítást (AUTH TLS) → titkosítás nélkül
	var te *textproto.Error
	if errors.As(err, &te) && (te.Code == 500 || te.Code == 502 || te.Code == 504 || te.Code == 534 || te.Code == 431) {
		c, err := dial(false)
		return c, false, ftpError(err)
	}
	return nil, false, ftpError(err)
}

// apiFTPTest kapcsolatpróba: bejelentkezés és a képtár mappájának listázása.
func (a *App) apiFTPTest(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Settings *FTPSettings `json:"settings"`
		Password string       `json:"password"`
	}
	_ = decode(r, &req)
	s := a.loadFTPSettings()
	if req.Settings != nil {
		var err error
		if s, err = normalizeFTP(*req.Settings); err != nil {
			return nil, err
		}
	}
	pw := req.Password
	if pw == "" {
		pw = a.loadFTPPassword()
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	c, secure, err := a.ftpConnect(ctx, s, pw)
	if err != nil {
		return nil, err
	}
	defer c.Quit()
	files, err := ftpRemote{c}.List(s.RemoteDir)
	if err != nil {
		return nil, err
	}
	n := 0
	for k := range files {
		if !strings.HasSuffix(k, "/") {
			n++
		}
	}
	return map[string]any{"ok": true, "secure": secure, "files": n}, nil
}

// uploadAssets összegyűjti és feltölti a képtárat; ha kérve van, a képtár webcímét is beállítja.
func (a *App) uploadAssets(ctx context.Context) (map[string]any, error) {
	s := a.loadFTPSettings()
	pw := a.loadFTPPassword()
	var dirs []string
	a.mu.Lock()
	for _, t := range a.tpls() {
		if t.AssetsDir != "" {
			dirs = append(dirs, t.AssetsDir)
		}
	}
	a.mu.Unlock()
	files, err := h.CollectAssets(a.assets, dirs, s.LocalDir)
	if err != nil {
		return nil, err
	}
	c, secure, err := a.ftpConnect(ctx, s, pw)
	if err != nil {
		return nil, err
	}
	defer c.Quit()
	start := time.Now()
	res, err := h.SyncAssets(ftpRemote{c}, s.RemoteDir, files, s.Overwrite, nil)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"result": res, "files": len(files), "secure": secure, "millis": time.Since(start).Milliseconds()}
	log.Printf("képtár feltöltése FTP-re: %d fájl, %d új, %d felülírt, %d már fent volt, %d eltérő (kihagyva), %d hiba",
		len(files), len(res.Uploaded), len(res.Replaced), len(res.Skipped), len(res.Changed), len(res.Failed))
	if s.PublicURL != "" {
		// a webcím egyezik-e a feltöltési mappával: az egyik feltöltött (vagy fent lévő) képet lekérjük
		probe := ""
		for _, f := range files {
			if !strings.Contains(f.Rel, "/") {
				probe = f.Rel
				break
			}
		}
		if probe != "" {
			out["probe"] = probeURL(ctx, s.PublicURL+"/"+probe)
			out["probeUrl"] = s.PublicURL + "/" + probe
		}
		if s.SetBase && len(res.Failed) == 0 {
			a.mu.Lock()
			if strings.TrimRight(strings.TrimSpace(a.state.Content["assets.base"]), "/") != s.PublicURL {
				a.state.Content["assets.base"] = s.PublicURL
				a.scheduleSave()
				out["assetsBaseSet"] = true
			}
			a.mu.Unlock()
			out["assetsBase"] = s.PublicURL
		}
	}
	return out, nil
}

// probeURL egy kép elérhetősége a weben („” = rendben, egyébként a hiba).
func probeURL(ctx context.Context, u string) string {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "hibás webcím"
	}
	req.Header.Set("Range", "bytes=0-0")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "nem érhető el: " + h.SafeNetErr(err)
	}
	res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Sprintf("a webszerver válasza HTTP %d – a webcím valószínűleg nem a feltöltési mappára mutat", res.StatusCode)
	}
	return ""
}

func (a *App) apiFTPUpload(w http.ResponseWriter, r *http.Request) (any, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	return a.uploadAssets(ctx)
}
