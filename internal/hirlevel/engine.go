package hirlevel

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A sablonok {{kulcs}} helyőrzőket használnak (lásd docs/MEZOK-ES-GENERATOR.md),
// kiegészítve három blokkal:
//
//	{{#if kulcs}} … {{else}} … {{/if}}   – feltételes rész (nem üres érték = igaz)
//	{{#grid offer.items}} … {{/grid}}    – terméktégla-rács, a téglán belül {{item.mező}}
//
// A rács a téglák közé az Outlook (MSO) táblázat-elválasztókat teszi, soronként 3 téglával.

type nodeKind int

const (
	nodeText nodeKind = iota
	nodeVar
	nodeIf
	nodeGrid
)

type node struct {
	kind     nodeKind
	text     string // nodeText
	key      string // nodeVar, nodeIf, nodeGrid
	then     []node
	otherwis []node
}

// Template egy betöltött hírlevélsablon.
type Template struct {
	ID      string   `json:"id"`      // pl. "v4-partnerjelentes"
	Short   string   `json:"short"`   // pl. "v4"
	Name    string   `json:"name"`    // pl. "Partnerjelentés"
	Desc    string   `json:"desc"`    // rövid leírás a felületre
	HasPoll bool     `json:"hasPoll"` // van-e „Egy kérdés” blokk
	Keys    []string `json:"-"`       // a sablonban használt kulcsok
	nodes   []node
}

var tagRe = regexp.MustCompile(`\{\{\s*(#if|#grid|else|/if|/grid)?\s*([\w.]*)\s*\}\}`)

// ParseTemplate feldolgozza a sablon szövegét.
func ParseTemplate(src string) ([]node, []string, error) {
	type frame struct {
		n      *node
		inElse bool
		tag    string
	}
	root := &node{}
	stack := []*frame{{n: root}}
	keys := map[string]bool{}
	appendNode := func(nd node) {
		f := stack[len(stack)-1]
		if f.inElse {
			f.n.otherwis = append(f.n.otherwis, nd)
		} else {
			f.n.then = append(f.n.then, nd)
		}
	}
	pos := 0
	for _, m := range tagRe.FindAllStringSubmatchIndex(src, -1) {
		if m[0] > pos {
			appendNode(node{kind: nodeText, text: src[pos:m[0]]})
		}
		pos = m[1]
		tag := ""
		if m[2] >= 0 {
			tag = src[m[2]:m[3]]
		}
		key := src[m[4]:m[5]]
		line := strings.Count(src[:m[0]], "\n") + 1
		switch tag {
		case "":
			if key == "" {
				return nil, nil, fmt.Errorf("%d. sor: üres helyőrző", line)
			}
			keys[key] = true
			appendNode(node{kind: nodeVar, key: key})
		case "#if", "#grid":
			if key == "" {
				return nil, nil, fmt.Errorf("%d. sor: %s kulcs nélkül", line, tag)
			}
			keys[key] = true
			k := nodeIf
			if tag == "#grid" {
				k = nodeGrid
			}
			nd := &node{kind: k, key: key}
			stack = append(stack, &frame{n: nd, tag: tag})
		case "else":
			f := stack[len(stack)-1]
			if f.tag != "#if" || f.inElse {
				return nil, nil, fmt.Errorf("%d. sor: váratlan {{else}}", line)
			}
			f.inElse = true
		case "/if", "/grid":
			f := stack[len(stack)-1]
			if f.tag != "#"+tag[1:] {
				return nil, nil, fmt.Errorf("%d. sor: váratlan {{%s}}", line, tag)
			}
			stack = stack[:len(stack)-1]
			appendNode(*f.n)
		}
	}
	if len(stack) != 1 {
		return nil, nil, fmt.Errorf("lezáratlan blokk: {{%s %s}}", stack[len(stack)-1].tag, stack[len(stack)-1].n.key)
	}
	if pos < len(src) {
		appendNode(node{kind: nodeText, text: src[pos:]})
	}
	list := make([]string, 0, len(keys))
	for k := range keys {
		list = append(list, k)
	}
	sort.Strings(list)
	return root.then, list, nil
}

// RenderData a kitöltéshez szükséges értékek.
type RenderData struct {
	Values map[string]string   // lapos kulcs → nyers (nem escape-elt) érték
	Items  []map[string]string // termékek: url, image, imageAlt, code, name, desc, price, deal, cta
}

func (d *RenderData) lookup(key string, item map[string]string) (string, bool) {
	if item != nil && strings.HasPrefix(key, "item.") {
		v, ok := item[key[5:]]
		return v, ok
	}
	if strings.HasPrefix(key, "offer.items.") {
		rest := key[len("offer.items."):]
		if i := strings.IndexByte(rest, '.'); i > 0 {
			if n, err := strconv.Atoi(rest[:i]); err == nil {
				if n >= 1 && n <= len(d.Items) {
					v, ok := d.Items[n-1][rest[i+1:]]
					return v, ok
				}
				return "", false
			}
		}
	}
	v, ok := d.Values[key]
	return v, ok
}

// Az Outlook-táblázat elválasztói a téglák között.
const (
	msoNextCell = `<!--[if mso]></td><td width="177" valign="top"><![endif]-->` + "\n"
	msoNextRow  = `<!--[if mso]></td></tr><tr><td width="177" valign="top"><![endif]-->` + "\n"
)

// GridColumns a rács oszlopainak száma asztali nézetben.
const GridColumns = 3

// Render kitölti a sablont. A hiányzó (a Values-ban nem szereplő) kulcsokat visszaadja.
func (t *Template) Render(d *RenderData) (string, []string) {
	var b strings.Builder
	b.Grow(40000)
	missing := map[string]bool{}
	renderNodes(&b, t.nodes, d, nil, missing)
	list := make([]string, 0, len(missing))
	for k := range missing {
		list = append(list, k)
	}
	sort.Strings(list)
	return b.String(), list
}

func renderNodes(b *strings.Builder, nodes []node, d *RenderData, item map[string]string, missing map[string]bool) {
	for i := range nodes {
		n := &nodes[i]
		switch n.kind {
		case nodeText:
			b.WriteString(n.text)
		case nodeVar:
			v, ok := d.lookup(n.key, item)
			if !ok {
				missing[n.key] = true
				continue
			}
			b.WriteString(EscapeHTML(v))
		case nodeIf:
			v, _ := d.lookup(n.key, item)
			if strings.TrimSpace(v) != "" {
				renderNodes(b, n.then, d, item, missing)
			} else {
				renderNodes(b, n.otherwis, d, item, missing)
			}
		case nodeGrid:
			for idx, it := range d.Items {
				if idx > 0 {
					if idx%GridColumns == 0 {
						b.WriteString(msoNextRow)
					} else {
						b.WriteString(msoNextCell)
					}
				}
				renderNodes(b, n.then, d, it, missing)
			}
		}
	}
}

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// EscapeHTML a specifikáció szerinti escape (& < > "), a sortörésekből <br> lesz.
func EscapeHTML(s string) string {
	s = htmlEscaper.Replace(s)
	if strings.ContainsAny(s, "\r\n") {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.ReplaceAll(s, "\n", "<br>")
	}
	return s
}
