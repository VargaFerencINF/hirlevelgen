package hirlevel

import (
	"regexp"
	"strings"
)

// Rugalmas termékrács. A sablonok a mobilos elrendezést (a termékek egymás alatt) a <style>
// blokk média-lekérdezéseivel érik el; több levelező viszont a teljes <style> blokkot eldobja
// (pl. a Gmail alkalmazás nem Gmailes fiókkal, több Android-levelező). Ott a terméktéglák
// soronként mindig 3 × 33,333%-osak maradtak: telefonon három 90 px-es, egymásra csúszó oszlop.
//
// FluidTiles a kész levélben a téglák százalékos szélességét 100%-ra cseréli (a sablon
// max-width értéke, pl. 177 px, megmarad), így a téglák a rendelkezésre álló helyhez igazodnak,
// és ha nem férnek el egymás mellett, maguktól a következő sorba kerülnek – <style> nélkül is.
// A rács középre kerül (a tégla saját tartalma balra igazított marad). Asztali nézetben
// (3 × 177 px a 600 px-es levélben), Outlookban (a rejtett MSO-táblázat) és a média-lekérdezést
// ismerő levelezőkben a kinézet nem változik.
var (
	tileDivRe  = regexp.MustCompile(`<div class="(?:[^"]*\s)?tile(?:\s[^"]*)?" style="([^"]*)"`)
	pctWidthRe = regexp.MustCompile(`(^|;)\s*width:\s*\d+(?:\.\d+)?%`)
	maxWidthRe = regexp.MustCompile(`(^|;)\s*max-width:\s*\d+px`)
	gridWrapRe = regexp.MustCompile(`<div style="([^"]*font-size:\s*0[^"]*)">\s*(?:<!--\[if mso\]>[\s\S]*?<!\[endif\]-->\s*)?<div class="(?:[^"]*\s)?tile(?:\s[^"]*)?"`)
)

// FluidTiles lásd fent; a már rugalmas (vagy tégla nélküli) levelet változatlanul adja vissza.
func FluidTiles(doc string) string {
	changed := false
	doc = tileDivRe.ReplaceAllStringFunc(doc, func(m string) string {
		st := tileDivRe.FindStringSubmatch(m)[1]
		if !maxWidthRe.MatchString(st) || !pctWidthRe.MatchString(st) {
			return m // fix px szélesség nélkül a 100% szétfeszítené a téglát
		}
		ns := pctWidthRe.ReplaceAllString(st, "${1}width:100%")
		if !strings.Contains(ns, "text-align") {
			ns = strings.TrimRight(ns, ";") + ";text-align:left;"
		}
		changed = true
		return strings.Replace(m, `style="`+st+`"`, `style="`+ns+`"`, 1)
	})
	if !changed {
		return doc
	}
	return gridWrapRe.ReplaceAllStringFunc(doc, func(m string) string {
		st := gridWrapRe.FindStringSubmatch(m)[1]
		ns := strings.Replace(st, "text-align:left", "text-align:center", 1)
		if ns == st && !strings.Contains(st, "text-align") {
			ns = strings.TrimRight(st, ";") + ";text-align:center;"
		}
		return strings.Replace(m, `style="`+st+`"`, `style="`+ns+`"`, 1)
	})
}
