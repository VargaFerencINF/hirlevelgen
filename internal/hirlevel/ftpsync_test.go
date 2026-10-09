package hirlevel

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// memRemote egy hamis FTP-szerver a memóriában.
type memRemote struct {
	files  map[string][]byte // teljes út → tartalom
	dirs   map[string]bool
	stored []string
	failOn string
}

func newMemRemote() *memRemote {
	return &memRemote{files: map[string][]byte{}, dirs: map[string]bool{"/": true}}
}

func (m *memRemote) List(dir string) (map[string]int64, error) {
	out := map[string]int64{}
	if !m.dirs[dir] {
		return out, nil
	}
	for p, b := range m.files {
		if path.Dir(p) == dir {
			out[path.Base(p)] = int64(len(b))
		}
	}
	for d := range m.dirs {
		if d != "/" && path.Dir(d) == dir {
			out[path.Base(d)+"/"] = -1
		}
	}
	return out, nil
}

func (m *memRemote) MakeDir(dir string) error {
	if !m.dirs[path.Dir(dir)] {
		return errors.New("a szülőmappa nem létezik")
	}
	m.dirs[dir] = true
	return nil
}

func (m *memRemote) Stor(file string, r io.Reader) error {
	if !m.dirs[path.Dir(file)] {
		return errors.New("a mappa nem létezik")
	}
	if strings.HasSuffix(file, m.failOn) && m.failOn != "" {
		return errors.New("553 nem engedélyezett")
	}
	b, _ := io.ReadAll(r)
	m.files[file] = b
	m.stored = append(m.stored, file)
	return nil
}

func TestCollectAssets(t *testing.T) {
	builtin := fstest.MapFS{"logo.png": {Data: []byte("beepitett")}, "cover.jpg": {Data: []byte("c")}, ".DS_Store": {Data: []byte("x")}}
	tplDir := filepath.Join(t.TempDir(), "sajat-sablon", "assets")
	_ = os.MkdirAll(filepath.Join(tplDir, "ikon"), 0o755)
	_ = os.WriteFile(filepath.Join(tplDir, "hullam.png"), []byte("h"), 0o644)
	_ = os.WriteFile(filepath.Join(tplDir, "ikon", "fb.png"), []byte("f"), 0o644)
	local := t.TempDir()
	_ = os.WriteFile(filepath.Join(local, "logo.png"), []byte("sajat logo"), 0o644)
	_ = os.WriteFile(filepath.Join(local, "uj-borito.webp"), []byte("w"), 0o644)
	_ = os.WriteFile(filepath.Join(local, "jegyzet.txt"), []byte("nem kep"), 0o644)
	files, err := CollectAssets(builtin, []string{tplDir, filepath.Join(t.TempDir(), "nincs")}, local)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range files {
		got = append(got, f.Rel+"="+f.Source)
	}
	want := "cover.jpg=beépített képtár,hullam.png=sablon: sajat-sablon,ikon/fb.png=sablon: sajat-sablon,logo.png=saját mappa,uj-borito.webp=saját mappa"
	if strings.Join(got, ",") != want {
		t.Errorf("%s\nvárt: %s", strings.Join(got, ","), want)
	}
	if _, err := CollectAssets(builtin, nil, filepath.Join(local, "nincs-ilyen")); err == nil {
		t.Error("nem létező saját mappa")
	}
}

func TestSyncAssets(t *testing.T) {
	builtin := fstest.MapFS{"a.png": {Data: []byte("aaaa")}, "b.png": {Data: []byte("bb")}, "ikon/c.png": {Data: []byte("c")}, "d.png": {Data: []byte("dddd")}}
	files, _ := CollectAssets(builtin, nil, "")
	m := newMemRemote()
	m.dirs["/www"] = true
	m.files["/www/hirlevel/assets/b.png"] = []byte("bb")   // már fent van, azonos méret
	m.files["/www/hirlevel/assets/d.png"] = []byte("regi") // fent van, azonos méretű (4 bájt) → kimarad
	m.dirs["/www/hirlevel"], m.dirs["/www/hirlevel/assets"] = true, true
	m.files["/www/hirlevel/assets/a.png"] = []byte("x") // fent van, eltérő méret
	var steps int
	res, err := SyncAssets(m, "www/hirlevel/assets/", files, false, func(done, total int, name string) { steps++ })
	if err != nil {
		t.Fatal(err)
	}
	// csak a hiányzó ikon/c.png megy fel (az almappa létrejön); a fent lévők maradnak
	if strings.Join(res.Uploaded, ",") != "ikon/c.png" || strings.Join(res.Skipped, ",") != "b.png,d.png" ||
		strings.Join(res.Changed, ",") != "a.png" || len(res.Replaced) != 0 || res.Bytes != 1 || steps != 5 {
		t.Errorf("eredmény: %+v", res)
	}
	if !m.dirs["/www/hirlevel/assets/ikon"] || string(m.files["/www/hirlevel/assets/a.png"]) != "x" {
		t.Error("almappa / felülírás")
	}
	// másodszor: semmi nem megy fel újra
	m.stored = nil
	res, _ = SyncAssets(m, "/www/hirlevel/assets", files, false, nil)
	if len(m.stored) != 0 || len(res.Skipped) != 3 {
		t.Errorf("ismételt feltöltés: %v %+v", m.stored, res)
	}
	// felülírással az eltérő méretű fájl frissül
	res, _ = SyncAssets(m, "/www/hirlevel/assets", files, true, nil)
	if strings.Join(res.Replaced, ",") != "a.png" || string(m.files["/www/hirlevel/assets/a.png"]) != "aaaa" {
		t.Errorf("felülírás: %+v", res)
	}
	// nem létező mappaszerkezet: létrehozza; egy fájl hibája nem állítja meg a többit
	m2 := newMemRemote()
	m2.failOn = "b.png"
	res, err = SyncAssets(m2, "/uj/kepek", files, false, nil)
	if err != nil || len(res.Uploaded) != 3 || len(res.Failed) != 1 || !strings.Contains(res.Failed[0], "b.png: 553") || !m2.dirs["/uj/kepek/ikon"] {
		t.Errorf("új mappa: %v %+v", err, res)
	}
}
