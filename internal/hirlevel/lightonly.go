package hirlevel

import (
	"regexp"
	"strings"
)

// Csak világos megjelenés. A tervezői szabály szerinti sablonok (RESZPONZIV-SZABALYOK.md, köztük
// minden beépített) maguk jelölik magukat „csak világos”-nak; ezekhez LightOnly nem nyúl. A régebbi,
// hozzáadott sablonok fejlécében „color-scheme: light dark” áll sötét színek nélkül, így sötét módú
// telefonon a levelező maga színezte át a levelet; ezeknél LightOnly a kész levelet „csak
// világos”-nak jelöli (meta + :root). A kézbesítést és a spamszűrést nem érinti.
var (
	colorSchemeMetaRe = regexp.MustCompile(`(?i)<meta\s+name="(color-scheme|supported-color-schemes)"\s+content="[^"]*"\s*/?>`)
	headCloseRe       = regexp.MustCompile(`(?i)</head>`)
	styleOpenRe       = regexp.MustCompile(`(?i)<style[^>]*>`)
)

const lightOnlyCSS = ":root{color-scheme:light only;supported-color-schemes:light only;}"

// LightOnly lásd fent; többszöri futtatásra nem változik tovább.
func LightOnly(doc string) string {
	if declaresLightOnly(doc) {
		return doc
	}
	found := false
	doc = colorSchemeMetaRe.ReplaceAllStringFunc(doc, func(m string) string {
		found = true
		name := colorSchemeMetaRe.FindStringSubmatch(m)[1]
		return `<meta name="` + name + `" content="light only">`
	})
	if !found {
		if loc := headCloseRe.FindStringIndex(doc); loc != nil {
			doc = doc[:loc[0]] + `<meta name="color-scheme" content="light only">` + "\n" +
				`<meta name="supported-color-schemes" content="light only">` + "\n" + doc[loc[0]:]
		}
	}
	if !strings.Contains(doc, lightOnlyCSS) {
		if loc := styleOpenRe.FindStringIndex(doc); loc != nil {
			doc = doc[:loc[1]] + "\n" + lightOnlyCSS + doc[loc[1]:]
		} else if loc := headCloseRe.FindStringIndex(doc); loc != nil {
			doc = doc[:loc[0]] + "<style>" + lightOnlyCSS + "</style>\n" + doc[loc[0]:]
		}
	}
	return doc
}

var lightOnlyMetaRe = regexp.MustCompile(`(?i)<meta\s+name="color-scheme"\s+content="(?:light\s+only|only\s+light)"`)

// declaresLightOnly igaz, ha a sablon már maga „csak világos” (meta és :root szabály is).
func declaresLightOnly(doc string) bool {
	return lightOnlyMetaRe.MatchString(doc) && strings.Contains(strings.ReplaceAll(doc, " ", ""), "color-scheme:lightonly")
}
