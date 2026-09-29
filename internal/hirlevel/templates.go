package hirlevel

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// LoadTemplates betölti a három sablont a fájlrendszerből (dir/<id>.html).
func LoadTemplates(fsys fs.FS, dir string) ([]*Template, error) {
	out := make([]*Template, 0, len(TemplateInfos))
	for _, info := range TemplateInfos {
		src, err := fs.ReadFile(fsys, path.Join(dir, info.ID+".html"))
		if err != nil {
			return nil, fmt.Errorf("%s sablon: %w", info.ID, err)
		}
		t := info
		// Windowsos git-checkout CRLF-re alakíthatja a sablont: egységesen LF.
		text := strings.ReplaceAll(string(src), "\r\n", "\n")
		nodes, keys, err := ParseTemplate(text)
		if err != nil {
			return nil, fmt.Errorf("%s sablon: %w", info.ID, err)
		}
		t.nodes, t.Keys = nodes, keys
		out = append(out, &t)
	}
	return out, nil
}

// FindTemplate azonosító vagy rövid név (v1, v2, v4) szerint keres.
func FindTemplate(list []*Template, id string) *Template {
	for _, t := range list {
		if t.ID == id || t.Short == id {
			return t
		}
	}
	return nil
}
