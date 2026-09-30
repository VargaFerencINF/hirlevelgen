package hirlevel

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

// itemFields a terméktégla mezői ({{item.x}} / {{offer.items.N.x}}).
var itemFields = map[string]bool{"url": true, "image": true, "imageAlt": true, "code": true, "name": true,
	"desc": true, "price": true, "deal": true, "cta": true}

// partnerKeys a partnerenként kitöltött kulcsok.
var partnerKeys = []string{"rep.name", "rep.initials", "rep.photo", "rep.region", "rep.phone", "rep.phone.url",
	"rep.email", "rep.email.url", "assets.base", "offer.items"}

var slotKeyRe = regexp.MustCompile(`^offer\.items\.\d+\.(\w+)$`)

// KnownKey igaz, ha a generátor ki tudja tölteni a kulcsot.
func KnownKey(k string) bool {
	if FieldByKey(k) != nil {
		return true
	}
	for _, p := range partnerKeys {
		if k == p {
			return true
		}
	}
	if strings.HasPrefix(k, "item.") {
		return itemFields[k[5:]]
	}
	if m := slotKeyRe.FindStringSubmatch(k); m != nil {
		return itemFields[m[1]]
	}
	return false
}

var assetRefRe = regexp.MustCompile(`\{\{assets\.base\}\}/([\w.\-]+)`)

// NewTemplate átalakítja és feldolgozza a tervezői formátumú sablont.
func NewTemplate(id, raw string) (*Template, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.Contains(raw, "{{") {
		return nil, fmt.Errorf("a fájlban nincs egyetlen {{kulcs}} helyőrző sem, nem hírlevélsablon")
	}
	conv, info := Convert(raw)
	nodes, keys, err := ParseTemplate(conv)
	if err != nil {
		return nil, fmt.Errorf("hibás sablon: %w", err)
	}
	var unknown []string
	for _, k := range keys {
		if !KnownKey(k) {
			unknown = append(unknown, "{{"+k+"}}")
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("a sablon ismeretlen mezőket használ: %s (a mezők listája: docs/MEZOK-ES-GENERATOR.md)", strings.Join(unknown, ", "))
	}
	t := &Template{ID: id, nodes: nodes, Keys: keys, Raw: raw, Size: len(raw), FixedSlots: info.FixedSlots,
		PhotoOK: info.Photo, Report: info.Notes}
	for _, k := range keys {
		if k == "poll.question" {
			t.HasPoll = true
		}
	}
	seen := map[string]bool{}
	for _, m := range assetRefRe.FindAllStringSubmatch(raw, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			t.AssetRefs = append(t.AssetRefs, m[1])
		}
	}
	sort.Strings(t.AssetRefs)
	if len(raw) > 90*1024 {
		t.Report = append(t.Report, fmt.Sprintf("A sablon %d KB: a Gmail 102 KB felett levágja a levelet, a kitöltött változat ennél nagyobb lehet.", len(raw)/1024))
	}
	return t, nil
}

// LoadTemplates betölti a beépített sablonokat (dir/<id>.html, tervezői formátum).
func LoadTemplates(fsys fs.FS, dir string) ([]*Template, error) {
	out := make([]*Template, 0, len(TemplateInfos))
	for _, info := range TemplateInfos {
		src, err := fs.ReadFile(fsys, path.Join(dir, info.ID+".html"))
		if err != nil {
			return nil, fmt.Errorf("%s sablon: %w", info.ID, err)
		}
		t, err := NewTemplate(info.ID, string(src))
		if err != nil {
			return nil, fmt.Errorf("%s sablon: %w", info.ID, err)
		}
		t.Short, t.Name, t.Desc, t.Builtin = info.Short, info.Name, info.Desc, true
		out = append(out, t)
	}
	return out, nil
}

// FindTemplate azonosító vagy rövid név (v1, v2, v4) szerint keres.
func FindTemplate(list []*Template, id string) *Template {
	for _, t := range list {
		if t.ID == id {
			return t
		}
	}
	for _, t := range list {
		if t.Short == id {
			return t
		}
	}
	return nil
}
