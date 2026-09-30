package hirlevel

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Library a beépített és a felhasználó által hozzáadott sablonok tára.
// Egyedi sablon: <Dir>/<id>/sablon.html (a tervezői, nyers változat), meta.json és assets/.
type Library struct {
	Dir      string
	assets   fs.FS // a beépített képtár
	builtins []*Template
	custom   []*Template
}

type customMeta struct {
	ID     string `json:"id"`
	Short  string `json:"short"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Source string `json:"source"`
	Added  string `json:"added"`
}

// ImportResult egy importált fájl eredménye.
type ImportResult struct {
	File     string   `json:"file"`
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name,omitempty"`
	OK       bool     `json:"ok"`
	Replaced bool     `json:"replaced,omitempty"` // meglévő egyedi sablont frissített
	Override bool     `json:"override,omitempty"` // beépített sablont frissít
	Error    string   `json:"error,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

// OpenLibrary betölti az egyedi sablonokat (a hibás mappákat kihagyja).
func OpenLibrary(dir string, builtins []*Template, assets fs.FS) (*Library, []error) {
	l := &Library{Dir: dir, builtins: builtins, assets: assets}
	if dir == "" {
		return l, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return l, nil
	}
	var errs []error
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := l.loadCustom(filepath.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
			continue
		}
		l.custom = append(l.custom, t)
	}
	l.sortCustom()
	return l, errs
}

func (l *Library) loadCustom(d string) (*Template, error) {
	raw, err := os.ReadFile(filepath.Join(d, "sablon.html"))
	if err != nil {
		return nil, err
	}
	var m customMeta
	if b, err := os.ReadFile(filepath.Join(d, "meta.json")); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	if m.ID == "" {
		m.ID = filepath.Base(d)
	}
	t, err := NewTemplate(m.ID, string(raw))
	if err != nil {
		return nil, err
	}
	t.Short, t.Name, t.Desc, t.Source, t.Added = m.Short, m.Name, m.Desc, m.Source, m.Added
	t.Custom = true
	if b := l.builtin(m.ID); b != nil {
		t.Overrides = true
		if t.Short == "" {
			t.Short = b.Short
		}
	}
	if t.Name == "" {
		t.Name = m.ID
	}
	if st, err := os.Stat(filepath.Join(d, "assets")); err == nil && st.IsDir() {
		t.AssetsDir = filepath.Join(d, "assets")
	}
	t.Report = append(t.Report, l.missingAssets(t)...)
	return t, nil
}

func (l *Library) sortCustom() {
	sort.SliceStable(l.custom, func(i, j int) bool { return l.custom[i].Added < l.custom[j].Added })
}

func (l *Library) builtin(id string) *Template {
	for _, b := range l.builtins {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// All a használható sablonok: a beépítettek (a frissített változattal), majd az egyediek.
func (l *Library) All() []*Template {
	out := make([]*Template, 0, len(l.builtins)+len(l.custom))
	for _, b := range l.builtins {
		t := b
		for _, c := range l.custom {
			if c.ID == b.ID {
				t = c
			}
		}
		out = append(out, t)
	}
	for _, c := range l.custom {
		if !c.Overrides {
			out = append(out, c)
		}
	}
	return out
}

// AssetFile a képtár egy fájlja: előbb az egyedi sablonok saját képei, majd a beépítettek.
func (l *Library) AssetFile(name string) ([]byte, bool) {
	name = path.Base(name)
	for _, c := range l.custom {
		if c.AssetsDir == "" {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(c.AssetsDir, name)); err == nil {
			return b, true
		}
	}
	if l.assets != nil {
		if b, err := fs.ReadFile(l.assets, name); err == nil {
			return b, true
		}
	}
	return nil, false
}

func (l *Library) hasBuiltinAsset(name string) bool {
	if l.assets == nil {
		return false
	}
	_, err := fs.Stat(l.assets, name)
	return err == nil
}

func (l *Library) missingAssets(t *Template) []string {
	var miss []string
	for _, a := range t.AssetRefs {
		if l.hasBuiltinAsset(a) {
			continue
		}
		if t.AssetsDir != "" {
			if _, err := os.Stat(filepath.Join(t.AssetsDir, a)); err == nil {
				continue
			}
		}
		miss = append(miss, a)
	}
	if len(miss) == 0 {
		return nil
	}
	return []string{"Hiányzó képek a képtárból: " + strings.Join(miss, ", ") + ". Tedd a sablon mellé egy assets mappába, és add hozzá újra a Tallózás gombbal (vagy a képekkel együtt ZIP-ben)."}
}

// ---------------------------------------------------------------------------
// Import

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug fájlnévből azonosító („V5 Tavaszi akció.html” → „v5-tavaszi-akcio”).
func Slug(name string) string {
	s := strings.ToLower(accentMap.Replace(name))
	s = strings.Trim(slugRe.ReplaceAllString(s, "-"), "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		s = "sablon"
	}
	return s
}

// baseName a fájlnév útvonal nélkül (Windows- és ZIP-útvonalnál is).
func baseName(p string) string { return path.Base(strings.ReplaceAll(p, "\\", "/")) }

var shortRe = regexp.MustCompile(`^[vV]\d{1,3}$`)

// prettyName a fájlnévből olvasható nevet és (ha van) rövid jelet készít.
func prettyName(file string) (name, short string) {
	file = baseName(file)
	base := strings.TrimSuffix(file, path.Ext(file))
	parts := strings.FieldsFunc(base, func(r rune) bool { return r == '-' || r == '_' || r == ' ' || r == '.' })
	if len(parts) > 0 && shortRe.MatchString(parts[0]) {
		short = strings.ToLower(parts[0])
		parts = parts[1:]
	}
	name = strings.Join(parts, " ")
	if name == "" {
		name = base
	}
	if r, size := utf8.DecodeRuneInString(name); r != utf8.RuneError {
		name = string(unicode.ToUpper(r)) + name[size:]
	}
	return name, short
}

// ImportFile egy .html vagy .zip fájlt ad hozzá. A findAsset a HTML mellől keres képeket (lehet nil).
func (l *Library) ImportFile(fileName string, data []byte, findAsset func(name string) ([]byte, bool)) ([]ImportResult, error) {
	if l.Dir == "" {
		return nil, errors.New("az egyedi sablonok mentéséhez nincs beállítási mappa")
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".zip":
		return l.importZip(fileName, data)
	case ".html", ".htm":
		return []ImportResult{l.importHTML(fileName, data, findAsset)}, nil
	}
	return nil, fmt.Errorf("%s: csak .html sablon vagy a tervezőtől kapott .zip csomag adható hozzá", filepath.Base(fileName))
}

func (l *Library) importZip(fileName string, data []byte) ([]ImportResult, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%s: a ZIP nem olvasható: %w", filepath.Base(fileName), err)
	}
	files := map[string]*zip.File{}
	var htmls []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || strings.HasPrefix(path.Base(f.Name), ".") || strings.Contains(f.Name, "__MACOSX") {
			continue
		}
		files[f.Name] = f
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
			htmls = append(htmls, f)
		}
	}
	read := func(f *zip.File) ([]byte, error) {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, 32<<20))
	}
	var out []ImportResult
	for _, hf := range htmls {
		raw, err := read(hf)
		if err != nil {
			out = append(out, ImportResult{File: hf.Name, Error: err.Error()})
			continue
		}
		// a kitöltött előnézetek (helyőrző nélkül) nem sablonok
		if !bytes.Contains(raw, []byte("{{")) {
			continue
		}
		dir := path.Dir(hf.Name)
		find := func(name string) ([]byte, bool) {
			for _, cand := range []string{path.Join(dir, "assets", name), path.Join(dir, name), path.Join(path.Dir(dir), "assets", name)} {
				if f, ok := files[cand]; ok {
					if b, err := read(f); err == nil {
						return b, true
					}
				}
			}
			for n, f := range files { // bárhol a csomagban
				if path.Base(n) == name {
					if b, err := read(f); err == nil {
						return b, true
					}
				}
			}
			return nil, false
		}
		out = append(out, l.importHTML(hf.Name, raw, find))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: a csomagban nincs {{kulcs}} helyőrzős HTML sablon", filepath.Base(fileName))
	}
	return out, nil
}

func (l *Library) importHTML(fileName string, raw []byte, findAsset func(string) ([]byte, bool)) ImportResult {
	res := ImportResult{File: baseName(fileName)}
	name, short := prettyName(fileName)
	id := Slug(strings.TrimSuffix(res.File, path.Ext(res.File)))
	tpl, err := NewTemplate(id, string(raw))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	var old *Template
	for _, c := range l.custom {
		if c.ID == id {
			old = c
		}
	}
	b := l.builtin(id)
	meta := customMeta{ID: id, Short: short, Name: name, Source: res.File, Added: time.Now().Format("2006-01-02 15:04:05")}
	switch {
	case old != nil: // frissítés: a név és a leírás marad
		meta.Short, meta.Name, meta.Desc, meta.Added = old.Short, old.Name, old.Desc, old.Added
		res.Replaced = true
	case b != nil:
		meta.Short, meta.Name, meta.Desc = b.Short, b.Name, b.Desc
		res.Override = true
	}
	if meta.Short == "" {
		meta.Short = l.nextShort()
	}
	if meta.Desc == "" {
		meta.Desc = "Hozzáadva: " + time.Now().Format("2006. 01. 02.") + " · " + res.File
	}

	d := filepath.Join(l.Dir, id)
	tmp := d + ".uj"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		res.Error = "a sablon nem menthető: " + err.Error()
		return res
	}
	fail := func(err error) ImportResult {
		_ = os.RemoveAll(tmp)
		res.Error = "a sablon nem menthető: " + err.Error()
		return res
	}
	if err := os.WriteFile(filepath.Join(tmp, "sablon.html"), raw, 0o644); err != nil {
		return fail(err)
	}
	mb, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, "meta.json"), mb, 0o644); err != nil {
		return fail(err)
	}
	for _, a := range tpl.AssetRefs {
		if findAsset == nil {
			break
		}
		data, ok := findAsset(a)
		if !ok {
			continue
		}
		if l.hasBuiltinAsset(a) {
			if bi, _ := fs.ReadFile(l.assets, a); bytes.Equal(bi, data) {
				continue // ugyanaz, mint a beépített
			}
		}
		if err := os.MkdirAll(filepath.Join(tmp, "assets"), 0o755); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "assets", a), data, 0o644); err != nil {
			return fail(err)
		}
	}
	_ = os.RemoveAll(d)
	if err := os.Rename(tmp, d); err != nil {
		return fail(err)
	}
	loaded, err := l.loadCustom(d)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	var keep []*Template
	for _, c := range l.custom {
		if c.ID != id {
			keep = append(keep, c)
		}
	}
	l.custom = append(keep, loaded)
	l.sortCustom()
	res.OK, res.ID, res.Name, res.Notes = true, id, loaded.Name, loaded.Report
	return res
}

func (l *Library) nextShort() string {
	used := map[string]bool{}
	for _, t := range l.All() {
		used[t.Short] = true
	}
	for i := 1; ; i++ {
		s := "S" + strconv.Itoa(i)
		if !used[s] {
			return s
		}
	}
}

// Update átnevez egy egyedi sablont.
func (l *Library) Update(id, name, desc string) error {
	for _, c := range l.custom {
		if c.ID != id {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return errors.New("a sablon neve nem lehet üres")
		}
		c.Name, c.Desc = name, strings.TrimSpace(desc)
		m := customMeta{ID: c.ID, Short: c.Short, Name: c.Name, Desc: c.Desc, Source: c.Source, Added: c.Added}
		b, _ := json.MarshalIndent(m, "", "  ")
		return os.WriteFile(filepath.Join(l.Dir, id, "meta.json"), b, 0o644)
	}
	return errors.New("csak a hozzáadott sablonok nevezhetők át")
}

// Delete töröl egy egyedi sablont (ha beépítettet frissített, az eredeti tér vissza).
func (l *Library) Delete(id string) error {
	for i, c := range l.custom {
		if c.ID != id {
			continue
		}
		if err := os.RemoveAll(filepath.Join(l.Dir, id)); err != nil {
			return err
		}
		l.custom = append(l.custom[:i], l.custom[i+1:]...)
		return nil
	}
	return errors.New("a beépített sablonok nem törölhetők")
}
