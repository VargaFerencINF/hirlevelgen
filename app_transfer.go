package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	h "energofish/hirlevel/internal/hirlevel"
)

// Beállítások átvitele másik gépre (pl. a területi képviselő gépére). Egyetlen, jelszóval
// titkosított fájlba (.efbeall) kerül minden, ami a program használatához kell:
//
//   - beallitasok.json (tartalom, termékek, sablonválasztás, cikktörzs-címek, partnerhalmazok);
//   - postmark.json, ftp.json és a titkos adatok: a Postmark-tokenek, az FTP-jelszó és a
//     partnertörzs-források (tokenes linkek);
//   - a hozzáadott sablonok (sablonok/ mappa) és a saját képek (köztük a képviselőfotók);
//   - a helyi leiratkozólista (leiratkozottak.txt).
//
// Nem kerül bele: a letöltött partnertörzs (valódi partneradatok; a forrásokból újra letöltődik),
// a cikktörzs-gyorsítótár, a naplók és az időzített küldés (különben két gép küldené ki).
//
// Titkosítás: a jelszóból PBKDF2-HMAC-SHA256 (600 000 kör, 16 bájtos véletlen só) képez
// 256 bites kulcsot, a tartalom AES-256-GCM-mel titkosított (a fejléc hitelesített adat), így
// a jelszó nélkül olvashatatlan, és bármilyen módosítás észrevehető. A célgépen a titkos adatok
// újra a gép saját titkosításával (Windowson DPAPI) kerülnek mentésre. A jelszó és a titkos
// adatok soha nem kerülnek naplóba, hibaüzenetbe vagy a felületre.

const (
	transferMagic   = "EFBEALL1\n"
	transferExt     = ".efbeall"
	transferIter    = 600_000
	transferMinPass = 8
	transferMaxSize = 512 << 20 // a titkosított fájl legnagyobb mérete
	transferMaxImgs = 200 << 20 // a saját képek legfeljebb ennyi bájtig kerülnek bele
	transferPending = 10 * time.Minute
)

// transferManifest a csomag tartalomjegyzéke (titkos adat nélkül).
type transferManifest struct {
	Format     int       `json:"format"`
	AppVersion string    `json:"appVersion"`
	Created    time.Time `json:"created"`
	Computer   string    `json:"computer,omitempty"`
}

// transferSecrets a titkos adatok (a csomagon belül is csak titkosítva léteznek).
type transferSecrets struct {
	Postmark    postmarkTokens    `json:"postmark"`
	FTPPassword string            `json:"ftpPassword,omitempty"`
	Sources     map[string]string `json:"sources,omitempty"`
}

// transferPayload a visszafejtett csomag.
type transferPayload struct {
	Manifest  transferManifest
	Settings  []byte // beallitasok.json (lehet üres)
	Postmark  *PostmarkSettings
	FTP       *FTPSettings
	Secrets   transferSecrets
	Unsubs    []string
	Templates map[string][]byte // sablonok/… relatív út → tartalom
	Images    map[string][]byte // saját képek: fájlnév → tartalom
}

// ---------------------------------------------------------------------------
// Titkosítás

func transferKey(pass string, salt []byte, iter int) ([]byte, error) {
	return pbkdf2.Key(sha256.New, pass, salt, iter, 32)
}

func encryptTransfer(plain []byte, pass string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := transferKey(pass, salt, transferIter)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	head := make([]byte, 0, len(transferMagic)+16+4)
	head = append(head, transferMagic...)
	head = append(head, salt...)
	head = binary.BigEndian.AppendUint32(head, transferIter)
	out := append(append([]byte{}, head...), nonce...)
	return gcm.Seal(out, nonce, plain, head), nil
}

var errTransferAuth = errors.New("hibás jelszó, vagy a fájl sérült / módosult")

