package hirlevel

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Képtár feltöltése FTP-re: a beépített képtár (a sablonok alapértelmezett képei), a hozzáadott
// sablonok saját képei és egy választható helyi mappa új képei. A szerveren már meglévő fájlt
// nem tölti fel újra (a módosított – eltérő méretű – csak kérésre íródik felül).

// RemoteFS a feltöltéshez szükséges FTP-műveletek (a tesztekben hamisítható).
type RemoteFS interface {
	// List a mappa fájljai névvel és mérettel; nem létező mappára üres listát és nil hibát ad.
	List(dir string) (map[string]int64, error)
	MakeDir(dir string) error
	Stor(file string, r io.Reader) error
}

// AssetFile egy feltöltendő fájl.
type AssetFile struct {
	Rel    string `json:"rel"`    // relatív út a képtárban („energofish-mark.png”, „ikonok/x.png”)
	Size   int64  `json:"size"`   // bájt
	Source string `json:"source"` // honnan jön (beépített képtár, sablon, saját mappa)
	read   func() ([]byte, error)
}

// képek és a sablonokhoz tartozó egyéb statikus fájlok
var assetExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true, ".ico": true}

// CollectAssets összegyűjti a feltöltendő fájlokat. Azonos relatív útnál a későbbi forrás nyer
// (a saját mappa képe felülírja a beépítettet). A rejtett fájlok kimaradnak.
func CollectAssets(builtin fs.FS, tplDirs []string, localDir string) ([]AssetFile, error) {
	byRel := map[string]AssetFile{}
	add := func(fsys fs.FS, source string, onlyImages bool) error {
		return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") && p != "." {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() || (onlyImages && !assetExt[strings.ToLower(path.Ext(name))]) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			f, p2 := fsys, p
			byRel[p] = AssetFile{Rel: p, Size: info.Size(), Source: source, read: func() ([]byte, error) { return fs.ReadFile(f, p2) }}
			return nil
		})
	}
	if builtin != nil {
		if err := add(builtin, "beépített képtár", false); err != nil {
			return nil, fmt.Errorf("a beépített képtár nem olvasható: %v", err)
		}
	}
	for _, d := range tplDirs {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			if err := add(os.DirFS(d), "sablon: "+filepath.Base(filepath.Dir(d)), false); err != nil {
				return nil, fmt.Errorf("a sablon képei nem olvashatók (%s): %v", d, err)
			}
		}
	}
	if strings.TrimSpace(localDir) != "" {
		st, err := os.Stat(localDir)
		if err != nil || !st.IsDir() {
			return nil, fmt.Errorf("a saját képek mappája nem található: %s", localDir)
		}
		if err := add(os.DirFS(localDir), "saját mappa", true); err != nil {
			return nil, fmt.Errorf("a saját képek mappája nem olvasható: %v", err)
		}
	}
	out := make([]AssetFile, 0, len(byRel))
	for _, f := range byRel {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// AssetSyncResult a feltöltés eredménye.
type AssetSyncResult struct {
	Uploaded []string `json:"uploaded"` // új fájlok
	Replaced []string `json:"replaced"` // felülírt (eltérő méretű) fájlok
	Skipped  []string `json:"skipped"`  // már fent volt
	Changed  []string `json:"changed"`  // fent van, de eltérő méretű (felülírás nélkül kimaradt)
	Failed   []string `json:"failed"`   // „fájl: hiba”
	Bytes    int64    `json:"bytes"`    // feltöltött bájtok
}

// SyncAssets feltölti a hiányzó fájlokat a távoli mappába (a szükséges almappákat létrehozza).
// A már fent lévő fájlt nem tölti fel újra; az eltérő méretűt csak overwrite esetén.
func SyncAssets(r RemoteFS, remoteDir string, files []AssetFile, overwrite bool, progress func(done, total int, name string)) (AssetSyncResult, error) {
	res := AssetSyncResult{Uploaded: []string{}, Replaced: []string{}, Skipped: []string{}, Changed: []string{}, Failed: []string{}}
	remoteDir = "/" + strings.Trim(path.Clean("/"+strings.ReplaceAll(remoteDir, "\\", "/")), "/")
	listed := map[string]map[string]int64{}
	list := func(dir string) (map[string]int64, error) {
		if m, ok := listed[dir]; ok {
			return m, nil
		}
		m, err := r.List(dir)
		if err != nil {
			return nil, err
		}
		if m == nil {
			m = map[string]int64{}
		}
		listed[dir] = m
		return m, nil
	}
	made := map[string]bool{}
	var mkdirAll func(dir string) error
	mkdirAll = func(dir string) error {
		if dir == "/" || dir == "." || made[dir] {
			return nil
		}
		parent, name := path.Split(strings.TrimRight(dir, "/"))
		parent = strings.TrimRight(parent, "/")
		if parent == "" {
			parent = "/"
		}
		if err := mkdirAll(parent); err != nil {
			return err
		}
		pm, err := list(parent)
		if err != nil {
			return err
		}
		if _, ok := pm[name+"/"]; !ok {
			if err := r.MakeDir(dir); err != nil {
				return fmt.Errorf("a(z) %s mappa nem hozható létre: %v", dir, err)
			}
			pm[name+"/"] = 0
			listed[dir] = map[string]int64{}
		}
		made[dir] = true
		return nil
	}
	for i, f := range files {
		if progress != nil {
			progress(i, len(files), f.Rel)
		}
		target := path.Join(remoteDir, f.Rel)
		dir, name := path.Split(target)
		dir = strings.TrimRight(dir, "/")
		if dir == "" {
			dir = "/"
		}
		if err := mkdirAll(dir); err != nil {
			return res, err
		}
		existing, err := list(dir)
		if err != nil {
			return res, fmt.Errorf("a(z) %s mappa nem listázható: %v", dir, err)
		}
		size, exists := existing[name]
		switch {
		case exists && (size == f.Size || size < 0):
			res.Skipped = append(res.Skipped, f.Rel)
			continue
		case exists && !overwrite:
			res.Changed = append(res.Changed, f.Rel)
			continue
		}
		data, err := f.read()
		if err != nil {
			res.Failed = append(res.Failed, f.Rel+": "+err.Error())
			continue
		}
		if err := r.Stor(target, bytes.NewReader(data)); err != nil {
			res.Failed = append(res.Failed, f.Rel+": "+err.Error())
			continue
		}
		existing[name] = int64(len(data))
		res.Bytes += int64(len(data))
		if exists {
			res.Replaced = append(res.Replaced, f.Rel)
		} else {
			res.Uploaded = append(res.Uploaded, f.Rel)
		}
	}
	if progress != nil {
		progress(len(files), len(files), "")
	}
	return res, nil
}
