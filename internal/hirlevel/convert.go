package hirlevel

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Convert a tervezői formátumú sablont (docs/MEZOK-ES-GENERATOR.md, {{kulcs}} helyőrzők,
// fix offer.items.1…N téglák) a generátor bővített sablonjává alakítja. Minden lépés
// csak akkor fut, ha a sablonban megtalálja a hozzá tartozó szerkezetet; ha nem, a
// sablon úgy marad, és a jelentés elmondja, mi nem érhető el. Alapesetben (minden
// mező kitöltve, a tervezett termékszámmal) a kimenet bájtra azonos az eredetivel.
//
//   - terméktégla-rács → {{#grid offer.items:N}} (tetszőleges termékszám);
//   - {{rep.initials}} monogram-cella → fotó, ha van {{rep.photo}};
//   - elhagyható részek: képviselő telefon- és e-mail-gombja, webes verzió link,
//     „teljes ajánlat” link, közösségi ikonok, kérdés-blokk és válaszai.
func Convert(src string) (string, ConvertInfo) {
	s := strings.ReplaceAll(src, "\r\n", "\n")
	var info ConvertInfo

	s = convertGrid(s, &info)
	s = convertRepPhoto(s, &info)

	s = wrapEnclosing(s, "{{rep.phone.url}}", "td", "rep.phone.url", func(seg string) bool {
		return strings.Contains(seg, "<table") && !strings.Contains(seg, "{{rep.email")
	})
	s = wrapEnclosing(s, "{{rep.email.url}}", "td", "rep.email.url", func(seg string) bool {
		return strings.Contains(seg, "<table") && !strings.Contains(seg, "{{rep.phone")
	})
	s = wrapAnchor(s, "utility.browserLink.url")
	s = wrapAnchor(s, "offer.more.url")
	for _, net := range []string{"facebook", "youtube", "instagram", "tiktok"} {
		key := "social." + net + ".url"
		before := s
		s = wrapEnclosing(s, "{{"+key+"}}", "td", key, func(seg string) bool {
			return strings.Count(seg, "{{social.") == 1
		})
		if s == before {
			s = wrapAnchor(s, key)
		}
	}
	s = convertPoll(s, &info)
	return s, info
}

// ConvertInfo az átalakítás eredménye.
type ConvertInfo struct {
	Grid       bool     // a termékrács változó termékszámú lett
	GridCols   int      // téglák soronként
	FixedSlots int      // ha a rács nem alakítható: a fix termékhelyek száma
	Photo      bool     // képviselő-fotó támogatott
	Notes      []string // megjegyzések a felhasználónak
}

func (i *ConvertInfo) note(format string, a ...any) {
	i.Notes = append(i.Notes, fmt.Sprintf(format, a...))
}

// ---------------------------------------------------------------------------
// HTML-címkék keresése (egyszerű, a táblázatos e-mail sablonokhoz elég)

func isTagStart(s string, i int, tag string) bool {
	if !strings.HasPrefix(s[i:], "<"+tag) {
		return false
	}
	j := i + 1 + len(tag)
	if j >= len(s) {
		return false
	}
	switch s[j] {
	case ' ', '>', '\t', '\n', '\r', '/':
		return true
	}
	return false
}

// matchClose a start helyen kezdődő <tag> elem lezárása utáni pozíciót adja (-1, ha nincs).
func matchClose(s string, start int, tag string) int {
	closeTag := "</" + tag + ">"
	depth := 0
	for i := start; i < len(s); {
		nc := strings.Index(s[i:], closeTag)
		if nc < 0 {
			return -1
		}
		no := -1
		for k := i; k < i+nc; k++ {
			if s[k] == '<' && isTagStart(s, k, tag) {
				no = k
				break
			}
		}
		if no >= 0 {
			depth++
			i = no + 1 + len(tag)
			continue
		}
		depth--
		i += nc + len(closeTag)
		if depth == 0 {
			return i
		}
	}
	return -1
}

// enclosing a pos helyet tartalmazó, legkisebb olyan <tag> elemet adja, amelyre ok igaz.
func enclosing(s string, pos int, tag string, ok func(seg string) bool) (int, int) {
	for i := pos; i >= 0; i-- {
		if s[i] != '<' || !isTagStart(s, i, tag) {
			continue
		}
		end := matchClose(s, i, tag)
		if end > pos && ok(s[i:end]) {
			return i, end
		}
	}
	return -1, -1
}

func wrapRange(s string, a, b int, cond string) string {
	return s[:a] + "{{#if " + cond + "}}" + s[a:b] + "{{/if}}" + s[b:]
}

func alreadyWrapped(s, cond string) bool { return strings.Contains(s, "{{#if "+cond+"}}") }

func wrapEnclosing(s, marker, tag, cond string, ok func(seg string) bool) string {
	if alreadyWrapped(s, cond) {
		return s
	}
	pos := strings.Index(s, marker)
	if pos < 0 || strings.Count(s, marker) != 1 {
		return s
	}
	a, b := enclosing(s, pos, tag, ok)
	if a < 0 {
		return s
	}
	return wrapRange(s, a, b, cond)
}

// wrapAnchor a href="{{key}}" linket (a teljes <a> elemet) teszi feltételbe.
func wrapAnchor(s, key string) string {
	if alreadyWrapped(s, key) {
		return s
	}
	pos := strings.Index(s, `href="{{`+key+`}}"`)
	if pos < 0 {
		return s
	}
	a, b := enclosing(s, pos, "a", func(string) bool { return true })
	if a < 0 {
		return s
	}
	return wrapRange(s, a, b, key)
}

// ---------------------------------------------------------------------------
// Terméktégla-rács

var (
	tileStartRe = regexp.MustCompile(`<div class="(?:[^"]*\s)?tile(?:\s[^"]*)?"`)
	msoSepRe    = regexp.MustCompile(`^<!--\[if mso\]></td>(</tr><tr>)?<td width="\d+" valign="top"><!\[endif\]-->\n?`)
	itemKeyRe   = regexp.MustCompile(`\{\{offer\.items\.(\d+)\.`)
)

func convertGrid(s string, info *ConvertInfo) string {
	maxSlot := 0
	for _, m := range itemKeyRe.FindAllStringSubmatch(s, -1) {
		if n, _ := strconv.Atoi(m[1]); n > maxSlot {
			maxSlot = n
		}
	}
	if strings.Contains(s, "{{#grid") {
		info.Grid = true
		return s
	}
	if maxSlot == 0 {
		if strings.Contains(s, "{{offer.") {
			info.note("A sablonban nincs terméktégla (offer.items…), a termékek nem jelennek meg benne.")
		}
		return s
	}
	fail := func(why string) string {
		info.FixedSlots = maxSlot
		info.note("A termékrács nem alakítható át (%s), ezért a sablon fix %d termékhelyes marad.", why, maxSlot)
		return s
	}
	loc := tileStartRe.FindStringIndex(s)
	if loc == nil {
		return fail(`nincs class="tile" tégla`)
	}
	first := loc[0]
	var tiles []string
	cols, rowBreak := 0, false
	pos := first
	for {
		if l := tileStartRe.FindStringIndex(s[pos:min(len(s), pos+200)]); l == nil || l[0] != 0 {
			return fail("a téglák szerkezete szokatlan")
		}
		end := matchClose(s, pos, "div")
		if end < 0 {
			return fail("lezáratlan tégla")
		}
		tiles = append(tiles, s[pos:end])
		m := msoSepRe.FindStringSubmatch(s[end:])
		if m == nil {
			pos = end
			break
		}
		if m[1] != "" {
			rowBreak = true
		} else if !rowBreak {
			cols++
		}
		pos = end + len(m[0])
	}
	last := pos
	if len(tiles) != maxSlot {
		return fail(fmt.Sprintf("%d tégla, de %d termékhely", len(tiles), maxSlot))
	}
	body := strings.ReplaceAll(tiles[0], "{{offer.items.1.", "{{item.")
	if strings.Contains(body, "{{offer.items.") {
		return fail("az első tégla más termék mezőit is használja")
	}
	for n, t := range tiles {
		if strings.ReplaceAll(t, fmt.Sprintf("{{offer.items.%d.", n+1), "{{item.") != body {
			return fail(fmt.Sprintf("a(z) %d. tégla eltér az elsőtől", n+1))
		}
	}
	if strings.Contains(s[:first]+s[last:], "{{offer.items.") {
		return fail("termékmező a rácson kívül is szerepel")
	}
	cols++
	if !rowBreak {
		cols = len(tiles)
	}
	info.Grid, info.GridCols = true, cols
	return s[:first] + "{{#grid offer.items:" + strconv.Itoa(cols) + "}}" + body + "{{/grid}}" + s[last:]
}

// ---------------------------------------------------------------------------
// Képviselő fotó

var (
	initialsTdRe = regexp.MustCompile(`<td\b[^>]*>\s*\{\{rep\.initials\}\}\s*</td>`)
	widthAttrRe  = regexp.MustCompile(`\bwidth="(\d+)"`)
)

func convertRepPhoto(s string, info *ConvertInfo) string {
	if strings.Contains(s, "{{rep.photo}}") {
		info.Photo = true
		return s
	}
	m := initialsTdRe.FindStringIndex(s)
	if m == nil {
		if strings.Contains(s, "{{rep.") {
			info.note("A képviselő monogramja nem külön cellában van, ezért a képviselő fotója ebben a sablonban nem jelenik meg.")
		}
		return s
	}
	td := s[m[0]:m[1]]
	size := 64
	if w := widthAttrRe.FindStringSubmatch(td); w != nil {
		if n, err := strconv.Atoi(w[1]); err == nil && n >= 24 && n <= 200 {
			size = n
		}
	}
	outer := size + 6
	photo := fmt.Sprintf(`<td width="%d" height="%d" valign="middle" style="width:%dpx;height:%dpx;">`+
		`<img src="{{rep.photo}}" alt="{{rep.name}}" width="%d" height="%d" `+
		`style="display:block;width:%dpx;height:%dpx;border:3px solid #F1A32B;border-radius:50%%;`+
		`background-color:#F1A32B;font-family:'Open Sans',Arial,Helvetica,sans-serif;font-size:12px;`+
		`line-height:16px;color:#1A171E;"></td>`, outer, outer, outer, outer, size, size, size, size)
	info.Photo = true
	return s[:m[0]] + "{{#if rep.photo}}" + photo + "{{else}}" + td + "{{/if}}" + s[m[1]:]
}

// ---------------------------------------------------------------------------
// Kérdés-blokk

var coverPollRe = regexp.MustCompile(`(?:&nbsp;&nbsp;·&nbsp;&nbsp;|<span[^>]*>(?:&nbsp;)*\s*·\s*(?:&nbsp;)*</span>\s?)<span[^>]*>\{\{cover\.meta\.poll\}\}</span>`)

func convertPoll(s string, info *ConvertInfo) string {
	if !strings.Contains(s, "{{poll.question}}") || alreadyWrapped(s, "poll.question") {
		return s
	}
	if m := coverPollRe.FindStringIndex(s); m != nil {
		s = wrapRange(s, m[0], m[1], "poll.question")
	}
	anchor := "{{poll.label}}"
	if !strings.Contains(s, anchor) {
		anchor = "{{poll.question}}"
	}
	pos := strings.Index(s, anchor)
	a, b := enclosing(s, pos, "tr", func(seg string) bool {
		return strings.Contains(seg, "{{poll.question}}") &&
			(!strings.Contains(s, "{{poll.note}}") || strings.Contains(seg, "{{poll.note}}")) &&
			!strings.Contains(seg, "{{offer.") && !strings.Contains(seg, "{{rep.") && !strings.Contains(seg, "{{note.")
	})
	if a < 0 {
		info.note("A kérdés-blokk nem tehető elhagyhatóvá: üres kérdésnél is megjelenik.")
		return s
	}
	// a blokk utáni sortörés is a blokkhoz tartozzon
	if b < len(s) && s[b] == '\n' {
		b++
	}
	s = wrapRange(s, a, b, "poll.question")
	for n := 1; n <= 3; n++ {
		key := fmt.Sprintf("poll.answers.%d", n)
		marker := "{{" + key + ".url}}"
		p := strings.Index(s, marker)
		if p < 0 || alreadyWrapped(s, key) {
			continue
		}
		x, y := enclosing(s, p, "table", func(seg string) bool { return strings.Count(seg, "{{poll.answers.") <= 2 })
		if x < 0 {
			continue
		}
		if y < len(s) && s[y] == '\n' {
			y++
		}
		s = wrapRange(s, x, y, key)
	}
	return s
}