func decryptTransfer(data []byte, pass string) ([]byte, error) {
	if !bytes.HasPrefix(data, []byte(transferMagic)) {
		return nil, errors.New("ez nem a program beállításfájlja (.efbeall)")
	}
	hl := len(transferMagic) + 16 + 4
	if len(data) < hl+12+16 {
		return nil, errors.New("a beállításfájl csonka")
	}
	head := data[:hl]
	salt := head[len(transferMagic) : len(transferMagic)+16]
	iter := int(binary.BigEndian.Uint32(head[len(transferMagic)+16:]))
	if iter < 100_000 || iter > 10_000_000 {
		return nil, errors.New("a beállításfájl fejléce hibás")
	}
	key, err := transferKey(pass, salt, iter)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := data[hl : hl+gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, data[hl+gcm.NonceSize():], head)
	if err != nil {
		return nil, errTransferAuth
	}
	return plain, nil
}

func checkTransferPass(pass string) error {
	if utf8.RuneCountInString(pass) < transferMinPass {
		return fmt.Errorf("a jelszó legalább %d karakter legyen", transferMinPass)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Csomag összeállítása

// localImagesDir a saját képek mappája (az FTP-beállításé, különben a beállítások sajat-kepek mappája).
func (a *App) localImagesDir(s FTPSettings) string {
	if d := strings.TrimSpace(s.LocalDir); d != "" {
		return d
	}
	if a.configDir == "" {
		return ""
	}
	return filepath.Join(a.configDir, "sajat-kepek")
}

// buildTransfer összeállítja a csomagot (zip, titkosítás előtt) és az összesítőt.
func (a *App) buildTransfer() ([]byte, map[string]any, error) {
	a.saveNow() // a legutóbbi szerkesztés is kerüljön bele
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	addJSON := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return add(name, b)
	}
	host, _ := os.Hostname()
	if err := addJSON("manifest.json", transferManifest{Format: 1, AppVersion: version, Created: time.Now(), Computer: host}); err != nil {
		return nil, nil, err
	}
	sum := map[string]any{}
	if a.configPath != "" {
		if b, err := os.ReadFile(a.configPath); err == nil {
			if err := add("beallitasok.json", b); err != nil {
				return nil, nil, err
			}
			sum["settings"] = true
		}
	}
	pm, ftpS := a.loadPostmarkSettings(), a.loadFTPSettings()
	if err := addJSON("postmark.json", pm); err != nil {
		return nil, nil, err
	}
	if err := addJSON("ftp.json", ftpS); err != nil {
		return nil, nil, err
	}
	sec := transferSecrets{Postmark: a.loadPostmarkTokens(), FTPPassword: a.loadFTPPassword(), Sources: a.loadSources()}
	if err := addJSON("titkok.json", sec); err != nil {
		return nil, nil, err
	}
	unsubs := sortedKeys(a.unsubList())
	if len(unsubs) > 0 {
		if err := add("leiratkozottak.txt", []byte(strings.Join(unsubs, "\n")+"\n")); err != nil {
			return nil, nil, err
		}
	}
	// hozzáadott sablonok (a félbemaradt „.uj” mappák nélkül)
	nTpl := 0
	if a.configDir != "" {
		root := filepath.Join(a.configDir, "sablonok")
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			rel, _ := filepath.Rel(root, p)
			if d.IsDir() {
				if strings.HasSuffix(d.Name(), ".uj") {
					return filepath.SkipDir
				}
				if rel != "." && !strings.Contains(rel, string(filepath.Separator)) {
					nTpl++
				}
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return add("sablonok/"+filepath.ToSlash(rel), b)
		})
		if err != nil {
			return nil, nil, fmt.Errorf("a hozzáadott sablonok nem olvashatók: %v", err)
		}
	}
	// saját képek (a mappa közvetlen képei)
	var imgs []string
	var skipped []string
	if dir := a.localImagesDir(ftpS); dir != "" {
		entries, _ := os.ReadDir(dir)
		total := 0
		for _, e := range entries {
			if e.IsDir() || !h.IsImageName(e.Name()) {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			if total+len(b) > transferMaxImgs {
				skipped = append(skipped, e.Name())
				continue
			}
			total += len(b)
			if err := add("kepek/"+e.Name(), b); err != nil {
				return nil, nil, err
			}
			imgs = append(imgs, e.Name())
		}
	}
	if err := zw.Close(); err != nil {
		return nil, nil, err
	}
	sum["postmarkLive"], sum["postmarkSandbox"] = sec.Postmark.Live != "", sec.Postmark.Sandbox != ""
	sum["ftp"] = ftpS.Host != ""
	sum["ftpPassword"] = sec.FTPPassword != ""
	sum["sources"] = sourceNames(sec.Sources)
	sum["templates"], sum["images"], sum["unsubs"] = nTpl, len(imgs), len(unsubs)
	if len(skipped) > 0 {
		sum["skippedImages"] = skipped
	}
	return buf.Bytes(), sum, nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sourceNames a partnerforrások csoportnevei (a linkek nélkül).
func sourceNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		if strings.TrimSpace(v) != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// parseTransfer a visszafejtett zip beolvasása.
func parseTransfer(plain []byte) (*transferPayload, error) {
	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		return nil, errors.New("a beállításfájl tartalma sérült")
	}
	p := &transferPayload{Templates: map[string][]byte{}, Images: map[string][]byte{}}
	read := func(f *zip.File) ([]byte, error) {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, transferMaxSize))
	}
	for _, f := range zr.File {
		name := f.Name
		// csak a várt, biztonságos utak (nincs „..”, abszolút út vagy meghajtó)
		if strings.Contains(name, "..") || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, ":") {
			continue
		}
		b, err := read(f)
		if err != nil {
			return nil, errors.New("a beállításfájl tartalma sérült")
		}
		switch {
		case name == "manifest.json":
			_ = json.Unmarshal(b, &p.Manifest)
		case name == "beallitasok.json":
			p.Settings = b
		case name == "postmark.json":
			var s PostmarkSettings
			if json.Unmarshal(b, &s) == nil {
				p.Postmark = &s
			}
		case name == "ftp.json":
			var s FTPSettings
			if json.Unmarshal(b, &s) == nil {
				p.FTP = &s
			}
		case name == "titkok.json":
			_ = json.Unmarshal(b, &p.Secrets)
		case name == "leiratkozottak.txt":
			for _, l := range strings.Split(string(b), "\n") {
				if l = strings.ToLower(strings.TrimSpace(l)); l != "" {
					p.Unsubs = append(p.Unsubs, l)
				}
			}
		case strings.HasPrefix(name, "sablonok/") && !strings.HasSuffix(name, "/"):
			p.Templates[strings.TrimPrefix(name, "sablonok/")] = b
		case strings.HasPrefix(name, "kepek/") && !strings.Contains(strings.TrimPrefix(name, "kepek/"), "/"):
			if n := strings.TrimPrefix(name, "kepek/"); h.IsImageName(n) {
				p.Images[n] = b
			}
		}
	}
	if p.Manifest.Format != 1 {
		return nil, errors.New("a beállításfájl ismeretlen formátumú (talán egy újabb programverzió készítette)")
	}
	if p.Settings != nil {
		var s Saved
		if err := json.Unmarshal(p.Settings, &s); err != nil {
			return nil, errors.New("a beállításfájlban a beállítások hibásak")
		}
	}
	return p, nil
}

