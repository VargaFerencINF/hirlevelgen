package hirlevel

import (
	"regexp"
	"strconv"
	"strings"
)

// Rugalmas termékrács. A sablonok a mobilos elrendezést (a termékek egymás alatt) a <style>
// blokk média-lekérdezéseivel érik el; több levelező viszont a <style> blokkot eldobja vagy a
// média-lekérdezést nem a levél szélességéhez méri (pl. Gmail alkalmazás nem Gmailes fiókkal,
// Thunderbird keskeny olvasópanelben). Ott a terméktéglák soronként 3 × 33,333%-osak maradtak.
//
// FluidTiles a kész levélben a téglákat „Fab Four” szélességre állítja:
//
//	width:33.333%; min-width:33.333%; max-width:100%; width:calc(230400px - 48000%)
//
// A calc a (480px − 100%) × 480 kifejtve (beágyazott zárójel nélkül, a levelezők így biztosan
// értelmezik): ha a rács 480 px-nél szélesebb, az érték óriási negatív, a min-width érvényesül
// (3 termék egy sorban, mint eddig); ha keskenyebb, óriási pozitív, a max-width érvényesül
// (a tégla teljes szélességű, a termékek egymás alá kerülnek). Mindez <style> nélkül, a
// rendelkezésre álló helyhez igazodva. A calc-ot nem ismerő levelező az első width-et (az
// eredeti 33,333%-ot) használja, Outlook asztali változata a rejtett MSO-táblázatot – nálunk
// a kinézet nem változik.
var (
	tileDivRe  = regexp.MustCompile(`<div class="(?:[^"]*\s)?tile(?:\s[^"]*)?" style="([^"]*)"`)
	pctWidthRe = regexp.MustCompile(`(?:^|;)\s*width:\s*(\d+(?:\.\d+)?%)\s*(?:;|$)`)
	maxWidthRe = regexp.MustCompile(`(?:^|;)\s*max-width:\s*\d+px\s*(?:;|$)`)
)

// fluidBreak alatt (a rács szélessége px-ben) kerülnek a termékek egymás alá.
const fluidBreak = 480

// A képviselő gombsora (telefon + e-mail) két egymás melletti táblázatcella; <style> nélkül nem
// tud egymás alá kerülni, és a hosszú e-mail-cím kitolja a levelet a telefon képernyőjéből.
// A cellák inline-block-ok lesznek (ha nem férnek el, egymás alá törnek; Outlook asztali
// változata a cellák display értékét figyelmen kívül hagyja), a gombok szövege pedig szükség
// esetén törhet (a sablon média-lekérdezése is ezt teszi: .rep-btn a{white-space:normal}).
var repBtnTdRe = regexp.MustCompile(`<td class="(?:[^"]*\s)?rep-btn(?:\s[^"]*)?" style="([^"]*)">`)

// FluidTiles lásd fent; a már rugalmas (vagy tégla nélküli) levelet változatlanul adja vissza.
func FluidTiles(doc string) string {
	doc = fluidRepButtons(doc)
	return tileDivRe.ReplaceAllStringFunc(doc, func(m string) string {
		st := tileDivRe.FindStringSubmatch(m)[1]
		pw := pctWidthRe.FindStringSubmatch(st)
		if pw == nil || !maxWidthRe.MatchString(st) || strings.Contains(st, "calc(") {
			return m // csak a sablonok „százalék + max px” tégláihoz nyúlunk
		}
		var keep []string
		for _, d := range strings.Split(st, ";") {
			p := strings.ToLower(strings.TrimSpace(d))
			if p == "" || strings.HasPrefix(p, "width:") || strings.HasPrefix(p, "max-width:") || strings.HasPrefix(p, "min-width:") {
				continue
			}
			keep = append(keep, strings.TrimSpace(d))
		}
		keep = append(keep, "width:"+pw[1], "min-width:"+pw[1], "max-width:100%",
			"width:calc("+strconv.Itoa(fluidBreak*fluidBreak)+"px - "+strconv.Itoa(fluidBreak*100)+"%)")
		ns := strings.Join(keep, ";") + ";"
		return strings.Replace(m, `style="`+st+`"`, `style="`+ns+`"`, 1)
	})
}

func fluidRepButtons(doc string) string {
	var b strings.Builder
	for {
		loc := repBtnTdRe.FindStringSubmatchIndex(doc)
		if loc == nil {
			b.WriteString(doc)
			return b.String()
		}
		st := doc[loc[2]:loc[3]]
		b.WriteString(doc[:loc[2]])
		if !strings.Contains(st, "display:") {
			b.WriteString("display:inline-block;vertical-align:top;")
		}
		b.WriteString(st)
		doc = doc[loc[3]:]
		// a cella gombja: a következő </a>-ig
		end := strings.Index(doc, "</a>")
		if end < 0 {
			continue
		}
		b.WriteString(strings.Replace(doc[:end], "white-space:nowrap;", "white-space:normal;word-break:break-word;", 1))
		doc = doc[end:]
	}
}
