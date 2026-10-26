package hirlevel

import (
	"regexp"
	"strings"
)

// Csak világos megjelenés. A sablonok fejlécében „color-scheme: light dark” áll, sötét színeket
// viszont nem adnak meg, így sötét módú telefonon a levelező maga színezi át a levelet (furcsa
// színekkel). LightOnly a kész levelet „csak világos”-nak jelöli: az Apple Mail, a Thunderbird,
// a Samsung Email és az Outlook egy része ilyenkor nem színez át. (A Gmail alkalmazás minden
// jelölést figyelmen kívül hagy.) A kézbesítést és a spamszűrést nem érinti.
var (
	colorSchemeMetaRe = regexp.MustCompile(`(?i)<meta\s+name="(color-scheme|supported-color-schemes)"\s+content="[^"]*"\s*/?>`)
	headCloseRe       = regexp.MustCompile(`(?i)</head>`)
	styleOpenRe       = regexp.MustCompile(`(?i)<style[^>]*>`)
)

const lightOnlyCSS = ":root{color-scheme:light only;supported-color-schemes:light only;}"

// LightOnly lásd fent; többszöri futtatásra nem változik tovább.
func LightOnly(doc string) string {
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