// summary a csomag összesítője a felületnek (titkos adat nélkül).
func (p *transferPayload) summary() map[string]any {
	tpls := map[string]bool{}
	for k := range p.Templates {
		if i := strings.Index(k, "/"); i > 0 {
			tpls[k[:i]] = true
		}
	}
	var tplNames []string
	for id := range tpls {
		name := id
		var m struct {
			Name string `json:"name"`
		}
		if b, ok := p.Templates[id+"/meta.json"]; ok && json.Unmarshal(b, &m) == nil && m.Name != "" {
			name = m.Name
		}
		tplNames = append(tplNames, name)
	}
	sort.Strings(tplNames)
	out := map[string]any{
		"created": p.Manifest.Created, "appVersion": p.Manifest.AppVersion, "computer": p.Manifest.Computer,
		"settings":     p.Settings != nil,
		"postmarkLive": p.Secrets.Postmark.Live != "", "postmarkSandbox": p.Secrets.Postmark.Sandbox != "",
		"ftp": p.FTP != nil && p.FTP.Host != "", "ftpPassword": p.Secrets.FTPPassword != "",
		"sources": sourceNames(p.Secrets.Sources), "templates": tplNames, "images": len(p.Images), "unsubs": len(p.Unsubs),
	}
	if p.Postmark != nil {
		out["from"] = p.Postmark.From
	}
	if p.FTP != nil && p.FTP.Host != "" {
		out["ftpTarget"] = p.FTP.User + "@" + p.FTP.Host
	}
	return out
}

// ---------------------------------------------------------------------------
// Visszaállítás a célgépen

func existsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// backupForTransfer a felülírandó fájlokról mentést készít (mentes-atvitel-…/), és visszaadja a mappát.
func (a *App) backupForTransfer(templateIDs []string) string {
	if a.configDir == "" {
		return ""
	}
	dir := filepath.Join(a.configDir, "mentes-atvitel-"+time.Now().Format("20060102-150405"))
	copied := 0
	for _, n := range []string{"beallitasok.json", "postmark.json", "ftp.json", "postmark-tokenek.dat", "ftp-jelszo.dat", "partnerforrasok.dat", "leiratkozottak.txt"} {
		b, err := os.ReadFile(filepath.Join(a.configDir, n))
		if err != nil {
			continue
		}
		if os.MkdirAll(dir, 0o700) == nil && os.WriteFile(filepath.Join(dir, n), b, 0o600) == nil {
			copied++
		}
	}
	for _, id := range templateIDs {
		src := filepath.Join(a.configDir, "sablonok", id)
		if !existsDir(src) {
			continue
		}
		if os.MkdirAll(filepath.Join(dir, "sablonok"), 0o700) == nil && os.Rename(src, filepath.Join(dir, "sablonok", id)) == nil {
			copied++
		}
	}
	if copied == 0 {
		return ""
	}
	return dir
}

// applyTransfer a csomag tartalmát a program beállításaivá teszi, és újratölti őket.
func (a *App) applyTransfer(p *transferPayload) (map[string]any, error) {
	if a.configDir == "" {
		return nil, errors.New("nincs beállítási mappa, ahova a beállítások menthetők")
	}
	var notes []string
	// a felülírt sablonmappák azonosítói (csak a csomagban lévők, a gépen lévő többi marad)
	tplIDs := map[string]bool{}
	for k := range p.Templates {
		if i := strings.Index(k, "/"); i > 0 {
			tplIDs[k[:i]] = true
		}
	}
	backup := a.backupForTransfer(sortedKeys(tplIDs))

	// beállítások: a gépfüggő mappák, ha itt nem léteznek, alapértékre állnak
	if p.Settings != nil {
		var s Saved
		_ = json.Unmarshal(p.Settings, &s)
		if s.State.Output.Dir != "" && !existsDir(s.State.Output.Dir) {
			notes = append(notes, fmt.Sprintf("A kimeneti mappa (%s) ezen a gépen nem létezik, helyette: %s.", s.State.Output.Dir, defaultOutputDir()))
			s.State.Output.Dir = defaultOutputDir()
		}
		if s.Import.ExcelDir != "" && !existsDir(s.Import.ExcelDir) {
			notes = append(notes, fmt.Sprintf("Az import Excel mappája (%s) ezen a gépen nem létezik, helyette a program mappája.", s.Import.ExcelDir))
			s.Import.ExcelDir = ""
		}
		if s.ExcelPath != "" {
			if _, err := os.Stat(s.ExcelPath); err != nil {
				s.ExcelPath, s.ExcelMod = "", time.Time{}
			}
		}
		b, _ := json.MarshalIndent(s, "", "  ")
		if err := writeFileAtomic(a.configPath, b); err != nil {
			return nil, fmt.Errorf("a beállítások nem menthetők: %v", err)
		}
	}
	// saját képek: a csomag szerinti mappába, ha itt is létezik, különben a sajat-kepek mappába
	ftpS := a.loadFTPSettings()
	if p.FTP != nil {
		ftpS = *p.FTP
	}
	if len(p.Images) > 0 {
		dir := strings.TrimSpace(ftpS.LocalDir)
		if dir == "" || !existsDir(dir) {
			if dir != "" {
				notes = append(notes, fmt.Sprintf("A saját képek mappája (%s) ezen a gépen nem létezik, a képek ide kerültek: %s.", dir, filepath.Join(a.configDir, "sajat-kepek")))
			}
			dir = filepath.Join(a.configDir, "sajat-kepek")
			ftpS.LocalDir = dir
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("a saját képek mappája nem hozható létre: %v", err)
		}
		for n, b := range p.Images {
			if err := os.WriteFile(filepath.Join(dir, n), b, 0o644); err != nil {
				return nil, fmt.Errorf("a(z) %s kép nem menthető: %v", n, err)
			}
		}
	} else if ftpS.LocalDir != "" && !existsDir(ftpS.LocalDir) {
		ftpS.LocalDir = ""
	}
	if p.FTP != nil || len(p.Images) > 0 {
		if err := a.saveFTPSettings(ftpS); err != nil {
			return nil, fmt.Errorf("az FTP-beállítások nem menthetők: %v", err)
		}
	}
	if p.Postmark != nil {
		if err := a.savePostmarkSettings(*p.Postmark); err != nil {
			return nil, fmt.Errorf("a Postmark-beállítások nem menthetők: %v", err)
		}
	}
	// titkos adatok: a gép saját titkosításával
	if p.Secrets.Postmark.Live != "" || p.Secrets.Postmark.Sandbox != "" {
		if err := a.savePostmarkTokens(p.Secrets.Postmark); err != nil {
			return nil, err
		}
	}
	if p.Secrets.FTPPassword != "" {
		if err := a.saveFTPPassword(p.Secrets.FTPPassword); err != nil {
			return nil, err
		}
	}
	if len(p.Secrets.Sources) > 0 {
		merged := a.loadSources()
		for k, v := range p.Secrets.Sources {
			if strings.TrimSpace(v) != "" {
				merged[k] = v
			}
		}
		if err := a.saveSources(merged); err != nil {
			return nil, err
		}
	}
	// leiratkozók: összefésülés (egy leiratkozás sem veszhet el)
	if len(p.Unsubs) > 0 {
		all := a.unsubList()
		for _, e := range p.Unsubs {
			all[e] = true
		}
		if err := writeFileAtomic(a.postmarkFile("leiratkozottak.txt"), []byte(strings.Join(sortedKeys(all), "\n")+"\n")); err != nil {
			return nil, fmt.Errorf("a leiratkozólista nem menthető: %v", err)
		}
	}
	// hozzáadott sablonok
	root := filepath.Join(a.configDir, "sablonok")
	for rel, b := range p.Templates {
		dst := filepath.Join(root, filepath.FromSlash(rel))
		if !strings.HasPrefix(dst, root+string(filepath.Separator)) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return nil, fmt.Errorf("a sablon nem menthető: %v", err)
		}
	}
	if err := a.reloadAfterTransfer(); err != nil {
		return nil, err
	}
	log.Printf("beállítások átvéve (%s, %s géptől); mentés: %s", p.Manifest.Created.Format("2006-01-02 15:04"), p.Manifest.Computer, backup)
	return map[string]any{"ok": true, "backup": backup, "notes": notes}, nil
}

// reloadAfterTransfer a lemezre írt beállítások újratöltése (mint egy újraindításkor).
func (a *App) reloadAfterTransfer() error {
	tpls, err := h.LoadTemplates(templateFS, "sablonok")
	if err != nil {
		return err
	}
	lib, errs := h.OpenLibrary(filepath.Join(a.configDir, "sablonok"), tpls, a.assets)
	for _, e := range errs {
		log.Printf("egyedi sablon kihagyva: %v", e)
	}
	a.mu.Lock()
	if a.saveTimer != nil {
		a.saveTimer.Stop()
	}
	a.lib = lib
	a.state = a.defaultState()
	a.excel, a.excelIssue = nil, nil
	a.imp, a.b2bSet = ImportSettings{}, B2BSettings{}
	a.mu.Unlock()
	a.loadSaved() // (a hívó nem tarthatja a zárat: az Excel és a partnerhalmaz betöltése maga zárol)
	return nil
}

// ---------------------------------------------------------------------------
// API

// függőben lévő import: a visszafejtett csomag a megerősítésig a memóriában marad
var transferPendingMu sync.Mutex
var transferPendingMap = map[string]pendingTransfer{}

type pendingTransfer struct {
	p   *transferPayload
	exp time.Time
}

func (a *App) apiTransferExport(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Password string `json:"password"`
		Path     string `json:"path"` // tesztekhez; különben mentési ablak
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if err := checkTransferPass(req.Password); err != nil {
		return nil, err
	}
	dst := req.Path
	if dst == "" {
		name := "Energofish-hirlevel-beallitasok-" + time.Now().Format("2006-01-02") + transferExt
		var err error
		if dst, err = a.pickSave("Beállítások exportálása", documentsDir(), name, []fileFilter{{"Energofish beállítások", []string{"*" + transferExt}}}); err != nil {
			return nil, err
		}
		if dst == "" {
			return map[string]any{"cancelled": true}, nil
		}
	}
	if !strings.EqualFold(filepath.Ext(dst), transferExt) {
		dst += transferExt
	}
	plain, sum, err := a.buildTransfer()
	if err != nil {
		return nil, err
	}
	enc, err := encryptTransfer(plain, req.Password)
	if err != nil {
		return nil, fmt.Errorf("a titkosítás nem sikerült: %v", err)
	}
	if err := writeFileAtomic(dst, enc); err != nil {
		return nil, fmt.Errorf("a fájl nem menthető: %v", err)
	}
	log.Printf("beállítások exportálva: %s (%d KB)", dst, len(enc)/1024)
	return map[string]any{"ok": true, "path": dst, "size": len(enc), "summary": sum}, nil
}

func (a *App) apiTransferInspect(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Password string `json:"password"`
		Path     string `json:"path"` // ha üres: fájlválasztó ablak
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if req.Password == "" {
		return nil, errors.New("add meg a fájl jelszavát")
	}
	src := req.Path
	if src == "" {
		var err error
		if src, err = a.pickFile("Beállításfájl megnyitása", documentsDir(), []fileFilter{{"Energofish beállítások", []string{"*" + transferExt}}}); err != nil {
			return nil, err
		}
		if src == "" {
			return map[string]any{"cancelled": true}, nil
		}
	}
	st, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("a fájl nem olvasható: %s", filepath.Base(src))
	}
	if st.Size() > transferMaxSize {
		return nil, errors.New("a fájl túl nagy, nem a program beállításfájlja")
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("a fájl nem olvasható: %s", filepath.Base(src))
	}
	plain, err := decryptTransfer(data, req.Password)
	if err != nil {
		return nil, err
	}
	p, err := parseTransfer(plain)
	if err != nil {
		return nil, err
	}
	idb := make([]byte, 12)
	_, _ = rand.Read(idb)
	id := hex.EncodeToString(idb)
	transferPendingMu.Lock()
	for k, v := range transferPendingMap {
		if time.Now().After(v.exp) {
			delete(transferPendingMap, k)
		}
	}
	transferPendingMap[id] = pendingTransfer{p: p, exp: time.Now().Add(transferPending)}
	transferPendingMu.Unlock()
	return map[string]any{"id": id, "file": path.Base(filepath.ToSlash(src)), "summary": p.summary()}, nil
}

func (a *App) apiTransferApply(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	transferPendingMu.Lock()
	pt, ok := transferPendingMap[req.ID]
	delete(transferPendingMap, req.ID)
	transferPendingMu.Unlock()
	if !ok || time.Now().After(pt.exp) {
		return nil, errors.New("a megnyitott beállításfájl lejárt – nyisd meg újra")
	}
	return a.applyTransfer(pt.p)
}
